package composable

import "fmt"

// baseDomains are ambient at every nesting level -- ContextDomain values
// with no entity identity of their own, so a child scope always keeps them
// without having to redeclare them. Nothing else carries forward
// automatically; see DeriveChildScope.
var baseDomains = []ContextDomain{
	ContextPage, ContextRoute, ContextParameters, ContextCurrentUser, ContextWorkspace,
}

// entityDomains are the three ContextDomain values ValidateBinding requires
// to name a real Machine id (§7 rule "record/collection context is
// explicit") -- the domains DeriveChildScope drops unless redeclared.
var entityDomains = map[ContextDomain]bool{
	ContextRecord:       true,
	ContextParentRecord: true,
	ContextQueryResult:  true,
}

// DeriveChildScope builds a nested Scope from parent -- the concrete
// mechanism behind §7's "child scopes cannot access undeclared parent
// values": baseDomains carry forward automatically (they never name
// anything level-specific), but record/parent_record/query_result/
// selection/variables do NOT -- whatever parent declared for them is gone
// unless add explicitly redeclares it. There is no path from the returned
// Scope back to parent's full Domains map at all, so this isn't a policy a
// caller could accidentally bypass.
func DeriveChildScope(parent Scope, add map[ContextDomain]string) Scope {
	domains := make(map[ContextDomain]string, len(baseDomains)+len(add))
	for _, d := range baseDomains {
		if v, ok := parent.Domains[d]; ok {
			domains[d] = v
		}
	}
	for d, v := range add {
		domains[d] = v
	}
	return Scope{Domains: domains}
}

// ScopeForDataset seeds a root Scope from a Phase 2 Dataset -- ds.Source
// becomes a query_result (a Dataset is a collection of rows), and every
// Relation Phase 2 already discovered becomes a parent_record. This is
// Phase 3 reusing Phase 2's own RelationRef discovery as real input,
// instead of re-deriving nesting from ViewConfig.Children independently.
// The always-ambient domains (page/route/parameters/current_user/
// workspace) are present but left unnamed ("") -- a root Scope, by
// definition, has no page/route identity of its own yet; a caller
// constructing an actual page's Scope sets those separately.
func ScopeForDataset(ds Dataset) Scope {
	domains := map[ContextDomain]string{
		ContextPage:        "",
		ContextRoute:       "",
		ContextParameters:  "",
		ContextCurrentUser: "",
		ContextWorkspace:   "",
		ContextQueryResult: ds.Source.MachineID,
	}
	// A Dataset's own Relations name every Machine its rows point AT, not
	// necessarily "the one parent this collection is scoped under" --
	// Phase 3 takes the common case (exactly one Relation) as the parent;
	// a Dataset with more than one is left for the caller to disambiguate
	// (Phase 3 doesn't need to resolve that yet, no case demands it).
	if len(ds.Relations) == 1 {
		domains[ContextParentRecord] = ds.Relations[0].TargetMachineID
	}
	return Scope{Domains: domains}
}

// ValidateBinding is §7 rule "bindings are validated before rendering" 's
// actual mechanism -- called wherever a Binding is constructed, never
// inside a renderer (nothing outside this package calls it, same posture
// Phases 1-2 already established). It enforces three rules at once:
//
//   - Target.Domain must be a key in scope.Domains at all (rule "child
//     scopes cannot access undeclared parent values" -- DeriveChildScope is
//     what makes an ancestor's domain actually absent, this is what refuses
//     to look past that absence);
//   - if Target.Domain is record/parent_record/query_result,
//     scope.Domains[Target.Domain] must be non-empty (rule "record/
//     collection context is explicit" -- declaring the domain isn't enough,
//     it must name a Machine);
//   - if Target.Domain is current_user, Source must be exactly
//     currentUserSentinel (rule "security context is never downgraded by a
//     child binding" made concrete: a Binding may REFERENCE current_user,
//     it may never RE-SOURCE it from arbitrary field data).
func ValidateBinding(scope Scope, b Binding) error {
	value, declared := scope.Domains[b.Target.Domain]
	if !declared {
		return fmt.Errorf("composable: binding targets undeclared domain %q", b.Target.Domain)
	}
	if entityDomains[b.Target.Domain] && value == "" {
		return fmt.Errorf("composable: domain %q is declared but names no machine -- record/collection context must be explicit", b.Target.Domain)
	}
	if b.Target.Domain == ContextCurrentUser && b.Source != currentUserSentinel {
		return fmt.Errorf("composable: binding to current_user must use the %q sentinel, got %q", currentUserSentinel, b.Source)
	}
	return nil
}
