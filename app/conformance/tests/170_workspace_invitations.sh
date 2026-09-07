#!/usr/bin/env bash
# CAP-O10: invite someone into an existing workspace by email (Study 35
# §5.3), built after CAP-O11 already separated identity from membership --
# internal/handler/invitations.go's own doc comments explain how that
# resolves Study 35's own flagged "email already has an account elsewhere"
# ambiguity (a new membership row on the real identity, never a second
# disconnected one). Sourced by run.sh after lib.sh.
#
# Needs SERVER_LOG (scripts/local-ci.sh) to recover an invitation's own
# token: it's never persisted anywhere except the emailed link text
# (internal/mailer's logSender fallback in this dev/CI configuration, no
# SMTP_HOST configured) -- the same "HTTP black-box exception, documented"
# precedent T19's own psql use already established. Every test below is
# skipped, not silently omitted, when SERVER_LOG is unset.

invitation_token_for() { # <email> -> echoes the most recent token emailed to it
    local email="$1" line body
    # Two jq passes, not one: the emailed body is a multi-line string, so a
    # first pass with -r (raw strings) would print several PHYSICAL lines
    # per JSON record, and a trailing `tail -1` (meant to pick the latest
    # of several invitations to the same address) would instead truncate to
    # that one body's own last line. -c (compact) keeps each JSON record on
    # exactly one line -- embedded newlines stay escaped as literal \n --
    # so `tail -1` selects among RECORDS, and only the second pass decodes
    # the chosen one's .body back into real newlines.
    line=$(grep '"msg":"email not sent' "$SERVER_LOG" 2>/dev/null \
        | jq -c --arg to "$email" 'select(.to == $to)' | tail -1)
    [ -n "$line" ] || return 1
    body=$(printf '%s' "$line" | jq -r '.body')
    printf '%s' "$body" | grep -oE 'token=[A-Za-z0-9_-]+' | tail -1 | sed 's/^token=//'
}

# invitation_id_for <path-suffix> -> echoes the newest invitation row's own
# id, resolved from whichever action form matches that suffix (e.g.
# "revoke", "resend") on the live /admin/invitations page -- rows are
# newest-first (InvitationStore.ListByWorkspace), so right after creating
# one it's always the first match.
invitation_id_for() {
    curl -s -b "$FRANK" "$BASE_URL/admin/invitations" \
        | grep -oE "admin/invitations/[^\"/]+/$1" | head -1 \
        | sed -E "s#admin/invitations/([^/]+)/$1#\1#"
}

if [ -z "${SERVER_LOG:-}" ]; then
    printf 'SKIP  T226-T235 %-22s %s\n' "CAP-O10" "SERVER_LOG not set -- invitation token unrecoverable"
else

# T226 -- admin-gate negative, same convention every other Admin-only
# endpoint already follows.
CODE=$(post_status "$BASE_URL/admin/invitations" "email=x%40example.com&workspace_role=Member" "$ALICE")
[ "$CODE" = "403" ]
check T226 "CAP-O10" "a non-Admin is denied POST /admin/invitations (got $CODE)" $?

# T227 -- Admin (Frank) invites a brand-new email; listed pending.
FRANK_CSRF=$(csrf_for "$FRANK" "$BASE_URL")
curl -s -o /dev/null -X POST -b "$FRANK" "$BASE_URL/admin/invitations" \
    --data-urlencode "email=newperson@example.com" --data-urlencode "workspace_role=Member" \
    --data-urlencode "csrf_token=$FRANK_CSRF"
body_contains "$BASE_URL/admin/invitations" "newperson@example.com" "$FRANK"
check T227 "CAP-O10" "Admin creates an invitation, listed on /admin/invitations" $?

NEW_TOKEN=$(invitation_token_for "newperson@example.com")

# T228 -- GET .../invite/accept?token=... for a brand-new email shows the
# new-account form (Name+Password) -- no identity exists yet.
ACCEPT_BODY=$(curl -s "$BASE_URL/invite/accept?token=$NEW_TOKEN")
echo "$ACCEPT_BODY" | grep -q 'name="name"'
check T228 "CAP-O10" "a new-email invitation's accept page shows the new-account form" $?

# T229 -- accepting it creates a real account + membership: the redirect
# lands in ws_default, and a completely fresh login with the just-set
# password succeeds afterward (invite/accept is CSRF-exempt, same reasoning
# /login's own POST already is -- no session exists yet to check a token
# against).
ACCEPT_HEADERS=$(mktemp)
curl -s -D "$ACCEPT_HEADERS" -o /dev/null -X POST "$BASE_URL/invite/accept" \
    --data-urlencode "token=$NEW_TOKEN" --data-urlencode "name=New Person" --data-urlencode "password=newpersonpw"
ACCEPT_LOCATION=$(grep -i '^location' "$ACCEPT_HEADERS" | tr -d '\r' | sed -E 's/^[Ll]ocation: //')
RELOGIN_HEADERS=$(mktemp)
curl -s -D "$RELOGIN_HEADERS" -o /dev/null -X POST "$ORIGIN/login" \
    --data-urlencode "email=newperson@example.com" --data-urlencode "password=newpersonpw"
RELOGIN_LOCATION=$(grep -i '^location' "$RELOGIN_HEADERS" | tr -d '\r' | sed -E 's/^[Ll]ocation: //')
[ "$ACCEPT_LOCATION" = "/ws_default/" ] && [ "$RELOGIN_LOCATION" = "/ws_default/" ]
check T229 "CAP-O10" "accepting creates a real account+membership (accept=$ACCEPT_LOCATION, relogin=$RELOGIN_LOCATION)" $?

# T230 -- inviting an email that already has a real identity elsewhere
# (staff@example.com, ws_acme's own Admin, no membership in ws_default yet)
# shows the password-confirm form, not a second new-account form -- CAP-O11's
# model means accepting only ever adds a membership, never a disconnected
# second identity.
curl -s -o /dev/null -X POST -b "$FRANK" "$BASE_URL/admin/invitations" \
    --data-urlencode "email=staff@example.com" --data-urlencode "workspace_role=Member" \
    --data-urlencode "csrf_token=$FRANK_CSRF"
STAFF_TOKEN=$(invitation_token_for "staff@example.com")
STAFF_BODY=$(curl -s "$BASE_URL/invite/accept?token=$STAFF_TOKEN")
echo "$STAFF_BODY" | grep -q 'already has a Menata account' && ! echo "$STAFF_BODY" | grep -q 'name="name"'
check T230 "CAP-O10" "inviting an email with an existing identity elsewhere shows the password-confirm form" $?

# T231 -- the wrong password is rejected, no membership granted.
WRONG_CODE=$(curl -s -o /dev/null -w '%{http_code}' -X POST "$BASE_URL/invite/accept" \
    --data-urlencode "token=$STAFF_TOKEN" --data-urlencode "password=totallywrongpw")
[ "$WRONG_CODE" = "400" ]
check T231 "CAP-O10" "an existing identity's wrong password is rejected (got $WRONG_CODE)" $?

# T232 -- the real password joins the workspace: staff@example.com now has
# 2 real memberships (ws_acme, from seeds, plus this new one) -- a
# completely fresh, cookie-less login is sent to the picker (CAP-O11),
# proving a real second membership row exists, not just a 303 that did
# nothing.
curl -s -o /dev/null -X POST "$BASE_URL/invite/accept" \
    --data-urlencode "token=$STAFF_TOKEN" --data-urlencode "password=password"
STAFF_RELOGIN_HEADERS=$(mktemp)
curl -s -D "$STAFF_RELOGIN_HEADERS" -o /dev/null -X POST "$ORIGIN/login" \
    --data-urlencode "email=staff@example.com" --data-urlencode "password=password"
STAFF_RELOGIN_LOCATION=$(grep -i '^location' "$STAFF_RELOGIN_HEADERS" | tr -d '\r' | sed -E 's/^[Ll]ocation: //')
[ "$STAFF_RELOGIN_LOCATION" = "/choose-workspace" ]
check T232 "CAP-O10" "the real password joins the workspace -- a fresh login now shows the picker (got $STAFF_RELOGIN_LOCATION)" $?

# T233 -- revoke takes effect immediately: the same token that used to
# resolve to the accept form now shows the generic invalid page.
curl -s -o /dev/null -X POST -b "$FRANK" "$BASE_URL/admin/invitations" \
    --data-urlencode "email=revoke-test@example.com" --data-urlencode "workspace_role=Member" \
    --data-urlencode "csrf_token=$FRANK_CSRF"
REVOKE_TOKEN=$(invitation_token_for "revoke-test@example.com")
REVOKE_ID=$(invitation_id_for revoke)
post_status "$BASE_URL/admin/invitations/$REVOKE_ID/revoke" "" "$FRANK" >/dev/null
body_contains "$BASE_URL/invite/accept?token=$REVOKE_TOKEN" "invalid or has expired" ""
check T233 "CAP-O10" "a revoked invitation's own link stops working" $?

# T234 -- an unknown/bogus token is the same generic outcome, not a crash or
# a distinguishing error (no token-guessing oracle, same hygiene CAP-X02's
# own login-failure handling already established).
body_contains "$BASE_URL/invite/accept?token=totally-bogus-token" "invalid or has expired" ""
check T234 "CAP-O10" "an unknown token shows the same generic invalid page, not a crash" $?

# T235 -- resend reissues a fresh token in place: the OLD one stops working,
# the NEW one still resolves to a real accept form.
curl -s -o /dev/null -X POST -b "$FRANK" "$BASE_URL/admin/invitations" \
    --data-urlencode "email=resend-test@example.com" --data-urlencode "workspace_role=Member" \
    --data-urlencode "csrf_token=$FRANK_CSRF"
OLD_TOKEN=$(invitation_token_for "resend-test@example.com")
RESEND_ID=$(invitation_id_for resend)
post_status "$BASE_URL/admin/invitations/$RESEND_ID/resend" "" "$FRANK" >/dev/null
NEW_RESEND_TOKEN=$(invitation_token_for "resend-test@example.com")
body_contains "$BASE_URL/invite/accept?token=$OLD_TOKEN" "invalid or has expired" "" \
    && body_contains "$BASE_URL/invite/accept?token=$NEW_RESEND_TOKEN" 'name="name"' ""
check T235 "CAP-O10" "resend invalidates the old token and issues a working new one" $?

fi
