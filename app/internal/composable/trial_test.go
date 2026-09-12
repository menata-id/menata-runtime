// trial_test.go is Phase 13's own proof (composable-runtime-roadmap.md
// §17, Trial Migration): assertFullSubstrateCoverage is called ONCE for
// Document Approval and ONCE for Project Management (Case 19) in
// trial_seed_test.go -- the shared function itself, with no per-case
// branch anywhere inside it, IS the proof that both applications run on
// the same composable substrate (the roadmap's own success criterion),
// not two parallel implementations that merely produce similar results.
package composable_test

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"menata.id/app/internal/composable"
	"menata.id/app/internal/model"
	"menata.id/app/internal/store"
)

// assertFullSubstrateCoverage exercises all seven substrate mechanisms
// §17's own success criterion names, in order, against one real
// Application: Page/Layout, Dataset/Projection, Component contract,
// Context/Binding, permission-aware dependency planning, CEP, and the
// renderer boundary. listMachineID/relationField name the one Machine
// whose own Dataset has a real Relation (Approval Step -> Document; PM
// Card -> List) -- the concrete case Context/Binding is checked against.
func assertFullSubstrateCoverage(
	t *testing.T,
	app *model.Application,
	hostMachineID, listMachineID, relationField, role string,
	sampleRecordID string,
	sampleRecordData map[string]any,
) {
	t.Helper()
	machineIdx := composable.IndexMachines(app)

	// 1. Page/Layout primitives.
	pages, err := composable.LowerApplication(app)
	if err != nil {
		t.Fatalf("LowerApplication: %v", err)
	}
	var hostPage *composable.UINode
	for i := range pages {
		if pages[i].Properties["machine_id"] == hostMachineID {
			hostPage = &pages[i]
		}
	}
	if hostPage == nil || hostPage.Kind != composable.UINodePage {
		t.Fatalf("missing well-formed page for %s", hostMachineID)
	}

	// 2. Dataset/Projection machinery.
	ir, err := composable.BuildDataIR(app)
	if err != nil {
		t.Fatalf("BuildDataIR: %v", err)
	}
	if len(ir.Datasets) == 0 {
		t.Fatal("BuildDataIR produced no Datasets")
	}

	// 3. Component contract.
	listMachine := machineIdx[listMachineID]
	if listMachine == nil {
		t.Fatalf("missing machine %s", listMachineID)
	}
	var listView *model.View
	for _, v := range listMachine.Views {
		if v.Type == model.ViewTypeList {
			listView = v
		}
	}
	if listView == nil {
		t.Fatalf("machine %s has no list view", listMachineID)
	}
	component, err := composable.LowerViewToComponent(listMachine, listView)
	if err != nil {
		t.Fatalf("LowerViewToComponent: %v", err)
	}
	if component.ComponentType != composable.ComponentCollection {
		t.Fatalf("ComponentType = %q, want %q", component.ComponentType, composable.ComponentCollection)
	}

	// 4. Context/Binding system -- the SAME mechanism Phase 3/7 proved for
	// Document Approval, now proven again for Project Management.
	ds, err := composable.BuildDatasetFromView(listMachine, listView)
	if err != nil {
		t.Fatalf("BuildDatasetFromView: %v", err)
	}
	var relation *composable.RelationRef
	for i, r := range ds.Relations {
		if r.ViaField == relationField {
			relation = &ds.Relations[i]
		}
	}
	if relation == nil {
		t.Fatalf("Dataset has no Relation via %s (Relations=%+v)", relationField, ds.Relations)
	}
	scope := composable.ScopeForDataset(ds)
	binding := composable.Binding{
		Target: composable.ContextRef{Domain: composable.ContextParentRecord, Key: relationField},
		Source: relationField,
	}
	if err := composable.ValidateBinding(scope, binding); err != nil {
		t.Fatalf("ValidateBinding: %v", err)
	}

	// 5. Permission-aware dependency planning.
	secScope := composable.ResolveSecurityScope(listMachine, role)
	nodes := []composable.UINode{{Identity: composable.NodeIdentity{Source: listView.ID}, Dataset: &ds}}
	dag := composable.BuildDependencyDAG(nodes, secScope)
	if len(dag.Nodes) == 0 {
		t.Fatal("BuildDependencyDAG produced no nodes")
	}

	// 6. CEP (Composable Execution Planner).
	plan := composable.BuildExecutionPlan(dag)
	if plan.DeduplicatedQueryCount > plan.NaiveQueryCount {
		t.Fatalf("DeduplicatedQueryCount (%d) > NaiveQueryCount (%d)", plan.DeduplicatedQueryCount, plan.NaiveQueryCount)
	}

	// 7. Renderer boundary -- resolve real field values from a real record.
	item, err := composable.ResolveCollectionItem(listMachine, composable.UINode{
		ComponentType: composable.ComponentCollection,
		Dataset:       &ds,
	}, sampleRecordID, sampleRecordData)
	if err != nil {
		t.Fatalf("ResolveCollectionItem: %v", err)
	}
	if len(item.Cells) == 0 {
		t.Fatal("ResolveCollectionItem produced no cells")
	}
}

// insertRecord inserts one real record via the real store write path,
// same pattern Phase 10 established (RLS requires app.workspace_id set on
// the same transaction the INSERT runs on).
func insertRecord(t *testing.T, ctx context.Context, pool *pgxpool.Pool, machineID string, data map[string]any) *store.Record {
	t.Helper()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	t.Cleanup(func() { tx.Rollback(ctx) })
	if _, err := tx.Exec(ctx, "SET LOCAL app.workspace_id = 'ws_default'"); err != nil {
		t.Fatalf("set workspace_id: %v", err)
	}
	rec, err := store.NewRecordStore(pool).Create(store.WithTx(ctx, tx), machineID, "ws_default", data)
	if err != nil {
		t.Fatalf("create record on %s: %v", machineID, err)
	}
	return rec
}
