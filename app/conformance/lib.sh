#!/usr/bin/env bash
# Menata Runtime — Conformance Suite: shared library
#
# Sourced by run.sh before every tests/*.sh file. Holds everything a test
# file needs regardless of which capability batch it covers: the check()
# harness, HTTP helpers (session_for/csrf_for/post_status/...), the T00
# reachability preflight, and the seeded-account session cache (ALICE,
# BOB, ... -- see seeds/007_authentication.sql). Split out of a single
# 2128-line run.sh (see prototype/go/docs/decisions/007-conformance-suite-
# split.md) -- a pure move, no assertion logic changed.
#
# A handful of helpers (add_business_days, overlay_lifecycle,
# overlay_negatives, process_map_has_shape) originally lived inline in the
# batch that first needed them; relocated here because overlay_lifecycle in
# particular is called again by a LATER batch (decompile-lift) -- centralizing
# every reusable helper here means any single tests/*.sh file can be sourced
# on its own (after this file) without depending on some other file having
# run first to define a function it needs.
# Menata Runtime — Conformance Suite
# Study 4 deliverable (runtime/capability-roadmap.md)
#
# Each test proves one or more capabilities from runtime/capability-registry.md.
# A capability marked ✅ in the registry must keep its test passing (ratchet rule).
#
# Usage:
#   ./conformance/run.sh                     # against http://localhost:4000
#   BASE_URL=https://menata.app ./conformance/run.sh
#
# Requires: seeds 001-007 applied, server running.
# Note: creates test records in the target database (prototype-acceptable).
# T19 is a deliberate, documented exception to "HTTP black-box": it uses psql
# to backdate a date field, simulating time passing without waiting years —
# a test fixture setup, not an inspection of runtime behavior. Set
# DATABASE_URL (same value as the server's .env) to enable it; skipped if unset.
#
# CAP-X02/CAP-O01 (2026-07-12): every test authenticates as a real seeded
# account (seeds/007_authentication.sql) instead of fabricating a
# menata_role/menata_identity/menata_workspace cookie triplet — session_for
# logs in once per account and caches the resulting cookie jar; csrf_for
# scrapes that session's CSRF token once (it's fixed per-session, the same
# value appears on every page, see store.Auth.CSRFToken) and appends it to
# every POST. Role is no longer global — which seeded account plays "the
# Employee" or "the Approver" for a given test is whoever seeds/007 actually
# assigned that role to, in that Application; see the ACCOUNTS map below.

set -u
BASE_URL="${BASE_URL:-http://localhost:4001}"
DATABASE_URL="${DATABASE_URL:-}"
PASS=0
FAIL=0

SESSION_DIR="$(mktemp -d)"
trap 'rm -rf "$SESSION_DIR"' EXIT

check() { # check <test-id> <cap-ids> <description> <condition-result>
    local id="$1" caps="$2" desc="$3" ok="$4"
    if [ "$ok" = "0" ]; then
        printf 'PASS  %-4s %-22s %s\n' "$id" "$caps" "$desc"
        PASS=$((PASS+1))
    else
        printf 'FAIL  %-4s %-22s %s\n' "$id" "$caps" "$desc"
        FAIL=$((FAIL+1))
    fi
}

# session_for <email> <password> -> echoes a cookie-jar path for this
# session. Logs in once per (email) — cached in SESSION_DIR, reused for
# every subsequent call in this run rather than re-authenticating per test.
session_for() {
    local email="$1" password="$2"
    local jar="$SESSION_DIR/$(printf '%s' "$email" | tr -c 'a-zA-Z0-9' '_').jar"
    if [ ! -s "$jar" ]; then
        curl -s -c "$jar" -o /dev/null -X POST "$BASE_URL/login" \
            --data-urlencode "email=$email" --data-urlencode "password=$password"
    fi
    printf '%s' "$jar"
}

# csrf_for <jar> -> echoes that session's CSRF token, scraped once from any
# authenticated page (the logout form in the nav bar carries it on every
# page) and cached alongside the jar.
csrf_for() {
    local jar="$1" cache="$1.csrf"
    if [ ! -s "$cache" ]; then
        curl -s -b "$jar" "$BASE_URL/" \
            | grep -oE 'name="csrf_token" value="[^"]*"' | head -1 \
            | sed -E 's/.*value="([^"]*)"/\1/' > "$cache"
    fi
    cat "$cache"
}

body_contains() { # <url> <needle> <jar>
    curl -s -b "$3" "$1" | grep -q "$2"
}

post_body_contains() { # <url> <data> <needle> <jar>
    local url="$1" data="$2" needle="$3" jar="$4" csrf
    csrf=$(csrf_for "$jar")
    curl -s -X POST -b "$jar" "$url" -d "$data&csrf_token=$csrf" | grep -q "$needle"
}

post_status() { # <url> <data> <jar> -> echoes http code
    local url="$1" data="$2" jar="$3" csrf
    csrf=$(csrf_for "$jar")
    curl -s -o /dev/null -w '%{http_code}' -X POST -b "$jar" "$url" -d "$data&csrf_token=$csrf"
}

post_status_no_csrf() { # <url> <data> <jar> -> echoes http code, csrf_token deliberately omitted
    curl -s -o /dev/null -w '%{http_code}' -X POST -b "$3" "$1" -d "$2"
}

post_redirect() { # <url> <data> <jar> -> echoes redirect url
    local url="$1" data="$2" jar="$3" csrf
    csrf=$(csrf_for "$jar")
    curl -s -o /dev/null -w '%{redirect_url}' -X POST -b "$jar" "$url" -d "$data&csrf_token=$csrf"
}

get_body() { # <url> <jar> -> echoes response body
    curl -s -b "$2" "$1"
}

# count_all_pages <url> <needle> <jar> -> echoes total occurrences of
# needle across EVERY page of a CAP-R05 paginated list, not just the
# default first page -- a plain get_body|grep only sees page 1, which
# silently undercounts (and makes a before/after delta wrong) once a
# Machine this suite reuses across many runs accumulates more than one
# page's worth of records. Reads "Page 1 of N" off the first response to
# learn N, then fetches 2..N.
count_all_pages() {
    local url="$1" needle="$2" jar="$3"
    local first total pages count
    first=$(curl -s -b "$jar" "$url?page=1")
    count=$(echo "$first" | grep -o "$needle" | wc -l)
    pages=$(echo "$first" | grep -oE 'Page [0-9]+ of [0-9]+' | grep -oE '[0-9]+$')
    pages=${pages:-1}
    local p=2
    while [ "$p" -le "$pages" ]; do
        count=$((count + $(curl -s -b "$jar" "$url?page=$p" | grep -o "$needle" | wc -l)))
        p=$((p + 1))
    done
    echo "$count"
}

# user_option_id <form_url> <jar> <display_name> -> echoes the real user id
# backing a `user` field's picker option whose visible text is exactly
# display_name (CAP-F05 -- these fields now store a real users.id, not a
# hand-typed name; scraped from the rendered <option value="ID">Name</option>
# rather than a new DATABASE_URL dependency, keeping this suite's HTTP
# black-box principle -- see conformance/README.md).
user_option_id() {
    local url="$1" jar="$2" name="$3"
    curl -s -b "$jar" "$url" \
        | grep -oE "value=\"[a-f0-9-]+\">$name</option>" \
        | head -1 \
        | grep -oE '"[a-f0-9-]+"' \
        | tr -d '"'
}

# group_member_checkbox_id <group_detail_url> <jar> <display_name> -> echoes
# the real user id backing a member checkbox on /admin/groups/{id} whose
# visible text is exactly display_name (CAP-O07) -- same scraping
# discipline as user_option_id, just a checkbox+label pair instead of an
# <option>, since GroupDetail's own membership list is real workspace user
# ids, not a separate vocabulary.
group_member_checkbox_id() {
    local url="$1" jar="$2" name="$3"
    curl -s -b "$jar" "$url" \
        | grep -oE "value=\"[a-f0-9-]+\"> $name</label>" \
        | head -1 \
        | grep -oE '"[a-f0-9-]+"' \
        | tr -d '"'
}

add_business_days() { # <n> -> echoes YYYY-MM-DD, n business days from today (weekends only)
    local n=$1 d wd
    d=$(date +%Y-%m-%d)
    while [ "$n" -gt 0 ]; do
        d=$(date -d "$d +1 day" +%Y-%m-%d 2>/dev/null || date -j -v+1d -f "%Y-%m-%d" "$d" +%Y-%m-%d)
        wd=$(date -d "$d" +%u 2>/dev/null || date -j -f "%Y-%m-%d" "$d" +%u)
        if [ "$wd" -lt 6 ]; then
            n=$((n - 1))
        fi
    done
    echo "$d"
}

# overlay_lifecycle <machine> <fld_prefix> <evt_prefix> -> creates a record
# assigned to Wati and drives it Open -> ... -> Closed, echoing "OK" only if
# every step behaves exactly as the process declares (incl. the automatic
# Submitted -> Review step firing without any user action).
overlay_lifecycle() {
    local M="$1" F="$2" E="$3" wati_id rec_url
    wati_id=$(user_option_id "$BASE_URL/$M/new" "$SURYA" "Wati Overlay")
    [ -n "$wati_id" ] || { echo "NO-PICKER"; return; }
    rec_url=$(post_redirect "$BASE_URL/$M" "${F}_title=Parity+$$&${F}_assignee=$wati_id" "$SURYA")
    [ -n "$rec_url" ] || { echo "NO-CREATE"; return; }
    body_contains "$rec_url" "Open" "$SURYA" || { echo "NO-INITIAL-STATE"; return; }
    [ "$(post_status "$rec_url/events/${E}_assign" "" "$SURYA")" = "303" ] || { echo "ASSIGN"; return; }
    [ "$(post_status "$rec_url/events/${E}_start" "" "$WATI")" = "303" ]  || { echo "START"; return; }
    [ "$(post_status "$rec_url/events/${E}_submit" "" "$WATI")" = "303" ] || { echo "SUBMIT"; return; }
    body_contains "$rec_url" "Review" "$WATI" || { echo "NO-AUTO"; return; }
    [ "$(post_status "$rec_url/events/${E}_approve" "" "$RIAN")" = "303" ] || { echo "APPROVE"; return; }
    [ "$(post_status "$rec_url/events/${E}_close" "" "$SURYA")" = "303" ]  || { echo "CLOSE"; return; }
    body_contains "$rec_url" "Closed" "$SURYA" || { echo "NO-FINAL-STATE"; return; }
    echo "OK $rec_url"
}

# overlay_negatives <machine> <fld_prefix> <evt_prefix> -> fresh record;
# echoes "OK" only if the state guard (400), the role guard (403), and the
# CAP-P02 ownership guard (403 for a Worker who is not the assignee) all
# reject identically.
overlay_negatives() {
    local M="$1" F="$2" E="$3" wati_id rec_url
    wati_id=$(user_option_id "$BASE_URL/$M/new" "$SURYA" "Wati Overlay")
    rec_url=$(post_redirect "$BASE_URL/$M" "${F}_title=Negative+$$&${F}_assignee=$wati_id" "$SURYA")
    [ -n "$rec_url" ] || { echo "NO-CREATE"; return; }
    [ "$(post_status "$rec_url/events/${E}_approve" "" "$RIAN")" = "400" ] || { echo "STATE-GUARD"; return; }
    [ "$(post_status "$rec_url/events/${E}_assign" "" "$WATI")" = "403" ]  || { echo "ROLE-GUARD"; return; }
    [ "$(post_status "$rec_url/events/${E}_assign" "" "$SURYA")" = "303" ] || { echo "ASSIGN"; return; }
    [ "$(post_status "$rec_url/events/${E}_start" "" "$WINDA")" = "403" ]  || { echo "OWNER-GUARD"; return; }
    [ "$(post_status "$rec_url/events/${E}_start" "" "$WATI")" = "303" ]   || { echo "OWNER-OK"; return; }
    echo "OK"
}

# process_map_has_shape <url> <jar> <fragment...> -> "OK" if every given
# fragment appears in the rendered page, else "MISSING:<fragment>". Shared
# by T140/T141 so both use the byte-identical assertion list.
process_map_has_shape() {
    local url="$1" jar="$2"; shift 2
    local body frag
    body=$(curl -s -b "$jar" "$url")
    for frag in "$@"; do
        echo "$body" | grep -qF "$frag" || { echo "MISSING:$frag"; return; }
    done
    echo "OK"
}

echo "Menata Runtime Conformance Suite"
echo "Target: $BASE_URL"
echo "--------------------------------------------------------------------"

# T00 — server reachable
curl -s -o /dev/null --max-time 5 "$BASE_URL/health"
check T00 "—" "server /health reachable" $?
[ "$FAIL" -gt 0 ] && { echo "Server unreachable — aborting."; exit 1; }

# --- Seeded accounts (seeds/007_authentication.sql), one session per person,
# reused throughout. Role is per-Application (CAP-O01) -- see each account's
# own comment for which Application(s) it holds a role in.
ALICE=$(session_for alice@example.com password)     # Requester (app_design), Submitter (app_approval)
BOB=$(session_for bob@example.com password)         # Designer (app_design), Approver (app_approval) -- assigned to AS1/PS1
CAROL=$(session_for carol@example.com password)     # Approver (app_approval) -- assigned to AS2/PS2, NOT AS1/PS1
WENDY=$(session_for submitter2@example.com password) # Submitter (app_approval) -- distinct from Alice, no records of her own
DAVE=$(session_for employee@example.com password)   # Employee (app_hr)
EVE=$(session_for manager@example.com password)     # Manager (app_hr)
FRANK=$(session_for hr@example.com password)        # HR (app_hr), workspace Admin (ws_default)
GRACE=$(session_for agent@example.com password)     # Agent (app_customer_service)
HENRY=$(session_for supervisor@example.com password) # Supervisor (app_customer_service)
IVAN=$(session_for staff@example.com password)      # Staff (app_ops), workspace Admin (ws_acme)
IVY=$(session_for accountant@example.com password)  # Accountant (app_accounting)
PAM=$(session_for pm@example.com password)          # PM (app_action_lab)
VERA=$(session_for vera@example.com password)       # Member (app_views_lab)
REX=$(session_for rex@example.com password)         # Member (app_record_lifecycle)
NORA=$(session_for nora@example.com password)       # Member (app_permissions_lab)
OMAR=$(session_for omar@example.com password)       # Member (app_permissions_lab)
HANA=$(session_for hana@example.com password)       # HR (app_permissions_lab)
IRIS=$(session_for iris@example.com password)       # Staff (app_permissions_lab)
SAM=$(session_for sam@example.com password)         # Member (app_event_sources)
THEO=$(session_for theo@example.com password)       # Member (app_integration_lab)
YARA=$(session_for yara@example.com password)       # Member (app_workspace_lab_hr, app_workspace_lab_ops)
ZARA=$(session_for zara@example.com password)       # Admin, Member (app_infra_lab)
WIRA=$(session_for wira@example.com password)       # Member (app_field_types_lab)

# Real user ids (CAP-F05) for the four genuine person-reference fields
# (fld_requester, fld_lr_employee, fld_ad_submitted_by, fld_as_approver) --
# these fields now store a real users.id, not a hand-typed name; resolved
# once here via user_option_id and reused throughout, rather than at every
# call site. ALICE_ID is scraped from mch_approval_document's own picker
# (fld_ad_submitted_by), not mch_design_request's -- fld_requester isn't in
# Design Request's own FormView fields (vw_request_form never exposed it, a
# pre-existing gap unrelated to CAP-F05), so no picker for it renders on
# /mch_design_request/new to scrape; a user id is application-scoped by
# role, not by which page it was read from, so the same id resolves either
# way -- Create still accepts and validates fld_requester from raw POST
# data even though no form control renders it.
ALICE_ID=$(user_option_id "$BASE_URL/mch_approval_document/new" "$ALICE" "Alice")
DAVE_ID=$(user_option_id "$BASE_URL/mch_leave_request/new" "$DAVE" "Dave")
BOB_ID=$(user_option_id "$BASE_URL/mch_approval_step/new" "$ALICE" "Bob")
CAROL_ID=$(user_option_id "$BASE_URL/mch_approval_step/new" "$ALICE" "Carol")
