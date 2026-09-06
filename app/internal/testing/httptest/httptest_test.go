package httptest

import (
	"net/http"
	stdhttptest "net/http/httptest"
	"testing"

	"menata.id/app/internal/store"
	"menata.id/app/internal/testing/testdb"
)

// TestNewHandler_Boots proves NewHandler's wiring actually works (not just
// compiles) against a real migrated+seeded database, and that WithAuth's
// injected Auth round-trips through store.AuthFromContext the same way
// production's sessionAuth middleware would deliver it -- see
// internal/testing/testdb's own DATABASE_URL contract.
func TestNewHandler_Boots(t *testing.T) {
	pool := testdb.Connect(t)
	if h := NewHandler(t, pool); h == nil {
		t.Fatal("NewHandler returned nil")
	}

	r := stdhttptest.NewRequest(http.MethodGet, "/", nil)
	r = WithAuth(r, Actor("ws_default", "Admin"))
	a, ok := store.AuthFromContext(r.Context())
	if !ok || a.User.WorkspaceRole != "Admin" {
		t.Fatalf("AuthFromContext after WithAuth = (%+v, %v), want an Admin actor", a, ok)
	}
}
