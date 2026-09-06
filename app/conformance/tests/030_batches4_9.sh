#!/usr/bin/env bash
# Batch 4 (Views) through Batch 9 (Workspace Services), 2026-07-12 -- the run this codebase's own commit history calls "Batch N".
# Sourced by run.sh after lib.sh -- assumes lib.sh's helpers/ACCOUNTS are already in scope.

# --- Batch 4: Views (2026-07-12) ---
# seeds/010_views_lab.sql. V11 (channel-independent rendering) stays
# excluded -- capability-registry.md HOLDs it at Proposed pending a second
# independent source, unaffected by this batch.

VERA_ID=$(user_option_id "$BASE_URL/mch_vl_task/new" "$VERA" "Vera")

# T74 -- CAP-V04/V05/V09 combined: "My Overdue Tasks" only shows a Task
# that is BOTH assigned to the viewer ($current_user, CAP-V05) AND overdue
# (a plain declarative clause, CAP-V09) -- sorted soonest-due-first
# (default_sort, CAP-V04).
post_redirect "$BASE_URL/mch_vl_task" "fld_vlt_title=Vera+Overdue+$$&fld_vlt_assignee=$VERA_ID&fld_vlt_due=2020-01-01" "$VERA" >/dev/null
body_contains "$BASE_URL/mch_vl_task" "Vera Overdue $$" "$VERA"
check T74 "CAP-V04,CAP-V05,CAP-V09" "my-records filter shows a Task assigned to me and overdue" $?

# T75 -- negative: a Task assigned to the SAME viewer but NOT overdue is
# excluded -- proves the filter's clauses are both actually enforced, not
# just "any Task of mine."
post_redirect "$BASE_URL/mch_vl_task" "fld_vlt_title=Vera+Future+$$&fld_vlt_assignee=$VERA_ID&fld_vlt_due=2099-01-01" "$VERA" >/dev/null
! body_contains "$BASE_URL/mch_vl_task" "Vera Future $$" "$VERA"
check T75 "CAP-V05,CAP-V09" "a not-yet-due Task of mine is excluded from My Overdue Tasks" $?

# T76 -- CAP-V08: free-text search (?q=) matches a substring of a visible
# column, case-insensitively; a differently-named record is excluded from
# the same result.
post_redirect "$BASE_URL/mch_al_project" "fld_alp_name=SearchTarget$$" "$PAM" >/dev/null
post_redirect "$BASE_URL/mch_al_project" "fld_alp_name=Unrelated$$" "$PAM" >/dev/null
SEARCH_BODY=$(get_body "$BASE_URL/mch_al_project?q=searchtarget$$" "$PAM")
echo "$SEARCH_BODY" | grep -q "SearchTarget$$" && ! echo "$SEARCH_BODY" | grep -q "Unrelated$$"
check T76 "CAP-V08" "?q= search matches one record's column, excludes an unrelated one" $?

# T77 -- CAP-V07: calendar view groups Action Lab Tasks by their own
# date_field (Follow Up Date).
CODE=$(curl -s -o /dev/null -w '%{http_code}' -b "$PAM" "$BASE_URL/mch_al_task/calendar")
[ "$CODE" = "200" ] && body_contains "$BASE_URL/mch_al_task/calendar" "Task Calendar" "$PAM"
check T77 "CAP-V07" "calendar view renders records grouped by date_field (got $CODE)" $?

# T78 -- CAP-V07: timeline view is the same grouping, read chronologically.
CODE=$(curl -s -o /dev/null -w '%{http_code}' -b "$PAM" "$BASE_URL/mch_al_task/timeline")
[ "$CODE" = "200" ]
check T78 "CAP-V07" "timeline view renders (got $CODE)" $?

# T79 -- CAP-V10: a dashboard composes sections from TWO different
# Machines (Task, Project) in one View.
DASH_BODY=$(get_body "$BASE_URL/mch_al_project/dashboard" "$PAM")
echo "$DASH_BODY" | grep -q "Tasks by Stage" && echo "$DASH_BODY" | grep -q "Projects"
check T79 "CAP-V10" "dashboard composes sections from more than one Machine" $?

# T80 -- CAP-V13: a report View sums Journal Entry Line's Debit/Credit,
# grouped by Account -- a per-run-unique Account name so the SUM is exactly
# this entry's own amount, not an accumulation across repeated suite runs.
JE_RPT_DATA="fld_je_date=2026-07-12&fld_je_memo=Conformance+T80&child_0_fld_jel_account=ConfAcct$$&child_0_fld_jel_debit=42"
post_redirect "$BASE_URL/mch_journal_entry" "$JE_RPT_DATA" "$IVY" >/dev/null
body_contains "$BASE_URL/mch_journal_entry_line/report" "ConfAcct$$" "$IVY" && \
    body_contains "$BASE_URL/mch_journal_entry_line/report" "42.00" "$IVY"
check T80 "CAP-V13" "report groups records by group_field, sums each declared sum_field" $?

# T81 -- CAP-V12: a multi-step wizard's final POST carries every earlier
# step's value forward (as hidden inputs) and creates one record from all
# of them combined, not just the last step's own fields.
WIZ_CSRF=$(csrf_for "$VERA")
STEP2_BODY=$(curl -s -b "$VERA" -X POST "$BASE_URL/mch_vl_onboarding" \
    -d "csrf_token=$WIZ_CSRF&wizard_step=0&fld_vlo_name=Wendy+Wizard+$$&fld_vlo_email=wendy$$@example.com")
echo "$STEP2_BODY" | grep -q 'Step 2 of 2' && echo "$STEP2_BODY" | grep -q "wendy$$@example.com"
check T81 "CAP-V12" "step 1 submission advances to step 2, carrying step 1's values forward" $?
ONB_URL=$(curl -s -o /dev/null -w '%{redirect_url}' -b "$VERA" -X POST "$BASE_URL/mch_vl_onboarding" \
    -d "csrf_token=$WIZ_CSRF&wizard_step=1&fld_vlo_name=Wendy+Wizard+$$&fld_vlo_email=wendy$$@example.com&fld_vlo_team=Engineering&fld_vlo_start=2026-08-01")
body_contains "$ONB_URL" "Wendy Wizard $$" "$VERA" && body_contains "$ONB_URL" "Engineering" "$VERA"
check T82 "CAP-V12" "final step creates one record combining every step's fields" $?

# T83 -- CAP-V14: moving a record up swaps its position with its immediate
# predecessor in a manual-order list. Sorted sort_order ASC (oldest first),
# so these two freshly-created items are always the LAST two -- on the
# LAST page once this suite's own repeated runs push mch_vl_backlog past
# CAP-R05's 25-row page size, not page 1.
B1_URL=$(post_redirect "$BASE_URL/mch_vl_backlog" "fld_vlb_title=Item+A+$$" "$VERA")
sleep 1
B2_URL=$(post_redirect "$BASE_URL/mch_vl_backlog" "fld_vlb_title=Item+B+$$" "$VERA")
B2_ID="${B2_URL##*/}"
LAST_PAGE=$(get_body "$BASE_URL/mch_vl_backlog?page=1" "$VERA" | grep -oE 'Page 1 of [0-9]+' | grep -oE '[0-9]+$')
LAST_PAGE=${LAST_PAGE:-1}
BEFORE=$(get_body "$BASE_URL/mch_vl_backlog?page=$LAST_PAGE" "$VERA" | grep -o "Item [AB] $$" | head -1)
post_status "$BASE_URL/mch_vl_backlog/$B2_ID/move/up" "" "$VERA" >/dev/null
AFTER=$(get_body "$BASE_URL/mch_vl_backlog?page=$LAST_PAGE" "$VERA" | grep -o "Item [AB] $$" | head -1)
[ "$BEFORE" = "Item A $$" ] && [ "$AFTER" = "Item B $$" ]
check T83 "CAP-V14" "moving a record up swaps it with its predecessor (before=$BEFORE, after=$AFTER)" $?

# --- Batch 5: Record Lifecycle (2026-07-12) ---
# seeds/011_record_lifecycle_lab.sql. CAP-R04 (audit log) shipped earlier,
# not part of this batch.

# T84 -- CAP-R03: an archived record disappears from the live list but is
# reachable via ?archived=1.
RLT_URL=$(post_redirect "$BASE_URL/mch_rl_ticket" "fld_rlt_title=Archive+Target+$$" "$REX")
RLT_ID="${RLT_URL##*/}"
post_status "$BASE_URL/mch_rl_ticket/$RLT_ID/archive" "" "$REX" >/dev/null
! body_contains "$BASE_URL/mch_rl_ticket" "Archive Target $$" "$REX" && \
    body_contains "$BASE_URL/mch_rl_ticket?archived=1" "Archive Target $$" "$REX"
check T84 "CAP-R03" "an archived record leaves the live list and appears under ?archived=1" $?

# T85 -- CAP-R03: restoring an archived record returns it to the live list.
post_status "$BASE_URL/mch_rl_ticket/$RLT_ID/restore" "" "$REX" >/dev/null
body_contains "$BASE_URL/mch_rl_ticket" "Archive Target $$" "$REX"
check T85 "CAP-R03" "restoring an archived record returns it to the live list" $?

# T86 -- CAP-R05: pagination splits a list into pages of 25 -- creates 26
# fresh records (guaranteeing a second page exists even on a database this
# suite has never touched before) and checks the page indicator, not an
# absolute row count that would drift as this suite re-runs.
for i in $(seq 1 26); do
    post_redirect "$BASE_URL/mch_rl_ticket" "fld_rlt_title=Page+Item+$$-$i" "$REX" >/dev/null
done
PAGE1=$(get_body "$BASE_URL/mch_rl_ticket?page=1" "$REX")
TOTAL_PAGES=$(echo "$PAGE1" | grep -oE 'Page 1 of [0-9]+' | grep -oE '[0-9]+$')
[ -n "$TOTAL_PAGES" ] && [ "$TOTAL_PAGES" -ge 2 ]
check T86 "CAP-R05" "a list with more than 25 records paginates into multiple pages (got $TOTAL_PAGES pages)" $?

# T87 -- CAP-R06: CSV export contains a real record's own field value.
RLD_URL=$(post_redirect "$BASE_URL/mch_rl_document" "fld_rld_title=Export+Me+$$&fld_rld_amount=42" "$REX")
body_contains "$BASE_URL/mch_rl_document/export.csv" "Export Me $$,42" "$REX"
check T87 "CAP-R06" "CSV export includes a real record's field values" $?

# T88 -- CAP-R06: CSV import creates a valid row and reports a specific
# per-row failure for an invalid one, in the same file -- one bad row
# doesn't block the rest.
RLD_CSRF=$(csrf_for "$REX")
printf 'fld_rld_title,fld_rld_amount\nImport+OK+%s,7\n,9\n' "$$" > "$SESSION_DIR/import_$$.csv"
IMPORT_BODY=$(curl -s -b "$REX" -F "csrf_token=$RLD_CSRF" -F "file=@$SESSION_DIR/import_$$.csv" \
    "$BASE_URL/mch_rl_document/import")
echo "$IMPORT_BODY" | grep -q "1 created, 1 failed" && echo "$IMPORT_BODY" | grep -q "Title is required"
check T88 "CAP-R06" "CSV import creates the valid row and reports the invalid row's own violation" $?

# T89 -- CAP-R07: a Draft (not yet immutable) Ledger Entry can still be
# edited normally.
RLLE_URL=$(post_redirect "$BASE_URL/mch_rl_ledger_entry" "fld_rlle_memo=Rent+$$" "$REX")
CODE=$(post_status "$RLLE_URL" "fld_rlle_memo=Rent+Updated+$$" "$REX")
[ "$CODE" = "303" ]
check T89 "CAP-R07" "a Draft (not yet immutable) record can still be edited (got $CODE)" $?

# T90 -- CAP-R07 negative: once Posted, the SAME record rejects both a
# direct Update and an Archive -- frozen against every mutation path, not
# just events.
RLLE_ID="${RLLE_URL##*/}"
post_status "$BASE_URL/mch_rl_ledger_entry/$RLLE_ID/events/evt_rlle_post" "" "$REX" >/dev/null
UPDATE_CODE=$(post_status "$RLLE_URL" "fld_rlle_memo=Sneaky+$$" "$REX")
ARCHIVE_CODE=$(post_status "$BASE_URL/mch_rl_ledger_entry/$RLLE_ID/archive" "" "$REX")
[ "$UPDATE_CODE" = "403" ] && [ "$ARCHIVE_CODE" = "403" ]
check T90 "CAP-R07" "a Posted (immutable) record rejects both Update and Archive (got $UPDATE_CODE/$ARCHIVE_CODE)" $?

# T91 -- CAP-R08: a record created directly into its declared scratch state
# (Cart) is NOT rejected for missing/invalid fields that would otherwise
# violate this Machine's own Constraints.
CART_CODE=$(post_status "$BASE_URL/mch_rl_cart" "" "$REX")
[ "$CART_CODE" = "303" ]
check T91 "CAP-R08" "a record created into scratch state skips normally-blocking Constraints (got $CART_CODE)" $?

# T92 -- CAP-R08: the SAME incomplete Cart rejects Checkout (the commit
# point re-enforces Constraints, CAP-C09's existing mechanism) -- then
# succeeds once fixed while still in scratch state.
CART_URL=$(post_redirect "$BASE_URL/mch_rl_cart" "" "$REX")
CART_ID="${CART_URL##*/}"
CHECKOUT_BEFORE=$(post_status "$BASE_URL/mch_rl_cart/$CART_ID/events/evt_rlc_checkout" "" "$REX")
post_status "$CART_URL" "fld_rlc_item=Widget+$$&fld_rlc_quantity=3" "$REX" >/dev/null
CHECKOUT_AFTER=$(post_status "$BASE_URL/mch_rl_cart/$CART_ID/events/evt_rlc_checkout" "" "$REX")
[ "$CHECKOUT_BEFORE" = "400" ] && [ "$CHECKOUT_AFTER" = "303" ]
check T92 "CAP-R08" "checkout on an incomplete Cart is rejected, succeeds once fixed (got $CHECKOUT_BEFORE/$CHECKOUT_AFTER)" $?

# --- Batch 6: Permissions (2026-07-12) ---
# seeds/012_permissions_lab.sql. CAP-P05 (deny-by-default CRUD) shipped
# earlier, not part of this batch.

NORA_ID=$(user_option_id "$BASE_URL/mch_pl_expense/new" "$NORA" "Nora")
OMAR_ID=$(user_option_id "$BASE_URL/mch_pl_expense/new" "$NORA" "Omar")
HANA_ID=$(user_option_id "$BASE_URL/mch_pl_expense/new" "$NORA" "Hana")

# T93 -- CAP-P03 negative: Nora submits an Expense, assigns herself as its
# Approver (passes CAP-P02's owner_field check), then tries to approve it --
# blocked by separation of duties even though she IS the assigned owner.
EXP1_DATA="fld_ple_title=SoD+Self+$$&fld_ple_submitted_by=$NORA_ID"
EXP1_URL=$(post_redirect "$BASE_URL/mch_pl_expense" "$EXP1_DATA" "$NORA")
EXP1_ID="${EXP1_URL##*/}"
EA1_URL=$(post_redirect "$BASE_URL/mch_pl_expense_approval" "fld_plea_expense=$EXP1_ID&fld_plea_approver=$NORA_ID" "$NORA")
EA1_ID="${EA1_URL##*/}"
CODE=$(post_status "$BASE_URL/mch_pl_expense_approval/$EA1_ID/events/evt_plea_approve" "" "$NORA")
[ "$CODE" = "400" ]
check T93 "CAP-P03" "the submitter of a record cannot also decide its own Approval, even as the assigned owner (got $CODE)" $?

# T94 -- CAP-P03 positive: Omar submits a DIFFERENT Expense; Nora (who did
# NOT submit it) approves normally -- proves the block is about self-
# dealing specifically, not a blanket failure.
EXP2_DATA="fld_ple_title=SoD+Other+$$&fld_ple_submitted_by=$OMAR_ID"
EXP2_URL=$(post_redirect "$BASE_URL/mch_pl_expense" "$EXP2_DATA" "$OMAR")
EXP2_ID="${EXP2_URL##*/}"
EA2_URL=$(post_redirect "$BASE_URL/mch_pl_expense_approval" "fld_plea_expense=$EXP2_ID&fld_plea_approver=$NORA_ID" "$NORA")
EA2_ID="${EA2_URL##*/}"
CODE=$(post_status "$BASE_URL/mch_pl_expense_approval/$EA2_ID/events/evt_plea_approve" "" "$NORA")
[ "$CODE" = "303" ]
check T94 "CAP-P03" "approving a DIFFERENT person's submission succeeds normally (got $CODE)" $?

# T95 -- CAP-P04: Omar (assigned Approver, not the submitter) delegates to
# Hana -- the Detail page's inline picker submits a fresh value at trigger
# time (input:<field>), and Delegated By stamps who handed it off.
EXP3_DATA="fld_ple_title=Delegation+$$&fld_ple_submitted_by=$NORA_ID"
EXP3_URL=$(post_redirect "$BASE_URL/mch_pl_expense" "$EXP3_DATA" "$NORA")
EXP3_ID="${EXP3_URL##*/}"
EA3_URL=$(post_redirect "$BASE_URL/mch_pl_expense_approval" "fld_plea_expense=$EXP3_ID&fld_plea_approver=$OMAR_ID" "$NORA")
EA3_ID="${EA3_URL##*/}"
OMAR_CSRF=$(csrf_for "$OMAR")
curl -s -b "$OMAR" -X POST "$BASE_URL/mch_pl_expense_approval/$EA3_ID/events/evt_plea_delegate" \
    -d "csrf_token=$OMAR_CSRF&fld_plea_approver=$HANA_ID" >/dev/null
body_contains "$EA3_URL" "Hana" "$OMAR" && body_contains "$EA3_URL" "Omar" "$OMAR"
check T95 "CAP-P04" "delegating reassigns the Approver and stamps Delegated By with who handed it off" $?

# T96 -- CAP-P06: a field a role's Permission hides never reaches List or
# Detail -- HR sees Salary, Staff (same Machine, real read access) does not.
EMP_DATA="fld_ple2_name=Priya+$$&fld_ple2_salary=95000"
EMP_URL=$(post_redirect "$BASE_URL/mch_pl_employee" "$EMP_DATA" "$HANA")
body_contains "$EMP_URL" "95000" "$HANA" && \
    ! body_contains "$EMP_URL" "95000" "$IRIS" && \
    ! body_contains "$BASE_URL/mch_pl_employee" "95000" "$IRIS"
check T96 "CAP-P06" "a hidden field is absent from Staff's List and Detail but visible to HR" $?

# T97 -- CAP-P07: an anonymous request (no session cookie at all) reaches a
# Machine whose Permissions grant role Visitor read access -- List and
# Detail both, GET only.
POST_DATA="fld_plp_title=Visitor+Post+$$&fld_plp_body=hello&fld_plp_status=Published"
POST_URL=$(post_redirect "$BASE_URL/mch_pl_post" "$POST_DATA" "$NORA")
ANON_LIST_CODE=$(curl -s -o /dev/null -w '%{http_code}' "$BASE_URL/mch_pl_post")
ANON_DETAIL_CODE=$(curl -s -o /dev/null -w '%{http_code}' "$POST_URL")
[ "$ANON_LIST_CODE" = "200" ] && [ "$ANON_DETAIL_CODE" = "200" ] && \
    curl -s "$POST_URL" | grep -q "Visitor Post $$"
check T97 "CAP-P07" "an anonymous request reads a Machine whose Permissions grant Visitor read access (list=$ANON_LIST_CODE, detail=$ANON_DETAIL_CODE)" $?

# T98 -- CAP-P07 negative: the SAME anonymous request is denied a Machine
# with no Visitor grant (redirected to /login, not silently allowed), and
# denied a POST even to the public Machine (read-only, not a write door).
ANON_OTHER_CODE=$(curl -s -o /dev/null -w '%{http_code}' "$BASE_URL/mch_pl_employee")
ANON_POST_CODE=$(curl -s -o /dev/null -w '%{http_code}' -X POST "$BASE_URL/mch_pl_post" -d "fld_plp_title=Hacked")
[ "$ANON_OTHER_CODE" = "303" ] && [ "$ANON_POST_CODE" = "401" ]
check T98 "CAP-P07" "anonymous access is still denied for a Machine with no Visitor grant, and for any POST (other=$ANON_OTHER_CODE, post=$ANON_POST_CODE)" $?

# --- Batch 7: Event Sources (2026-07-12) ---
# seeds/013_event_sources_lab.sql. T99-T101 wait for a real background
# scheduler tick (cmd/server/main.go's runScheduler, once a minute) rather
# than a manual stand-in -- CAP-E05's own T38 used one of those for this
# exact gap before the real thing existed; this batch builds the real
# thing, so it gets proven against the real thing.

TOMORROW=$(date -d "+1 day" +%Y-%m-%d 2>/dev/null || date -v+1d +%Y-%m-%d)
FAR_DATE=$(date -d "+5 days" +%Y-%m-%d 2>/dev/null || date -v+5d +%Y-%m-%d)

# Set up all three scheduler-dependent records BEFORE the one shared wait,
# so this batch pays a single ~65s tick delay, not three.
REM_URL=$(post_redirect "$BASE_URL/mch_es_reminder" "fld_esr_title=Reminder+$$" "$SAM")
DUE_SOON_URL=$(post_redirect "$BASE_URL/mch_es_task" "fld_est_title=Due+Soon+$$&fld_est_due=$TOMORROW" "$SAM")
DUE_FAR_URL=$(post_redirect "$BASE_URL/mch_es_task" "fld_est_title=Due+Far+$$&fld_est_due=$FAR_DATE" "$SAM")

echo "(waiting ~65s for the real background scheduler tick -- CAP-E02/E03)"
sleep 65

# T99 -- CAP-E02: a time-driven Event ("00:00", satisfied at any hour)
# fires on the very next scheduler tick after the record exists.
body_contains "$REM_URL" ">Yes<" "$SAM"
check T99 "CAP-E02" "a time-driven Event fires on its own, no user action, once the scheduled time is reached" $?

# T100 -- CAP-E03 positive: a Task due TOMORROW hits its own "Due Date - 1
# Day" trigger point TODAY.
body_contains "$DUE_SOON_URL" ">Yes<" "$SAM"
check T100 "CAP-E03" "a date-driven Event fires when today equals the record's own date field plus the declared offset" $?

# T101 -- CAP-E03 negative: a Task due in 5 days is nowhere near its own
# trigger point yet -- proves the check is per-record, not "any Task with
# this Event fires."
! body_contains "$DUE_FAR_URL" ">Yes<" "$SAM"
check T101 "CAP-E03" "a date-driven Event does NOT fire for a record whose own date field hasn't reached the offset yet" $?

# T102 -- CAP-E04: an external POST with the Machine's own webhook_secret
# triggers an event with no session and no CSRF token -- and the payload's
# own field (via InputFields/"input:<field>") is stamped onto the record.
PAY_URL=$(post_redirect "$BASE_URL/mch_es_payment" "fld_esp_amount=250" "$SAM")
PAY_ID="${PAY_URL##*/}"
WEBHOOK_CODE=$(curl -s -o /dev/null -w '%{http_code}' -X POST \
    -H "X-Webhook-Secret: demo-webhook-secret-2026" \
    -d "fld_esp_reference=txn_$$" \
    "$ORIGIN/webhooks/mch_es_payment/$PAY_ID/evt_esp_confirm")
[ "$WEBHOOK_CODE" = "200" ] && body_contains "$PAY_URL" "Paid" "$SAM" && body_contains "$PAY_URL" "txn_$$" "$SAM"
check T102 "CAP-E04" "a webhook with the correct secret triggers an event with no session, stamping its own payload field (got $WEBHOOK_CODE)" $?

# T103 -- CAP-E04 negative: the wrong secret is rejected, and the record is
# left untouched -- the secret is a real credential, not decorative.
PAY2_URL=$(post_redirect "$BASE_URL/mch_es_payment" "fld_esp_amount=99" "$SAM")
PAY2_ID="${PAY2_URL##*/}"
WRONG_CODE=$(curl -s -o /dev/null -w '%{http_code}' -X POST \
    -H "X-Webhook-Secret: wrong-secret" \
    -d "fld_esp_reference=txn_bad_$$" \
    "$ORIGIN/webhooks/mch_es_payment/$PAY2_ID/evt_esp_confirm")
[ "$WRONG_CODE" = "401" ] && ! body_contains "$PAY2_URL" "Paid" "$SAM"
check T103 "CAP-E04" "a webhook with the wrong secret is rejected, record left untouched (got $WRONG_CODE)" $?

# --- Batch 8: Cross-Machine Integration (2026-07-12) ---
# seeds/014_integration_lab.sql. CAP-I04 (correlation trace) shipped
# earlier, not part of this batch.

# T104 -- CAP-I01: an Order Placed event fires a Subscription declared on
# an ENTIRELY DIFFERENT Machine (Audit Log) -- Order's own metadata never
# names Audit Log at all.
ORD1_DATA="fld_into_customer=SmallCust$$&fld_into_total=50"
ORD1_URL=$(post_redirect "$BASE_URL/mch_int_order" "$ORD1_DATA" "$THEO")
ORD1_ID="${ORD1_URL##*/}"
post_status "$BASE_URL/mch_int_order/$ORD1_ID/events/evt_into_placed" "" "$THEO" >/dev/null
# CAP-W06 (2026-08-23): Subscription fan-out is now dispatcher-mediated
# (enqueued atomically with the publisher's own write, performed off the
# request path shortly after) rather than synchronous within this POST --
# give the dispatcher's ~2s tick time to run before asserting on delivery,
# the same "sleep past the known tick interval" style T99/T100 already use
# for the scheduler's own async nature. T105 is a negative case that must
# wait too, or it would trivially pass for the wrong reason (dispatcher
# hasn't run yet, not "Contract correctly skipped this Subscription").
sleep 3
body_contains "$BASE_URL/mch_int_audit_log" "SmallCust$$" "$THEO"
check T104 "CAP-I01" "a cross-machine Subscription creates a record on a Machine the publisher's own metadata never names" $?

# T105 -- CAP-I03 negative: the SAME Order also has a Points Ledger
# Subscription, but ITS OWN Contract (total >= 100) isn't met by this
# order -- skipped, not created.
! body_contains "$BASE_URL/mch_int_points" "SmallCust$$" "$THEO"
check T105 "CAP-I03" "a Subscription's Contract violation skips that Subscription's own action (negative case)" $?

# T106 -- CAP-I03 positive: a large order (total >= 100) satisfies the same
# Contract, so the Points Ledger Subscription fires.
ORD2_DATA="fld_into_customer=BigCust$$&fld_into_total=150"
ORD2_URL=$(post_redirect "$BASE_URL/mch_int_order" "$ORD2_DATA" "$THEO")
ORD2_ID="${ORD2_URL##*/}"
post_status "$BASE_URL/mch_int_order/$ORD2_ID/events/evt_into_placed" "" "$THEO" >/dev/null
sleep 3
body_contains "$BASE_URL/mch_int_points" "BigCust$$" "$THEO"
check T106 "CAP-I03" "a Subscription's Contract being satisfied lets its own action fire" $?

# T107 -- CAP-I05: a SECOND, unrelated publisher Event (Referral Completed,
# a different Machine entirely from Order) ALSO contributes to the SAME
# Points Ledger -- one shared KPI Machine, fed by two independent sources,
# neither aware of the other.
REF_DATA="fld_intr_referrer=Referrer$$"
REF_URL=$(post_redirect "$BASE_URL/mch_int_referral" "$REF_DATA" "$THEO")
REF_ID="${REF_URL##*/}"
post_status "$BASE_URL/mch_int_referral/$REF_ID/events/evt_intr_completed" "" "$THEO" >/dev/null
sleep 3
body_contains "$BASE_URL/mch_int_points" "Referrer$$" "$THEO" && body_contains "$BASE_URL/mch_int_points" "BigCust$$" "$THEO"
check T107 "CAP-I05" "the same shared Machine accumulates contributions from two different, unrelated publisher Events" $?

# T108 -- CAP-I02: a deprecated Event still functions (backward compat),
# and its own Detail page shows a Deprecated indicator.
CODE=$(post_status "$BASE_URL/mch_int_order/$ORD1_ID/events/evt_into_legacy_notify" "" "$THEO")
[ "$CODE" = "303" ] && body_contains "$ORD1_URL" "Deprecated" "$THEO"
check T108 "CAP-I02" "a deprecated Event still works and shows a Deprecated indicator (got $CODE)" $?

# --- Batch 9: Workspace Services (2026-07-12) ---
# seeds/015_workspace_services_lab.sql. CAP-O01/CAP-O03 shipped earlier, not
# part of this batch.

# T109 -- CAP-O02: a `reference` field on a Machine in a DIFFERENT
# Application can target a master-data Machine's record -- cross-app
# referenceability, proven deliberately (already implied by CAP-F13 alone,
# but never exercised across an Application boundary until now).
WSXE1_URL=$(post_redirect "$BASE_URL/mch_wsx_employee" "fld_wsxe_name=Priya+$$" "$YARA")
WSXE1_ID="${WSXE1_URL##*/}"
WSXP1_URL=$(post_redirect "$BASE_URL/mch_wsx_project" "fld_wsxp_title=Cross+App+Project+$$&fld_wsxp_lead=$WSXE1_ID" "$YARA")
body_contains "$WSXP1_URL" "Priya $$" "$YARA"
check T109 "CAP-O02" "a reference field on a Machine in a different Application targets a master-data record" $?

# T110 -- CAP-O02 negative: archiving a master-data record still referenced
# by another Machine's record (any Application) is blocked, not silently
# allowed to dangle the reference.
YARA_CSRF=$(csrf_for "$YARA")
ARCHIVE_BODY=$(curl -s -b "$YARA" -X POST "$BASE_URL/mch_wsx_employee/$WSXE1_ID/archive" -d "csrf_token=$YARA_CSRF")
ARCHIVE_CODE=$(post_status "$BASE_URL/mch_wsx_employee/$WSXE1_ID/archive" "" "$YARA")
[ "$ARCHIVE_CODE" = "409" ] && echo "$ARCHIVE_BODY" | grep -q "still referenced by Project (via Lead)"
check T110 "CAP-O02" "archiving a master-data record still referenced elsewhere is rejected (got $ARCHIVE_CODE)" $?

# T111 -- CAP-O02 positive: an UNREFERENCED master-data record archives
# normally -- the block is specific to standing references, not a blanket
# ban on archiving master data at all.
WSXE2_URL=$(post_redirect "$BASE_URL/mch_wsx_employee" "fld_wsxe_name=Unreferenced+$$" "$YARA")
WSXE2_ID="${WSXE2_URL##*/}"
ARCHIVE2_CODE=$(post_status "$BASE_URL/mch_wsx_employee/$WSXE2_ID/archive" "" "$YARA")
[ "$ARCHIVE2_CODE" = "303" ] && ! body_contains "$BASE_URL/mch_wsx_employee" "Unreferenced $$" "$YARA"
check T111 "CAP-O02" "an unreferenced master-data record archives normally (got $ARCHIVE2_CODE)" $?

# T112 -- CAP-O04: workspace-wide search finds a match on a Machine the
# searching role can read, scanning across every Application in the
# workspace, not just one Machine at a time.
EMP_PROBE_URL=$(post_redirect "$BASE_URL/mch_pl_employee" "fld_ple2_name=SearchProbe$$&fld_ple2_salary=1000" "$HANA")
body_contains "$BASE_URL/search?q=SearchProbe$$" ">SearchProbe$$</a>" "$HANA"
check T112 "CAP-O04" "workspace-wide search finds a match on a Machine the searcher can read" $?

# T113 -- CAP-O04 negative: the SAME record is invisible to a search by a
# role with no Permission on that Machine at all -- results are
# permission-trimmed per Machine, not a raw text index over everything.
# (Checking for the real result anchor, not just the query string, since
# the search box's own input echoes the query back regardless of results.)
body_contains "$BASE_URL/search?q=SearchProbe$$" "No matches for" "$YARA" && \
    ! body_contains "$BASE_URL/search?q=SearchProbe$$" ">SearchProbe$$</a>" "$YARA"
check T113 "CAP-O04" "search results are permission-trimmed -- a role with no access to the Machine finds nothing" $?

# T114 -- CAP-O05: the notification inbox's per-user delivery preference
# toggles between "immediate" (flat list) and "digest" (grouped by day) --
# same underlying notifications either way, just how they're presented.
DIGEST_CODE=$(post_status "$BASE_URL/notifications/preference" "preference=digest" "$PAM")
body_contains "$BASE_URL/notifications" "Switch to immediate" "$PAM" && \
    body_contains "$BASE_URL/notifications" "Task: Complete" "$PAM"
check T114 "CAP-O05" "switching to digest preference groups the same notifications by day (got $DIGEST_CODE)" $?
# Restore PAM's preference so a re-run of this suite starts from the same
# state every time.
post_status "$BASE_URL/notifications/preference" "preference=immediate" "$PAM" >/dev/null

# T115 -- CAP-O06: "N Business Days" date arithmetic skips weekends,
# reimplemented independently in bash (same T66 precedent for plain-day
# arithmetic). The holiday-specific half of this rule (workspace_holidays)
# is proven by direct DB insert + restart only, documented as a manual,
# DB-inspection exception in conformance/README.md alongside T19/T42/T43/T52
# -- a seeded static holiday date would go stale relative to whenever this
# suite actually runs, so it can't be an automated assertion here.
EXPECTED_BUSDAY=$(add_business_days 5)
WSXT_URL=$(post_redirect "$BASE_URL/mch_wsx_task" "fld_wsxt_title=Business+Day+Test+$$" "$YARA")
WSXT_ID="${WSXT_URL##*/}"
post_status "$BASE_URL/mch_wsx_task/$WSXT_ID/events/evt_wsxt_schedule" "" "$YARA" >/dev/null
body_contains "$WSXT_URL" "$EXPECTED_BUSDAY" "$YARA"
check T115 "CAP-O06" "\"N Business Days\" date arithmetic skips weekends, matching an independent bash reimplementation (expected $EXPECTED_BUSDAY)" $?

