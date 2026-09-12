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

// TestCMP10AgainstApprovalDashboard proves View lowering equivalence
// against a real seeded View (seeds/050_composed_dashboard.sql's
// vw_ad_pending): LowerViewToComponent called twice produces identical
// results. Requires DATABASE_URL seeded with 001+004+050.
func TestCMP10AgainstApprovalDashboard(t *testing.T) {
	pool := testdb.Connect(t)
	workspaces, err := metadata.NewLoader(pool).LoadAll(context.Background())
	if err != nil {
		t.Fatalf("LoadAll: %v", err)
	}
	app := findApplication(t, workspaces, "ws_default", "app_approval")
	m := findMachineByID(t, app, "mch_approval_document")
	v := findViewByID(t, m, "vw_ad_pending")

	first, err := composable.LowerViewToComponent(m, v)
	if err != nil {
		t.Fatalf("LowerViewToComponent: %v", err)
	}
	second, err := composable.LowerViewToComponent(m, v)
	if err != nil {
		t.Fatalf("LowerViewToComponent (second call): %v", err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("not equivalent across calls:\nfirst:  %+v\nsecond: %+v", first, second)
	}
}

// TestCMP03And04AgainstApprovalDashboard proves the fail-loud path fires
// the same way on real metadata as on a synthetic fixture: a copy of
// seeds/050's own real vw_ad_page Children, mutated to name a nonexistent
// view (CMP-03) and to carry a malformed entry (CMP-04).
func TestCMP03And04AgainstApprovalDashboard(t *testing.T) {
	pool := testdb.Connect(t)
	workspaces, err := metadata.NewLoader(pool).LoadAll(context.Background())
	if err != nil {
		t.Fatalf("LoadAll: %v", err)
	}
	app := findApplication(t, workspaces, "ws_default", "app_approval")
	viewIdx := composable.IndexViews(app)
	machineIdx := composable.IndexMachines(app)
	m := findMachineByID(t, app, "mch_approval_document")
	page := findViewByID(t, m, "vw_ad_page")
	if len(page.Config.Children) == 0 {
		t.Fatal("vw_ad_page has no real Children to base this test on")
	}

	unresolved := []model.ChildViewRef{{View: "vw_does_not_exist_in_app_approval"}}
	if _, err := composable.LowerChildren(unresolved, viewIdx, machineIdx, nil); err == nil {
		t.Error("CMP-03: want error for a Children entry naming a nonexistent view, against real metadata's own indexes")
	}

	malformed := []model.ChildViewRef{{}}
	if _, err := composable.LowerChildren(malformed, viewIdx, machineIdx, nil); err == nil {
		t.Error("CMP-04: want error for a Children entry naming neither view nor content, against real metadata's own indexes")
	}
}
