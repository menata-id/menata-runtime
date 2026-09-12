package composable_test

import (
	"testing"

	"menata.id/app/internal/composable"
	"menata.id/app/internal/model"
	"menata.id/app/internal/testing/builders"
)

func TestLowerHeading(t *testing.T) {
	node, err := composable.LowerHeading("Section Title")
	if err != nil {
		t.Fatalf("LowerHeading: %v", err)
	}
	if node.ComponentType != composable.ComponentHeading {
		t.Errorf("ComponentType = %q, want %q", node.ComponentType, composable.ComponentHeading)
	}
	if node.Properties["text"] != "Section Title" {
		t.Errorf("Properties[text] = %q, want %q", node.Properties["text"], "Section Title")
	}
}

func TestLowerText(t *testing.T) {
	node, err := composable.LowerText("Body copy")
	if err != nil {
		t.Fatalf("LowerText: %v", err)
	}
	if node.ComponentType != composable.ComponentText {
		t.Errorf("ComponentType = %q, want %q", node.ComponentType, composable.ComponentText)
	}
}

func TestLowerStatusBadge_RejectsNonValueListField(t *testing.T) {
	m := builders.Machine("mch_task").
		WithField(builders.Field("fld_title", model.FieldTypeText).Build()).
		Build()
	if _, err := composable.LowerStatusBadge(m, "fld_title"); err == nil {
		t.Fatal("want error for a text field, StatusBadge requires value_list")
	}
}

func TestLowerStatusBadge_RejectsUnknownField(t *testing.T) {
	m := builders.Machine("mch_task").Build()
	if _, err := composable.LowerStatusBadge(m, "fld_missing"); err == nil {
		t.Fatal("want error for a field that doesn't exist on the machine")
	}
}

func TestLowerActionBar_RejectsUnknownEvent(t *testing.T) {
	m := builders.Machine("mch_task").
		WithEvent(builders.Event("evt_submit", "Submit").Build()).
		Build()
	if _, err := composable.LowerActionBar(m, []string{"evt_submit", "evt_missing"}); err == nil {
		t.Fatal("want error for an event id not declared on the machine")
	}
}

func TestLowerActionBar_Accepts(t *testing.T) {
	m := builders.Machine("mch_task").
		WithEvent(builders.Event("evt_submit", "Submit").Build()).
		Build()
	node, err := composable.LowerActionBar(m, []string{"evt_submit"})
	if err != nil {
		t.Fatalf("LowerActionBar: %v", err)
	}
	if len(node.Actions) != 1 || node.Actions[0] != "evt_submit" {
		t.Errorf("Actions = %v, want [evt_submit]", node.Actions)
	}
}

func TestLowerMetric_RejectsGroupedDataset(t *testing.T) {
	ds := composable.Dataset{
		Source:   composable.DataSource{MachineID: "mch_x"},
		GroupBy:  []string{"fld_stage"},
		Measures: []composable.Measure{{Kind: composable.MeasureCount}},
	}
	if _, err := composable.LowerMetric(ds); err == nil {
		t.Fatal("want error for a grouped Dataset -- that's a breakdown, not a single Metric")
	}
}

func TestLowerMetric_Accepts(t *testing.T) {
	ds := composable.Dataset{
		Source:   composable.DataSource{MachineID: "mch_x"},
		Measures: []composable.Measure{{Kind: composable.MeasureCount}},
	}
	if _, err := composable.LowerMetric(ds); err != nil {
		t.Fatalf("LowerMetric: %v", err)
	}
}

func TestLowerRecordSummaryCard(t *testing.T) {
	ds := composable.Dataset{Source: composable.DataSource{MachineID: "mch_x"}}
	node, err := composable.LowerRecordSummaryCard(ds, "fld_title", "fld_status")
	if err != nil {
		t.Fatalf("LowerRecordSummaryCard: %v", err)
	}
	if node.Properties["title_field"] != "fld_title" || node.Properties["subtitle_field"] != "fld_status" {
		t.Errorf("Properties = %+v, want title_field=fld_title subtitle_field=fld_status", node.Properties)
	}
}

func TestLowerCollection(t *testing.T) {
	ds := composable.Dataset{Source: composable.DataSource{MachineID: "mch_x"}}
	node, err := composable.LowerCollection(ds)
	if err != nil {
		t.Fatalf("LowerCollection: %v", err)
	}
	if node.Dataset == nil {
		t.Fatal("Dataset = nil, want the wrapped Dataset")
	}
}
