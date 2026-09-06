#!/usr/bin/env bash
# CAP-O07: Groups/Teams as an intermediate role-assignment grouping -- a
# role granted to a Group composes with a person's own direct assignment
# (union semantics), takes effect without a restart, and revokes live when
# membership changes. Sourced by run.sh after lib.sh -- assumes lib.sh's
# helpers/ACCOUNTS (incl. FRANK, the ws_default workspace Admin) are already
# in scope. seeds/034_groups_lab.sql provides the Application/Machine/Users;
# no Group exists until this test creates one, proving the /admin/groups UI
# itself, not just GroupStore's own data model.

WATI=$(session_for wati.gl@example.com password)
YUDA=$(session_for yuda.gl@example.com password)

# T194 -- admin-gate negative, same convention every other Admin-only
# endpoint in this suite uses (CAP-X04 T152, CAP-X08 T181, ...): a
# non-Admin (Wati, workspace tier "Member") is denied group creation.
CODE=$(post_status "$BASE_URL/admin/groups" "name=Should+Not+Exist" "$WATI")
[ "$CODE" = "403" ]
check T194 "CAP-O07" "a non-Admin is denied POST /admin/groups (got $CODE)" $?

# T195 -- Admin (Frank) creates a Group. No API to read the new id back
# directly, so scrape it the same way user_option_id already scrapes a
# picker's id from rendered HTML -- the Groups section links each row to
# /admin/groups/{id}.
post_status "$BASE_URL/admin/groups" "name=Groups+Lab+Approvers" "$FRANK" >/dev/null
GROUP_ID=$(curl -s -b "$FRANK" "$BASE_URL/admin/users" | grep -oE '/admin/groups/[a-f0-9-]+' | head -1 | sed 's#/admin/groups/##')
[ -n "$GROUP_ID" ]
check T195 "CAP-O07" "Admin creates a Group, listed on /admin/users (id=$GROUP_ID)" $?

# T196 -- baseline negative: Yuda holds no role at all in app_groups_lab
# (no direct assignment, no Group membership yet) -- denied read,
# CAP-P05's deny-by-default, unaffected by any of this.
CODE=$(curl -s -o /dev/null -w '%{http_code}' -b "$YUDA" "$BASE_URL/mch_gl_ticket")
[ "$CODE" = "403" ]
check T196 "CAP-O07" "a user with no direct role and no Group membership is denied read (got $CODE)" $?

# T197 -- Wati's DIRECT "Editor" assignment (seeds/034) still works exactly
# as CAP-O01 always has -- this change must not regress the direct path.
TICKET_URL=$(post_redirect "$BASE_URL/mch_gl_ticket" "fld_gl_title=Groups+Lab+Ticket" "$WATI")
[ -n "$TICKET_URL" ]
check T197 "CAP-O07" "Wati's direct Editor role still creates a record normally" $?

# T198 -- Editor alone does not grant the Approve event (its own Permission
# row declares no Events) -- Wati is denied BEFORE any Group grant exists,
# the contrast T201 proves against.
CODE=$(post_status "$TICKET_URL/events/evt_gl_approve" "" "$WATI")
[ "$CODE" = "403" ]
check T198 "CAP-O07" "direct Editor role alone does not grant the Approve event (got $CODE)" $?

# Admin assigns "Approver" to the new Group for app_groups_lab, and adds
# Wati as a member -- the two POSTs CAP-O07's whole mechanism rests on.
WATI_ID=$(group_member_checkbox_id "$BASE_URL/admin/groups/$GROUP_ID" "$FRANK" "Groups Lab Wati")
post_status "$BASE_URL/admin/groups/$GROUP_ID/roles" "app_role_app_groups_lab=Approver" "$FRANK" >/dev/null
post_status "$BASE_URL/admin/groups/$GROUP_ID/members" "member_id=$WATI_ID" "$FRANK" >/dev/null

# T199 -- the role assignment actually persisted -- the Group's own edit
# page shows "Approver" selected for Groups Lab.
body_contains "$BASE_URL/admin/groups/$GROUP_ID" 'value="Approver" selected' "$FRANK"
check T199 "CAP-O07" "the Group's Application-role select shows Approver selected after saving" $?

# T200 -- membership actually persisted -- Wati's checkbox is checked (the
# only member added so far, so a bare "checked" substring is unambiguous).
body_contains "$BASE_URL/admin/groups/$GROUP_ID" "checked" "$FRANK"
check T200 "CAP-O07" "Wati appears as a checked member on the Group's edit page" $?

# T201 -- UNION: Wati now holds Editor (direct) + Approver (Group) at
# once -- neither alone was enough (T198), together the Approve event
# succeeds. Session-resolved fresh on this next request, no restart or
# reload needed.
CODE=$(post_status "$TICKET_URL/events/evt_gl_approve" "" "$WATI")
[ "$CODE" = "303" ]
check T201 "CAP-O07" "direct Editor + Group-granted Approver together permit the Approve event (got $CODE)" $?

# T202 -- the Group's grant is scoped to its own members -- Yuda, still not
# a member, still can't even read the Machine.
CODE=$(curl -s -o /dev/null -w '%{http_code}' -b "$YUDA" "$BASE_URL/mch_gl_ticket")
[ "$CODE" = "403" ]
check T202 "CAP-O07" "a Group's grant does not leak to a non-member (got $CODE)" $?

# T203 -- CAP-A03 notify(role="Approver"), fired by T201's Approve action,
# reaches Wati's inbox through Group membership alone -- she never holds
# "Approver" as a direct user_application_roles row. Proves recipientMatch's
# new group_application_roles/group_members OR-clause, not just the
# authorization half. CAP-W06: notify is async (outbox dispatcher ticks
# every 2s) -- bounded wait, same style T99/T100 already use.
sleep 3
body_contains "$BASE_URL/notifications" "Ticket: Approve" "$WATI"
check T203 "CAP-O07" "notify(role=Approver) reaches a user who holds it only via Group membership" $?

# A second ticket, still in Draft, for the revocation check below -- the
# first ticket is already Approved (T201), so re-posting evt_gl_approve to
# it would 400 on CAP-E06's own state guard, not 403 on permission, and
# conflate two different failure reasons.
TICKET2_URL=$(post_redirect "$BASE_URL/mch_gl_ticket" "fld_gl_title=Groups+Lab+Ticket+2" "$WATI")
[ -n "$TICKET2_URL" ]
check T204 "CAP-O07" "Wati (still Editor+Approver) creates and could approve a second ticket before revocation" $?

# Admin removes Wati from the Group (empty member_id set -- SetMembers'
# "full value, not a delta" convention: submitting none clears it).
post_status "$BASE_URL/admin/groups/$GROUP_ID/members" "" "$FRANK" >/dev/null

# T205 -- revocation is just as live as the grant was: Wati's next request
# re-resolves her role set fresh (no separate cache), Approver is gone, only
# direct Editor remains -- Approve is denied again, on the still-Draft
# second ticket, isolating this from CAP-E06's own state guard.
CODE=$(post_status "$TICKET2_URL/events/evt_gl_approve" "" "$WATI")
[ "$CODE" = "403" ]
check T205 "CAP-O07" "removing Wati from the Group revokes Approver immediately, no restart (got $CODE)" $?
