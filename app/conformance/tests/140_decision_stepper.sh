#!/usr/bin/env bash
# CAP-V20: sequential decision stepper (Study 32,
# ../../benchmarks/024-pdf-signature-approval-study.md §3). Sourced by
# run.sh after lib.sh -- reuses ALICE/BOB/CAROL and their pre-resolved
# *_ID, same as 120_pdf_signature.sh/130_coord_placement.sh.
# seeds/037_decision_stepper_lab.sql provides the new View (vw_ad_progress)
# on mch_approval_document; no new fields, no new write path -- this batch
# proves REUSE (PermittedEventsForRecord's own CAP-P02 filter, the existing
# event-trigger route) renders and drives correctly from a page whose own
# URL machine is the PARENT, not the child Step being decided.
#
# occurrence counts below use `grep -o ... | wc -l`, not `grep -c` --
# grep -c counts MATCHING LINES, not total occurrences, and this
# templ's output can render more than one match on the same line.

# T212 -- Sequential, initial state: Step 1 (Bob) renders Current with a
# real Approve button, Step 2 (Carol) renders Pending with none -- checked
# from BOTH sides (Bob sees his own button; Carol, viewing the SAME page,
# sees her still-Pending step with none) before anything is approved, so
# neither assertion is stale by the time it runs.
AD_DATA="fld_ad_title=Stepper+Seq+$$&fld_ad_document_type=Policy&fld_ad_file=x.pdf&fld_ad_submitted_by=$ALICE_ID&fld_ad_approval_mode=Sequential"
AD_URL=$(post_redirect "$BASE_URL/mch_approval_document" "$AD_DATA" "$ALICE")
AD_ID="${AD_URL##*/}"
post_redirect "$BASE_URL/mch_approval_step" "fld_as_document=$AD_ID&fld_as_approver=$BOB_ID&fld_as_sequence=1" "$ALICE" >/dev/null
post_redirect "$BASE_URL/mch_approval_step" "fld_as_document=$AD_ID&fld_as_approver=$CAROL_ID&fld_as_sequence=2" "$ALICE" >/dev/null

PROG_BODY=$(get_body "$BASE_URL/mch_approval_document/$AD_ID/progress" "$BOB")
HAS_CURRENT=$(echo "$PROG_BODY" | grep -o '>Current<' | wc -l | tr -d ' ')
HAS_PENDING=$(echo "$PROG_BODY" | grep -o '>Pending<' | wc -l | tr -d ' ')
HAS_APPROVE_BTN=$(echo "$PROG_BODY" | grep -o 'Approve<' | wc -l | tr -d ' ')
APPROVE_ACTION=$(echo "$PROG_BODY" | grep -oE 'action="/[a-z0-9_-]+/mch_approval_step/[a-f0-9-]+/events/evt_as_approve"' | head -1 | sed -E 's/action="(.*)"/\1/')
CAROL_INITIAL_BTNS=$(get_body "$BASE_URL/mch_approval_document/$AD_ID/progress" "$CAROL" | grep -o 'Approve<' | wc -l | tr -d ' ')
[ "$HAS_CURRENT" = "1" ] && [ "$HAS_PENDING" = "1" ] && [ "$HAS_APPROVE_BTN" -ge 1 ] && [ -n "$APPROVE_ACTION" ] && [ "$CAROL_INITIAL_BTNS" = "0" ]
check T212 "CAP-V20" "Sequential initial state: Bob's own Current step has a real Approve button, Carol's still-Pending step has none (from her own view of the same page)" $?

# T213 -- approving through the stepper's own rendered button advances the
# state -- recomputed fresh from nothing but the children's own fields, no
# separate stepper state to keep in sync.
APPROVE_CODE=$(post_status "$ORIGIN$APPROVE_ACTION" "" "$BOB")
AFTER_BODY=$(get_body "$BASE_URL/mch_approval_document/$AD_ID/progress" "$BOB")
HAS_DONE=$(echo "$AFTER_BODY" | grep -o '>Done<' | wc -l | tr -d ' ')
NOW_CURRENT=$(echo "$AFTER_BODY" | grep -o '>Current<' | wc -l | tr -d ' ')
[ "$APPROVE_CODE" = "303" ] && [ "$HAS_DONE" = "1" ] && [ "$NOW_CURRENT" = "1" ]
check T213 "CAP-V20" "approving through the stepper's own rendered button (got $APPROVE_CODE) advances Step 1 to Done, Step 2 to Current" $?

# T214 -- now that Step 2 has legitimately become current, Carol (its real
# owner) sees her OWN button appear -- same reuse of PermittedEventsFor
# Record, this time proving it grants rather than withholds.
CAROL_NOW_BTNS=$(get_body "$BASE_URL/mch_approval_document/$AD_ID/progress" "$CAROL" | grep -o 'Approve<' | wc -l | tr -d ' ')
[ "$CAROL_NOW_BTNS" -ge 1 ]
check T214 "CAP-V20" "once Step 2 becomes current, its real owner (Carol) sees her own Approve button appear" $?

# T215 -- Parallel: both Steps render Current simultaneously (CAP-A07's own
# no-gating behavior for that mode, reflected here without re-deriving it).
AD2_DATA="fld_ad_title=Stepper+Par+$$&fld_ad_document_type=Policy&fld_ad_file=x.pdf&fld_ad_submitted_by=$ALICE_ID&fld_ad_approval_mode=Parallel"
AD2_URL=$(post_redirect "$BASE_URL/mch_approval_document" "$AD2_DATA" "$ALICE")
AD2_ID="${AD2_URL##*/}"
post_redirect "$BASE_URL/mch_approval_step" "fld_as_document=$AD2_ID&fld_as_approver=$BOB_ID&fld_as_sequence=1" "$ALICE" >/dev/null
post_redirect "$BASE_URL/mch_approval_step" "fld_as_document=$AD2_ID&fld_as_approver=$CAROL_ID&fld_as_sequence=2" "$ALICE" >/dev/null
PAR_CURRENT=$(get_body "$BASE_URL/mch_approval_document/$AD2_ID/progress" "$BOB" | grep -o '>Current<' | wc -l | tr -d ' ')
[ "$PAR_CURRENT" = "2" ]
check T215 "CAP-V20" "Parallel: both Steps render Current simultaneously (got $PAR_CURRENT), no sequential gating" $?
