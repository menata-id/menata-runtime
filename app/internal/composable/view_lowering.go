package composable

import "menata.id/app/internal/model"

// firstTwoColumns is the heuristic card-mode Collection lowering uses when
// metadata names no explicit title/subtitle field -- same "prototype-
// honest heuristic, graceful degrade" posture as displayLabel/initials()
// elsewhere in this codebase: the first declared column is the title, the
// second (if any) the subtitle.
func firstTwoColumns(columns []string) (title, subtitle string) {
	if len(columns) > 0 {
		title = columns[0]
	}
	if len(columns) > 1 {
		subtitle = columns[1]
	}
	return title, subtitle
}

// LowerViewToComponent lowers v (List/Board/Calendar/Timeline) into its
// Phase 6 Collection-shaped equivalent, per composable-runtime-roadmap.md
// §10's own "Initial lowering targets" table -- an ALTERNATE path
// alongside lowerViewRefChild (view_ref), not a replacement:
// LowerPage/LowerChildren still produce view_ref nodes unchanged, per
// §10's own compatibility rule ("existing View handlers may remain behind
// the lowering layer until equivalence is proven").
//
// display is a Component-level property naming the render mode -- List
// carries Config.Display verbatim ("" table, "cards" card list); Board,
// Calendar, and Timeline each get their own fixed display name. It is
// never folded into the underlying Dataset (Phase 2's own boundary rule).
func LowerViewToComponent(m *model.Machine, v *model.View) (UINode, error) {
	ds, err := BuildDatasetFromView(m, v)
	if err != nil {
		return UINode{}, err
	}

	var display string
	switch v.Type {
	case model.ViewTypeList:
		display = v.Config.Display
	case model.ViewTypeBoard:
		display = "board"
	case model.ViewTypeCalendar:
		display = "calendar"
	case model.ViewTypeTimeline:
		display = "timeline"
	}

	props := map[string]string{}
	if display != "" {
		props["display"] = display
	}
	return ResolveComponent(ComponentCollection, props, &ds, nil)
}

// LowerCardRowComponent resolves the RecordSummaryCard shape a card-mode
// Collection's own rows use (v.Config.Display == "cards"), via
// firstTwoColumns' heuristic -- proves the two components compose
// correctly, as a sibling fact next to the Collection LowerViewToComponent
// already returns. Row-shape composition stays a render-time concern, not
// an authored-metadata Children entry -- both Phase 5 contracts declare
// AllowsChildren:false.
func LowerCardRowComponent(v *model.View, ds Dataset) (UINode, error) {
	title, subtitle := firstTwoColumns(v.Config.Columns)
	return LowerRecordSummaryCard(ds, title, subtitle)
}

// LowerDashboardView lowers a "dashboard" View's own Sections into a
// layout-kind UINode -- LayoutNode's Phase 1 placeholder, graduated into
// UINodeLayout in Phase 4, gets its first real emitter here. One Metric or
// Collection child per section, dispatched by GroupBy exactly the way
// LowerMetric's own semantic rule already distinguishes a single total
// (Metric) from a breakdown (Collection, DashboardTile's own Total/
// Breakdown distinction, internal/ui/types.go).
func LowerDashboardView(idx MachineIndex, v *model.View) (UINode, error) {
	layout := UINode{
		Identity: NodeIdentity{Kind: "layout", Source: v.ID},
		Kind:     UINodeLayout,
	}
	for _, sec := range v.Config.Sections {
		ds, err := BuildDatasetFromDashboardSection(idx, sec)
		if err != nil {
			return UINode{}, err
		}
		var child UINode
		if len(ds.GroupBy) == 0 {
			child, err = LowerMetric(ds)
		} else {
			child, err = LowerCollection(ds)
		}
		if err != nil {
			return UINode{}, err
		}
		layout.Children = append(layout.Children, child)
	}
	return layout, nil
}
