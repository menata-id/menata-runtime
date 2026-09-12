#!/usr/bin/env bash
# composable-runtime-roadmap.md §17a (Live Wiring Pilot) -- the first real
# HTTP route driven end-to-end by internal/composable (LowerCardRowComponent
# + ResolveRecordSummary) over real seeded Postgres data, additive to
# mch_approval_document's own vw_ad_all (display: cards, seeds/048), same
# machine T248 (210_v02t2_v09t2.sh) already proves the real cards feature
# against. This route is a narrower preview (title+subtitle only, no status
# badge) -- see the handler's own doc comment for why that's a named
# limitation, not a bug. Reuses ALICE/ALICE_ID (010's own resolution) and
# FRANK (HR, app_hr -- no role at all on app_approval, same "wrong app"
# shape as every other cross-app 403 test in this suite).

# T262 -- a real Document's own title/document_type reach the page via the
# composable substrate (Dataset -> UINode -> ResolveRecordSummary), not a
# stub or empty page.
CPILOT_DATA="fld_ad_title=T262+Composable+Pilot+$$&fld_ad_document_type=Report&fld_ad_file=t262.pdf&fld_ad_submitted_by=$ALICE_ID&fld_ad_approval_mode=Sequential"
post_redirect "$BASE_URL/mch_approval_document" "$CPILOT_DATA" "$ALICE" >/dev/null
CPILOT_BODY=$(get_body "$BASE_URL/mch_approval_document/composable-preview" "$ALICE")
printf '%s' "$CPILOT_BODY" | grep -q "T262 Composable Pilot $$" && printf '%s' "$CPILOT_BODY" | grep -q "Report"
check T262 "composable-runtime-roadmap.md §17a" "a real record's title+subtitle reach the page via internal/composable's own Dataset/UINode/ResolveRecordSummary pipeline" $?

# T263 -- a role with no permission row on this app at all (FRANK, HR/
# app_hr) is denied exactly like any other per-machine route -- the pilot
# route reuses the same CanRead guard, not a bypass.
CPILOT_DENIED_CODE=$(curl -s -o /dev/null -w '%{http_code}' -b "$FRANK" "$BASE_URL/mch_approval_document/composable-preview")
[ "$CPILOT_DENIED_CODE" = "403" ]
check T263 "composable-runtime-roadmap.md §17a" "a role with no permission on this Machine's app is denied (got $CPILOT_DENIED_CODE)" $?

# T264 -- a Machine from another Workspace 404s on this route too (CAP-X06
# pattern, same as List/Board/Detail/Page).
CPILOT_CROSS_CODE=$(curl -s -o /dev/null -w '%{http_code}' -b "$ALICE" "$BASE_URL_ACME/mch_approval_document/composable-preview")
[ "$CPILOT_CROSS_CODE" = "404" ]
check T264 "composable-runtime-roadmap.md §17a" "a Machine from another Workspace 404s on the composable preview route too (got $CPILOT_CROSS_CODE)" $?

# T265 -- composable-runtime-roadmap.md §17b (Dependency DAG + Execution
# Planner on the live path): this route now also lowers mch_approval_document's
# FULL page (every View, not just the cards list rendered above) through
# BuildDependencyDAG + BuildExecutionPlan as a side-channel diagnostic --
# the same real, previously Go-test-only proof (planner_seed_test.go's own
# TestGroupByMachineAgainstApprovalCase/TestBuildExecutionPlanAgainstApprovalCase)
# now happening on an actual HTTP request. mch_approval_document has three
# distinct real Datasets (vw_ad_form, vw_ad_all, vw_ad_pending), with
# vw_ad_pending's own real duplicate consumption (Phase 7's dedup proof --
# once as an ordinary child, once inside vw_ad_page's own Slots["main"]):
# naive=4 collapses to dedup=3 in one ExecutionGroup. Exposed as a hidden
# data-composable-plan attribute, inspectable from the response body alone,
# no server-log access needed.
CPLAN_BODY=$(get_body "$BASE_URL/mch_approval_document/composable-preview" "$ALICE")
printf '%s' "$CPLAN_BODY" | grep -q 'data-composable-plan="ExecutionPlan: 1 group(s), naive=4 dedup=3' \
  && printf '%s' "$CPLAN_BODY" | grep -q 'mch_approval_document: 3 node(s)'
check T265 "composable-runtime-roadmap.md §17b" "Dependency DAG/Execution Planner runs on this live request (1 group, naive=4 dedup=3, mch_approval_document: 3 node(s))" $?

# T266 -- composable-runtime-roadmap.md §17c generalizes this route beyond
# the one machine with a cards-display List View: mch_approval_step has
# none (its only List, vw_as_progress, is a plain table), so before 17c
# this URL 400'd outright. It now returns 200 with an empty card grid,
# reaching this second real machine's own Dependency DAG/Execution Planner
# for the first time on the live path.
CSTEP_CODE=$(curl -s -o /dev/null -w '%{http_code}' -b "$ALICE" "$BASE_URL/mch_approval_step/composable-preview")
[ "$CSTEP_CODE" = "200" ]
check T266 "composable-runtime-roadmap.md §17c" "a machine with no cards-display List View still returns 200, not 400 (got $CSTEP_CODE)" $?

# T267 -- the same response still carries a real plan diagnostic (proves
# explainComposablePlan, 17b, ran against mch_approval_step -- not just
# that the route stopped 400ing). Deliberately NOT asserting specific
# node/group numbers: vw_as_detail's own embedded decision_stepper/
# coord_placement children (17c's own LowerPage fix) have no representable
# Dataset yet (BuildDatasetFromView has no case for either type), so the
# plan's naive/dedup counts here reflect only vw_as_form/vw_as_progress --
# unchanged by 17c's fix, by design (see composable-runtime-roadmap.md's
# own 17c section for why asserting otherwise here would be dishonest).
CSTEP_BODY=$(get_body "$BASE_URL/mch_approval_step/composable-preview" "$ALICE")
printf '%s' "$CSTEP_BODY" | grep -q 'data-composable-plan="ExecutionPlan: '
check T267 "composable-runtime-roadmap.md §17c" "the Dependency DAG/Execution Planner diagnostic runs against mch_approval_step too (a second real machine on the live path)" $?
