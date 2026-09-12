package composable_test

import (
	"context"
	"testing"

	"menata.id/app/internal/metadata"
	"menata.id/app/internal/testing/testdb"
)

// TestTrialSubstrateCoverage_DocumentApproval is Phase 13's own proof
// (composable-runtime-roadmap.md §17) for Document Approval -- calls the
// SAME assertFullSubstrateCoverage helper TestTrialSubstrateCoverage_
// ProjectManagement below also calls, with no per-case branch anywhere in
// it. Requires DATABASE_URL seeded with 001+004.
func TestTrialSubstrateCoverage_DocumentApproval(t *testing.T) {
	pool := testdb.Connect(t)
	ctx := context.Background()
	workspaces, err := metadata.NewLoader(pool).LoadAll(ctx)
	if err != nil {
		t.Fatalf("LoadAll: %v", err)
	}
	app := findApplication(t, workspaces, "ws_default", "app_approval")

	// seeds/004_approval.sql has real metadata but no seeded record rows
	// (same gap Phase 10 already worked around) -- insert one via the real
	// store write path.
	rec := insertRecord(t, ctx, pool, "mch_approval_step", map[string]any{
		"fld_as_document": "doc-placeholder",
		"fld_as_approver": "user-placeholder",
		"fld_as_sequence": "1",
	})

	assertFullSubstrateCoverage(t, app,
		"mch_approval_document", "mch_approval_step", "fld_as_document", "Submitter",
		rec.ID, rec.Data)
}

// TestTrialSubstrateCoverage_ProjectManagement is Phase 13's own proof for
// Project Management (Case 19, seeds/052_project_management.sql) -- the
// SAME helper, same order of assertions, no case-specific code anywhere.
// Case 19 is seeded at composable-substrate-proof scope only (real
// CAP-F13 references, CAP-V06 child-lists, CAP-F16 ChildLines) -- NOT a
// faithful Trello-style board; see seeds/052's own header comment and
// composable-runtime-roadmap.md §17's status note for why. Requires
// DATABASE_URL seeded with 001+052.
func TestTrialSubstrateCoverage_ProjectManagement(t *testing.T) {
	pool := testdb.Connect(t)
	ctx := context.Background()
	workspaces, err := metadata.NewLoader(pool).LoadAll(ctx)
	if err != nil {
		t.Fatalf("LoadAll: %v", err)
	}
	app := findApplication(t, workspaces, "ws_default", "app_project_management")

	rows, err := pool.Query(ctx, "SELECT id, data FROM records WHERE machine_id = 'mch_pm_card' ORDER BY created_at LIMIT 1")
	if err != nil {
		t.Fatalf("query record: %v", err)
	}
	defer rows.Close()
	if !rows.Next() {
		t.Fatal("no seeded mch_pm_card record found")
	}
	var id string
	var data map[string]any
	if err := rows.Scan(&id, &data); err != nil {
		t.Fatalf("scan: %v", err)
	}

	assertFullSubstrateCoverage(t, app,
		"mch_pm_board", "mch_pm_card", "fld_pmc_list", "Member",
		id, data)
}
