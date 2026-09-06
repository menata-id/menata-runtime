package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrDuplicateSlug is returned by Create when migrations/025's
// UNIQUE(slug) constraint rejects a slug already claimed by another
// Workspace -- detected via the real Postgres error code (23505
// unique_violation), not a pre-check SELECT, same "atomic INSERT, never
// SELECT-then-INSERT" discipline GroupStore.Create's ErrDuplicateName
// already established (a check-then-act pattern races two near-simultaneous
// signups against each other).
var ErrDuplicateSlug = errors.New("a workspace with that URL is already taken")

// Workspace is CAP-O09's self-service-created row (migrations/025 adds
// `slug` to the table CAP-X06's own workspace isolation already relies on).
// Every prior Workspace row came from seed SQL -- this is the first Go-side
// INSERT path.
type Workspace struct {
	ID        string
	Name      string
	Slug      string
	CreatedAt time.Time
}

type WorkspaceStore struct {
	pool *pgxpool.Pool
}

func NewWorkspaceStore(pool *pgxpool.Pool) *WorkspaceStore {
	return &WorkspaceStore{pool: pool}
}

func (s *WorkspaceStore) db(ctx context.Context) querier {
	return dbFromContext(ctx, s.pool)
}

// Create inserts a new Workspace. id is set equal to slug at creation time
// only (Study 35 §5.5) -- workspaces.id has no DEFAULT (every existing row
// is a hand-chosen string, e.g. "ws_default"), so a self-service signup
// needs some id, and reusing the slug avoids inventing a second
// ID-generation scheme; id and slug stay logically distinct columns from
// here on; a future slug rename (not built) would only ever touch slug.
func (s *WorkspaceStore) Create(ctx context.Context, id, name, slug string) (*Workspace, error) {
	ws := &Workspace{}
	err := s.db(ctx).QueryRow(ctx,
		`INSERT INTO workspaces (id, name, slug) VALUES ($1, $2, $3)
		 RETURNING id, name, slug, created_at`,
		id, name, slug).Scan(&ws.ID, &ws.Name, &ws.Slug, &ws.CreatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, ErrDuplicateSlug
		}
		return nil, fmt.Errorf("create workspace: %w", err)
	}
	return ws, nil
}
