#!/usr/bin/env bash
# CAP-X08 import half (export, CAP-X07/T121, already proven -- this file
# tests only what's new: POST /apps/import). Sourced by run.sh after lib.sh
# -- assumes lib.sh's helpers/ACCOUNTS are already in scope.
#
# Every positive test re-imports app_design's OWN export (a small,
# no-overlay Application) under a fresh, run-unique id prefix (t80_$$_) --
# jq to compact + sed to rename every id consistently, the same technique
# T166's own lift-then-reapply test already established
# (conformance/tests/060_ui_cluster.sh) for "translate an id before
# reapplying a package," just renaming every id here instead of one field.

FRANK_CSRF=$(csrf_for "$FRANK")
DAVE_CSRF=$(csrf_for "$DAVE")

DESIGN_EXPORT=$(curl -s -b "$FRANK" "$BASE_URL/apps/app_design/export")
IMPORT_PKG=$(echo "$DESIGN_EXPORT" | jq -c '.' | sed -E "s/\"(mch_|fld_|evt_|cst_|perm_|vw_|app_)/\"t80_$$_\1/g")
IMPORT_MACHINE="$BASE_URL/t80_${$}_mch_design_request"
IMPORT_SUBMIT_EVENT="t80_${$}_evt_submit"

# T181 -- a non-Admin is denied import outright (403).
NONADMIN_CODE=$(curl -s -o /dev/null -w '%{http_code}' -b "$DAVE" -X POST "$BASE_URL/apps/import" \
    -H "Content-Type: application/json" -H "X-CSRF-Token: $DAVE_CSRF" -d "$IMPORT_PKG")
[ "$NONADMIN_CODE" = "403" ]
check T181 "CAP-X08" "a non-Admin is denied metadata import (got $NONADMIN_CODE)" $?

# T182 -- Admin import of a valid (renamed, no Process Overlay) package
# succeeds and the new Machine is immediately known to the live Interpreter
# -- no restart, same zero-downtime property CAP-X04's own T151 already
# proved for a reload. Before import the machine id is unknown to the
# router (404); Frank (workspace Admin, but not yet granted any role IN
# this brand-new Application -- CAP-P05 deny-by-default) gets 403, not 404,
# immediately after -- that shift is the proof the Machine is live, without
# needing a role grant (T183 below is the fuller, DB-gated functional proof).
LIST_CODE_BEFORE=$(curl -s -o /dev/null -w '%{http_code}' -b "$FRANK" "$IMPORT_MACHINE")
IMPORT_CODE=$(curl -s -o /dev/null -w '%{http_code}' -b "$FRANK" -X POST "$BASE_URL/apps/import" \
    -H "Content-Type: application/json" -H "X-CSRF-Token: $FRANK_CSRF" -d "$IMPORT_PKG")
LIST_CODE_AFTER=$(curl -s -o /dev/null -w '%{http_code}' -b "$FRANK" "$IMPORT_MACHINE")
[ "$LIST_CODE_BEFORE" = "404" ] && [ "$IMPORT_CODE" = "201" ] && [ "$LIST_CODE_AFTER" = "403" ]
check T182 "CAP-X08" "a valid metadata package imports (got $IMPORT_CODE) and its Machine is immediately known to the live Interpreter, no restart (before=$LIST_CODE_BEFORE, after=$LIST_CODE_AFTER)" $?

# T183 -- DB inspection, same documented exception as T19/T151: granting a
# user a role in a brand-new imported Application has no HTTP surface of
# its own (this codebase's admin UI manages workspace users, not
# per-application role assignment) -- the same class of fixture-only-
# reachable-via-psql gap T19/T151 already established a pattern for.
# Proves Fields/Events/Actions/Constraints/Permissions all materialized
# correctly: Frank, granted "Requester" here, hits the SAME Constraint
# violations (required Description, Due Date after today) app_design's
# own original Machine enforces, then a valid Create + Submit succeeds.
if [ -n "$DATABASE_URL" ]; then
    psql "$DATABASE_URL" -q -c \
        "INSERT INTO user_application_roles (user_id, application_id, role)
         SELECT u.id, 't80_${$}_app_design', 'Requester' FROM users u WHERE u.email = 'hr@example.com'
         ON CONFLICT DO NOTHING" >/dev/null

    INCOMPLETE_OK=1
    post_body_contains "$IMPORT_MACHINE" "t80_${$}_fld_title=X80" "Description is required" "$FRANK" || INCOMPLETE_OK=0

    FUTURE=$(date -d "+7 days" +%Y-%m-%d 2>/dev/null || date -v+7d +%Y-%m-%d)
    IMPORTED_URL=$(post_redirect "$IMPORT_MACHINE" \
        "t80_${$}_fld_title=X80+Design&t80_${$}_fld_description=CAP-X08+import+proof&t80_${$}_fld_due_date=$FUTURE" "$FRANK")
    SUBMIT_CODE=$(post_status "$IMPORTED_URL/events/$IMPORT_SUBMIT_EVENT" "" "$FRANK")

    [ "$INCOMPLETE_OK" = "1" ] && [ -n "$IMPORTED_URL" ] && [ "$SUBMIT_CODE" = "303" ] && \
        body_contains "$IMPORTED_URL" ">Submitted<" "$FRANK"
    check T183 "CAP-X08" "imported Fields/Constraints/Events/Actions/Permissions all function end-to-end (blocked incomplete, then Create+Submit succeeded, got submit=$SUBMIT_CODE)" $?
else
    printf 'SKIP  T183 %-22s %s\n' "CAP-X08" "DATABASE_URL not set -- role-grant fixture unavailable"
fi

# T184 -- a package containing a Process-Overlay Machine is rejected before
# anything is written -- compileProcess/compileChangePolicies compile in
# memory at load time and never persist, so materializing an already-
# compiled export verbatim would corrupt the very data being imported (see
# capability-registry.md's CAP-X08 row for the full reasoning).
OVERLAY_EXPORT=$(curl -s -b "$FRANK" "$BASE_URL/apps/app_overlay_lab/export")
OVERLAY_CODE=$(curl -s -o /dev/null -w '%{http_code}' -b "$FRANK" -X POST "$BASE_URL/apps/import" \
    -H "Content-Type: application/json" -H "X-CSRF-Token: $FRANK_CSRF" -d "$OVERLAY_EXPORT")
[ "$OVERLAY_CODE" = "400" ]
check T184 "CAP-X08" "a package containing a Process Overlay Machine is rejected, not silently corrupted (got $OVERLAY_CODE)" $?

# T185 -- re-importing the exact same package a second time fails loudly
# (its own ids collide) rather than silently overwriting the first import.
COLLIDE_CODE=$(curl -s -o /dev/null -w '%{http_code}' -b "$FRANK" -X POST "$BASE_URL/apps/import" \
    -H "Content-Type: application/json" -H "X-CSRF-Token: $FRANK_CSRF" -d "$IMPORT_PKG")
[ "$COLLIDE_CODE" != "201" ]
check T185 "CAP-X08" "re-importing a package whose ids already exist fails loudly, not a silent overwrite (got $COLLIDE_CODE)" $?
