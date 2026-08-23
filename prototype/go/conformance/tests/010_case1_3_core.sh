#!/usr/bin/env bash
# Design Request / Leave Request / Approval Document+Step / Complaint -- the original single-capability-per-commit era (Cases 1-3), before the "Batch N" naming convention started.
# Sourced by run.sh after lib.sh -- assumes lib.sh's helpers/ACCOUNTS are already in scope.

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

