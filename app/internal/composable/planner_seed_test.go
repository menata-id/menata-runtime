package composable_test

import (
	"context"
	"testing"

	"menata.id/app/internal/composable"
	"menata.id/app/internal/metadata"
	"menata.id/app/internal/testing/testdb"
)

// buildApprovalDocumentDAG loads app_approval and builds the Dependency
// DAG for mch_approval_document -- shared setup for both Phase 8 seed
// tests below. mch_approval_document has three real, distinct Datasets
// (vw_ad_form, vw_ad_all, vw_ad_pending) plus the naturally-duplicated
// vw_ad_pending consumption Phase 7's own dedup test already established
// (once as an ordinary child, once inside vw_ad_page's own Slots["main"]).
//
// Status update (2026-09-12, composable-runtime-roadmap.md 17k): vw_ad_page
// now also carries a fourth Children entry, a component+dataset slot bound
// to ds_ad_steps_by_document (seeds/053_composable_data_plane_lab.sql) --
// a Dataset over mch_approval_step, a DIFFERENT Machine than the page's
// own host. That's a real, deliberate cross-machine composition (the
// Experience-plane closure this increment adds), not an accident --
// TestGroupByMachineAgainstApprovalCase/TestBuildExecutionPlanAgainstApprovalCase's
// own counts below were updated to match, not loosened to hide it.
func buildApprovalDocumentDAG(t *testing.T) composable.DependencyDAG {
	t.Helper()
	pool := testdb.Connect(t)
	workspaces, err := metadata.NewLoader(pool).LoadAll(context.Background())
	if err != nil {
		t.Fatalf("LoadAll: %v", err)
	}
	app := findApplication(t, workspaces, "ws_default", "app_approval")
	viewIdx := composable.IndexViews(app)
	machineIdx := composable.IndexMachines(app)
	datasetIdx := composable.IndexDatasets(app)
	m := findMachineByID(t, app, "mch_approval_document")

	page, err := composable.LowerPage(m, viewIdx, machineIdx, datasetIdx)
	if err != nil {
		t.Fatalf("LowerPage: %v", err)
	}
	return composable.BuildDependencyDAG([]composable.UINode{page}, composable.SecurityScope{Role: "Submitter"})
}

// TestGroupByMachineAgainstApprovalCase proves Phase 8's own stage 1
// (execution groups) against real seeded metadata: mch_approval_document's
// three distinct Datasets share one MachineID and must land in one
// ExecutionGroup, grouped (not merged) -- Phase 7 already proved they're
// legitimately distinct dependencies. A second group, for
// mch_approval_step, is the 17k cross-machine component (see
// buildApprovalDocumentDAG's own doc comment) -- two groups, not a merge
// of the two Machines into one. Requires DATABASE_URL seeded with
// 001+004+050+053.
func TestGroupByMachineAgainstApprovalCase(t *testing.T) {
	dag := buildApprovalDocumentDAG(t)
	groups := composable.GroupByMachine(dag)

	if len(groups) != 2 {
		t.Fatalf("len(groups) = %d, want 2", len(groups))
	}
	if groups[0].MachineID != "mch_approval_document" {
		t.Errorf("groups[0].MachineID = %q, want mch_approval_document", groups[0].MachineID)
	}
	if len(groups[0].Nodes) != 3 {
		t.Fatalf("len(groups[0].Nodes) = %d, want 3 (vw_ad_form, vw_ad_all, vw_ad_pending)", len(groups[0].Nodes))
	}
	if groups[1].MachineID != "mch_approval_step" {
		t.Errorf("groups[1].MachineID = %q, want mch_approval_step", groups[1].MachineID)
	}
	if len(groups[1].Nodes) != 1 {
		t.Fatalf("len(groups[1].Nodes) = %d, want 1 (ds_ad_steps_by_document)", len(groups[1].Nodes))
	}
}

// TestBuildExecutionPlanAgainstApprovalCase proves Phase 8's own
// counting-based "lower physical work" comparison against the same real
// DAG: form=1 consumer, all=1 consumer, pending=2 consumers (Phase 7's own
// real duplicate), plus the 17k Metric component's own dataset=1 consumer
// -- naive=5, deduplicated=4.
func TestBuildExecutionPlanAgainstApprovalCase(t *testing.T) {
	dag := buildApprovalDocumentDAG(t)
	plan := composable.BuildExecutionPlan(dag)

	if plan.NaiveQueryCount != 5 {
		t.Errorf("NaiveQueryCount = %d, want 5", plan.NaiveQueryCount)
	}
	if plan.DeduplicatedQueryCount != 4 {
		t.Errorf("DeduplicatedQueryCount = %d, want 4", plan.DeduplicatedQueryCount)
	}
	if plan.DeduplicatedQueryCount >= plan.NaiveQueryCount {
		t.Errorf("DeduplicatedQueryCount (%d) not lower than NaiveQueryCount (%d)", plan.DeduplicatedQueryCount, plan.NaiveQueryCount)
	}
}
