#!/usr/bin/env bash
# CAP-V10 Tier 2: a `page` View composing several independently-sourced
# Views (plus static content) onto one real screen -- seeds/
# 050_composed_dashboard.sql realizes approval-dashboard.html's own
# Summary + Pending Documents + Recent Activity shape. Sourced by run.sh
# after lib.sh and 130_coord_placement.sh -- reuses ALICE/ALICE_ID and
# post_redirect_multipart (both already in scope by this point).

CP_PDF=$(mktemp --suffix=.pdf)
cat > "$CP_PDF" << 'PDFEOF'
%PDF-1.4
1 0 obj<</Type/Catalog/Pages 2 0 R>>endobj
2 0 obj<</Type/Pages/Kids[3 0 R]/Count 1>>endobj
3 0 obj<</Type/Page/Parent 2 0 R/MediaBox[0 0 200 300]>>endobj
trailer<</Size 4/Root 1 0 R>>
%%EOF
PDFEOF

# A real In-Review Document -- should appear in the composed page's own
# Pending Documents section.
CP_REVIEW_URL=$(post_redirect_multipart "$BASE_URL/mch_approval_document" "$ALICE" \
    "fld_ad_title=Composed Page Review $$" "fld_ad_document_type=Policy" \
    "fld_ad_file=@$CP_PDF;type=application/pdf" \
    "fld_ad_submitted_by=$ALICE_ID" "fld_ad_approval_mode=Sequential")
CP_REVIEW_ID="${CP_REVIEW_URL##*/}"
post_status "$BASE_URL/mch_approval_document/$CP_REVIEW_ID/events/evt_ad_submit" "" "$ALICE" >/dev/null

# A Draft Document (never submitted) -- must NOT appear, proving the
# section's own Filter (Status = In Review) actually applies inside the
# composed page, not just on the standalone list.
post_redirect_multipart "$BASE_URL/mch_approval_document" "$ALICE" \
    "fld_ad_title=Composed Page Draft $$" "fld_ad_document_type=Policy" \
    "fld_ad_file=@$CP_PDF;type=application/pdf" \
    "fld_ad_submitted_by=$ALICE_ID" "fld_ad_approval_mode=Sequential" >/dev/null

CP_BODY=$(get_body "$BASE_URL/mch_approval_document/page" "$ALICE")

echo "$CP_BODY" | grep -q "Summary" && echo "$CP_BODY" | grep -q "Pending Documents" && echo "$CP_BODY" | grep -q "Recent Activity"
check T253 "CAP-V10" "the composed page renders all three declared sections (Summary/Pending Documents/Recent Activity)" $?

echo "$CP_BODY" | grep -q "Composed Page Review $$"
check T254a "CAP-V10" "the Pending Documents section includes a real In-Review document" $?

! echo "$CP_BODY" | grep -q "Composed Page Draft $$"
check T254b "CAP-V10" "the same section excludes a real Draft (never-submitted) document -- the section's own filter applies inside composition" $?

# Status update (2026-09-12, composable-runtime-roadmap.md 17p): CAP-R04's
# own "R28" gap this placeholder named is now closed
# (seeds/055_activity_log.sql) -- the Recent Activity section is real now,
# not a placeholder. By this point in the suite, earlier files (010's own
# submit flow, at minimum) have already fired real evt_ad_submit events on
# real Documents, so a real Event name is real content to check for
# instead of the placeholder's own former text.
! echo "$CP_BODY" | grep -q "Recent Activity requires a record-history"
check T255 "composable-runtime-roadmap.md 17p" "the Recent Activity section no longer shows the placeholder -- it's real now" $?

# T256 -- same CAP-X06 cross-workspace gate every other collection-level
# route already enforces (T49's own pattern) -- a ws_default account given
# a direct URL naming another workspace's Machine 404s on THIS new route
# too, not a separate check CAP-V10 Tier 2 could accidentally skip.
CP_DENIED_CODE=$(curl -s -o /dev/null -w '%{http_code}' -b "$ALICE" "$BASE_URL_ACME/mch_approval_document/page")
[ "$CP_DENIED_CODE" = "404" ]
check T256 "CAP-V10" "a Machine from another Workspace 404s on the composed page route too (got $CP_DENIED_CODE)" $?

# T257 -- the Pending Documents section has NO "View all ->" link: vw_ad_
# pending isn't the Machine's own DefaultListView, so a link to the bare
# Machine URL would silently show a DIFFERENT, unfiltered View instead
# (caught live re-verifying this same page) -- omitted rather than
# misleading, not just present-but-wrong.
! echo "$CP_BODY" | grep -q 'href="/ws_default/mch_approval_document"[^>]*>View all'
check T257 "CAP-V10" "the Pending Documents section has no misleading 'View all' link to a different, unfiltered View" $?

# T258 -- the sub-nav strip highlights EXACTLY ONE entry on the composed
# page (Dashboard, an exact path match), not two at once -- caught live:
# "Approval Document" (coarse Machine-target match) and "Dashboard" (exact
# View-target match) both lit up simultaneously before this fix.
CP_ACTIVE_COUNT=$(echo "$CP_BODY" | grep -o 'bg-white text-blue-700 shadow-sm">[[:space:]]*\(Approval Document\|Dashboard\)' | wc -l)
[ "$CP_ACTIVE_COUNT" -eq 1 ]
check T258 "CAP-O03" "the composed page's own sub-nav strip highlights exactly one entry, not two at once (got $CP_ACTIVE_COUNT)" $?

# T280 -- composable-runtime-roadmap.md 17l: the composed page's fourth
# section (component:"Metric" bound to ds_ad_total_steps,
# seeds/054_composable_metric_live_fix.sql) is the first real physical
# execution internal/composable has ever driven on a live route -- a real
# count over mch_approval_step's own records (internal/handler/
# page_component.go's renderPageComponentChild), not a diagnostic. Proven
# by before/after, not a hardcoded number (the exact T271 lesson from
# 17f): create a real, independently-known batch of Approval Steps, then
# confirm the rendered count increases by exactly that batch size --
# "Total Approval Steps</div>" uniquely matches MetricContent's own label
# div (page.templ), never SectionWrapper's <h3> title, so the very next
# line is always the real number.
CP_STEPS_BEFORE=$(printf '%s' "$CP_BODY" | tr -d '\n' | grep -oE 'Total Approval Steps</div><div class="mt-1 text-2xl font-semibold text-slate-900">[0-9]+' | grep -oE '[0-9]+$')
CP_STEPS_BATCH=3
for CP_SEQ in 1 2 3; do
    post_redirect "$BASE_URL/mch_approval_step" "fld_as_document=$CP_REVIEW_ID&fld_as_approver=$ALICE_ID&fld_as_sequence=$CP_SEQ" "$ALICE" >/dev/null
done
CP_BODY_AFTER=$(get_body "$BASE_URL/mch_approval_document/page" "$ALICE")
CP_STEPS_AFTER=$(printf '%s' "$CP_BODY_AFTER" | tr -d '\n' | grep -oE 'Total Approval Steps</div><div class="mt-1 text-2xl font-semibold text-slate-900">[0-9]+' | grep -oE '[0-9]+$')
[ -n "$CP_STEPS_BEFORE" ] && [ -n "$CP_STEPS_AFTER" ] && [ "$CP_STEPS_AFTER" -eq "$((CP_STEPS_BEFORE + CP_STEPS_BATCH))" ]
check T280 "composable-runtime-roadmap.md 17l" "the composed page's Total Approval Steps section is a real, live-computed count ($CP_STEPS_BEFORE -> $CP_STEPS_AFTER after creating $CP_STEPS_BATCH real steps)" $?

rm -f "$CP_PDF"
