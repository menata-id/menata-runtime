package composable_test

import (
	"context"
	"strings"
	"testing"

	"menata.id/app/internal/composable"
	"menata.id/app/internal/metadata"
	"menata.id/app/internal/testing/testdb"
)

// TestBuildPhysicalPlanAgainstApprovalCase proves the baseline cases
// against real seeded metadata: vw_ad_form (no filter/sort/measures),
// vw_ad_all (sort only), and vw_ad_pending (a real safe equals filter,
// seeds/050_composed_dashboard.sql). Requires DATABASE_URL seeded with
// 001+004+050.
func TestBuildPhysicalPlanAgainstApprovalCase(t *testing.T) {
	pool := testdb.Connect(t)
	workspaces, err := metadata.NewLoader(pool).LoadAll(context.Background())
	if err != nil {
		t.Fatalf("LoadAll: %v", err)
	}
	app := findApplication(t, workspaces, "ws_default", "app_approval")
	m := findMachineByID(t, app, "mch_approval_document")

	form := findViewByID(t, m, "vw_ad_form")
	formDS, err := composable.BuildDatasetFromView(m, form)
	if err != nil {
		t.Fatalf("BuildDatasetFromView(vw_ad_form): %v", err)
	}
	formPlan := composable.BuildPhysicalPlan(composable.DependencyNode{Dataset: formDS})
	if !formPlan.FilterPushdown {
		t.Error("vw_ad_form: FilterPushdown = false, want true (trivial, no filter)")
	}

	all := findViewByID(t, m, "vw_ad_all")
	allDS, err := composable.BuildDatasetFromView(m, all)
	if err != nil {
		t.Fatalf("BuildDatasetFromView(vw_ad_all): %v", err)
	}
	allPlan := composable.BuildPhysicalPlan(composable.DependencyNode{Dataset: allDS})
	if !allPlan.SortPushdown {
		t.Error("vw_ad_all: SortPushdown = false, want true (real default_sort)")
	}

	pending := findViewByID(t, m, "vw_ad_pending")
	pendingDS, err := composable.BuildDatasetFromView(m, pending)
	if err != nil {
		t.Fatalf("BuildDatasetFromView(vw_ad_pending): %v", err)
	}
	pendingPlan := composable.BuildPhysicalPlan(composable.DependencyNode{Dataset: pendingDS})
	if !pendingPlan.FilterPushdown {
		t.Error("vw_ad_pending: FilterPushdown = false, want true (a real equals filter)")
	}
}

// TestBuildPhysicalPlanAgainstViewsLabOverdueTasks proves the fail-loud
// mixed-filter case against a real seeded View: seeds/010_views_lab.sql's
// vw_vlt_list ("My Overdue Tasks") declares BOTH a safe equals filter
// (assignee) and an unsafe before filter (due date) -- FilterPushdown
// must be false, naming "before"; the same View's own real default_sort
// still gives SortPushdown: true. Requires DATABASE_URL seeded with
// 001+010.
func TestBuildPhysicalPlanAgainstViewsLabOverdueTasks(t *testing.T) {
	pool := testdb.Connect(t)
	workspaces, err := metadata.NewLoader(pool).LoadAll(context.Background())
	if err != nil {
		t.Fatalf("LoadAll: %v", err)
	}
	app := findApplication(t, workspaces, "ws_default", "app_views_lab")
	m := findMachineByID(t, app, "mch_vl_task")
	v := findViewByID(t, m, "vw_vlt_list")

	ds, err := composable.BuildDatasetFromView(m, v)
	if err != nil {
		t.Fatalf("BuildDatasetFromView(vw_vlt_list): %v", err)
	}
	plan := composable.BuildPhysicalPlan(composable.DependencyNode{Dataset: ds})

	if plan.FilterPushdown {
		t.Error("FilterPushdown = true, want false (the 'before' filter isn't pushdown-safe)")
	}
	if !plan.SortPushdown {
		t.Error("SortPushdown = false, want true (a real default_sort)")
	}
	found := false
	for _, n := range plan.Notes {
		if strings.Contains(n, "before") {
			found = true
		}
	}
	if !found {
		t.Errorf("Notes = %v, want one naming the 'before' operator", plan.Notes)
	}
}
