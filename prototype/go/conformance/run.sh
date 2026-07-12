#!/usr/bin/env bash
# Menata Runtime — Conformance Suite
# Study 4 deliverable (runtime/capability-roadmap.md)
#
# Each test proves one or more capabilities from runtime/capability-registry.md.
# A capability marked ✅ in the registry must keep its test passing (ratchet rule).
#
# Usage:
#   ./conformance/run.sh                     # against http://localhost:4000
#   BASE_URL=https://aksi.menata.id ./conformance/run.sh
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

echo "--------------------------------------------------------------------"
echo "Result: $PASS passed, $FAIL failed"
[ "$FAIL" -eq 0 ] || exit 1
