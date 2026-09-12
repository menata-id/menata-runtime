package composable_test

import (
	"context"
	"testing"

	"menata.id/app/internal/composable"
	"menata.id/app/internal/metadata"
	"menata.id/app/internal/testing/testdb"
)

// TestBenchmark_SecurityScopes is composable-runtime-roadmap.md §16's own
// scenario 7 (same logical request, different security scopes): reuses
// Phase 7's real HR/Staff split (seeds/012_permissions_lab.sql) over the
// same real List View, wrapped in MeasureComposition -- two separate
// DAGNodes despite being "the same logical request." Requires
// DATABASE_URL seeded with 001+012.
func TestBenchmark_SecurityScopes(t *testing.T) {
	pool := testdb.Connect(t)
	workspaces, err := metadata.NewLoader(pool).LoadAll(context.Background())
	if err != nil {
		t.Fatalf("LoadAll: %v", err)
	}
	app := findApplication(t, workspaces, "ws_default", "app_permissions_lab")
	m := findMachineByID(t, app, "mch_pl_employee")
	v := findViewByID(t, m, "vw_ple2_list")

	ds, err := composable.BuildDatasetFromView(m, v)
	if err != nil {
		t.Fatalf("BuildDatasetFromView: %v", err)
	}
	nodes := []composable.UINode{{Identity: composable.NodeIdentity{Source: v.ID}, Dataset: &ds}}

	hr := composable.MeasureComposition(nodes, composable.ResolveSecurityScope(m, "HR"))
	staff := composable.MeasureComposition(nodes, composable.ResolveSecurityScope(m, "Staff"))

	if hr.DAGNodes != 1 || staff.DAGNodes != 1 {
		t.Errorf("HR=%+v Staff=%+v, want DAGNodes=1 each (measured separately per scope)", hr, staff)
	}
	t.Logf("scenario 7 (same logical request, different security scopes): HR=%+v Staff=%+v -- separate DAGs per scope, never coalesced", hr, staff)
}

// TestBenchmark_ApprovalDetailAndStepper is scenario 9 (embedded
// dependency sharing). The roadmap's own literal example is Approval
// Step's Detail embedding Approval Document's decision stepper
// (seeds/037_decision_stepper_lab.sql + seeds/042_inline_view_
// composition.sql's own vw_as_detail, a DETAIL-type View carrying
// Config.Children). That can't be measured here: ui.go's LowerPage only
// ever calls LowerChildren when v.Type == model.ViewTypePage (grep
// confirms exactly two such guards, both page-only) -- it never lowers a
// detail-type View's own Children at all. This is a real, newly-found
// scope gap, named here rather than quietly worked around; fixing it is
// out of scope for a benchmark phase and belongs with whoever next
// touches View lowering.
//
// This scenario is instead measured against the already-working PAGE-type
// embedding Phases 4/7/8/11 already proved: seeds/050_composed_
// dashboard.sql's own vw_ad_page/vw_ad_pending pair, the closest real
// "embedded dependency sharing" proxy already in scope. Requires
// DATABASE_URL seeded with 001+004+050.
func TestBenchmark_ApprovalDetailAndStepper(t *testing.T) {
	pool := testdb.Connect(t)
	workspaces, err := metadata.NewLoader(pool).LoadAll(context.Background())
	if err != nil {
		t.Fatalf("LoadAll: %v", err)
	}
	app := findApplication(t, workspaces, "ws_default", "app_approval")
	viewIdx := composable.IndexViews(app)
	machineIdx := composable.IndexMachines(app)
	m := findMachineByID(t, app, "mch_approval_document")

	page, err := composable.LowerPage(m, viewIdx, machineIdx)
	if err != nil {
		t.Fatalf("LowerPage: %v", err)
	}

	metrics := composable.MeasureComposition([]composable.UINode{page}, composable.SecurityScope{Role: "Submitter"})
	if metrics.NaiveQueryCount <= metrics.DeduplicatedQueryCount {
		t.Errorf("metrics = %+v, want NaiveQueryCount > DeduplicatedQueryCount (vw_ad_pending is a real embedded duplicate)", metrics)
	}
	t.Logf("scenario 9 (Approval detail + stepper -- proxy: vw_ad_page/vw_ad_pending, detail-type Children not yet lowered): %+v", metrics)
}
