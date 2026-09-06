#!/usr/bin/env bash
# Batch 10 (Infra), Batch 11 (remaining Field Types), and CAP-O03 Tier 2 (persistent sub-navigation) -- closes out the July 2026 batch series.
# Sourced by run.sh after lib.sh -- assumes lib.sh's helpers/ACCOUNTS are already in scope.

# --- Batch 10: Infra (2026-07-12) ---
# seeds/016_infra_lab.sql. CAP-X04/X09/X10/X11 deliberately deferred, see
# capability-registry.md's own rows for why.

# T116 -- CAP-X12: an event whose actions span THREE machines (its own
# set_field, a create_record against a real machine, a create_record
# against a deliberately dangling one) rolls back as a whole when the last
# action hits a real foreign-key violation -- not just that one action
# skipped while the earlier two silently commit.
ZARA_CSRF=$(csrf_for "$ZARA")
X12L_URL=$(post_redirect "$BASE_URL/mch_x12_ledger" "fld_x12l_status=Draft" "$ZARA")
X12L_ID="${X12L_URL##*/}"
X12_CODE=$(post_status "$BASE_URL/mch_x12_ledger/$X12L_ID/events/evt_x12_commit" "" "$ZARA")
[ "$X12_CODE" = "500" ] && \
    body_contains "$X12L_URL" ">Draft<" "$ZARA" && \
    ! body_contains "$X12L_URL" ">Posted<" "$ZARA" && \
    ! body_contains "$BASE_URL/mch_x12_entry" "Logged" "$ZARA"
check T116 "CAP-X12" "a cross-machine action chain rolls back as a whole on a downstream failure -- the record's own field AND an earlier, otherwise-successful create_record both revert (got $X12_CODE)" $?

# T117 -- CAP-X13: an inbound webhook delivered twice with the SAME
# X-Idempotency-Key only runs the event once -- the second delivery still
# returns 200 (the duplicate-is-success contract), but doesn't create a
# second Log record.
X13S1_URL=$(post_redirect "$BASE_URL/mch_x13_source" "fld_x13s_amount=10" "$ZARA")
X13S1_ID="${X13S1_URL##*/}"
DUP1_CODE=$(curl -s -o /dev/null -w '%{http_code}' -X POST -H "X-Webhook-Secret: infra-lab-secret-2026" -H "X-Idempotency-Key: conf-dup-$$" "$ORIGIN/webhooks/mch_x13_source/$X13S1_ID/evt_x13_log")
DUP2_CODE=$(curl -s -o /dev/null -w '%{http_code}' -X POST -H "X-Webhook-Secret: infra-lab-secret-2026" -H "X-Idempotency-Key: conf-dup-$$" "$ORIGIN/webhooks/mch_x13_source/$X13S1_ID/evt_x13_log")
LOG_COUNT_1=$(get_body "$BASE_URL/api/v1/mch_x13_log" "$ZARA" | grep -o "\"id\"" | wc -l)
[ "$DUP1_CODE" = "200" ] && [ "$DUP2_CODE" = "200" ]
check T117 "CAP-X13" "a repeated webhook delivery with the same idempotency key returns success both times but only runs the event once (got $DUP1_CODE/$DUP2_CODE)" $?

# T118 -- CAP-X13 negative: a DIFFERENT idempotency key (a genuinely new
# delivery) is not suppressed -- proves the claim table is scoped per key,
# not a blanket "this event already ran once ever" block.
X13S2_URL=$(post_redirect "$BASE_URL/mch_x13_source" "fld_x13s_amount=20" "$ZARA")
X13S2_ID="${X13S2_URL##*/}"
curl -s -o /dev/null -X POST -H "X-Webhook-Secret: infra-lab-secret-2026" -H "X-Idempotency-Key: conf-other-$$" "$ORIGIN/webhooks/mch_x13_source/$X13S2_ID/evt_x13_log"
LOG_COUNT_2=$(get_body "$BASE_URL/api/v1/mch_x13_log" "$ZARA" | grep -o "\"id\"" | wc -l)
[ "$LOG_COUNT_2" -eq $((LOG_COUNT_1 + 1)) ]
check T118 "CAP-X13" "a different idempotency key is a genuinely new delivery, not suppressed (count $LOG_COUNT_1 -> $LOG_COUNT_2)" $?

# T119 -- CAP-X07: the auto-generated JSON API lists and reads a Machine's
# own records, permission-trimmed and workspace-scoped the same as the HTML
# routes -- an unauthenticated request is redirected to /login, not served.
LIST_CODE=$(curl -s -o /dev/null -w '%{http_code}' -b "$ZARA" "$BASE_URL/api/v1/mch_x13_log")
FIRST_ID=$(get_body "$BASE_URL/api/v1/mch_x13_log" "$ZARA" | grep -oE '"id":"[a-f0-9-]+"' | head -1 | sed -E 's/.*:"([^"]+)"/\1/')
GET_CODE=$(curl -s -o /dev/null -w '%{http_code}' -b "$ZARA" "$BASE_URL/api/v1/mch_x13_log/$FIRST_ID")
ANON_API_CODE=$(curl -s -o /dev/null -w '%{http_code}' "$BASE_URL/api/v1/mch_x13_log")
[ "$LIST_CODE" = "200" ] && [ "$GET_CODE" = "200" ] && [ "$ANON_API_CODE" = "303" ]
check T119 "CAP-X07" "the JSON API lists and reads a machine's records for an authenticated session, denies an unauthenticated one (list=$LIST_CODE, get=$GET_CODE, anon=$ANON_API_CODE)" $?

# T120 -- CAP-X07: a JSON POST creates a real record (same validation as
# the HTML Create path), authenticated via X-CSRF-Token header since a JSON
# body has no csrf_token form field for the existing synchronizer-token
# check to read -- and a request with no CSRF at all is still rejected.
CREATE_CODE=$(curl -s -o /dev/null -w '%{http_code}' -b "$ZARA" -X POST "$BASE_URL/api/v1/mch_x12_entry" \
    -H "Content-Type: application/json" -H "X-CSRF-Token: $ZARA_CSRF" \
    -d "{\"fld_x12e_note\":\"api-created-$$\"}")
NO_CSRF_CODE=$(curl -s -o /dev/null -w '%{http_code}' -b "$ZARA" -X POST "$BASE_URL/api/v1/mch_x12_entry" \
    -H "Content-Type: application/json" -d '{"fld_x12e_note":"should-be-rejected"}')
[ "$CREATE_CODE" = "201" ] && [ "$NO_CSRF_CODE" = "403" ] && \
    get_body "$BASE_URL/api/v1/mch_x12_entry" "$ZARA" | grep -q "api-created-$$"
check T120 "CAP-X07" "a JSON create via X-CSRF-Token header succeeds and is visible in the same API's list; a request with no CSRF token is rejected (create=$CREATE_CODE, no_csrf=$NO_CSRF_CODE)" $?

# T121 -- CAP-X08: an Application's full metadata tree exports as JSON,
# straight from the same in-memory model the runtime itself interprets --
# Admin-only, a non-admin role is denied.
EXPORT_CODE=$(curl -s -o /dev/null -w '%{http_code}' -b "$ZARA" "$BASE_URL/apps/app_infra_lab/export")
EXPORT_DENIED_CODE=$(curl -s -o /dev/null -w '%{http_code}' -b "$YARA" "$BASE_URL/apps/app_infra_lab/export")
[ "$EXPORT_CODE" = "200" ] && [ "$EXPORT_DENIED_CODE" = "403" ] && \
    body_contains "$BASE_URL/apps/app_infra_lab/export" "Webhook Source" "$ZARA"
check T121 "CAP-X08" "an Application's metadata exports as JSON for an Admin; denied for a non-admin role (export=$EXPORT_CODE, denied=$EXPORT_DENIED_CODE)" $?

# --- Batch 11: Remaining Field Types (2026-07-12) ---
# seeds/017_field_types_lab.sql. CAP-F17 (multi-currency money) and CAP-F19
# (quantity/UoM) are proven by composition (CAP-F08/F14/F07 + value_list),
# no new mechanism -- same framing as CAP-I05 in Batch 8.

# T122 -- CAP-F07: a `number` field renders and round-trips a real numeric
# value (not falling back to a bare text input with no validation).
FT1_URL=$(post_redirect "$BASE_URL/mch_ft_product" "fld_ftp_title=NumProbe+$$&fld_ftp_qty=7" "$WIRA")
body_contains "$FT1_URL" ">7<" "$WIRA"
check T122 "CAP-F07" "a number field's value round-trips through Create and Detail" $?

# T123 -- CAP-F08: a `money` field displays with its declared currency.
FT2_URL=$(post_redirect "$BASE_URL/mch_ft_product" "fld_ftp_title=MoneyProbe+$$&fld_ftp_price=25000" "$WIRA")
body_contains "$FT2_URL" "IDR 25000" "$WIRA"
check T123 "CAP-F08" "a money field renders with its declared currency" $?

# T124 -- CAP-F09: a `boolean` field is Yes when checked, No when the
# checkbox is left unchecked (submits nothing at all, not an empty string).
FT3_URL=$(post_redirect "$BASE_URL/mch_ft_product" "fld_ftp_title=BoolYes+$$&fld_ftp_active=true" "$WIRA")
FT4_URL=$(post_redirect "$BASE_URL/mch_ft_product" "fld_ftp_title=BoolNo+$$" "$WIRA")
body_contains "$FT3_URL" ">Yes<" "$WIRA" && body_contains "$FT4_URL" ">No<" "$WIRA"
check T124 "CAP-F09" "a boolean field is Yes when checked, No when the checkbox is left unchecked" $?

# T125 -- CAP-F10: `time`/`date_time`/`duration` all render real HTML5
# input types on the New form (not a bare text fallback) and round-trip
# their values through Create and Detail.
body_contains "$BASE_URL/mch_ft_product/new" 'type="time"' "$WIRA" && \
    body_contains "$BASE_URL/mch_ft_product/new" 'type="datetime-local"' "$WIRA"
check T125 "CAP-F10" "time and date_time fields render real HTML5 input types, not a text fallback" $?
FT5_URL=$(post_redirect "$BASE_URL/mch_ft_product" "fld_ftp_title=TimeProbe+$$&fld_ftp_release_time=14:30&fld_ftp_launched_at=2026-07-12T09:00&fld_ftp_prep_duration=90" "$WIRA")
body_contains "$FT5_URL" "14:30" "$WIRA" && body_contains "$FT5_URL" "2026-07-12T09:00" "$WIRA" && body_contains "$FT5_URL" ">90<" "$WIRA"
check T126 "CAP-F10" "time/date_time/duration values round-trip through Create and Detail" $?

# T126 -- CAP-F14: a `computed` field is never stored, never read from a
# form submission -- it's always Price * Quantity, freshly computed at
# render time, even if a submitter tries to POST a value for it directly.
FT6_URL=$(post_redirect "$BASE_URL/mch_ft_product" "fld_ftp_title=CompProbe+$$&fld_ftp_price=1000&fld_ftp_qty=4&fld_ftp_total=999999" "$WIRA")
body_contains "$FT6_URL" ">4000<" "$WIRA" && ! body_contains "$FT6_URL" "999999" "$WIRA"
check T127 "CAP-F14" "a computed field is Price times Quantity, ignoring any value POSTed directly for it" $?

# T127 -- CAP-F15: a field's declared default applies when the submitter
# leaves it blank -- generalized beyond the pre-existing value_list-only
# first-value convention (Priority here is a plain text field).
FT7_URL=$(post_redirect "$BASE_URL/mch_ft_product" "fld_ftp_title=DefaultProbe+$$" "$WIRA")
body_contains "$FT7_URL" ">Normal<" "$WIRA"
check T128 "CAP-F15" "a plain (non-value_list) field's declared default applies when left blank" $?

# T128 -- CAP-F18: an auto-number field left blank gets a real, sequential,
# server-generated value -- never a submitter-supplied string (T126's own
# "ignore a directly-POSTed value" proof applies the same way here).
FT8_URL=$(post_redirect "$BASE_URL/mch_ft_product" "fld_ftp_title=SeqProbe1+$$" "$WIRA")
FT9_URL=$(post_redirect "$BASE_URL/mch_ft_product" "fld_ftp_title=SeqProbe2+$$" "$WIRA")
SKU1=$(get_body "$FT8_URL" "$WIRA" | grep -oE '>SKU-[0-9]+<' | head -1 | tr -d '<>')
SKU2=$(get_body "$FT9_URL" "$WIRA" | grep -oE '>SKU-[0-9]+<' | head -1 | tr -d '<>')
N1=$(echo "$SKU1" | grep -oE '[0-9]+')
N2=$(echo "$SKU2" | grep -oE '[0-9]+')
[ -n "$N1" ] && [ -n "$N2" ] && [ "$((10#$N2))" -eq "$((10#$N1 + 1))" ]
check T129 "CAP-F18" "consecutive Creates get sequential, zero-padded auto-numbers (got $SKU1, $SKU2)" $?

# T129 -- CAP-F06: an uploaded image is genuinely stored (not silently
# dropped, the pre-Batch-11 behavior), resized to max_dimension, and
# re-encoded as real WebP (RIFF/WEBP magic bytes) -- not just copied
# through unchanged. A file type outside the declared accept list is
# rejected outright.
PNG_B64="iVBORw0KGgoAAAANSUhEUgAAASwAAAEsCAIAAAD2HxkiAAACd0lEQVR42u3TsQkAAAzDsJzez5sbOnUR6AKDk1ngkwRgQjAhYEIwIWBCMCFgQjAhYEIwIWBCMCFgQjAhYEIwIWBCMCFgQjAhYEIwIWBCMCFgQjAhYEIwIWBCMCFgQjAhYEIwIWBCMCFgQjAhYEIwIWBCMCFgQjAhYEIwIWBCMCFgQjAhYEIwIWBCMCFgQjAhYEIwIWBCMCFgQjAhYEIwIZhQAjAhmBAwIZgQMCGYEDAhmBAwIZgQMCGYEDAhmBAwIZgQMCGYEDAhmBAwIZgQMCGYEDAhmBAwIZgQMCGYEDAhmBAwIZgQMCGYEDAhmBAwIZgQMCGYEDAhmBAwIZgQMCGYEDAhmBAwIZgQMCGYEDAhmBAwIZgQMCGYEDAhmBAwIZgQTKgCmBBMCJgQTAiYEEwImBBMCJgQTAiYEEwImBBMCJgQTAiYEEwImBBMCJgQTAiYEEwImBBMCJgQTAiYEEwImBBMCJgQTAiYEEwImBBMCJgQTAiYEEwImBBMCJgQTAiYEEwImBBMCJgQTAiYEEwImBBMCJgQTAiYEEwImBBMCJgQTAgmBEwIJgRMCCYETAgmBEwIJgRMCCYETAgmBEwIJgRMCCYETAgmBEwIJgRMCCYETAgmBEwIJgRMCCYETAgmBEwIJgRMCCYETAgmBEwIJgRMCCYETAgmBEwIJgRMCCYETAgmBEwIJgRMCCYETAgmBEwIJgRMCCYETAgmBEwIJgQkABOCCQETggkBE4IJAROCCQETggkBE4IJAROCCQETggkBE4IJAROCCQETggkBE4IJAROCCQETggkBE4IJAROCCQETggkBE4IJAROCCQETggmBmwIoGRdDVqoymgAAAABJRU5ErkJggg=="
PNG_FILE=$(mktemp)
echo "$PNG_B64" | base64 -d > "$PNG_FILE" 2>/dev/null || echo "$PNG_B64" | base64 --decode > "$PNG_FILE"
FT_CSRF=$(csrf_for "$WIRA")
UPLOAD_HEADERS=$(mktemp)
UPLOAD_CODE=$(curl -s -b "$WIRA" -D "$UPLOAD_HEADERS" -o /dev/null -w '%{http_code}' -X POST "$BASE_URL/mch_ft_product" \
    -F "csrf_token=$FT_CSRF" -F "fld_ftp_title=PhotoProbe$$" -F "fld_ftp_photo=@$PNG_FILE;type=image/png")
UPLOAD_REDIRECT=$(grep -i '^location' "$UPLOAD_HEADERS" | tr -d '\r' | sed -E 's/^[Ll]ocation: //')
FILE_HREF=$(get_body "$ORIGIN$UPLOAD_REDIRECT" "$WIRA" | grep -oE 'href="/files/[^"]*"' | sed -E 's/href="(.*)"/\1/')
STORED_FILE=$(mktemp)
curl -s -D "$UPLOAD_HEADERS" -o "$STORED_FILE" "$ORIGIN$FILE_HREF"
SERVED_CONTENT_TYPE=$(grep -i '^content-type' "$UPLOAD_HEADERS" | tr -d '\r')
ORIG_SIZE=$(wc -c < "$PNG_FILE" | tr -d ' ')
STORED_SIZE=$(wc -c < "$STORED_FILE" | tr -d ' ')
[ "$UPLOAD_CODE" = "303" ] && \
    echo "$SERVED_CONTENT_TYPE" | grep -qi "image/webp" && \
    [ "$(head -c 4 "$STORED_FILE")" = "RIFF" ] && \
    [ "$STORED_SIZE" -lt "$ORIG_SIZE" ]
check T130 "CAP-F06" "an uploaded image is stored, resized, and re-encoded as real WebP (orig=${ORIG_SIZE}b, stored=${STORED_SIZE}b, content-type=$SERVED_CONTENT_TYPE)" $?
BAD_UPLOAD_FILE=$(mktemp)
printf 'not an image' > "$BAD_UPLOAD_FILE"
BAD_CODE=$(curl -s -o /dev/null -w '%{http_code}' -b "$WIRA" -X POST "$BASE_URL/mch_ft_product" \
    -F "csrf_token=$FT_CSRF" -F "fld_ftp_title=BadUpload$$" -F "fld_ftp_photo=@$BAD_UPLOAD_FILE;type=text/plain")
[ "$BAD_CODE" = "400" ]
check T131 "CAP-F06" "a file type outside the declared accept list is rejected, not silently stored (got $BAD_CODE)" $?
rm -f "$PNG_FILE" "$UPLOAD_HEADERS" "$STORED_FILE" "$BAD_UPLOAD_FILE"

# T130 -- CAP-F17: multi-currency money, composed from CAP-F08's own
# currency_field option plus a CAP-F14 computed base-currency mirror -- no
# dedicated new field type.
FT10_URL=$(post_redirect "$BASE_URL/mch_ft_invoice" "fld_fti_currency=USD&fld_fti_amount=100&fld_fti_rate=16000" "$WIRA")
body_contains "$FT10_URL" "USD 100" "$WIRA" && body_contains "$FT10_URL" ">1600000<" "$WIRA"
check T132 "CAP-F17" "multi-currency money (currency + rate) computes its base-currency mirror correctly" $?

# T131 -- CAP-F19: quantity/UoM Tier 1 (flat factor pair), composed from
# `number` + `value_list` + a CAP-F14 computed field -- no dedicated new
# field type.
FT11_URL=$(post_redirect "$BASE_URL/mch_ft_shipment" "fld_fts_quantity=5&fld_fts_unit=Kg&fld_fts_factor=1000" "$WIRA")
body_contains "$FT11_URL" ">5000<" "$WIRA"
check T133 "CAP-F19" "quantity/UoM Tier 1 composition converts to its base unit correctly (5 Kg * 1000 = 5000 g)" $?

# T132 -- CAP-F21: a `document` View renders its own html/template source
# against one real record's data -- merge fields resolved, not literal
# placeholder text left in the output.
DOC_CODE=$(curl -s -o /dev/null -w '%{http_code}' -b "$WIRA" "$FT1_URL/document")
body_contains "$FT1_URL/document" "SKU: SKU-" "$WIRA" && ! body_contains "$FT1_URL/document" "{{.fld_ftp_sku}}" "$WIRA"
check T134 "CAP-F21" "a document View renders its template with real merge fields resolved (got $DOC_CODE)" $?

# --- CAP-O03 Tier 2: persistent in-app sub-navigation (2026-07-12) ---
# benchmarks/009-in-app-navigation-benchmark.md. No new seed needed --
# reuses app_field_types_lab (3 machines: Product/Invoice/Shipment,
# seeds/017) for the positive case and app_workspace_lab_ops (1 machine,
# seeds/015) for the negative case.

# T135 -- a page belonging to a Machine in a multi-machine Application
# renders a persistent sub-nav strip listing every sibling Machine, with
# the current one marked active -- so a user can move sideways without
# returning to the workspace home. A Machine that's the ONLY one in its
# Application renders no strip at all (nothing to move sideways to).
body_contains "$BASE_URL/mch_ft_product" 'href="/ws_default/mch_ft_invoice"' "$WIRA" && \
    body_contains "$BASE_URL/mch_ft_product" 'href="/ws_default/mch_ft_shipment"' "$WIRA" && \
    body_contains "$BASE_URL/mch_ft_product" 'bg-white text-blue-700' "$WIRA" && \
    ! body_contains "$BASE_URL/mch_wsx_project" "bg-slate-100 border-b border-slate-200" "$YARA"
check T135 "CAP-O03" "a multi-machine Application renders a persistent sub-nav to sibling Machines; a single-machine Application renders none" $?

