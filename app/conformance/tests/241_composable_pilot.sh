#!/usr/bin/env bash
# composable-runtime-roadmap.md §17a (Live Wiring Pilot) -- the first real
# HTTP route driven end-to-end by internal/composable (LowerCardRowComponent
# + ResolveRecordSummary) over real seeded Postgres data, additive to
# mch_approval_document's own vw_ad_all (display: cards, seeds/048), same
# machine T248 (210_v02t2_v09t2.sh) already proves the real cards feature
# against. As of 17e (T268/T269 below), this route reuses the exact same
# RecordSummaryCard/StatusBadge/Avatar rendering the real cards feature
# does, proven equivalent, not just visually similar. As of 17f
# (T270/T271), sort/search/pagination are wired through the same shared
# helpers (list_query.go) the real List route uses. Reuses ALICE/
# ALICE_ID (010's own resolution) and FRANK (HR, app_hr -- no role at all
# on app_approval, same "wrong app" shape as every other cross-app 403
# test in this suite).

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
# naive=4 collapses to dedup=3 in one ExecutionGroup.
#
# Status update (2026-09-12, composable-runtime-roadmap.md 17k): vw_ad_page
# now also carries a fourth Children entry, a component+dataset slot bound
# to ds_ad_steps_by_document (seeds/053_composable_data_plane_lab.sql) --
# a Dataset over mch_approval_step, a DIFFERENT Machine than the page's own
# host. That's a real, deliberate cross-machine composition (17k's own
# Experience-plane closure), so the live plan now has a SECOND
# ExecutionGroup too: naive=5 dedup=4 across two groups
# (mch_approval_document: 3, mch_approval_step: 1) -- the assertion below
# was updated to match, not loosened to hide it (same update already made
# to planner_seed_test.go's own Go-test counterpart).
#
# Status update (2026-09-12, composable-runtime-roadmap.md 17m):
# vw_ad_detail (a real `detail`-type View) now also carries its own real
# Dataset -- BuildDatasetFromView used to error for ViewTypeDetail, so
# this view_ref node's own Dataset was previously nil, invisible to the
# DAG. 17m's own Detail-Page Composition Pilot closes that gap: a genuine
# FOURTH node on mch_approval_document's own group, naive=6 dedup=5.
CPLAN_BODY=$(get_body "$BASE_URL/mch_approval_document/composable-preview" "$ALICE")
printf '%s' "$CPLAN_BODY" | grep -q 'data-composable-plan="ExecutionPlan: 2 group(s), naive=6 dedup=5' \
  && printf '%s' "$CPLAN_BODY" | grep -q 'mch_approval_document: 4 node(s)' \
  && printf '%s' "$CPLAN_BODY" | grep -q 'mch_approval_step: 1 node(s)'
check T265 "composable-runtime-roadmap.md §17b" "Dependency DAG/Execution Planner runs on this live request (2 groups, naive=6 dedup=5, mch_approval_document: 4 node(s), mch_approval_step: 1 node(s))" $?

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

# T268/T269 -- composable-runtime-roadmap.md §17e: real render-output
# equivalence, not just "a card renders." vw_ad_all's own columns are
# [fld_ad_title, fld_ad_document_type, fld_ad_approval_mode, fld_ad_status]
# -- the LAST three are all value_list-typed (list.templ's own cardSummary
# comment names this exact scenario: "Document Type ... and Status ... are
# both badge-eligible"), so CardBadgeField must pick fld_ad_status (the
# last one), leaving document_type+approval_mode to join the subtitle.
# initials() is a pure function of the title (first letter of first word +
# first letter of last word) -- computed here from the same title used to
# create the record, not guessed. "Draft" is the real, evented default for
# a never-submitted document (evt_ad_submit's own precondition requires
# status="Draft" to fire at all; T254b already established this fact).
CEQ_INITIALS="T$(printf '%s' "$$" | cut -c1)"
CEQ_DATA="fld_ad_title=T268+Equivalence+$$&fld_ad_document_type=Report&fld_ad_file=t268.pdf&fld_ad_submitted_by=$ALICE_ID&fld_ad_approval_mode=Sequential"
post_redirect "$BASE_URL/mch_approval_document" "$CEQ_DATA" "$ALICE" >/dev/null

# T268 -- ground truth: the REAL cards page shows exactly these three
# facts for this record (T248 already proves a card renders at all; this
# is the specific avatar/subtitle-join/badge shape this equivalence claim
# rests on).
CEQ_REAL_BODY=$(get_body "$BASE_URL/mch_approval_document" "$ALICE")
printf '%s' "$CEQ_REAL_BODY" | grep -q ">$CEQ_INITIALS<" \
  && printf '%s' "$CEQ_REAL_BODY" | grep -q "Report · Sequential" \
  && printf '%s' "$CEQ_REAL_BODY" | grep -q ">Draft<"
check T268 "composable-runtime-roadmap.md §17e" "the real cards page shows initials=$CEQ_INITIALS, subtitle=Report · Sequential, badge=Draft (ground truth)" $?

# T269 -- the composable-preview route now reproduces the SAME three facts
# for the SAME record, via LowerCardRowComponent/CardBadgeField/
# ResolveRecordSummary/LowerStatusBadge/ResolveStatusValue -- not merely a
# visually-similar approximation, the actual cardSummary logic.
CEQ_COMPOSABLE_BODY=$(get_body "$BASE_URL/mch_approval_document/composable-preview" "$ALICE")
printf '%s' "$CEQ_COMPOSABLE_BODY" | grep -q ">$CEQ_INITIALS<" \
  && printf '%s' "$CEQ_COMPOSABLE_BODY" | grep -q "Report · Sequential" \
  && printf '%s' "$CEQ_COMPOSABLE_BODY" | grep -q ">Draft<"
check T269 "composable-runtime-roadmap.md §17e" "composable-preview reproduces the same initials/subtitle/badge -- real render-output equivalence, not an approximation" $?

# T270 -- composable-runtime-roadmap.md §17f: free-text search (?q=) is now
# wired via searchListRecords (extracted from record_crud.go's own List
# into list_query.go, so both routes share the identical substring/case-
# insensitive match, not two independently-maintained copies) -- proves
# the query param is genuinely applied on this route, not merely present
# in the code.
CSEARCH_TITLE="T270 Search Wiring $$"
CSEARCH_DATA="fld_ad_title=${CSEARCH_TITLE// /+}&fld_ad_document_type=Report&fld_ad_file=t270.pdf&fld_ad_submitted_by=$ALICE_ID&fld_ad_approval_mode=Sequential"
post_redirect "$BASE_URL/mch_approval_document" "$CSEARCH_DATA" "$ALICE" >/dev/null
CSEARCH_MATCH_BODY=$(get_body "$BASE_URL/mch_approval_document/composable-preview?q=T270+Search+Wiring+$$" "$ALICE")
CSEARCH_NOMATCH_BODY=$(get_body "$BASE_URL/mch_approval_document/composable-preview?q=NoSuchTitleAnywhere$$" "$ALICE")
printf '%s' "$CSEARCH_MATCH_BODY" | grep -q "T270 Search Wiring $$" \
  && ! printf '%s' "$CSEARCH_NOMATCH_BODY" | grep -q "T270 Search Wiring $$"
check T270 "composable-runtime-roadmap.md §17f" "?q= is genuinely wired -- a matching query keeps the record, a non-matching query excludes it" $?

# T271 -- composable-runtime-roadmap.md §17f: pagination (?page=) is now
# wired via paginateListRecords (list_query.go) -- ?page=999 (far beyond
# any real page count) clamps to the actual last page and still shows real
# content, not an empty/error response. 26 fresh records are created here
# (pageSize=25) so mch_approval_document is guaranteed to have more than
# one page regardless of how many earlier tests happened to create,
# rather than relying on incidental accumulation from the rest of the
# suite.
for CPAGE_I in $(seq 1 26); do
  post_redirect "$BASE_URL/mch_approval_document" "fld_ad_title=T271+Page+Filler+${$}_${CPAGE_I}&fld_ad_document_type=Report&fld_ad_file=t271.pdf&fld_ad_submitted_by=$ALICE_ID&fld_ad_approval_mode=Sequential" "$ALICE" >/dev/null
done
CPAGE_BODY=$(get_body "$BASE_URL/mch_approval_document/composable-preview?page=999" "$ALICE")
! printf '%s' "$CPAGE_BODY" | grep -q "No records yet" \
  && printf '%s' "$CPAGE_BODY" | grep -qE 'Page [0-9]+ of [0-9]+'
check T271 "composable-runtime-roadmap.md §17f" "?page=999 clamps to the real last page with real content, via genuine pagination wiring" $?

# T272/T273/T274 -- composable-runtime-roadmap.md §17g: the REAL,
# standalone List route (not /composable-preview) now runs on the
# composable substrate for this cards-display View -- the actual
# cutover, not just another preview. T272 checks the honest, checkable
# signal (data-composable-plan, the same diagnostic 17b already proved on
# the preview route) now appears on the real route too -- proof that THIS
# request was genuinely served by listCardsViaComposable/
# resolveComposableCardSummaries/explainComposablePlan, not merely "still
# shows a card correctly" (which T248 already established and would hold
# identically regardless of which code path renders it). T273/T274 prove
# the cutover is functionally complete, not just carrying a diagnostic
# marker: search and pagination (17f's own proof shape) work identically
# on the real route now, reusing the T270/T271 records already created.
CCUTOVER_BODY=$(get_body "$BASE_URL/mch_approval_document" "$ALICE")
printf '%s' "$CCUTOVER_BODY" | grep -q 'data-composable-plan="ExecutionPlan: '
check T272 "composable-runtime-roadmap.md §17g" "the real /mch_approval_document List route now runs through the composable substrate (data-composable-plan present), not just a preview" $?

CCUTOVER_SEARCH_BODY=$(get_body "$BASE_URL/mch_approval_document?q=T270+Search+Wiring+$$" "$ALICE")
printf '%s' "$CCUTOVER_SEARCH_BODY" | grep -q "T270 Search Wiring $$"
check T273 "composable-runtime-roadmap.md §17g" "?q= also works correctly on the real cutover route, not just the preview" $?

CCUTOVER_PAGE_BODY=$(get_body "$BASE_URL/mch_approval_document?page=999" "$ALICE")
! printf '%s' "$CCUTOVER_PAGE_BODY" | grep -q "No records yet" \
  && printf '%s' "$CCUTOVER_PAGE_BODY" | grep -qE 'Page [0-9]+ of [0-9]+'
check T274 "composable-runtime-roadmap.md §17g" "?page=999 also clamps correctly on the real cutover route, not just the preview" $?
