package composable_test

import (
	"strings"
	"testing"

	"menata.id/app/internal/composable"
	"menata.id/app/internal/model"
	"menata.id/app/internal/testing/builders"
)

func TestResolveFieldValue_PlainField(t *testing.T) {
	m := builders.Machine("mch_task").
		WithField(builders.Field("fld_title", model.FieldTypeText).Build()).
		Build()
	fv, err := composable.ResolveFieldValue(m, "fld_title", map[string]any{"fld_title": "Write proposal"})
	if err != nil {
		t.Fatalf("ResolveFieldValue: %v", err)
	}
	if fv.Kind != composable.FieldValueField || fv.Display != "Write proposal" {
		t.Errorf("fv = %+v, want {field Write proposal}", fv)
	}
}

func TestResolveFieldValue_ReferenceField(t *testing.T) {
	m := builders.Machine("mch_step").
		WithField(builders.Field("fld_document", model.FieldTypeReference).
			Options(model.FieldOptions{TargetMachine: "mch_document"}).Build()).
		Build()
	fv, err := composable.ResolveFieldValue(m, "fld_document", map[string]any{"fld_document": "doc-123"})
	if err != nil {
		t.Fatalf("ResolveFieldValue: %v", err)
	}
	if fv.Kind != composable.FieldValueRelation || fv.Display != "doc-123" {
		t.Errorf("fv = %+v, want {relation doc-123}", fv)
	}
}

func TestResolveFieldValue_ComputedWithExpression(t *testing.T) {
	m := builders.Machine("mch_order").
		WithField(builders.Field("fld_qty", model.FieldTypeNumber).Build()).
		WithField(builders.Field("fld_total", model.FieldTypeComputed).
			Options(model.FieldOptions{Expression: "double(record.fld_qty) * 2.0"}).Build()).
		Build()
	fv, err := composable.ResolveFieldValue(m, "fld_total", map[string]any{"fld_qty": "3"})
	if err != nil {
		t.Fatalf("ResolveFieldValue: %v", err)
	}
	if fv.Kind != composable.FieldValueExpression || fv.Display != "6" {
		t.Errorf("fv = %+v, want {expression 6}", fv)
	}
}

func TestResolveFieldValue_ComputedWithoutExpressionErrors(t *testing.T) {
	m := builders.Machine("mch_order").
		WithField(builders.Field("fld_total", model.FieldTypeComputed).
			Options(model.FieldOptions{SourceField: "fld_qty", Factor: 2}).Build()).
		Build()
	if _, err := composable.ResolveFieldValue(m, "fld_total", map[string]any{"fld_qty": "3"}); err == nil {
		t.Fatal("want error for a computed field with no declared Expression (SourceField/Factor sugar not resolved here)")
	}
}

func TestResolveRecordSummary_WrongComponentType(t *testing.T) {
	m := builders.Machine("mch_x").Build()
	node := composable.UINode{ComponentType: composable.ComponentText}
	if _, err := composable.ResolveRecordSummary(m, node, "r1", nil); err == nil {
		t.Fatal("want error for a non-RecordSummaryCard node")
	}
}

func TestResolveCollectionItem_OrdersCellsByProjection(t *testing.T) {
	m := builders.Machine("mch_task").
		WithField(builders.Field("fld_a", model.FieldTypeText).Build()).
		WithField(builders.Field("fld_b", model.FieldTypeText).Build()).
		Build()
	ds := composable.Dataset{Projection: composable.Projection{Fields: []string{"fld_b", "fld_a"}}}
	node := composable.UINode{ComponentType: composable.ComponentCollection, Dataset: &ds}

	item, err := composable.ResolveCollectionItem(m, node, "r1", map[string]any{"fld_a": "A", "fld_b": "B"})
	if err != nil {
		t.Fatalf("ResolveCollectionItem: %v", err)
	}
	if len(item.Cells) != 2 || item.Cells[0].Display != "B" || item.Cells[1].Display != "A" {
		t.Errorf("Cells = %+v, want [B A]", item.Cells)
	}
}

func TestResolveStatusValue_WrongComponentType(t *testing.T) {
	m := builders.Machine("mch_x").Build()
	node := composable.UINode{ComponentType: composable.ComponentMetric}
	if _, err := composable.ResolveStatusValue(m, node, nil); err == nil {
		t.Fatal("want error for a non-StatusBadge node")
	}
}

func TestResolveActionSet_RejectsUnknownEvent(t *testing.T) {
	m := builders.Machine("mch_task").
		WithEvent(builders.Event("evt_submit", "Submit").Build()).
		Build()
	node := composable.UINode{ComponentType: composable.ComponentActionBar, Actions: []string{"evt_missing"}}
	if _, err := composable.ResolveActionSet(m, node, "r1"); err == nil {
		t.Fatal("want error for an action naming an event not on the machine")
	}
}

func TestResolveActionSet_ResolvesLabels(t *testing.T) {
	m := builders.Machine("mch_task").
		WithEvent(builders.Event("evt_submit", "Submit").Build()).
		Build()
	node := composable.UINode{ComponentType: composable.ComponentActionBar, Actions: []string{"evt_submit"}}
	set, err := composable.ResolveActionSet(m, node, "r1")
	if err != nil {
		t.Fatalf("ResolveActionSet: %v", err)
	}
	if len(set.Actions) != 1 || set.Actions[0].Label != "Submit" {
		t.Errorf("Actions = %+v, want [{evt_submit Submit}]", set.Actions)
	}
}

func TestResolveMetricValue(t *testing.T) {
	node := composable.UINode{ComponentType: composable.ComponentMetric}
	mv := composable.ResolveMetricValue(node, "Total Projects", 42)
	if mv.Label != "Total Projects" || !strings.Contains(mv.Value, "42") {
		t.Errorf("mv = %+v, want Label=Total Projects Value containing 42", mv)
	}
}
