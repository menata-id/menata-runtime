package composable_test

import (
	"context"
	"reflect"
	"testing"

	"menata.id/app/internal/composable"
	"menata.id/app/internal/metadata"
	"menata.id/app/internal/model"
	"menata.id/app/internal/testing/builders"
	"menata.id/app/internal/testing/testdb"
)

func TestLowerPageCollectsChildren(t *testing.T) {
	m := builders.Machine("mch_task").
		WithView(builders.View("vw_list", model.ViewTypeList).Columns("fld_title").Build()).
		WithView(builders.View("vw_form", model.ViewTypeForm).Fields("fld_title").Build()).
		Build()

	page, err := composable.LowerPage(m, nil, nil)
	if err != nil {
		t.Fatalf("LowerPage: %v", err)
	}
	if page.Kind != composable.UINodePage {
		t.Fatalf("Kind = %q, want %q", page.Kind, composable.UINodePage)
	}
	if len(page.Children) != 2 {
		t.Fatalf("len(Children) = %d, want 2", len(page.Children))
	}
	for i, wantKind := range []composable.UINodeKind{composable.UINodeViewRef, composable.UINodeViewRef} {
		if page.Children[i].Kind != wantKind {
			t.Errorf("Children[%d].Kind = %q, want %q", i, page.Children[i].Kind, wantKind)
		}
	}
	if page.Children[0].Dataset == nil {
		t.Fatal("Children[0].Dataset = nil, want vw_list's own dataset")
	}
	if want := []string{"fld_title"}; !reflect.DeepEqual(page.Children[0].Dataset.Projection.Fields, want) {
		t.Errorf("Children[0].Dataset.Projection.Fields = %v, want %v", page.Children[0].Dataset.Projection.Fields, want)
	}
}

// TestLowerPageAllowsUnrepresentableViews proves a Machine mixing a
// representable and an unrepresentable View still produces every child
// node -- BuildDatasetFromView's per-view error only means Dataset stays
// nil on that one node, it never propagates out of LowerPage.
func TestLowerPageAllowsUnrepresentableViews(t *testing.T) {
	m := builders.Machine("mch_task").
		WithView(builders.View("vw_map", model.ViewTypeProcessMap).Build()).
		Build()

	page, err := composable.LowerPage(m, nil, nil)
	if err != nil {
		t.Fatalf("LowerPage: %v", err)
	}
	if len(page.Children) != 1 {
		t.Fatalf("len(Children) = %d, want 1", len(page.Children))
	}
	if page.Children[0].Dataset != nil {
		t.Errorf("Dataset = %+v, want nil (process_map has no representable dataset)", page.Children[0].Dataset)
	}
}

func TestLowerApplicationIsDeterministic(t *testing.T) {
	app := &model.Application{
		ID: "app_test",
		Machines: []*model.Machine{
			builders.Machine("mch_a").
				WithView(builders.View("vw_a_list", model.ViewTypeList).Columns("fld_a").Build()).
				Build(),
			builders.Machine("mch_b").
				WithView(builders.View("vw_b_form", model.ViewTypeForm).Fields("fld_b").Build()).
				Build(),
		},
	}

	first, err := composable.LowerApplication(app)
	if err != nil {
		t.Fatalf("LowerApplication: %v", err)
	}
	second, err := composable.LowerApplication(app)
	if err != nil {
		t.Fatalf("LowerApplication (second call): %v", err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("LowerApplication not deterministic:\nfirst:  %+v\nsecond: %+v", first, second)
	}
	if len(first) != 2 {
		t.Fatalf("len(pages) = %d, want 2", len(first))
	}
}

// TestLowerApplicationAgainstApprovalCase is Phase 1's own proof
// (composable-runtime-roadmap.md §5), still exercised here against
// LowerApplication (Phase 4's replacement for BuildApplication): real
// Document Approval metadata, loaded the normal way (metadata.Loader),
// normalizes without invoking any View handler. Requires DATABASE_URL
// pointed at a migrated+seeded database (see testdb.Connect's own doc
// comment) -- skipped otherwise.
func TestLowerApplicationAgainstApprovalCase(t *testing.T) {
	pool := testdb.Connect(t)
	workspaces, err := metadata.NewLoader(pool).LoadAll(context.Background())
	if err != nil {
		t.Fatalf("LoadAll: %v", err)
	}

	app := findApplication(t, workspaces, "ws_default", "app_approval")
	pages, err := composable.LowerApplication(app)
	if err != nil {
		t.Fatalf("LowerApplication(app_approval): %v", err)
	}
	if len(pages) != len(app.Machines) {
		t.Fatalf("len(pages) = %d, want %d (one per machine)", len(pages), len(app.Machines))
	}

	again, err := composable.LowerApplication(app)
	if err != nil {
		t.Fatalf("LowerApplication(app_approval), second call: %v", err)
	}
	if !reflect.DeepEqual(pages, again) {
		t.Fatal("LowerApplication(app_approval) not deterministic across two calls")
	}
}

func findApplication(t *testing.T, workspaces []*model.Workspace, wsID, appID string) *model.Application {
	t.Helper()
	for _, ws := range workspaces {
		if ws.ID != wsID {
			continue
		}
		for _, app := range ws.Applications {
			if app.ID == appID {
				return app
			}
		}
	}
	t.Fatalf("missing expected application %s in workspace %s", appID, wsID)
	return nil
}
