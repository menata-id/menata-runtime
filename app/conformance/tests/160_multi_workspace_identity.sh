#!/usr/bin/env bash
# CAP-O11: multi-workspace identity + login-time workspace picker (Study 36,
# ../../benchmarks/028-multi-workspace-identity-benchmark.md). Sourced by
# run.sh after lib.sh. seeds/039_multi_workspace_identity_lab.sql provides
# the one fixture with genuinely two real memberships (ws_default, ws_acme)
# -- every other seeded account has exactly one, which is exactly what
# T225 below re-proves stays a zero-friction, unchanged path.
#
# Deliberately raw curl throughout, not session_for/csrf_for -- those cache
# one session per email for the whole suite run, but this file needs
# several genuinely fresh logins for the SAME email to prove the picker,
# the remembered-choice skip, and the regression check each independently.

# T220 -- an identity with 2+ real workspace memberships is sent to the
# picker on login, not auto-entered into either one.
MW_JAR=$(mktemp)
MW_HEADERS=$(mktemp)
curl -s -c "$MW_JAR" -D "$MW_HEADERS" -o /dev/null -X POST "$ORIGIN/login" \
    --data-urlencode "email=multiworkspace@example.com" --data-urlencode "password=password"
MW_LOCATION=$(grep -i '^location' "$MW_HEADERS" | tr -d '\r' | sed -E 's/^[Ll]ocation: //')
[ "$MW_LOCATION" = "/choose-workspace" ]
check T220 "CAP-O11" "an identity with 2+ workspace memberships is sent to the picker, not auto-entered (got $MW_LOCATION)" $?

# T221 -- the picker itself lists every real membership (not a hardcoded
# set) -- both ws_default and ws_acme.
PICKER_BODY=$(curl -s -b "$MW_JAR" "$ORIGIN/choose-workspace")
echo "$PICKER_BODY" | grep -q 'menata.app/ws_default/' && echo "$PICKER_BODY" | grep -q 'menata.app/ws_acme/'
check T221 "CAP-O11" "the picker lists every real membership (ws_default and ws_acme)" $?

MW_CSRF=$(echo "$PICKER_BODY" | grep -oE 'name="csrf_token" value="[^"]*"' | head -1 | sed -E 's/.*value="([^"]*)"/\1/')

# T222 -- picking a real membership lands in that exact workspace, and the
# session is now usable there (not stuck mid-picker).
CHOOSE_HEADERS=$(mktemp)
curl -s -b "$MW_JAR" -c "$MW_JAR" -D "$CHOOSE_HEADERS" -o /dev/null -X POST "$ORIGIN/choose-workspace" \
    --data-urlencode "workspace_id=ws_default" --data-urlencode "csrf_token=$MW_CSRF"
CHOOSE_LOCATION=$(grep -i '^location' "$CHOOSE_HEADERS" | tr -d '\r' | sed -E 's/^[Ll]ocation: //')
[ "$CHOOSE_LOCATION" = "/ws_default/" ] && body_contains "$ORIGIN/ws_default/" "Yusuf" "$MW_JAR"
check T222 "CAP-O11" "picking a real membership lands in that exact workspace, session usable there (got $CHOOSE_LOCATION)" $?

# T223 -- negative: a submitted workspace_id is checked against this
# identity's OWN real memberships again server-side, never trusted from the
# client blindly (CAP-P05's own "deny by default" discipline) -- rejected
# with a clean message, not silently honored.
INVALID_BODY=$(curl -s -b "$MW_JAR" -X POST "$ORIGIN/choose-workspace" \
    --data-urlencode "workspace_id=ws_bogus_nonexistent" --data-urlencode "csrf_token=$MW_CSRF")
echo "$INVALID_BODY" | grep -q "not one of your workspaces"
check T223 "CAP-O11" "a workspace_id this identity does not actually hold is rejected, not silently honored" $?

# T224 -- a second, genuinely fresh login remembers the last chosen
# workspace (a plain cookie, UX default only) and skips the picker
# entirely -- matching Notion/Basecamp's own "don't ask again unless
# there's a reason to."
RELOGIN_HEADERS=$(mktemp)
curl -s -b "$MW_JAR" -c "$MW_JAR" -D "$RELOGIN_HEADERS" -o /dev/null -X POST "$ORIGIN/login" \
    --data-urlencode "email=multiworkspace@example.com" --data-urlencode "password=password"
RELOGIN_LOCATION=$(grep -i '^location' "$RELOGIN_HEADERS" | tr -d '\r' | sed -E 's/^[Ll]ocation: //')
[ "$RELOGIN_LOCATION" = "/ws_default/" ]
check T224 "CAP-O11" "a second login remembers the last chosen workspace, skips the picker (got $RELOGIN_LOCATION)" $?

# T225 -- regression, the one that matters most: an existing
# single-membership account still auto-enters with zero picker friction,
# byte-identical to before CAP-O11 existed.
ALICE_HEADERS=$(mktemp)
curl -s -D "$ALICE_HEADERS" -o /dev/null -X POST "$ORIGIN/login" \
    --data-urlencode "email=alice@example.com" --data-urlencode "password=password"
ALICE_LOCATION=$(grep -i '^location' "$ALICE_HEADERS" | tr -d '\r' | sed -E 's/^[Ll]ocation: //')
[ "$ALICE_LOCATION" = "/ws_default/" ]
check T225 "CAP-O11" "an existing single-membership account still auto-enters with zero picker friction (got $ALICE_LOCATION)" $?
