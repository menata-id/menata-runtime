package constraint

import (
	"fmt"
	"strconv"
	"time"

	"menata.id/app/internal/expr"
	"menata.id/app/internal/model"
)

type Engine struct{}

// EvalContext (CAP-C13) carries the extra variables an "expression"
// operator can read beyond the plain field/operator/value triples every
// other operator already works with: Old (the record's pre-mutation
// snapshot -- nil where no real "before" exists, e.g. Create or a list
// filter) and CurrentUser (the acting identity's id -- "" where none is
// resolved at this call site). Every EXISTING caller keeps compiling
// unchanged: Eval/Violations take it as an optional trailing argument, zero
// value when omitted, so an "expression" condition at a call site that
// hasn't been upgraded to pass real Old/CurrentUser values simply sees them
// empty (internal/expr's own graceful-degrade behavior), not a compile
// error or a panic.
type EvalContext struct {
	Old         map[string]any
	CurrentUser string
}

// Violations returns human-readable messages for every constraint the data breaks.
// An empty slice means the data satisfies all constraints. "unique" constraints
// (CAP-C12) and CrossRecord constraints (CAP-C08) are skipped here -- both
// need cross-record access this Engine deliberately doesn't have (same
// Executor/Handler boundary as everywhere else in this codebase); see
// handler.uniquenessViolations/crossRecordViolations for those. A
// CrossRecord constraint's own Expression is a placeholder (never evaluated
// by anything) -- the real check lives entirely in crossRecordViolations.
func (e *Engine) Violations(machine *model.Machine, data map[string]any, evalCtx ...EvalContext) []string {
	var out []string
	for _, c := range machine.Constraints {
		if c.Expression.Operator == "unique" || c.CrossRecord != nil {
			continue
		}
		if c.Condition != nil && !Eval(*c.Condition, data, evalCtx...) {
			continue
		}
		if !Eval(c.Expression, data, evalCtx...) {
			out = append(out, c.Rule)
		}
	}
	return out
}

// resolveCompareValue (CAP-C07) resolves what an expression's Value side
// actually is: ValueField wins when set (compare against another field on
// the same record, e.g. "End Date after Start Date"), else the literal
// Value, "today" resolving the same special-cased way on either side.
func resolveCompareValue(ce model.ConstraintExpression, data map[string]any) string {
	if ce.ValueField != "" {
		v := data[ce.ValueField]
		if v == nil {
			return ""
		}
		return fmt.Sprintf("%v", v)
	}
	return ce.Value
}

func parseMaybeToday(s string) (time.Time, bool) {
	if s == "today" {
		return time.Now().Truncate(24 * time.Hour), true
	}
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return time.Time{}, false
	}
	return t, true
}

// Eval evaluates a single expression against record data. Shared by
// Constraint evaluation, Event guard evaluation (CAP-E06), CAP-A09's action
// "if" guard, and View filters (CAP-V05/V09) — all are the same "does this
// data satisfy this expression" question. evalCtx (CAP-C13) is optional --
// see EvalContext's own doc comment -- and only meaningful for the
// "expression" operator; every other operator below ignores it completely.
func Eval(ce model.ConstraintExpression, data map[string]any, evalCtx ...EvalContext) bool {
	if ce.Operator == "expression" {
		var ctx EvalContext
		if len(evalCtx) > 0 {
			ctx = evalCtx[0]
		}
		return expr.Bool(ce.Expression, expr.Vars{Record: data, Old: ctx.Old, CurrentUser: ctx.CurrentUser})
	}

	raw := data[ce.Field]
	str := fmt.Sprintf("%v", raw)
	if raw == nil {
		str = ""
	}

	switch ce.Operator {
	case "required":
		return str != "" && str != "<nil>"

	case "equals":
		return str == resolveCompareValue(ce, data)

	case "not_equals":
		return str != resolveCompareValue(ce, data)

	case "after", "before", "on_or_after", "on_or_before":
		cmpStr := resolveCompareValue(ce, data)
		if str == "" || cmpStr == "" {
			return false
		}
		t, ok := parseMaybeToday(str)
		if !ok {
			return false
		}
		cmp, ok := parseMaybeToday(cmpStr)
		if !ok {
			return false
		}
		switch ce.Operator {
		case "after":
			return t.After(cmp)
		case "before":
			return t.Before(cmp)
		case "on_or_after":
			return !t.Before(cmp)
		default: // "on_or_before"
			return !t.After(cmp)
		}

	case "in": // CAP-W07 -- membership check, e.g. change_policy's records_in_states
		for _, v := range ce.Values {
			if str == v {
				return true
			}
		}
		return false

	case "greater_than", "less_than", "greater_than_or_equal", "less_than_or_equal":
		cmpStr := resolveCompareValue(ce, data)
		n, err1 := strconv.ParseFloat(str, 64)
		cmpN, err2 := strconv.ParseFloat(cmpStr, 64)
		if err1 != nil || err2 != nil {
			return false
		}
		switch ce.Operator {
		case "greater_than":
			return n > cmpN
		case "less_than":
			return n < cmpN
		case "greater_than_or_equal":
			return n >= cmpN
		default:
			return n <= cmpN
		}
	}
	return true
}
