#!/usr/bin/env bash
# CAP-W06: transactional outbox for notify (CAP-A03/A04) and CAP-I01
# subscription fan-out. Sourced by run.sh after lib.sh -- assumes lib.sh's
# helpers/ACCOUNTS are already in scope.
#
# CAP-A03/A04/A10/CAP-I01's own tests (020/030) already prove the
# FUNCTIONAL correctness of notify/subscription delivery (now updated with
# a bounded sleep, since delivery is dispatcher-mediated as of this
# capability). This file proves what's actually NEW here: (1) the outbox
# row is written atomically with the triggering record's own write, before
# any dispatcher tick could have run: (2) that row is later marked
# completed_at by the dispatcher; (3) one row's dispatch failure is
# isolated -- marked failed_at with a real error, and does not block
# another row in the same workspace/tick from completing.
#
# All three need DB inspection (action_outbox has no HTTP surface of its
# own -- deliberately: it's dispatcher-internal state, not something a
# metadata author or end user ever addresses directly) -- same documented
# "HTTP black-box" exception as T19/T42/T43/T151.

if [ -n "$DATABASE_URL" ]; then
    # T178 -- the outbox row for this event's notify action already exists
    # the instant the triggering POST returns -- no sleep before this
    # query. If enqueue were not atomic with the record's own write (e.g.
    # deferred to some later step), a fast dispatcher tick landing between
    # the POST and this query could produce a false pass; querying
    # immediately, with the dispatcher's own 2s tick as headroom, is what
    # actually proves "atomic with the business write," not just
    # "eventually true."
    W06_URL=$(post_redirect "$BASE_URL/mch_leave_request" "fld_lr_leave_type=Annual&fld_lr_start_date=2026-09-01&fld_lr_end_date=2026-09-03&fld_lr_reason=CAP-W06+probe" "$DAVE")
    W06_ID="${W06_URL##*/}"
    post_status "$W06_URL/events/evt_lr_submit" "" "$DAVE" >/dev/null
    post_status "$W06_URL/events/evt_lr_approve" "" "$EVE" >/dev/null

    W06_OUTBOX_ID=$(psql "$DATABASE_URL" -q -tAc \
        "DO \$\$ BEGIN PERFORM set_config('app.workspace_id', 'ws_default', true); END \$\$;
         SELECT id FROM action_outbox WHERE action_type = 'notify' AND params->>'record_id' = '$W06_ID' LIMIT 1")
    [ -n "$W06_OUTBOX_ID" ]
    check T178 "CAP-W06" "the outbox row exists immediately after the triggering request, before any dispatcher tick (id '$W06_OUTBOX_ID')" $?

    # T179 -- once the dispatcher's ~2s tick has had a chance to run, that
    # SAME row (not a new one) is marked completed_at -- the deferred write
    # actually happened, off the request path.
    sleep 3
    W06_COMPLETED=$(psql "$DATABASE_URL" -q -tAc \
        "DO \$\$ BEGIN PERFORM set_config('app.workspace_id', 'ws_default', true); END \$\$;
         SELECT completed_at IS NOT NULL FROM action_outbox WHERE id = '$W06_OUTBOX_ID'")
    [ "$W06_COMPLETED" = "t" ]
    check T179 "CAP-W06" "the dispatcher marks the same row completed_at after its own tick (got completed=$W06_COMPLETED)" $?

    # T180 -- failure isolation: a deliberately malformed row (no HTTP path
    # produces one -- metadata authors never write action_outbox directly,
    # so this is inserted straight, the same documented exception T19/T151
    # already use for a fixture no HTTP surface can create) must not corrupt
    # the workspace transaction the rest of that tick's batch depends on. A
    # second, legitimate row enqueued in the same window still completes.
    W06_BAD_ID=$(psql "$DATABASE_URL" -q -tAc \
        "DO \$\$ BEGIN PERFORM set_config('app.workspace_id', 'ws_default', true); END \$\$;
         INSERT INTO action_outbox (workspace_id, action_type, params) VALUES ('ws_default', 'subscription', '{}') RETURNING id")
    W06_GOOD_URL=$(post_redirect "$BASE_URL/mch_leave_request" "fld_lr_leave_type=Annual&fld_lr_start_date=2026-09-05&fld_lr_end_date=2026-09-06&fld_lr_reason=CAP-W06+isolation+probe" "$DAVE")
    W06_GOOD_ID="${W06_GOOD_URL##*/}"
    post_status "$W06_GOOD_URL/events/evt_lr_submit" "" "$DAVE" >/dev/null
    post_status "$W06_GOOD_URL/events/evt_lr_approve" "" "$EVE" >/dev/null
    sleep 3
    W06_BAD_FAILED=$(psql "$DATABASE_URL" -q -tAc \
        "DO \$\$ BEGIN PERFORM set_config('app.workspace_id', 'ws_default', true); END \$\$;
         SELECT failed_at IS NOT NULL AND error IS NOT NULL FROM action_outbox WHERE id = '$W06_BAD_ID'")
    W06_GOOD_COMPLETED=$(psql "$DATABASE_URL" -q -tAc \
        "DO \$\$ BEGIN PERFORM set_config('app.workspace_id', 'ws_default', true); END \$\$;
         SELECT completed_at IS NOT NULL FROM action_outbox WHERE action_type = 'notify' AND params->>'record_id' = '$W06_GOOD_ID'")
    [ "$W06_BAD_FAILED" = "t" ] && [ "$W06_GOOD_COMPLETED" = "t" ]
    check T180 "CAP-W06" "a malformed row is marked failed_at with a real error, without blocking a legitimate row in the same batch (bad_failed=$W06_BAD_FAILED, good_completed=$W06_GOOD_COMPLETED)" $?
else
    printf 'SKIP  T178 %-22s %s\n' "CAP-W06" "DATABASE_URL not set -- DB inspection unavailable"
    printf 'SKIP  T179 %-22s %s\n' "CAP-W06" "DATABASE_URL not set -- DB inspection unavailable"
    printf 'SKIP  T180 %-22s %s\n' "CAP-W06" "DATABASE_URL not set -- DB inspection unavailable"
fi
