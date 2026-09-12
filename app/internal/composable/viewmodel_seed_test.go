package composable_test

import (
	"context"
	"strings"
	"testing"

	"menata.id/app/internal/composable"
	"menata.id/app/internal/metadata"
	"menata.id/app/internal/store"
	"menata.id/app/internal/testing/testdb"
)

func TestResolveRecordSummaryAgainstKanbanLab(t *testing.T) {
	pool := testdb.Connect(t)
	ctx := context.Background()
	workspaces, err := metadata.NewLoader(pool).LoadAll(ctx)
	if err != nil {
		t.Fatalf("LoadAll: %v", err)
	}
	app := findApplication(t, workspaces, "ws_default", "app_kanban_lab")
	m := findMachineByID(t, app, "mch_kanban_task")
	v := findViewByID(t, m, "vw_kbt_board")

	ds, err := composable.BuildDatasetFromView(m, v)
	if err != nil {
		t.Fatalf("BuildDatasetFromView: %v", err)
	}
	card, err := composable.LowerCardRowComponent(v, ds)
	if err != nil {
		t.Fatalf("LowerRecordSummaryCard: %v", err)
	}

	rows, err := pool.Query(ctx, "SELECT id, data FROM records WHERE machine_id = 'mch_kanban_task' ORDER BY created_at LIMIT 1")
	if err != nil {
		t.Fatalf("query record: %v", err)
	}
	defer rows.Close()
	if !rows.Next() {
		t.Fatal("no seeded mch_kanban_task record found")
	}
	var id string
	var data map[string]any
	if err := rows.Scan(&id, &data); err != nil {
		t.Fatalf("scan: %v", err)
	}

	summary, err := composable.ResolveRecordSummary(m, card, id, data)
	if err != nil {
		t.Fatalf("ResolveRecordSummary: %v", err)
	}
	if summary.Title.Kind != composable.FieldValueField || summary.Title.Display == "" {
		t.Errorf("Title = %+v, want a non-empty field value", summary.Title)
	}
}

func TestResolveCollectionItemAgainstKanbanLab(t *testing.T) {
	pool := testdb.Connect(t)
	ctx := context.Background()
	workspaces, err := metadata.NewLoader(pool).LoadAll(ctx)
	if err != nil {
		t.Fatalf("LoadAll: %v", err)
	}
	app := findApplication(t, workspaces, "ws_default", "app_kanban_lab")
	m := findMachineByID(t, app, "mch_kanban_task")
	v := findViewByID(t, m, "vw_kbt_board")

	node, err := composable.LowerViewToComponent(m, v)
	if err != nil {
		t.Fatalf("LowerViewToComponent: %v", err)
	}

	rows, err := pool.Query(ctx, "SELECT id, data FROM records WHERE machine_id = 'mch_kanban_task' ORDER BY created_at LIMIT 1")
	if err != nil {
		t.Fatalf("query record: %v", err)
	}
	defer rows.Close()
	if !rows.Next() {
		t.Fatal("no seeded mch_kanban_task record found")
	}
	var id string
	var data map[string]any
	if err := rows.Scan(&id, &data); err != nil {
		t.Fatalf("scan: %v", err)
	}

	item, err := composable.ResolveCollectionItem(m, node, id, data)
	if err != nil {
		t.Fatalf("ResolveCollectionItem: %v", err)
	}
	if len(item.Cells) == 0 {
		t.Fatal("Cells is empty, want at least one resolved cell")
	}
}

// TestResolveFieldValueRelationAndExpression covers the relation and
// expression Kinds against real metadata, inserting the record rows those
// seed files don't themselves provide (seeds/031_typeahead_lab.sql has 30
// real Product rows but zero Order rows; seeds/041_expression_lab.sql has
// a real computed field but zero rows at all) via the real
// store.RecordStore.Create write path. Requires DATABASE_URL seeded with
// 001+031+041.
func TestResolveFieldValueRelationAndExpression(t *testing.T) {
	pool := testdb.Connect(t)
	ctx := context.Background()
	workspaces, err := metadata.NewLoader(pool).LoadAll(ctx)
	if err != nil {
		t.Fatalf("LoadAll: %v", err)
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
	rs := store.NewRecordStore(pool)

	// Relation: a real Order pointing at one of the 30 real seeded Product ids.
	typeaheadApp := findApplication(t, workspaces, "ws_default", "app_typeahead_lab")
	orderMachine := findMachineByID(t, typeaheadApp, "mch_v16_order")
	order, err := rs.Create(txCtx, "mch_v16_order", "ws_default", map[string]any{
		"fld_v16o_title":   "Order 1",
		"fld_v16o_product": "11111111-2222-3333-4444-000000000001",
	})
	if err != nil {
		t.Fatalf("create order: %v", err)
	}
	fv, err := composable.ResolveFieldValue(orderMachine, "fld_v16o_product", order.Data)
	if err != nil {
		t.Fatalf("ResolveFieldValue(fld_v16o_product): %v", err)
	}
	if fv.Kind != composable.FieldValueRelation || fv.Display != "11111111-2222-3333-4444-000000000001" {
		t.Errorf("fv = %+v, want {relation 11111111-2222-3333-4444-000000000001}", fv)
	}

	// Expression: a real computed Total over a real Order record.
	exprApp := findApplication(t, workspaces, "ws_default", "app_expression_lab")
	exprMachine := findMachineByID(t, exprApp, "mch_expr_order")
	exprOrder, err := rs.Create(txCtx, "mch_expr_order", "ws_default", map[string]any{
		"fld_eo_customer":     "Acme",
		"fld_eo_quantity":     "10",
		"fld_eo_unit_price":   "5",
		"fld_eo_discount_pct": "0",
	})
	if err != nil {
		t.Fatalf("create order: %v", err)
	}
	totalFV, err := composable.ResolveFieldValue(exprMachine, "fld_eo_total", exprOrder.Data)
	if err != nil {
		t.Fatalf("ResolveFieldValue(fld_eo_total): %v", err)
	}
	if totalFV.Kind != composable.FieldValueExpression || !strings.Contains(totalFV.Display, "50") {
		t.Errorf("totalFV = %+v, want {expression <containing 50>}", totalFV)
	}
}

// TestResolveActionSetAgainstApprovalCase inserts one real Document record
// (seeds/004_approval.sql has no seeded records) and resolves a real
// ActionBar naming evt_ad_submit. Requires DATABASE_URL seeded with
// 001+004.
func TestResolveActionSetAgainstApprovalCase(t *testing.T) {
	pool := testdb.Connect(t)
	ctx := context.Background()
	workspaces, err := metadata.NewLoader(pool).LoadAll(ctx)
	if err != nil {
		t.Fatalf("LoadAll: %v", err)
	}
	app := findApplication(t, workspaces, "ws_default", "app_approval")
	m := findMachineByID(t, app, "mch_approval_document")

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
		"fld_ad_title":  "Q3 Policy",
		"fld_ad_status": "Draft",
	})
	if err != nil {
		t.Fatalf("create document: %v", err)
	}

	node := composable.UINode{ComponentType: composable.ComponentActionBar, Actions: []string{"evt_ad_submit"}}
	set, err := composable.ResolveActionSet(m, node, rec.ID)
	if err != nil {
		t.Fatalf("ResolveActionSet: %v", err)
	}
	if len(set.Actions) != 1 || set.Actions[0].Label != "Submit" {
		t.Errorf("Actions = %+v, want [{evt_ad_submit Submit}]", set.Actions)
	}
}
