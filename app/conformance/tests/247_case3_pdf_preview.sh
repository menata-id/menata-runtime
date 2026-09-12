#!/usr/bin/env bash
# case-03-case-19-completion-checklist.md's own Stage A item 2
# (composable-runtime-roadmap.md 17r): closes Case 3's "no inline PDF
# preview" gap against document-approval.html (ui-sample) -- the file
# field only ever rendered a plain download link before this. Reuses
# isPDFPreview (internal/ui/coordplace.templ), already proven for
# CAP-V21's own coordinate-placement preview -- a real storage key's own
# extension (LocalDisk.Put, internal/storage/storage.go) already tells
# this generically, no new backend capability needed.

CAL_PDF=$(mktemp --suffix=.pdf)
cat > "$CAL_PDF" << 'PDFEOF'
%PDF-1.4
1 0 obj<</Type/Catalog/Pages 2 0 R>>endobj
2 0 obj<</Type/Pages/Kids[3 0 R]/Count 1>>endobj
3 0 obj<</Type/Page/Parent 2 0 R/MediaBox[0 0 200 300]>>endobj
trailer<</Size 4/Root 1 0 R>>
%%EOF
PDFEOF

PDF_URL=$(post_redirect_multipart "$BASE_URL/mch_approval_document" "$ALICE" \
    "fld_ad_title=T307+Inline+PDF+Preview+$$" "fld_ad_document_type=Report" \
    "fld_ad_file=@$CAL_PDF;type=application/pdf" \
    "fld_ad_submitted_by=$ALICE_ID" "fld_ad_approval_mode=Sequential")
PDF_ID="${PDF_URL##*/}"

# T307 -- Detail renders a real inline <object type="application/pdf">
# preview for a real uploaded PDF file field, not just the plain
# download link.
PDF_DETAIL_BODY=$(get_body "$BASE_URL/mch_approval_document/$PDF_ID" "$ALICE")
echo "$PDF_DETAIL_BODY" | grep -qo '<object type="application/pdf" data="/files/[^"]*\.pdf"'
check T307 "composable-runtime-roadmap.md 17r" "Detail renders a real inline PDF preview for a real uploaded file, not just a download link" $?

# T308 -- the preview's own src resolves to the real, fetchable PDF
# bytes -- not just markup that looks right.
PDF_SRC=$(echo "$PDF_DETAIL_BODY" | grep -o '/files/[^"]*\.pdf' | head -1)
PDF_STATUS=$(curl -s -o /dev/null -w '%{http_code}' "$ORIGIN$PDF_SRC")
[ "$PDF_STATUS" = "200" ]
check T308 "composable-runtime-roadmap.md 17r" "the inline preview's own src resolves to the real, fetchable PDF bytes (got HTTP $PDF_STATUS)" $?

rm -f "$CAL_PDF"
