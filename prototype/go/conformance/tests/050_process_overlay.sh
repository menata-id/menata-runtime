#!/usr/bin/env bash
# Process Overlay B1-B4 (parity, process map, requirement cardinality, SLA, quorum-of-N) and CAP-X04 (metadata live reload) -- Study 21 and its direct follow-on.
# Sourced by run.sh after lib.sh -- assumes lib.sh's helpers/ACCOUNTS are already in scope.

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

