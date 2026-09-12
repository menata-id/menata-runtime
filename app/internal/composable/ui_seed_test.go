package composable_test

import (
	"context"
	"testing"

	"menata.id/app/internal/composable"
	"menata.id/app/internal/metadata"
	"menata.id/app/internal/testing/testdb"
)

// TestLowerPageAgainstApprovalDashboard is Phase 4's real lowering proof
// (composable-runtime-roadmap.md §8): seeds/050_composed_dashboard.sql's
// vw_ad_page is a real, live `page` View whose Config.Children mixes a
// default-slot entry, a "main" view_ref, and an "aside" static_content --
// proving view_ref, static_content, and named Slots all lower correctly
// from one real seeded example, not a synthetic fixture. Requires
// DATABASE_URL seeded with 001+004+050 -- skipped otherwise.
func TestLowerPageAgainstApprovalDashboard(t *testing.T) {
	pool := testdb.Connect(t)
	workspaces, err := metadata.NewLoader(pool).LoadAll(context.Background())
	if err != nil {
		t.Fatalf("LoadAll: %v", err)
	}
	app := findApplication(t, workspaces, "ws_default", "app_approval")

	viewIdx := composable.IndexViews(app)
	machineIdx := composable.IndexMachines(app)
	datasetIdx := composable.IndexDatasets(app)
	m, ok := machineIdx["mch_approval_document"]
	if !ok {
		t.Fatal("missing machine mch_approval_document")
	}

	page, err := composable.LowerPage(m, viewIdx, machineIdx, datasetIdx)
	if err != nil {
		t.Fatalf("LowerPage(mch_approval_document): %v", err)
	}

	var composed *composable.UINode
	for i := range page.Children {
		if page.Children[i].Properties["view_id"] == "vw_ad_page" {
			composed = &page.Children[i]
		}
	}
	if composed == nil {
		t.Fatal("missing expected child node for vw_ad_page")
	}

	// vw_ad_page's own view_ref child node carries no Slots of its own --
	// LowerPage attaches Slots to the PAGE node when it finds a page-type
	// View among m's Views, since Children compose the page itself.
	if page.Slots == nil {
		t.Fatal("page.Slots = nil, want the vw_ad_page's own Children lowered onto the page node")
	}

	mainSlot := page.Slots["main"]
	if len(mainSlot) != 1 || mainSlot[0].Properties["view_id"] != "vw_ad_pending" {
		t.Fatalf("Slots[\"main\"] = %+v, want one node for vw_ad_pending", mainSlot)
	}
	if mainSlot[0].Dataset == nil {
		t.Fatal("Slots[\"main\"][0].Dataset = nil, want vw_ad_pending's own filtered-list dataset")
	}
	if len(mainSlot[0].Dataset.Filter) == 0 {
		t.Error("Slots[\"main\"][0].Dataset.Filter is empty, want vw_ad_pending's Status=In Review filter lowered")
	}

	// Phase 5 migrates PageContent{Type:"text"} through the Text component
	// contract (ui.go's lowerStaticContent) instead of a bare
	// static_content node -- this is that migration's own regression test.
	asideSlot := page.Slots["aside"]
	if len(asideSlot) != 1 || asideSlot[0].Kind != composable.UINodeComponent || asideSlot[0].ComponentType != composable.ComponentText {
		t.Fatalf("Slots[\"aside\"] = %+v, want one Text component node", asideSlot)
	}
	if asideSlot[0].Properties["title"] != "Recent Activity" {
		t.Errorf("Slots[\"aside\"][0] title = %q, want %q", asideSlot[0].Properties["title"], "Recent Activity")
	}
}

// TestLowerPageAttachesDetailChildrenToOwnNode is composable-runtime-
// roadmap.md 17c's own proof: closes the gap Phase 12's benchmark work
// named (a detail-type View's own Children were never lowered at all,
// unlike a page-type View's). seeds/042_inline_view_composition.sql's
// vw_as_detail (mch_approval_step's own Detail view) embeds vw_ad_progress
// (a decision_stepper View on the DIFFERENT machine mch_approval_document,
// seeds/037_decision_stepper_lab.sql) and vw_as_place (coord_placement,
// same machine, seeds/036_coord_placement_lab.sql) -- a real cross-machine
// composition case, not a synthetic fixture. Requires DATABASE_URL seeded
// with 001+004+036+037+042.
func TestLowerPageAttachesDetailChildrenToOwnNode(t *testing.T) {
	pool := testdb.Connect(t)
	workspaces, err := metadata.NewLoader(pool).LoadAll(context.Background())
	if err != nil {
		t.Fatalf("LoadAll: %v", err)
	}
	app := findApplication(t, workspaces, "ws_default", "app_approval")
	viewIdx := composable.IndexViews(app)
	machineIdx := composable.IndexMachines(app)
	datasetIdx := composable.IndexDatasets(app)
	m := findMachineByID(t, app, "mch_approval_step")

	page, err := composable.LowerPage(m, viewIdx, machineIdx, datasetIdx)
	if err != nil {
		t.Fatalf("LowerPage(mch_approval_step): %v", err)
	}

	// The Detail view's own Children attach to ITS OWN node's Slots, not
	// the whole page's -- page.Slots must stay nil here since
	// mch_approval_step has no page-type View at all.
	if page.Slots != nil {
		t.Errorf("page.Slots = %+v, want nil (mch_approval_step has no page-type View)", page.Slots)
	}

	detail := findChildByViewID(t, page.Children, "vw_as_detail")
	if len(detail.Slots[""]) != 2 {
		t.Fatalf("detail.Slots[\"\"] = %+v, want 2 nodes (vw_ad_progress, vw_as_place)", detail.Slots[""])
	}

	byViewID := indexEmbeddedNodesByViewID(detail.Slots[""])
	assertEmbeddedMachineID(t, byViewID, "vw_ad_progress", "mch_approval_document") // cross-machine resolution
	assertEmbeddedMachineID(t, byViewID, "vw_as_place", "mch_approval_step")
}

func findChildByViewID(t *testing.T, children []composable.UINode, viewID string) *composable.UINode {
	t.Helper()
	for i := range children {
		if children[i].Properties["view_id"] == viewID {
			return &children[i]
		}
	}
	t.Fatalf("missing expected child node for %s", viewID)
	return nil
}

func indexEmbeddedNodesByViewID(nodes []composable.UINode) map[string]composable.UINode {
	byViewID := make(map[string]composable.UINode, len(nodes))
	for _, n := range nodes {
		if n.ViewRef != nil {
			byViewID[n.ViewRef.ViewID] = n
		}
	}
	return byViewID
}

func assertEmbeddedMachineID(t *testing.T, byViewID map[string]composable.UINode, viewID, wantMachineID string) {
	t.Helper()
	node, ok := byViewID[viewID]
	if !ok {
		t.Fatalf("missing embedded node for %s", viewID)
	}
	if node.ViewRef.MachineID != wantMachineID {
		t.Errorf("%s ViewRef.MachineID = %q, want %q", viewID, node.ViewRef.MachineID, wantMachineID)
	}
}

// TestLowerApplicationAgainstKanbanLab is Phase 4's project-management-
// shaped proof case -- same explicitly-flagged Case-19 stand-in Phases 1-3
// already established (Case 19 itself isn't seeded in app/ yet).
func TestLowerApplicationAgainstKanbanLab(t *testing.T) {
	pool := testdb.Connect(t)
	workspaces, err := metadata.NewLoader(pool).LoadAll(context.Background())
	if err != nil {
		t.Fatalf("LoadAll: %v", err)
	}
	app := findApplication(t, workspaces, "ws_default", "app_kanban_lab")

	pages, err := composable.LowerApplication(app)
	if err != nil {
		t.Fatalf("LowerApplication(app_kanban_lab): %v", err)
	}

	var boardPage *composable.UINode
	for i := range pages {
		if pages[i].Properties["machine_id"] == "mch_kanban_task" {
			boardPage = &pages[i]
		}
	}
	if boardPage == nil {
		t.Fatal("missing expected page for mch_kanban_task")
	}

	var boardChild *composable.UINode
	for i := range boardPage.Children {
		if boardPage.Children[i].Properties["view_id"] == "vw_kbt_board" {
			boardChild = &boardPage.Children[i]
		}
	}
	if boardChild == nil {
		t.Fatal("missing expected child node for vw_kbt_board")
	}
	if boardChild.Dataset == nil || len(boardChild.Dataset.GroupBy) == 0 {
		t.Fatalf("vw_kbt_board node Dataset = %+v, want a GroupBy-bearing dataset", boardChild.Dataset)
	}
}
