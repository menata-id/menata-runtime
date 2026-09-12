#!/usr/bin/env bash
# CAP-V14 Tier 3: a `board` View whose group_field is a `reference` Field
# groups records into one lane per real record of the reference's own
# target machine (dynamic, user-creatable lanes) instead of a fixed
# value_list's Options.Values -- seeds/052_project_management.sql's own
# Card board (vw_pmc_board, group_field: fld_pmc_list) is the real proof
# case (roadmap.md Study 41, capability-registry.md's CAP-V14 Tier 2 row).
# The drag gesture itself is client-side JS this HTTP-black-box suite
# cannot execute, but the write it triggers (BoardMove) is an ordinary
# POST -- both halves below are fully HTTP-testable without a browser,
# same posture T176/T177 (CAP-V14 Tier 2) already established.

PM_MEMBER=$(session_for project.member@example.com password)

TODO_LIST_ID='33333333-4444-5555-6666-000000000011'
DOING_LIST_ID='33333333-4444-5555-6666-000000000012'
DONE_LIST_ID='33333333-4444-5555-6666-000000000013'
WIREFRAME_CARD_ID='33333333-4444-5555-6666-000000000021'

# T259 -- one lane per real List record (dynamic, not a fixed
# value_list), lane header shows the List's own display label (not its
# raw id), and each Card renders in the lane matching its own
# fld_pmc_list value.
BOARD_BODY=$(get_body "$BASE_URL/mch_pm_card/board" "$PM_MEMBER")
V14T3_LANES=$(python3 -c "
import sys
body = sys.argv[1]
todo_id, doing_id, done_id = sys.argv[2], sys.argv[3], sys.argv[4]
def section(lane_id):
    marker = 'data-lane=\"' + lane_id + '\"'
    start = body.find(marker)
    if start == -1:
        return ''
    nxt = body.find('data-lane=\"', start + len(marker))
    return body[start:nxt if nxt != -1 else len(body)]
todo, doing, done = section(todo_id), section(doing_id), section(done_id)
ok = (
    '>To Do' in todo and 'Wireframe homepage' in todo and 'Collect brand assets' in todo
    and 'Build landing page' not in todo
    and '>Doing' in doing and 'Build landing page' in doing
    and '>Done' in done and 'Kickoff meeting notes' in done
)
print('OK' if ok else 'FAIL')
" "$BOARD_BODY" "$TODO_LIST_ID" "$DOING_LIST_ID" "$DONE_LIST_ID")
[ "$V14T3_LANES" = "OK" ]
check T259 "CAP-V14" "a reference-typed group_field renders one lane per real target record, labeled by its own display name (got $V14T3_LANES)" $?

# T286 -- composable-runtime-roadmap.md 17n: this real route now renders
# through internal/composable (boardViaComposable) -- carries the same
# live Dependency DAG/Execution Planner diagnostic the List cutover
# (17g) already proves on its own real route, T272. T259 above, run
# against this exact same real page, is the actual equivalence proof
# (unchanged lane/card content); this only proves the plumbing is real,
# not a hardcoded diagnostic string.
printf '%s' "$BOARD_BODY" | grep -q 'data-composable-plan="ExecutionPlan: '
check T286 "composable-runtime-roadmap.md 17n" "the real Card Board route now carries a real Dependency DAG/Execution Planner diagnostic too" $?

# T260 -- POST .../board-move with a real target-machine record id moves
# the Card into that List (a real CAP-F13 referential-integrity write, the
# same MoveToLane mechanism T177 already proved, now over a dynamic lane).
MOVE_URL=$(post_redirect "$BASE_URL/mch_pm_card/$WIREFRAME_CARD_ID/board-move" "lane=$DOING_LIST_ID" "$PM_MEMBER")
case "$MOVE_URL" in
    */mch_pm_card/board) HAS_REDIRECT=0 ;;
    *) HAS_REDIRECT=1 ;;
esac
AFTER_MOVE_BODY=$(get_body "$BASE_URL/mch_pm_card/board" "$PM_MEMBER")
V14T3_MOVED=$(python3 -c "
import sys
body = sys.argv[1]
todo_id, doing_id = sys.argv[2], sys.argv[3]
def section(lane_id):
    marker = 'data-lane=\"' + lane_id + '\"'
    start = body.find(marker)
    if start == -1:
        return ''
    nxt = body.find('data-lane=\"', start + len(marker))
    return body[start:nxt if nxt != -1 else len(body)]
todo, doing = section(todo_id), section(doing_id)
ok = 'Wireframe homepage' not in todo and 'Wireframe homepage' in doing
print('OK' if ok else 'FAIL')
" "$AFTER_MOVE_BODY" "$TODO_LIST_ID" "$DOING_LIST_ID")
[ "$HAS_REDIRECT" -eq 0 ] && [ "$V14T3_MOVED" = "OK" ]
check T260 "CAP-V14" "POST .../board-move moves a Card into a real dynamic lane (got redirect=$MOVE_URL, moved=$V14T3_MOVED)" $?

# T261 -- a lane id that isn't a real record on the reference's own target
# machine is rejected (CAP-X05 "Unknown = explicit"), not written as an
# arbitrary string into the Card's own data.
BAD_LANE_STATUS=$(curl -s -o /dev/null -w '%{http_code}' -b "$PM_MEMBER" \
    -X POST "$BASE_URL/mch_pm_card/$WIREFRAME_CARD_ID/board-move" \
    --data-urlencode "lane=00000000-0000-0000-0000-000000000000" \
    --data-urlencode "csrf_token=$(csrf_for "$PM_MEMBER" "$BASE_URL/mch_pm_card/board")")
[ "$BAD_LANE_STATUS" = "400" ]
check T261 "CAP-V14" "board-move rejects a lane id that isn't a real record on the target machine (got HTTP $BAD_LANE_STATUS)" $?
