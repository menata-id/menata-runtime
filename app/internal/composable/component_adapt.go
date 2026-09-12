package composable

import (
	"fmt"

	"menata.id/app/internal/model"
)

// LowerHeading resolves a Heading component from a static heading text --
// model.PageContent{Type:"heading"}'s own real input, no live seed uses it
// yet (see ui.go's lowerStaticContent, which only migrates "heading" and
// "text" through the component contract; "button"/"image" stay plain
// static_content nodes).
func LowerHeading(text string) (UINode, error) {
	return ResolveComponent(ComponentHeading, map[string]string{"text": text}, nil, nil)
}

// LowerText resolves a Text component from static body text --
// model.PageContent{Type:"text"}'s own real input (seeds/050_composed_
// dashboard.sql's "Recent Activity" aside).
func LowerText(text string) (UINode, error) {
	return ResolveComponent(ComponentText, map[string]string{"text": text}, nil, nil)
}

// LowerCollection resolves a Collection component wrapping ds -- a plain
// row-projection Dataset (a list/board/calendar/timeline's own, per
// BuildDatasetFromView).
func LowerCollection(ds Dataset) (UINode, error) {
	return ResolveComponent(ComponentCollection, nil, &ds, nil)
}

// LowerRecordSummaryCard resolves a RecordSummaryCard component -- ds is
// the underlying row source (a CAP-V02 Tier 2 Display:"cards" list's own
// Dataset, e.g. seeds/050's vw_ad_pending); titleField/subtitleField name
// which projected fields the card shows (subtitleField optional).
func LowerRecordSummaryCard(ds Dataset, titleField, subtitleField string) (UINode, error) {
	props := map[string]string{"title_field": titleField}
	if subtitleField != "" {
		props["subtitle_field"] = subtitleField
	}
	return ResolveComponent(ComponentRecordSummaryCard, props, &ds, nil)
}

// LowerMetric resolves a Metric component from ds, enforcing a semantic
// rule beyond the generic contract: ds must have exactly one Measure and
// no GroupBy -- a grouped Dataset is a breakdown (DashboardTile's own
// Total/Breakdown distinction, internal/ui/types.go), not a single Metric.
func LowerMetric(ds Dataset) (UINode, error) {
	if len(ds.GroupBy) > 0 {
		return UINode{}, fmt.Errorf("composable: Metric requires an ungrouped Dataset, got GroupBy %v (a grouped Dataset is a breakdown, not a single metric)", ds.GroupBy)
	}
	if len(ds.Measures) != 1 {
		return UINode{}, fmt.Errorf("composable: Metric requires exactly one Measure, got %d", len(ds.Measures))
	}
	return ResolveComponent(ComponentMetric, nil, &ds, nil)
}

// LowerStatusBadge resolves a StatusBadge component for fieldID on m,
// checking fieldID names a real, value_list-typed Field -- fail-loud
// (CAP-X05), not a silently-wrong badge.
func LowerStatusBadge(m *model.Machine, fieldID string) (UINode, error) {
	var field *model.Field
	for _, f := range m.Fields {
		if f.ID == fieldID {
			field = f
			break
		}
	}
	if field == nil {
		return UINode{}, fmt.Errorf("composable: StatusBadge: machine %s has no field %s", m.ID, fieldID)
	}
	if field.Type != model.FieldTypeValueList {
		return UINode{}, fmt.Errorf("composable: StatusBadge: field %s on machine %s is type %q, want value_list", fieldID, m.ID, field.Type)
	}
	return ResolveComponent(ComponentStatusBadge, map[string]string{"field": fieldID}, nil, nil)
}

// LowerActionBar resolves an ActionBar component for eventIDs on m,
// checking every id names a real Event -- Actions is never arbitrary
// strings, only verified Event ids (the concrete answer to "no arbitrary
// component code" from §9's own "Do not do" list).
func LowerActionBar(m *model.Machine, eventIDs []string) (UINode, error) {
	known := make(map[string]bool, len(m.Events))
	for _, e := range m.Events {
		known[e.ID] = true
	}
	for _, id := range eventIDs {
		if !known[id] {
			return UINode{}, fmt.Errorf("composable: ActionBar: machine %s has no event %s", m.ID, id)
		}
	}
	return ResolveComponent(ComponentActionBar, nil, nil, eventIDs)
}
