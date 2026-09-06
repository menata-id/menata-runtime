package metadata_test

import (
	"context"
	"testing"

	"menata.id/app/internal/metadata"
	"menata.id/app/internal/model"
	"menata.id/app/internal/testing/testdb"
)

// TestLoadAllAgainstApprovalCase is app/ROADMAP.md's own Phase 1 verify
// step: a fresh migrate-up + one real seed file (Case 3, Approval) followed
// by Loader.LoadAll producing a correct in-memory Application Model.
// Requires DATABASE_URL pointed at a database that has already run
// `make migrate-up && make seed` -- skipped otherwise, same posture
// prototype/go's own conformance suite takes toward needing a real server.
func TestLoadAllAgainstApprovalCase(t *testing.T) {
	pool := testdb.Connect(t)

	workspaces, err := metadata.NewLoader(pool).LoadAll(context.Background())
	if err != nil {
		t.Fatalf("LoadAll: %v", err)
	}

	ws := findWorkspace(workspaces, "ws_default")
	if ws == nil {
		t.Fatalf("LoadAll: missing expected workspace ws_default (got %d workspaces)", len(workspaces))
	}

	app := findApplication(ws.Applications, "app_approval")
	if app == nil {
		t.Fatalf("workspace ws_default: missing expected application app_approval")
	}

	for _, wantMachine := range []string{"mch_approval_document", "mch_approval_step"} {
		if findMachine(app.Machines, wantMachine) == nil {
			t.Errorf("app_approval: missing expected machine %s", wantMachine)
		}
	}
}

func findWorkspace(workspaces []*model.Workspace, id string) *model.Workspace {
	for _, w := range workspaces {
		if w.ID == id {
			return w
		}
	}
	return nil
}

func findApplication(apps []*model.Application, id string) *model.Application {
	for _, a := range apps {
		if a.ID == id {
			return a
		}
	}
	return nil
}

func findMachine(machines []*model.Machine, id string) *model.Machine {
	for _, m := range machines {
		if m.ID == id {
			return m
		}
	}
	return nil
}
