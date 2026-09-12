#!/usr/bin/env bash
# case-03-case-19-completion-checklist.md's own Stage A item 1
# (composable-runtime-roadmap.md 17r): closes Case 3's "SLA chips not
# wired" gap against document-approval.html (ui-sample) -- the mockup's
# own real card markup shows "OVERDUE"/"SLA breached · Nh" badges keyed
# on a real due date; mch_approval_document never had one at all.
# seeds/058_case3_sla_wiring.sql seeds two fixed-id demo Documents with
# real due dates (one far past -> overdue, one far future -> not).

# T302 -- the real List page (cards mode, the production composable
# cutover route, 17g) shows a real SLA-overdue badge for the far-past
# due date, not the raw date text. Scoped via ?q= (like T270's own
# search test) -- by the time this test runs, hundreds of other real
# Documents this whole suite creates have pushed the seeded one off
# page 1's own default_sort:created_at-desc ordering.
LIST_BODY=$(get_body "$BASE_URL/mch_approval_document?q=Vendor+Contract+Q3" "$ALICE")
echo "$LIST_BODY" | grep -q "Vendor Contract Q3" && echo "$LIST_BODY" | grep -q "Overdue by"
check T302 "composable-runtime-roadmap.md 17r" "the real List page shows a real SLA-overdue badge for a real far-past due date" $?

# T303 -- the same page shows a real "day(s) left" badge (not overdue)
# for the far-future due date.
LIST_BODY2=$(get_body "$BASE_URL/mch_approval_document?q=Procurement+SOP" "$ALICE")
echo "$LIST_BODY2" | grep -q "Procurement SOP" && echo "$LIST_BODY2" | grep -q "day(s) left"
check T303 "composable-runtime-roadmap.md 17r" "the same List page shows a real not-overdue SLA badge for a real far-future due date" $?

# T304 -- Detail (composable path, 17o) shows the same real SLA badge
# for its own Due Date field, not the raw date.
DETAIL_BODY=$(get_body "$BASE_URL/mch_approval_document/44444444-5555-6666-7777-000000000001" "$ALICE")
echo "$DETAIL_BODY" | grep -q "Overdue by"
check T304 "composable-runtime-roadmap.md 17r" "Detail shows the same real SLA badge for the real overdue Document" $?

# T305 -- composable-preview reproduces the exact same real badge --
# real equivalence, not a passing resemblance (same discipline T268/T269,
# T281/T282 already established for other fields).
PREVIEW_BODY=$(get_body "$BASE_URL/mch_approval_document/composable-preview?q=Vendor+Contract+Q3" "$ALICE")
echo "$PREVIEW_BODY" | grep -q "Vendor Contract Q3" && echo "$PREVIEW_BODY" | grep -q "Overdue by"
check T305 "composable-runtime-roadmap.md 17r" "composable-preview reproduces the same real SLA-overdue badge the live List page shows" $?

# T306 -- a real Document with NO due date at all (the common case --
# due date is optional, every pre-existing Document has none) still
# shows its own real Status badge, falling back correctly instead of
# showing no badge at all -- the real bug caught live building this
# increment (the per-record fallback in resolveCardBadge/
# composable_preview.go).
NDD_CSRF=$(csrf_for "$ALICE" "$BASE_URL/mch_approval_document/new")
NDD_URL=$(post_redirect "$BASE_URL/mch_approval_document" \
    "fld_ad_title=T306+No+Due+Date+$$&fld_ad_document_type=Report&fld_ad_file=x.pdf&fld_ad_submitted_by=$ALICE_ID&fld_ad_approval_mode=Sequential" \
    "$ALICE")
NDD_ID="${NDD_URL##*/}"
NDD_LIST_BODY=$(get_body "$BASE_URL/mch_approval_document" "$ALICE")
NDD_CARD=$(python3 -c "
import sys
body, rid = sys.argv[1], sys.argv[2]
marker = '/mch_approval_document/' + rid + '\"'
start = body.find(marker)
print('' if start == -1 else body[start:body.find('</a>', start)])
" "$NDD_LIST_BODY" "$NDD_ID")
[ -n "$NDD_CARD" ] && echo "$NDD_CARD" | grep -q "Draft"
check T306 "composable-runtime-roadmap.md 17r" "a real Document with no due date still shows its own real Status badge (fallback), not a missing one" $?
