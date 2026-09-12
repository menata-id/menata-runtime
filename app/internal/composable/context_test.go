package composable_test

import (
	"testing"

	"menata.id/app/internal/composable"
)

func TestDeriveChildScopeDropsUndeclaredParentValues(t *testing.T) {
	parent := composable.Scope{Domains: map[composable.ContextDomain]string{
		composable.ContextPage:        "",
		composable.ContextCurrentUser: "",
		composable.ContextRecord:      "mch_x",
	}}

	child := composable.DeriveChildScope(parent, map[composable.ContextDomain]string{
		composable.ContextQueryResult: "mch_y",
	})

	if _, ok := child.Domains[composable.ContextRecord]; ok {
		t.Fatalf("child scope carried forward undeclared parent domain 'record': %+v", child.Domains)
	}
	if _, ok := child.Domains[composable.ContextPage]; !ok {
		t.Error("child scope dropped an ambient domain ('page') that should always carry forward")
	}

	err := composable.ValidateBinding(child, composable.Binding{
		Target: composable.ContextRef{Domain: composable.ContextRecord, Key: "fld_x"},
	})
	if err == nil {
		t.Fatal("ValidateBinding: want error for a binding to a domain the child scope never declared")
	}
}

func TestValidateBinding_RecordContextMustBeExplicit(t *testing.T) {
	scope := composable.Scope{Domains: map[composable.ContextDomain]string{
		composable.ContextRecord: "", // declared, but names no machine
	}}
	err := composable.ValidateBinding(scope, composable.Binding{
		Target: composable.ContextRef{Domain: composable.ContextRecord, Key: "fld_x"},
	})
	if err == nil {
		t.Fatal("ValidateBinding: want error for a record domain declared without naming a machine")
	}
}

func TestValidateBinding_CurrentUserSentinelOnly(t *testing.T) {
	scope := composable.Scope{Domains: map[composable.ContextDomain]string{
		composable.ContextCurrentUser: "",
	}}

	bad := composable.Binding{
		Target: composable.ContextRef{Domain: composable.ContextCurrentUser},
		Source: "fld_as_approver",
	}
	if err := composable.ValidateBinding(scope, bad); err == nil {
		t.Fatal("ValidateBinding: want error for current_user re-sourced from arbitrary field data")
	}

	good := composable.Binding{
		Target: composable.ContextRef{Domain: composable.ContextCurrentUser},
		Source: "$current_user",
	}
	if err := composable.ValidateBinding(scope, good); err != nil {
		t.Fatalf("ValidateBinding: want no error for the reserved $current_user sentinel, got %v", err)
	}
}

// TestFourLevelSyntheticNesting is Phase 3's own project-management-shaped
// proof case -- composable-runtime-roadmap.md §7's "Project → List → Card
// → Checklist" nesting. Case 19 isn't seeded in app/ yet (same gap flagged
// in Phases 1-2), so this is a synthetic, builder-only Scope chain proving
// DeriveChildScope composes to arbitrary depth generically: Board (a
// collection of Lists) → one List (a collection of Cards) → one Card (a
// collection of Checklist items) → one Checklist item.
func TestFourLevelSyntheticNesting(t *testing.T) {
	board := composable.Scope{Domains: map[composable.ContextDomain]string{
		composable.ContextQueryResult: "mch_board",
	}}
	lists := composable.DeriveChildScope(board, map[composable.ContextDomain]string{
		composable.ContextParentRecord: "mch_board",
		composable.ContextQueryResult:  "mch_list",
	})
	oneList := composable.DeriveChildScope(lists, map[composable.ContextDomain]string{
		composable.ContextRecord: "mch_list",
	})
	cards := composable.DeriveChildScope(oneList, map[composable.ContextDomain]string{
		composable.ContextParentRecord: "mch_list",
		composable.ContextQueryResult:  "mch_card",
	})
	oneCard := composable.DeriveChildScope(cards, map[composable.ContextDomain]string{
		composable.ContextRecord: "mch_card",
	})
	checklist := composable.DeriveChildScope(oneCard, map[composable.ContextDomain]string{
		composable.ContextParentRecord: "mch_card",
		composable.ContextQueryResult:  "mch_checklist_item",
	})

	// Every intermediate scope must carry the CURRENT level's identity only
	// -- an ancestor's stale query_result/record must never survive past
	// the level that redeclared it.
	if checklist.Domains[composable.ContextQueryResult] != "mch_checklist_item" {
		t.Errorf("checklist query_result = %q, want mch_checklist_item", checklist.Domains[composable.ContextQueryResult])
	}
	if checklist.Domains[composable.ContextParentRecord] != "mch_card" {
		t.Errorf("checklist parent_record = %q, want mch_card", checklist.Domains[composable.ContextParentRecord])
	}
	if _, ok := checklist.Domains[composable.ContextRecord]; ok {
		t.Errorf("checklist scope still carries a stale 'record' from the Card level: %+v", checklist.Domains)
	}

	// A binding into the checklist collection's own row-level record domain
	// must be rejected here -- it hasn't been declared yet at this
	// (collection) level, one more level down would.
	err := composable.ValidateBinding(checklist, composable.Binding{
		Target: composable.ContextRef{Domain: composable.ContextRecord, Key: "fld_done"},
	})
	if err == nil {
		t.Fatal("ValidateBinding: want error -- checklist collection scope has no singular 'record' yet")
	}
}
