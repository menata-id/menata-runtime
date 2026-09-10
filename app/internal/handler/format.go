package handler

import (
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"menata.id/app/internal/expr"
	"menata.id/app/internal/model"
)

// formatAutoNumber (CAP-F18) renders a sequence value as "<prefix><padded
// number>", e.g. prefix "INV-" + padding 4 + n=7 -> "INV-0007". padding 0
// (or omitted) means no zero-padding at all -- just the prefix and the
// plain number.
func formatAutoNumber(opts model.FieldOptions, n int64) string {
	if opts.AutoNumberPadding > 0 {
		return fmt.Sprintf("%s%0*d", opts.AutoNumberPrefix, opts.AutoNumberPadding, n)
	}
	return fmt.Sprintf("%s%d", opts.AutoNumberPrefix, n)
}

// boolLabel (CAP-F09) renders a boolean field's stored "true"/"false"
// string as a human-readable Yes/No -- anything else (unset, malformed) is
// treated as No, the same "absent = false" convention Create/Update's own
// checkbox handling already uses.
func boolLabel(val string) string {
	if val == "true" {
		return "Yes"
	}
	return "No"
}

// formatMoney (CAP-F08) appends the resolved currency code to a money
// field's raw numeric value -- Options.Currency (fixed) or
// data[Options.CurrencyField] (CAP-F17's per-transaction currency).
func formatMoney(val string, f *model.Field, data map[string]any) string {
	currency := f.Options.Currency
	if f.Options.CurrencyField != "" {
		currency = fmt.Sprintf("%v", data[f.Options.CurrencyField])
	}
	if currency == "" || currency == "<nil>" {
		return val
	}
	return currency + " " + val
}

// computedValue (CAP-F14) resolves a `computed` field's display value at
// render time -- never stored, matching CAP-V13's own "computed at render
// time" precedent. Options.Expression (CAP-F14 completion, CAP-C13's own
// expression layer), when set, REPLACES the plain SourceField*multiplier
// calculation below entirely -- see FieldOptions.Expression's own doc
// comment for why the two are mutually exclusive. A compile/eval error
// (bad expression text, a type mismatch, a reference to a field this
// record doesn't have) renders blank and logs a warning -- the same
// "prototype-honest heuristic, graceful degrade" posture this codebase
// already applies elsewhere (FieldOptions.RestrictToGroup's own doc
// comment), not a 500 for one bad computed Field on an otherwise-normal
// page. A non-bool/non-numeric/non-string CEL result (a list, a map) also
// renders blank -- not a shape this Field type is meant to display.
func computedValue(f *model.Field, data map[string]any) string {
	if f.Options.Expression != "" {
		return formatExpressionValue(f, data)
	}
	raw, ok := data[f.Options.SourceField]
	if !ok {
		return ""
	}
	n, err := strconv.ParseFloat(fmt.Sprintf("%v", raw), 64)
	if err != nil {
		return ""
	}
	multiplier := f.Options.Factor
	if f.Options.FactorField != "" {
		fv, ok := data[f.Options.FactorField]
		if !ok {
			return ""
		}
		multiplier, err = strconv.ParseFloat(fmt.Sprintf("%v", fv), 64)
		if err != nil {
			return ""
		}
	}
	return strconv.FormatFloat(n*multiplier, 'f', -1, 64)
}

// formatExpressionValue is computedValue's CAP-C13 branch -- evaluates
// Options.Expression against this record's own data (no `old`/
// `current_user`: a rendered value has no notion of "before this request"
// or "who's viewing," unlike a Constraint/Event condition) and renders
// whatever CEL type comes back in the same plain-string shape every other
// Field value already is.
func formatExpressionValue(f *model.Field, data map[string]any) string {
	v, err := expr.Eval(f.Options.Expression, expr.Vars{Record: data})
	if err != nil {
		slog.Warn("computed field expression failed", "field", f.ID, "error", err)
		return ""
	}
	switch val := v.(type) {
	case float64:
		return strconv.FormatFloat(val, 'f', -1, 64)
	case int64:
		return strconv.FormatInt(val, 10)
	case bool:
		return strconv.FormatBool(val)
	case string:
		return val
	default:
		slog.Warn("computed field expression returned an unrenderable type", "field", f.ID, "type", fmt.Sprintf("%T", v))
		return ""
	}
}

// slaUrgency (CAP-V17) computes a countdown badge's label and urgency
// bucket from a `date`-typed field's raw string value, at render time --
// same "computed at render time, nothing stored" precedent as CAP-F14's
// computedValue above and CAP-V13's Report. No existing date-subtraction
// helper covers this: CAP-A11's own resolveDateArithmetic/addBusinessDays
// (internal/executor/executor.go) only add a forward offset to a base
// date, never compare two dates -- this is the reverse operation, small
// enough not to warrant reusing/generalizing those.
//
// ok=false means dueDate didn't parse (blank field, bad data) -- render
// nothing, not a broken badge.
func slaUrgency(dueDate string, warningDays int) (label, urgency string, ok bool) {
	due, err := time.Parse("2006-01-02", dueDate)
	if err != nil {
		return "", "", false
	}
	days := int(time.Until(due).Hours() / 24)
	switch {
	case days < 0:
		return fmt.Sprintf("Overdue by %d day(s)", -days), "overdue", true
	case warningDays > 0 && days <= warningDays:
		return fmt.Sprintf("%d day(s) left", days), "warning", true
	default:
		return fmt.Sprintf("%d day(s) left", days), "ok", true
	}
}

// displayLabel picks a record's own human-readable label: its Name field, or
// the first plain-text field, or -- for a Machine with neither (e.g. Approval
// Step: only reference/user/number/value_list/rich_text fields) -- the
// record's own id, passed in explicitly since a Record's id lives on the
// store.Record struct itself, never inside its own Data map (caught live:
// this fallback used to read data["id"], which never exists, so it silently
// rendered "" for any such Machine instead of ever reaching this branch).
func displayLabel(machine *model.Machine, id string, data map[string]any) string {
	if machine != nil {
		var firstText *model.Field
		for _, f := range machine.Fields {
			if f.Type != model.FieldTypeText {
				continue
			}
			if firstText == nil {
				firstText = f
			}
			if strings.EqualFold(f.Name, "name") {
				firstText = f
				break
			}
		}
		if firstText != nil {
			if v, ok := data[firstText.ID]; ok {
				if s := fmt.Sprintf("%v", v); s != "" {
					return s
				}
			}
		}

		// No plain-text Field at all (Approval Step's own exact shape:
		// reference/user/number/value_list/rich_text only) -- "Machine
		// Name value" from the first `number` Field (preferring one
		// literally named "sequence") is still meaningfully better than
		// the bare id for a human reading a reverse-reference list
		// (CAP-V06, caught live: a Document's own "Approval Step (via
		// Document)" sub-list showed raw UUIDs) scoped to one specific
		// parent record, where "Approval Step 3" is unambiguous even
		// though the same label recurs across different parents' own
		// child sets. Same priority pattern as the text-field tier above,
		// one field type down -- not a Machine-specific special case.
		var firstNumber *model.Field
		for _, f := range machine.Fields {
			if f.Type != model.FieldTypeNumber {
				continue
			}
			if firstNumber == nil {
				firstNumber = f
			}
			if strings.EqualFold(f.Name, "sequence") {
				firstNumber = f
				break
			}
		}
		if firstNumber != nil {
			if v, ok := data[firstNumber.ID]; ok {
				if s := fmt.Sprintf("%v", v); s != "" {
					return machine.Name + " " + s
				}
			}
		}
	}
	return id
}
