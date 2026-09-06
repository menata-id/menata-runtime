package handler

import (
	"context"
	"fmt"
	"math"
	"strconv"

	"menata.id/app/internal/constraint"
	"menata.id/app/internal/model"
)

// crossRecordViolations (CAP-C08 -- benchmarks/018-....md, case-portfolio.md
// Case 9) checks every Constraint on machine whose CrossRecord is set.
// constraint.Eval never touches storage (the same boundary
// uniquenessViolations already respects, see validation.go) -- this is a
// separate pass for exactly that reason. Gating ("only check at Post") is
// the Constraint's own ordinary Condition, evaluated the same way
// Engine.Violations does for every other Constraint -- no new mechanism.
//
// recordID is the record being validated -- "" at Create, before a real id
// exists. In practice a fresh record starts at whatever value_list default
// the loader stamps (Draft, per the existing first-value convention), so a
// Condition gated on a later state (Posted) is false and this never reaches
// the recordID=="" branch below -- that branch exists only for the edge
// case of a Create that directly sets the gating state, and fails loud
// rather than silently evaluating against a record that doesn't exist yet.
func (h *Handler) crossRecordViolations(ctx context.Context, machine *model.Machine, data map[string]any, recordID string) ([]string, error) {
	var out []string
	for _, c := range machine.Constraints {
		cr := c.CrossRecord
		if cr == nil {
			continue
		}
		if c.Condition != nil && !constraint.Eval(*c.Condition, data) {
			continue
		}
		if recordID == "" {
			out = append(out, fmt.Sprintf("%s (cannot be checked until the record is saved)", c.Rule))
			continue
		}
		var violated bool
		var err error
		switch cr.Kind {
		case "aggregate":
			violated, err = h.aggregateCrossRecordViolated(ctx, cr, recordID)
		case "reference_field":
			violated, err = h.referenceFieldCrossRecordViolated(ctx, cr, data)
		}
		if err != nil {
			return nil, err
		}
		if violated {
			out = append(out, c.Rule)
		}
	}
	return out, nil
}

// aggregateCrossRecordViolated (CAP-C10) compares SUM(FieldA) against
// SUM(FieldB) -- or the literal Value if FieldB is empty -- across every
// ChildMachine record whose ScopeField points at recordID. Compared
// NUMERICALLY (compareNumeric below), not string-equal like constraint.Eval
// -- debit "100" and credit "100.00" must compare equal, which a plain
// string comparison would get wrong.
func (h *Handler) aggregateCrossRecordViolated(ctx context.Context, cr *model.CrossRecordCheck, recordID string) (bool, error) {
	sumA, err := h.records.SumField(ctx, cr.ChildMachine, cr.FieldA, cr.ScopeField, recordID)
	if err != nil {
		return false, err
	}
	sumB := 0.0
	if cr.FieldB != "" {
		sumB, err = h.records.SumField(ctx, cr.ChildMachine, cr.FieldB, cr.ScopeField, recordID)
		if err != nil {
			return false, err
		}
	} else {
		sumB, _ = strconv.ParseFloat(cr.Value, 64)
	}
	return !compareNumeric(sumA, sumB, cr.Operator), nil
}

// referenceFieldCrossRecordViolated (CAP-C11) looks up the record
// data[cr.ReferenceField] points to and checks its TargetField against
// Value/Operator. A dangling or empty reference is silently not-a-violation
// here -- CAP-F13's own referenceViolations already reports that (it walks
// every `reference` Field on the Machine, including this one), so surfacing
// it again here would just duplicate the message under a different Rule.
func (h *Handler) referenceFieldCrossRecordViolated(ctx context.Context, cr *model.CrossRecordCheck, data map[string]any) (bool, error) {
	refID, _ := data[cr.ReferenceField].(string)
	if refID == "" {
		return false, nil
	}
	target, err := h.records.Get(ctx, refID)
	if err != nil {
		return false, nil
	}
	expr := model.ConstraintExpression{Field: cr.TargetField, Operator: cr.Operator, Value: cr.Value}
	return !constraint.Eval(expr, target.Data), nil
}

func compareNumeric(a, b float64, operator string) bool {
	const eps = 1e-9
	switch operator {
	case "equals":
		return math.Abs(a-b) < eps
	case "not_equals":
		return math.Abs(a-b) >= eps
	case "greater_than":
		return a > b+eps
	case "less_than":
		return a < b-eps
	case "greater_than_or_equal":
		return a >= b-eps
	case "less_than_or_equal":
		return a <= b+eps
	default:
		return true // unsupported operator already rejected at load time (validateReferences)
	}
}
