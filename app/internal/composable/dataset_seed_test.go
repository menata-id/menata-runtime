package composable_test

import (
	"context"
	"reflect"
	"testing"

	"menata.id/app/internal/composable"
	"menata.id/app/internal/metadata"
	"menata.id/app/internal/model"
	"menata.id/app/internal/testing/testdb"
)

// TestBuildDataIRAgainstApprovalCase is Phase 2 proof case 4 (composable-
// runtime-roadmap.md §6): Document Approval's own approval-step list
// (seeds/004_approval.sql's vw_as_progress) has a real DefaultSort and
// projects a `reference` field (fld_as_document) -- proving Sort lowering
// and Relation discovery against real metadata, not just builder fixtures.
// Requires DATABASE_URL against a database seeded with 001+004 (see
// testdb.Connect's own doc comment) -- skipped otherwise.
func TestBuildDataIRAgainstApprovalCase(t *testing.T) {
	pool := testdb.Connect(t)
	workspaces, err := metadata.NewLoader(pool).LoadAll(context.Background())
	if err != nil {
		t.Fatalf("LoadAll: %v", err)
	}
	app := findApplication(t, workspaces, "ws_default", "app_approval")

	ir, err := composable.BuildDataIR(app)
	if err != nil {
		t.Fatalf("BuildDataIR(app_approval): %v", err)
	}

	// app_approval has several sorted/relation-projecting Views (the
	// Document list also sorts by created_at; the Step form also projects
	// fld_as_document) -- the combination of "sorted" AND "relates to
	// mch_approval_document" uniquely picks out vw_as_progress.
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
		t.Fatal("missing expected Dataset (vw_as_progress: default_sort by fld_as_sequence, relates to mch_approval_document)")
	}
	wantSort := []composable.Sort{{Field: "fld_as_sequence", Direction: "asc"}}
	if !reflect.DeepEqual(ds.Sort, wantSort) {
		t.Errorf("Sort = %+v, want %+v", ds.Sort, wantSort)
	}
	wantRelations := []composable.RelationRef{{TargetMachineID: "mch_approval_document", ViaField: "fld_as_document"}}
	if !reflect.DeepEqual(ds.Relations, wantRelations) {
		t.Errorf("Relations = %+v, want %+v", ds.Relations, wantRelations)
	}
}

// TestBuildDataIRAgainstKanbanLab is Phase 2 proof case 3 (Project
// Management collection/card) -- same explicitly-flagged Case-19 stand-in
// Phase 1 already established (seeds/032_kanban_lab.sql): a board partitions
// rows into lanes, it doesn't aggregate them, so its Dataset must show
// GroupBy without any Measure.
func TestBuildDataIRAgainstKanbanLab(t *testing.T) {
	pool := testdb.Connect(t)
	workspaces, err := metadata.NewLoader(pool).LoadAll(context.Background())
	if err != nil {
		t.Fatalf("LoadAll: %v", err)
	}
	app := findApplication(t, workspaces, "ws_default", "app_kanban_lab")

	ir, err := composable.BuildDataIR(app)
	if err != nil {
		t.Fatalf("BuildDataIR(app_kanban_lab): %v", err)
	}

	ds := findDatasetByGroupBy(ir, "fld_kbt_status")
	if ds == nil {
		t.Fatal("missing expected Dataset grouped by fld_kbt_status (vw_kbt_board)")
	}
	if len(ds.Measures) != 0 {
		t.Errorf("Measures = %+v, want none (a board partitions rows, it doesn't aggregate them)", ds.Measures)
	}
}

// TestBuildDataIRAgainstActionLab is Phase 2 proof case 2 (existing
// Dashboard section) plus its report sibling: seeds/009_action_lab.sql +
// seeds/010_views_lab.sql's vw_alp_dashboard has two sections (mch_al_task
// grouped by fld_alt_stage, mch_al_project ungrouped), and
// seeds/008_*.sql's vw_jel_report (added by seeds/010) is a grouped-sum
// report over mch_journal_entry_line. Requires DATABASE_URL seeded with
// 001+008+009+010.
func TestBuildDataIRAgainstActionLab(t *testing.T) {
	pool := testdb.Connect(t)
	workspaces, err := metadata.NewLoader(pool).LoadAll(context.Background())
	if err != nil {
		t.Fatalf("LoadAll: %v", err)
	}

	// The dashboard/report Views were added by seeds/010_views_lab.sql onto
	// Machines declared by other seed files -- find them by Machine id
	// directly rather than assuming they live in one Application.
	var actionLab, accounting *model.Application
	for _, ws := range workspaces {
		if ws.ID != "ws_default" {
			continue
		}
		for _, a := range ws.Applications {
			switch a.ID {
			case "app_action_lab":
				actionLab = a
			case "app_accounting":
				accounting = a
			}
		}
	}
	if actionLab == nil {
		t.Fatal("missing application app_action_lab -- seed 009_action_lab.sql")
	}
	if accounting == nil {
		t.Fatal("missing application app_accounting -- seed 008_*.sql")
	}

	dashboardIR, err := composable.BuildDataIR(actionLab)
	if err != nil {
		t.Fatalf("BuildDataIR(app_action_lab): %v", err)
	}
	grouped := findDatasetByGroupBy(dashboardIR, "fld_alt_stage")
	if grouped == nil {
		t.Fatal("missing grouped Dataset for dashboard section 'Tasks by Stage'")
	}
	if want := []composable.Measure{{Kind: composable.MeasureCount}}; !reflect.DeepEqual(grouped.Measures, want) {
		t.Errorf("grouped section Measures = %+v, want %+v", grouped.Measures, want)
	}

	ungrouped := findDataset(dashboardIR, func(d composable.Dataset) bool {
		return d.Source.MachineID == "mch_al_project" && len(d.GroupBy) == 0
	})
	if ungrouped == nil {
		t.Fatal("missing ungrouped Dataset for dashboard section 'Projects'")
	}

	reportIR, err := composable.BuildDataIR(accounting)
	if err != nil {
		t.Fatalf("BuildDataIR(app_accounting): %v", err)
	}
	report := findDatasetByGroupBy(reportIR, "fld_jel_account")
	if report == nil {
		t.Fatal("missing report Dataset grouped by fld_jel_account (vw_jel_report)")
	}
	wantMeasures := []composable.Measure{
		{Kind: composable.MeasureSum, Field: "fld_jel_debit"},
		{Kind: composable.MeasureSum, Field: "fld_jel_credit"},
	}
	if !reflect.DeepEqual(report.Measures, wantMeasures) {
		t.Errorf("report Measures = %+v, want %+v", report.Measures, wantMeasures)
	}
}

func findDataset(ir composable.DataIR, match func(composable.Dataset) bool) *composable.Dataset {
	for i := range ir.Datasets {
		if match(ir.Datasets[i]) {
			return &ir.Datasets[i]
		}
	}
	return nil
}

func findDatasetByGroupBy(ir composable.DataIR, field string) *composable.Dataset {
	return findDataset(ir, func(d composable.Dataset) bool {
		return len(d.GroupBy) == 1 && d.GroupBy[0] == field
	})
}

