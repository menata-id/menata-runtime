#!/usr/bin/env bash
# composable-runtime-roadmap.md 17m (Detail-Page Composition Pilot) -- the
# first real HTTP route driven by internal/composable's own Detail-shaped
# primitive (BuildDatasetFromView's new ViewTypeDetail case ->
# LowerCollection -> ResolveCollectionItem), additive to
# mch_approval_document's own vw_ad_detail, same machine T262-T269
# (241_composable_pilot.sh) already exercise for the List-shaped pilot.
# Reuses ALICE/ALICE_ID/FRANK (already in scope by this point, same as
# 241's own reuse).

CDP_DATA="fld_ad_title=T281+Detail+Pilot+$$&fld_ad_document_type=Contract&fld_ad_file=t281.pdf&fld_ad_submitted_by=$ALICE_ID&fld_ad_approval_mode=Sequential"
CDP_URL=$(post_redirect "$BASE_URL/mch_approval_document" "$CDP_DATA" "$ALICE")
CDP_ID="${CDP_URL##*/}"

# extract_dd_after_dt <body> <label> -- the label's own <dt>...</dt>,
# immediately followed by its <dd>...</dd> content's own text, skipping
# any wrapping tags in between (the real Detail page's own detail.templ
# always wraps a non-empty, non-link, non-SLA value in @StatusBadge's own
# <span>, even for a plain text field like Title -- caught live: the
# first version of this extractor assumed plain text directly inside
# <dd>, which matched this preview's own markup but not the real page's).
extract_dd_after_dt() {
    printf '%s' "$1" | tr -d '\n' | grep -oE "<dt[^>]*>$2</dt><dd[^>]*>(<[^>]*>)*[^<]*" | sed -E 's/.*>([^<]*)$/\1/'
}

CDP_REAL_BODY=$(get_body "$BASE_URL/mch_approval_document/$CDP_ID" "$ALICE")
CDP_PREVIEW_BODY=$(get_body "$BASE_URL/mch_approval_document/$CDP_ID/composable-preview" "$ALICE")

# T281 -- ground truth: the REAL Detail page shows the real title/
# document_type/approval_mode/status for a record just created via the
# ordinary Create flow, not a stub.
CDP_REAL_TITLE=$(extract_dd_after_dt "$CDP_REAL_BODY" "Title")
[ "$CDP_REAL_TITLE" = "T281 Detail Pilot $$" ]
check T281 "composable-runtime-roadmap.md 17m" "the real Detail page shows title=\"T281 Detail Pilot $$\" (ground truth, got \"$CDP_REAL_TITLE\")" $?

# T282 -- composable-preview reproduces the same real Title/Document
# Type/Approval Mode/Status -- real render-output equivalence for every
# field ResolveFieldValue can actually resolve byte-identically (plain
# text/value_list; reference/user/file fields are a named, NOT asserted
# boundary here -- ResolveFieldValue's own doc comment names raw stored
# ids only, no label dereferencing, so fld_ad_submitted_by/fld_ad_file
# would NOT match and are deliberately not compared).
CDP_OK=true
for CDP_LABEL_PAIR in "Title:T281 Detail Pilot $$" "Document Type:Contract" "Approval Mode:Sequential" "Status:Draft"; do
    CDP_LABEL="${CDP_LABEL_PAIR%%:*}"
    CDP_WANT="${CDP_LABEL_PAIR#*:}"
    CDP_GOT=$(extract_dd_after_dt "$CDP_PREVIEW_BODY" "$CDP_LABEL")
    [ "$CDP_GOT" = "$CDP_WANT" ] || CDP_OK=false
done
[ "$CDP_OK" = true ]
check T282 "composable-runtime-roadmap.md 17m" "composable-preview reproduces the same real Title/Document Type/Approval Mode/Status -- real equivalence, not a passing resemblance" $?

# T283 -- a role with no permission on this Machine's app is denied,
# same guard as the List-shaped pilot's own T263.
CDP_DENIED_CODE=$(curl -s -o /dev/null -w '%{http_code}' -b "$FRANK" "$BASE_URL/mch_approval_document/$CDP_ID/composable-preview")
[ "$CDP_DENIED_CODE" = "403" ]
check T283 "composable-runtime-roadmap.md 17m" "a role with no permission on this Machine's app is denied (got $CDP_DENIED_CODE)" $?

# T284 -- a Machine from another Workspace 404s on this route too
# (CAP-X06 pattern, same as every other per-machine route).
CDP_CROSS_CODE=$(curl -s -o /dev/null -w '%{http_code}' -b "$ALICE" "$BASE_URL_ACME/mch_approval_document/$CDP_ID/composable-preview")
[ "$CDP_CROSS_CODE" = "404" ]
check T284 "composable-runtime-roadmap.md 17m" "a Machine from another Workspace 404s on the composable detail preview route too (got $CDP_CROSS_CODE)" $?

# T285 -- an unknown record 404s, not a 500 (CAP-X05 pattern).
CDP_UNKNOWN_CODE=$(curl -s -o /dev/null -w '%{http_code}' -b "$ALICE" "$BASE_URL/mch_approval_document/mch_ghost_record/composable-preview")
[ "$CDP_UNKNOWN_CODE" = "404" ]
check T285 "composable-runtime-roadmap.md 17m" "an unknown record 404s on the composable detail preview route (got $CDP_UNKNOWN_CODE)" $?
