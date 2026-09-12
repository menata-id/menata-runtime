package composable_test

import (
	"context"
	"testing"

	"menata.id/app/internal/composable"
	"menata.id/app/internal/metadata"
	"menata.id/app/internal/model"
	"menata.id/app/internal/testing/testdb"
)

// TestLowerViewToComponentAgainstApprovalCase proves List lowering in both
// render modes against real seeded metadata: vw_as_progress (table,
// seeds/004_approval.sql) and vw_ad_pending (cards, seeds/050_composed_
// dashboard.sql). Requires DATABASE_URL seeded with 001+004+050.
func TestLowerViewToComponentAgainstApprovalCase(t *testing.T) {
	pool := testdb.Connect(t)
	workspaces, err := metadata.NewLoader(pool).LoadAll(context.Background())
	if err != nil {
		t.Fatalf("LoadAll: %v", err)
	}
	app := findApplication(t, workspaces, "ws_default", "app_approval")

	step := findMachineByID(t, app, "mch_approval_step")
	progress := findViewByID(t, step, "vw_as_progress")
	node, err := composable.LowerViewToComponent(step, progress)
	if err != nil {
		t.Fatalf("LowerViewToComponent(vw_as_progress): %v", err)
	}
	if node.Properties["display"] != "" {
		t.Errorf("vw_as_progress display = %q, want empty (table)", node.Properties["display"])
	}

	doc := findMachineByID(t, app, "mch_approval_document")
	pending := findViewByID(t, doc, "vw_ad_pending")
	node, err = composable.LowerViewToComponent(doc, pending)
	if err != nil {
		t.Fatalf("LowerViewToComponent(vw_ad_pending): %v", err)
	}
	if node.Properties["display"] != "cards" {
		t.Errorf("vw_ad_pending display = %q, want %q", node.Properties["display"], "cards")
	}
	if _, err := composable.LowerCardRowComponent(doc, pending, *node.Dataset); err != nil {
		t.Errorf("LowerCardRowComponent(vw_ad_pending): %v", err)
	}
}

// TestLowerViewToComponentAgainstKanbanLab proves Board lowering against
// the same explicitly-flagged Case-19 stand-in Phases 1-5 already
// established.
func TestLowerViewToComponentAgainstKanbanLab(t *testing.T) {
	pool := testdb.Connect(t)
	workspaces, err := metadata.NewLoader(pool).LoadAll(context.Background())
	if err != nil {
		t.Fatalf("LoadAll: %v", err)
	}
	app := findApplication(t, workspaces, "ws_default", "app_kanban_lab")
	m := findMachineByID(t, app, "mch_kanban_task")
	v := findViewByID(t, m, "vw_kbt_board")

	node, err := composable.LowerViewToComponent(m, v)
	if err != nil {
		t.Fatalf("LowerViewToComponent(vw_kbt_board): %v", err)
	}
	if node.Properties["display"] != "board" {
		t.Errorf("display = %q, want %q", node.Properties["display"], "board")
	}
	if len(node.Dataset.GroupBy) == 0 {
		t.Error("Dataset.GroupBy is empty, want the board's own lane field")
	}
}

// TestLowerViewToComponentAgainstActionLabCalendar proves the real Date
// Dimension gap this phase closes: seeds/010_views_lab.sql's vw_alt_calendar
// declares date_field:"fld_alt_follow_up" -- BuildDatasetFromView must now
// fold it into GroupBy. Requires DATABASE_URL seeded with 001+009+010.
func TestLowerViewToComponentAgainstActionLabCalendar(t *testing.T) {
	pool := testdb.Connect(t)
	workspaces, err := metadata.NewLoader(pool).LoadAll(context.Background())
	if err != nil {
		t.Fatalf("LoadAll: %v", err)
	}
	app := findApplication(t, workspaces, "ws_default", "app_action_lab")
	m := findMachineByID(t, app, "mch_al_task")
	v := findViewByID(t, m, "vw_alt_calendar")

	node, err := composable.LowerViewToComponent(m, v)
	if err != nil {
		t.Fatalf("LowerViewToComponent(vw_alt_calendar): %v", err)
	}
	if node.Properties["display"] != "calendar" {
		t.Errorf("display = %q, want %q", node.Properties["display"], "calendar")
	}
	found := false
	for _, f := range node.Dataset.GroupBy {
		if f == "fld_alt_follow_up" {
			found = true
		}
	}
	if !found {
		t.Errorf("Dataset.GroupBy = %v, want it to contain fld_alt_follow_up (the Date Dimension)", node.Dataset.GroupBy)
	}
}

// TestLowerDashboardViewAgainstActionLab proves the Metric/Collection
// dispatch against seeds/010_views_lab.sql's own real vw_alp_dashboard --
// two real sections, one grouped, one not.
func TestLowerDashboardViewAgainstActionLab(t *testing.T) {
	pool := testdb.Connect(t)
	workspaces, err := metadata.NewLoader(pool).LoadAll(context.Background())
	if err != nil {
		t.Fatalf("LoadAll: %v", err)
	}
	app := findApplication(t, workspaces, "ws_default", "app_action_lab")
	idx := composable.IndexMachines(app)
	m := findMachineByID(t, app, "mch_al_project")
	v := findViewByID(t, m, "vw_alp_dashboard")

	layout, err := composable.LowerDashboardView(idx, v)
	if err != nil {
		t.Fatalf("LowerDashboardView(vw_alp_dashboard): %v", err)
	}
	if len(layout.Children) != 2 {
		t.Fatalf("len(Children) = %d, want 2", len(layout.Children))
	}

	var sawMetric, sawCollection bool
	for _, c := range layout.Children {
		switch c.ComponentType {
		case composable.ComponentMetric:
			sawMetric = true
		case composable.ComponentCollection:
			sawCollection = true
		}
	}
	if !sawMetric {
		t.Error("no Metric child -- want one for the ungrouped 'Projects' section")
	}
	if !sawCollection {
		t.Error("no Collection child -- want one for the grouped 'Tasks by Stage' section")
	}
}

func findViewByID(t *testing.T, m *model.Machine, id string) *model.View {
	t.Helper()
	for _, v := range m.Views {
		if v.ID == id {
			return v
		}
	}
	t.Fatalf("machine %s: missing expected view %s", m.ID, id)
	return nil
}
