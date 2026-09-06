// Package httptest wires a full *handler.Handler for handler-level tests.
// Named to match portal-ga3's own internal/testing/httptest layout; when a
// test also needs the stdlib package, import it under an alias (e.g.
// stdhttptest "net/http/httptest") to avoid the name collision.
package httptest

import (
	"context"
	"net/http"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"menata.id/app/internal/handler"
	"menata.id/app/internal/interpreter"
	"menata.id/app/internal/metadata"
	"menata.id/app/internal/storage"
	"menata.id/app/internal/store"
)

// NewHandler wires a full *handler.Handler against pool, mirroring
// cmd/server/main.go's own construction so a handler-level test exercises
// the real interpreter/executor/permission stack rather than a hand-rolled
// subset that could drift from production wiring. pool must already point
// at a migrated + seeded schema -- see internal/testing/testdb.
//
// Uploaded files land under t.TempDir(), cleaned up with the test.
func NewHandler(t *testing.T, pool *pgxpool.Pool) *handler.Handler {
	t.Helper()
	loader := metadata.NewLoader(pool)
	workspaces, err := loader.LoadAll(context.Background())
	if err != nil {
		t.Fatalf("httptest.NewHandler: LoadAll: %v", err)
	}
	interpStore := interpreter.NewStore(interpreter.New(workspaces))

	records := store.NewRecordStore(pool)
	notifications := store.NewNotificationStore(pool)
	outbox := store.NewOutboxStore(pool)
	sessions := store.NewSessionStore(pool)
	users := store.NewUserStore(pool)
	workspaceStore := store.NewWorkspaceStore(pool)
	groups := store.NewGroupStore(pool)
	fileStorage := storage.NewLocalDisk(t.TempDir())

	return handler.New(interpStore, loader, pool, records, notifications, outbox, sessions, users, workspaceStore, groups, false, fileStorage)
}

// Actor returns a minimal *store.Auth for role in workspaceID -- enough to
// pass permission.Guard checks keyed on WorkspaceRole/ApplicationRoles.
// Attach it to a request with WithAuth.
func Actor(workspaceID, role string) *store.Auth {
	return &store.Auth{
		User: &store.User{ID: "usr_test", WorkspaceID: workspaceID, Name: "Test Actor", WorkspaceRole: role},
	}
}

// WithAuth attaches a as the request's resolved session, the same context
// key cmd/server/main.go's sessionAuth middleware populates in production
// -- see store.WithAuth.
func WithAuth(r *http.Request, a *store.Auth) *http.Request {
	return r.WithContext(store.WithAuth(r.Context(), a))
}
