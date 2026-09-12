package composable_test

import (
	"context"
	"testing"

	"menata.id/app/internal/composable"
	"menata.id/app/internal/metadata"
	"menata.id/app/internal/testing/testdb"
)

// TestBuildDependencyDAGDedupsAgainstApprovalDashboard is Phase 7's own
// dedup proof (composable-runtime-roadmap.md §11): seeds/050_composed_
// dashboard.sql's vw_ad_pending is a real View on mch_approval_document
// that appears TWICE in one page's own UI IR -- once as LowerPage's
// ordinary view_ref child (it's in m.Views), and again inside vw_ad_page's
// own Slots["main"] (LowerChildren resolves the same View by id).
// BuildDependencyDAG must collapse both into ONE DependencyNode with TWO
// Consumers. Requires DATABASE_URL seeded with 001+004+050.
func TestBuildDependencyDAGDedupsAgainstApprovalDashboard(t *testing.T) {
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

	dag := composable.BuildDependencyDAG([]composable.UINode{page}, composable.SecurityScope{Role: "Submitter"})

	var pendingNode *composable.DependencyNode
	for i := range dag.Nodes {
		for _, f := range dag.Nodes[i].Dataset.Projection.Fields {
			if f == "fld_ad_submitted_by" {
				pendingNode = &dag.Nodes[i]
			}
		}
	}
	if pendingNode == nil {
		t.Fatal("missing expected dependency node for vw_ad_pending's own Dataset")
	}
	if len(pendingNode.Consumers) != 2 {
		t.Fatalf("len(Consumers) = %d, want 2 (the ordinary view_ref child AND the Slots[\"main\"] child)", len(pendingNode.Consumers))
	}
}

// TestBuildDependencyDAGSeparatesRolesAgainstPermissionsLab is Phase 7's
// own security-separation proof: seeds/012_permissions_lab.sql's
// mch_pl_employee has two real Permissions (HR: no HiddenFields; Staff:
// HiddenFields=[fld_ple2_salary]) over the same real List View
// (vw_ple2_list). Requires DATABASE_URL seeded with 001+012.
func TestBuildDependencyDAGSeparatesRolesAgainstPermissionsLab(t *testing.T) {
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
	node := composable.UINode{Identity: composable.NodeIdentity{Kind: "component", Source: v.ID}, Dataset: &ds}

	hrScope := composable.ResolveSecurityScope(m, "HR")
	staffScope := composable.ResolveSecurityScope(m, "Staff")

	hrDAG := composable.BuildDependencyDAG([]composable.UINode{node}, hrScope)
	staffDAG := composable.BuildDependencyDAG([]composable.UINode{node}, staffScope)

	if hrDAG.Nodes[0].Identity.String() == staffDAG.Nodes[0].Identity.String() {
		t.Fatal("HR and Staff dependency identities are equal, want different")
	}

	hasSalary := func(fields []string) bool {
		for _, f := range fields {
			if f == "fld_ple2_salary" {
				return true
			}
		}
		return false
	}
	if !hasSalary(hrDAG.Nodes[0].Dataset.Projection.Fields) {
		t.Error("HR's own effective Dataset excludes fld_ple2_salary, want it included")
	}
	if hasSalary(staffDAG.Nodes[0].Dataset.Projection.Fields) {
		t.Error("Staff's own effective Dataset includes fld_ple2_salary, want it hidden")
	}
}
