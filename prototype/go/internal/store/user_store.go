package store

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// User is one account row (CAP-X02/CAP-O01). WorkspaceRole is the
// workspace-wide tier (Admin/Member) — the separate, per-Application tier
// lives in user_application_roles, loaded via ApplicationRoles, not on this
// struct: a user has exactly one WorkspaceRole but zero-or-more Application
// role assignments, and callers that only need identity/workspace shouldn't
// pay for the join.
type User struct {
	ID            string
	WorkspaceID   string
	Name          string
	Email         string
	PasswordHash  string
	WorkspaceRole string
	CreatedAt     time.Time
}

type UserStore struct {
	pool *pgxpool.Pool
}

func NewUserStore(pool *pgxpool.Pool) *UserStore {
	return &UserStore{pool: pool}
}

// db returns the request-scoped transaction when one is attached to ctx --
// see RecordStore.db's doc comment, same convention. users/sessions/
// user_application_roles are not RLS-scoped (auth infrastructure, readable
// before a workspace context exists at all -- see migrations/010's header),
// so it makes no difference to these queries whether ctx carries a tx or
// falls back to the pool.
func (s *UserStore) db(ctx context.Context) querier {
	return dbFromContext(ctx, s.pool)
}

// GetByEmail looks up a user by email alone, not scoped to a workspace --
// login itself is how a session's workspace gets resolved (the returned
// row's own WorkspaceID), not a separate cookie/form field. Schema still
// permits the same email in two workspaces (UNIQUE(workspace_id, email)) for
// a person who legitimately has an account in each; this prototype takes
// the first match by email alone, an accepted simplification, not enforced
// against here.
func (s *UserStore) GetByEmail(ctx context.Context, email string) (*User, error) {
	u := &User{}
	err := s.db(ctx).QueryRow(ctx,
		`SELECT id, workspace_id, name, email, password_hash, workspace_role, created_at
		 FROM users WHERE email = $1 LIMIT 1`, email).
		Scan(&u.ID, &u.WorkspaceID, &u.Name, &u.Email, &u.PasswordHash, &u.WorkspaceRole, &u.CreatedAt)
	if err != nil {
		return nil, err
	}
	return u, nil
}

func (s *UserStore) GetByID(ctx context.Context, id string) (*User, error) {
	u := &User{}
	err := s.db(ctx).QueryRow(ctx,
		`SELECT id, workspace_id, name, email, password_hash, workspace_role, created_at
		 FROM users WHERE id = $1`, id).
		Scan(&u.ID, &u.WorkspaceID, &u.Name, &u.Email, &u.PasswordHash, &u.WorkspaceRole, &u.CreatedAt)
	if err != nil {
		return nil, err
	}
	return u, nil
}

// ApplicationRoles returns every (application_id -> role) assignment for a
// user -- CAP-O01's actual per-application role. Loaded once at session
// resolution and cached on the request context (see cmd/server/main.go's
// session middleware), not re-queried on every permission check.
func (s *UserStore) ApplicationRoles(ctx context.Context, userID string) (map[string]string, error) {
	rows, err := s.db(ctx).Query(ctx,
		`SELECT application_id, role FROM user_application_roles WHERE user_id = $1`, userID)
	if err != nil {
		return nil, fmt.Errorf("list application roles: %w", err)
	}
	defer rows.Close()

	out := map[string]string{}
	for rows.Next() {
		var appID, role string
		if err := rows.Scan(&appID, &role); err != nil {
			return nil, err
		}
		out[appID] = role
	}
	return out, rows.Err()
}

// ListByWorkspace lists every user in a workspace, for the admin
// (/admin/users) page -- ordered by name so the page reads deterministically.
func (s *UserStore) ListByWorkspace(ctx context.Context, workspaceID string) ([]*User, error) {
	rows, err := s.db(ctx).Query(ctx,
		`SELECT id, workspace_id, name, email, password_hash, workspace_role, created_at
		 FROM users WHERE workspace_id = $1 ORDER BY name`, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("list workspace users: %w", err)
	}
	defer rows.Close()

	var out []*User
	for rows.Next() {
		u := &User{}
		if err := rows.Scan(&u.ID, &u.WorkspaceID, &u.Name, &u.Email, &u.PasswordHash, &u.WorkspaceRole, &u.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

// SetWorkspaceRole changes a user's workspace-wide tier (Admin/Member) --
// the admin page's "workspace role" control.
func (s *UserStore) SetWorkspaceRole(ctx context.Context, userID, role string) error {
	_, err := s.db(ctx).Exec(ctx,
		`UPDATE users SET workspace_role = $2 WHERE id = $1`, userID, role)
	if err != nil {
		return fmt.Errorf("set workspace role: %w", err)
	}
	return nil
}

// SetApplicationRole assigns (or reassigns) userID's role for one
// Application -- upsert, since the admin page always submits the full
// current value for that (user, application) pair, not a delta.
func (s *UserStore) SetApplicationRole(ctx context.Context, userID, applicationID, role string) error {
	_, err := s.db(ctx).Exec(ctx,
		`INSERT INTO user_application_roles (user_id, application_id, role) VALUES ($1, $2, $3)
		 ON CONFLICT (user_id, application_id) DO UPDATE SET role = EXCLUDED.role`,
		userID, applicationID, role)
	if err != nil {
		return fmt.Errorf("set application role: %w", err)
	}
	return nil
}

// RemoveApplicationRole revokes userID's assignment for one Application --
// the admin page's "no role" option resolves to a deleted row, not a stored
// empty string (deny-by-default: no row = no access, same convention
// CAP-P05's permissions table already established).
func (s *UserStore) RemoveApplicationRole(ctx context.Context, userID, applicationID string) error {
	_, err := s.db(ctx).Exec(ctx,
		`DELETE FROM user_application_roles WHERE user_id = $1 AND application_id = $2`,
		userID, applicationID)
	if err != nil {
		return fmt.Errorf("remove application role: %w", err)
	}
	return nil
}
