#!/usr/bin/env bash
# composable-runtime-roadmap.md 17h -- the first live Project Management
# footprint on internal/composable: mch_pm_card's own dynamic-lane Board
# (vw_pmc_board, group_field: fld_pmc_list, CAP-V14) rendered read-only
# via /composable-preview, using the new ResolveBoardLanes View Model
# (viewmodel_resolve.go) -- deliberately narrower than the real
# board.templ (plain cell text, no drag-and-drop) -- see
# ComposablePreview's own handler doc comment for why. Reuses PM_MEMBER/
# TODO_LIST_ID/DOING_LIST_ID/DONE_LIST_ID/WIREFRAME_CARD_ID, already
# resolved into shared shell scope by 240_dynamic_board_lanes.sh (file
# order 240 < 242, sourced first by run.sh) -- written against the
# POST-T260 state: that file's own T260 already moved WIREFRAME_CARD_ID
# from TODO_LIST_ID into DOING_LIST_ID by the time this file runs.

# T275 -- the composable-preview route now reaches Board rendering at
# all for mch_pm_card (200, real lane markup present) -- before this
# session it always fell through to the empty-grid fallback (17c),
# since DefaultListView never returns a board-type View.
CPMB_BODY=$(get_body "$BASE_URL/mch_pm_card/composable-preview" "$PM_MEMBER")
printf '%s' "$CPMB_BODY" | grep -q "data-lane=\"$TODO_LIST_ID\"" \
  && printf '%s' "$CPMB_BODY" | grep -q "data-lane=\"$DOING_LIST_ID\"" \
  && printf '%s' "$CPMB_BODY" | grep -q "data-lane=\"$DONE_LIST_ID\""
check T275 "composable-runtime-roadmap.md §17h" "composable-preview now renders real Board lanes for mch_pm_card, not the empty-grid fallback" $?

# T276 -- lane grouping matches the real board's own current state
# (post-T260: Wireframe homepage has already moved into Doing), and lane
# headers show real display names (via displayLabel reuse -- the same
# label logic the real board itself uses), not raw list ids.
python3 -c "
import sys
body = sys.argv[1]
todo_id, doing_id, done_id = sys.argv[2], sys.argv[3], sys.argv[4]
def section(lane_id):
    marker = 'data-lane=\"' + lane_id + '\"'
    start = body.find(marker)
    if start == -1:
        sys.exit(1)
    nxt = body.find('data-lane=\"', start + len(marker))
    return body[start:nxt if nxt != -1 else len(body)]
todo, doing, done = section(todo_id), section(doing_id), section(done_id)
ok = (
    '>To Do' in todo and 'Collect brand assets' in todo
    and 'Wireframe homepage' not in todo and 'Build landing page' not in todo
    and '>Doing' in doing and 'Wireframe homepage' in doing and 'Build landing page' in doing
    and '>Done' in done and 'Kickoff meeting notes' in done
)
sys.exit(0 if ok else 1)
" "$CPMB_BODY" "$TODO_LIST_ID" "$DOING_LIST_ID" "$DONE_LIST_ID"
check T276 "composable-runtime-roadmap.md §17h" "lane grouping matches the real board's current state (post board-move), with real display-name headers" $?

# T277 -- a fresh, deliberately empty List still renders as an empty lane
# -- the concrete, real-HTTP form of ResolveBoardLanes' own "every lane
# renders, even with zero cards" guarantee (CAP-V14's "an unused lane
# still renders empty" rule) -- T259 itself never exercises this since
# every one of its own three lanes already has at least one card.
BACKLOG_URL=$(post_redirect "$BASE_URL/mch_pm_list" "fld_pml_board=33333333-4444-5555-6666-000000000001&fld_pml_name=Backlog+$$" "$PM_MEMBER")
BACKLOG_ID="${BACKLOG_URL##*/}"
CPMB_BODY2=$(get_body "$BASE_URL/mch_pm_card/composable-preview" "$PM_MEMBER")
python3 -c "
import sys
body, backlog_id = sys.argv[1], sys.argv[2]
marker = 'data-lane=\"' + backlog_id + '\"'
start = body.find(marker)
if start == -1:
    sys.exit(1)
nxt = body.find('data-lane=\"', start + len(marker))
section = body[start:nxt if nxt != -1 else len(body)]
sys.exit(0 if ('No records' in section and 'Backlog' in section) else 1)
" "$CPMB_BODY2" "$BACKLOG_ID"
check T277 "composable-runtime-roadmap.md §17h" "a fresh, empty List still renders as an empty lane, not a missing one" $?

# T278/T279 -- composable-runtime-roadmap.md §17i: visual equivalence,
# not just "a lane renders." Both checks look for the same real card
# (WIREFRAME_CARD_ID) rendered as a real link to its own record, carrying
# board.templ's own visual classes (rounded-md/border-slate-200/bg-white/
# p-3/shadow-sm/hover:shadow) -- drag-only classes (board-card,
# cursor-move) are deliberately excluded from this check, since
# composable_preview.templ never claims drag support (see its own doc
# comment). T278 is the ground truth (the real board); T279 proves
# composable-preview reproduces it.
BOARD_CARD_CHECK='
import re, sys
body, record_id = sys.argv[1], sys.argv[2]
m = re.search(r"<a[^>]*href=\"[^\"]*" + re.escape(record_id) + r"\"[^>]*>", body)
if not m:
    sys.exit(1)
required = ["rounded-md", "border-slate-200", "bg-white", "p-3", "shadow-sm", "hover:shadow"]
sys.exit(0 if all(c in m.group(0) for c in required) else 1)
'
REAL_BOARD_BODY=$(get_body "$BASE_URL/mch_pm_card/board" "$PM_MEMBER")
python3 -c "$BOARD_CARD_CHECK" "$REAL_BOARD_BODY" "$WIREFRAME_CARD_ID"
check T278 "composable-runtime-roadmap.md §17i" "the real board renders WIREFRAME_CARD_ID as a real link with board.templ's own visual classes (ground truth)" $?

CPMB_BODY3=$(get_body "$BASE_URL/mch_pm_card/composable-preview" "$PM_MEMBER")
python3 -c "$BOARD_CARD_CHECK" "$CPMB_BODY3" "$WIREFRAME_CARD_ID"
check T279 "composable-runtime-roadmap.md §17i" "composable-preview reproduces the same real link + visual classes -- real equivalence, not a passing resemblance" $?
