package composable

// ContextDomain is one of the initial context domains named by
// composable-runtime-roadmap.md §7.
type ContextDomain string

const (
	ContextPage         ContextDomain = "page"
	ContextRoute        ContextDomain = "route"
	ContextParameters   ContextDomain = "parameters"
	ContextCurrentUser  ContextDomain = "current_user"
	ContextWorkspace    ContextDomain = "workspace"
	ContextRecord       ContextDomain = "record"
	ContextParentRecord ContextDomain = "parent_record"
	ContextSelection    ContextDomain = "selection"
	ContextQueryResult  ContextDomain = "query_result"
	ContextVariables    ContextDomain = "variables"
)

// currentUserSentinel is the reserved Source value a Binding to
// ContextCurrentUser must use verbatim -- the same $current_user sentinel
// CAP-V05 already uses in model.FilterCondition, reused here rather than
// inventing a second one. See ValidateBinding's own doc comment for why
// nothing else is accepted.
const currentUserSentinel = "$current_user"

// Scope is the set of context domains visible to a node, naming what each
// domain concretely refers to. Domains carrying no entity identity (page,
// route, parameters, current_user, workspace, selection, variables) map to
// "" -- the three that DO name something real (record, parent_record,
// query_result) must map to a non-empty Machine id, enforced by
// ValidateBinding (§7 rule "record/collection context is explicit").
type Scope struct {
	Domains map[ContextDomain]string
}

// ContextRef points at one value within a Scope's domain -- Key is a plain
// field id, never an expression or query fragment (§7 rule "context does
// not become an arbitrary query API" is true by construction: neither this
// type nor Binding has a field that could hold one).
type ContextRef struct {
	Domain ContextDomain
	Key    string
}

// Binding connects a component's own property to a ContextRef. Source is a
// plain string -- a component-side property name, a literal, or (for
// ContextCurrentUser only) the reserved currentUserSentinel -- never an
// expression language. Validated by ValidateBinding, always before
// rendering, never inside a renderer.
type Binding struct {
	Target ContextRef
	Source string
}
