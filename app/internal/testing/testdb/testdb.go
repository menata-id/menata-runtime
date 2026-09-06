// Package testdb provides the one piece of database test infrastructure
// every DB-backed test in this codebase already needed ad hoc:
// skip-if-no-DATABASE_URL, connect, and clean up.
package testdb

import (
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"menata.id/app/internal/db"
)

// Connect returns a pool against DATABASE_URL, calling t.Skip if it's
// unset. DATABASE_URL must already point at a database that has run
// `make migrate-up && make seed` (see DEVELOPMENT.md's "database setup"
// section) -- this package does not run migrations or seed data itself.
// app/scripts/local-ci.sh's own isolated-schema setup (CREATE SCHEMA,
// migrate-up, seed against a throwaway search_path) is what every
// DATABASE_URL used against this package should point at, so a test run
// never touches the shared persistent dev database.
//
// The returned pool is closed automatically via t.Cleanup.
func Connect(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Skip(`DATABASE_URL not set -- see DEVELOPMENT.md's "database setup" section`)
	}
	pool, err := db.Connect(dbURL)
	if err != nil {
		t.Fatalf("testdb.Connect: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}
