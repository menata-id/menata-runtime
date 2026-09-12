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
	m, ok := machineIdx["mch_approval_document"]
	if !ok {
		t.Fatal("missing machine mch_approval_document")
	}

	page, err := composable.LowerPage(m, viewIdx, machineIdx)
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
