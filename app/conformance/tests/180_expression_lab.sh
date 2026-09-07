#!/usr/bin/env bash
# CAP-C13 (expression operator, internal/expr, CEL) + CAP-F14 completion
# (computed Field via a general expression). seeds/041_expression_lab.sql
# provides mch_expr_order. Sourced by run.sh after lib.sh.

EO_URL="$BASE_URL/mch_expr_order"
SALES=$(session_for expr.sales@example.com password)
EMANAGER=$(session_for expr.manager@example.com password)

# T236 -- CAP-F14 completion: the computed Total field is a real formula
# (quantity * unit_price * (1 - discount%/100)), not just one multiply.
# 10 * 100 * (1 - 10/100) = 900.
TOTAL_URL=$(post_redirect "$EO_URL" "fld_eo_customer=TotalCustomer&fld_eo_quantity=10&fld_eo_unit_price=100&fld_eo_discount_pct=10" "$SALES")
body_contains "$TOTAL_URL" "900" "$SALES"
check T236 "CAP-F14" "computed field's general CEL expression combines quantity/price/discount, not just one multiply (900)" $?

# T237 -- CAP-C13 Constraint reading `old` vs `record`: a fresh Draft order
# triggering Close DIRECTLY (no event enforces "must be Approved" on its
# own) is caught by the Machine-level Constraint instead, since it compares
# old.status == "Draft" against record.status == "Closed" post-Simulate.
BLOCKED_URL=$(post_redirect "$EO_URL" "fld_eo_customer=BlockedCustomer&fld_eo_quantity=10&fld_eo_unit_price=100" "$SALES")
BLOCKED_ID="${BLOCKED_URL##*/}"
CLOSE_CODE=$(post_status "$BLOCKED_URL/events/evt_eo_close" "" "$EMANAGER")
[ "$CLOSE_CODE" = "400" ] && post_body_contains "$BLOCKED_URL/events/evt_eo_close" "" "must be Approved first" "$EMANAGER"
check T237 "CAP-C13" "a Constraint comparing old vs record blocks a direct Draft->Closed transition (got $CLOSE_CODE)" $?

# T238 -- CAP-C13 Event condition with real arithmetic (quantity * price >
# 100), something the plain field/operator/value grammar could never
# express in one clause. Negative: a low-value order is rejected. Positive:
# a qualifying order approves, and THEN closes normally (old.status is now
# "Approved", not "Draft" -- the Constraint above does not fire).
LOWVALUE_URL=$(post_redirect "$EO_URL" "fld_eo_customer=LowValueCustomer&fld_eo_quantity=1&fld_eo_unit_price=50" "$SALES")
LOWVALUE_APPROVE_CODE=$(post_status "$LOWVALUE_URL/events/evt_eo_approve" "" "$EMANAGER")

CLOSABLE_URL=$(post_redirect "$EO_URL" "fld_eo_customer=ClosableCustomer&fld_eo_quantity=10&fld_eo_unit_price=100" "$SALES")
CLOSABLE_APPROVE_CODE=$(post_status "$CLOSABLE_URL/events/evt_eo_approve" "" "$EMANAGER")
CLOSABLE_CLOSE_CODE=$(post_status "$CLOSABLE_URL/events/evt_eo_close" "" "$EMANAGER")

[ "$LOWVALUE_APPROVE_CODE" = "400" ] && [ "$CLOSABLE_APPROVE_CODE" = "303" ] && [ "$CLOSABLE_CLOSE_CODE" = "303" ]
check T238 "CAP-C13" "an Event condition's own arithmetic (qty*price>100) rejects a low-value order (got $LOWVALUE_APPROVE_CODE) and allows a qualifying one through Approve->Close (got $CLOSABLE_APPROVE_CODE/$CLOSABLE_CLOSE_CODE)" $?

# T239 -- CAP-C13 flowing into the Machine's own list View filter
# (CAP-V05/V09) via the same FilterCondition.Expression field -- no second
# filtering mechanism. The list's own declared filter (qty*price>500) shows
# BigCustomer, not SmallCustomer.
post_redirect "$EO_URL" "fld_eo_customer=BigCustomer&fld_eo_quantity=100&fld_eo_unit_price=100" "$SALES" >/dev/null
post_redirect "$EO_URL" "fld_eo_customer=SmallCustomer&fld_eo_quantity=1&fld_eo_unit_price=1" "$SALES" >/dev/null
BIG_BODY=$(curl -s -b "$SALES" "$EO_URL")
echo "$BIG_BODY" | grep -q "BigCustomer" && ! echo "$BIG_BODY" | grep -q "SmallCustomer"
check T239 "CAP-C13" "the list View's own \"expression\" filter (qty*price>500) includes BigCustomer, excludes SmallCustomer" $?
