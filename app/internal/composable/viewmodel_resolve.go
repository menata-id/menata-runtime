package composable

import (
	"fmt"
	"strings"

	"menata.id/app/internal/expr"
	"menata.id/app/internal/model"
)

func findField(m *model.Machine, fieldID string) *model.Field {
	for _, f := range m.Fields {
		if f.ID == fieldID {
			return f
		}
	}
	return nil
}

// ResolveFieldValue resolves fieldID's own value on record (the same
// map[string]any shape internal/store.Record.Data uses) into a
// renderer-neutral FieldValue.
//
// Deliberately narrow, named honestly rather than silently promising
// more: FieldTypeReference/FieldTypeUser/FieldTypeGroup resolve to their
// RAW STORED id, not a dereferenced display label -- full label
// resolution is internal/handler's own displayLabel, which needs a
// SECOND store lookup this package doesn't perform (consistent with
// Phase 2's own RelationRef, which only NAMES a relationship, never
// traverses it). FieldTypeComputed resolves only when Options.Expression
// is set (CAP-C13's general form), via internal/expr.Eval -- the same
// I/O-free, pure evaluator internal/handler already uses, reused
// directly rather than reimplemented; the older SourceField/Factor sugar
// (no Expression) returns an error instead of a silently wrong answer.
func ResolveFieldValue(m *model.Machine, fieldID string, record map[string]any) (FieldValue, error) {
	field := findField(m, fieldID)
	if field == nil {
		return FieldValue{}, fmt.Errorf("composable: machine %s has no field %s", m.ID, fieldID)
	}

	if field.Type == model.FieldTypeComputed {
		if field.Options.Expression == "" {
			return FieldValue{}, fmt.Errorf("composable: field %s is computed via SourceField/Factor sugar, not a declared Expression -- not resolved here", fieldID)
		}
		out, err := expr.Eval(field.Options.Expression, expr.Vars{Record: record})
		if err != nil {
			return FieldValue{}, fmt.Errorf("composable: evaluate field %s: %w", fieldID, err)
		}
		return FieldValue{Kind: FieldValueExpression, Display: fmt.Sprintf("%v", out)}, nil
	}

	raw, ok := record[fieldID]
	display := ""
	if ok && raw != nil {
		display = fmt.Sprintf("%v", raw)
	}

	if field.Type == model.FieldTypeReference {
		return FieldValue{Kind: FieldValueRelation, Display: display}, nil
	}
	return FieldValue{Kind: FieldValueField, Display: display}, nil
}

// ResolveRecordSummary builds a RecordSummaryCard component's own
// RecordSummary from record -- fails loud if node isn't that ComponentType.
//
// composable-runtime-roadmap.md 17e: subtitle_field's own value may name
// several field ids, comma-separated (LowerCardRowComponent's own way of
// excluding whichever column CardBadgeField claims) -- each resolves
// independently and their non-empty Display values join with " · ",
// matching internal/ui/list.templ's own cardSummary exactly. A single id
// (no comma, every pre-17e caller) resolves byte-identically to before:
// the same one FieldValue, unjoined.
func ResolveRecordSummary(m *model.Machine, node UINode, recordID string, record map[string]any) (RecordSummary, error) {
	if node.ComponentType != ComponentRecordSummaryCard {
		return RecordSummary{}, fmt.Errorf("composable: ResolveRecordSummary: node is a %q component, want %q", node.ComponentType, ComponentRecordSummaryCard)
	}
	title, err := ResolveFieldValue(m, node.Properties["title_field"], record)
	if err != nil {
		return RecordSummary{}, err
	}
	summary := RecordSummary{RecordID: recordID, Title: title}
	if subtitleField := node.Properties["subtitle_field"]; subtitleField != "" {
		fieldIDs := strings.Split(subtitleField, ",")
		if len(fieldIDs) == 1 {
			subtitle, err := ResolveFieldValue(m, fieldIDs[0], record)
			if err != nil {
				return RecordSummary{}, err
			}
			summary.Subtitle = subtitle
		} else {
			var parts []string
			for _, fieldID := range fieldIDs {
				fv, err := ResolveFieldValue(m, fieldID, record)
				if err != nil {
					return RecordSummary{}, err
				}
				if fv.Display != "" {
					parts = append(parts, fv.Display)
				}
			}
			summary.Subtitle = FieldValue{Kind: FieldValueField, Display: strings.Join(parts, " · ")}
		}
	}
	return summary, nil
}

// ResolveCollectionItem builds one row from a Collection component's own
// Dataset.Projection.Fields, in order -- fails loud if node isn't that
// ComponentType or carries no Dataset.
func ResolveCollectionItem(m *model.Machine, node UINode, recordID string, record map[string]any) (CollectionItem, error) {
	if node.ComponentType != ComponentCollection {
		return CollectionItem{}, fmt.Errorf("composable: ResolveCollectionItem: node is a %q component, want %q", node.ComponentType, ComponentCollection)
	}
	if node.Dataset == nil {
		return CollectionItem{}, fmt.Errorf("composable: ResolveCollectionItem: node has no Dataset")
	}
	item := CollectionItem{RecordID: recordID}
	for _, fieldID := range node.Dataset.Projection.Fields {
		fv, err := ResolveFieldValue(m, fieldID, record)
		if err != nil {
			return CollectionItem{}, err
		}
		item.Cells = append(item.Cells, fv)
	}
	return item, nil
}

// ResolveStatusValue resolves a StatusBadge component's own declared
// field -- fails loud if node isn't that ComponentType.
func ResolveStatusValue(m *model.Machine, node UINode, record map[string]any) (StatusValue, error) {
	if node.ComponentType != ComponentStatusBadge {
		return StatusValue{}, fmt.Errorf("composable: ResolveStatusValue: node is a %q component, want %q", node.ComponentType, ComponentStatusBadge)
	}
	fv, err := ResolveFieldValue(m, node.Properties["field"], record)
	if err != nil {
		return StatusValue{}, err
	}
	return StatusValue(fv), nil
}

// ResolveMetricValue formats an already-computed number for display --
// this package performs no aggregation itself; value comes from wherever
// physical execution (Phase 9's own aggregate-pushdown-aware
// RecordStore.CountGroupedBy/SumFieldsGroupedBy/SumField) actually ran.
func ResolveMetricValue(node UINode, label string, value float64) MetricValue {
	return MetricValue{Label: label, Value: fmt.Sprintf("%g", value)}
}

// ResolveActionSet resolves an ActionBar component's own declared Actions
// (Event ids) into real {EventID, Label} pairs. Re-verifies each id
// exists on m (LowerActionBar already checks this when it BUILDS the
// node, but a hand-built UINode could bypass that) -- fail-loud rather
// than a label-less action. Does NOT apply CAP-P02 ownership/permission
// filtering -- that is internal/handler.PermittedEventsForRecord's own
// job, needing live session data this package has never touched.
func ResolveActionSet(m *model.Machine, node UINode, recordID string) (ActionSet, error) {
	if node.ComponentType != ComponentActionBar {
		return ActionSet{}, fmt.Errorf("composable: ResolveActionSet: node is a %q component, want %q", node.ComponentType, ComponentActionBar)
	}
	set := ActionSet{RecordID: recordID}
	for _, eventID := range node.Actions {
		var event *model.Event
		for _, e := range m.Events {
			if e.ID == eventID {
				event = e
				break
			}
		}
		if event == nil {
			return ActionSet{}, fmt.Errorf("composable: ResolveActionSet: machine %s has no event %s", m.ID, eventID)
		}
		set.Actions = append(set.Actions, ActionValue{EventID: event.ID, Label: event.Name})
	}
	return set, nil
}
