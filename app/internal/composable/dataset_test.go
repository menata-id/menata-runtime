package composable_test

import (
	"reflect"
	"testing"

	"menata.id/app/internal/composable"
	"menata.id/app/internal/model"
	"menata.id/app/internal/testing/builders"
)

// TestDatasetIdenticalAcrossDisplayModes is the sharpest, most literal proof
// of Phase 2's exit criteria: a table renderer and a card renderer are "two
// different experience consumers" of the exact same list View type -- they
// differ only in ViewConfig.Display (model.go's own doc comment already
// names it a "rendering-mode toggle", CAP-V02 Tier 2). Proving Display
// never enters the built Dataset proves the §6 "Important boundary" rule
// directly: no renderer-specific configuration leaks into the semantic
// layer.
func TestDatasetIdenticalAcrossDisplayModes(t *testing.T) {
	m := builders.Machine("mch_task").Build()
	table := builders.View("vw_table", model.ViewTypeList).Columns("fld_title", "fld_status").Build()
	cards := builders.View("vw_cards", model.ViewTypeList).Columns("fld_title", "fld_status").Display("cards").Build()

	dsTable, err := composable.BuildDatasetFromView(m, table)
	if err != nil {
		t.Fatalf("BuildDatasetFromView(table): %v", err)
	}
	dsCards, err := composable.BuildDatasetFromView(m, cards)
	if err != nil {
		t.Fatalf("BuildDatasetFromView(cards): %v", err)
	}
	if !reflect.DeepEqual(dsTable, dsCards) {
		t.Fatalf("Display leaked into Dataset:\ntable: %+v\ncards: %+v", dsTable, dsCards)
	}
}

func TestBuildDatasetFromView_FilterAndSortLowered(t *testing.T) {
	m := builders.Machine("mch_task").Build()
	v := builders.View("vw_list", model.ViewTypeList).
		Columns("fld_title", "fld_due").
		Filter(model.FilterCondition{Field: "fld_due", Operator: "before", Value: "today"}).
		DefaultSort("fld_due", "asc").
		Build()

	ds, err := composable.BuildDatasetFromView(m, v)
	if err != nil {
		t.Fatalf("BuildDatasetFromView: %v", err)
	}
	wantFilter := []composable.Filter{{Field: "fld_due", Operator: "before", Value: "today"}}
	if !reflect.DeepEqual(ds.Filter, wantFilter) {
		t.Errorf("Filter = %+v, want %+v", ds.Filter, wantFilter)
	}
	wantSort := []composable.Sort{{Field: "fld_due", Direction: "asc"}}
	if !reflect.DeepEqual(ds.Sort, wantSort) {
		t.Errorf("Sort = %+v, want %+v", ds.Sort, wantSort)
	}
}

func TestBuildDatasetFromView_RelationDiscovered(t *testing.T) {
	m := builders.Machine("mch_step").
		WithField(builders.Field("fld_document", model.FieldTypeReference).
			Options(model.FieldOptions{TargetMachine: "mch_document"}).Build()).
		WithField(builders.Field("fld_notes", model.FieldTypeText).Build()).
		Build()
	v := builders.View("vw_list", model.ViewTypeList).Columns("fld_document", "fld_notes").Build()

	ds, err := composable.BuildDatasetFromView(m, v)
	if err != nil {
		t.Fatalf("BuildDatasetFromView: %v", err)
	}
	want := []composable.RelationRef{{TargetMachineID: "mch_document", ViaField: "fld_document"}}
	if !reflect.DeepEqual(ds.Relations, want) {
		t.Errorf("Relations = %+v, want %+v", ds.Relations, want)
	}
}

func TestBuildDatasetFromView_BoardGroupField(t *testing.T) {
	m := builders.Machine("mch_task").Build()
	v := builders.View("vw_board", model.ViewTypeBoard).Columns("fld_title").GroupField("fld_status").Build()

	ds, err := composable.BuildDatasetFromView(m, v)
	if err != nil {
		t.Fatalf("BuildDatasetFromView: %v", err)
	}
	if want := []string{"fld_status"}; !reflect.DeepEqual(ds.GroupBy, want) {
		t.Errorf("GroupBy = %v, want %v", ds.GroupBy, want)
	}
	if len(ds.Measures) != 0 {
		t.Errorf("Measures = %+v, want none (a board partitions rows, it doesn't aggregate them)", ds.Measures)
	}
}

func TestBuildDatasetFromReport(t *testing.T) {
	idx := composable.MachineIndex{"mch_jel": builders.Machine("mch_jel").Build()}
	cfg := &model.ReportConfig{Machine: "mch_jel", GroupField: "fld_account", SumFields: []string{"fld_debit", "fld_credit"}}

	ds, err := composable.BuildDatasetFromReport(idx, cfg)
	if err != nil {
		t.Fatalf("BuildDatasetFromReport: %v", err)
	}
	if ds.Source.MachineID != "mch_jel" {
		t.Errorf("Source.MachineID = %q, want mch_jel", ds.Source.MachineID)
	}
	if want := []string{"fld_account"}; !reflect.DeepEqual(ds.GroupBy, want) {
		t.Errorf("GroupBy = %v, want %v", ds.GroupBy, want)
	}
	wantMeasures := []composable.Measure{
		{Kind: composable.MeasureSum, Field: "fld_debit"},
		{Kind: composable.MeasureSum, Field: "fld_credit"},
	}
	if !reflect.DeepEqual(ds.Measures, wantMeasures) {
		t.Errorf("Measures = %+v, want %+v", ds.Measures, wantMeasures)
	}
}

func TestBuildDatasetFromReport_UnknownMachine(t *testing.T) {
	idx := composable.MachineIndex{}
	cfg := &model.ReportConfig{Machine: "mch_missing"}
	if _, err := composable.BuildDatasetFromReport(idx, cfg); err == nil {
		t.Fatal("want error for a report naming a machine not present in the index")
	}
}

func TestBuildDatasetFromDashboardSection_Grouped(t *testing.T) {
	idx := composable.MachineIndex{"mch_task": builders.Machine("mch_task").Build()}
	sec := model.DashboardSection{Title: "Tasks by Stage", Machine: "mch_task", GroupField: "fld_stage"}

	ds, err := composable.BuildDatasetFromDashboardSection(idx, sec)
	if err != nil {
		t.Fatalf("BuildDatasetFromDashboardSection: %v", err)
	}
	if want := []string{"fld_stage"}; !reflect.DeepEqual(ds.GroupBy, want) {
		t.Errorf("GroupBy = %v, want %v", ds.GroupBy, want)
	}
	wantMeasures := []composable.Measure{{Kind: composable.MeasureCount}}
	if !reflect.DeepEqual(ds.Measures, wantMeasures) {
		t.Errorf("Measures = %+v, want %+v", ds.Measures, wantMeasures)
	}
}

// TestBuildDatasetFromDashboardSection_Ungrouped mirrors DashboardTile's own
// doc comment (internal/ui/types.go): a section always has an overall
// count, Breakdown (here: GroupBy) only when GroupField was declared.
func TestBuildDatasetFromDashboardSection_Ungrouped(t *testing.T) {
	idx := composable.MachineIndex{"mch_project": builders.Machine("mch_project").Build()}
	sec := model.DashboardSection{Title: "Projects", Machine: "mch_project"}

	ds, err := composable.BuildDatasetFromDashboardSection(idx, sec)
	if err != nil {
		t.Fatalf("BuildDatasetFromDashboardSection: %v", err)
	}
	if len(ds.GroupBy) != 0 {
		t.Errorf("GroupBy = %v, want none", ds.GroupBy)
	}
	wantMeasures := []composable.Measure{{Kind: composable.MeasureCount}}
	if !reflect.DeepEqual(ds.Measures, wantMeasures) {
		t.Errorf("Measures = %+v, want %+v", ds.Measures, wantMeasures)
	}
}

// TestSameDatasetSharedAcrossDashboardAndBoardConsumers is Phase 2's own
// exit-criteria proof made concrete: a Dashboard section and a Board view
// are two entirely different experience shapes, built by two different
// adapter functions, over the same Machine and the same grouping field.
// Their Datasets must agree on everything except Measures -- a Dashboard
// section always contributes a count aggregate (DashboardTile's own "always
// at least a total" rule), a Board never aggregates at all (it partitions
// rows into lanes) -- proving the shared part (Source/GroupBy/Relations/
// Projection) really is one reusable Dataset definition, not two
// independently duplicated translations.
func TestSameDatasetSharedAcrossDashboardAndBoardConsumers(t *testing.T) {
	m := builders.Machine("mch_task").Build()
	idx := composable.MachineIndex{"mch_task": m}

	dashboardDS, err := composable.BuildDatasetFromDashboardSection(idx, model.DashboardSection{
		Title: "Tasks by Stage", Machine: "mch_task", GroupField: "fld_stage",
	})
	if err != nil {
		t.Fatalf("BuildDatasetFromDashboardSection: %v", err)
	}

	boardView := builders.View("vw_board", model.ViewTypeBoard).GroupField("fld_stage").Build()
	boardDS, err := composable.BuildDatasetFromView(m, boardView)
	if err != nil {
		t.Fatalf("BuildDatasetFromView: %v", err)
	}

	if !reflect.DeepEqual(dashboardDS.Source, boardDS.Source) {
		t.Errorf("Source differs: dashboard %+v, board %+v", dashboardDS.Source, boardDS.Source)
	}
	if !reflect.DeepEqual(dashboardDS.GroupBy, boardDS.GroupBy) {
		t.Errorf("GroupBy differs: dashboard %v, board %v", dashboardDS.GroupBy, boardDS.GroupBy)
	}
	if !reflect.DeepEqual(dashboardDS.Relations, boardDS.Relations) {
		t.Errorf("Relations differ: dashboard %+v, board %+v", dashboardDS.Relations, boardDS.Relations)
	}
	if !reflect.DeepEqual(dashboardDS.Projection, boardDS.Projection) {
		t.Errorf("Projection differs: dashboard %+v, board %+v", dashboardDS.Projection, boardDS.Projection)
	}
	// The one expected difference: a dashboard section always carries a
	// count Measure, a board never carries any.
	if len(boardDS.Measures) != 0 {
		t.Errorf("board Measures = %+v, want none", boardDS.Measures)
	}
	if len(dashboardDS.Measures) == 0 {
		t.Error("dashboard Measures = none, want a count measure")
	}
}

// TestBuildDatasetFromDeclaredDataset_ConvergesWithDashboardSection is
// CR-03's own two-adapters-one-Dataset precedent applied to a DECLARED
// source (CR-21, 17k): a model.Dataset (real, loadable metadata) and an
// equivalent DashboardSection (inferred from a View's own Config) built
// over the same Machine and grouping field must converge on the same
// GroupBy/Measures shape -- the concrete proof that a declared source and
// an inferred source produce the exact same composable.Dataset contract.
func TestBuildDatasetFromDeclaredDataset_ConvergesWithDashboardSection(t *testing.T) {
	m := builders.Machine("mch_task").Build()
	idx := composable.MachineIndex{"mch_task": m}

	dashboardDS, err := composable.BuildDatasetFromDashboardSection(idx, model.DashboardSection{
		Title: "Tasks by Stage", Machine: "mch_task", GroupField: "fld_stage",
	})
	if err != nil {
		t.Fatalf("BuildDatasetFromDashboardSection: %v", err)
	}

	declared := &model.Dataset{ID: "ds_tasks", BaseMachineID: "mch_task", Config: model.DatasetConfig{
		Dimensions: []model.DatasetDimension{{ID: "dim_stage", Field: "fld_stage"}},
		Measures:   []model.DatasetMeasure{{ID: "mea_count", Aggregate: "count"}},
	}}
	declaredDS, err := composable.BuildDatasetFromDeclaredDataset(idx, declared)
	if err != nil {
		t.Fatalf("BuildDatasetFromDeclaredDataset: %v", err)
	}

	if !reflect.DeepEqual(dashboardDS.Source, declaredDS.Source) {
		t.Errorf("Source differs: dashboard %+v, declared %+v", dashboardDS.Source, declaredDS.Source)
	}
	if !reflect.DeepEqual(dashboardDS.GroupBy, declaredDS.GroupBy) {
		t.Errorf("GroupBy differs: dashboard %v, declared %v", dashboardDS.GroupBy, declaredDS.GroupBy)
	}
	if !reflect.DeepEqual(dashboardDS.Measures, declaredDS.Measures) {
		t.Errorf("Measures differ: dashboard %+v, declared %+v", dashboardDS.Measures, declaredDS.Measures)
	}
}

func TestBuildDatasetFromDeclaredDataset_RelationAndSumMeasure(t *testing.T) {
	m := builders.Machine("mch_step").
		WithField(builders.Field("fld_document", model.FieldTypeReference).
			Options(model.FieldOptions{TargetMachine: "mch_document"}).Build()).
		WithField(builders.Field("fld_amount", model.FieldTypeNumber).Build()).
		Build()
	idx := composable.MachineIndex{"mch_step": m}

	declared := &model.Dataset{ID: "ds_steps", BaseMachineID: "mch_step", Config: model.DatasetConfig{
		Relations: []model.DatasetRelation{{ID: "rel_document", Via: "fld_document"}},
		Measures:  []model.DatasetMeasure{{ID: "mea_total", Aggregate: "sum", Field: "fld_amount"}},
	}}
	ds, err := composable.BuildDatasetFromDeclaredDataset(idx, declared)
	if err != nil {
		t.Fatalf("BuildDatasetFromDeclaredDataset: %v", err)
	}

	wantRelations := []composable.RelationRef{{TargetMachineID: "mch_document", ViaField: "fld_document"}}
	if !reflect.DeepEqual(ds.Relations, wantRelations) {
		t.Errorf("Relations = %+v, want %+v", ds.Relations, wantRelations)
	}
	wantMeasures := []composable.Measure{{Kind: composable.MeasureSum, Field: "fld_amount"}}
	if !reflect.DeepEqual(ds.Measures, wantMeasures) {
		t.Errorf("Measures = %+v, want %+v", ds.Measures, wantMeasures)
	}
	wantProjection := composable.Projection{Fields: []string{"fld_amount"}}
	if !reflect.DeepEqual(ds.Projection, wantProjection) {
		t.Errorf("Projection = %+v, want %+v", ds.Projection, wantProjection)
	}
}

func TestBuildDatasetFromDeclaredDataset_UnknownMachine(t *testing.T) {
	declared := &model.Dataset{ID: "ds_ghost", BaseMachineID: "mch_ghost"}
	if _, err := composable.BuildDatasetFromDeclaredDataset(composable.MachineIndex{}, declared); err == nil {
		t.Fatal("want error for a dataset naming a machine not present in the index")
	}
}
