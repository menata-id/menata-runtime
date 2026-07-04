package constraint

import (
	"fmt"
	"time"

	"menata.id/runtime/internal/model"
)

type Engine struct{}

// Violations returns human-readable messages for every constraint the data breaks.
// An empty slice means the data satisfies all constraints.
func (e *Engine) Violations(machine *model.Machine, data map[string]any) []string {
	var out []string
	for _, c := range machine.Constraints {
		if c.Condition != nil && !eval(*c.Condition, data) {
			continue
		}
		if !eval(c.Expression, data) {
			out = append(out, c.Rule)
		}
	}
	return out
}

func eval(expr model.ConstraintExpression, data map[string]any) bool {
	raw := data[expr.Field]
	str := fmt.Sprintf("%v", raw)
	if raw == nil {
		str = ""
	}

	switch expr.Operator {
	case "required":
		return str != "" && str != "<nil>"

	case "equals":
		return str == expr.Value

	case "not_equals":
		return str != expr.Value

	case "after":
		if expr.Value != "today" {
			return true
		}
		if str == "" {
			return false
		}
		t, err := time.Parse("2006-01-02", str)
		if err != nil {
			return false
		}
		today := time.Now().Truncate(24 * time.Hour)
		return t.After(today)
	}
	return true
}
