package composable_test

import (
	"context"
	"testing"

	"menata.id/app/internal/composable"
	"menata.id/app/internal/metadata"
	"menata.id/app/internal/model"
	"menata.id/app/internal/testing/testdb"
)

// TestLowerStatusBadgeAgainstApprovalCase proves StatusBadge against
// Document Approval's own real Status field (seeds/004_approval.sql).
// Requires DATABASE_URL seeded with 001+004 -- skipped otherwise.
func TestLowerStatusBadgeAgainstApprovalCase(t *testing.T) {
	pool := testdb.Connect(t)
	workspaces, err := metadata.NewLoader(pool).LoadAll(context.Background())
	if err != nil {
		t.Fatalf("LoadAll: %v", err)
	}
	app := findApplication(t, workspaces, "ws_default", "app_approval")
	m := findMachineByID(t, app, "mch_approval_document")

	if _, err := composable.LowerStatusBadge(m, "fld_ad_status"); err != nil {
		t.Fatalf("LowerStatusBadge: %v", err)
	}
}

// TestLowerActionBarAgainstApprovalCase proves ActionBar against Document
// Approval's own real declared Events.
func TestLowerActionBarAgainstApprovalCase(t *testing.T) {
	pool := testdb.Connect(t)
	workspaces, err := metadata.NewLoader(pool).LoadAll(context.Background())
	if err != nil {
		t.Fatalf("LoadAll: %v", err)
	}
	app := findApplication(t, workspaces, "ws_default", "app_approval")
	m := findMachineByID(t, app, "mch_approval_document")

	if _, err := composable.LowerActionBar(m, []string{"evt_ad_submit", "evt_ad_approve"}); err != nil {
		t.Fatalf("LowerActionBar: %v", err)
	}
}

// TestLowerCollectionAgainstApprovalCase proves Collection against Approval
// Step's own real list Dataset (vw_as_progress, already used in Phases 2-4).
func TestLowerCollectionAgainstApprovalCase(t *testing.T) {
	pool := testdb.Connect(t)
	workspaces, err := metadata.NewLoader(pool).LoadAll(context.Background())
	if err != nil {
		t.Fatalf("LoadAll: %v", err)
	}
	app := findApplication(t, workspaces, "ws_default", "app_approval")

	ir, err := composable.BuildDataIR(app)
	if err != nil {
		t.Fatalf("BuildDataIR: %v", err)
	}
	ds := findDataset(ir, func(d composable.Dataset) bool {
		if len(d.Sort) == 0 {
			return false
		}
		for _, r := range d.Relations {
			if r.TargetMachineID == "mch_approval_document" {
				return true
			}
		}
		return false
	})
	if ds == nil {
		t.Fatal("missing expected Approval Step Dataset (vw_as_progress)")
	}
	if _, err := composable.LowerCollection(*ds); err != nil {
		t.Fatalf("LowerCollection: %v", err)
	}
}

// TestLowerRecordSummaryCardAgainstApprovalDashboard proves
// RecordSummaryCard against seeds/050_composed_dashboard.sql's own real
// Display:"cards" list (vw_ad_pending). Requires DATABASE_URL seeded with
// 001+004+050.
func TestLowerRecordSummaryCardAgainstApprovalDashboard(t *testing.T) {
	pool := testdb.Connect(t)
	workspaces, err := metadata.NewLoader(pool).LoadAll(context.Background())
	if err != nil {
		t.Fatalf("LoadAll: %v", err)
	}
	app := findApplication(t, workspaces, "ws_default", "app_approval")
	m := findMachineByID(t, app, "mch_approval_document")

	var pending *model.View
	for _, v := range m.Views {
		if v.ID == "vw_ad_pending" {
			pending = v
		}
	}
	if pending == nil {
		t.Fatal("missing expected view vw_ad_pending -- seeds/050_composed_dashboard.sql")
	}

	ds, err := composable.BuildDatasetFromView(m, pending)
	if err != nil {
		t.Fatalf("BuildDatasetFromView(vw_ad_pending): %v", err)
	}
	if _, err := composable.LowerRecordSummaryCard(ds, "fld_ad_title", "fld_ad_status"); err != nil {
		t.Fatalf("LowerRecordSummaryCard: %v", err)
	}
}

// TestLowerMetricAgainstRealDashboards proves Metric's semantic rule
// against two real dashboard sections: Action Lab's ungrouped
// mch_al_project section accepts, Approval's own vw_ad_dashboard section
// (grouped by fld_ad_status) rejects. Requires DATABASE_URL seeded with
// 001+004+008+009+010+044.
func TestLowerMetricAgainstRealDashboards(t *testing.T) {
	pool := testdb.Connect(t)
	workspaces, err := metadata.NewLoader(pool).LoadAll(context.Background())
	if err != nil {
		t.Fatalf("LoadAll: %v", err)
	}

	actionLab := findApplication(t, workspaces, "ws_default", "app_action_lab")
	actionLabIR, err := composable.BuildDataIR(actionLab)
	if err != nil {
		t.Fatalf("BuildDataIR(app_action_lab): %v", err)
	}
	// mch_al_project also has its own ordinary list/form Views (GroupBy
	// empty, Measures empty too) -- len(Measures) > 0 is what actually
	// picks out the dashboard SECTION's own Dataset, not just any
	// ungrouped one on that machine.
	ungrouped := findDataset(actionLabIR, func(d composable.Dataset) bool {
		return d.Source.MachineID == "mch_al_project" && len(d.GroupBy) == 0 && len(d.Measures) > 0
	})
	if ungrouped == nil {
		t.Fatal("missing expected ungrouped Dataset for mch_al_project")
	}
	if _, err := composable.LowerMetric(*ungrouped); err != nil {
		t.Fatalf("LowerMetric(ungrouped): %v", err)
	}

	approval := findApplication(t, workspaces, "ws_default", "app_approval")
	approvalIR, err := composable.BuildDataIR(approval)
	if err != nil {
		t.Fatalf("BuildDataIR(app_approval): %v", err)
	}
	grouped := findDatasetByGroupBy(approvalIR, "fld_ad_status")
	if grouped == nil {
		t.Fatal("missing expected grouped Dataset for vw_ad_dashboard (fld_ad_status)")
	}
	if _, err := composable.LowerMetric(*grouped); err == nil {
		t.Fatal("LowerMetric(grouped): want error -- a grouped Dataset is a breakdown, not a single Metric")
	}
}

func findMachineByID(t *testing.T, app *model.Application, id string) *model.Machine {
	t.Helper()
	for _, m := range app.Machines {
		if m.ID == id {
			return m
		}
	}
	t.Fatalf("missing expected machine %s", id)
	return nil
}
