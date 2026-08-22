package constraint

import (
	"fmt"
	"strconv"
	"time"

	"menata.id/runtime/internal/model"
)

type Engine struct{}

// Violations returns human-readable messages for every constraint the data breaks.
// An empty slice means the data satisfies all constraints. "unique" constraints
// (CAP-C12) are skipped here -- they need cross-record access this Engine
// deliberately doesn't have (same Executor/Handler boundary as everywhere
// else in this codebase); see handler.uniquenessViolations for those.
func (e *Engine) Violations(machine *model.Machine, data map[string]any) []string {
	var out []string
	for _, c := range machine.Constraints {
		if c.Expression.Operator == "unique" {
			continue
		}
		if c.Condition != nil && !Eval(*c.Condition, data) {
			continue
		}
		if !Eval(c.Expression, data) {
			out = append(out, c.Rule)
		}
	}
	return out
}

// resolveCompareValue (CAP-C07) resolves what an expression's Value side
// actually is: ValueField wins when set (compare against another field on
// the same record, e.g. "End Date after Start Date"), else the literal
// Value, "today" resolving the same special-cased way on either side.
func resolveCompareValue(expr model.ConstraintExpression, data map[string]any) string {
	if expr.ValueField != "" {
		v := data[expr.ValueField]
		if v == nil {
			return ""
		}
		return fmt.Sprintf("%v", v)
	}
	return expr.Value
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
// Constraint evaluation and Event guard evaluation (CAP-E06) — both are the
// same "does this data satisfy this expression" question.
func Eval(expr model.ConstraintExpression, data map[string]any) bool {
	raw := data[expr.Field]
	str := fmt.Sprintf("%v", raw)
	if raw == nil {
		str = ""
	}

	switch expr.Operator {
	case "required":
		return str != "" && str != "<nil>"

	case "equals":
		return str == resolveCompareValue(expr, data)

	case "not_equals":
		return str != resolveCompareValue(expr, data)

	case "after", "before", "on_or_after", "on_or_before":
		cmpStr := resolveCompareValue(expr, data)
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
		switch expr.Operator {
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
		for _, v := range expr.Values {
			if str == v {
				return true
			}
		}
		return false

	case "greater_than", "less_than", "greater_than_or_equal", "less_than_or_equal":
		cmpStr := resolveCompareValue(expr, data)
		n, err1 := strconv.ParseFloat(str, 64)
		cmpN, err2 := strconv.ParseFloat(cmpStr, 64)
		if err1 != nil || err2 != nil {
			return false
		}
		switch expr.Operator {
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
