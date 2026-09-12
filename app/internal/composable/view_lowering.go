package composable

import (
	"strings"

	"menata.id/app/internal/model"
)

// CardBadgeField names which of columns should render as a card's status
// badge -- the LAST value_list-typed column among the non-title columns,
// exactly matching the real CAP-V02 Tier 2 cards convention
// (internal/ui/list.templ's own cardSummary: "the LAST badge-eligible
// column, not the first" -- caught live 2026-09-10, since an app's cards
// Views always put Status last but an earlier value_list column, e.g.
// Document Type, is merely descriptive text). Returns "" when no column
// is value_list-typed -- not every card has a badge.
func CardBadgeField(m *model.Machine, columns []string) string {
	badge := ""
	for i := 1; i < len(columns); i++ {
		if f := findField(m, columns[i]); f != nil && f.Type == model.FieldTypeValueList {
			badge = columns[i]
		}
	}
	return badge
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
// Collection's own rows use (v.Config.Display == "cards") -- proves the
// two components compose correctly, as a sibling fact next to the
// Collection LowerViewToComponent already returns. Row-shape composition
// stays a render-time concern, not an authored-metadata Children entry --
// both Phase 5 contracts declare AllowsChildren:false.
//
// composable-runtime-roadmap.md 17e: the first column is always the
// title (same heuristic as before); CardBadgeField's own column is
// excluded from the subtitle rather than shown twice, and every OTHER
// non-title column is joined into subtitle_field's own value as a
// comma-separated list of field ids -- ResolveRecordSummary (viewmodel_
// resolve.go) splits and joins their resolved Display values with " · ",
// matching cardSummary's own multi-column subtitle exactly. A single id
// (no comma) resolves byte-identically to before this change.
func LowerCardRowComponent(m *model.Machine, v *model.View, ds Dataset) (UINode, error) {
	columns := v.Config.Columns
	var title string
	if len(columns) > 0 {
		title = columns[0]
	}
	badge := CardBadgeField(m, columns)
	var subtitleFields []string
	for i := 1; i < len(columns); i++ {
		if columns[i] != badge {
			subtitleFields = append(subtitleFields, columns[i])
		}
	}
	return LowerRecordSummaryCard(ds, title, strings.Join(subtitleFields, ","))
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
