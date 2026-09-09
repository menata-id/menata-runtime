#!/usr/bin/env bash
# CAP-V02 Tier 2 (card-per-record list display) and CAP-V09 Tier 2
# (computed-SLA-bucket list filter) -- both admitted 2026-09-09
# (capability-registry.md, roadmap.md item 25.3), realized by
# seeds/048_v02t2_v09t2_realization.sql. Sourced by run.sh after lib.sh --
# reuses ALICE/ALICE_ID (010's own resolution, still in scope, every test
# file is `source`d into the same shell).

# T248 -- Approval Document's own "All Documents" list (vw_ad_all) now
# renders `display: "cards"`: a real Document's own title reaches the
# page, but no `<table` element does -- the positive/negative pair proving
# the card branch actually replaced the table, not just added to it.
V02T2_DATA="fld_ad_title=T248+Card+List+$$&fld_ad_document_type=Report&fld_ad_file=v02t2.pdf&fld_ad_submitted_by=$ALICE_ID&fld_ad_approval_mode=Sequential"
post_redirect "$BASE_URL/mch_approval_document" "$V02T2_DATA" "$ALICE" >/dev/null
V02T2_BODY=$(get_body "$BASE_URL/mch_approval_document" "$ALICE")
printf '%s' "$V02T2_BODY" | grep -q "T248 Card List $$" && ! printf '%s' "$V02T2_BODY" | grep -q '<table'
check T248 "CAP-V02" "list view's own display:cards renders a card for a real record, not inside a <table>" $?

# T249 -- SLA Filter Lab (mch_v09t2_ticket): its one list View is already
# the $sla_urgency=overdue filter (seeds/048's own design -- a second List
# View on one Machine is unreachable, Interpreter.DefaultListView only
# ever resolves the first). Positive/negative pair: an overdue ticket
# appears, a far-future one does not -- proving the filter actually
# discriminates, not vacuously empty or vacuously everything.
SLAT2=$(session_for sla.filter.agent@example.com password)
V09T2_OVERDUE=$(post_redirect "$BASE_URL/mch_v09t2_ticket" "fld_v09t2_title=T249+Overdue+$$&fld_v09t2_due=2020-01-01" "$SLAT2")
V09T2_FUTURE=$(post_redirect "$BASE_URL/mch_v09t2_ticket" "fld_v09t2_title=T249+Future+$$&fld_v09t2_due=2099-01-01" "$SLAT2")
V09T2_LIST=$(get_body "$BASE_URL/mch_v09t2_ticket" "$SLAT2")
printf '%s' "$V09T2_LIST" | grep -q "T249 Overdue $$"
check T249a "CAP-V09" "the \$sla_urgency=overdue filter includes a real overdue ticket" $?

! printf '%s' "$V09T2_LIST" | grep -q "T249 Future $$"
check T249b "CAP-V09" "the same filter excludes a ticket due far in the future" $?
