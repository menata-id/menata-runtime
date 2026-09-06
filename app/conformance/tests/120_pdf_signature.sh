#!/usr/bin/env bash
# CAP-F22: binary PDF signature compositing (Study 32,
# ../../benchmarks/024-pdf-signature-approval-study.md), extending Case 3
# (Document Approval) rather than replacing it. Sourced by run.sh after
# lib.sh -- reuses ALICE/BOB/CAROL and their pre-resolved *_ID (lib.sh's own
# comment: "sourced before every tests/*.sh file", seeds/007_authentication.
# sql). seeds/035_pdf_signature_lab.sql provides the new Signature Machine
# and Approval Step/Document fields; no Signature record exists for anyone
# until this test creates one.
#
# Case 3's OLDER tests (010_case1_3_core.sh) submit fld_ad_file as a bare
# string ("policy.pdf") -- fine for testing the approval workflow itself,
# but CAP-F22 needs a REAL stored file to open, so this batch does a genuine
# multipart upload for both the source PDF and the signature image, same
# discipline as CAP-F06's own upload test (040_batches10_11_subnav.sh).

# post_redirect_multipart <url> <jar> <field=value|@file;type=... ...> ->
# echoes redirect url. Local to this file (only this batch needs multipart +
# redirect together) -- lib.sh's own header comment says centralize a helper
# only once a LATER batch needs it too.
post_redirect_multipart() {
    local url="$1" jar="$2" csrf; shift 2
    csrf=$(csrf_for "$jar")
    local -a form_args=(-F "csrf_token=$csrf")
    for kv in "$@"; do form_args+=(-F "$kv"); done
    curl -s -o /dev/null -w '%{redirect_url}' -X POST -b "$jar" "${form_args[@]}" "$url"
}

# Minimal hand-written single-page PDF (no xref table -- pdfcpu's own repair
# path reconstructs one; verified locally against this exact byte content
# before wiring it in here). 200x300pt MediaBox.
PDF_FILE=$(mktemp --suffix=.pdf)
cat > "$PDF_FILE" << 'PDFEOF'
%PDF-1.4
1 0 obj<</Type/Catalog/Pages 2 0 R>>endobj
2 0 obj<</Type/Pages/Kids[3 0 R]/Count 1>>endobj
3 0 obj<</Type/Page/Parent 2 0 R/MediaBox[0 0 200 300]>>endobj
trailer<</Size 4/Root 1 0 R>>
%%EOF
PDFEOF

# Tiny 1x1 red PNG, same embedding style as 040_batches10_11_subnav.sh's own
# PNG_B64 fixture.
SIG_B64="iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg=="
SIG_FILE=$(mktemp --suffix=.png)
echo "$SIG_B64" | base64 -d > "$SIG_FILE" 2>/dev/null || echo "$SIG_B64" | base64 --decode > "$SIG_FILE"

# T206 -- Bob registers his own Signature (Study 32 §4.3: an ordinary,
# workspace-authored per-user record, CAP-P02 owner_field-scoped to the
# acting identity).
SIG_URL=$(post_redirect_multipart "$BASE_URL/mch_signature" "$BOB" \
    "fld_sig_owner=$BOB_ID" "fld_sig_image=@$SIG_FILE;type=image/png")
[ -n "$SIG_URL" ]
check T206 "CAP-F22" "an approver registers their own Signature record (image upload)" $?

# T207 -- positive path: Alice submits a real PDF, assigns Bob (who now has
# a Signature) to a single Sequential step with a declared placement, Bob
# approves, and the Document's Signed File is a genuinely new, different PDF
# -- not a copy of the original and not the placeholder string the older
# Case 3 tests use.
ORIG_SIZE=$(wc -c < "$PDF_FILE" | tr -d ' ')
AD_URL=$(post_redirect_multipart "$BASE_URL/mch_approval_document" "$ALICE" \
    "fld_ad_title=Sig Test $$" "fld_ad_document_type=Policy" \
    "fld_ad_file=@$PDF_FILE;type=application/pdf" \
    "fld_ad_submitted_by=$ALICE_ID" "fld_ad_approval_mode=Sequential")
AD_ID="${AD_URL##*/}"
post_status "$BASE_URL/mch_approval_document/$AD_ID/events/evt_ad_submit" "" "$ALICE" >/dev/null
AS_DATA="fld_as_document=$AD_ID&fld_as_approver=$BOB_ID&fld_as_sequence=1&fld_as_signature_page=1&fld_as_signature_x=50&fld_as_signature_y=50"
AS_URL=$(post_redirect "$BASE_URL/mch_approval_step" "$AS_DATA" "$ALICE")
AS_ID="${AS_URL##*/}"
APPROVE_CODE=$(post_status "$BASE_URL/mch_approval_step/$AS_ID/events/evt_as_approve" "" "$BOB")

SIGNED_HREF=$(get_body "$BASE_URL/mch_approval_document/$AD_ID" "$ALICE" \
    | grep -oE 'href="/files/[^"]*"' | tail -1 | sed -E 's/href="(.*)"/\1/')
SIGNED_FILE=$(mktemp)
[ -n "$SIGNED_HREF" ] && curl -s -o "$SIGNED_FILE" "$ORIGIN$SIGNED_HREF"
SIGNED_SIZE=$(wc -c < "$SIGNED_FILE" 2>/dev/null | tr -d ' ')

[ "$APPROVE_CODE" = "303" ] && [ -n "$SIGNED_HREF" ] && \
    [ "$(head -c 4 "$SIGNED_FILE" 2>/dev/null)" = "%PDF" ] && \
    [ "${SIGNED_SIZE:-0}" -ne "$ORIG_SIZE" ]
check T207 "CAP-F22" "approving a Step with a registered Signature composites a real new PDF onto the Document (orig=${ORIG_SIZE}b, signed=${SIGNED_SIZE:-0}b)" $?

# T208 -- negative: Carol is a valid Approver (CAP-P02 ownership check
# passes) but has registered no Signature record. The action returns a
# plain error (not a ruleViolation -- see internal/executor/executor.go's
# own doc comment on doCompositeSignature for why), so this surfaces as 500
# like every other cross-record action's own failure path in this codebase
# (doCreateRecord/doCrossSetField/doBatchGenerate), not a special-cased 400.
AD2_URL=$(post_redirect_multipart "$BASE_URL/mch_approval_document" "$ALICE" \
    "fld_ad_title=Sig Test No Sig $$" "fld_ad_document_type=Policy" \
    "fld_ad_file=@$PDF_FILE;type=application/pdf" \
    "fld_ad_submitted_by=$ALICE_ID" "fld_ad_approval_mode=Sequential")
AD2_ID="${AD2_URL##*/}"
post_status "$BASE_URL/mch_approval_document/$AD2_ID/events/evt_ad_submit" "" "$ALICE" >/dev/null
AS2_DATA="fld_as_document=$AD2_ID&fld_as_approver=$CAROL_ID&fld_as_sequence=1&fld_as_signature_page=1&fld_as_signature_x=10&fld_as_signature_y=10"
AS2_URL=$(post_redirect "$BASE_URL/mch_approval_step" "$AS2_DATA" "$ALICE")
AS2_ID="${AS2_URL##*/}"
NOSIG_CODE=$(post_status "$BASE_URL/mch_approval_step/$AS2_ID/events/evt_as_approve" "" "$CAROL")
NOSIG_DECISION=$(get_body "$BASE_URL/mch_approval_step/$AS2_ID" "$ALICE" | grep -oE '>Pending<' | head -1)
[ "$NOSIG_CODE" = "500" ] && [ -n "$NOSIG_DECISION" ]
check T208 "CAP-F22" "an Approver with no Signature record fails cleanly (got $NOSIG_CODE), and the Step's own decision is left Pending, not half-applied" $?

rm -f "$PDF_FILE" "$SIG_FILE" "$SIGNED_FILE"
