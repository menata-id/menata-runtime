#!/usr/bin/env bash
# CAP-W03's declarative quorum form, the Case 9 completion batch (CAP-C08/C10/C11), decompile-lift (B6), and the full Track D UI/Interaction cluster (CAP-V17/V18/V16/V15/V19/V14 Tier 2) -- the most recent work as of this split.
# Sourced by run.sh after lib.sh -- assumes lib.sh's helpers/ACCOUNTS are already in scope.

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

# --- CAP-V18 (resource-grouped calendar) -- seeds/030, boot-time. Section
# isolation is checked via python3 (already present on this host) rather
# than fragile line-based grep -A/-B, since templ's own line-wrapping isn't
# a contract this suite should depend on.

RES_SCHED=$(session_for resourcecal.scheduler@example.com password)

STAFF_A_URL=$(post_redirect "$BASE_URL/mch_v18_staff" "fld_v18s_name=Dr.+Amara+$$" "$RES_SCHED")
STAFF_A_ID="${STAFF_A_URL##*/}"
STAFF_B_URL=$(post_redirect "$BASE_URL/mch_v18_staff" "fld_v18s_name=Dr.+Budi+$$" "$RES_SCHED")
STAFF_B_ID="${STAFF_B_URL##*/}"
STAFF_C_URL=$(post_redirect "$BASE_URL/mch_v18_staff" "fld_v18s_name=Dr.+Citra+$$" "$RES_SCHED")
STAFF_C_ID="${STAFF_C_URL##*/}"

post_redirect "$BASE_URL/mch_v18_appointment" "fld_v18a_title=Amara-Only-$$&fld_v18a_staff=$STAFF_A_ID&fld_v18a_date=2026-09-01" "$RES_SCHED" > /dev/null
post_redirect "$BASE_URL/mch_v18_appointment" "fld_v18a_title=Budi-Only-$$&fld_v18a_staff=$STAFF_B_ID&fld_v18a_date=2026-09-01" "$RES_SCHED" > /dev/null

CAL_BODY=$(get_body "$BASE_URL/mch_v18_appointment/calendar" "$RES_SCHED")

# T169 -- two staff each with a same-day appointment: each staff's OWN
# section contains only their own appointment title, not the other's.
V18_ISOLATION=$(python3 -c "
import sys
body, amara_h, budi_h, amara_a, budi_a = sys.argv[1:6]
def section(heading):
    start = body.find(heading)
    if start == -1:
        return ''
    nxt = body.find('<h2', start + 3)
    return body[start:nxt if nxt != -1 else len(body)]
amara_sec = section(amara_h)
budi_sec = section(budi_h)
ok = (amara_a in amara_sec and budi_a not in amara_sec and
      budi_a in budi_sec and amara_a not in budi_sec)
print('OK' if ok else 'FAIL')
" "$CAL_BODY" "Dr. Amara $$" "Dr. Budi $$" "Amara-Only-$$" "Budi-Only-$$")
[ "$V18_ISOLATION" = "OK" ]
check T169 "CAP-V18" "two staff with same-day appointments each show only their own (got $V18_ISOLATION)" $?

# T170 -- a staff member with zero appointments still gets a (empty)
# section, proving the grouping is resource-driven, not a filtered date
# list that would silently drop an idle resource.
V18_IDLE=$(python3 -c "
import sys
body, citra_h = sys.argv[1:3]
start = body.find(citra_h)
nxt = body.find('<h2', start + 3) if start != -1 else -1
section = body[start:nxt if nxt != -1 else len(body)] if start != -1 else ''
print('OK' if start != -1 and 'No dated records' in section else 'FAIL')
" "$CAL_BODY" "Dr. Citra $$")
[ "$V18_IDLE" = "OK" ]
check T170 "CAP-V18" "a staff member with zero appointments still gets its own (empty) section (got $V18_IDLE)" $?

# --- CAP-V16 (typeahead/autocomplete) -- seeds/031, boot-time (30
# pre-seeded Product records, past the 25-record eager-<select> threshold).

TA_CLERK=$(session_for typeahead.clerk@example.com password)

# T171 -- the New Order form renders the typeahead input (referencing the
# new field-options endpoint), not a 30-option <select> (no eagerly-listed
# product name anywhere in the page).
NEW_ORDER_BODY=$(get_body "$BASE_URL/mch_v16_order/new" "$TA_CLERK")
echo "$NEW_ORDER_BODY" | grep -q 'field-options?field=fld_v16o_product'
HAS_TYPEAHEAD=$?
! echo "$NEW_ORDER_BODY" | grep -q "Widget 015"
HAS_NO_EAGER_OPTION=$?
[ "$HAS_TYPEAHEAD" -eq 0 ] && [ "$HAS_NO_EAGER_OPTION" -eq 0 ]
check T171 "CAP-V16" "a 30-candidate reference field renders the typeahead input, not an eager 30-option select" $?

# T172 -- the search endpoint returns only matching candidates, not all 30.
OPTS_BODY=$(get_body "$BASE_URL/mch_v16_order/field-options?field=fld_v16o_product&q=029" "$TA_CLERK")
echo "$OPTS_BODY" | grep -q "Widget 029"
HAS_MATCH=$?
! echo "$OPTS_BODY" | grep -q "Widget 001"
HAS_NO_OTHER=$?
[ "$HAS_MATCH" -eq 0 ] && [ "$HAS_NO_OTHER" -eq 0 ]
check T172 "CAP-V16" "GET .../field-options?q=029 returns only the matching candidate, not all 30" $?

# T173 -- submitting a typeahead-selected value still creates the record
# correctly end-to-end -- the real regression check, proving the hidden
# field round-trips a valid id exactly like the eager <select> already did.
PRODUCT_ID=$(echo "$OPTS_BODY" | grep -oE 'data-id="[a-f0-9-]+"' | head -1 | sed -E 's/data-id="([^"]+)"/\1/')
ORDER_URL=$(post_redirect "$BASE_URL/mch_v16_order" "fld_v16o_title=Typeahead+Order+$$&fld_v16o_product=$PRODUCT_ID" "$TA_CLERK")
body_contains "$ORDER_URL" "Widget 029" "$TA_CLERK"
check T173 "CAP-V16" "submitting a typeahead-selected value creates the record correctly end-to-end" $?

# --- CAP-V15 (live aggregate preview) -- reuses seeds/027_case9_completion_lab.sql
# (already declares a CrossRecord{Kind:"aggregate"} constraint on
# mch_c9_journal_entry, and its own form now embeds child_lines rows for
# mch_c9_journal_entry_line, added same-day for this phase). This project's
# conformance suite is HTTP black-box (curl) -- it cannot execute JS or
# observe a live DOM update. This test proves the server emits the correct
# wiring (the data-sum-*-field attributes handler.buildChildLinesData
# derives from the parent's own Constraint); the actual live-sum behavior
# was manually verified in a real browser-equivalent request/response cycle
# before this phase was reported complete, not claimed as automated here.

# T174 -- the Journal Entry form's child_lines section is wired for a live
# debit/credit total, derived from the SAME Constraint T161/T162 already
# prove server-side -- no separate config needed.
NEW_JE_BODY=$(get_body "$BASE_URL/mch_c9_journal_entry/new" "$C9ACCT")
echo "$NEW_JE_BODY" | grep -q 'data-sum-a-field="fld_c9jel_debit"'
HAS_SUM_A=$?
echo "$NEW_JE_BODY" | grep -q 'data-sum-b-field="fld_c9jel_credit"'
HAS_SUM_B=$?
[ "$HAS_SUM_A" -eq 0 ] && [ "$HAS_SUM_B" -eq 0 ]
check T174 "CAP-V15" "the Journal Entry form's child_lines section is wired for a live debit/credit total" $?

# --- CAP-V19 (live cross-record balance preview) -- reuses the SAME
# seeds/027_case9_completion_lab.sql fixture again (its CrossRecord{Kind:
# "reference_field"} constraint already targets mch_c9_fiscal_period's own
# Status field). Zero new routes -- CAP-X07's existing GET
# /api/{machine}/{record} is reused directly. Same HTTP-black-box
# limitation as T174: this proves the correct wiring is emitted, not that a
# browser executes it -- manually verified in a real request/response cycle
# before this phase was reported complete.

# T175 -- the Journal Entry form's Fiscal Period picker is wired for a live
# preview of that period's own Status, fetched from the existing JSON API.
NEW_JE_BODY2=$(get_body "$BASE_URL/mch_c9_journal_entry/new" "$C9ACCT")
echo "$NEW_JE_BODY2" | grep -q 'data-preview-url="/ws_default/api/v1/mch_c9_fiscal_period/"'
HAS_PREVIEW_URL=$?
echo "$NEW_JE_BODY2" | grep -q 'data-preview-field="fld_c9fp_status"'
HAS_PREVIEW_FIELD=$?
[ "$HAS_PREVIEW_URL" -eq 0 ] && [ "$HAS_PREVIEW_FIELD" -eq 0 ]
check T175 "CAP-V19" "the Journal Entry form's Fiscal Period picker is wired for a live status preview" $?

# --- CAP-V14 Tier 2 (kanban board) -- seeds/032_kanban_lab.sql. A "board"
# View groups records into lanes from an existing value_list Field
# (ViewConfig.GroupField) -- narrower than Case 19's own "user-creatable
# Lists" model, a deliberate scope cut (see capability-registry.md's own
# CAP-V14 row). The drag gesture itself is client-side JS this HTTP-black-box
# suite cannot execute, but the write it triggers (BoardMove) is an ordinary
# POST -- both halves below are fully HTTP-testable without a browser.

KB_LEAD=$(session_for kanban.lead@example.com password)

# T176 -- the board groups records into the lane matching each record's
# current Status value, and a lane nobody's in yet ("Done") still renders
# with its own (empty) section -- same "every declared option is a valid
# drop target" proof T170 already made for CAP-V18's resource sections.
BOARD_BODY=$(get_body "$BASE_URL/mch_kanban_task/board" "$KB_LEAD")
V14T2_LANES=$(python3 -c "
import sys
body = sys.argv[1]
def section(lane):
    marker = 'data-lane=\"' + lane + '\"'
    start = body.find(marker)
    if start == -1:
        return ''
    nxt = body.find('data-lane=\"', start + len(marker))
    return body[start:nxt if nxt != -1 else len(body)]
todo, doing, done = section('Todo'), section('Doing'), section('Done')
ok = (
    'Write proposal' in todo and 'Review budget' in todo and 'Draft contract' not in todo
    and 'Draft contract' in doing and 'Write proposal' not in doing
    and 'No records' in done
)
print('OK' if ok else 'FAIL')
" "$BOARD_BODY")
[ "$V14T2_LANES" = "OK" ]
check T176 "CAP-V14" "the board groups records into their current lane, and an unused lane still renders empty (got $V14T2_LANES)" $?

# T177 -- POST .../board-move updates the group field AND appends the record
# to the end of the target lane's own sort order -- the "one action, two
# writes" a kanban card-drop needs (RecordStore.MoveToLane).
DRAFT_CONTRACT_ID='22222222-3333-4444-5555-000000000003'
MOVE_URL=$(post_redirect "$BASE_URL/mch_kanban_task/$DRAFT_CONTRACT_ID/board-move" "lane=Done" "$KB_LEAD")
case "$MOVE_URL" in
    */mch_kanban_task/board) HAS_REDIRECT=0 ;;
    *) HAS_REDIRECT=1 ;;
esac
AFTER_MOVE_BODY=$(get_body "$BASE_URL/mch_kanban_task/board" "$KB_LEAD")
V14T2_MOVED=$(python3 -c "
import sys
body = sys.argv[1]
def section(lane):
    marker = 'data-lane=\"' + lane + '\"'
    start = body.find(marker)
    if start == -1:
        return ''
    nxt = body.find('data-lane=\"', start + len(marker))
    return body[start:nxt if nxt != -1 else len(body)]
doing, done = section('Doing'), section('Done')
ok = 'Draft contract' not in doing and 'Draft contract' in done
print('OK' if ok else 'FAIL')
" "$AFTER_MOVE_BODY")
[ "$HAS_REDIRECT" -eq 0 ] && [ "$V14T2_MOVED" = "OK" ]
check T177 "CAP-V14" "POST .../board-move moves the record into the target lane (got redirect=$MOVE_URL, moved=$V14T2_MOVED)" $?

