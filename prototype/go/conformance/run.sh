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
# Requires: seeds 001-003 applied, server running.
# Note: creates test records in the target database (prototype-acceptable).
# T19 is a deliberate, documented exception to "HTTP black-box": it uses psql
# to backdate a date field, simulating time passing without waiting years —
# a test fixture setup, not an inspection of runtime behavior. Set
# DATABASE_URL (same value as the server's .env) to enable it; skipped if unset.

set -u
BASE_URL="${BASE_URL:-http://localhost:4000}"
DATABASE_URL="${DATABASE_URL:-}"
PASS=0
FAIL=0

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

body_contains() { # <url> <needle> [cookie]
    local url="$1" needle="$2" cookie="${3:-}"
    if [ -n "$cookie" ]; then
        curl -s -H "Cookie: $cookie" "$url" | grep -q "$needle"
    else
        curl -s "$url" | grep -q "$needle"
    fi
}

post_body_contains() { # <url> <data> <needle> [cookie]
    local url="$1" data="$2" needle="$3" cookie="${4:-}"
    if [ -n "$cookie" ]; then
        curl -s -X POST -H "Cookie: $cookie" "$url" -d "$data" | grep -q "$needle"
    else
        curl -s -X POST "$url" -d "$data" | grep -q "$needle"
    fi
}

post_status() { # <url> <data> [cookie] -> echoes http code
    local url="$1" data="$2" cookie="${3:-}"
    if [ -n "$cookie" ]; then
        curl -s -o /dev/null -w '%{http_code}' -X POST -H "Cookie: $cookie" "$url" -d "$data"
    else
        curl -s -o /dev/null -w '%{http_code}' -X POST "$url" -d "$data"
    fi
}

post_redirect() { # <url> <data> [cookie] -> echoes redirect url
    local url="$1" data="$2" cookie="${3:-}"
    if [ -n "$cookie" ]; then
        curl -s -o /dev/null -w '%{redirect_url}' -X POST -H "Cookie: $cookie" "$url" -d "$data"
    else
        curl -s -o /dev/null -w '%{redirect_url}' -X POST "$url" -d "$data"
    fi
}

get_body() { # <url> [cookie] -> echoes response body
    local url="$1" cookie="${2:-}"
    if [ -n "$cookie" ]; then
        curl -s -H "Cookie: $cookie" "$url"
    else
        curl -s "$url"
    fi
}

echo "Menata Runtime Conformance Suite"
echo "Target: $BASE_URL"
echo "--------------------------------------------------------------------"

# T00 — server reachable
curl -s -o /dev/null --max-time 5 "$BASE_URL/health"
check T00 "—" "server /health reachable" $?
[ "$FAIL" -gt 0 ] && { echo "Server unreachable — aborting."; exit 1; }

# T01 — CAP-X01 multi-application, multi-machine, now observed through
# CAP-O03: the workspace home lists Applications (role-aware), not a flat
# machine list -- Requester sees Design, HR sees HR, both exist in one
# workspace and are independently reachable.
body_contains "$BASE_URL/" "Design" "menata_role=Requester" && body_contains "$BASE_URL/" "HR" "menata_role=HR"
check T01 "CAP-X01" "workspace home lists applications, role-scoped, from both applications" $?

# T02 — CAP-V01 form view: fields config drives inputs; status excluded.
# Employee is Leave Request's real submitting role (CAP-P05: NewForm needs
# CanCreate, no permission row means denied).
curl -s -H "Cookie: menata_role=Employee" "$BASE_URL/mch_leave_request/new" | grep -q 'for="fld_lr_reason"' \
  && ! curl -s -H "Cookie: menata_role=Employee" "$BASE_URL/mch_leave_request/new" | grep -q 'for="fld_lr_status"'
check T02 "CAP-V01" "form renders configured fields, excludes status" $?

# T03 — CAP-V02 list view: columns config drives table
body_contains "$BASE_URL/mch_leave_request" "Leave Type" "menata_role=Employee"
check T03 "CAP-V02" "list renders configured columns" $?

# T04 — CAP-C01 required constraint
post_body_contains "$BASE_URL/mch_leave_request" "" "Reason is required." "menata_role=Employee"
check T04 "CAP-C01" "empty submit rejected: required violation" $?

# T05 — CAP-C02 after:today constraint
post_body_contains "$BASE_URL/mch_leave_request" "" "Start Date must be after today." "menata_role=Employee"
check T05 "CAP-C02" "empty submit rejected: date-future violation" $?

# T06 — CAP-C03+C04 conditional constraint fires (Banner without attachment).
# Requester is Design Request's real submitting role (CAP-P05: Create needs
# CanCreate).
DR_DATA_BANNER="fld_requester=ConformanceBot&fld_design_type=Banner+2%3A1&fld_due_date=2030-01-01&fld_title=Conformance+T06&fld_description=Test"
post_body_contains "$BASE_URL/mch_design_request" "$DR_DATA_BANNER" "Attachment is required" "menata_role=Requester"
check T06 "CAP-C03,CAP-C04" "conditional constraint fires when condition true" $?

# T07 — CAP-C04 conditional constraint silent when condition false (Poster)
DR_DATA_POSTER="fld_requester=ConformanceBot&fld_design_type=Poster&fld_due_date=2030-01-01&fld_title=Conformance+T07&fld_description=Test"
CODE=$(post_status "$BASE_URL/mch_design_request" "$DR_DATA_POSTER" "menata_role=Requester")
[ "$CODE" = "303" ]
check T07 "CAP-C04" "conditional constraint silent when condition false (got $CODE)" $?

# T08 — CAP-R01 create record with default status
LR_DATA="fld_lr_employee=ConformanceBot&fld_lr_leave_type=Annual+Leave&fld_lr_start_date=2030-01-01&fld_lr_end_date=2030-01-03&fld_lr_reason=Conformance+run"
DETAIL_URL=$(post_redirect "$BASE_URL/mch_leave_request" "$LR_DATA" "menata_role=Employee")
[ -n "$DETAIL_URL" ] && body_contains "$DETAIL_URL" "Draft" "menata_role=Employee"
check T08 "CAP-R01" "valid create redirects to detail with default status Draft" $?

# derive record id from redirect
REC_ID="${DETAIL_URL##*/}"

# T09 — CAP-V03 detail view shows all fields
body_contains "$DETAIL_URL" "Reason" "menata_role=Employee"
check T09 "CAP-V03" "detail renders machine fields" $?

# T10 — CAP-E01+A01 permitted event executes set_field
CODE=$(post_status "$BASE_URL/mch_leave_request/$REC_ID/events/evt_lr_submit" "" "menata_role=Employee")
[ "$CODE" = "303" ] && body_contains "$DETAIL_URL" "Submitted" "menata_role=Employee"
check T10 "CAP-E01,CAP-A01" "Employee triggers Submit; status becomes Submitted" $?

# T11 — CAP-P01 permission guard denies unpermitted role
CODE=$(post_status "$BASE_URL/mch_leave_request/$REC_ID/events/evt_lr_approve" "" "menata_role=Employee")
[ "$CODE" = "403" ]
check T11 "CAP-P01" "Employee denied Approve (got $CODE)" $?

# T12 — CAP-P01+E01 permitted role executes cross-role transition
CODE=$(post_status "$BASE_URL/mch_leave_request/$REC_ID/events/evt_lr_approve" "" "menata_role=Manager")
[ "$CODE" = "303" ] && body_contains "$DETAIL_URL" "Approved" "menata_role=Manager"
check T12 "CAP-P01,CAP-E01" "Manager triggers Approve; status becomes Approved" $?

# --- CAP-F13 (reference fields) — requires seeds/003_hr_employee.sql ---

# T13 — CAP-F13 form renders a picker (<select>), not a bare text input.
# HR is Employee's only real permission-bearing role (CAP-P05).
body_contains "$BASE_URL/mch_employee/new" 'select id="fld_emp_manager"' "menata_role=HR"
check T13 "CAP-F13" "reference field renders as a picker" $?

# T14 — CAP-F13 create with an empty (optional) reference succeeds
MGR_DATA="fld_emp_id=CB-MGR&fld_emp_name=ConformanceBot+Manager&fld_emp_hire_date=2020-01-01"
MGR_URL=$(post_redirect "$BASE_URL/mch_employee" "$MGR_DATA" "menata_role=HR")
[ -n "$MGR_URL" ]
check T14 "CAP-F13" "create with empty reference succeeds (Manager is optional)" $?
MGR_ID="${MGR_URL##*/}"

# T15 — CAP-F13 create with a valid reference succeeds; detail links to the target
REPORT_DATA="fld_emp_id=CB-EMP&fld_emp_name=ConformanceBot+Report&fld_emp_hire_date=2024-01-01&fld_emp_manager=$MGR_ID"
REPORT_URL=$(post_redirect "$BASE_URL/mch_employee" "$REPORT_DATA" "menata_role=HR")
[ -n "$REPORT_URL" ] && body_contains "$REPORT_URL" "href=\"/mch_employee/$MGR_ID\"" "menata_role=HR"
check T15 "CAP-F13" "create with valid reference succeeds; detail links to target record" $?

# T16 — CAP-F13 dangling reference rejected (security NFR gate: negative case)
BAD_DATA="fld_emp_id=CB-GHOST&fld_emp_name=ConformanceBot+Ghost&fld_emp_hire_date=2024-01-01&fld_emp_manager=00000000-0000-0000-0000-000000000000"
post_body_contains "$BASE_URL/mch_employee" "$BAD_DATA" "does not reference an existing" "menata_role=HR"
check T16 "CAP-F13" "dangling reference value rejected, not silently accepted" $?

# --- CAP-E06 (state-conditional event availability) ---

# T17 — Approve rejected while a record is still Draft (never Submitted)
DRAFT_DATA="fld_lr_employee=ConformanceBot&fld_lr_leave_type=Annual+Leave&fld_lr_start_date=2030-01-01&fld_lr_end_date=2030-01-03&fld_lr_reason=T17"
DRAFT_URL=$(post_redirect "$BASE_URL/mch_leave_request" "$DRAFT_DATA" "menata_role=Employee")
DRAFT_ID="${DRAFT_URL##*/}"
CODE=$(post_status "$BASE_URL/mch_leave_request/$DRAFT_ID/events/evt_lr_approve" "" "menata_role=Manager")
[ "$CODE" = "400" ]
check T17 "CAP-E06" "Approve rejected while record is still Draft (got $CODE)" $?

# T18 — the headline fix: an already-Approved record can no longer be Rejected
# (reuses $REC_ID from T08-T12, which is Approved by this point in the run)
CODE=$(post_status "$BASE_URL/mch_leave_request/$REC_ID/events/evt_lr_reject" "" "menata_role=Manager")
[ "$CODE" = "400" ]
check T18 "CAP-E06" "Reject rejected on an already-Approved record (got $CODE) -- the exact Study 1 gap" $?

# --- CAP-C09 (constraints evaluated on event trigger, not just Create) ---

# T19 — a record valid at Create can become invalid by the time an event fires;
# the already-declared "Start Date must be after today" constraint must block
# Approve too, not just Create. See header note re: psql use here.
if [ -n "$DATABASE_URL" ]; then
    C09_DATA="fld_lr_employee=ConformanceBot&fld_lr_leave_type=Sick+Leave&fld_lr_start_date=2030-01-01&fld_lr_end_date=2030-01-03&fld_lr_reason=T19"
    C09_URL=$(post_redirect "$BASE_URL/mch_leave_request" "$C09_DATA" "menata_role=Employee")
    C09_ID="${C09_URL##*/}"
    post_status "$BASE_URL/mch_leave_request/$C09_ID/events/evt_lr_submit" "" "menata_role=Employee" >/dev/null
    # CAP-X06: RLS is live (migrations/009) -- a raw psql connection has no
    # app.workspace_id set, so this UPDATE would match zero rows (fail
    # closed) without setting it first, same statement/transaction (multiple
    # ;-separated statements in one -c invocation share Postgres's implicit
    # per-message transaction, so set_config's is_local=true still applies).
    psql "$DATABASE_URL" -q -c \
        "SELECT set_config('app.workspace_id', 'ws_default', true);
         UPDATE records SET data = jsonb_set(data, '{fld_lr_start_date}', '\"2020-01-01\"') WHERE id = '$C09_ID'" \
        >/dev/null 2>&1
    post_body_contains "$BASE_URL/mch_leave_request/$C09_ID/events/evt_lr_approve" "" "Start Date must be after today" "menata_role=Manager"
    check T19 "CAP-C09" "Approve blocked by a Constraint the event trigger re-checks, not just Create" $?
else
    printf 'SKIP  T19  %-22s %s\n' "CAP-C09" "DATABASE_URL not set -- backdating fixture unavailable"
fi

# T20 — CAP-A02 dynamic values: "today"/"current_user" resolve to real values,
# not stored as the literal token. Reuses $REC_ID/$DETAIL_URL from T08-T12,
# which evt_lr_approve already stamped with Approved Date/Approved By.
TODAY=$(date +%Y-%m-%d)
body_contains "$DETAIL_URL" "$TODAY" "menata_role=Manager" && ! body_contains "$DETAIL_URL" "current_user" "menata_role=Manager"
check T20 "CAP-A02" "Approve stamps real today's date and the acting role, not literal tokens" $?

# T21 — CAP-V06 child records sub-list: reuses $MGR_URL/CB-EMP's Manager
# relationship already established by T14/T15 (CAP-F13).
body_contains "$MGR_URL" "ConformanceBot Report" "menata_role=HR"
check T21 "CAP-V06" "manager's detail page lists its direct report via reverse reference" $?

# --- CAP-A07 (activate_next / sequential step guard), CAP-A08 (aggregate_status
# rollup), CAP-X03 (machine config) — requires seeds/004_approval.sql ---

# Sequential-mode document with two steps (Bob seq 1, Carol seq 2). Submitter
# creates both the Document and its Steps (CAP-P05: perm_ad_submitter_steps).
AD_SEQ_DATA="fld_ad_title=T22+Policy&fld_ad_document_type=Policy&fld_ad_file=policy.pdf&fld_ad_submitted_by=Alice&fld_ad_approval_mode=Sequential"
AD_SEQ_URL=$(post_redirect "$BASE_URL/mch_approval_document" "$AD_SEQ_DATA" "menata_role=Submitter")
AD_SEQ_ID="${AD_SEQ_URL##*/}"
post_status "$BASE_URL/mch_approval_document/$AD_SEQ_ID/events/evt_ad_submit" "" "menata_role=Submitter" >/dev/null
AS1_URL=$(post_redirect "$BASE_URL/mch_approval_step" "fld_as_document=$AD_SEQ_ID&fld_as_approver=Bob&fld_as_sequence=1" "menata_role=Submitter")
AS1_ID="${AS1_URL##*/}"
AS2_URL=$(post_redirect "$BASE_URL/mch_approval_step" "fld_as_document=$AD_SEQ_ID&fld_as_approver=Carol&fld_as_sequence=2" "menata_role=Submitter")
AS2_ID="${AS2_URL##*/}"

# T22 — CAP-A07 hard block: approving Step 2 before Step 1 is rejected.
# Carol is AS2's actual assigned Approver (CAP-P02) -- must pass ownership to
# even reach the sequential guard being tested here.
CODE=$(post_status "$BASE_URL/mch_approval_step/$AS2_ID/events/evt_as_approve" "" "menata_role=Approver; menata_identity=Carol")
[ "$CODE" = "400" ]
check T22 "CAP-A07" "out-of-sequence Approve rejected in Sequential mode (got $CODE)" $?

# T23 — CAP-A07 in-order approval succeeds. Bob is AS1's assigned Approver.
CODE=$(post_status "$BASE_URL/mch_approval_step/$AS1_ID/events/evt_as_approve" "" "menata_role=Approver; menata_identity=Bob")
[ "$CODE" = "303" ]
check T23 "CAP-A07" "in-sequence Approve succeeds (got $CODE)" $?

# T24 — CAP-A08 all-approved rollup: Document stays In Review until every
# step is Approved, then transitions automatically (no direct Approve call
# on the Document itself -- only System may trigger it).
body_contains "$AD_SEQ_URL" "In Review" "menata_role=Submitter"
post_status "$BASE_URL/mch_approval_step/$AS2_ID/events/evt_as_approve" "" "menata_role=Approver; menata_identity=Carol" >/dev/null
body_contains "$AD_SEQ_URL" "Approved" "menata_role=Submitter"
check T24 "CAP-A08" "Document auto-transitions to Approved once every Step is Approved" $?

# T25 — Parallel-mode document: no sequential gating (approve Step 2 before
# Step 1 succeeds, unlike T22's Sequential-mode document)
AD_PAR_DATA="fld_ad_title=T25+Contract&fld_ad_document_type=Contract&fld_ad_file=contract.pdf&fld_ad_submitted_by=Alice&fld_ad_approval_mode=Parallel"
AD_PAR_URL=$(post_redirect "$BASE_URL/mch_approval_document" "$AD_PAR_DATA" "menata_role=Submitter")
AD_PAR_ID="${AD_PAR_URL##*/}"
post_status "$BASE_URL/mch_approval_document/$AD_PAR_ID/events/evt_ad_submit" "" "menata_role=Submitter" >/dev/null
PS1_URL=$(post_redirect "$BASE_URL/mch_approval_step" "fld_as_document=$AD_PAR_ID&fld_as_approver=Bob&fld_as_sequence=1&fld_as_notes=T26+rejection+note" "menata_role=Submitter")
PS1_ID="${PS1_URL##*/}"
PS2_URL=$(post_redirect "$BASE_URL/mch_approval_step" "fld_as_document=$AD_PAR_ID&fld_as_approver=Carol&fld_as_sequence=2" "menata_role=Submitter")
PS2_ID="${PS2_URL##*/}"
CODE=$(post_status "$BASE_URL/mch_approval_step/$PS2_ID/events/evt_as_approve" "" "menata_role=Approver; menata_identity=Carol")
[ "$CODE" = "303" ]
check T25 "CAP-A07" "Parallel mode has no sequential gating -- Step 2 decided before Step 1 (got $CODE)" $?

# T26 — CAP-A08 any-rejected cascade: fires immediately, doesn't wait for the
# still-pending... (here, already-Approved) sibling.
post_status "$BASE_URL/mch_approval_step/$PS1_ID/events/evt_as_reject" "" "menata_role=Approver; menata_identity=Bob" >/dev/null
body_contains "$AD_PAR_URL" "Rejected" "menata_role=Submitter"
check T26 "CAP-A08" "Document cascades to Rejected as soon as any Step rejects, not waiting for the rest" $?

# --- CAP-P02 (record-level ownership) ---

# T36 — negative: Carol holds the correct "Approver" role but is not AS1's
# assigned Approver (Bob is) -- role alone is not enough, direct allocation
# denies her.
CODE=$(post_status "$BASE_URL/mch_approval_step/$AS1_ID/events/evt_as_reject" "" "menata_role=Approver; menata_identity=Carol")
[ "$CODE" = "403" ]
check T36 "CAP-P02" "correct role but wrong identity denied deciding another Approver's Step (got $CODE)" $?

# --- CAP-E05 (same-record trigger_event) — requires seeds/005_complaint.sql ---

CMP_DATA="fld_cmp_complainant_name=Conformance+Tester"
CMP_URL=$(post_redirect "$BASE_URL/mch_complaint" "$CMP_DATA" "menata_role=Agent")
CMP_ID="${CMP_URL##*/}"

# T37 — negative: Run SLA Check blocked while Status is still New (events.condition,
# CAP-E06) — the chained Escalate must not fire, Assigned To stays empty.
CODE=$(post_status "$BASE_URL/mch_complaint/$CMP_ID/events/evt_cmp_run_sla_check" "" "menata_role=Supervisor")
[ "$CODE" = "400" ] && ! body_contains "$CMP_URL" "Supervisor" "menata_role=Agent"
check T37 "CAP-E05" "Run SLA Check blocked outside Investigating; chained Escalate did not fire (got $CODE)" $?

# T38 — positive: once Status reaches Investigating, Run SLA Check's trigger_event
# action fires Escalate on the SAME record — Assigned To becomes Supervisor.
post_status "$BASE_URL/mch_complaint/$CMP_ID/events/evt_cmp_triage" "" "menata_role=Agent" >/dev/null
post_status "$BASE_URL/mch_complaint/$CMP_ID/events/evt_cmp_add_investigation_note" "" "menata_role=Agent" >/dev/null
CODE=$(post_status "$BASE_URL/mch_complaint/$CMP_ID/events/evt_cmp_run_sla_check" "" "menata_role=Supervisor")
[ "$CODE" = "303" ] && body_contains "$CMP_URL" "Supervisor" "menata_role=Agent"
check T38 "CAP-E05" "Run SLA Check while Investigating chains into Escalate on the same record (got $CODE)" $?

# --- CAP-R02 (edit / update record via form) ---
# Reuses $REC_ID (Leave Request, Approved by T12) and $MGR_ID/$REPORT_URL
# (Employee, CAP-F13 references established by T14/T15).

# T27 — edit form pre-fills the record's current values
body_contains "$BASE_URL/mch_leave_request/$REC_ID/edit" 'value="ConformanceBot"' "menata_role=Employee"
check T27 "CAP-R02" "edit form pre-fills existing field values" $?

# T28 — valid update persists the change and leaves fields the form doesn't
# expose (Status, still Approved from T12) untouched
LR_UPDATE_DATA="fld_lr_employee=ConformanceBot&fld_lr_leave_type=Annual+Leave&fld_lr_start_date=2030-01-01&fld_lr_end_date=2030-01-03&fld_lr_reason=T28+updated+reason"
CODE=$(post_status "$BASE_URL/mch_leave_request/$REC_ID" "$LR_UPDATE_DATA" "menata_role=Employee")
[ "$CODE" = "303" ] && body_contains "$DETAIL_URL" "T28 updated reason" "menata_role=Employee" \
  && body_contains "$DETAIL_URL" "Approved" "menata_role=Employee"
check T28 "CAP-R02" "valid update persists changed field, preserves Status outside the form (got $CODE)" $?

# T29 — update re-validates Constraints, same as Create (required violation)
LR_BAD_DATA="fld_lr_employee=ConformanceBot&fld_lr_leave_type=Annual+Leave&fld_lr_start_date=2030-01-01&fld_lr_end_date=2030-01-03&fld_lr_reason="
post_body_contains "$BASE_URL/mch_leave_request/$REC_ID" "$LR_BAD_DATA" "Reason is required." "menata_role=Employee"
check T29 "CAP-R02,CAP-C09" "update rejected on required-field violation, same as Create" $?

# T30 — update re-validates CAP-F13 referential integrity (dangling reference,
# and a hand-typed non-UUID value — the latter caught a real 500 during
# development, see internal/store/record_store.go's Exists comment)
REPORT_ID="${REPORT_URL##*/}"
EMP_BAD_DATA="fld_emp_id=CB-EMP&fld_emp_name=ConformanceBot+Report&fld_emp_hire_date=2024-01-01&fld_emp_manager=not-a-real-id"
post_body_contains "$BASE_URL/mch_employee/$REPORT_ID" "$EMP_BAD_DATA" "does not reference an existing" "menata_role=HR"
check T30 "CAP-R02,CAP-F13" "update rejected on a malformed (non-UUID) reference value, not a 500" $?

# --- CAP-A03 (notify to role), CAP-A04 (notify to dynamic recipient),
# CAP-A10 (in-app notification delivery channel) ---

# T31 — CAP-A03 + CAP-A10: a static `notify: {role: ...}` action delivers a
# real in-app Notification, not just a log line. Reuses $REC_ID's Approve
# (T12), which fired `notify: {role: Employee}`.
body_contains "$BASE_URL/notifications" "Leave Request: Approve" "menata_role=Employee"
check T31 "CAP-A03,CAP-A10" "static-role notify delivers a real in-app notification" $?

# T32 — CAP-A10: the unread count badge appears on an unrelated page (Home),
# not just the notifications page itself.
body_contains "$BASE_URL/" 'class="ml-1 inline-flex items-center rounded-full bg-red-100' "menata_role=Employee"
check T32 "CAP-A10" "unread notification count badge renders on the nav bar" $?

# T33 — CAP-A10: mark-read actually persists (button disappears for that
# notification, not just a redirect).
NOTIF_PATH=$(get_body "$BASE_URL/notifications" "menata_role=Employee" | grep -oE '/notifications/[a-f0-9-]+/read' | head -1)
CODE="000"
if [ -n "$NOTIF_PATH" ]; then
    CODE=$(post_status "$BASE_URL$NOTIF_PATH" "" "menata_role=Employee")
fi
[ "$CODE" = "303" ] && ! body_contains "$BASE_URL$NOTIF_PATH" "Mark read" "menata_role=Employee"
check T33 "CAP-A10" "marking a notification read persists (got $CODE, path $NOTIF_PATH)" $?

# T34 — CAP-A04: `notify: {recipient_field: fld_ad_submitted_by, role: Submitter}`
# resolves to the record's OWN Submitted By value ("Alice", from T22's
# AD_SEQ_DATA), not the generic Submitter role. Reuses $AD_SEQ_ID, all-
# approved by T24 (fires evt_ad_approve, which carries this action).
body_contains "$BASE_URL/notifications" "Approval Document: Approve" "menata_role=Alice"
check T34 "CAP-A04" "dynamic recipient_field notifies the record's specific submitter, not a role" $?

# T35 — CAP-A04 negative case: the generic Submitter role must NOT also
# receive it — proves recipient_field actually overrode role, rather than
# both firing.
! body_contains "$BASE_URL/notifications" "Approval Document: Approve" "menata_role=Submitter"
check T35 "CAP-A04" "generic Submitter role does not also receive the dynamically-targeted notification" $?

# --- CAP-P05 (CRUD-level permissions, deny-by-default) ---

# T39 — negative: Manager has no permission row at all on mch_employee (only
# HR does) -- read is denied, not implicitly allowed.
CODE=$(curl -s -o /dev/null -w '%{http_code}' -H "Cookie: menata_role=Manager" "$BASE_URL/mch_employee")
[ "$CODE" = "403" ]
check T39 "CAP-P05" "role with no permission row on a machine is denied List (got $CODE)" $?

# T40 — negative: Employee has no permission row at all on
# mch_approval_document (only Submitter/System do) -- Create is denied.
CODE=$(post_status "$BASE_URL/mch_approval_document" "fld_ad_title=T40" "menata_role=Employee")
[ "$CODE" = "403" ]
check T40 "CAP-P05" "role with no permission row on a machine is denied Create (got $CODE)" $?

# T41 — negative: same reasoning, the Edit form -- distinct code path from
# T40's Create denial on the same machine.
CODE=$(curl -s -o /dev/null -w '%{http_code}' -H "Cookie: menata_role=Employee" "$BASE_URL/mch_approval_document/$AD_SEQ_ID/edit")
[ "$CODE" = "403" ]
check T41 "CAP-P05" "role with no permission row on a machine is denied the Edit form (got $CODE)" $?

# --- CAP-R04 (audit log actor attribution) + CAP-I04 (correlation trace) ---
# DB inspection, same documented exception as T19 -- these prove a DB-level
# fact (record_events columns) an HTTP black-box test can't observe.

if [ -n "$DATABASE_URL" ]; then
    # T42 -- performed_by carries the real acting identity (Manager, from
    # T12's Approve on $REC_ID), not NULL and not the literal role string
    # where identity was actually set.
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
    [ "$PERFORMED_BY" = "Manager" ]
    check T42 "CAP-R04" "record_events.performed_by carries the real acting role/identity (got '$PERFORMED_BY')" $?

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

# T44 -- Approver can now read Approval Document (perm_ad_approver_read)
CODE=$(curl -s -o /dev/null -w '%{http_code}' -H "Cookie: menata_role=Approver; menata_identity=Bob" "$BASE_URL/mch_approval_document")
[ "$CODE" = "200" ]
check T44 "CAP-P05" "Approver can read Approval Document, needed for context on the Step they're deciding (got $CODE)" $?

# T45 -- negative: still can't create/edit Documents, read-only
CODE=$(post_status "$BASE_URL/mch_approval_document" "fld_ad_title=T45" "menata_role=Approver; menata_identity=Bob")
[ "$CODE" = "403" ]
check T45 "CAP-P05" "Approver still denied Create on Approval Document -- read-only, not full access (got $CODE)" $?

# --- CAP-O03 (Navigation metadata: app grouping, role-aware menus, workspace
# home) -- Workspace > Application > Machine, 006-runtime-model.md ---

# T46 -- drilling into an Application lists its own Machines
body_contains "$BASE_URL/apps/app_design" "Design Request" "menata_role=Requester"
check T46 "CAP-O03" "drilling into an Application lists its own Machines" $?

# T47 -- role-aware: a role with zero readable machines in an Application
# never sees that Application's card on the workspace home at all
! body_contains "$BASE_URL/" 'href="/apps/app_hr"' "menata_role=Requester"
check T47 "CAP-O03" "Requester (no access anywhere in HR) never sees the HR application card" $?

# T48 -- negative: drilling into an Application whose Machines the role
# can't read still 404s the Application route itself correctly, and shows
# no Machine cards for one it partially can't -- Manager can read
# mch_leave_request (HR) but not mch_employee (also HR); the HR app page
# must show Leave Request without Employee.
body_contains "$BASE_URL/apps/app_hr" "Leave Request" "menata_role=Manager" \
  && ! body_contains "$BASE_URL/apps/app_hr" 'href="/mch_employee"' "menata_role=Manager"
check T48 "CAP-O03" "within an Application, only individually-readable Machines are listed" $?

# --- CAP-X06 (workspace isolation) -- requires seeds/006_second_workspace.sql ---
# ws_acme (Operations/Task) is a deliberately separate Workspace from every
# other case's ws_default, existing purely to prove isolation.

# T49 -- negative: a ws_default-scoped session given a direct URL to
# ws_acme's Machine is denied -- app-layer guard (Interpreter.ScopeFor),
# independent of RLS at the DB layer.
CODE=$(curl -s -o /dev/null -w '%{http_code}' -H "Cookie: menata_role=Staff" "$BASE_URL/mch_task")
[ "$CODE" = "404" ]
check T49 "CAP-X06" "ws_default session denied direct access to another workspace's Machine (got $CODE)" $?

# T50 -- same, for the Application route
CODE=$(curl -s -o /dev/null -w '%{http_code}' -H "Cookie: menata_role=Staff" "$BASE_URL/apps/app_ops")
[ "$CODE" = "404" ]
check T50 "CAP-X06" "ws_default session denied direct access to another workspace's Application (got $CODE)" $?

# T51 -- positive: switching to ws_acme (menata_workspace cookie) grants
# access to its own Machine, end to end (create + trigger event).
TASK_URL=$(post_redirect "$BASE_URL/mch_task" "fld_task_title=Conformance+Task" "menata_workspace=ws_acme; menata_role=Staff")
TASK_ID="${TASK_URL##*/}"
CODE=$(post_status "$BASE_URL/mch_task/$TASK_ID/events/evt_task_complete" "" "menata_workspace=ws_acme; menata_role=Staff")
[ "$CODE" = "303" ] && body_contains "$TASK_URL" "Done" "menata_workspace=ws_acme; menata_role=Staff"
check T51 "CAP-X06" "switching workspace grants access to its own Machine end to end (got $CODE)" $?

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

echo "--------------------------------------------------------------------"
echo "Result: $PASS passed, $FAIL failed"
[ "$FAIL" -eq 0 ] || exit 1
