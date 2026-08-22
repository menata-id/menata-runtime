package handler

import (
	"fmt"
	"strconv"
	"strings"

	"menata.id/runtime/internal/model"
)

// displayLabel picks a human-readable stand-in for a record: the target
// Machine's `text` field named "Name" if one exists, else its first `text`
// field, falling back to the record id. Menata Language doesn't (yet) let a
// business author declare "this is the field people should see when
// referencing a record" — this heuristic is a prototype stand-in for that
// missing capability, not a final design.
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
// render time -- data[Options.SourceField] * Options.Factor -- never
// stored, matching CAP-V13's own "computed at render time" precedent. A
// non-numeric or missing source renders blank rather than "0", the same
// "don't fabricate a number for missing data" posture SumField already
// takes.
func computedValue(f *model.Field, data map[string]any) string {
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

func displayLabel(machine *model.Machine, data map[string]any) string {
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
	}
	if id, ok := data["id"]; ok {
		return fmt.Sprintf("%v", id)
	}
	return ""
}
