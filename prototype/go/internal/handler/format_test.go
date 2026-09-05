package handler

import (
	"testing"

	"menata.id/runtime/internal/model"
)

// displayLabel is exactly the kind of heuristic roadmap.md's own item 14
// named as needing a fast unit-test feedback loop the 200+-test HTTP
// black-box conformance suite doesn't give: caught live 2026-09-05 with a
// dead fallback (`data["id"]`, which store.Record never populates) that had
// apparently gone unnoticed for a long time because no conformance test
// happened to exercise a Machine with zero `text` fields until CAP-V20's
// Decision Stepper (Approval Step: reference/user/number/value_list/
// rich_text only) did. These tests pin the documented contract down so a
// future regression fails in milliseconds, not by someone eyeballing a
// blank label in a browser.

func fieldsMachine(fields ...*model.Field) *model.Machine {
	return &model.Machine{ID: "mch_test", Fields: fields}
}

func TestDisplayLabel_PrefersFieldNamedName(t *testing.T) {
	machine := fieldsMachine(
		&model.Field{ID: "fld_a", Name: "Description", Type: model.FieldTypeText},
		&model.Field{ID: "fld_b", Name: "Name", Type: model.FieldTypeText},
	)
	data := map[string]any{"fld_a": "some description", "fld_b": "Alice"}

	got := displayLabel(machine, "rec-id", data)
	if got != "Alice" {
		t.Fatalf("displayLabel() = %q, want %q (the Name-named field, not the first-declared one)", got, "Alice")
	}
}

func TestDisplayLabel_FirstTextFieldWhenNoNameField(t *testing.T) {
	machine := fieldsMachine(
		&model.Field{ID: "fld_a", Name: "Title", Type: model.FieldTypeText},
		&model.Field{ID: "fld_b", Name: "Subtitle", Type: model.FieldTypeText},
	)
	data := map[string]any{"fld_a": "Kebijakan Kerja Jarak Jauh", "fld_b": "v2"}

	got := displayLabel(machine, "rec-id", data)
	if got != "Kebijakan Kerja Jarak Jauh" {
		t.Fatalf("displayLabel() = %q, want the first-declared text field's value", got)
	}
}

func TestDisplayLabel_FallsBackToIDWithNoTextField(t *testing.T) {
	// The exact shape of Approval Step (CAP-V20's Decision Stepper) that
	// surfaced the dead data["id"] fallback bug: no text field at all,
	// only reference/user/number/value_list/rich_text.
	machine := fieldsMachine(
		&model.Field{ID: "fld_document", Name: "Document", Type: model.FieldTypeReference},
		&model.Field{ID: "fld_approver", Name: "Approver", Type: model.FieldTypeUser},
		&model.Field{ID: "fld_sequence", Name: "Sequence", Type: model.FieldTypeNumber},
	)
	data := map[string]any{"fld_sequence": float64(1)}

	got := displayLabel(machine, "step-id-123", data)
	if got != "step-id-123" {
		t.Fatalf("displayLabel() = %q, want the passed-in id %q", got, "step-id-123")
	}
}

func TestDisplayLabel_NilMachineFallsBackToID(t *testing.T) {
	got := displayLabel(nil, "rec-id", map[string]any{"fld_a": "ignored, no machine to resolve it against"})
	if got != "rec-id" {
		t.Fatalf("displayLabel(nil, ...) = %q, want the passed-in id %q", got, "rec-id")
	}
}

func TestDisplayLabel_EmptyNameFieldValueFallsBackToID(t *testing.T) {
	// Documents a real, possibly-surprising branch: a declared Name field
	// with an EMPTY string value does not return "" -- it falls all the
	// way through to the id, same as if the field didn't exist at all.
	machine := fieldsMachine(
		&model.Field{ID: "fld_name", Name: "Name", Type: model.FieldTypeText},
	)
	data := map[string]any{"fld_name": ""}

	got := displayLabel(machine, "rec-id", data)
	if got != "rec-id" {
		t.Fatalf("displayLabel() with an empty Name value = %q, want fallback to id %q, not an empty label", got, "rec-id")
	}
}

func TestDisplayLabel_MissingDataKeyFallsBackToID(t *testing.T) {
	// A declared text Field whose id simply isn't a key in data at all
	// (not even present, not just empty) -- same fallback as above.
	machine := fieldsMachine(
		&model.Field{ID: "fld_name", Name: "Name", Type: model.FieldTypeText},
	)
	got := displayLabel(machine, "rec-id", map[string]any{})
	if got != "rec-id" {
		t.Fatalf("displayLabel() with no data at all = %q, want fallback to id %q", got, "rec-id")
	}
}

func TestDisplayLabel_NonStringValueIsFormatted(t *testing.T) {
	// data is map[string]any -- decoded JSON numbers arrive as float64, not
	// string. displayLabel must still render a real label, via fmt.Sprintf,
	// not a Go-syntax type dump.
	machine := fieldsMachine(
		&model.Field{ID: "fld_name", Name: "Name", Type: model.FieldTypeText},
	)
	got := displayLabel(machine, "rec-id", map[string]any{"fld_name": float64(42)})
	if got != "42" {
		t.Fatalf("displayLabel() with a non-string value = %q, want %q", got, "42")
	}
}
