package executor

import (
	"testing"
	"time"
)

// resolveDateArithmetic/addBusinessDays are CAP-A11/CAP-O06's pure date-math
// heuristics, exactly the kind roadmap.md's own item 14 named as needing a
// fast unit-test loop the HTTP black-box conformance suite doesn't give.
// Writing this file's very first test (the "-" operator case below) is what
// caught a real bug the same day: guides/writing-runtime-metadata.md has
// documented "<basis> - N <unit>" as valid syntax since CAP-A11 shipped, but
// dateArithRe only ever matched a literal "+" -- "today - 7 Days" silently
// fell through unevaluated as a literal string instead of a real date, and
// no seed or conformance test happened to exercise the documented "-" form
// to catch it sooner. Fixed in the same pass as this test file.

func TestResolveDateArithmetic_PlusDays(t *testing.T) {
	data := map[string]any{"fld_start": "2026-01-01"}
	got, ok := resolveDateArithmetic("fld_start + 7 Days", data, nil)
	if !ok {
		t.Fatal("resolveDateArithmetic() ok = false, want true")
	}
	if got != "2026-01-08" {
		t.Fatalf("resolveDateArithmetic() = %q, want %q", got, "2026-01-08")
	}
}

func TestResolveDateArithmetic_MinusDays(t *testing.T) {
	// The bug: this exact form ("<basis> - N <unit>") is documented in
	// guides/writing-runtime-metadata.md as valid CAP-A11 syntax.
	data := map[string]any{"fld_start": "2026-01-08"}
	got, ok := resolveDateArithmetic("fld_start - 7 Days", data, nil)
	if !ok {
		t.Fatal("resolveDateArithmetic() ok = false, want true -- the documented \"-\" operator form should match")
	}
	if got != "2026-01-01" {
		t.Fatalf("resolveDateArithmetic() = %q, want %q", got, "2026-01-01")
	}
}

func TestResolveDateArithmetic_Weeks(t *testing.T) {
	data := map[string]any{"fld_start": "2026-01-01"}
	got, ok := resolveDateArithmetic("fld_start + 2 Weeks", data, nil)
	if !ok || got != "2026-01-15" {
		t.Fatalf("resolveDateArithmetic() = (%q, %v), want (%q, true)", got, ok, "2026-01-15")
	}
}

func TestResolveDateArithmetic_MonthsAndYears(t *testing.T) {
	data := map[string]any{"fld_start": "2026-01-15"}

	if got, ok := resolveDateArithmetic("fld_start + 1 Month", data, nil); !ok || got != "2026-02-15" {
		t.Fatalf("+1 Month = (%q, %v), want (%q, true)", got, ok, "2026-02-15")
	}
	if got, ok := resolveDateArithmetic("fld_start - 1 Year", data, nil); !ok || got != "2025-01-15" {
		t.Fatalf("-1 Year = (%q, %v), want (%q, true)", got, ok, "2025-01-15")
	}
}

func TestResolveDateArithmetic_TodayBasis(t *testing.T) {
	// The one non-deterministic basis -- checked against a real time.Now()
	// computed the same way, not a hardcoded date, so this test survives
	// whatever day it actually runs on.
	want := time.Now().Truncate(24 * time.Hour).AddDate(0, 0, 3).Format("2006-01-02")
	got, ok := resolveDateArithmetic("today + 3 Days", nil, nil)
	if !ok || got != want {
		t.Fatalf("resolveDateArithmetic(today + 3 Days) = (%q, %v), want (%q, true)", got, ok, want)
	}
}

func TestResolveDateArithmetic_BusinessDaysSkipsWeekends(t *testing.T) {
	// 2026-01-01 is a Thursday. +3 Business Days should land on Tue
	// 2026-01-06 (Fri, Mon, Tue -- Sat/Sun skipped).
	data := map[string]any{"fld_start": "2026-01-01"}
	got, ok := resolveDateArithmetic("fld_start + 3 Business Days", data, nil)
	if !ok || got != "2026-01-06" {
		t.Fatalf("resolveDateArithmetic() = (%q, %v), want (%q, true)", got, ok, "2026-01-06")
	}
}

func TestResolveDateArithmetic_BusinessDaysSkipsHolidays(t *testing.T) {
	// Same +3 Business Days from 2026-01-01, but 2026-01-02 (Fri) is also
	// declared a holiday -- CAP-O06's workspace_holidays -- so it should
	// additionally skip past it, landing one weekday later than the
	// weekend-only case above.
	data := map[string]any{"fld_start": "2026-01-01"}
	holidays := map[string]bool{"2026-01-02": true}
	got, ok := resolveDateArithmetic("fld_start + 3 Business Days", data, holidays)
	if !ok || got != "2026-01-07" {
		t.Fatalf("resolveDateArithmetic() = (%q, %v), want (%q, true)", got, ok, "2026-01-07")
	}
}

func TestResolveDateArithmetic_NotAMatch(t *testing.T) {
	tests := []string{
		"today",                 // no operator/unit at all -- a plain token, handled elsewhere in resolveValue
		"just a literal string", // not this shape
		"fld_start + Days",      // missing the number
	}
	for _, value := range tests {
		if _, ok := resolveDateArithmetic(value, map[string]any{"fld_start": "2026-01-01"}, nil); ok {
			t.Errorf("resolveDateArithmetic(%q) ok = true, want false", value)
		}
	}
}

func TestResolveDateArithmetic_BaseFieldMissingOrNotADate(t *testing.T) {
	tests := []struct {
		name string
		data map[string]any
	}{
		{"field not present in data at all", map[string]any{}},
		{"field present but not a valid date string", map[string]any{"fld_start": "not-a-date"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, ok := resolveDateArithmetic("fld_start + 7 Days", tt.data, nil); ok {
				t.Errorf("resolveDateArithmetic() ok = true, want false")
			}
		})
	}
}

func TestAddBusinessDays_SkipsWeekendsOnly(t *testing.T) {
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC) // Thursday
	got := addBusinessDays(base, 3, nil)
	want := time.Date(2026, 1, 6, 0, 0, 0, 0, time.UTC) // Tuesday (Fri, Mon, Tue)
	if !got.Equal(want) {
		t.Fatalf("addBusinessDays(+3) = %v, want %v", got.Format("2006-01-02"), want.Format("2006-01-02"))
	}
}

func TestAddBusinessDays_Negative(t *testing.T) {
	base := time.Date(2026, 1, 6, 0, 0, 0, 0, time.UTC) // Tuesday
	got := addBusinessDays(base, -3, nil)
	want := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC) // Thursday (Mon, Fri, Thu going backward)
	if !got.Equal(want) {
		t.Fatalf("addBusinessDays(-3) = %v, want %v", got.Format("2006-01-02"), want.Format("2006-01-02"))
	}
}

func TestAddBusinessDays_NilHolidaysIsSafe(t *testing.T) {
	// nil map read (holidays[key]) must not panic -- a Workspace with no
	// declared holidays passes nil here, not an empty map.
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	got := addBusinessDays(base, 1, nil)
	if got.Weekday() == time.Saturday || got.Weekday() == time.Sunday {
		t.Fatalf("addBusinessDays landed on a weekend: %v", got)
	}
}
