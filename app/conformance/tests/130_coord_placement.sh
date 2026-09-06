#!/usr/bin/env bash
# CAP-V21: coordinate-placement editor (Study 32,
# ../../benchmarks/024-pdf-signature-approval-study.md §5). Sourced by
# run.sh after lib.sh -- reuses ALICE/BOB/CAROL and their pre-resolved
# *_ID, same as 120_pdf_signature.sh. seeds/036_coord_placement_lab.sql
# provides the new View (vw_as_place) on mch_approval_step; Approver role
# already has CanEdit=true machine-wide (migrations/006's own default), so
# the interesting boundary this batch proves is CAP-V21's own per-record
# ownership check (coordPlaceOwnerOK, internal/handler/coordplace.go) --
# NOT CanEdit itself.

# post_redirect_multipart -- see 120_pdf_signature.sh's own definition;
# duplicated here rather than centralized in lib.sh for the same reason
# that file gives (only two batches need it so far, not a third).
post_redirect_multipart() {
    local url="$1" jar="$2" csrf; shift 2
    csrf=$(csrf_for "$jar")
    local -a form_args=(-F "csrf_token=$csrf")
    for kv in "$@"; do form_args+=(-F "$kv"); done
    curl -s -o /dev/null -w '%{redirect_url}' -X POST -b "$jar" "${form_args[@]}" "$url"
}

PDF_FILE=$(mktemp --suffix=.pdf)
cat > "$PDF_FILE" << 'PDFEOF'
%PDF-1.4
1 0 obj<</Type/Catalog/Pages 2 0 R>>endobj
2 0 obj<</Type/Pages/Kids[3 0 R]/Count 1>>endobj
3 0 obj<</Type/Page/Parent 2 0 R/MediaBox[0 0 200 300]>>endobj
trailer<</Size 4/Root 1 0 R>>
%%EOF
PDFEOF

# Bob needs a Signature record for CAP-F22's own T206 to still pass, but
# this batch doesn't depend on it existing -- CAP-V21 previews the
# Document's original file regardless of who has or hasn't signed anything.
AD_URL=$(post_redirect_multipart "$BASE_URL/mch_approval_document" "$ALICE" \
    "fld_ad_title=Place Test $$" "fld_ad_document_type=Policy" \
    "fld_ad_file=@$PDF_FILE;type=application/pdf" \
    "fld_ad_submitted_by=$ALICE_ID" "fld_ad_approval_mode=Sequential")
AD_ID="${AD_URL##*/}"
post_status "$BASE_URL/mch_approval_document/$AD_ID/events/evt_ad_submit" "" "$ALICE" >/dev/null
AS_DATA="fld_as_document=$AD_ID&fld_as_approver=$BOB_ID&fld_as_sequence=1"
AS_URL=$(post_redirect "$BASE_URL/mch_approval_step" "$AS_DATA" "$ALICE")
AS_ID="${AS_URL##*/}"

# T209 -- the assigned Approver (Bob) reaches the page from Detail, sees an
# editable preview with the pin defaulted to center (no placement declared
# yet), and a drop-simulating POST persists.
DETAIL_HAS_LINK=$(get_body "$BASE_URL/mch_approval_step/$AS_ID" "$BOB" | grep -c "Set Position")
GET_BODY=$(get_body "$BASE_URL/mch_approval_step/$AS_ID/place" "$BOB")
HAS_EDIT_MARKERS=$(echo "$GET_BODY" | grep -c 'data-coordplace-machine')
HAS_DEFAULT_CENTER=$(echo "$GET_BODY" | grep -c 'left:50.00%; top:50.00%')
SET_CODE=$(post_status "$BASE_URL/mch_approval_step/$AS_ID/place" "page=1&x=72.5&y=33.25" "$BOB")
PERSISTED=$(get_body "$BASE_URL/mch_approval_step/$AS_ID/place" "$BOB" | grep -c 'left:72.50%; top:33.25%')
[ "$DETAIL_HAS_LINK" -ge 1 ] && [ "$HAS_EDIT_MARKERS" -ge 1 ] && [ "$HAS_DEFAULT_CENTER" -ge 1 ] && \
    [ "$SET_CODE" = "303" ] && [ "$PERSISTED" -ge 1 ]
check T209 "CAP-V21" "the assigned Approver reaches /place from Detail, sees a centered default pin, and a position write persists (got set=$SET_CODE)"  $?

# T210 -- Carol holds "Approver" (same role, CanEdit=true machine-wide,
# migrations/006's own default) but is NOT this Step's own approver.
# CanRead still lets her see the page, but the read-only variant -- no
# draggable pin, no data-coordplace-* attributes at all, same markup
# otherwise (the pin itself still renders at Bob's own just-saved position).
CAROL_BODY=$(get_body "$BASE_URL/mch_approval_step/$AS_ID/place" "$CAROL")
CAROL_HAS_EDIT_MARKERS=$(echo "$CAROL_BODY" | grep -c 'data-coordplace-machine')
CAROL_HAS_PIN=$(echo "$CAROL_BODY" | grep -c 'coordplace-pin')
[ "$CAROL_HAS_EDIT_MARKERS" -eq 0 ] && [ "$CAROL_HAS_PIN" -ge 1 ]
check T210 "CAP-V21" "a same-role Approver who is NOT this Step's own sees the read-only variant (no drag markers), not a 403 or a crash" $?

# T211 -- the same non-owning Approver is denied the WRITE outright, not
# just steered to a read-only view -- the real enforcement this capability
# adds over BoardMove's own machine-level-only CanEdit gate.
CAROL_SET_CODE=$(post_status "$BASE_URL/mch_approval_step/$AS_ID/place" "page=1&x=1&y=1" "$CAROL")
STILL_BOBS_POSITION=$(get_body "$BASE_URL/mch_approval_step/$AS_ID/place" "$BOB" | grep -c 'left:72.50%; top:33.25%')
[ "$CAROL_SET_CODE" = "403" ] && [ "$STILL_BOBS_POSITION" -ge 1 ]
check T211 "CAP-V21" "a same-role Approver who is NOT this Step's own is denied POST /place (got $CAROL_SET_CODE), position unchanged" $?

rm -f "$PDF_FILE"
