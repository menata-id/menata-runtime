#!/usr/bin/env bash
# CAP-R04 "R28" (composable-runtime-roadmap.md 17p) -- the read side of
# the record_events audit trail, admitted this same session
# (capability-registry.md's CAP-R04 row). One new activity_log View,
# embedded in both modes: record-scoped (Document/Card Detail) and
# cross-record (the composed page's own "Recent Activity", closing the
# placeholder 17k left there). Reuses ALICE/ALICE_ID and
# WIREFRAME_CARD_ID (already resolved into shared shell scope by earlier
# files, sourced first by run.sh).

CAL_PDF=$(mktemp --suffix=.pdf)
cat > "$CAL_PDF" << 'PDFEOF'
%PDF-1.4
1 0 obj<</Type/Catalog/Pages 2 0 R>>endobj
2 0 obj<</Type/Pages/Kids[3 0 R]/Count 1>>endobj
3 0 obj<</Type/Page/Parent 2 0 R/MediaBox[0 0 200 300]>>endobj
trailer<</Size 4/Root 1 0 R>>
%%EOF
PDFEOF

CAL_URL=$(post_redirect_multipart "$BASE_URL/mch_approval_document" "$ALICE" \
    "fld_ad_title=T287+Activity+Log+$$" "fld_ad_document_type=Policy" \
    "fld_ad_file=@$CAL_PDF;type=application/pdf" \
    "fld_ad_submitted_by=$ALICE_ID" "fld_ad_approval_mode=Sequential")
CAL_ID="${CAL_URL##*/}"
post_status "$BASE_URL/mch_approval_document/$CAL_ID/events/evt_ad_submit" "" "$ALICE" >/dev/null

# T287 -- the real Document Detail page now has a real Activity section
# (record-scoped mode, embedded via CAP-V20's existing Children
# mechanism) showing the real Event that fired ("Submit", the Event's
# own declared Name) by the real actor.
CAL_DETAIL_BODY=$(get_body "$BASE_URL/mch_approval_document/$CAL_ID" "$ALICE")
echo "$CAL_DETAIL_BODY" | grep -q "Submit"
check T287 "composable-runtime-roadmap.md 17p" "the real Document Detail page shows a real Activity section naming the real Submit event" $?

# T288 -- the real composed page's "Recent Activity" section no longer
# shows 17k's own honest placeholder text -- it now renders the real
# thing (cross-record mode, embedded via CAP-V10 Tier 2's existing
# Children mechanism).
CAL_PAGE_BODY=$(get_body "$BASE_URL/mch_approval_document/page" "$ALICE")
! echo "$CAL_PAGE_BODY" | grep -q "Recent Activity requires a record-history"
check T288 "composable-runtime-roadmap.md 17p" "the composed page's Recent Activity section no longer shows the 17k placeholder" $?

echo "$CAL_PAGE_BODY" | grep -q "Submit"
check T289 "composable-runtime-roadmap.md 17p" "the composed page's Recent Activity section shows real cross-record activity (the real Submit event)" $?

# T290 -- Case 19: mch_pm_card has zero declared Events today (a real,
# named fact, not a bug -- see 17p's own roadmap section) -- its own
# Activity section renders the honest empty state, not a broken page.
CAL_CARD_BODY=$(get_body "$BASE_URL/mch_pm_card/$WIREFRAME_CARD_ID" "$PM_MEMBER")
echo "$CAL_CARD_BODY" | grep -q "No activity yet"
check T290 "composable-runtime-roadmap.md 17p" "Card Detail's own Activity section renders the honest empty state (mch_pm_card has no declared Events yet)" $?

rm -f "$CAL_PDF"
