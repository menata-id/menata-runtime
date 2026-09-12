package metadata_test

import (
	"context"
	"testing"

	"menata.id/app/internal/metadata"
	"menata.id/app/internal/model"
	"menata.id/app/internal/testing/testdb"
)

// findApprovalApp loads app_approval fresh via a real Loader.LoadAll --
// shared setup for the three TestLoadAllAgainstComposableDataPlaneLab_*
// tests below. Each gets its own top-level function (Gate 3: gocyclo
// attributes a t.Run closure's complexity to its enclosing function, so
// separate functions, not subtests, are what actually keeps each one's own
// complexity small).
func findApprovalApp(t *testing.T) *model.Application {
	t.Helper()
	pool := testdb.Connect(t)
	workspaces, err := metadata.NewLoader(pool).LoadAll(context.Background())
	if err != nil {
		t.Fatalf("LoadAll: %v", err)
	}
	ws := findWorkspace(workspaces, "ws_default")
	if ws == nil {
		t.Fatalf("LoadAll: missing expected workspace ws_default")
	}
	app := findApplication(ws.Applications, "app_approval")
	if app == nil {
		t.Fatalf("workspace ws_default: missing expected application app_approval")
	}
	return app
}

// TestLoadAllAgainstComposableDataPlaneLab_Dataset is CR-21's (composable-
// runtime-roadmap.md 17k) own verify step, mirroring
// TestLoadAllAgainstApprovalCase's own shape: a real seed file
// (seeds/053_composable_data_plane_lab.sql) loaded end-to-end via
// Loader.LoadAll, proving the declared Dataset row really round-trips
// through the datasets table into a model.Dataset, not just parse in
// isolation.
func TestLoadAllAgainstComposableDataPlaneLab_Dataset(t *testing.T) {
	app := findApprovalApp(t)
	var ds *model.Dataset
	for _, d := range app.Datasets {
		if d.ID == "ds_ad_steps_by_document" {
			ds = d
			break
		}
	}
	if ds == nil {
		t.Fatalf("app_approval: missing expected dataset ds_ad_steps_by_document (got %d datasets)", len(app.Datasets))
	}
	if ds.BaseMachineID != "mch_approval_step" {
		t.Errorf("base_machine_id = %q, want mch_approval_step", ds.BaseMachineID)
	}
	assertDatasetConfigShape(t, ds)
}

// assertDatasetConfigShape checks ds.Config's own Relations/Dimensions/
// Measures against seeds/053's declared shape. Split out of the test
// function itself (Gate 3: keeps its own complexity small).
func assertDatasetConfigShape(t *testing.T, ds *model.Dataset) {
	t.Helper()
	if len(ds.Config.Relations) != 1 || ds.Config.Relations[0].Via != "fld_as_document" {
		t.Errorf("relations = %+v, want one relation via fld_as_document", ds.Config.Relations)
	}
	if len(ds.Config.Dimensions) != 1 || ds.Config.Dimensions[0].Field != "fld_as_decision" {
		t.Errorf("dimensions = %+v, want one dimension on fld_as_decision", ds.Config.Dimensions)
	}
	if len(ds.Config.Measures) != 1 || ds.Config.Measures[0].Aggregate != "count" {
		t.Errorf("measures = %+v, want one count measure", ds.Config.Measures)
	}
}

// TestLoadAllAgainstComposableDataPlaneLab_Query proves the companion
// Query row (seeds/053) round-trips through the queries table too.
func TestLoadAllAgainstComposableDataPlaneLab_Query(t *testing.T) {
	app := findApprovalApp(t)
	var q *model.Query
	for _, qq := range app.Queries {
		if qq.ID == "qry_ad_steps_by_decision" {
			q = qq
			break
		}
	}
	if q == nil {
		t.Fatalf("app_approval: missing expected query qry_ad_steps_by_decision (got %d queries)", len(app.Queries))
	}
	if q.DatasetID != "ds_ad_steps_by_document" {
		t.Errorf("dataset_id = %q, want ds_ad_steps_by_document", q.DatasetID)
	}
	if len(q.Config.Projection) != 2 {
		t.Errorf("projection = %v, want 2 entries", q.Config.Projection)
	}
	if q.Config.Sort == nil || q.Config.Sort.Direction != "desc" {
		t.Errorf("sort = %+v, want direction desc", q.Config.Sort)
	}
}

// TestLoadAllAgainstComposableDataPlaneLab_ExperiencePlaneChild proves
// Part B's own closure: vw_ad_page's own Children now includes the
// component+dataset entry appended by the seed's own jsonb-concat UPDATE,
// alongside its three pre-existing entries (Summary/Pending
// Documents/Recent Activity) untouched.
//
// Status update (2026-09-12, composable-runtime-roadmap.md 17l): the
// dataset_id this entry names is now ds_ad_total_steps, not
// ds_ad_steps_by_document -- 17l's own seeds/054_composable_metric_live_
// fix.sql repointed it after fixing a real bug (a grouped Dataset must
// not bind to a Metric component; ds_ad_steps_by_document has a Dimension
// and is no longer accepted there). ds_ad_steps_by_document itself is
// untouched, still a real declared Dataset -- just not this one's.
func TestLoadAllAgainstComposableDataPlaneLab_ExperiencePlaneChild(t *testing.T) {
	app := findApprovalApp(t)
	adDoc := findMachine(app.Machines, "mch_approval_document")
	if adDoc == nil {
		t.Fatalf("app_approval: missing expected machine mch_approval_document")
	}
	page := findViewByID(adDoc.Views, "vw_ad_page")
	if page == nil {
		t.Fatalf("mch_approval_document: missing expected view vw_ad_page")
	}
	if len(page.Config.Children) != 4 {
		t.Fatalf("children = %d entries, want 4 (3 pre-existing + 1 component)", len(page.Config.Children))
	}
	if !hasComponentChild(page.Config.Children, "Metric", "ds_ad_total_steps") {
		t.Errorf("no children entry with component=Metric dataset_id=ds_ad_total_steps, got %+v", page.Config.Children)
	}
}

func findViewByID(views []*model.View, id string) *model.View {
	for _, v := range views {
		if v.ID == id {
			return v
		}
	}
	return nil
}

func hasComponentChild(children []model.ChildViewRef, component, datasetID string) bool {
	for _, child := range children {
		if child.Component == component && child.DatasetID == datasetID {
			return true
		}
	}
	return false
}
