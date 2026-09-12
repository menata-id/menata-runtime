package composable

import (
	"fmt"

	"menata.id/app/internal/model"
)

// maxCompositionDepth bounds how many levels deep a page-type View's own
// Children may recurse (composable-runtime-roadmap.md §15's CMP-08,
// "bounded fan-out"). Today's real metadata/validate.go allow-lists
// (PageEmbeddableViewTypes/EmbeddableChildViewTypes) never let a page
// embed another page, so this bound is defense in depth for the substrate
// itself, not a fix to a reachable limit -- see LowerChildren's own doc
// comment.
const maxCompositionDepth = 5

// IndexViews resolves a View by id across every Machine in app -- needed
// because a Children entry may reference a View on a DIFFERENT Machine
// (seeds/042_inline_view_composition.sql: Approval Step's own Detail
// embeds Approval Document's own decision stepper).
func IndexViews(app *model.Application) map[string]*model.View {
	idx := make(map[string]*model.View)
	for _, m := range app.Machines {
		for _, v := range m.Views {
			idx[v.ID] = v
		}
	}
	return idx
}

// DatasetIndex resolves a declared model.Dataset by id -- what a
// component+dataset Children entry's own DatasetID names (CR-21, 17k).
type DatasetIndex map[string]*model.Dataset

// IndexDatasets builds a DatasetIndex over every Dataset declared on app.
func IndexDatasets(app *model.Application) DatasetIndex {
	idx := make(DatasetIndex, len(app.Datasets))
	for _, ds := range app.Datasets {
		idx[ds.ID] = ds
	}
	return idx
}

// lowerViewRefChild builds a view_ref UINode for v -- its own Dataset (via
// BuildDatasetFromView, nil when v's type has no representable data
// requirement) and, when that Dataset has Relations, validated
// parent_record Bindings (ScopeForDataset + ValidateBinding) -- the
// concrete wiring Phase 3 explicitly deferred to "whenever a real UI IR
// node tree exists."
func lowerViewRefChild(m *model.Machine, v *model.View) UINode {
	node := UINode{
		Identity: NodeIdentity{
			Kind:   "component",
			Source: m.ID,
			Params: map[string]string{"view": v.ID},
		},
		Kind:       UINodeViewRef,
		Properties: map[string]string{"view_id": v.ID, "view_type": string(v.Type)},
		ViewRef:    &ViewRef{ViewID: v.ID, MachineID: m.ID, Type: v.Type},
	}

	ds, err := BuildDatasetFromView(m, v)
	if err != nil {
		return node
	}
	node.Dataset = &ds

	if len(ds.Relations) == 0 {
		return node
	}
	scope := ScopeForDataset(ds)
	for _, rel := range ds.Relations {
		b := Binding{
			Target: ContextRef{Domain: ContextParentRecord, Key: rel.ViaField},
			Source: rel.ViaField,
		}
		if ValidateBinding(scope, b) == nil {
			node.Bindings = append(node.Bindings, b)
		}
	}
	return node
}

// lowerStaticContent builds a UINode from a Children entry's PageContent --
// a small closed vocabulary (model.go's own doc comment: "heading" |
// "text" | "button" | "image"). "heading"/"text" are Phase 5's own two
// proven components (LowerHeading/LowerText) -- migrated in place per §9's
// "migrate only a small number of proven generic components first" -- so
// they resolve through the component contract instead of a bare
// static_content node. "button"/"image" have no component contract yet and
// keep falling back to the original generic static_content shape,
// unchanged.
func lowerStaticContent(c *model.PageContent) UINode {
	switch c.Type {
	case "heading":
		if node, err := LowerHeading(c.Text); err == nil {
			return node
		}
	case "text":
		if node, err := LowerText(c.Text); err == nil {
			return node
		}
	}
	return UINode{
		Kind: UINodeStaticContent,
		Properties: map[string]string{
			"type": c.Type,
			"text": c.Text,
			"href": c.Href,
			"src":  c.Src,
		},
	}
}

// LowerChildren lowers a page-type View's own Config.Children (CAP-V10/
// CAP-V20's existing composition mechanism, model.ChildViewRef) into a
// Slots map -- "" is the default (unnamed) slot, "main"/"aside" are named
// slots (CAP-V10 Tier 2's own 2/3+1/3 row). A {content: ...} entry lowers
// directly via lowerStaticContent. A {view: id} entry resolves via
// viewIdx/machineIdx into a view_ref UINode; when that resolved View is
// ITSELF a page-type View with its own Children, LowerChildren genuinely
// recurses into them.
//
// visited names every View id already in the current composition path --
// checked before resolving a {view: id} entry (CMP-02, cyclic composition
// rejected) and its size doubles as the depth counter, bounded at
// maxCompositionDepth (CMP-08, bounded fan-out). Today's real metadata/
// validate.go allow-lists never let a page embed another page, so no real
// seeded metadata can actually cycle or recurse this deep yet -- this is
// defense in depth for the substrate itself, exercised by builder-only
// fixtures in conformance_test.go, not a fix to a reachable bug.
//
// An entry naming a View (or that View's own Machine) not present in the
// indexes now fails loud (CMP-03, unresolved reference rejected) rather
// than silently skipping, and an entry naming neither View nor Content
// does too (CMP-04, slot/type mismatch rejected) -- both are genuine
// authoring errors, this codebase's own CAP-X05 "Unknown = explicit"
// posture applied here for the first time (Phase 1-10 had no case that
// needed it: metadata/validate.go's own load-time checks already
// guarantee every REAL seeded Children entry resolves; this only ever
// fires for a hand-built UINode/ChildViewRef bypassing that).
// lowerComponentChild resolves a component+dataset Children entry (CR-21,
// 17k) -- the Experience-plane counterpart to view/content, declaring
// "render Component X bound to declared Dataset Y" directly. Split out of
// LowerChildren itself (Gate 3: keeps its own complexity from growing).
// entry.DatasetID's own validity (real Dataset on this Application) was
// already checked at load time (metadata/validate.go's validateDatasets);
// only an unresolved id here would mean a hand-built ChildViewRef
// bypassing that, same CMP-03 posture as an unresolved view.
func lowerComponentChild(entry model.ChildViewRef, machineIdx MachineIndex, datasetIdx DatasetIndex) (UINode, error) {
	declared, ok := datasetIdx[entry.DatasetID]
	if !ok {
		return UINode{}, fmt.Errorf("composable: children entry names unknown dataset %q (CMP-03 unresolved reference rejected)", entry.DatasetID)
	}
	ds, err := BuildDatasetFromDeclaredDataset(machineIdx, declared)
	if err != nil {
		return UINode{}, err
	}
	node, err := ResolveComponent(ComponentType(entry.Component), entry.Properties, &ds, nil)
	if err != nil {
		return UINode{}, fmt.Errorf("composable: children entry component %q: %w", entry.Component, err)
	}
	return node, nil
}

func LowerChildren(children []model.ChildViewRef, viewIdx map[string]*model.View, machineIdx MachineIndex, datasetIdx DatasetIndex, visited map[string]bool) (map[string][]UINode, error) {
	if len(children) == 0 {
		return nil, nil
	}
	if len(visited) >= maxCompositionDepth {
		return nil, fmt.Errorf("composable: composition depth exceeds %d (CMP-08 bounded fan-out)", maxCompositionDepth)
	}
	slots := make(map[string][]UINode)
	for _, entry := range children {
		var node UINode
		switch {
		case entry.Content != nil:
			node = lowerStaticContent(entry.Content)
		case entry.Component != "":
			var err error
			node, err = lowerComponentChild(entry, machineIdx, datasetIdx)
			if err != nil {
				return nil, err
			}
		case entry.View != "":
			if visited[entry.View] {
				return nil, fmt.Errorf("composable: cyclic composition detected at view %s (CMP-02 cyclic composition rejected)", entry.View)
			}
			v, ok := viewIdx[entry.View]
			if !ok {
				return nil, fmt.Errorf("composable: children entry names unknown view %q (CMP-03 unresolved reference rejected)", entry.View)
			}
			m, ok := machineIdx[v.MachineID]
			if !ok {
				return nil, fmt.Errorf("composable: view %s names unknown machine %q (CMP-03 unresolved reference rejected)", v.ID, v.MachineID)
			}
			node = lowerViewRefChild(m, v)
			if v.Type == model.ViewTypePage && len(v.Config.Children) > 0 {
				nextVisited := make(map[string]bool, len(visited)+1)
				for id := range visited {
					nextVisited[id] = true
				}
				nextVisited[v.ID] = true
				childSlots, err := LowerChildren(v.Config.Children, viewIdx, machineIdx, datasetIdx, nextVisited)
				if err != nil {
					return nil, err
				}
				node.Slots = childSlots
			}
		default:
			return nil, fmt.Errorf("composable: children entry names neither a view, content, nor component (CMP-04 slot/type mismatch rejected)")
		}
		if entry.Title != "" {
			if node.Properties == nil {
				node.Properties = map[string]string{}
			}
			node.Properties["title"] = entry.Title
		}
		slots[entry.Layout] = append(slots[entry.Layout], node)
	}
	return slots, nil
}

// LowerPage lowers m's own declared Views into a page-kind UINode. Each
// View becomes a view_ref child (lowerViewRefChild). A page-type View's own
// Config.Children compose the PAGE itself (model.go's own ViewTypePage doc
// comment), so they populate the whole page's own Slots. A detail-type
// View's own Children are different: CAP-V20's embedded-view mechanism
// (decision stepper, coordinate placement) embeds INTO that one Detail
// view's own render, not into the machine's whole page -- so they populate
// that view_ref node's own Slots instead (composable-runtime-roadmap.md
// 17c, closing the gap Phase 12's own benchmark work named: a Detail
// View's Children were silently never lowered at all). Two Detail Views
// on the same machine (or a Detail alongside a Page view) therefore never
// collide on one shared slot map, unlike hoisting both to page.Slots
// would. visited is seeded with the hosting View's own id so an immediate
// self-reference is caught on the very first check (CMP-02).
func LowerPage(m *model.Machine, viewIdx map[string]*model.View, machineIdx MachineIndex, datasetIdx DatasetIndex) (UINode, error) {
	page := UINode{
		Identity:   NodeIdentity{Kind: "page", Source: m.ID},
		Kind:       UINodePage,
		Properties: map[string]string{"machine_id": m.ID},
	}
	for _, v := range m.Views {
		node := lowerViewRefChild(m, v)
		if (v.Type == model.ViewTypePage || v.Type == model.ViewTypeDetail) && len(v.Config.Children) > 0 {
			slots, err := LowerChildren(v.Config.Children, viewIdx, machineIdx, datasetIdx, map[string]bool{v.ID: true})
			if err != nil {
				return UINode{}, fmt.Errorf("composable: page %s: %w", m.ID, err)
			}
			if v.Type == model.ViewTypePage {
				page.Slots = slots
			} else {
				node.Slots = slots
			}
		}
		page.Children = append(page.Children, node)
	}
	return page, nil
}
