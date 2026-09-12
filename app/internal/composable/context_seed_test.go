package composable_test

import (
	"context"
	"testing"

	"menata.id/app/internal/composable"
	"menata.id/app/internal/metadata"
	"menata.id/app/internal/testing/testdb"
)

// TestScopeChainAgainstApprovalCase is Phase 3's real proof case
// (composable-runtime-roadmap.md §7: "Approval Document → Approval Step →
// Approver"). It reuses Phase 2's own BuildDataIR output on real seeded
// metadata (seeds/004_approval.sql) instead of re-deriving nesting from the
// Children/embedding mechanism separately -- the Approval Step Dataset
// already carries a RelationRef to mch_approval_document
// (TestBuildDataIRAgainstApprovalCase proves this same fact for Phase 2).
// Requires DATABASE_URL seeded with 001+004 -- skipped otherwise.
func TestScopeChainAgainstApprovalCase(t *testing.T) {
	pool := testdb.Connect(t)
	workspaces, err := metadata.NewLoader(pool).LoadAll(context.Background())
	if err != nil {
		t.Fatalf("LoadAll: %v", err)
	}
	app := findApplication(t, workspaces, "ws_default", "app_approval")

	ir, err := composable.BuildDataIR(app)
	if err != nil {
		t.Fatalf("BuildDataIR(app_approval): %v", err)
	}

	// Same selector TestBuildDataIRAgainstApprovalCase (dataset_seed_test.go)
	// already uses to pick out vw_as_progress specifically.
	stepDS := findDataset(ir, func(d composable.Dataset) bool {
		if len(d.Sort) == 0 {
			return false
		}
		for _, r := range d.Relations {
			if r.TargetMachineID == "mch_approval_document" {
				return true
			}
		}
		return false
	})
	if stepDS == nil {
		t.Fatal("missing expected Approval Step Dataset (vw_as_progress)")
	}

	collectionScope := composable.ScopeForDataset(*stepDS)
	if collectionScope.Domains[composable.ContextParentRecord] != "mch_approval_document" {
		t.Fatalf("ScopeForDataset: parent_record = %q, want mch_approval_document", collectionScope.Domains[composable.ContextParentRecord])
	}

	approverBinding := composable.Binding{
		Target: composable.ContextRef{Domain: composable.ContextRecord, Key: "fld_as_approver"},
		Source: "record.fld_as_approver",
	}

	// (b) Rejected before descending -- the collection-level scope has no
	// singular 'record' yet, only query_result/parent_record.
	if err := composable.ValidateBinding(collectionScope, approverBinding); err == nil {
		t.Fatal("ValidateBinding: want error against the collection-level scope (no 'record' declared yet)")
	}

	// (a) Accepted once descended to a single Step record.
	recordScope := composable.DeriveChildScope(collectionScope, map[composable.ContextDomain]string{
		composable.ContextRecord: "mch_approval_step",
	})
	if err := composable.ValidateBinding(recordScope, approverBinding); err != nil {
		t.Fatalf("ValidateBinding: want no error at record scope, got %v", err)
	}

	// (c) A binding redefining current_user from arbitrary field data is
	// rejected regardless of scope level.
	badCurrentUser := composable.Binding{
		Target: composable.ContextRef{Domain: composable.ContextCurrentUser},
		Source: "fld_as_approver",
	}
	if err := composable.ValidateBinding(recordScope, badCurrentUser); err == nil {
		t.Fatal("ValidateBinding: want error for current_user re-sourced from fld_as_approver")
	}
}
