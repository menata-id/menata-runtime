package store

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// User is one account row (CAP-X02/CAP-O01). WorkspaceRole is the
// workspace-wide tier (Admin/Member) — the separate, per-Application tier
// lives in user_application_roles, loaded via ApplicationRoles, not on this
// struct: a user has exactly one WorkspaceRole but zero-or-more Application
// role assignments, and callers that only need identity/workspace shouldn't
// pay for the join.
type User struct {
	ID                     string
	WorkspaceID            string
	Name                   string
	Email                  string
	PasswordHash           string
	WorkspaceRole          string
	CreatedAt              time.Time
	NotificationPreference string // CAP-O05: "immediate" (default, CAP-A10's own flat inbox) | "digest" (same inbox, grouped by day)
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
		`SELECT id, workspace_id, name, email, password_hash, workspace_role, created_at, notification_preference
		 FROM users WHERE email = $1 LIMIT 1`, email).
		Scan(&u.ID, &u.WorkspaceID, &u.Name, &u.Email, &u.PasswordHash, &u.WorkspaceRole, &u.CreatedAt, &u.NotificationPreference)
	if err != nil {
		return nil, err
	}
	return u, nil
}

// Exists reports whether id names a real user -- CAP-F05's referential
// integrity check for `user`-typed fields, the same tier as CAP-F13's
// RecordStore.Exists (a required-field violation, not a 500). Pre-validates
// UUID syntax in Go before querying, same discipline RecordStore.Exists
// established: under CAP-X06's RLS, a Postgres-level error (22P02 on a
// malformed UUID) poisons the rest of that request's shared transaction,
// so a hand-typed non-UUID value must never reach Postgres at all.
func (s *UserStore) Exists(ctx context.Context, id string) (bool, error) {
	if (&pgtype.UUID{}).Scan(id) != nil {
		return false, nil
	}
	var exists bool
	err := s.db(ctx).QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM users WHERE id = $1)`, id).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check user exists: %w", err)
	}
	return exists, nil
}

func (s *UserStore) GetByID(ctx context.Context, id string) (*User, error) {
	u := &User{}
	err := s.db(ctx).QueryRow(ctx,
		`SELECT id, workspace_id, name, email, password_hash, workspace_role, created_at, notification_preference
		 FROM users WHERE id = $1`, id).
		Scan(&u.ID, &u.WorkspaceID, &u.Name, &u.Email, &u.PasswordHash, &u.WorkspaceRole, &u.CreatedAt, &u.NotificationPreference)
	if err != nil {
		return nil, err
	}
	return u, nil
}

// SetNotificationPreference (CAP-O05) updates a user's own inbox grouping
// preference -- "immediate" (CAP-A10's existing flat list) or "digest"
// (the same inbox, grouped by day, handler.Notifications).
func (s *UserStore) SetNotificationPreference(ctx context.Context, userID, preference string) error {
	_, err := s.db(ctx).Exec(ctx, `UPDATE users SET notification_preference = $1 WHERE id = $2`, preference, userID)
	return err
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

// ListForApplicationRole returns every user holding ANY role in applicationID
// (a row in user_application_roles) -- the candidate pool for a `user`
// field's picker (CAP-F05), scoped by CAP-O01's own role model rather than
// listing every workspace user unfiltered. A prototype-honest heuristic, not
// maximally role-precise (doesn't filter to one specific role, e.g. only
// "Approver") -- the same query-time-filter, no-new-metadata-concept
// discipline CAP-O01's picker/admin page already established.
func (s *UserStore) ListForApplicationRole(ctx context.Context, applicationID string) ([]*User, error) {
	rows, err := s.db(ctx).Query(ctx,
		`SELECT u.id, u.workspace_id, u.name, u.email, u.password_hash, u.workspace_role, u.created_at
		 FROM users u
		 WHERE EXISTS (
		     SELECT 1 FROM user_application_roles uar
		     WHERE uar.user_id = u.id AND uar.application_id = $1
		 )
		 ORDER BY u.name`, applicationID)
	if err != nil {
		return nil, fmt.Errorf("list users for application role: %w", err)
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
