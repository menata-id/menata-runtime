#!/usr/bin/env bash
# Notifications, CRUD permissions, audit log, navigation, workspace isolation, real auth, comparison operators, line items, Action Lab, aggregate-conditioned actions -- still pre-"Batch N", Cases 3-12 territory.
# Sourced by run.sh after lib.sh -- assumes lib.sh's helpers/ACCOUNTS are already in scope.

# --- CAP-A03 (notify to role), CAP-A04 (notify to dynamic recipient),
# CAP-A10 (in-app notification delivery channel) ---

# CAP-W06 (2026-08-23): notify delivery is now dispatcher-mediated (enqueued
# atomically with the triggering event's own write, performed off the
# request path shortly after) rather than synchronous within the request --
# give the dispatcher's ~2s tick time to run before asserting on delivery,
# the same "sleep past the known tick interval" style T99/T100 already use
# for the scheduler's own async nature. Covers T31 and T34/T35 below (their
# own triggering events, T12/T24, already ran earlier in the suite).
sleep 3

# T31 — CAP-A03 + CAP-A10: a static `notify: {role: ...}` action delivers a
# real in-app Notification, not just a log line. Reuses $REC_ID's Approve
# (T12), which fired `notify: {role: Employee}` -- Dave holds Employee in
# app_hr (CAP-O01: matched via his own user_application_roles row, not a
# global role string).
body_contains "$BASE_URL/notifications" "Leave Request: Approve" "$DAVE"
check T31 "CAP-A03,CAP-A10" "static-role notify delivers a real in-app notification" $?

# T32 — CAP-A10: the unread count badge appears on an unrelated page (Home),
# not just the notifications page itself.
body_contains "$BASE_URL/" 'class="ml-1 inline-flex items-center rounded-full bg-red-100' "$DAVE"
check T32 "CAP-A10" "unread notification count badge renders on the nav bar" $?

# T33 — CAP-A10: mark-read actually persists (button disappears for that
# notification, not just a redirect).
NOTIF_PATH=$(get_body "$BASE_URL/notifications" "$DAVE" | grep -oE '/notifications/[a-f0-9-]+/read' | head -1)
CODE="000"
if [ -n "$NOTIF_PATH" ]; then
    CODE=$(post_status "$BASE_URL$NOTIF_PATH" "" "$DAVE")
fi
[ "$CODE" = "303" ] && ! body_contains "$BASE_URL$NOTIF_PATH" "Mark read" "$DAVE"
check T33 "CAP-A10" "marking a notification read persists (got $CODE, path $NOTIF_PATH)" $?

# T34 — CAP-A04: `notify: {recipient_field: fld_ad_submitted_by, role: Submitter}`
# resolves to the record's OWN Submitted By value ("Alice", from T22's
# AD_SEQ_DATA) -- matched by identity (CAP-O01's recipientMatch), not the
# generic Submitter role. Reuses $AD_SEQ_ID, all-approved by T24 (fires
# evt_ad_approve, which carries this action).
body_contains "$BASE_URL/notifications" "Approval Document: Approve" "$ALICE"
check T34 "CAP-A04" "dynamic recipient_field notifies the record's specific submitter, not a role" $?

# T35 — CAP-A04 negative case: Wendy, a real second Submitter in app_approval
# who is NOT the record's own submitter, must NOT also receive it — proves
# recipient_field actually overrode a role-wide broadcast, rather than both
# firing.
! body_contains "$BASE_URL/notifications" "Approval Document: Approve" "$WENDY"
check T35 "CAP-A04" "a different real Submitter does not also receive the dynamically-targeted notification" $?

# --- CAP-P05 (CRUD-level permissions, deny-by-default) ---

# T39 — negative: Eve (Manager) has no permission row at all on mch_employee
# (only HR does) -- read is denied, not implicitly allowed.
CODE=$(curl -s -o /dev/null -w '%{http_code}' -b "$EVE" "$BASE_URL/mch_employee")
[ "$CODE" = "403" ]
check T39 "CAP-P05" "role with no permission row on a machine is denied List (got $CODE)" $?

# T40 — negative: Dave (Employee) has no user_application_roles assignment at
# all in app_approval (only in app_hr) -- Create on mch_approval_document is
# denied.
CODE=$(post_status "$BASE_URL/mch_approval_document" "fld_ad_title=T40" "$DAVE")
[ "$CODE" = "403" ]
check T40 "CAP-P05" "no role assignment in an Application at all is denied Create (got $CODE)" $?

# T41 — negative: same reasoning, the Edit form -- distinct code path from
# T40's Create denial on the same machine.
CODE=$(curl -s -o /dev/null -w '%{http_code}' -b "$DAVE" "$BASE_URL/mch_approval_document/$AD_SEQ_ID/edit")
[ "$CODE" = "403" ]
check T41 "CAP-P05" "no role assignment in an Application at all is denied the Edit form (got $CODE)" $?

# --- CAP-R04 (audit log actor attribution) + CAP-I04 (correlation trace) ---
# DB inspection, same documented exception as T19 -- these prove a DB-level
# fact (record_events columns) an HTTP black-box test can't observe.

if [ -n "$DATABASE_URL" ]; then
    # T42 -- performed_by carries the real acting identity (Eve, who holds
    # Manager in app_hr and triggered T12's Approve on $REC_ID) -- CAP-X02:
    # identity is now always a real account name, never a bare role string
    # (actorLabel prefers identity, and every authenticated request has one).
    # CAP-X06: RLS is live -- a raw psql connection has no app.workspace_id
    # set by default and fails closed (same reasoning as T19's fix above).
    # A DO block sets it as its own top-level statement, sharing the same
    # implicit per-message transaction as the SELECT that follows (so
    # is_local=true still applies) -- a FROM-clause subquery was tried first
    # but is unreliable: the planner is free to choose a join order that
    # evaluates the RLS-filtered scan before the subquery's side effect
    # fires. -q suppresses the DO block's own "DO" completion tag so -tAc's
    # capture is just the real query's output.
    PERFORMED_BY=$(psql "$DATABASE_URL" -q -tAc \
        "DO \$\$ BEGIN PERFORM set_config('app.workspace_id', 'ws_default', true); END \$\$;
         SELECT performed_by FROM record_events WHERE record_id = '$REC_ID' AND event_id = 'evt_lr_approve' LIMIT 1")
    [ "$PERFORMED_BY" = "Eve" ]
    check T42 "CAP-R04" "record_events.performed_by carries the real acting identity (got '$PERFORMED_BY')" $?

    # T43 -- one HTTP request's correlation_id is shared across every
    # record_events row it produces, even across a cascade: AS2's Approve
    # (T24, Carol) triggers aggregate_status, which fires evt_ad_approve on
    # a DIFFERENT record (AD_SEQ_ID) -- both rows must carry the same id.
    CIDS=$(psql "$DATABASE_URL" -q -tAc \
        "DO \$\$ BEGIN PERFORM set_config('app.workspace_id', 'ws_default', true); END \$\$;
         SELECT DISTINCT correlation_id FROM record_events
         WHERE (record_id = '$AS2_ID' AND event_id = 'evt_as_approve') OR (record_id = '$AD_SEQ_ID' AND event_id = 'evt_ad_approve')")
    [ "$(echo "$CIDS" | grep -c .)" = "1" ] && [ -n "$CIDS" ]
    check T43 "CAP-I04" "one correlation_id shared across a same-request cross-record cascade" $?
else
    printf 'SKIP  T42  %-22s %s\n' "CAP-R04" "DATABASE_URL not set -- DB inspection unavailable"
    printf 'SKIP  T43  %-22s %s\n' "CAP-I04" "DATABASE_URL not set -- DB inspection unavailable"
fi

# --- CAP-P05 follow-up fix: Approver could decide a Step but never read the
# Document it belongs to (surfaced by production log data, not a case) ---

# T44 -- Approver (Bob) can now read Approval Document (perm_ad_approver_read)
CODE=$(curl -s -o /dev/null -w '%{http_code}' -b "$BOB" "$BASE_URL/mch_approval_document")
[ "$CODE" = "200" ]
check T44 "CAP-P05" "Approver can read Approval Document, needed for context on the Step they're deciding (got $CODE)" $?

# T45 -- negative: still can't create/edit Documents, read-only
CODE=$(post_status "$BASE_URL/mch_approval_document" "fld_ad_title=T45" "$BOB")
[ "$CODE" = "403" ]
check T45 "CAP-P05" "Approver still denied Create on Approval Document -- read-only, not full access (got $CODE)" $?

# --- CAP-O03 (Navigation metadata: app grouping, role-aware menus, workspace
# home) -- Workspace > Application > Machine, 006-runtime-model.md ---

# T46 -- drilling into an Application lists its own Machines
body_contains "$BASE_URL/apps/app_design" "Design Request" "$ALICE"
check T46 "CAP-O03" "drilling into an Application lists its own Machines" $?

# T47 -- role-aware: someone with zero readable machines in an Application
# never sees that Application's card on the workspace home at all. Alice has
# no user_application_roles row in app_hr at all.
! body_contains "$BASE_URL/" 'href="/ws_default/apps/app_hr"' "$ALICE"
check T47 "CAP-O03" "Alice (no access anywhere in HR) never sees the HR application card" $?

# T48 -- negative: drilling into an Application whose Machines the role
# can't read still 404s the Application route itself correctly, and shows
# no Machine cards for one it partially can't -- Eve (Manager) can read
# mch_leave_request (HR) but not mch_employee (also HR); the HR app page
# must show Leave Request without Employee.
body_contains "$BASE_URL/apps/app_hr" "Leave Request" "$EVE" \
  && ! body_contains "$BASE_URL/apps/app_hr" 'href="/ws_default/mch_employee"' "$EVE"
check T48 "CAP-O03" "within an Application, only individually-readable Machines are listed" $?

# --- CAP-X06 (workspace isolation) -- requires seeds/006_second_workspace.sql ---
# ws_acme (Operations/Task) is a deliberately separate Workspace from every
# other case's ws_default, existing purely to prove isolation.

# T49 -- negative: a ws_default account (Alice) given a direct URL NAMING
# ws_acme (CAP-X14's own /{slug}/ segment) is denied by
# handler.RequireWorkspaceSlug before the request ever reaches a Machine
# lookup at all -- Alice's own session doesn't match the URL's workspace.
# CAP-X06's original app-layer guard (Interpreter.ScopeFor, independent of
# RLS at the DB layer) still exists underneath and would catch this too if
# a Machine ID ever collided across workspaces, but this URL shape now
# trips the earlier, CAP-X14 guard first.
CODE=$(curl -s -o /dev/null -w '%{http_code}' -b "$ALICE" "$BASE_URL_ACME/mch_task")
[ "$CODE" = "404" ]
check T49 "CAP-X06,CAP-X14" "ws_default account denied a URL naming another workspace's Machine (got $CODE)" $?

# T50 -- same, for the Application route
CODE=$(curl -s -o /dev/null -w '%{http_code}' -b "$ALICE" "$BASE_URL_ACME/apps/app_ops")
[ "$CODE" = "404" ]
check T50 "CAP-X06,CAP-X14" "ws_default account denied a URL naming another workspace's Application (got $CODE)" $?

# T51 -- positive: Ivan's own account (ws_acme, Staff in app_ops) can use its
# own Machine end to end (create + trigger event) via its OWN workspace's URL
# (CAP-X14) -- workspace still ultimately comes from the authenticated
# account (CAP-X02), not a client-suppliable cookie; the URL's slug must
# additionally agree with it (RequireWorkspaceSlug), which it does here.
TASK_URL=$(post_redirect "$BASE_URL_ACME/mch_task" "fld_task_title=Conformance+Task" "$IVAN")
TASK_ID="${TASK_URL##*/}"
CODE=$(post_status "$BASE_URL_ACME/mch_task/$TASK_ID/events/evt_task_complete" "" "$IVAN")
[ "$CODE" = "303" ] && body_contains "$TASK_URL" "Done" "$IVAN"
check T51 "CAP-X06" "a workspace account's own session grants access to its own Machine end to end (got $CODE)" $?

if [ -n "$DATABASE_URL" ]; then
    # T52 -- RLS probe (Study 8's own stated pass threshold: "zero
    # cross-workspace rows under RLS probe -- deliberately query with wrong
    # app.workspace_id"). Only meaningful after migrations/009's cutover;
    # skips cleanly before that (RLS not yet enabled -- see that migration's
    # header for why it's applied separately, not as part of migrate-up).
    RLS_ENABLED=$(psql "$DATABASE_URL" -tAc "SELECT relrowsecurity FROM pg_class WHERE relname = 'records'")
    if [ "$RLS_ENABLED" = "t" ]; then
        LEAKED=$(psql "$DATABASE_URL" -q -tAc "
            DO \$\$ BEGIN PERFORM set_config('app.workspace_id', 'ws_default', true); END \$\$;
            SELECT COUNT(*) FROM records WHERE id = '$TASK_ID';")
        [ "$LEAKED" = "0" ]
        check T52 "CAP-X06" "RLS probe: ws_acme's own record invisible under app.workspace_id=ws_default (got count=$LEAKED)" $?
    else
        printf 'SKIP  T52  %-22s %s\n' "CAP-X06" "RLS not yet enabled (migrations/009 applied only at cutover)"
    fi
else
    printf 'SKIP  T52  %-22s %s\n' "CAP-X06" "DATABASE_URL not set -- DB inspection unavailable"
fi

# --- CAP-X02 (real authentication) + CAP-O01 (two-tier role registry) ---
# ADR-005's single largest open item at the time this suite was last
# rewritten: a self-declared role/identity/workspace cookie triplet, no
# password, no session, no CSRF. T53-T59 prove the replacement directly.

# T53 -- wrong password rejected, not silently treated as any other role
CODE=$(curl -s -o /dev/null -w '%{http_code}' -X POST "$ORIGIN/login" \
    --data-urlencode "email=alice@example.com" --data-urlencode "password=wrong")
[ "$CODE" = "401" ]
check T53 "CAP-X02" "login with wrong password rejected (got $CODE)" $?

# T54 -- no session at all: a GET to a protected page redirects to /login
# rather than serving content or erroring
CODE=$(curl -s -o /dev/null -w '%{http_code}' "$BASE_URL/")
[ "$CODE" = "303" ]
check T54 "CAP-X02" "unauthenticated GET redirected to /login (got $CODE)" $?

# T55 -- no session on a POST: 401, not a redirect (nothing useful to
# redirect a form submission to)
CODE=$(curl -s -o /dev/null -w '%{http_code}' -X POST "$BASE_URL/notifications/00000000-0000-0000-0000-000000000000/read")
[ "$CODE" = "401" ]
check T55 "CAP-X02" "unauthenticated POST rejected with 401 (got $CODE)" $?

# T56 -- CSRF: a real, authenticated session's POST with no csrf_token at all
# is rejected -- proves the check runs even when the session itself is valid.
CODE=$(post_status_no_csrf "$BASE_URL/notifications/00000000-0000-0000-0000-000000000000/read" "" "$DAVE")
[ "$CODE" = "403" ]
check T56 "CAP-X02" "authenticated request with no CSRF token rejected (got $CODE)" $?

# T57 -- admin page denied to a non-Admin (Alice: workspace_role Member)
CODE=$(curl -s -o /dev/null -w '%{http_code}' -b "$ALICE" "$BASE_URL/admin/users")
[ "$CODE" = "403" ]
check T57 "CAP-O01" "non-Admin denied /admin/users (got $CODE)" $?

# T58 -- admin page reachable to a real workspace Admin (Frank), and lists
# users -- the "manage user access" half of CAP-O01's workspace-tier role.
CODE=$(curl -s -o /dev/null -w '%{http_code}' -b "$FRANK" "$BASE_URL/admin/users")
[ "$CODE" = "200" ] && body_contains "$BASE_URL/admin/users" "Manage users" "$FRANK"
check T58 "CAP-O01" "workspace Admin can reach /admin/users (got $CODE)" $?

# T59 -- the actual point of the two-tier role model: one identity (Alice),
# one session, no role-switch step, resolving a DIFFERENT role in each of two
# Applications -- Requester in Design, Submitter in Approval.
body_contains "$BASE_URL/apps/app_design" "Design Request" "$ALICE" \
  && body_contains "$BASE_URL/apps/app_approval" "Approval Document" "$ALICE"
check T59 "CAP-O01" "one identity resolves a different role per Application in the same session, no switch" $?

# --- CAP-C05 (comparison operators), CAP-C07 (cross-field comparison),
# CAP-C12 (uniqueness constraint), CAP-X05 (operator validated at load time) ---

# T60 -- CAP-C07: End Date before Start Date rejected -- compares against
# another Field's own value (fld_lr_start_date), not a hardcoded literal.
LR_BADRANGE_DATA="fld_lr_employee=$DAVE_ID&fld_lr_leave_type=Annual+Leave&fld_lr_start_date=2030-01-10&fld_lr_end_date=2030-01-05&fld_lr_reason=T60"
post_body_contains "$BASE_URL/mch_leave_request" "$LR_BADRANGE_DATA" "End Date must be after Start Date" "$DAVE"
check T60 "CAP-C07" "cross-field comparison rejects End Date before Start Date" $?

# T61 -- CAP-C05: a non-positive Sequence rejected -- generalizes "after" and
# adds greater_than/less_than beyond the old today-only special case.
AS_BADSEQ_DATA="fld_as_document=$AD_SEQ_ID&fld_as_approver=$BOB_ID&fld_as_sequence=0"
post_body_contains "$BASE_URL/mch_approval_step" "$AS_BADSEQ_DATA" "Sequence must be a positive number" "$ALICE"
check T61 "CAP-C05" "greater_than rejects a non-positive Sequence" $?

# T62 -- CAP-C12: a second Step at the same Sequence on the same Document
# rejected -- composite uniqueness (document, sequence); reusing sequence 1
# on a DIFFERENT document (T22-T26's Parallel-mode AD_PAR_ID) is fine, only
# a collision within the SAME document's own Steps is rejected.
AS_DUPSEQ_DATA="fld_as_document=$AD_SEQ_ID&fld_as_approver=$CAROL_ID&fld_as_sequence=1"
post_body_contains "$BASE_URL/mch_approval_step" "$AS_DUPSEQ_DATA" "already has a Step at that Sequence" "$ALICE"
check T62 "CAP-C12" "composite uniqueness rejects a duplicate (document, sequence) pair" $?

# --- CAP-F16 (line items / child table inside a record) ---

# T63 -- positive: a Journal Entry and both its Lines are created atomically
# from one form submission (child_N_<field> indexed rows), not N+1 separate
# Create trips -- the child sub-list (CAP-V06) then shows both on the
# parent's own detail page.
JE_DATA="fld_je_date=2026-07-12&fld_je_memo=Conformance+T63&child_0_fld_jel_account=Office+Expense&child_0_fld_jel_debit=100&child_1_fld_jel_account=Cash&child_1_fld_jel_credit=100"
JE_URL=$(post_redirect "$BASE_URL/mch_journal_entry" "$JE_DATA" "$IVY")
[ -n "$JE_URL" ] && body_contains "$JE_URL" "Office Expense" "$IVY" && body_contains "$JE_URL" "Cash" "$IVY"
check T63 "CAP-F16" "Journal Entry and both Lines created atomically from one submission" $?

# T64 -- negative: an invalid child row (Account required, left blank)
# blocks the WHOLE submission -- no orphan parent, matching this
# capability's own "atomic-with-parent" promise, not just a UI convenience.
JE_BAD_DATA="fld_je_date=2026-07-12&fld_je_memo=Conformance+T64&child_0_fld_jel_debit=50"
post_body_contains "$BASE_URL/mch_journal_entry" "$JE_BAD_DATA" "Journal Entry Line, row 1: Account is required" "$IVY"
check T64 "CAP-F16" "an invalid child row blocks the whole atomic create, not just that row" $?

# --- CAP-A06/A09/A11/A12/A13/A15 (Action Lab: one event, six actions) ---

AL_PROJ_URL=$(post_redirect "$BASE_URL/mch_al_project" "fld_alp_name=Conformance+Project" "$PAM")
AL_PROJ_ID="${AL_PROJ_URL##*/}"
AL_TASK_URL=$(post_redirect "$BASE_URL/mch_al_task" "fld_alt_title=Ship+the+thing&fld_alt_priority=Urgent&fld_alt_stage=Todo&fld_alt_project=$AL_PROJ_ID" "$PAM")
AL_TASK_ID="${AL_TASK_URL##*/}"
AL_TASKS_BEFORE=$(count_all_pages "$BASE_URL/mch_al_task" "Follow-up" "$PAM")
post_status "$BASE_URL/mch_al_task/$AL_TASK_ID/events/evt_al_task_complete" "" "$PAM" >/dev/null
# CAP-W06 (2026-08-23): T67 below asserts on notify delivery, now
# dispatcher-mediated -- give the ~2s tick time to run first (same style
# T99/T100 already use for the scheduler's own async nature).
sleep 3

# T65 -- CAP-A12: Stage steps to the next declared value_list option (Todo ->
# Doing) -- checked on the Detail page as plain rendered text (StatusBadge),
# not a form picker's "selected" attribute, which only a New/Edit form renders.
body_contains "$AL_TASK_URL" ">Doing<" "$PAM"
check T65 "CAP-A12" "value_list field steps to its next declared option" $?

# T66 -- CAP-A11: Follow Up Date = today + 7 Days (flat date arithmetic).
EXPECTED_FOLLOWUP=$(date -d "+7 days" +%Y-%m-%d 2>/dev/null || date -v+7d +%Y-%m-%d)
body_contains "$AL_TASK_URL" "$EXPECTED_FOLLOWUP" "$PAM"
check T66 "CAP-A11" "date arithmetic resolves today + 7 Days to the real date (expected $EXPECTED_FOLLOWUP)" $?

# T67 -- CAP-A09: the notify action's own "if" fired (Priority was Urgent).
body_contains "$BASE_URL/notifications" "Task: Complete" "$PAM"
check T67 "CAP-A09" "a conditional action's \"if\" runs the action when its condition is true" $?

# T68 -- CAP-A06: create_record made a Task Log entry, copying this Task's
# own Title via "field:<id>", not a literal.
body_contains "$BASE_URL/mch_al_task_log" "Ship the thing" "$PAM"
check T68 "CAP-A06" "create_record creates a real record on another Machine, copying a source field" $?

# T69 -- CAP-A13: cross_set_field stamped the linked Project's own field,
# not this Task's -- record_field resolved the Task's reference to find it.
body_contains "$AL_PROJ_URL" "$(date +%Y-%m-%d)" "$PAM"
check T69 "CAP-A13" "cross_set_field updates a field on a DIFFERENT record via a reference field" $?

# T70 -- CAP-A15: batch_generate created 2 new Tasks from one action.
AL_TASKS_AFTER=$(count_all_pages "$BASE_URL/mch_al_task" "Follow-up" "$PAM")
[ "$((AL_TASKS_AFTER - AL_TASKS_BEFORE))" = "2" ]
check T70 "CAP-A15" "batch_generate creates N records from one action (got $((AL_TASKS_AFTER - AL_TASKS_BEFORE)), want 2)" $?

# T71 -- CAP-A09 negative: a Normal-priority Task's Complete does NOT notify
# -- proves "if" actually gates the action, not a coincidence of T67's data.
# Before/after count (not an absolute number) since this suite's own earlier
# runs against a persistent database leave prior "Task: Complete"
# notifications in place -- same reasoning as T70's before/after.
NOTIF_COUNT_BEFORE=$(get_body "$BASE_URL/notifications" "$PAM" | grep -o "Task: Complete" | wc -l)
AL_TASK2_URL=$(post_redirect "$BASE_URL/mch_al_task" "fld_alt_title=Quiet+Task&fld_alt_priority=Normal&fld_alt_stage=Todo" "$PAM")
AL_TASK2_ID="${AL_TASK2_URL##*/}"
post_status "$BASE_URL/mch_al_task/$AL_TASK2_ID/events/evt_al_task_complete" "" "$PAM" >/dev/null
# CAP-W06: must wait here too, or this negative case would trivially pass
# for the wrong reason (dispatcher hasn't ticked yet), not because CAP-A09's
# "if" correctly skipped the action -- same reasoning as T105.
sleep 3
NOTIF_COUNT_AFTER=$(get_body "$BASE_URL/notifications" "$PAM" | grep -o "Task: Complete" | wc -l)
[ "$NOTIF_COUNT_AFTER" = "$NOTIF_COUNT_BEFORE" ]
check T71 "CAP-A09" "a Normal-priority Task's Complete does not also notify (count stayed $NOTIF_COUNT_BEFORE)" $?

# --- CAP-A14 (aggregate-conditioned action) ---
# MEMBER is unique per run ($$, this script's own PID) -- SUM(points) must
# start at 0 for a member no earlier run's leftover Point Entry rows could
# have touched, or T72's "still under threshold" premise breaks the second
# time this suite runs against a persistent database.

AL_MEMBER="Conformance+Member+$$"

# T72 -- negative: Award Badge rejected while this Member's own Point Entry
# total is still under the threshold (SUM < 100).
PE1_URL=$(post_redirect "$BASE_URL/mch_al_point_entry" "fld_alpe_member=$AL_MEMBER&fld_alpe_points=40" "$PAM")
PE1_ID="${PE1_URL##*/}"
CODE=$(post_status "$BASE_URL/mch_al_point_entry/$PE1_ID/events/evt_pe_award" "" "$PAM")
[ "$CODE" = "400" ]
check T72 "CAP-A14" "aggregate-conditioned trigger rejected while SUM is still under threshold (got $CODE)" $?

# T73 -- positive: once this same Member's total reaches >= 100 (40 + 65),
# the SAME event on a new Point Entry row succeeds and its own action
# (CAP-A06 again) creates the Badge.
PE2_URL=$(post_redirect "$BASE_URL/mch_al_point_entry" "fld_alpe_member=$AL_MEMBER&fld_alpe_points=65" "$PAM")
PE2_ID="${PE2_URL##*/}"
CODE=$(post_status "$BASE_URL/mch_al_point_entry/$PE2_ID/events/evt_pe_award" "" "$PAM")
[ "$CODE" = "303" ] && body_contains "$BASE_URL/mch_al_badge" "Conformance Member $$" "$PAM"
check T73 "CAP-A14" "aggregate-conditioned trigger succeeds once SUM crosses the threshold, action creates the Badge (got $CODE)" $?

