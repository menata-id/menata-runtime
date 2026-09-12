package composable_test

import (
	"context"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"

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
	card, err := composable.LowerCardRowComponent(m, v, ds)
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

// TestResolveBoardLanesAgainstProjectManagement is composable-runtime-
// roadmap.md 17h's own real-seed proof: seeds/052_project_management.sql's
// mch_pm_card board (vw_pmc_board, group_field: fld_pmc_list, a reference
// to mch_pm_list) groups its four real seeded Cards into their three real
// seeded Lists (To Do/Doing/Done) exactly like conformance/tests/
// 240_dynamic_board_lanes.sh's own T259 already proves over HTTP -- plus
// one case T259 itself never exercises: a lane record with zero matching
// cards still produces an empty BoardLane, not a missing one (a fresh
// "Backlog" list, created here with no cards). Requires DATABASE_URL
// seeded with 001+052.
func TestResolveBoardLanesAgainstProjectManagement(t *testing.T) {
	pool := testdb.Connect(t)
	ctx := context.Background()
	workspaces, err := metadata.NewLoader(pool).LoadAll(ctx)
	if err != nil {
		t.Fatalf("LoadAll: %v", err)
	}
	app := findApplication(t, workspaces, "ws_default", "app_project_management")
	cardMachine := findMachineByID(t, app, "mch_pm_card")
	board := findViewByID(t, cardMachine, "vw_pmc_board")

	node, err := composable.LowerViewToComponent(cardMachine, board)
	if err != nil {
		t.Fatalf("LowerViewToComponent: %v", err)
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, "SET LOCAL app.workspace_id = 'ws_default'"); err != nil {
		t.Fatalf("set workspace_id: %v", err)
	}
	backlog, err := store.NewRecordStore(pool).Create(store.WithTx(ctx, tx), "mch_pm_list", "ws_default", map[string]any{"fld_pml_name": "Backlog"})
	if err != nil {
		t.Fatalf("create Backlog lane: %v", err)
	}

	laneRefs := queryRecordRefs(t, ctx, tx, "mch_pm_list")
	cardRefs := queryRecordRefs(t, ctx, tx, "mch_pm_card")

	lanes, err := composable.ResolveBoardLanes(cardMachine, node, cardRefs, laneRefs)
	if err != nil {
		t.Fatalf("ResolveBoardLanes: %v", err)
	}

	byLane := make(map[string]composable.BoardLane, len(lanes))
	for _, lane := range lanes {
		byLane[lane.LaneRecordID] = lane
	}

	assertLaneTitles(t, byLane, "33333333-4444-5555-6666-000000000011", "Wireframe homepage", "Collect brand assets")
	assertLaneTitles(t, byLane, "33333333-4444-5555-6666-000000000012", "Build landing page")
	assertLaneTitles(t, byLane, "33333333-4444-5555-6666-000000000013", "Kickoff meeting notes")

	backlogLane, ok := byLane[backlog.ID]
	if !ok {
		t.Fatal("missing lane for the freshly created, zero-card Backlog list")
	}
	if len(backlogLane.Cards) != 0 {
		t.Errorf("Backlog lane Cards = %+v, want empty (CAP-V14's own 'an unused lane still renders empty' rule)", backlogLane.Cards)
	}
}

func queryRecordRefs(t *testing.T, ctx context.Context, tx pgx.Tx, machineID string) []composable.RecordRef {
	t.Helper()
	rows, err := tx.Query(ctx, "SELECT id, data FROM records WHERE machine_id = $1 ORDER BY created_at", machineID)
	if err != nil {
		t.Fatalf("query %s: %v", machineID, err)
	}
	defer rows.Close()
	var refs []composable.RecordRef
	for rows.Next() {
		var id string
		var data map[string]any
		if err := rows.Scan(&id, &data); err != nil {
			t.Fatalf("scan %s: %v", machineID, err)
		}
		refs = append(refs, composable.RecordRef{ID: id, Data: data})
	}
	return refs
}

func assertLaneTitles(t *testing.T, byLane map[string]composable.BoardLane, laneID string, wantTitles ...string) {
	t.Helper()
	lane, ok := byLane[laneID]
	if !ok {
		t.Fatalf("missing lane %s", laneID)
	}
	if len(lane.Cards) != len(wantTitles) {
		t.Fatalf("lane %s has %d cards, want %d (%v)", laneID, len(lane.Cards), len(wantTitles), wantTitles)
	}
	got := make(map[string]bool, len(lane.Cards))
	for _, c := range lane.Cards {
		if len(c.Cells) > 0 {
			got[c.Cells[0].Display] = true
		}
	}
	for _, want := range wantTitles {
		if !got[want] {
			t.Errorf("lane %s cards = %v, missing want %q", laneID, lane.Cards, want)
		}
	}
}
