package composable

import "menata.id/app/internal/model"

// ViewRef wraps an existing *model.View by reference -- the View itself
// remains the real source of truth (the roadmap's "View compatibility
// first" principle, §3.2); this is only a handle a UINode can carry
// without importing *model.View's full ViewConfig.
type ViewRef struct {
	ViewID    string
	MachineID string
	Type      model.ViewType
}

// UINodeKind is the small closed set of node kinds §8's own vocabulary
// names. UINodeComponent is declared here because §8 names it, but nothing
// in this package emits one yet -- Phase 5's real Component contract is
// what a "component" kind actually needs; every real node Phase 4 produces
// is UINodeViewRef or UINodeStaticContent ("Existing View types are
// adapted as view_ref/preset nodes first", §8's own adapter posture).
type UINodeKind string

const (
	UINodePage          UINodeKind = "page"
	UINodeLayout        UINodeKind = "layout"
	UINodeComponent     UINodeKind = "component"
	UINodeViewRef       UINodeKind = "view_ref"
	UINodeStaticContent UINodeKind = "static_content"
)

// UINode is the actual UI IR (composable-runtime-roadmap.md §8) -- the one
// generic tree every experience composition lowers into, superseding
// Phase 1/2's flatter PageNode/ComponentNode (nothing outside this package
// ever used them). ViewRef and Dataset are set only on a "view_ref" node
// (nil otherwise) -- unlike Phase 1/2's page-level "first child wins"
// simplification, every view_ref node now carries its OWN Dataset.
//
// Actions/Conditions are declared because §8's own vocabulary names them,
// but this phase's adapter never populates either -- the same "declared
// because a later phase needs the shape, not because this one populates
// it" posture Phase 1's own LayoutNode used to have, before it graduated
// into the UINodeLayout kind here.
type UINode struct {
	Identity      NodeIdentity
	Kind          UINodeKind
	ComponentType ComponentType // set only when Kind == UINodeComponent
	Properties    map[string]string
	ViewRef       *ViewRef
	Dataset       *Dataset
	Bindings      []Binding
	Children      []UINode
	Slots         map[string][]UINode
	Actions       []string
	Conditions    []Filter
}
