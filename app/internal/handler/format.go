package handler

import (
	"fmt"
	"strconv"
	"strings"
	"time"

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
	}
	return id
}
