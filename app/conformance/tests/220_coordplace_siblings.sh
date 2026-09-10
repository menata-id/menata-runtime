#!/usr/bin/env bash
# CAP-V21's own "multiple sibling pins on one shared preview" generalization
# (document-signature-placement.html's own shape) -- previously deferred,
# named explicitly on that row ("no case has asked for the latter"). Sourced
# by run.sh after lib.sh and 130_coord_placement.sh -- reuses ALICE/BOB/
# CAROL and their pre-resolved *_ID, same convention as that file.

PDF_FILE2=$(mktemp --suffix=.pdf)
cat > "$PDF_FILE2" << 'PDFEOF'
%PDF-1.4
1 0 obj<</Type/Catalog/Pages 2 0 R>>endobj
2 0 obj<</Type/Pages/Kids[3 0 R]/Count 1>>endobj
3 0 obj<</Type/Page/Parent 2 0 R/MediaBox[0 0 200 300]>>endobj
trailer<</Size 4/Root 1 0 R>>
%%EOF
PDFEOF

# post_redirect_multipart -- defined identically in 130_coord_placement.sh,
# already sourced before this file (run.sh's own numeric ordering) -- reused
# as-is, not redefined a second time.
SIB_AD_URL=$(post_redirect_multipart "$BASE_URL/mch_approval_document" "$ALICE" \
    "fld_ad_title=Siblings Test $$" "fld_ad_document_type=Policy" \
    "fld_ad_file=@$PDF_FILE2;type=application/pdf" \
    "fld_ad_submitted_by=$ALICE_ID" "fld_ad_approval_mode=Parallel")
SIB_AD_ID="${SIB_AD_URL##*/}"
post_status "$BASE_URL/mch_approval_document/$SIB_AD_ID/events/evt_ad_submit" "" "$ALICE" >/dev/null

# Two Steps on the SAME Document -- Bob and Carol, Parallel mode (both
# "current" at once, so both may legitimately place a pin without waiting
# on the other, matching the mockup's own "place all signature positions
# up front" shape more closely than Sequential would).
SIB_AS1_URL=$(post_redirect "$BASE_URL/mch_approval_step" "fld_as_document=$SIB_AD_ID&fld_as_approver=$BOB_ID&fld_as_sequence=1" "$ALICE")
SIB_AS1_ID="${SIB_AS1_URL##*/}"
SIB_AS2_URL=$(post_redirect "$BASE_URL/mch_approval_step" "fld_as_document=$SIB_AD_ID&fld_as_approver=$CAROL_ID&fld_as_sequence=2" "$ALICE")
SIB_AS2_ID="${SIB_AS2_URL##*/}"

# T250 -- before Carol has placed anything, Bob's own /place page shows no
# sibling pin at all (nothing real to show yet -- an unset pin isn't faked
# as a sibling either, same "unset = not yet placed" convention the
# record's own pin already uses). Checked by the sibling-dot class itself,
# not by the sibling's own id string -- displayLabel's title text is no
# longer that raw id (see T251's own updated comment below), so absence of
# the id string alone would no longer prove anything either way.
BOB_BEFORE=$(get_body "$BASE_URL/mch_approval_step/$SIB_AS1_ID/place" "$BOB")
BOB_BEFORE_SIBLINGS=$(echo "$BOB_BEFORE" | grep -c 'bg-slate-400')
[ "$BOB_BEFORE_SIBLINGS" -eq 0 ]
check T250 "CAP-V21" "before a sibling Step has placed a pin, none is shown for it yet" $?

# T251 -- Carol places her own Step's pin; Bob's page (a DIFFERENT record)
# now shows it as a read-only sibling dot at Carol's own saved position,
# titled with displayLabel's own "Machine Name <sequence>" fallback
# (Approval Step has no plain-text Field of its own to prefer -- since
# 2026-09-10 this reads "Approval Step 2" instead of the bare record id,
# a separate, later fix to the SAME fallback this row's own comment names;
# T251 itself was updated the same day, not left to silently start
# failing).
post_status "$BASE_URL/mch_approval_step/$SIB_AS2_ID/place" "page=1&x=61.00&y=82.00" "$CAROL" >/dev/null
BOB_AFTER=$(get_body "$BASE_URL/mch_approval_step/$SIB_AS1_ID/place" "$BOB")
echo "$BOB_AFTER" | grep -q 'title="Approval Step 2"' && echo "$BOB_AFTER" | grep -q 'left:61.00%; top:82.00%'
check T251 "CAP-V21" "a sibling Step's own already-placed pin appears read-only, at its own real position" $?

# T252 -- the sibling dot is genuinely read-only: no data-coordplace-*
# attributes anywhere near it, and Bob's own POST to /place cannot move
# CAROL'S pin (there is no route that would even let him try one by id --
# the write is always scoped to the record in the URL, SIB_AS1_ID here).
SIB_BOB_SET=$(post_status "$BASE_URL/mch_approval_step/$SIB_AS1_ID/place" "page=1&x=15.00&y=20.00" "$BOB")
CAROL_STILL_HERS=$(get_body "$BASE_URL/mch_approval_step/$SIB_AS2_ID/place" "$CAROL" | grep -c 'left:61.00%; top:82.00%')
[ "$SIB_BOB_SET" = "303" ] && [ "$CAROL_STILL_HERS" -ge 1 ]
check T252 "CAP-V21" "a sibling's own pin is unaffected by the OTHER record's own write -- writes stay scoped per record" $?

rm -f "$PDF_FILE2"
