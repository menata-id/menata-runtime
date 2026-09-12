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
