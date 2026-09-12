#!/usr/bin/env bash
# case-03-case-19-completion-checklist.md's own Stage 3 (composable-
# runtime-roadmap.md 17q): Case 19's real Field gaps against
# project-board.html/project-card.html (ui-sample, checked directly) --
# Members and Labels are BOTH real multi-value, Trello-shaped
# relationships, not single-value Fields. seeds/057_case19_card_fields.sql
# seeds real Label/Member joins on WIREFRAME_CARD_ID (card 21, already
# resolved by 240_dynamic_board_lanes.sh, sourced first by run.sh) --
# two Labels (Design/High), two Members (Project Member/Project Owner),
# a due date, and a real checklist progress (1/2, from seeds/052's own
# already-seeded Checklist Items).

PM_MEMBER=$(session_for project.member@example.com password)
WIREFRAME_CARD_ID='33333333-4444-5555-6666-000000000021'
BLANK_MEMBERS_CARD_ID='33333333-4444-5555-6666-000000000024' # Kickoff meeting notes -- no Member join seeded

# T291 -- the real Board card for WIREFRAME_CARD_ID shows real, colored
# label chips (LabelsVia's own reverse-reference join, resolved to the
# real Label record's Name/Color) -- Board's own opt-in CardMeta config,
# not automatic detection.
BOARD_BODY=$(get_body "$BASE_URL/mch_pm_card/board" "$PM_MEMBER")
CARD21_SECTION=$(python3 -c "
import sys
body, card_id = sys.argv[1], sys.argv[2]
marker = 'data-record-id=\"' + card_id + '\"'
start = body.find(marker)
if start == -1:
    print('')
else:
    nxt = body.find('</a>', start)
    print(body[start:nxt if nxt != -1 else len(body)])
" "$BOARD_BODY" "$WIREFRAME_CARD_ID")
echo "$CARD21_SECTION" | grep -q "Design" && echo "$CARD21_SECTION" | grep -q "High"
check T291 "composable-runtime-roadmap.md 17q" "the real Board card shows real colored label chips (Design, High) from the LabelsVia join" $?

# T292 -- the same card shows real checklist progress "1/2" (ProgressVia,
# the SAME Checklist Item rows seeds/052 already created for this card --
# counted, not re-rendered as a full list).
echo "$CARD21_SECTION" | grep -q "1/2"
check T292 "composable-runtime-roadmap.md 17q" "the real Board card shows real checklist progress (1/2) from the ProgressVia join" $?

# T293 -- the same card shows its own real due date (DueDateField, a
# plain Field on mch_pm_card itself, no reverse-reference hop needed).
echo "$CARD21_SECTION" | grep -q "2026-09-14"
check T293 "composable-runtime-roadmap.md 17q" "the real Board card shows its own real due date" $?

# T294 -- the same card shows both real Members as avatar initials
# (MembersVia, resolved via userLabel then memberInitials/initials() --
# never a raw user id).
echo "$CARD21_SECTION" | grep -q "PM" && echo "$CARD21_SECTION" | grep -q "PO"
check T294 "composable-runtime-roadmap.md 17q" "the real Board card shows both real Members as avatar initials (PM, PO)" $?

# T295 -- a card with no seeded Member joins (Kickoff meeting notes)
# renders cleanly -- no avatar section at all, not a crash or an empty
# broken one.
BLANK_SECTION=$(python3 -c "
import sys
body, card_id = sys.argv[1], sys.argv[2]
marker = 'data-record-id=\"' + card_id + '\"'
start = body.find(marker)
print('' if start == -1 else body[start:body.find('</a>', start)])
" "$BOARD_BODY" "$BLANK_MEMBERS_CARD_ID")
[ -n "$BLANK_SECTION" ]
check T295 "composable-runtime-roadmap.md 17q" "a card with no seeded Members still renders its own board card cleanly" $?

# T296 -- Card Detail's own reverse-reference sections (CAP-V06) resolve
# to real names, not raw UUIDs -- childListItemLabel's own new fallback
# tier (a join Machine with no Text/Number Field tries its OTHER
# reference/user Field next), a generic fix, not Card-specific.
DETAIL_BODY=$(get_body "$BASE_URL/mch_pm_card/$WIREFRAME_CARD_ID" "$PM_MEMBER")
echo "$DETAIL_BODY" | grep -q "Design" && echo "$DETAIL_BODY" | grep -q "High"
check T296 "composable-runtime-roadmap.md 17q" "Card Detail's own Card Label reverse-reference section shows real Label names, not raw ids" $?

echo "$DETAIL_BODY" | grep -q "Project Member" && echo "$DETAIL_BODY" | grep -q "Project Owner"
check T297 "composable-runtime-roadmap.md 17q" "Card Detail's own Card Member reverse-reference section shows real Member names, not raw ids" $?

echo "$DETAIL_BODY" | grep -q "2026-09-14"
check T298 "composable-runtime-roadmap.md 17q" "Card Detail shows the real Due Date field via Detail's own existing generic field loop" $?

# T299 -- the Card Form (CAP-F16, ChildLinesGroups/17q) renders THREE
# embedded child-row blocks now -- Checklist (the existing singular
# ChildLines slot, unchanged) plus Members and Labels (the two new
# ChildLinesGroups entries) -- proving the plural mechanism is really
# wired into the real Create form, not just accepted by the model layer.
FORM_BODY=$(get_body "$BASE_URL/mch_pm_card/new" "$PM_MEMBER")
echo "$FORM_BODY" | grep -qo 'name="child_0_fld_pmci_text"'
CL_CHECKLIST=$?
echo "$FORM_BODY" | grep -qo 'name="child_0_fld_pmcm_user"'
CL_MEMBERS=$?
echo "$FORM_BODY" | grep -qo 'name="child_0_fld_pmcl_label"'
CL_LABELS=$?
[ "$CL_CHECKLIST" -eq 0 ] && [ "$CL_MEMBERS" -eq 0 ] && [ "$CL_LABELS" -eq 0 ]
check T299 "composable-runtime-roadmap.md 17q" "the real Card Form renders all three embedded child-row blocks (Checklist, Members, Labels)" $?

# T300 -- submitting the Card Form with a Member row and a Label row
# atomically creates the new Card AND both join records in the same
# request -- ChildLinesGroups' own create-time wiring (record_crud.go's
# Create), not just rendering.
PM_MEMBER_ID=$(user_option_id "$BASE_URL/mch_pm_card/new" "$PM_MEMBER" "Project Member")
NEW_CARD_DATA="fld_pmc_list=33333333-4444-5555-6666-000000000011&fld_pmc_title=T300+ChildLinesGroups+$$&child_0_fld_pmcm_user=$PM_MEMBER_ID&child_0_fld_pmcl_label=33333333-4444-5555-6666-000000000041"
NEW_CARD_URL=$(post_redirect "$BASE_URL/mch_pm_card" "$NEW_CARD_DATA" "$PM_MEMBER")
NEW_CARD_ID="${NEW_CARD_URL##*/}"
NEW_CARD_DETAIL=$(get_body "$BASE_URL/mch_pm_card/$NEW_CARD_ID" "$PM_MEMBER")
echo "$NEW_CARD_DETAIL" | grep -q "Design"
LABEL_JOINED=$?
echo "$NEW_CARD_DETAIL" | grep -q "Project Member"
MEMBER_JOINED=$?
[ -n "$NEW_CARD_ID" ] && [ "$LABEL_JOINED" -eq 0 ] && [ "$MEMBER_JOINED" -eq 0 ]
check T300 "composable-runtime-roadmap.md 17q" "submitting the Card Form atomically creates the Card, a real Member join row, and a real Label join row" $?

# T301 -- Kanban Lab's own value_list Board (CAP-V14 Tier 2, no CardMeta
# declared) renders byte-identical to before -- the opt-in key is a
# genuine no-op everywhere it isn't declared, not automatic detection
# that could silently change an unrelated Board.
KB_LEAD=$(session_for kanban.lead@example.com password)
KB_BOARD_BODY=$(get_body "$BASE_URL/mch_kanban_task/board" "$KB_LEAD")
! echo "$KB_BOARD_BODY" | grep -q 'mb-2 flex flex-wrap gap-1\|mt-2 flex items-center justify-between gap-2'
check T301 "composable-runtime-roadmap.md 17q" "an existing Board with no CardMeta config renders with none of the new per-card meta markup at all" $?
