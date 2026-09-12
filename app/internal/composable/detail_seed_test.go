package composable_test

import (
	"context"
	"testing"

	"menata.id/app/internal/composable"
	"menata.id/app/internal/metadata"
	"menata.id/app/internal/store"
	"menata.id/app/internal/testing/testdb"
)

// TestResolveCollectionItemAgainstApprovalDetail (composable-runtime-
// roadmap.md 17m) is the real-seed proof of Part A/B's own design
// decision: a Detail view's Dataset (every Machine Field, Part A) lowers
// via the existing LowerCollection into a Collection component, and the
// existing ResolveCollectionItem -- unchanged, no new resolver -- resolves
// the host record's own field values in the correct order. Requires
// DATABASE_URL seeded with 001+004.
func TestResolveCollectionItemAgainstApprovalDetail(t *testing.T) {
	pool := testdb.Connect(t)
	ctx := context.Background()
	workspaces, err := metadata.NewLoader(pool).LoadAll(ctx)
	if err != nil {
		t.Fatalf("LoadAll: %v", err)
	}
	app := findApplication(t, workspaces, "ws_default", "app_approval")
	m := findMachineByID(t, app, "mch_approval_document")
	v := findViewByID(t, m, "vw_ad_detail")

	ds, err := composable.BuildDatasetFromView(m, v)
	if err != nil {
		t.Fatalf("BuildDatasetFromView: %v", err)
	}
	node, err := composable.LowerCollection(ds)
	if err != nil {
		t.Fatalf("LowerCollection: %v", err)
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, "SET LOCAL app.workspace_id = 'ws_default'"); err != nil {
		t.Fatalf("set workspace_id: %v", err)
	}
	txCtx := store.WithTx(ctx, tx)
	rec, err := store.NewRecordStore(pool).Create(txCtx, "mch_approval_document", "ws_default", map[string]any{
		"fld_ad_title":         "Q3 Vendor Contract",
		"fld_ad_document_type": "Contract",
		"fld_ad_status":        "Draft",
	})
	if err != nil {
		t.Fatalf("create document: %v", err)
	}

	item, err := composable.ResolveCollectionItem(m, node, rec.ID, rec.Data)
	if err != nil {
		t.Fatalf("ResolveCollectionItem: %v", err)
	}
	if len(item.Cells) != len(ds.Projection.Fields) {
		t.Fatalf("len(Cells) = %d, want %d (one per Machine Field)", len(item.Cells), len(ds.Projection.Fields))
	}
	want := map[string]string{
		"fld_ad_title":         "Q3 Vendor Contract",
		"fld_ad_document_type": "Contract",
		"fld_ad_status":        "Draft",
	}
	assertCellsMatch(t, ds.Projection.Fields, item.Cells, want)
}

// assertCellsMatch checks item.Cells against want (keyed by field id) --
// split out of the test function itself (Gate 3: keeps its own
// complexity small).
func assertCellsMatch(t *testing.T, projectionFields []string, cells []composable.FieldValue, want map[string]string) {
	t.Helper()
	for i, fieldID := range projectionFields {
		wantVal, ok := want[fieldID]
		if !ok {
			continue
		}
		if cells[i].Display != wantVal {
			t.Errorf("Cells[%d] (%s) = %q, want %q", i, fieldID, cells[i].Display, wantVal)
		}
	}
}
