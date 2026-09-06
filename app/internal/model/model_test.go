package model

import "testing"

// FindFieldByName/FindReferenceFieldTo are the two name/type-matching
// heuristics this codebase leans on wherever Menata Language has no grammar
// yet for a business author to name "the field that means X" (CAP-A07's
// own Sequence/Decision/Approver lookup, displayLabel's Name-or-first-text-
// field rule, CAP-W03's declarative quorum target resolution). They're pure
// functions over a *Machine with no dependency on a database, HTTP request,
// or Interpreter -- exactly the "fast feedback loop the conformance suite
// alone doesn't give" case roadmap.md's own item 14 named, and the class of
// bug caught live 2026-09-05 in displayLabel (a sibling heuristic in
// internal/handler/format.go) went unnoticed for a long time precisely
// because nothing but the 200+-test HTTP black-box suite ever exercised it.

func TestFindFieldByName(t *testing.T) {
	machine := &Machine{
		ID: "mch_test",
		Fields: []*Field{
			{ID: "fld_title", Name: "Title", Type: FieldTypeText},
			{ID: "fld_seq", Name: "Sequence", Type: FieldTypeNumber},
		},
	}

	tests := []struct {
		name    string
		lookup  string
		wantID  string
		wantNil bool
	}{
		{"exact match", "Sequence", "fld_seq", false},
		{"case-insensitive match", "sequence", "fld_seq", false},
		{"case-insensitive match, different case", "SEQUENCE", "fld_seq", false},
		{"no match", "Decision", "", true},
		{"empty machine", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := machine
			if tt.name == "empty machine" {
				m = &Machine{}
			}
			got := FindFieldByName(m, tt.lookup)
			if tt.wantNil {
				if got != nil {
					t.Fatalf("FindFieldByName(%q) = %v, want nil", tt.lookup, got)
				}
				return
			}
			if got == nil {
				t.Fatalf("FindFieldByName(%q) = nil, want field %q", tt.lookup, tt.wantID)
			}
			if got.ID != tt.wantID {
				t.Fatalf("FindFieldByName(%q) = field %q, want %q", tt.lookup, got.ID, tt.wantID)
			}
		})
	}
}

func TestFindFieldByName_NilMachine(t *testing.T) {
	// Every call site in this codebase guards against a nil *Machine before
	// reaching this heuristic (displayLabel's own "if machine != nil" being
	// the pattern to copy) -- this test documents that FindFieldByName
	// itself has NO such guard, so a future caller that skips the nil check
	// gets a real, immediate panic here rather than a confusing one two
	// frames up inside the range loop.
	defer func() {
		if recover() == nil {
			t.Fatal("FindFieldByName(nil, ...) did not panic -- if this changed intentionally, update this test and check every call site's own nil-guard is still needed")
		}
	}()
	FindFieldByName(nil, "Title")
}

func TestFindReferenceFieldTo(t *testing.T) {
	machine := &Machine{
		ID: "mch_approval_step",
		Fields: []*Field{
			{ID: "fld_as_document", Name: "Document", Type: FieldTypeReference, Options: FieldOptions{TargetMachine: "mch_approval_document"}},
			{ID: "fld_as_approver", Name: "Approver", Type: FieldTypeUser},
			{ID: "fld_as_other_ref", Name: "Other", Type: FieldTypeReference, Options: FieldOptions{TargetMachine: "mch_something_else"}},
		},
	}

	tests := []struct {
		name        string
		targetID    string
		wantFieldID string
	}{
		{"reference field to target exists", "mch_approval_document", "fld_as_document"},
		{"reference field to a different target", "mch_something_else", "fld_as_other_ref"},
		{"no reference field to this target", "mch_does_not_exist", ""},
		{"a user-typed field never matches, even by coincidence", "mch_approval_step", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FindReferenceFieldTo(machine, tt.targetID)
			if tt.wantFieldID == "" {
				if got != nil {
					t.Fatalf("FindReferenceFieldTo(%q) = %v, want nil", tt.targetID, got)
				}
				return
			}
			if got == nil {
				t.Fatalf("FindReferenceFieldTo(%q) = nil, want field %q", tt.targetID, tt.wantFieldID)
			}
			if got.ID != tt.wantFieldID {
				t.Fatalf("FindReferenceFieldTo(%q) = field %q, want %q", tt.targetID, got.ID, tt.wantFieldID)
			}
		})
	}
}

func TestFindReferenceFieldTo_FirstMatchWins(t *testing.T) {
	// Documents the (undocumented until now) tie-breaking rule: if two
	// Fields on the same Machine both reference the same target, the FIRST
	// one declared (Fields is already position-ordered by the loader) wins,
	// silently -- not an error, not the last one. Worth pinning down: this
	// is exactly the kind of implicit behavior a future refactor could
	// invert by accident with nothing catching it.
	machine := &Machine{
		Fields: []*Field{
			{ID: "fld_first", Type: FieldTypeReference, Options: FieldOptions{TargetMachine: "mch_x"}},
			{ID: "fld_second", Type: FieldTypeReference, Options: FieldOptions{TargetMachine: "mch_x"}},
		},
	}
	got := FindReferenceFieldTo(machine, "mch_x")
	if got == nil || got.ID != "fld_first" {
		t.Fatalf("FindReferenceFieldTo() = %v, want the first-declared field fld_first", got)
	}
}
