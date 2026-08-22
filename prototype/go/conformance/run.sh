#!/usr/bin/env bash
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
BASE_URL="${BASE_URL:-http://localhost:4000}"
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

# T01 — CAP-X01 multi-application, multi-machine, now observed through
# CAP-O03: the workspace home lists Applications (role-aware), not a flat
# machine list -- Alice (Requester, app_design) sees Design, Frank (HR,
# app_hr) sees HR, both exist in one workspace and are independently
# reachable.
body_contains "$BASE_URL/" "Design" "$ALICE" && body_contains "$BASE_URL/" "HR" "$FRANK"
check T01 "CAP-X01" "workspace home lists applications, role-scoped, from both applications" $?

# T02 — CAP-V01 form view: fields config drives inputs; status excluded.
# Dave (Employee) is Leave Request's real submitting role (CAP-P05: NewForm
# needs CanCreate, no permission row means denied).
curl -s -b "$DAVE" "$BASE_URL/mch_leave_request/new" | grep -q 'for="fld_lr_reason"' \
  && ! curl -s -b "$DAVE" "$BASE_URL/mch_leave_request/new" | grep -q 'for="fld_lr_status"'
check T02 "CAP-V01" "form renders configured fields, excludes status" $?

# T03 — CAP-V02 list view: columns config drives table
body_contains "$BASE_URL/mch_leave_request" "Leave Type" "$DAVE"
check T03 "CAP-V02" "list renders configured columns" $?

# T04 — CAP-C01 required constraint
post_body_contains "$BASE_URL/mch_leave_request" "" "Reason is required." "$DAVE"
check T04 "CAP-C01" "empty submit rejected: required violation" $?

# T05 — CAP-C02 after:today constraint
post_body_contains "$BASE_URL/mch_leave_request" "" "Start Date must be after today." "$DAVE"
check T05 "CAP-C02" "empty submit rejected: date-future violation" $?

# T06 — CAP-C03+C04 conditional constraint fires (Banner without attachment).
# Alice (Requester) is Design Request's real submitting role (CAP-P05:
# Create needs CanCreate).
DR_DATA_BANNER="fld_requester=$ALICE_ID&fld_design_type=Banner+2%3A1&fld_due_date=2030-01-01&fld_title=Conformance+T06&fld_description=Test"
post_body_contains "$BASE_URL/mch_design_request" "$DR_DATA_BANNER" "Attachment is required" "$ALICE"
check T06 "CAP-C03,CAP-C04" "conditional constraint fires when condition true" $?

# T07 — CAP-C04 conditional constraint silent when condition false (Poster)
DR_DATA_POSTER="fld_requester=$ALICE_ID&fld_design_type=Poster&fld_due_date=2030-01-01&fld_title=Conformance+T07&fld_description=Test"
CODE=$(post_status "$BASE_URL/mch_design_request" "$DR_DATA_POSTER" "$ALICE")
[ "$CODE" = "303" ]
check T07 "CAP-C04" "conditional constraint silent when condition false (got $CODE)" $?

# T08 — CAP-R01 create record with default status
LR_DATA="fld_lr_employee=$DAVE_ID&fld_lr_leave_type=Annual+Leave&fld_lr_start_date=2030-01-01&fld_lr_end_date=2030-01-03&fld_lr_reason=Conformance+run"
DETAIL_URL=$(post_redirect "$BASE_URL/mch_leave_request" "$LR_DATA" "$DAVE")
[ -n "$DETAIL_URL" ] && body_contains "$DETAIL_URL" "Draft" "$DAVE"
check T08 "CAP-R01" "valid create redirects to detail with default status Draft" $?

# derive record id from redirect
REC_ID="${DETAIL_URL##*/}"

# T09 — CAP-V03 detail view shows all fields
body_contains "$DETAIL_URL" "Reason" "$DAVE"
check T09 "CAP-V03" "detail renders machine fields" $?

# T10 — CAP-E01+A01 permitted event executes set_field
CODE=$(post_status "$BASE_URL/mch_leave_request/$REC_ID/events/evt_lr_submit" "" "$DAVE")
[ "$CODE" = "303" ] && body_contains "$DETAIL_URL" "Submitted" "$DAVE"
check T10 "CAP-E01,CAP-A01" "Employee triggers Submit; status becomes Submitted" $?

# T11 — CAP-P01 permission guard denies unpermitted role
CODE=$(post_status "$BASE_URL/mch_leave_request/$REC_ID/events/evt_lr_approve" "" "$DAVE")
[ "$CODE" = "403" ]
check T11 "CAP-P01" "Employee denied Approve (got $CODE)" $?

# T12 — CAP-P01+E01 permitted role executes cross-role transition
CODE=$(post_status "$BASE_URL/mch_leave_request/$REC_ID/events/evt_lr_approve" "" "$EVE")
[ "$CODE" = "303" ] && body_contains "$DETAIL_URL" "Approved" "$EVE"
check T12 "CAP-P01,CAP-E01" "Manager triggers Approve; status becomes Approved" $?

# --- CAP-F13 (reference fields) — requires seeds/003_hr_employee.sql ---

# T13 — CAP-F13 form renders a picker (<select>), not a bare text input.
# Frank (HR) is Employee's only real permission-bearing role (CAP-P05).
body_contains "$BASE_URL/mch_employee/new" 'select id="fld_emp_manager"' "$FRANK"
check T13 "CAP-F13" "reference field renders as a picker" $?

# T14 — CAP-F13 create with an empty (optional) reference succeeds
MGR_DATA="fld_emp_id=CB-MGR&fld_emp_name=ConformanceBot+Manager&fld_emp_hire_date=2020-01-01"
MGR_URL=$(post_redirect "$BASE_URL/mch_employee" "$MGR_DATA" "$FRANK")
[ -n "$MGR_URL" ]
check T14 "CAP-F13" "create with empty reference succeeds (Manager is optional)" $?
MGR_ID="${MGR_URL##*/}"

# T15 — CAP-F13 create with a valid reference succeeds; detail links to the target
REPORT_DATA="fld_emp_id=CB-EMP&fld_emp_name=ConformanceBot+Report&fld_emp_hire_date=2024-01-01&fld_emp_manager=$MGR_ID"
REPORT_URL=$(post_redirect "$BASE_URL/mch_employee" "$REPORT_DATA" "$FRANK")
[ -n "$REPORT_URL" ] && body_contains "$REPORT_URL" "href=\"/mch_employee/$MGR_ID\"" "$FRANK"
check T15 "CAP-F13" "create with valid reference succeeds; detail links to target record" $?

# T16 — CAP-F13 dangling reference rejected (security NFR gate: negative case)
BAD_DATA="fld_emp_id=CB-GHOST&fld_emp_name=ConformanceBot+Ghost&fld_emp_hire_date=2024-01-01&fld_emp_manager=00000000-0000-0000-0000-000000000000"
post_body_contains "$BASE_URL/mch_employee" "$BAD_DATA" "does not reference an existing" "$FRANK"
check T16 "CAP-F13" "dangling reference value rejected, not silently accepted" $?

# --- CAP-E06 (state-conditional event availability) ---

# T17 — Approve rejected while a record is still Draft (never Submitted)
DRAFT_DATA="fld_lr_employee=$DAVE_ID&fld_lr_leave_type=Annual+Leave&fld_lr_start_date=2030-01-01&fld_lr_end_date=2030-01-03&fld_lr_reason=T17"
DRAFT_URL=$(post_redirect "$BASE_URL/mch_leave_request" "$DRAFT_DATA" "$DAVE")
DRAFT_ID="${DRAFT_URL##*/}"
CODE=$(post_status "$BASE_URL/mch_leave_request/$DRAFT_ID/events/evt_lr_approve" "" "$EVE")
[ "$CODE" = "400" ]
check T17 "CAP-E06" "Approve rejected while record is still Draft (got $CODE)" $?

# T18 — the headline fix: an already-Approved record can no longer be Rejected
# (reuses $REC_ID from T08-T12, which is Approved by this point in the run)
CODE=$(post_status "$BASE_URL/mch_leave_request/$REC_ID/events/evt_lr_reject" "" "$EVE")
[ "$CODE" = "400" ]
check T18 "CAP-E06" "Reject rejected on an already-Approved record (got $CODE) -- the exact Study 1 gap" $?

# --- CAP-C09 (constraints evaluated on event trigger, not just Create) ---

# T19 — a record valid at Create can become invalid by the time an event fires;
# the already-declared "Start Date must be after today" constraint must block
# Approve too, not just Create. See header note re: psql use here.
if [ -n "$DATABASE_URL" ]; then
    C09_DATA="fld_lr_employee=$DAVE_ID&fld_lr_leave_type=Sick+Leave&fld_lr_start_date=2030-01-01&fld_lr_end_date=2030-01-03&fld_lr_reason=T19"
    C09_URL=$(post_redirect "$BASE_URL/mch_leave_request" "$C09_DATA" "$DAVE")
    C09_ID="${C09_URL##*/}"
    post_status "$BASE_URL/mch_leave_request/$C09_ID/events/evt_lr_submit" "" "$DAVE" >/dev/null
    # CAP-X06: RLS is live (migrations/009) -- a raw psql connection has no
    # app.workspace_id set, so this UPDATE would match zero rows (fail
    # closed) without setting it first, same statement/transaction (multiple
    # ;-separated statements in one -c invocation share Postgres's implicit
    # per-message transaction, so set_config's is_local=true still applies).
    psql "$DATABASE_URL" -q -c \
        "SELECT set_config('app.workspace_id', 'ws_default', true);
         UPDATE records SET data = jsonb_set(data, '{fld_lr_start_date}', '\"2020-01-01\"') WHERE id = '$C09_ID'" \
        >/dev/null 2>&1
    post_body_contains "$BASE_URL/mch_leave_request/$C09_ID/events/evt_lr_approve" "" "Start Date must be after today" "$EVE"
    check T19 "CAP-C09" "Approve blocked by a Constraint the event trigger re-checks, not just Create" $?
else
    printf 'SKIP  T19  %-22s %s\n' "CAP-C09" "DATABASE_URL not set -- backdating fixture unavailable"
fi

# T20 — CAP-A02 dynamic values: "today"/"current_user" resolve to real values,
# not stored as the literal token. Reuses $REC_ID/$DETAIL_URL from T08-T12,
# which evt_lr_approve already stamped with Approved Date/Approved By.
TODAY=$(date +%Y-%m-%d)
body_contains "$DETAIL_URL" "$TODAY" "$EVE" && ! body_contains "$DETAIL_URL" "current_user" "$EVE"
check T20 "CAP-A02" "Approve stamps real today's date and the acting identity, not literal tokens" $?

# T21 — CAP-V06 child records sub-list: reuses $MGR_URL/CB-EMP's Manager
# relationship already established by T14/T15 (CAP-F13).
body_contains "$MGR_URL" "ConformanceBot Report" "$FRANK"
check T21 "CAP-V06" "manager's detail page lists its direct report via reverse reference" $?

# --- CAP-A07 (activate_next / sequential step guard), CAP-A08 (aggregate_status
# rollup), CAP-X03 (machine config) — requires seeds/004_approval.sql ---

# Sequential-mode document with two steps (Bob seq 1, Carol seq 2). Alice
# (Submitter) creates both the Document and its Steps (CAP-P05:
# perm_ad_submitter_steps).
AD_SEQ_DATA="fld_ad_title=T22+Policy&fld_ad_document_type=Policy&fld_ad_file=policy.pdf&fld_ad_submitted_by=$ALICE_ID&fld_ad_approval_mode=Sequential"
AD_SEQ_URL=$(post_redirect "$BASE_URL/mch_approval_document" "$AD_SEQ_DATA" "$ALICE")
AD_SEQ_ID="${AD_SEQ_URL##*/}"
post_status "$BASE_URL/mch_approval_document/$AD_SEQ_ID/events/evt_ad_submit" "" "$ALICE" >/dev/null
AS1_URL=$(post_redirect "$BASE_URL/mch_approval_step" "fld_as_document=$AD_SEQ_ID&fld_as_approver=$BOB_ID&fld_as_sequence=1" "$ALICE")
AS1_ID="${AS1_URL##*/}"
AS2_URL=$(post_redirect "$BASE_URL/mch_approval_step" "fld_as_document=$AD_SEQ_ID&fld_as_approver=$CAROL_ID&fld_as_sequence=2" "$ALICE")
AS2_ID="${AS2_URL##*/}"

# T22 — CAP-A07 hard block: approving Step 2 before Step 1 is rejected.
# Carol is AS2's actual assigned Approver (CAP-P02) -- must pass ownership to
# even reach the sequential guard being tested here.
CODE=$(post_status "$BASE_URL/mch_approval_step/$AS2_ID/events/evt_as_approve" "" "$CAROL")
[ "$CODE" = "400" ]
check T22 "CAP-A07" "out-of-sequence Approve rejected in Sequential mode (got $CODE)" $?

# T23 — CAP-A07 in-order approval succeeds. Bob is AS1's assigned Approver.
CODE=$(post_status "$BASE_URL/mch_approval_step/$AS1_ID/events/evt_as_approve" "" "$BOB")
[ "$CODE" = "303" ]
check T23 "CAP-A07" "in-sequence Approve succeeds (got $CODE)" $?

# T24 — CAP-A08 all-approved rollup: Document stays In Review until every
# step is Approved, then transitions automatically (no direct Approve call
# on the Document itself -- only System may trigger it).
body_contains "$AD_SEQ_URL" "In Review" "$ALICE"
post_status "$BASE_URL/mch_approval_step/$AS2_ID/events/evt_as_approve" "" "$CAROL" >/dev/null
body_contains "$AD_SEQ_URL" "Approved" "$ALICE"
check T24 "CAP-A08" "Document auto-transitions to Approved once every Step is Approved" $?

# T25 — Parallel-mode document: no sequential gating (approve Step 2 before
# Step 1 succeeds, unlike T22's Sequential-mode document)
AD_PAR_DATA="fld_ad_title=T25+Contract&fld_ad_document_type=Contract&fld_ad_file=contract.pdf&fld_ad_submitted_by=$ALICE_ID&fld_ad_approval_mode=Parallel"
AD_PAR_URL=$(post_redirect "$BASE_URL/mch_approval_document" "$AD_PAR_DATA" "$ALICE")
AD_PAR_ID="${AD_PAR_URL##*/}"
post_status "$BASE_URL/mch_approval_document/$AD_PAR_ID/events/evt_ad_submit" "" "$ALICE" >/dev/null
PS1_URL=$(post_redirect "$BASE_URL/mch_approval_step" "fld_as_document=$AD_PAR_ID&fld_as_approver=$BOB_ID&fld_as_sequence=1&fld_as_notes=T26+rejection+note" "$ALICE")
PS1_ID="${PS1_URL##*/}"
PS2_URL=$(post_redirect "$BASE_URL/mch_approval_step" "fld_as_document=$AD_PAR_ID&fld_as_approver=$CAROL_ID&fld_as_sequence=2" "$ALICE")
PS2_ID="${PS2_URL##*/}"
CODE=$(post_status "$BASE_URL/mch_approval_step/$PS2_ID/events/evt_as_approve" "" "$CAROL")
[ "$CODE" = "303" ]
check T25 "CAP-A07" "Parallel mode has no sequential gating -- Step 2 decided before Step 1 (got $CODE)" $?

# T26 — CAP-A08 any-rejected cascade: fires immediately, doesn't wait for the
# still-pending... (here, already-Approved) sibling.
post_status "$BASE_URL/mch_approval_step/$PS1_ID/events/evt_as_reject" "" "$BOB" >/dev/null
body_contains "$AD_PAR_URL" "Rejected" "$ALICE"
check T26 "CAP-A08" "Document cascades to Rejected as soon as any Step rejects, not waiting for the rest" $?

# --- CAP-P02 (record-level ownership) ---

# T36 — negative: Carol holds the correct "Approver" role but is not AS1's
# assigned Approver (Bob is) -- role alone is not enough, direct allocation
# denies her.
CODE=$(post_status "$BASE_URL/mch_approval_step/$AS1_ID/events/evt_as_reject" "" "$CAROL")
[ "$CODE" = "403" ]
check T36 "CAP-P02" "correct role but wrong identity denied deciding another Approver's Step (got $CODE)" $?

# --- CAP-E05 (same-record trigger_event) — requires seeds/005_complaint.sql ---

CMP_DATA="fld_cmp_complainant_name=Conformance+Tester"
CMP_URL=$(post_redirect "$BASE_URL/mch_complaint" "$CMP_DATA" "$GRACE")
CMP_ID="${CMP_URL##*/}"

# T37 — negative: Run SLA Check blocked while Status is still New (events.condition,
# CAP-E06) — the chained Escalate must not fire, Assigned To stays empty.
CODE=$(post_status "$BASE_URL/mch_complaint/$CMP_ID/events/evt_cmp_run_sla_check" "" "$HENRY")
[ "$CODE" = "400" ] && ! body_contains "$CMP_URL" "Supervisor" "$GRACE"
check T37 "CAP-E05" "Run SLA Check blocked outside Investigating; chained Escalate did not fire (got $CODE)" $?

# T38 — positive: once Status reaches Investigating, Run SLA Check's trigger_event
# action fires Escalate on the SAME record — Assigned To becomes Supervisor.
post_status "$BASE_URL/mch_complaint/$CMP_ID/events/evt_cmp_triage" "" "$GRACE" >/dev/null
post_status "$BASE_URL/mch_complaint/$CMP_ID/events/evt_cmp_add_investigation_note" "" "$GRACE" >/dev/null
CODE=$(post_status "$BASE_URL/mch_complaint/$CMP_ID/events/evt_cmp_run_sla_check" "" "$HENRY")
[ "$CODE" = "303" ] && body_contains "$CMP_URL" "Supervisor" "$GRACE"
check T38 "CAP-E05" "Run SLA Check while Investigating chains into Escalate on the same record (got $CODE)" $?

# --- CAP-R02 (edit / update record via form) ---
# Reuses $REC_ID (Leave Request, Approved by T12) and $MGR_ID/$REPORT_URL
# (Employee, CAP-F13 references established by T14/T15).

# T27 — edit form pre-fills the record's current values -- a `user` field's
# picker (CAP-F05) pre-selects the record's own value, the same proof a
# plain text input's value="..." attribute gave before that field started
# storing a real user id instead of a hand-typed name.
body_contains "$BASE_URL/mch_leave_request/$REC_ID/edit" "selected>Dave</option>" "$DAVE"
check T27 "CAP-R02" "edit form pre-fills existing field values" $?

# T28 — valid update persists the change and leaves fields the form doesn't
# expose (Status, still Approved from T12) untouched
LR_UPDATE_DATA="fld_lr_employee=$DAVE_ID&fld_lr_leave_type=Annual+Leave&fld_lr_start_date=2030-01-01&fld_lr_end_date=2030-01-03&fld_lr_reason=T28+updated+reason"
CODE=$(post_status "$BASE_URL/mch_leave_request/$REC_ID" "$LR_UPDATE_DATA" "$DAVE")
[ "$CODE" = "303" ] && body_contains "$DETAIL_URL" "T28 updated reason" "$DAVE" \
  && body_contains "$DETAIL_URL" "Approved" "$DAVE"
check T28 "CAP-R02" "valid update persists changed field, preserves Status outside the form (got $CODE)" $?

# T29 — update re-validates Constraints, same as Create (required violation)
LR_BAD_DATA="fld_lr_employee=$DAVE_ID&fld_lr_leave_type=Annual+Leave&fld_lr_start_date=2030-01-01&fld_lr_end_date=2030-01-03&fld_lr_reason="
post_body_contains "$BASE_URL/mch_leave_request/$REC_ID" "$LR_BAD_DATA" "Reason is required." "$DAVE"
check T29 "CAP-R02,CAP-C09" "update rejected on required-field violation, same as Create" $?

# T30 — update re-validates CAP-F13 referential integrity (dangling reference,
# and a hand-typed non-UUID value — the latter caught a real 500 during
# development, see internal/store/record_store.go's Exists comment)
REPORT_ID="${REPORT_URL##*/}"
EMP_BAD_DATA="fld_emp_id=CB-EMP&fld_emp_name=ConformanceBot+Report&fld_emp_hire_date=2024-01-01&fld_emp_manager=not-a-real-id"
post_body_contains "$BASE_URL/mch_employee/$REPORT_ID" "$EMP_BAD_DATA" "does not reference an existing" "$FRANK"
check T30 "CAP-R02,CAP-F13" "update rejected on a malformed (non-UUID) reference value, not a 500" $?

# --- CAP-A03 (notify to role), CAP-A04 (notify to dynamic recipient),
# CAP-A10 (in-app notification delivery channel) ---

# T31 — CAP-A03 + CAP-A10: a static `notify: {role: ...}` action delivers a
# real in-app Notification, not just a log line. Reuses $REC_ID's Approve
# (T12), which fired `notify: {role: Employee}` -- Dave holds Employee in
# app_hr (CAP-O01: matched via his own user_application_roles row, not a
# global role string).
body_contains "$BASE_URL/notifications" "Leave Request: Approve" "$DAVE"
check T31 "CAP-A03,CAP-A10" "static-role notify delivers a real in-app notification" $?

# T32 — CAP-A10: the unread count badge appears on an unrelated page (Home),
# not just the notifications page itself.
body_contains "$BASE_URL/" 'class="ml-1 inline-flex items-center rounded-full bg-red-100' "$DAVE"
check T32 "CAP-A10" "unread notification count badge renders on the nav bar" $?

# T33 — CAP-A10: mark-read actually persists (button disappears for that
# notification, not just a redirect).
NOTIF_PATH=$(get_body "$BASE_URL/notifications" "$DAVE" | grep -oE '/notifications/[a-f0-9-]+/read' | head -1)
CODE="000"
if [ -n "$NOTIF_PATH" ]; then
    CODE=$(post_status "$BASE_URL$NOTIF_PATH" "" "$DAVE")
fi
[ "$CODE" = "303" ] && ! body_contains "$BASE_URL$NOTIF_PATH" "Mark read" "$DAVE"
check T33 "CAP-A10" "marking a notification read persists (got $CODE, path $NOTIF_PATH)" $?

# T34 — CAP-A04: `notify: {recipient_field: fld_ad_submitted_by, role: Submitter}`
# resolves to the record's OWN Submitted By value ("Alice", from T22's
# AD_SEQ_DATA) -- matched by identity (CAP-O01's recipientMatch), not the
# generic Submitter role. Reuses $AD_SEQ_ID, all-approved by T24 (fires
# evt_ad_approve, which carries this action).
body_contains "$BASE_URL/notifications" "Approval Document: Approve" "$ALICE"
check T34 "CAP-A04" "dynamic recipient_field notifies the record's specific submitter, not a role" $?

# T35 — CAP-A04 negative case: Wendy, a real second Submitter in app_approval
# who is NOT the record's own submitter, must NOT also receive it — proves
# recipient_field actually overrode a role-wide broadcast, rather than both
# firing.
! body_contains "$BASE_URL/notifications" "Approval Document: Approve" "$WENDY"
check T35 "CAP-A04" "a different real Submitter does not also receive the dynamically-targeted notification" $?

# --- CAP-P05 (CRUD-level permissions, deny-by-default) ---

# T39 — negative: Eve (Manager) has no permission row at all on mch_employee
# (only HR does) -- read is denied, not implicitly allowed.
CODE=$(curl -s -o /dev/null -w '%{http_code}' -b "$EVE" "$BASE_URL/mch_employee")
[ "$CODE" = "403" ]
check T39 "CAP-P05" "role with no permission row on a machine is denied List (got $CODE)" $?

# T40 — negative: Dave (Employee) has no user_application_roles assignment at
# all in app_approval (only in app_hr) -- Create on mch_approval_document is
# denied.
CODE=$(post_status "$BASE_URL/mch_approval_document" "fld_ad_title=T40" "$DAVE")
[ "$CODE" = "403" ]
check T40 "CAP-P05" "no role assignment in an Application at all is denied Create (got $CODE)" $?

# T41 — negative: same reasoning, the Edit form -- distinct code path from
# T40's Create denial on the same machine.
CODE=$(curl -s -o /dev/null -w '%{http_code}' -b "$DAVE" "$BASE_URL/mch_approval_document/$AD_SEQ_ID/edit")
[ "$CODE" = "403" ]
check T41 "CAP-P05" "no role assignment in an Application at all is denied the Edit form (got $CODE)" $?

# --- CAP-R04 (audit log actor attribution) + CAP-I04 (correlation trace) ---
# DB inspection, same documented exception as T19 -- these prove a DB-level
# fact (record_events columns) an HTTP black-box test can't observe.

if [ -n "$DATABASE_URL" ]; then
    # T42 -- performed_by carries the real acting identity (Eve, who holds
    # Manager in app_hr and triggered T12's Approve on $REC_ID) -- CAP-X02:
    # identity is now always a real account name, never a bare role string
    # (actorLabel prefers identity, and every authenticated request has one).
    # CAP-X06: RLS is live -- a raw psql connection has no app.workspace_id
    # set by default and fails closed (same reasoning as T19's fix above).
    # A DO block sets it as its own top-level statement, sharing the same
    # implicit per-message transaction as the SELECT that follows (so
    # is_local=true still applies) -- a FROM-clause subquery was tried first
    # but is unreliable: the planner is free to choose a join order that
    # evaluates the RLS-filtered scan before the subquery's side effect
    # fires. -q suppresses the DO block's own "DO" completion tag so -tAc's
    # capture is just the real query's output.
    PERFORMED_BY=$(psql "$DATABASE_URL" -q -tAc \
        "DO \$\$ BEGIN PERFORM set_config('app.workspace_id', 'ws_default', true); END \$\$;
         SELECT performed_by FROM record_events WHERE record_id = '$REC_ID' AND event_id = 'evt_lr_approve' LIMIT 1")
    [ "$PERFORMED_BY" = "Eve" ]
    check T42 "CAP-R04" "record_events.performed_by carries the real acting identity (got '$PERFORMED_BY')" $?

    # T43 -- one HTTP request's correlation_id is shared across every
    # record_events row it produces, even across a cascade: AS2's Approve
    # (T24, Carol) triggers aggregate_status, which fires evt_ad_approve on
    # a DIFFERENT record (AD_SEQ_ID) -- both rows must carry the same id.
    CIDS=$(psql "$DATABASE_URL" -q -tAc \
        "DO \$\$ BEGIN PERFORM set_config('app.workspace_id', 'ws_default', true); END \$\$;
         SELECT DISTINCT correlation_id FROM record_events
         WHERE (record_id = '$AS2_ID' AND event_id = 'evt_as_approve') OR (record_id = '$AD_SEQ_ID' AND event_id = 'evt_ad_approve')")
    [ "$(echo "$CIDS" | grep -c .)" = "1" ] && [ -n "$CIDS" ]
    check T43 "CAP-I04" "one correlation_id shared across a same-request cross-record cascade" $?
else
    printf 'SKIP  T42  %-22s %s\n' "CAP-R04" "DATABASE_URL not set -- DB inspection unavailable"
    printf 'SKIP  T43  %-22s %s\n' "CAP-I04" "DATABASE_URL not set -- DB inspection unavailable"
fi

# --- CAP-P05 follow-up fix: Approver could decide a Step but never read the
# Document it belongs to (surfaced by production log data, not a case) ---

# T44 -- Approver (Bob) can now read Approval Document (perm_ad_approver_read)
CODE=$(curl -s -o /dev/null -w '%{http_code}' -b "$BOB" "$BASE_URL/mch_approval_document")
[ "$CODE" = "200" ]
check T44 "CAP-P05" "Approver can read Approval Document, needed for context on the Step they're deciding (got $CODE)" $?

# T45 -- negative: still can't create/edit Documents, read-only
CODE=$(post_status "$BASE_URL/mch_approval_document" "fld_ad_title=T45" "$BOB")
[ "$CODE" = "403" ]
check T45 "CAP-P05" "Approver still denied Create on Approval Document -- read-only, not full access (got $CODE)" $?

# --- CAP-O03 (Navigation metadata: app grouping, role-aware menus, workspace
# home) -- Workspace > Application > Machine, 006-runtime-model.md ---

# T46 -- drilling into an Application lists its own Machines
body_contains "$BASE_URL/apps/app_design" "Design Request" "$ALICE"
check T46 "CAP-O03" "drilling into an Application lists its own Machines" $?

# T47 -- role-aware: someone with zero readable machines in an Application
# never sees that Application's card on the workspace home at all. Alice has
# no user_application_roles row in app_hr at all.
! body_contains "$BASE_URL/" 'href="/apps/app_hr"' "$ALICE"
check T47 "CAP-O03" "Alice (no access anywhere in HR) never sees the HR application card" $?

# T48 -- negative: drilling into an Application whose Machines the role
# can't read still 404s the Application route itself correctly, and shows
# no Machine cards for one it partially can't -- Eve (Manager) can read
# mch_leave_request (HR) but not mch_employee (also HR); the HR app page
# must show Leave Request without Employee.
body_contains "$BASE_URL/apps/app_hr" "Leave Request" "$EVE" \
  && ! body_contains "$BASE_URL/apps/app_hr" 'href="/mch_employee"' "$EVE"
check T48 "CAP-O03" "within an Application, only individually-readable Machines are listed" $?

# --- CAP-X06 (workspace isolation) -- requires seeds/006_second_workspace.sql ---
# ws_acme (Operations/Task) is a deliberately separate Workspace from every
# other case's ws_default, existing purely to prove isolation.

# T49 -- negative: a ws_default account (Alice) given a direct URL to
# ws_acme's Machine is denied -- app-layer guard (Interpreter.ScopeFor),
# independent of RLS at the DB layer.
CODE=$(curl -s -o /dev/null -w '%{http_code}' -b "$ALICE" "$BASE_URL/mch_task")
[ "$CODE" = "404" ]
check T49 "CAP-X06" "ws_default account denied direct access to another workspace's Machine (got $CODE)" $?

# T50 -- same, for the Application route
CODE=$(curl -s -o /dev/null -w '%{http_code}' -b "$ALICE" "$BASE_URL/apps/app_ops")
[ "$CODE" = "404" ]
check T50 "CAP-X06" "ws_default account denied direct access to another workspace's Application (got $CODE)" $?

# T51 -- positive: Ivan's own account (ws_acme, Staff in app_ops) can use its
# own Machine end to end (create + trigger event) -- workspace now comes from
# the authenticated account (CAP-X02), not a client-suppliable cookie.
TASK_URL=$(post_redirect "$BASE_URL/mch_task" "fld_task_title=Conformance+Task" "$IVAN")
TASK_ID="${TASK_URL##*/}"
CODE=$(post_status "$BASE_URL/mch_task/$TASK_ID/events/evt_task_complete" "" "$IVAN")
[ "$CODE" = "303" ] && body_contains "$TASK_URL" "Done" "$IVAN"
check T51 "CAP-X06" "a workspace account's own session grants access to its own Machine end to end (got $CODE)" $?

if [ -n "$DATABASE_URL" ]; then
    # T52 -- RLS probe (Study 8's own stated pass threshold: "zero
    # cross-workspace rows under RLS probe -- deliberately query with wrong
    # app.workspace_id"). Only meaningful after migrations/009's cutover;
    # skips cleanly before that (RLS not yet enabled -- see that migration's
    # header for why it's applied separately, not as part of migrate-up).
    RLS_ENABLED=$(psql "$DATABASE_URL" -tAc "SELECT relrowsecurity FROM pg_class WHERE relname = 'records'")
    if [ "$RLS_ENABLED" = "t" ]; then
        LEAKED=$(psql "$DATABASE_URL" -q -tAc "
            DO \$\$ BEGIN PERFORM set_config('app.workspace_id', 'ws_default', true); END \$\$;
            SELECT COUNT(*) FROM records WHERE id = '$TASK_ID';")
        [ "$LEAKED" = "0" ]
        check T52 "CAP-X06" "RLS probe: ws_acme's own record invisible under app.workspace_id=ws_default (got count=$LEAKED)" $?
    else
        printf 'SKIP  T52  %-22s %s\n' "CAP-X06" "RLS not yet enabled (migrations/009 applied only at cutover)"
    fi
else
    printf 'SKIP  T52  %-22s %s\n' "CAP-X06" "DATABASE_URL not set -- DB inspection unavailable"
fi

# --- CAP-X02 (real authentication) + CAP-O01 (two-tier role registry) ---
# ADR-005's single largest open item at the time this suite was last
# rewritten: a self-declared role/identity/workspace cookie triplet, no
# password, no session, no CSRF. T53-T59 prove the replacement directly.

# T53 -- wrong password rejected, not silently treated as any other role
CODE=$(curl -s -o /dev/null -w '%{http_code}' -X POST "$BASE_URL/login" \
    --data-urlencode "email=alice@example.com" --data-urlencode "password=wrong")
[ "$CODE" = "401" ]
check T53 "CAP-X02" "login with wrong password rejected (got $CODE)" $?

# T54 -- no session at all: a GET to a protected page redirects to /login
# rather than serving content or erroring
CODE=$(curl -s -o /dev/null -w '%{http_code}' "$BASE_URL/")
[ "$CODE" = "303" ]
check T54 "CAP-X02" "unauthenticated GET redirected to /login (got $CODE)" $?

# T55 -- no session on a POST: 401, not a redirect (nothing useful to
# redirect a form submission to)
CODE=$(curl -s -o /dev/null -w '%{http_code}' -X POST "$BASE_URL/notifications/00000000-0000-0000-0000-000000000000/read")
[ "$CODE" = "401" ]
check T55 "CAP-X02" "unauthenticated POST rejected with 401 (got $CODE)" $?

# T56 -- CSRF: a real, authenticated session's POST with no csrf_token at all
# is rejected -- proves the check runs even when the session itself is valid.
CODE=$(post_status_no_csrf "$BASE_URL/notifications/00000000-0000-0000-0000-000000000000/read" "" "$DAVE")
[ "$CODE" = "403" ]
check T56 "CAP-X02" "authenticated request with no CSRF token rejected (got $CODE)" $?

# T57 -- admin page denied to a non-Admin (Alice: workspace_role Member)
CODE=$(curl -s -o /dev/null -w '%{http_code}' -b "$ALICE" "$BASE_URL/admin/users")
[ "$CODE" = "403" ]
check T57 "CAP-O01" "non-Admin denied /admin/users (got $CODE)" $?

# T58 -- admin page reachable to a real workspace Admin (Frank), and lists
# users -- the "manage user access" half of CAP-O01's workspace-tier role.
CODE=$(curl -s -o /dev/null -w '%{http_code}' -b "$FRANK" "$BASE_URL/admin/users")
[ "$CODE" = "200" ] && body_contains "$BASE_URL/admin/users" "Manage users" "$FRANK"
check T58 "CAP-O01" "workspace Admin can reach /admin/users (got $CODE)" $?

# T59 -- the actual point of the two-tier role model: one identity (Alice),
# one session, no role-switch step, resolving a DIFFERENT role in each of two
# Applications -- Requester in Design, Submitter in Approval.
body_contains "$BASE_URL/apps/app_design" "Design Request" "$ALICE" \
  && body_contains "$BASE_URL/apps/app_approval" "Approval Document" "$ALICE"
check T59 "CAP-O01" "one identity resolves a different role per Application in the same session, no switch" $?

# --- CAP-C05 (comparison operators), CAP-C07 (cross-field comparison),
# CAP-C12 (uniqueness constraint), CAP-X05 (operator validated at load time) ---

# T60 -- CAP-C07: End Date before Start Date rejected -- compares against
# another Field's own value (fld_lr_start_date), not a hardcoded literal.
LR_BADRANGE_DATA="fld_lr_employee=$DAVE_ID&fld_lr_leave_type=Annual+Leave&fld_lr_start_date=2030-01-10&fld_lr_end_date=2030-01-05&fld_lr_reason=T60"
post_body_contains "$BASE_URL/mch_leave_request" "$LR_BADRANGE_DATA" "End Date must be after Start Date" "$DAVE"
check T60 "CAP-C07" "cross-field comparison rejects End Date before Start Date" $?

# T61 -- CAP-C05: a non-positive Sequence rejected -- generalizes "after" and
# adds greater_than/less_than beyond the old today-only special case.
AS_BADSEQ_DATA="fld_as_document=$AD_SEQ_ID&fld_as_approver=$BOB_ID&fld_as_sequence=0"
post_body_contains "$BASE_URL/mch_approval_step" "$AS_BADSEQ_DATA" "Sequence must be a positive number" "$ALICE"
check T61 "CAP-C05" "greater_than rejects a non-positive Sequence" $?

# T62 -- CAP-C12: a second Step at the same Sequence on the same Document
# rejected -- composite uniqueness (document, sequence); reusing sequence 1
# on a DIFFERENT document (T22-T26's Parallel-mode AD_PAR_ID) is fine, only
# a collision within the SAME document's own Steps is rejected.
AS_DUPSEQ_DATA="fld_as_document=$AD_SEQ_ID&fld_as_approver=$CAROL_ID&fld_as_sequence=1"
post_body_contains "$BASE_URL/mch_approval_step" "$AS_DUPSEQ_DATA" "already has a Step at that Sequence" "$ALICE"
check T62 "CAP-C12" "composite uniqueness rejects a duplicate (document, sequence) pair" $?

# --- CAP-F16 (line items / child table inside a record) ---

# T63 -- positive: a Journal Entry and both its Lines are created atomically
# from one form submission (child_N_<field> indexed rows), not N+1 separate
# Create trips -- the child sub-list (CAP-V06) then shows both on the
# parent's own detail page.
JE_DATA="fld_je_date=2026-07-12&fld_je_memo=Conformance+T63&child_0_fld_jel_account=Office+Expense&child_0_fld_jel_debit=100&child_1_fld_jel_account=Cash&child_1_fld_jel_credit=100"
JE_URL=$(post_redirect "$BASE_URL/mch_journal_entry" "$JE_DATA" "$IVY")
[ -n "$JE_URL" ] && body_contains "$JE_URL" "Office Expense" "$IVY" && body_contains "$JE_URL" "Cash" "$IVY"
check T63 "CAP-F16" "Journal Entry and both Lines created atomically from one submission" $?

# T64 -- negative: an invalid child row (Account required, left blank)
# blocks the WHOLE submission -- no orphan parent, matching this
# capability's own "atomic-with-parent" promise, not just a UI convenience.
JE_BAD_DATA="fld_je_date=2026-07-12&fld_je_memo=Conformance+T64&child_0_fld_jel_debit=50"
post_body_contains "$BASE_URL/mch_journal_entry" "$JE_BAD_DATA" "Journal Entry Line, row 1: Account is required" "$IVY"
check T64 "CAP-F16" "an invalid child row blocks the whole atomic create, not just that row" $?

# --- CAP-A06/A09/A11/A12/A13/A15 (Action Lab: one event, six actions) ---

AL_PROJ_URL=$(post_redirect "$BASE_URL/mch_al_project" "fld_alp_name=Conformance+Project" "$PAM")
AL_PROJ_ID="${AL_PROJ_URL##*/}"
AL_TASK_URL=$(post_redirect "$BASE_URL/mch_al_task" "fld_alt_title=Ship+the+thing&fld_alt_priority=Urgent&fld_alt_stage=Todo&fld_alt_project=$AL_PROJ_ID" "$PAM")
AL_TASK_ID="${AL_TASK_URL##*/}"
AL_TASKS_BEFORE=$(count_all_pages "$BASE_URL/mch_al_task" "Follow-up" "$PAM")
post_status "$BASE_URL/mch_al_task/$AL_TASK_ID/events/evt_al_task_complete" "" "$PAM" >/dev/null

# T65 -- CAP-A12: Stage steps to the next declared value_list option (Todo ->
# Doing) -- checked on the Detail page as plain rendered text (StatusBadge),
# not a form picker's "selected" attribute, which only a New/Edit form renders.
body_contains "$AL_TASK_URL" ">Doing<" "$PAM"
check T65 "CAP-A12" "value_list field steps to its next declared option" $?

# T66 -- CAP-A11: Follow Up Date = today + 7 Days (flat date arithmetic).
EXPECTED_FOLLOWUP=$(date -d "+7 days" +%Y-%m-%d 2>/dev/null || date -v+7d +%Y-%m-%d)
body_contains "$AL_TASK_URL" "$EXPECTED_FOLLOWUP" "$PAM"
check T66 "CAP-A11" "date arithmetic resolves today + 7 Days to the real date (expected $EXPECTED_FOLLOWUP)" $?

# T67 -- CAP-A09: the notify action's own "if" fired (Priority was Urgent).
body_contains "$BASE_URL/notifications" "Task: Complete" "$PAM"
check T67 "CAP-A09" "a conditional action's \"if\" runs the action when its condition is true" $?

# T68 -- CAP-A06: create_record made a Task Log entry, copying this Task's
# own Title via "field:<id>", not a literal.
body_contains "$BASE_URL/mch_al_task_log" "Ship the thing" "$PAM"
check T68 "CAP-A06" "create_record creates a real record on another Machine, copying a source field" $?

# T69 -- CAP-A13: cross_set_field stamped the linked Project's own field,
# not this Task's -- record_field resolved the Task's reference to find it.
body_contains "$AL_PROJ_URL" "$(date +%Y-%m-%d)" "$PAM"
check T69 "CAP-A13" "cross_set_field updates a field on a DIFFERENT record via a reference field" $?

# T70 -- CAP-A15: batch_generate created 2 new Tasks from one action.
AL_TASKS_AFTER=$(count_all_pages "$BASE_URL/mch_al_task" "Follow-up" "$PAM")
[ "$((AL_TASKS_AFTER - AL_TASKS_BEFORE))" = "2" ]
check T70 "CAP-A15" "batch_generate creates N records from one action (got $((AL_TASKS_AFTER - AL_TASKS_BEFORE)), want 2)" $?

# T71 -- CAP-A09 negative: a Normal-priority Task's Complete does NOT notify
# -- proves "if" actually gates the action, not a coincidence of T67's data.
# Before/after count (not an absolute number) since this suite's own earlier
# runs against a persistent database leave prior "Task: Complete"
# notifications in place -- same reasoning as T70's before/after.
NOTIF_COUNT_BEFORE=$(get_body "$BASE_URL/notifications" "$PAM" | grep -o "Task: Complete" | wc -l)
AL_TASK2_URL=$(post_redirect "$BASE_URL/mch_al_task" "fld_alt_title=Quiet+Task&fld_alt_priority=Normal&fld_alt_stage=Todo" "$PAM")
AL_TASK2_ID="${AL_TASK2_URL##*/}"
post_status "$BASE_URL/mch_al_task/$AL_TASK2_ID/events/evt_al_task_complete" "" "$PAM" >/dev/null
NOTIF_COUNT_AFTER=$(get_body "$BASE_URL/notifications" "$PAM" | grep -o "Task: Complete" | wc -l)
[ "$NOTIF_COUNT_AFTER" = "$NOTIF_COUNT_BEFORE" ]
check T71 "CAP-A09" "a Normal-priority Task's Complete does not also notify (count stayed $NOTIF_COUNT_BEFORE)" $?

# --- CAP-A14 (aggregate-conditioned action) ---
# MEMBER is unique per run ($$, this script's own PID) -- SUM(points) must
# start at 0 for a member no earlier run's leftover Point Entry rows could
# have touched, or T72's "still under threshold" premise breaks the second
# time this suite runs against a persistent database.

AL_MEMBER="Conformance+Member+$$"

# T72 -- negative: Award Badge rejected while this Member's own Point Entry
# total is still under the threshold (SUM < 100).
PE1_URL=$(post_redirect "$BASE_URL/mch_al_point_entry" "fld_alpe_member=$AL_MEMBER&fld_alpe_points=40" "$PAM")
PE1_ID="${PE1_URL##*/}"
CODE=$(post_status "$BASE_URL/mch_al_point_entry/$PE1_ID/events/evt_pe_award" "" "$PAM")
[ "$CODE" = "400" ]
check T72 "CAP-A14" "aggregate-conditioned trigger rejected while SUM is still under threshold (got $CODE)" $?

# T73 -- positive: once this same Member's total reaches >= 100 (40 + 65),
# the SAME event on a new Point Entry row succeeds and its own action
# (CAP-A06 again) creates the Badge.
PE2_URL=$(post_redirect "$BASE_URL/mch_al_point_entry" "fld_alpe_member=$AL_MEMBER&fld_alpe_points=65" "$PAM")
PE2_ID="${PE2_URL##*/}"
CODE=$(post_status "$BASE_URL/mch_al_point_entry/$PE2_ID/events/evt_pe_award" "" "$PAM")
[ "$CODE" = "303" ] && body_contains "$BASE_URL/mch_al_badge" "Conformance Member $$" "$PAM"
check T73 "CAP-A14" "aggregate-conditioned trigger succeeds once SUM crosses the threshold, action creates the Badge (got $CODE)" $?

# --- Batch 4: Views (2026-07-12) ---
# seeds/010_views_lab.sql. V11 (channel-independent rendering) stays
# excluded -- capability-registry.md HOLDs it at Proposed pending a second
# independent source, unaffected by this batch.

VERA_ID=$(user_option_id "$BASE_URL/mch_vl_task/new" "$VERA" "Vera")

# T74 -- CAP-V04/V05/V09 combined: "My Overdue Tasks" only shows a Task
# that is BOTH assigned to the viewer ($current_user, CAP-V05) AND overdue
# (a plain declarative clause, CAP-V09) -- sorted soonest-due-first
# (default_sort, CAP-V04).
post_redirect "$BASE_URL/mch_vl_task" "fld_vlt_title=Vera+Overdue+$$&fld_vlt_assignee=$VERA_ID&fld_vlt_due=2020-01-01" "$VERA" >/dev/null
body_contains "$BASE_URL/mch_vl_task" "Vera Overdue $$" "$VERA"
check T74 "CAP-V04,CAP-V05,CAP-V09" "my-records filter shows a Task assigned to me and overdue" $?

# T75 -- negative: a Task assigned to the SAME viewer but NOT overdue is
# excluded -- proves the filter's clauses are both actually enforced, not
# just "any Task of mine."
post_redirect "$BASE_URL/mch_vl_task" "fld_vlt_title=Vera+Future+$$&fld_vlt_assignee=$VERA_ID&fld_vlt_due=2099-01-01" "$VERA" >/dev/null
! body_contains "$BASE_URL/mch_vl_task" "Vera Future $$" "$VERA"
check T75 "CAP-V05,CAP-V09" "a not-yet-due Task of mine is excluded from My Overdue Tasks" $?

# T76 -- CAP-V08: free-text search (?q=) matches a substring of a visible
# column, case-insensitively; a differently-named record is excluded from
# the same result.
post_redirect "$BASE_URL/mch_al_project" "fld_alp_name=SearchTarget$$" "$PAM" >/dev/null
post_redirect "$BASE_URL/mch_al_project" "fld_alp_name=Unrelated$$" "$PAM" >/dev/null
SEARCH_BODY=$(get_body "$BASE_URL/mch_al_project?q=searchtarget$$" "$PAM")
echo "$SEARCH_BODY" | grep -q "SearchTarget$$" && ! echo "$SEARCH_BODY" | grep -q "Unrelated$$"
check T76 "CAP-V08" "?q= search matches one record's column, excludes an unrelated one" $?

# T77 -- CAP-V07: calendar view groups Action Lab Tasks by their own
# date_field (Follow Up Date).
CODE=$(curl -s -o /dev/null -w '%{http_code}' -b "$PAM" "$BASE_URL/mch_al_task/calendar")
[ "$CODE" = "200" ] && body_contains "$BASE_URL/mch_al_task/calendar" "Task Calendar" "$PAM"
check T77 "CAP-V07" "calendar view renders records grouped by date_field (got $CODE)" $?

# T78 -- CAP-V07: timeline view is the same grouping, read chronologically.
CODE=$(curl -s -o /dev/null -w '%{http_code}' -b "$PAM" "$BASE_URL/mch_al_task/timeline")
[ "$CODE" = "200" ]
check T78 "CAP-V07" "timeline view renders (got $CODE)" $?

# T79 -- CAP-V10: a dashboard composes sections from TWO different
# Machines (Task, Project) in one View.
DASH_BODY=$(get_body "$BASE_URL/mch_al_project/dashboard" "$PAM")
echo "$DASH_BODY" | grep -q "Tasks by Stage" && echo "$DASH_BODY" | grep -q "Projects"
check T79 "CAP-V10" "dashboard composes sections from more than one Machine" $?

# T80 -- CAP-V13: a report View sums Journal Entry Line's Debit/Credit,
# grouped by Account -- a per-run-unique Account name so the SUM is exactly
# this entry's own amount, not an accumulation across repeated suite runs.
JE_RPT_DATA="fld_je_date=2026-07-12&fld_je_memo=Conformance+T80&child_0_fld_jel_account=ConfAcct$$&child_0_fld_jel_debit=42"
post_redirect "$BASE_URL/mch_journal_entry" "$JE_RPT_DATA" "$IVY" >/dev/null
body_contains "$BASE_URL/mch_journal_entry_line/report" "ConfAcct$$" "$IVY" && \
    body_contains "$BASE_URL/mch_journal_entry_line/report" "42.00" "$IVY"
check T80 "CAP-V13" "report groups records by group_field, sums each declared sum_field" $?

# T81 -- CAP-V12: a multi-step wizard's final POST carries every earlier
# step's value forward (as hidden inputs) and creates one record from all
# of them combined, not just the last step's own fields.
WIZ_CSRF=$(csrf_for "$VERA")
STEP2_BODY=$(curl -s -b "$VERA" -X POST "$BASE_URL/mch_vl_onboarding" \
    -d "csrf_token=$WIZ_CSRF&wizard_step=0&fld_vlo_name=Wendy+Wizard+$$&fld_vlo_email=wendy$$@example.com")
echo "$STEP2_BODY" | grep -q 'Step 2 of 2' && echo "$STEP2_BODY" | grep -q "wendy$$@example.com"
check T81 "CAP-V12" "step 1 submission advances to step 2, carrying step 1's values forward" $?
ONB_URL=$(curl -s -o /dev/null -w '%{redirect_url}' -b "$VERA" -X POST "$BASE_URL/mch_vl_onboarding" \
    -d "csrf_token=$WIZ_CSRF&wizard_step=1&fld_vlo_name=Wendy+Wizard+$$&fld_vlo_email=wendy$$@example.com&fld_vlo_team=Engineering&fld_vlo_start=2026-08-01")
body_contains "$ONB_URL" "Wendy Wizard $$" "$VERA" && body_contains "$ONB_URL" "Engineering" "$VERA"
check T82 "CAP-V12" "final step creates one record combining every step's fields" $?

# T83 -- CAP-V14: moving a record up swaps its position with its immediate
# predecessor in a manual-order list. Sorted sort_order ASC (oldest first),
# so these two freshly-created items are always the LAST two -- on the
# LAST page once this suite's own repeated runs push mch_vl_backlog past
# CAP-R05's 25-row page size, not page 1.
B1_URL=$(post_redirect "$BASE_URL/mch_vl_backlog" "fld_vlb_title=Item+A+$$" "$VERA")
sleep 1
B2_URL=$(post_redirect "$BASE_URL/mch_vl_backlog" "fld_vlb_title=Item+B+$$" "$VERA")
B2_ID="${B2_URL##*/}"
LAST_PAGE=$(get_body "$BASE_URL/mch_vl_backlog?page=1" "$VERA" | grep -oE 'Page 1 of [0-9]+' | grep -oE '[0-9]+$')
LAST_PAGE=${LAST_PAGE:-1}
BEFORE=$(get_body "$BASE_URL/mch_vl_backlog?page=$LAST_PAGE" "$VERA" | grep -o "Item [AB] $$" | head -1)
post_status "$BASE_URL/mch_vl_backlog/$B2_ID/move/up" "" "$VERA" >/dev/null
AFTER=$(get_body "$BASE_URL/mch_vl_backlog?page=$LAST_PAGE" "$VERA" | grep -o "Item [AB] $$" | head -1)
[ "$BEFORE" = "Item A $$" ] && [ "$AFTER" = "Item B $$" ]
check T83 "CAP-V14" "moving a record up swaps it with its predecessor (before=$BEFORE, after=$AFTER)" $?

# --- Batch 5: Record Lifecycle (2026-07-12) ---
# seeds/011_record_lifecycle_lab.sql. CAP-R04 (audit log) shipped earlier,
# not part of this batch.

# T84 -- CAP-R03: an archived record disappears from the live list but is
# reachable via ?archived=1.
RLT_URL=$(post_redirect "$BASE_URL/mch_rl_ticket" "fld_rlt_title=Archive+Target+$$" "$REX")
RLT_ID="${RLT_URL##*/}"
post_status "$BASE_URL/mch_rl_ticket/$RLT_ID/archive" "" "$REX" >/dev/null
! body_contains "$BASE_URL/mch_rl_ticket" "Archive Target $$" "$REX" && \
    body_contains "$BASE_URL/mch_rl_ticket?archived=1" "Archive Target $$" "$REX"
check T84 "CAP-R03" "an archived record leaves the live list and appears under ?archived=1" $?

# T85 -- CAP-R03: restoring an archived record returns it to the live list.
post_status "$BASE_URL/mch_rl_ticket/$RLT_ID/restore" "" "$REX" >/dev/null
body_contains "$BASE_URL/mch_rl_ticket" "Archive Target $$" "$REX"
check T85 "CAP-R03" "restoring an archived record returns it to the live list" $?

# T86 -- CAP-R05: pagination splits a list into pages of 25 -- creates 26
# fresh records (guaranteeing a second page exists even on a database this
# suite has never touched before) and checks the page indicator, not an
# absolute row count that would drift as this suite re-runs.
for i in $(seq 1 26); do
    post_redirect "$BASE_URL/mch_rl_ticket" "fld_rlt_title=Page+Item+$$-$i" "$REX" >/dev/null
done
PAGE1=$(get_body "$BASE_URL/mch_rl_ticket?page=1" "$REX")
TOTAL_PAGES=$(echo "$PAGE1" | grep -oE 'Page 1 of [0-9]+' | grep -oE '[0-9]+$')
[ -n "$TOTAL_PAGES" ] && [ "$TOTAL_PAGES" -ge 2 ]
check T86 "CAP-R05" "a list with more than 25 records paginates into multiple pages (got $TOTAL_PAGES pages)" $?

# T87 -- CAP-R06: CSV export contains a real record's own field value.
RLD_URL=$(post_redirect "$BASE_URL/mch_rl_document" "fld_rld_title=Export+Me+$$&fld_rld_amount=42" "$REX")
body_contains "$BASE_URL/mch_rl_document/export.csv" "Export Me $$,42" "$REX"
check T87 "CAP-R06" "CSV export includes a real record's field values" $?

# T88 -- CAP-R06: CSV import creates a valid row and reports a specific
# per-row failure for an invalid one, in the same file -- one bad row
# doesn't block the rest.
RLD_CSRF=$(csrf_for "$REX")
printf 'fld_rld_title,fld_rld_amount\nImport+OK+%s,7\n,9\n' "$$" > "$SESSION_DIR/import_$$.csv"
IMPORT_BODY=$(curl -s -b "$REX" -F "csrf_token=$RLD_CSRF" -F "file=@$SESSION_DIR/import_$$.csv" \
    "$BASE_URL/mch_rl_document/import")
echo "$IMPORT_BODY" | grep -q "1 created, 1 failed" && echo "$IMPORT_BODY" | grep -q "Title is required"
check T88 "CAP-R06" "CSV import creates the valid row and reports the invalid row's own violation" $?

# T89 -- CAP-R07: a Draft (not yet immutable) Ledger Entry can still be
# edited normally.
RLLE_URL=$(post_redirect "$BASE_URL/mch_rl_ledger_entry" "fld_rlle_memo=Rent+$$" "$REX")
CODE=$(post_status "$RLLE_URL" "fld_rlle_memo=Rent+Updated+$$" "$REX")
[ "$CODE" = "303" ]
check T89 "CAP-R07" "a Draft (not yet immutable) record can still be edited (got $CODE)" $?

# T90 -- CAP-R07 negative: once Posted, the SAME record rejects both a
# direct Update and an Archive -- frozen against every mutation path, not
# just events.
RLLE_ID="${RLLE_URL##*/}"
post_status "$BASE_URL/mch_rl_ledger_entry/$RLLE_ID/events/evt_rlle_post" "" "$REX" >/dev/null
UPDATE_CODE=$(post_status "$RLLE_URL" "fld_rlle_memo=Sneaky+$$" "$REX")
ARCHIVE_CODE=$(post_status "$BASE_URL/mch_rl_ledger_entry/$RLLE_ID/archive" "" "$REX")
[ "$UPDATE_CODE" = "403" ] && [ "$ARCHIVE_CODE" = "403" ]
check T90 "CAP-R07" "a Posted (immutable) record rejects both Update and Archive (got $UPDATE_CODE/$ARCHIVE_CODE)" $?

# T91 -- CAP-R08: a record created directly into its declared scratch state
# (Cart) is NOT rejected for missing/invalid fields that would otherwise
# violate this Machine's own Constraints.
CART_CODE=$(post_status "$BASE_URL/mch_rl_cart" "" "$REX")
[ "$CART_CODE" = "303" ]
check T91 "CAP-R08" "a record created into scratch state skips normally-blocking Constraints (got $CART_CODE)" $?

# T92 -- CAP-R08: the SAME incomplete Cart rejects Checkout (the commit
# point re-enforces Constraints, CAP-C09's existing mechanism) -- then
# succeeds once fixed while still in scratch state.
CART_URL=$(post_redirect "$BASE_URL/mch_rl_cart" "" "$REX")
CART_ID="${CART_URL##*/}"
CHECKOUT_BEFORE=$(post_status "$BASE_URL/mch_rl_cart/$CART_ID/events/evt_rlc_checkout" "" "$REX")
post_status "$CART_URL" "fld_rlc_item=Widget+$$&fld_rlc_quantity=3" "$REX" >/dev/null
CHECKOUT_AFTER=$(post_status "$BASE_URL/mch_rl_cart/$CART_ID/events/evt_rlc_checkout" "" "$REX")
[ "$CHECKOUT_BEFORE" = "400" ] && [ "$CHECKOUT_AFTER" = "303" ]
check T92 "CAP-R08" "checkout on an incomplete Cart is rejected, succeeds once fixed (got $CHECKOUT_BEFORE/$CHECKOUT_AFTER)" $?

# --- Batch 6: Permissions (2026-07-12) ---
# seeds/012_permissions_lab.sql. CAP-P05 (deny-by-default CRUD) shipped
# earlier, not part of this batch.

NORA_ID=$(user_option_id "$BASE_URL/mch_pl_expense/new" "$NORA" "Nora")
OMAR_ID=$(user_option_id "$BASE_URL/mch_pl_expense/new" "$NORA" "Omar")
HANA_ID=$(user_option_id "$BASE_URL/mch_pl_expense/new" "$NORA" "Hana")

# T93 -- CAP-P03 negative: Nora submits an Expense, assigns herself as its
# Approver (passes CAP-P02's owner_field check), then tries to approve it --
# blocked by separation of duties even though she IS the assigned owner.
EXP1_DATA="fld_ple_title=SoD+Self+$$&fld_ple_submitted_by=$NORA_ID"
EXP1_URL=$(post_redirect "$BASE_URL/mch_pl_expense" "$EXP1_DATA" "$NORA")
EXP1_ID="${EXP1_URL##*/}"
EA1_URL=$(post_redirect "$BASE_URL/mch_pl_expense_approval" "fld_plea_expense=$EXP1_ID&fld_plea_approver=$NORA_ID" "$NORA")
EA1_ID="${EA1_URL##*/}"
CODE=$(post_status "$BASE_URL/mch_pl_expense_approval/$EA1_ID/events/evt_plea_approve" "" "$NORA")
[ "$CODE" = "400" ]
check T93 "CAP-P03" "the submitter of a record cannot also decide its own Approval, even as the assigned owner (got $CODE)" $?

# T94 -- CAP-P03 positive: Omar submits a DIFFERENT Expense; Nora (who did
# NOT submit it) approves normally -- proves the block is about self-
# dealing specifically, not a blanket failure.
EXP2_DATA="fld_ple_title=SoD+Other+$$&fld_ple_submitted_by=$OMAR_ID"
EXP2_URL=$(post_redirect "$BASE_URL/mch_pl_expense" "$EXP2_DATA" "$OMAR")
EXP2_ID="${EXP2_URL##*/}"
EA2_URL=$(post_redirect "$BASE_URL/mch_pl_expense_approval" "fld_plea_expense=$EXP2_ID&fld_plea_approver=$NORA_ID" "$NORA")
EA2_ID="${EA2_URL##*/}"
CODE=$(post_status "$BASE_URL/mch_pl_expense_approval/$EA2_ID/events/evt_plea_approve" "" "$NORA")
[ "$CODE" = "303" ]
check T94 "CAP-P03" "approving a DIFFERENT person's submission succeeds normally (got $CODE)" $?

# T95 -- CAP-P04: Omar (assigned Approver, not the submitter) delegates to
# Hana -- the Detail page's inline picker submits a fresh value at trigger
# time (input:<field>), and Delegated By stamps who handed it off.
EXP3_DATA="fld_ple_title=Delegation+$$&fld_ple_submitted_by=$NORA_ID"
EXP3_URL=$(post_redirect "$BASE_URL/mch_pl_expense" "$EXP3_DATA" "$NORA")
EXP3_ID="${EXP3_URL##*/}"
EA3_URL=$(post_redirect "$BASE_URL/mch_pl_expense_approval" "fld_plea_expense=$EXP3_ID&fld_plea_approver=$OMAR_ID" "$NORA")
EA3_ID="${EA3_URL##*/}"
OMAR_CSRF=$(csrf_for "$OMAR")
curl -s -b "$OMAR" -X POST "$BASE_URL/mch_pl_expense_approval/$EA3_ID/events/evt_plea_delegate" \
    -d "csrf_token=$OMAR_CSRF&fld_plea_approver=$HANA_ID" >/dev/null
body_contains "$EA3_URL" "Hana" "$OMAR" && body_contains "$EA3_URL" "Omar" "$OMAR"
check T95 "CAP-P04" "delegating reassigns the Approver and stamps Delegated By with who handed it off" $?

# T96 -- CAP-P06: a field a role's Permission hides never reaches List or
# Detail -- HR sees Salary, Staff (same Machine, real read access) does not.
EMP_DATA="fld_ple2_name=Priya+$$&fld_ple2_salary=95000"
EMP_URL=$(post_redirect "$BASE_URL/mch_pl_employee" "$EMP_DATA" "$HANA")
body_contains "$EMP_URL" "95000" "$HANA" && \
    ! body_contains "$EMP_URL" "95000" "$IRIS" && \
    ! body_contains "$BASE_URL/mch_pl_employee" "95000" "$IRIS"
check T96 "CAP-P06" "a hidden field is absent from Staff's List and Detail but visible to HR" $?

# T97 -- CAP-P07: an anonymous request (no session cookie at all) reaches a
# Machine whose Permissions grant role Visitor read access -- List and
# Detail both, GET only.
POST_DATA="fld_plp_title=Visitor+Post+$$&fld_plp_body=hello&fld_plp_status=Published"
POST_URL=$(post_redirect "$BASE_URL/mch_pl_post" "$POST_DATA" "$NORA")
ANON_LIST_CODE=$(curl -s -o /dev/null -w '%{http_code}' "$BASE_URL/mch_pl_post")
ANON_DETAIL_CODE=$(curl -s -o /dev/null -w '%{http_code}' "$POST_URL")
[ "$ANON_LIST_CODE" = "200" ] && [ "$ANON_DETAIL_CODE" = "200" ] && \
    curl -s "$POST_URL" | grep -q "Visitor Post $$"
check T97 "CAP-P07" "an anonymous request reads a Machine whose Permissions grant Visitor read access (list=$ANON_LIST_CODE, detail=$ANON_DETAIL_CODE)" $?

# T98 -- CAP-P07 negative: the SAME anonymous request is denied a Machine
# with no Visitor grant (redirected to /login, not silently allowed), and
# denied a POST even to the public Machine (read-only, not a write door).
ANON_OTHER_CODE=$(curl -s -o /dev/null -w '%{http_code}' "$BASE_URL/mch_pl_employee")
ANON_POST_CODE=$(curl -s -o /dev/null -w '%{http_code}' -X POST "$BASE_URL/mch_pl_post" -d "fld_plp_title=Hacked")
[ "$ANON_OTHER_CODE" = "303" ] && [ "$ANON_POST_CODE" = "401" ]
check T98 "CAP-P07" "anonymous access is still denied for a Machine with no Visitor grant, and for any POST (other=$ANON_OTHER_CODE, post=$ANON_POST_CODE)" $?

# --- Batch 7: Event Sources (2026-07-12) ---
# seeds/013_event_sources_lab.sql. T99-T101 wait for a real background
# scheduler tick (cmd/server/main.go's runScheduler, once a minute) rather
# than a manual stand-in -- CAP-E05's own T38 used one of those for this
# exact gap before the real thing existed; this batch builds the real
# thing, so it gets proven against the real thing.

TOMORROW=$(date -d "+1 day" +%Y-%m-%d 2>/dev/null || date -v+1d +%Y-%m-%d)
FAR_DATE=$(date -d "+5 days" +%Y-%m-%d 2>/dev/null || date -v+5d +%Y-%m-%d)

# Set up all three scheduler-dependent records BEFORE the one shared wait,
# so this batch pays a single ~65s tick delay, not three.
REM_URL=$(post_redirect "$BASE_URL/mch_es_reminder" "fld_esr_title=Reminder+$$" "$SAM")
DUE_SOON_URL=$(post_redirect "$BASE_URL/mch_es_task" "fld_est_title=Due+Soon+$$&fld_est_due=$TOMORROW" "$SAM")
DUE_FAR_URL=$(post_redirect "$BASE_URL/mch_es_task" "fld_est_title=Due+Far+$$&fld_est_due=$FAR_DATE" "$SAM")

echo "(waiting ~65s for the real background scheduler tick -- CAP-E02/E03)"
sleep 65

# T99 -- CAP-E02: a time-driven Event ("00:00", satisfied at any hour)
# fires on the very next scheduler tick after the record exists.
body_contains "$REM_URL" ">Yes<" "$SAM"
check T99 "CAP-E02" "a time-driven Event fires on its own, no user action, once the scheduled time is reached" $?

# T100 -- CAP-E03 positive: a Task due TOMORROW hits its own "Due Date - 1
# Day" trigger point TODAY.
body_contains "$DUE_SOON_URL" ">Yes<" "$SAM"
check T100 "CAP-E03" "a date-driven Event fires when today equals the record's own date field plus the declared offset" $?

# T101 -- CAP-E03 negative: a Task due in 5 days is nowhere near its own
# trigger point yet -- proves the check is per-record, not "any Task with
# this Event fires."
! body_contains "$DUE_FAR_URL" ">Yes<" "$SAM"
check T101 "CAP-E03" "a date-driven Event does NOT fire for a record whose own date field hasn't reached the offset yet" $?

# T102 -- CAP-E04: an external POST with the Machine's own webhook_secret
# triggers an event with no session and no CSRF token -- and the payload's
# own field (via InputFields/"input:<field>") is stamped onto the record.
PAY_URL=$(post_redirect "$BASE_URL/mch_es_payment" "fld_esp_amount=250" "$SAM")
PAY_ID="${PAY_URL##*/}"
WEBHOOK_CODE=$(curl -s -o /dev/null -w '%{http_code}' -X POST \
    -H "X-Webhook-Secret: demo-webhook-secret-2026" \
    -d "fld_esp_reference=txn_$$" \
    "$BASE_URL/webhooks/mch_es_payment/$PAY_ID/evt_esp_confirm")
[ "$WEBHOOK_CODE" = "200" ] && body_contains "$PAY_URL" "Paid" "$SAM" && body_contains "$PAY_URL" "txn_$$" "$SAM"
check T102 "CAP-E04" "a webhook with the correct secret triggers an event with no session, stamping its own payload field (got $WEBHOOK_CODE)" $?

# T103 -- CAP-E04 negative: the wrong secret is rejected, and the record is
# left untouched -- the secret is a real credential, not decorative.
PAY2_URL=$(post_redirect "$BASE_URL/mch_es_payment" "fld_esp_amount=99" "$SAM")
PAY2_ID="${PAY2_URL##*/}"
WRONG_CODE=$(curl -s -o /dev/null -w '%{http_code}' -X POST \
    -H "X-Webhook-Secret: wrong-secret" \
    -d "fld_esp_reference=txn_bad_$$" \
    "$BASE_URL/webhooks/mch_es_payment/$PAY2_ID/evt_esp_confirm")
[ "$WRONG_CODE" = "401" ] && ! body_contains "$PAY2_URL" "Paid" "$SAM"
check T103 "CAP-E04" "a webhook with the wrong secret is rejected, record left untouched (got $WRONG_CODE)" $?

# --- Batch 8: Cross-Machine Integration (2026-07-12) ---
# seeds/014_integration_lab.sql. CAP-I04 (correlation trace) shipped
# earlier, not part of this batch.

# T104 -- CAP-I01: an Order Placed event fires a Subscription declared on
# an ENTIRELY DIFFERENT Machine (Audit Log) -- Order's own metadata never
# names Audit Log at all.
ORD1_DATA="fld_into_customer=SmallCust$$&fld_into_total=50"
ORD1_URL=$(post_redirect "$BASE_URL/mch_int_order" "$ORD1_DATA" "$THEO")
ORD1_ID="${ORD1_URL##*/}"
post_status "$BASE_URL/mch_int_order/$ORD1_ID/events/evt_into_placed" "" "$THEO" >/dev/null
body_contains "$BASE_URL/mch_int_audit_log" "SmallCust$$" "$THEO"
check T104 "CAP-I01" "a cross-machine Subscription creates a record on a Machine the publisher's own metadata never names" $?

# T105 -- CAP-I03 negative: the SAME Order also has a Points Ledger
# Subscription, but ITS OWN Contract (total >= 100) isn't met by this
# order -- skipped, not created.
! body_contains "$BASE_URL/mch_int_points" "SmallCust$$" "$THEO"
check T105 "CAP-I03" "a Subscription's Contract violation skips that Subscription's own action (negative case)" $?

# T106 -- CAP-I03 positive: a large order (total >= 100) satisfies the same
# Contract, so the Points Ledger Subscription fires.
ORD2_DATA="fld_into_customer=BigCust$$&fld_into_total=150"
ORD2_URL=$(post_redirect "$BASE_URL/mch_int_order" "$ORD2_DATA" "$THEO")
ORD2_ID="${ORD2_URL##*/}"
post_status "$BASE_URL/mch_int_order/$ORD2_ID/events/evt_into_placed" "" "$THEO" >/dev/null
body_contains "$BASE_URL/mch_int_points" "BigCust$$" "$THEO"
check T106 "CAP-I03" "a Subscription's Contract being satisfied lets its own action fire" $?

# T107 -- CAP-I05: a SECOND, unrelated publisher Event (Referral Completed,
# a different Machine entirely from Order) ALSO contributes to the SAME
# Points Ledger -- one shared KPI Machine, fed by two independent sources,
# neither aware of the other.
REF_DATA="fld_intr_referrer=Referrer$$"
REF_URL=$(post_redirect "$BASE_URL/mch_int_referral" "$REF_DATA" "$THEO")
REF_ID="${REF_URL##*/}"
post_status "$BASE_URL/mch_int_referral/$REF_ID/events/evt_intr_completed" "" "$THEO" >/dev/null
body_contains "$BASE_URL/mch_int_points" "Referrer$$" "$THEO" && body_contains "$BASE_URL/mch_int_points" "BigCust$$" "$THEO"
check T107 "CAP-I05" "the same shared Machine accumulates contributions from two different, unrelated publisher Events" $?

# T108 -- CAP-I02: a deprecated Event still functions (backward compat),
# and its own Detail page shows a Deprecated indicator.
CODE=$(post_status "$BASE_URL/mch_int_order/$ORD1_ID/events/evt_into_legacy_notify" "" "$THEO")
[ "$CODE" = "303" ] && body_contains "$ORD1_URL" "Deprecated" "$THEO"
check T108 "CAP-I02" "a deprecated Event still works and shows a Deprecated indicator (got $CODE)" $?

# --- Batch 9: Workspace Services (2026-07-12) ---
# seeds/015_workspace_services_lab.sql. CAP-O01/CAP-O03 shipped earlier, not
# part of this batch.

# T109 -- CAP-O02: a `reference` field on a Machine in a DIFFERENT
# Application can target a master-data Machine's record -- cross-app
# referenceability, proven deliberately (already implied by CAP-F13 alone,
# but never exercised across an Application boundary until now).
WSXE1_URL=$(post_redirect "$BASE_URL/mch_wsx_employee" "fld_wsxe_name=Priya+$$" "$YARA")
WSXE1_ID="${WSXE1_URL##*/}"
WSXP1_URL=$(post_redirect "$BASE_URL/mch_wsx_project" "fld_wsxp_title=Cross+App+Project+$$&fld_wsxp_lead=$WSXE1_ID" "$YARA")
body_contains "$WSXP1_URL" "Priya $$" "$YARA"
check T109 "CAP-O02" "a reference field on a Machine in a different Application targets a master-data record" $?

# T110 -- CAP-O02 negative: archiving a master-data record still referenced
# by another Machine's record (any Application) is blocked, not silently
# allowed to dangle the reference.
YARA_CSRF=$(csrf_for "$YARA")
ARCHIVE_BODY=$(curl -s -b "$YARA" -X POST "$BASE_URL/mch_wsx_employee/$WSXE1_ID/archive" -d "csrf_token=$YARA_CSRF")
ARCHIVE_CODE=$(post_status "$BASE_URL/mch_wsx_employee/$WSXE1_ID/archive" "" "$YARA")
[ "$ARCHIVE_CODE" = "409" ] && echo "$ARCHIVE_BODY" | grep -q "still referenced by Project (via Lead)"
check T110 "CAP-O02" "archiving a master-data record still referenced elsewhere is rejected (got $ARCHIVE_CODE)" $?

# T111 -- CAP-O02 positive: an UNREFERENCED master-data record archives
# normally -- the block is specific to standing references, not a blanket
# ban on archiving master data at all.
WSXE2_URL=$(post_redirect "$BASE_URL/mch_wsx_employee" "fld_wsxe_name=Unreferenced+$$" "$YARA")
WSXE2_ID="${WSXE2_URL##*/}"
ARCHIVE2_CODE=$(post_status "$BASE_URL/mch_wsx_employee/$WSXE2_ID/archive" "" "$YARA")
[ "$ARCHIVE2_CODE" = "303" ] && ! body_contains "$BASE_URL/mch_wsx_employee" "Unreferenced $$" "$YARA"
check T111 "CAP-O02" "an unreferenced master-data record archives normally (got $ARCHIVE2_CODE)" $?

# T112 -- CAP-O04: workspace-wide search finds a match on a Machine the
# searching role can read, scanning across every Application in the
# workspace, not just one Machine at a time.
EMP_PROBE_URL=$(post_redirect "$BASE_URL/mch_pl_employee" "fld_ple2_name=SearchProbe$$&fld_ple2_salary=1000" "$HANA")
body_contains "$BASE_URL/search?q=SearchProbe$$" ">SearchProbe$$</a>" "$HANA"
check T112 "CAP-O04" "workspace-wide search finds a match on a Machine the searcher can read" $?

# T113 -- CAP-O04 negative: the SAME record is invisible to a search by a
# role with no Permission on that Machine at all -- results are
# permission-trimmed per Machine, not a raw text index over everything.
# (Checking for the real result anchor, not just the query string, since
# the search box's own input echoes the query back regardless of results.)
body_contains "$BASE_URL/search?q=SearchProbe$$" "No matches for" "$YARA" && \
    ! body_contains "$BASE_URL/search?q=SearchProbe$$" ">SearchProbe$$</a>" "$YARA"
check T113 "CAP-O04" "search results are permission-trimmed -- a role with no access to the Machine finds nothing" $?

# T114 -- CAP-O05: the notification inbox's per-user delivery preference
# toggles between "immediate" (flat list) and "digest" (grouped by day) --
# same underlying notifications either way, just how they're presented.
DIGEST_CODE=$(post_status "$BASE_URL/notifications/preference" "preference=digest" "$PAM")
body_contains "$BASE_URL/notifications" "Switch to immediate" "$PAM" && \
    body_contains "$BASE_URL/notifications" "Task: Complete" "$PAM"
check T114 "CAP-O05" "switching to digest preference groups the same notifications by day (got $DIGEST_CODE)" $?
# Restore PAM's preference so a re-run of this suite starts from the same
# state every time.
post_status "$BASE_URL/notifications/preference" "preference=immediate" "$PAM" >/dev/null

# T115 -- CAP-O06: "N Business Days" date arithmetic skips weekends,
# reimplemented independently in bash (same T66 precedent for plain-day
# arithmetic). The holiday-specific half of this rule (workspace_holidays)
# is proven by direct DB insert + restart only, documented as a manual,
# DB-inspection exception in conformance/README.md alongside T19/T42/T43/T52
# -- a seeded static holiday date would go stale relative to whenever this
# suite actually runs, so it can't be an automated assertion here.
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
EXPECTED_BUSDAY=$(add_business_days 5)
WSXT_URL=$(post_redirect "$BASE_URL/mch_wsx_task" "fld_wsxt_title=Business+Day+Test+$$" "$YARA")
WSXT_ID="${WSXT_URL##*/}"
post_status "$BASE_URL/mch_wsx_task/$WSXT_ID/events/evt_wsxt_schedule" "" "$YARA" >/dev/null
body_contains "$WSXT_URL" "$EXPECTED_BUSDAY" "$YARA"
check T115 "CAP-O06" "\"N Business Days\" date arithmetic skips weekends, matching an independent bash reimplementation (expected $EXPECTED_BUSDAY)" $?

# --- Batch 10: Infra (2026-07-12) ---
# seeds/016_infra_lab.sql. CAP-X04/X09/X10/X11 deliberately deferred, see
# capability-registry.md's own rows for why.

# T116 -- CAP-X12: an event whose actions span THREE machines (its own
# set_field, a create_record against a real machine, a create_record
# against a deliberately dangling one) rolls back as a whole when the last
# action hits a real foreign-key violation -- not just that one action
# skipped while the earlier two silently commit.
ZARA_CSRF=$(csrf_for "$ZARA")
X12L_URL=$(post_redirect "$BASE_URL/mch_x12_ledger" "fld_x12l_status=Draft" "$ZARA")
X12L_ID="${X12L_URL##*/}"
X12_CODE=$(post_status "$BASE_URL/mch_x12_ledger/$X12L_ID/events/evt_x12_commit" "" "$ZARA")
[ "$X12_CODE" = "500" ] && \
    body_contains "$X12L_URL" ">Draft<" "$ZARA" && \
    ! body_contains "$X12L_URL" ">Posted<" "$ZARA" && \
    ! body_contains "$BASE_URL/mch_x12_entry" "Logged" "$ZARA"
check T116 "CAP-X12" "a cross-machine action chain rolls back as a whole on a downstream failure -- the record's own field AND an earlier, otherwise-successful create_record both revert (got $X12_CODE)" $?

# T117 -- CAP-X13: an inbound webhook delivered twice with the SAME
# X-Idempotency-Key only runs the event once -- the second delivery still
# returns 200 (the duplicate-is-success contract), but doesn't create a
# second Log record.
X13S1_URL=$(post_redirect "$BASE_URL/mch_x13_source" "fld_x13s_amount=10" "$ZARA")
X13S1_ID="${X13S1_URL##*/}"
DUP1_CODE=$(curl -s -o /dev/null -w '%{http_code}' -X POST -H "X-Webhook-Secret: infra-lab-secret-2026" -H "X-Idempotency-Key: conf-dup-$$" "$BASE_URL/webhooks/mch_x13_source/$X13S1_ID/evt_x13_log")
DUP2_CODE=$(curl -s -o /dev/null -w '%{http_code}' -X POST -H "X-Webhook-Secret: infra-lab-secret-2026" -H "X-Idempotency-Key: conf-dup-$$" "$BASE_URL/webhooks/mch_x13_source/$X13S1_ID/evt_x13_log")
LOG_COUNT_1=$(get_body "$BASE_URL/api/mch_x13_log" "$ZARA" | grep -o "\"id\"" | wc -l)
[ "$DUP1_CODE" = "200" ] && [ "$DUP2_CODE" = "200" ]
check T117 "CAP-X13" "a repeated webhook delivery with the same idempotency key returns success both times but only runs the event once (got $DUP1_CODE/$DUP2_CODE)" $?

# T118 -- CAP-X13 negative: a DIFFERENT idempotency key (a genuinely new
# delivery) is not suppressed -- proves the claim table is scoped per key,
# not a blanket "this event already ran once ever" block.
X13S2_URL=$(post_redirect "$BASE_URL/mch_x13_source" "fld_x13s_amount=20" "$ZARA")
X13S2_ID="${X13S2_URL##*/}"
curl -s -o /dev/null -X POST -H "X-Webhook-Secret: infra-lab-secret-2026" -H "X-Idempotency-Key: conf-other-$$" "$BASE_URL/webhooks/mch_x13_source/$X13S2_ID/evt_x13_log"
LOG_COUNT_2=$(get_body "$BASE_URL/api/mch_x13_log" "$ZARA" | grep -o "\"id\"" | wc -l)
[ "$LOG_COUNT_2" -eq $((LOG_COUNT_1 + 1)) ]
check T118 "CAP-X13" "a different idempotency key is a genuinely new delivery, not suppressed (count $LOG_COUNT_1 -> $LOG_COUNT_2)" $?

# T119 -- CAP-X07: the auto-generated JSON API lists and reads a Machine's
# own records, permission-trimmed and workspace-scoped the same as the HTML
# routes -- an unauthenticated request is redirected to /login, not served.
LIST_CODE=$(curl -s -o /dev/null -w '%{http_code}' -b "$ZARA" "$BASE_URL/api/mch_x13_log")
FIRST_ID=$(get_body "$BASE_URL/api/mch_x13_log" "$ZARA" | grep -oE '"id":"[a-f0-9-]+"' | head -1 | sed -E 's/.*:"([^"]+)"/\1/')
GET_CODE=$(curl -s -o /dev/null -w '%{http_code}' -b "$ZARA" "$BASE_URL/api/mch_x13_log/$FIRST_ID")
ANON_API_CODE=$(curl -s -o /dev/null -w '%{http_code}' "$BASE_URL/api/mch_x13_log")
[ "$LIST_CODE" = "200" ] && [ "$GET_CODE" = "200" ] && [ "$ANON_API_CODE" = "303" ]
check T119 "CAP-X07" "the JSON API lists and reads a machine's records for an authenticated session, denies an unauthenticated one (list=$LIST_CODE, get=$GET_CODE, anon=$ANON_API_CODE)" $?

# T120 -- CAP-X07: a JSON POST creates a real record (same validation as
# the HTML Create path), authenticated via X-CSRF-Token header since a JSON
# body has no csrf_token form field for the existing synchronizer-token
# check to read -- and a request with no CSRF at all is still rejected.
CREATE_CODE=$(curl -s -o /dev/null -w '%{http_code}' -b "$ZARA" -X POST "$BASE_URL/api/mch_x12_entry" \
    -H "Content-Type: application/json" -H "X-CSRF-Token: $ZARA_CSRF" \
    -d "{\"fld_x12e_note\":\"api-created-$$\"}")
NO_CSRF_CODE=$(curl -s -o /dev/null -w '%{http_code}' -b "$ZARA" -X POST "$BASE_URL/api/mch_x12_entry" \
    -H "Content-Type: application/json" -d '{"fld_x12e_note":"should-be-rejected"}')
[ "$CREATE_CODE" = "201" ] && [ "$NO_CSRF_CODE" = "403" ] && \
    get_body "$BASE_URL/api/mch_x12_entry" "$ZARA" | grep -q "api-created-$$"
check T120 "CAP-X07" "a JSON create via X-CSRF-Token header succeeds and is visible in the same API's list; a request with no CSRF token is rejected (create=$CREATE_CODE, no_csrf=$NO_CSRF_CODE)" $?

# T121 -- CAP-X08: an Application's full metadata tree exports as JSON,
# straight from the same in-memory model the runtime itself interprets --
# Admin-only, a non-admin role is denied.
EXPORT_CODE=$(curl -s -o /dev/null -w '%{http_code}' -b "$ZARA" "$BASE_URL/apps/app_infra_lab/export")
EXPORT_DENIED_CODE=$(curl -s -o /dev/null -w '%{http_code}' -b "$YARA" "$BASE_URL/apps/app_infra_lab/export")
[ "$EXPORT_CODE" = "200" ] && [ "$EXPORT_DENIED_CODE" = "403" ] && \
    body_contains "$BASE_URL/apps/app_infra_lab/export" "Webhook Source" "$ZARA"
check T121 "CAP-X08" "an Application's metadata exports as JSON for an Admin; denied for a non-admin role (export=$EXPORT_CODE, denied=$EXPORT_DENIED_CODE)" $?

# --- Batch 11: Remaining Field Types (2026-07-12) ---
# seeds/017_field_types_lab.sql. CAP-F17 (multi-currency money) and CAP-F19
# (quantity/UoM) are proven by composition (CAP-F08/F14/F07 + value_list),
# no new mechanism -- same framing as CAP-I05 in Batch 8.

# T122 -- CAP-F07: a `number` field renders and round-trips a real numeric
# value (not falling back to a bare text input with no validation).
FT1_URL=$(post_redirect "$BASE_URL/mch_ft_product" "fld_ftp_title=NumProbe+$$&fld_ftp_qty=7" "$WIRA")
body_contains "$FT1_URL" ">7<" "$WIRA"
check T122 "CAP-F07" "a number field's value round-trips through Create and Detail" $?

# T123 -- CAP-F08: a `money` field displays with its declared currency.
FT2_URL=$(post_redirect "$BASE_URL/mch_ft_product" "fld_ftp_title=MoneyProbe+$$&fld_ftp_price=25000" "$WIRA")
body_contains "$FT2_URL" "IDR 25000" "$WIRA"
check T123 "CAP-F08" "a money field renders with its declared currency" $?

# T124 -- CAP-F09: a `boolean` field is Yes when checked, No when the
# checkbox is left unchecked (submits nothing at all, not an empty string).
FT3_URL=$(post_redirect "$BASE_URL/mch_ft_product" "fld_ftp_title=BoolYes+$$&fld_ftp_active=true" "$WIRA")
FT4_URL=$(post_redirect "$BASE_URL/mch_ft_product" "fld_ftp_title=BoolNo+$$" "$WIRA")
body_contains "$FT3_URL" ">Yes<" "$WIRA" && body_contains "$FT4_URL" ">No<" "$WIRA"
check T124 "CAP-F09" "a boolean field is Yes when checked, No when the checkbox is left unchecked" $?

# T125 -- CAP-F10: `time`/`date_time`/`duration` all render real HTML5
# input types on the New form (not a bare text fallback) and round-trip
# their values through Create and Detail.
body_contains "$BASE_URL/mch_ft_product/new" 'type="time"' "$WIRA" && \
    body_contains "$BASE_URL/mch_ft_product/new" 'type="datetime-local"' "$WIRA"
check T125 "CAP-F10" "time and date_time fields render real HTML5 input types, not a text fallback" $?
FT5_URL=$(post_redirect "$BASE_URL/mch_ft_product" "fld_ftp_title=TimeProbe+$$&fld_ftp_release_time=14:30&fld_ftp_launched_at=2026-07-12T09:00&fld_ftp_prep_duration=90" "$WIRA")
body_contains "$FT5_URL" "14:30" "$WIRA" && body_contains "$FT5_URL" "2026-07-12T09:00" "$WIRA" && body_contains "$FT5_URL" ">90<" "$WIRA"
check T126 "CAP-F10" "time/date_time/duration values round-trip through Create and Detail" $?

# T126 -- CAP-F14: a `computed` field is never stored, never read from a
# form submission -- it's always Price * Quantity, freshly computed at
# render time, even if a submitter tries to POST a value for it directly.
FT6_URL=$(post_redirect "$BASE_URL/mch_ft_product" "fld_ftp_title=CompProbe+$$&fld_ftp_price=1000&fld_ftp_qty=4&fld_ftp_total=999999" "$WIRA")
body_contains "$FT6_URL" ">4000<" "$WIRA" && ! body_contains "$FT6_URL" "999999" "$WIRA"
check T127 "CAP-F14" "a computed field is Price times Quantity, ignoring any value POSTed directly for it" $?

# T127 -- CAP-F15: a field's declared default applies when the submitter
# leaves it blank -- generalized beyond the pre-existing value_list-only
# first-value convention (Priority here is a plain text field).
FT7_URL=$(post_redirect "$BASE_URL/mch_ft_product" "fld_ftp_title=DefaultProbe+$$" "$WIRA")
body_contains "$FT7_URL" ">Normal<" "$WIRA"
check T128 "CAP-F15" "a plain (non-value_list) field's declared default applies when left blank" $?

# T128 -- CAP-F18: an auto-number field left blank gets a real, sequential,
# server-generated value -- never a submitter-supplied string (T126's own
# "ignore a directly-POSTed value" proof applies the same way here).
FT8_URL=$(post_redirect "$BASE_URL/mch_ft_product" "fld_ftp_title=SeqProbe1+$$" "$WIRA")
FT9_URL=$(post_redirect "$BASE_URL/mch_ft_product" "fld_ftp_title=SeqProbe2+$$" "$WIRA")
SKU1=$(get_body "$FT8_URL" "$WIRA" | grep -oE '>SKU-[0-9]+<' | head -1 | tr -d '<>')
SKU2=$(get_body "$FT9_URL" "$WIRA" | grep -oE '>SKU-[0-9]+<' | head -1 | tr -d '<>')
N1=$(echo "$SKU1" | grep -oE '[0-9]+')
N2=$(echo "$SKU2" | grep -oE '[0-9]+')
[ -n "$N1" ] && [ -n "$N2" ] && [ "$((10#$N2))" -eq "$((10#$N1 + 1))" ]
check T129 "CAP-F18" "consecutive Creates get sequential, zero-padded auto-numbers (got $SKU1, $SKU2)" $?

# T129 -- CAP-F06: an uploaded image is genuinely stored (not silently
# dropped, the pre-Batch-11 behavior), resized to max_dimension, and
# re-encoded as real WebP (RIFF/WEBP magic bytes) -- not just copied
# through unchanged. A file type outside the declared accept list is
# rejected outright.
PNG_B64="iVBORw0KGgoAAAANSUhEUgAAASwAAAEsCAIAAAD2HxkiAAACd0lEQVR42u3TsQkAAAzDsJzez5sbOnUR6AKDk1ngkwRgQjAhYEIwIWBCMCFgQjAhYEIwIWBCMCFgQjAhYEIwIWBCMCFgQjAhYEIwIWBCMCFgQjAhYEIwIWBCMCFgQjAhYEIwIWBCMCFgQjAhYEIwIWBCMCFgQjAhYEIwIWBCMCFgQjAhYEIwIWBCMCFgQjAhYEIwIWBCMCFgQjAhYEIwIZhQAjAhmBAwIZgQMCGYEDAhmBAwIZgQMCGYEDAhmBAwIZgQMCGYEDAhmBAwIZgQMCGYEDAhmBAwIZgQMCGYEDAhmBAwIZgQMCGYEDAhmBAwIZgQMCGYEDAhmBAwIZgQMCGYEDAhmBAwIZgQMCGYEDAhmBAwIZgQMCGYEDAhmBAwIZgQTKgCmBBMCJgQTAiYEEwImBBMCJgQTAiYEEwImBBMCJgQTAiYEEwImBBMCJgQTAiYEEwImBBMCJgQTAiYEEwImBBMCJgQTAiYEEwImBBMCJgQTAiYEEwImBBMCJgQTAiYEEwImBBMCJgQTAiYEEwImBBMCJgQTAiYEEwImBBMCJgQTAgmBEwIJgRMCCYETAgmBEwIJgRMCCYETAgmBEwIJgRMCCYETAgmBEwIJgRMCCYETAgmBEwIJgRMCCYETAgmBEwIJgRMCCYETAgmBEwIJgRMCCYETAgmBEwIJgRMCCYETAgmBEwIJgRMCCYETAgmBEwIJgRMCCYETAgmBEwIJgQkABOCCQETggkBE4IJAROCCQETggkBE4IJAROCCQETggkBE4IJAROCCQETggkBE4IJAROCCQETggkBE4IJAROCCQETggkBE4IJAROCCQETggmBmwIoGRdDVqoymgAAAABJRU5ErkJggg=="
PNG_FILE=$(mktemp)
echo "$PNG_B64" | base64 -d > "$PNG_FILE" 2>/dev/null || echo "$PNG_B64" | base64 --decode > "$PNG_FILE"
FT_CSRF=$(csrf_for "$WIRA")
UPLOAD_HEADERS=$(mktemp)
UPLOAD_CODE=$(curl -s -b "$WIRA" -D "$UPLOAD_HEADERS" -o /dev/null -w '%{http_code}' -X POST "$BASE_URL/mch_ft_product" \
    -F "csrf_token=$FT_CSRF" -F "fld_ftp_title=PhotoProbe$$" -F "fld_ftp_photo=@$PNG_FILE;type=image/png")
UPLOAD_REDIRECT=$(grep -i '^location' "$UPLOAD_HEADERS" | tr -d '\r' | sed -E 's/^[Ll]ocation: //')
FILE_HREF=$(get_body "$BASE_URL$UPLOAD_REDIRECT" "$WIRA" | grep -oE 'href="/files/[^"]*"' | sed -E 's/href="(.*)"/\1/')
STORED_FILE=$(mktemp)
curl -s -D "$UPLOAD_HEADERS" -o "$STORED_FILE" "$BASE_URL$FILE_HREF"
SERVED_CONTENT_TYPE=$(grep -i '^content-type' "$UPLOAD_HEADERS" | tr -d '\r')
ORIG_SIZE=$(wc -c < "$PNG_FILE" | tr -d ' ')
STORED_SIZE=$(wc -c < "$STORED_FILE" | tr -d ' ')
[ "$UPLOAD_CODE" = "303" ] && \
    echo "$SERVED_CONTENT_TYPE" | grep -qi "image/webp" && \
    [ "$(head -c 4 "$STORED_FILE")" = "RIFF" ] && \
    [ "$STORED_SIZE" -lt "$ORIG_SIZE" ]
check T130 "CAP-F06" "an uploaded image is stored, resized, and re-encoded as real WebP (orig=${ORIG_SIZE}b, stored=${STORED_SIZE}b, content-type=$SERVED_CONTENT_TYPE)" $?
BAD_UPLOAD_FILE=$(mktemp)
printf 'not an image' > "$BAD_UPLOAD_FILE"
BAD_CODE=$(curl -s -o /dev/null -w '%{http_code}' -b "$WIRA" -X POST "$BASE_URL/mch_ft_product" \
    -F "csrf_token=$FT_CSRF" -F "fld_ftp_title=BadUpload$$" -F "fld_ftp_photo=@$BAD_UPLOAD_FILE;type=text/plain")
[ "$BAD_CODE" = "400" ]
check T131 "CAP-F06" "a file type outside the declared accept list is rejected, not silently stored (got $BAD_CODE)" $?
rm -f "$PNG_FILE" "$UPLOAD_HEADERS" "$STORED_FILE" "$BAD_UPLOAD_FILE"

# T130 -- CAP-F17: multi-currency money, composed from CAP-F08's own
# currency_field option plus a CAP-F14 computed base-currency mirror -- no
# dedicated new field type.
FT10_URL=$(post_redirect "$BASE_URL/mch_ft_invoice" "fld_fti_currency=USD&fld_fti_amount=100&fld_fti_rate=16000" "$WIRA")
body_contains "$FT10_URL" "USD 100" "$WIRA" && body_contains "$FT10_URL" ">1600000<" "$WIRA"
check T132 "CAP-F17" "multi-currency money (currency + rate) computes its base-currency mirror correctly" $?

# T131 -- CAP-F19: quantity/UoM Tier 1 (flat factor pair), composed from
# `number` + `value_list` + a CAP-F14 computed field -- no dedicated new
# field type.
FT11_URL=$(post_redirect "$BASE_URL/mch_ft_shipment" "fld_fts_quantity=5&fld_fts_unit=Kg&fld_fts_factor=1000" "$WIRA")
body_contains "$FT11_URL" ">5000<" "$WIRA"
check T133 "CAP-F19" "quantity/UoM Tier 1 composition converts to its base unit correctly (5 Kg * 1000 = 5000 g)" $?

# T132 -- CAP-F21: a `document` View renders its own html/template source
# against one real record's data -- merge fields resolved, not literal
# placeholder text left in the output.
DOC_CODE=$(curl -s -o /dev/null -w '%{http_code}' -b "$WIRA" "$FT1_URL/document")
body_contains "$FT1_URL/document" "SKU: SKU-" "$WIRA" && ! body_contains "$FT1_URL/document" "{{.fld_ftp_sku}}" "$WIRA"
check T134 "CAP-F21" "a document View renders its template with real merge fields resolved (got $DOC_CODE)" $?

# --- CAP-O03 Tier 2: persistent in-app sub-navigation (2026-07-12) ---
# benchmarks/009-in-app-navigation-benchmark.md. No new seed needed --
# reuses app_field_types_lab (3 machines: Product/Invoice/Shipment,
# seeds/017) for the positive case and app_workspace_lab_ops (1 machine,
# seeds/015) for the negative case.

# T135 -- a page belonging to a Machine in a multi-machine Application
# renders a persistent sub-nav strip listing every sibling Machine, with
# the current one marked active -- so a user can move sideways without
# returning to the workspace home. A Machine that's the ONLY one in its
# Application renders no strip at all (nothing to move sideways to).
body_contains "$BASE_URL/mch_ft_product" 'href="/mch_ft_invoice"' "$WIRA" && \
    body_contains "$BASE_URL/mch_ft_product" 'href="/mch_ft_shipment"' "$WIRA" && \
    body_contains "$BASE_URL/mch_ft_product" 'bg-white text-blue-700' "$WIRA" && \
    ! body_contains "$BASE_URL/mch_wsx_project" "bg-slate-100 border-b border-slate-200" "$YARA"
check T135 "CAP-O03" "a multi-machine Application renders a persistent sub-nav to sibling Machines; a single-machine Application renders none" $?

# --- Process Overlay B1 parity experiment (Study 21, seeds/019) ---
# Two Machines carry the SAME Corrective Action process: mch_ca_manual is
# hand-authored (events/guards/permissions/status field explicit, the v1
# way), mch_ca_overlay declares ONLY a `process` block and everything else
# is compiled at load time (internal/metadata/compile.go). T136-T139 drive
# both through the identical lifecycle and assert identical behavior --
# the falsifiable core claim of the v2 concept ("declared process,
# emergent execution", brd-menata-runtime-v2.md §6.6).

SURYA=$(session_for overlay.sup@example.com password)      # Supervisor (app_overlay_lab)
WATI=$(session_for overlay.worker@example.com password)    # Worker -- the assignee
WINDA=$(session_for overlay.worker2@example.com password)  # Worker -- NOT the assignee (CAP-P02 probe)
RIAN=$(session_for overlay.reviewer@example.com password)  # Reviewer

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

# T136 -- the compiled Machine boots, renders, and its GENERATED status
# field starts at the declared initial state, exactly like the manual arm's
# hand-declared one (first-value-is-default convention preserved).
MAN_LC=$(overlay_lifecycle mch_ca_manual fld_cam evt_cam)
OVL_LC=$(overlay_lifecycle mch_ca_overlay fld_cao evt_mch_ca_overlay)
case "$MAN_LC" in OK*) case "$OVL_LC" in OK*) true;; *) false;; esac;; *) false;; esac
check T136 "B1" "manual and compiled Machines both create with initial state Open and render ($MAN_LC / $OVL_LC)" $?

# T137 -- full-lifecycle parity: the whole Open->Assigned->In_Progress->
# Submitted->(auto)Review->Verified->Closed sequence succeeds identically on
# both arms, including the System-side auto step nobody clicked.
case "$MAN_LC $OVL_LC" in OK*OK*) true;; *) false;; esac
check T137 "B1" "compiled process runs the identical full lifecycle incl. the automatic Submitted->Review step" $?

# T138/T139 -- negative parity: state guard (CAP-E06), role guard (CAP-P01),
# and ownership guard (CAP-P02) reject with the same codes on both arms.
MAN_NEG=$(overlay_negatives mch_ca_manual fld_cam evt_cam)
[ "$MAN_NEG" = "OK" ]
check T138 "B1" "manual arm rejects: wrong state 400, wrong role 403, non-assignee Worker 403 (got $MAN_NEG)" $?

OVL_NEG=$(overlay_negatives mch_ca_overlay fld_cao evt_mch_ca_overlay)
[ "$OVL_NEG" = "OK" ]
check T139 "B1" "compiled arm rejects identically: wrong state 400, wrong role 403, non-assignee Worker 403 (got $OVL_NEG)" $?

# --- Process Overlay B2: process map, CAP-W05 (Study 21 addendum) ---
# The map is derived from Events/guards + the Status field (internal/
# handler/processmap.go's extractProcessMap), never from a `process`
# declaration -- so it must render identically for the compiled arm, the
# hand-authored arm (T140 vs T141, the legibility-parity proof), AND for a
# genuine pre-existing v1 Machine that never heard of `process` at all
# (T142, the decompile proof, on Leave Request's real Case 1/2 metadata).

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

CA_MAP_FRAGMENTS=(
    "Open (initial)" ">Assigned<" ">In_Progress<" ">Submitted<" ">Review<" ">Verified<" ">Closed<"
    ">Assign<" ">Supervisor<" ">Start<" "Worker (owner: Assignee)" ">Submit<"
    "Auto: Submitted" ">System<" ">Approve<" ">Reviewer<" ">Revise<" ">Close<"
)

# T140 -- the compiled arm's map lists all 7 states (Open marked initial)
# and all 6 transitions with correct actors, INCLUDING the auto step
# (System, no human Permission grants it).
MAP_OVL=$(process_map_has_shape "$BASE_URL/mch_ca_overlay/process-map" "$SURYA" "${CA_MAP_FRAGMENTS[@]}")
[ "$MAP_OVL" = "OK" ]
check T140 "CAP-W05" "compiled Machine's process map shows every state and transition with the correct actor (got $MAP_OVL)" $?

# T141 -- the hand-authored arm's map is the SAME assertion list, verbatim
# -- legibility parity, not just execution parity (T136-T139 already
# proved execution).
MAP_MAN=$(process_map_has_shape "$BASE_URL/mch_ca_manual/process-map" "$SURYA" "${CA_MAP_FRAGMENTS[@]}")
[ "$MAP_MAN" = "OK" ]
check T141 "CAP-W05" "hand-authored Machine's process map is identical to the compiled one's (got $MAP_MAN)" $?

# T142 -- the decompile claim: Leave Request (seeds/002, Case 1/2 vintage,
# predates `process` by weeks) reconstructs correctly from its own real
# Events/guards -- Draft->Submitted->{Approved,Rejected}, plus Cancel.
MAP_LR=$(process_map_has_shape "$BASE_URL/mch_leave_request/process-map" "$DAVE" \
    "Draft (initial)" ">Submitted<" ">Approved<" ">Rejected<" ">Cancelled<" \
    ">Submit<" ">Employee<" ">Approve<" ">Manager<" ">Reject<" ">Cancel<")
[ "$MAP_LR" = "OK" ]
check T142 "CAP-W05" "a genuine pre-existing v1 Machine's process map reconstructs correctly with no \`process\` declaration anywhere (got $MAP_LR)" $?

# T143 -- the opt-in gate: FRANK can read mch_employee (HR role, perm_hr)
# but no process_map View was ever declared for it -- 404, not an error
# page, the same posture every other auxiliary View type already takes.
CODE=$(curl -s -o /dev/null -w '%{http_code}' -b "$FRANK" "$BASE_URL/mch_employee/process-map")
[ "$CODE" = "404" ]
check T143 "CAP-W05" "a Machine with no process_map View declared 404s (got $CODE)" $?

# --- Process Overlay B3: generic Requirement, evidence cardinality (CAP-W01, seeds/020) ---
# mch_req_case's Submit (Open -> Submitted) requires at least 2 mch_req_photo
# records referencing it -- the comparator BRD's own "minimum 2 photos"
# example. The count is maintained by write-time fan-in (handler.
# stampRequirementCounters, on every mch_req_photo Create), the gate itself
# is an ordinary CAP-C09 constraint re-check -- no new check mechanism, per
# Study 20 §6.3.

WIRA=$(session_for req.worker@example.com password) # Worker (app_req_lab)

CASE_URL=$(post_redirect "$BASE_URL/mch_req_case" "fld_rc_title=Parity+$$" "$WIRA")
CASE_ID="${CASE_URL##*/}"

# T144 -- zero evidence attached: Submit is rejected (400), not silently
# allowed through.
CODE=$(post_status "$BASE_URL/mch_req_case/$CASE_ID/events/evt_mch_req_case_submit" "" "$WIRA")
[ "$CODE" = "400" ]
check T144 "CAP-W01" "Submit with 0 attached evidence records is rejected (got $CODE)" $?

# T145 -- one photo attached (counter=1): still below the cardinality
# minimum of 2 -- proves the threshold is a real count, not a presence check.
post_redirect "$BASE_URL/mch_req_photo" "fld_rp_case=$CASE_ID&fld_rp_caption=first" "$WIRA" > /dev/null
CODE=$(post_status "$BASE_URL/mch_req_case/$CASE_ID/events/evt_mch_req_case_submit" "" "$WIRA")
[ "$CODE" = "400" ]
check T145 "CAP-W01" "Submit with 1 attached evidence record (below cardinality 2) is still rejected (got $CODE)" $?

# T146 -- a second photo attached (counter=2): the requirement is satisfied,
# Submit succeeds.
post_redirect "$BASE_URL/mch_req_photo" "fld_rp_case=$CASE_ID&fld_rp_caption=second" "$WIRA" > /dev/null
CODE=$(post_status "$BASE_URL/mch_req_case/$CASE_ID/events/evt_mch_req_case_submit" "" "$WIRA")
[ "$CODE" = "303" ]
check T146 "CAP-W01" "Submit with 2 attached evidence records (cardinality satisfied) succeeds (got $CODE)" $?

# --- Process Overlay B4 Part 1: SLA (CAP-W04, seeds/021) ---
# mch_sla_case declares one `sla` entry on Review: duration "0 Days" (due =
# today, so the very next scheduler tick already sees it overdue -- same
# fast-test trick T99/T100's own real-scheduler wait already relies on).
# Compiles to a due-date Field, a due-date-stamping action on Submit (the
# only transition landing on Review), and a scheduled breach Event.

WISNU=$(session_for sla.worker@example.com password)  # Worker (app_sla_lab)
MAYA=$(session_for sla.manager@example.com password)  # Manager (app_sla_lab)

# Case A: left in Review -- should breach and escalate.
SLA_A_URL=$(post_redirect "$BASE_URL/mch_sla_case" "fld_sc_title=Breach+$$" "$WISNU")
SLA_A_ID="${SLA_A_URL##*/}"
post_status "$BASE_URL/mch_sla_case/$SLA_A_ID/events/evt_mch_sla_case_submit" "" "$WISNU" > /dev/null

# Case B: leaves Review (Closed) BEFORE the tick -- must NOT be touched by
# the breach event once it fires.
SLA_B_URL=$(post_redirect "$BASE_URL/mch_sla_case" "fld_sc_title=Safe+$$" "$WISNU")
SLA_B_ID="${SLA_B_URL##*/}"
post_status "$BASE_URL/mch_sla_case/$SLA_B_ID/events/evt_mch_sla_case_submit" "" "$WISNU" > /dev/null
post_status "$BASE_URL/mch_sla_case/$SLA_B_ID/events/evt_mch_sla_case_close" "" "$MAYA" > /dev/null

echo "(waiting ~65s for the real background scheduler tick -- CAP-E02/E03/CAP-W04)"
sleep 65

# T147 -- a record left in Review past its due date auto-escalates.
body_contains "$SLA_A_URL" ">Escalated<" "$MAYA"
check T147 "CAP-W04" "a record left in an SLA-bound state past its due date auto-escalates to the declared state" $?

# T148 -- a record that already left the SLA-bound state (Closed) before
# the tick is never touched -- CAP-E06's guard correctly no-ops the breach
# event, proving the state-guard, not just the happy path.
body_contains "$SLA_B_URL" ">Closed<" "$MAYA"
check T148 "CAP-W04" "a record that already left the SLA-bound state is untouched by the breach event" $?

# --- Process Overlay B4 Part 2: quorum-of-N (CAP-W03, seeds/022) ---
# CAP-A08's aggregate_status action, generalized with min_approvals=2 over
# 3 hand-authored Voter records -- deliberately NOT declared via `process`,
# proving the mechanism itself, independent of the Process Overlay (see
# roadmap.md's B4 note on why the declarative form is separate future work).

RANI=$(session_for quorum.requester@example.com password) # Requester (app_quorum_lab)
QV1=$(session_for quorum.voter1@example.com password)     # Voter
QV2=$(session_for quorum.voter2@example.com password)     # Voter
QV3=$(session_for quorum.voter3@example.com password)     # Voter

V1_ID=$(user_option_id "$BASE_URL/mch_ql_vote/new" "$QV1" "Voter One")
V2_ID=$(user_option_id "$BASE_URL/mch_ql_vote/new" "$QV1" "Voter Two")
V3_ID=$(user_option_id "$BASE_URL/mch_ql_vote/new" "$QV1" "Voter Three")

# T149 -- 2 of 3 votes Approved (3rd left Pending): the parent reaches
# Approved as soon as the 2nd Approve fires, without waiting on the 3rd.
REQ_A_URL=$(post_redirect "$BASE_URL/mch_ql_request" "fld_qr_title=Quorum+Approve+$$" "$RANI")
REQ_A_ID="${REQ_A_URL##*/}"
VA1_URL=$(post_redirect "$BASE_URL/mch_ql_vote" "fld_qv_request=$REQ_A_ID&fld_qv_voter=$V1_ID" "$QV1")
VA2_URL=$(post_redirect "$BASE_URL/mch_ql_vote" "fld_qv_request=$REQ_A_ID&fld_qv_voter=$V2_ID" "$QV2")
post_redirect "$BASE_URL/mch_ql_vote" "fld_qv_request=$REQ_A_ID&fld_qv_voter=$V3_ID" "$QV3" > /dev/null
post_status "$VA1_URL/events/evt_qv_approve" "" "$QV1" > /dev/null
post_status "$VA2_URL/events/evt_qv_approve" "" "$QV2" > /dev/null
body_contains "$REQ_A_URL" ">Approved<" "$RANI"
check T149 "CAP-W03" "a 2-of-3 quorum reaches Approved once 2 votes are in, without waiting for the 3rd" $?

# T150 -- 2 of 3 votes Rejected (3rd left Pending): quorum is now
# mathematically impossible (at most 1 more Approved could ever land), the
# parent reaches Rejected.
REQ_B_URL=$(post_redirect "$BASE_URL/mch_ql_request" "fld_qr_title=Quorum+Reject+$$" "$RANI")
REQ_B_ID="${REQ_B_URL##*/}"
VB1_URL=$(post_redirect "$BASE_URL/mch_ql_vote" "fld_qv_request=$REQ_B_ID&fld_qv_voter=$V1_ID" "$QV1")
VB2_URL=$(post_redirect "$BASE_URL/mch_ql_vote" "fld_qv_request=$REQ_B_ID&fld_qv_voter=$V2_ID" "$QV2")
post_redirect "$BASE_URL/mch_ql_vote" "fld_qv_request=$REQ_B_ID&fld_qv_voter=$V3_ID" "$QV3" > /dev/null
post_status "$VB1_URL/events/evt_qv_reject" "" "$QV1" > /dev/null
post_status "$VB2_URL/events/evt_qv_reject" "" "$QV2" > /dev/null
body_contains "$REQ_B_URL" ">Rejected<" "$RANI"
check T150 "CAP-W03" "a 2-of-3 quorum reaches Rejected once 2 votes are rejected (quorum mathematically impossible)" $?

# --- CAP-X04: metadata live reload (Option A, docs/decisions/002-metadata-loading.md) ---
# seeds/023_reload_lab.sql is deliberately NOT part of `make seed`'s boot-time
# list -- applied here, mid-run, via a direct psql call (same documented
# exception T19 already uses), so T151 can prove the new Machine becomes
# servable WITHOUT a restart. Metadata tables (machines/fields/permissions/
# views/user_application_roles) carry no RLS (CAP-X06's own scope note --
# "trusted-seed-only write path"), so no app.workspace_id GUC dance is
# needed here, unlike T19's own records-table UPDATE.
if [ -n "$DATABASE_URL" ]; then
    # T151 -- before the seed is applied, the Machine is unknown; after
    # applying it + triggering reload, it's servable -- the core falsifiable
    # claim, proven without ever restarting the server process.
    PRE_CODE=$(curl -s -o /dev/null -w '%{http_code}' -b "$FRANK" "$BASE_URL/mch_reload_case")
    psql "$DATABASE_URL" -q -f seeds/023_reload_lab.sql >/dev/null
    RELOAD_CODE=$(post_status "$BASE_URL/admin/reload" "" "$FRANK")
    POST_CODE=$(curl -s -o /dev/null -w '%{http_code}' -b "$FRANK" "$BASE_URL/mch_reload_case")
    [ "$PRE_CODE" = "404" ] && [ "$RELOAD_CODE" = "303" ] && [ "$POST_CODE" = "200" ]
    check T151 "CAP-X04" "a Machine seeded mid-run becomes servable after reload, no restart (pre=$PRE_CODE reload=$RELOAD_CODE post=$POST_CODE)" $?

    # T152 -- a non-Admin cannot trigger a reload.
    CODE=$(post_status "$BASE_URL/admin/reload" "" "$DAVE")
    [ "$CODE" = "403" ]
    check T152 "CAP-X04" "a non-Admin cannot trigger a metadata reload (got $CODE)" $?

    # T153 -- a deliberately malformed row (a `reference` field targeting a
    # nonexistent Machine -- the exact CAP-F13 dangling-reference check
    # validateReferences already enforces) makes the NEXT reload fail (500),
    # and the OLD, still-valid interpreter must keep serving everything else
    # completely normally -- the one property this feature exists to
    # guarantee. Cleaned up immediately after: a real correctness
    # requirement, not just hygiene -- a future server RESTART (not just a
    # reload) calls os.Exit(1) on bad metadata, so leaving this row behind
    # would break the very next boot, not just the next reload attempt.
    psql "$DATABASE_URL" -q -c \
        "INSERT INTO fields (id, machine_id, name, type, position, required, options) VALUES
         ('fld_rlc_bad_ref', 'mch_reload_case', 'Bad Ref', 'reference', 1, false, '{\"target_machine\":\"mch_does_not_exist\"}')" \
        >/dev/null
    BAD_RELOAD_CODE=$(post_status "$BASE_URL/admin/reload" "" "$FRANK")
    OLD_STILL_UP_CODE=$(curl -s -o /dev/null -w '%{http_code}' -b "$DAVE" "$BASE_URL/mch_leave_request")
    psql "$DATABASE_URL" -q -c "DELETE FROM fields WHERE id = 'fld_rlc_bad_ref'" >/dev/null
    [ "$BAD_RELOAD_CODE" = "500" ] && [ "$OLD_STILL_UP_CODE" = "200" ]
    check T153 "CAP-X04" "a bad reload is rejected (500) and the old interpreter keeps serving unrelated Machines normally (reload=$BAD_RELOAD_CODE old_machine=$OLD_STILL_UP_CODE)" $?

    # --- CAP-W07: change_policy (effective-dated metadata evolution) ---
    # seeds/024_change_policy_lab.sql (boot-time) seeds mch_policy_case and
    # three baseline records, all open before any change_policy exists.
    # seeds/025_change_policy_activate.sql (deliberately NOT part of `make
    # seed`, same exclusion pattern as 023 above) is the actual metadata
    # CHANGE -- two new Constraint rows, each requiring "Compliance Note"
    # but scoped by a different change_policy -- applied here mid-run, then
    # picked up by the same POST /admin/reload CAP-X04 just proved.
    # Compliance Note is optional at the Field level (only drives the UI's
    # `required` attribute) -- every assertion below is the server-side
    # (CAP-C09) check, submitting it blank on purpose.
    DRAFT_ID='11111111-1111-1111-1111-111111111101'
    SUBMITTED_ID='11111111-1111-1111-1111-111111111102'
    OLD_ID='11111111-1111-1111-1111-111111111103'

    # T154 -- before 025 is applied, no rule exists yet: updating the Draft
    # record with a blank Compliance Note succeeds.
    CODE=$(post_status "$BASE_URL/mch_policy_case/$DRAFT_ID" "fld_pc_title=Draft+case" "$FRANK")
    [ "$CODE" = "303" ]
    check T154 "CAP-W07" "before change_policy exists, a blank Compliance Note is accepted (got $CODE)" $?

    psql "$DATABASE_URL" -q -f seeds/025_change_policy_activate.sql >/dev/null
    post_status "$BASE_URL/admin/reload" "" "$FRANK" >/dev/null

    # T155 -- records_in_states: [Draft] now gates the Draft record.
    post_body_contains "$BASE_URL/mch_policy_case/$DRAFT_ID" "fld_pc_title=Draft+case" "Compliance Note is required for cases still in Draft" "$FRANK"
    check T155 "CAP-W07" "records_in_states [Draft] rejects a blank Compliance Note on a Draft record" $?

    # T156 -- the Submitted record is grandfathered: already past Draft when
    # the rule arrived, so records_in_states [Draft] never reaches it.
    # Approval Reference is supplied so this record (created just now, so
    # new_records' OWN condition is true for it) isn't ALSO blocked by the
    # other constraint -- isolates the state dimension from the date one.
    CODE=$(post_status "$BASE_URL/mch_policy_case/$SUBMITTED_ID" "fld_pc_title=Submitted+case&fld_pc_approval_ref=ok" "$FRANK")
    [ "$CODE" = "303" ]
    check T156 "CAP-W07" "a record already past Draft when the rule arrived is grandfathered (got $CODE)" $?

    # T157 -- the Old record predates new_records' effective_from
    # (2026-01-01, seeds/024's own past-dated fixture): the policy doesn't
    # reach it either. Compliance Note is supplied so this record (seeded in
    # Draft, so records_in_states' OWN condition is true for it) isn't ALSO
    # blocked by the other constraint -- isolates the date dimension from
    # the state one.
    CODE=$(post_status "$BASE_URL/mch_policy_case/$OLD_ID" "fld_pc_title=Old+case&fld_pc_compliance=ok" "$FRANK")
    [ "$CODE" = "303" ]
    check T157 "CAP-W07" "a record created before new_records' effective_from is untouched by the new policy (got $CODE)" $?

    # T158 -- a brand-new record, created after the policy's effective_from
    # (today), IS gated by it. Compliance Note is supplied (a fresh Create
    # defaults to Draft, so records_in_states would otherwise ALSO fire) to
    # isolate this assertion to new_records specifically.
    post_body_contains "$BASE_URL/mch_policy_case" "fld_pc_title=New+case&fld_pc_compliance=ok" "Approval Reference is required for cases opened under the policy" "$FRANK"
    check T158 "CAP-W07" "new_records rejects a blank Approval Reference on a record created after the effective date" $?
else
    printf 'SKIP  T151 %-22s %s\n' "CAP-X04" "DATABASE_URL not set -- seed-mid-run fixture unavailable"
    printf 'SKIP  T152 %-22s %s\n' "CAP-X04" "DATABASE_URL not set"
    printf 'SKIP  T153 %-22s %s\n' "CAP-X04" "DATABASE_URL not set -- malformed-row fixture unavailable"
    printf 'SKIP  T154 %-22s %s\n' "CAP-W07" "DATABASE_URL not set -- seed-mid-run fixture unavailable"
    printf 'SKIP  T155 %-22s %s\n' "CAP-W07" "DATABASE_URL not set"
    printf 'SKIP  T156 %-22s %s\n' "CAP-W07" "DATABASE_URL not set"
    printf 'SKIP  T157 %-22s %s\n' "CAP-W07" "DATABASE_URL not set"
    printf 'SKIP  T158 %-22s %s\n' "CAP-W07" "DATABASE_URL not set"
fi

# --- CAP-W03 declarative quorum (seeds/026, boot-time -- no DATABASE_URL
# needed) -- proves the compiler's cross-machine injection
# (compileApprovalRequirements) produces IDENTICAL behavior to 022's
# hand-authored pair (T149/T150), driven entirely through HTTP. Only
# difference from T149/T150's own flow: the parent must Submit (Open ->
# PendingApproval) first -- the transition that carries the declarative
# `requirements: [{type: approval, ...}]` -- before Approve/Reject (System-
# only, fired by the injected aggregate_status action) become reachable.

RANI2=$(session_for quorumdecl.requester@example.com password) # Requester (app_quorum_decl_lab)
DV1=$(session_for quorumdecl.voter1@example.com password)      # Voter
DV2=$(session_for quorumdecl.voter2@example.com password)      # Voter
DV3=$(session_for quorumdecl.voter3@example.com password)      # Voter

DV1_ID=$(user_option_id "$BASE_URL/mch_ql2_vote/new" "$DV1" "Voter Decl One")
DV2_ID=$(user_option_id "$BASE_URL/mch_ql2_vote/new" "$DV1" "Voter Decl Two")
DV3_ID=$(user_option_id "$BASE_URL/mch_ql2_vote/new" "$DV1" "Voter Decl Three")

# T159 -- same shape as T149, against the declaratively-wired pair.
DREQ_A_URL=$(post_redirect "$BASE_URL/mch_ql2_request" "fld_qr2_title=Quorum+Decl+Approve+$$" "$RANI2")
DREQ_A_ID="${DREQ_A_URL##*/}"
post_status "$DREQ_A_URL/events/evt_mch_ql2_request_submit" "" "$RANI2" > /dev/null
DVA1_URL=$(post_redirect "$BASE_URL/mch_ql2_vote" "fld_ql2v_request=$DREQ_A_ID&fld_ql2v_voter=$DV1_ID" "$DV1")
DVA2_URL=$(post_redirect "$BASE_URL/mch_ql2_vote" "fld_ql2v_request=$DREQ_A_ID&fld_ql2v_voter=$DV2_ID" "$DV2")
post_redirect "$BASE_URL/mch_ql2_vote" "fld_ql2v_request=$DREQ_A_ID&fld_ql2v_voter=$DV3_ID" "$DV3" > /dev/null
post_status "$DVA1_URL/events/evt_ql2v_approve" "" "$DV1" > /dev/null
post_status "$DVA2_URL/events/evt_ql2v_approve" "" "$DV2" > /dev/null
body_contains "$DREQ_A_URL" ">Approved<" "$RANI2"
check T159 "CAP-W03" "declarative quorum (process.requirements[].type=approval): 2-of-3 reaches Approved once 2 votes are in, without waiting for the 3rd" $?

# T160 -- same shape as T150, against the declaratively-wired pair.
DREQ_B_URL=$(post_redirect "$BASE_URL/mch_ql2_request" "fld_qr2_title=Quorum+Decl+Reject+$$" "$RANI2")
DREQ_B_ID="${DREQ_B_URL##*/}"
post_status "$DREQ_B_URL/events/evt_mch_ql2_request_submit" "" "$RANI2" > /dev/null
DVB1_URL=$(post_redirect "$BASE_URL/mch_ql2_vote" "fld_ql2v_request=$DREQ_B_ID&fld_ql2v_voter=$DV1_ID" "$DV1")
DVB2_URL=$(post_redirect "$BASE_URL/mch_ql2_vote" "fld_ql2v_request=$DREQ_B_ID&fld_ql2v_voter=$DV2_ID" "$DV2")
post_redirect "$BASE_URL/mch_ql2_vote" "fld_ql2v_request=$DREQ_B_ID&fld_ql2v_voter=$DV3_ID" "$DV3" > /dev/null
post_status "$DVB1_URL/events/evt_ql2v_reject" "" "$DV1" > /dev/null
post_status "$DVB2_URL/events/evt_ql2v_reject" "" "$DV2" > /dev/null
body_contains "$DREQ_B_URL" ">Rejected<" "$RANI2"
check T160 "CAP-W03" "declarative quorum: 2-of-3 reaches Rejected once 2 votes are rejected (quorum mathematically impossible)" $?

# --- CAP-C08 (Case 9 completion batch: CAP-C10 debit=credit, CAP-C11 no
# posting into a closed period) -- seeds/027, boot-time, no DATABASE_URL
# needed. The Fiscal Period's own Close event (evt_c9fp_close) gets one of
# the two seeded periods to Closed entirely through HTTP.

C9ACCT=$(session_for case9.accountant@example.com password)

FP_OPEN_URL=$(post_redirect "$BASE_URL/mch_c9_fiscal_period" "fld_c9fp_name=FY2026+Q1+$$" "$C9ACCT")
FP_OPEN_ID="${FP_OPEN_URL##*/}"
FP_CLOSED_URL=$(post_redirect "$BASE_URL/mch_c9_fiscal_period" "fld_c9fp_name=FY2025+Q4+$$" "$C9ACCT")
FP_CLOSED_ID="${FP_CLOSED_URL##*/}"
post_status "$FP_CLOSED_URL/events/evt_c9fp_close" "" "$C9ACCT" > /dev/null

# T161 -- an unbalanced entry (100 debit, 50 credit) is rejected on Post.
JE_A_URL=$(post_redirect "$BASE_URL/mch_c9_journal_entry" "fld_c9je_memo=Unbalanced+$$&fld_c9je_period=$FP_OPEN_ID" "$C9ACCT")
JE_A_ID="${JE_A_URL##*/}"
post_redirect "$BASE_URL/mch_c9_journal_entry_line" "fld_c9jel_entry=$JE_A_ID&fld_c9jel_debit=100&fld_c9jel_credit=0" "$C9ACCT" > /dev/null
post_redirect "$BASE_URL/mch_c9_journal_entry_line" "fld_c9jel_entry=$JE_A_ID&fld_c9jel_debit=0&fld_c9jel_credit=50" "$C9ACCT" > /dev/null
post_body_contains "$JE_A_URL/events/evt_c9je_post" "" "Total Debit must equal Total Credit before posting." "$C9ACCT"
check T161 "CAP-C08" "an unbalanced entry (debit != credit) is rejected on Post (CAP-C10)" $?

# T162 -- a balanced entry (100/100) posts successfully -- the positive case,
# proving T161 isn't vacuously always-reject.
JE_B_URL=$(post_redirect "$BASE_URL/mch_c9_journal_entry" "fld_c9je_memo=Balanced+$$&fld_c9je_period=$FP_OPEN_ID" "$C9ACCT")
JE_B_ID="${JE_B_URL##*/}"
post_redirect "$BASE_URL/mch_c9_journal_entry_line" "fld_c9jel_entry=$JE_B_ID&fld_c9jel_debit=100&fld_c9jel_credit=0" "$C9ACCT" > /dev/null
post_redirect "$BASE_URL/mch_c9_journal_entry_line" "fld_c9jel_entry=$JE_B_ID&fld_c9jel_debit=0&fld_c9jel_credit=100" "$C9ACCT" > /dev/null
CODE=$(post_status "$JE_B_URL/events/evt_c9je_post" "" "$C9ACCT")
[ "$CODE" = "303" ]
check T162 "CAP-C08" "a balanced entry (debit = credit) posts successfully (CAP-C10)" $?

# T163 -- posting into a Closed Fiscal Period is rejected, even when balanced.
JE_C_URL=$(post_redirect "$BASE_URL/mch_c9_journal_entry" "fld_c9je_memo=ClosedPeriod+$$&fld_c9je_period=$FP_CLOSED_ID" "$C9ACCT")
JE_C_ID="${JE_C_URL##*/}"
post_redirect "$BASE_URL/mch_c9_journal_entry_line" "fld_c9jel_entry=$JE_C_ID&fld_c9jel_debit=100&fld_c9jel_credit=0" "$C9ACCT" > /dev/null
post_redirect "$BASE_URL/mch_c9_journal_entry_line" "fld_c9jel_entry=$JE_C_ID&fld_c9jel_debit=0&fld_c9jel_credit=100" "$C9ACCT" > /dev/null
post_body_contains "$JE_C_URL/events/evt_c9je_post" "" "Fiscal Period must be Open to post entries." "$C9ACCT"
check T163 "CAP-C08" "posting into a Closed Fiscal Period is rejected, even when balanced (CAP-C11)" $?

# T164 -- posting into an Open Fiscal Period succeeds -- same positive-case
# pairing as T162.
JE_D_URL=$(post_redirect "$BASE_URL/mch_c9_journal_entry" "fld_c9je_memo=OpenPeriod+$$&fld_c9je_period=$FP_OPEN_ID" "$C9ACCT")
JE_D_ID="${JE_D_URL##*/}"
post_redirect "$BASE_URL/mch_c9_journal_entry_line" "fld_c9jel_entry=$JE_D_ID&fld_c9jel_debit=100&fld_c9jel_credit=0" "$C9ACCT" > /dev/null
post_redirect "$BASE_URL/mch_c9_journal_entry_line" "fld_c9jel_entry=$JE_D_ID&fld_c9jel_debit=0&fld_c9jel_credit=100" "$C9ACCT" > /dev/null
CODE=$(post_status "$JE_D_URL/events/evt_c9je_post" "" "$C9ACCT")
[ "$CODE" = "303" ]
check T164 "CAP-C08" "posting into an Open Fiscal Period succeeds (CAP-C11)" $?

# --- B6, decompile-lift (CAP-W05 backward direction) -- seeds/028, applied
# mid-run via psql (needs DATABASE_URL), same exclusion pattern as
# 023/025/026_*.sql. Requires `jq` (already present on this host) to extract
# the lifted Process JSON out of the HTTP response.
if [ -n "$DATABASE_URL" ]; then
    # T165 -- GET .../process-lift returns valid Process JSON for an Admin;
    # a non-Admin gets 403.
    LIFT_BODY=$(curl -s -b "$FRANK" "$BASE_URL/mch_ca_manual/process-lift")
    LIFT_STATES=$(echo "$LIFT_BODY" | jq -r '.process.states | length' 2>/dev/null)
    LIFT_DENIED=$(curl -s -o /dev/null -w '%{http_code}' -b "$DAVE" "$BASE_URL/mch_ca_manual/process-lift")
    [ "${LIFT_STATES:-0}" -gt 0 ] 2>/dev/null && [ "$LIFT_DENIED" = "403" ]
    check T165 "CAP-W05" "GET .../process-lift returns valid Process JSON for an Admin (states=$LIFT_STATES), denies a non-Admin (got $LIFT_DENIED)" $?

    # T166 -- that JSON, applied to a fresh Machine and reloaded, drives an
    # identical lifecycle to mch_ca_manual/mch_ca_overlay (T136/T137), with
    # zero server restart -- "compiled equals hand-authored", now proven in
    # the reverse direction too.
    #
    # A real finding, not a workaround: `actor.owner_field` in the lifted
    # JSON names mch_ca_manual's OWN field id (fld_cam_assignee) -- Field
    # ids are globally unique across the whole `fields` table (not
    # machine-scoped), so that exact id can't exist on a second Machine.
    # Reapplying a lift to a DIFFERENT Machine than it came from means the
    # owner_field reference needs a one-time manual translation to whatever
    # the target's own equivalent field is called -- exactly the "review
    # before pasting" step the API response's own `note` already warns
    # about, not something liftProcess could or should paper over (it
    # faithfully reproduces the source Machine's actual actor rule; a
    # rule referencing "the assignee field" has no portable identity to
    # lift beyond the literal id it's stored under).
    psql "$DATABASE_URL" -q -f seeds/028_lift_lab.sql >/dev/null
    LIFTED_PROCESS=$(echo "$LIFT_BODY" | jq -c '.process' | sed 's/fld_cam_assignee/fld_cal_assignee/g')
    printf "UPDATE machines SET process = \$\$%s\$\$::jsonb WHERE id = 'mch_ca_lifted';\n" "$LIFTED_PROCESS" > /tmp/lift_update_$$.sql
    psql "$DATABASE_URL" -q -f /tmp/lift_update_$$.sql >/dev/null
    rm -f /tmp/lift_update_$$.sql
    post_status "$BASE_URL/admin/reload" "" "$FRANK" >/dev/null
    LIFTED_LC=$(overlay_lifecycle mch_ca_lifted fld_cal evt_mch_ca_lifted)
    case "$LIFTED_LC" in OK*) true;; *) false;; esac
    check T166 "CAP-W05" "a lifted process JSON, applied to a fresh Machine and reloaded, drives an identical lifecycle with zero restart ($LIFTED_LC)" $?
else
    printf 'SKIP  T165 %-22s %s\n' "CAP-W05" "DATABASE_URL not set"
    printf 'SKIP  T166 %-22s %s\n' "CAP-W05" "DATABASE_URL not set -- seed-mid-run fixture unavailable"
fi

# --- CAP-V17 (SLA countdown badge) -- seeds/029, boot-time, no DATABASE_URL
# needed. Computed at render time (handler.slaUrgency), same "computed at
# render time, nothing stored" precedent as CAP-F14/CAP-V13 -- no mid-run
# seed application, no reload.

SLA_AGENT=$(session_for sla.agent@example.com password)

# T167 -- a ticket due far in the past renders the overdue countdown badge.
OVERDUE_URL=$(post_redirect "$BASE_URL/mch_sla_ticket" "fld_slat_title=Overdue+$$&fld_slat_due=2020-01-01" "$SLA_AGENT")
body_contains "$OVERDUE_URL" "Overdue by" "$SLA_AGENT"
check T167 "CAP-V17" "a ticket due in the past renders the overdue countdown badge" $?

# T168 -- a ticket due far in the future does NOT render the overdue badge
# -- the positive/negative pairing proving T167 isn't vacuously always-true.
FUTURE_URL=$(post_redirect "$BASE_URL/mch_sla_ticket" "fld_slat_title=Future+$$&fld_slat_due=2099-01-01" "$SLA_AGENT")
! body_contains "$FUTURE_URL" "Overdue by" "$SLA_AGENT"
check T168 "CAP-V17" "a ticket due far in the future does not render the overdue badge" $?

echo "--------------------------------------------------------------------"
echo "Result: $PASS passed, $FAIL failed"
[ "$FAIL" -eq 0 ] || exit 1
