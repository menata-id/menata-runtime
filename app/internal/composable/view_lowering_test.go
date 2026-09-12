package composable_test

import (
	"reflect"
	"testing"

	"menata.id/app/internal/composable"
	"menata.id/app/internal/model"
	"menata.id/app/internal/testing/builders"
)

func TestLowerViewToComponent_List(t *testing.T) {
	m := builders.Machine("mch_task").Build()
	v := builders.View("vw_list", model.ViewTypeList).Columns("fld_title", "fld_status").Build()

	node, err := composable.LowerViewToComponent(m, v)
	if err != nil {
		t.Fatalf("LowerViewToComponent: %v", err)
	}
	if node.ComponentType != composable.ComponentCollection {
		t.Errorf("ComponentType = %q, want %q", node.ComponentType, composable.ComponentCollection)
	}
	if node.Properties["display"] != "" {
		t.Errorf("Properties[display] = %q, want empty (table)", node.Properties["display"])
	}
	if node.Dataset == nil {
		t.Fatal("Dataset = nil")
	}
}

func TestLowerViewToComponent_CardsDisplay(t *testing.T) {
	m := builders.Machine("mch_task").Build()
	v := builders.View("vw_pending", model.ViewTypeList).Columns("fld_title", "fld_status").Display("cards").Build()

	node, err := composable.LowerViewToComponent(m, v)
	if err != nil {
		t.Fatalf("LowerViewToComponent: %v", err)
	}
	if node.Properties["display"] != "cards" {
		t.Errorf("Properties[display] = %q, want %q", node.Properties["display"], "cards")
	}

	card, err := composable.LowerCardRowComponent(v, *node.Dataset)
	if err != nil {
		t.Fatalf("LowerCardRowComponent: %v", err)
	}
	if card.Properties["title_field"] != "fld_title" || card.Properties["subtitle_field"] != "fld_status" {
		t.Errorf("card Properties = %+v, want title_field=fld_title subtitle_field=fld_status", card.Properties)
	}
}

func TestLowerViewToComponent_Board(t *testing.T) {
	m := builders.Machine("mch_task").Build()
	v := builders.View("vw_board", model.ViewTypeBoard).Columns("fld_title").GroupField("fld_status").Build()

	node, err := composable.LowerViewToComponent(m, v)
	if err != nil {
		t.Fatalf("LowerViewToComponent: %v", err)
	}
	if node.Properties["display"] != "board" {
		t.Errorf("Properties[display] = %q, want %q", node.Properties["display"], "board")
	}
	if want := []string{"fld_status"}; !reflect.DeepEqual(node.Dataset.GroupBy, want) {
		t.Errorf("Dataset.GroupBy = %v, want %v", node.Dataset.GroupBy, want)
	}
}

func TestLowerViewToComponent_Calendar(t *testing.T) {
	m := builders.Machine("mch_task").Build()
	v := builders.View("vw_calendar", model.ViewTypeCalendar).Columns("fld_title").Build()
	v.Config.DateField = "fld_due"

	node, err := composable.LowerViewToComponent(m, v)
	if err != nil {
		t.Fatalf("LowerViewToComponent: %v", err)
	}
	if node.Properties["display"] != "calendar" {
		t.Errorf("Properties[display] = %q, want %q", node.Properties["display"], "calendar")
	}
	if want := []string{"fld_due"}; !reflect.DeepEqual(node.Dataset.GroupBy, want) {
		t.Errorf("Dataset.GroupBy = %v, want %v (the Date Dimension)", node.Dataset.GroupBy, want)
	}
}

func TestLowerDashboardView_DispatchesMetricVsCollection(t *testing.T) {
	idx := composable.MachineIndex{
		"mch_task":    builders.Machine("mch_task").Build(),
		"mch_project": builders.Machine("mch_project").Build(),
	}
	v := builders.View("vw_dash", model.ViewTypeDashboard).Build()
	v.Config.Sections = []model.DashboardSection{
		{Title: "Projects", Machine: "mch_project"},
		{Title: "Tasks by Stage", Machine: "mch_task", GroupField: "fld_stage"},
	}

	layout, err := composable.LowerDashboardView(idx, v)
	if err != nil {
		t.Fatalf("LowerDashboardView: %v", err)
	}
	if layout.Kind != composable.UINodeLayout {
		t.Fatalf("Kind = %q, want %q", layout.Kind, composable.UINodeLayout)
	}
	if len(layout.Children) != 2 {
		t.Fatalf("len(Children) = %d, want 2", len(layout.Children))
	}
	if layout.Children[0].ComponentType != composable.ComponentMetric {
		t.Errorf("Children[0].ComponentType = %q, want %q (ungrouped section)", layout.Children[0].ComponentType, composable.ComponentMetric)
	}
	if layout.Children[1].ComponentType != composable.ComponentCollection {
		t.Errorf("Children[1].ComponentType = %q, want %q (grouped section)", layout.Children[1].ComponentType, composable.ComponentCollection)
	}
}
