package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrDuplicateName is returned by Create when migrations/024's
// UNIQUE(workspace_id, name) constraint rejects a name already used by
// another Group in the same workspace -- CAP-F23 needs that uniqueness for
// a reliable name-based lookup (GetByName), so the constraint can now
// reject a Create that used to always succeed. Detected via the real
// Postgres error code (23505 unique_violation), not a pre-check SELECT --
// the same "atomic INSERT, never SELECT-then-INSERT" discipline CAP-X13's
// webhook idempotency claim already established (a check-then-act pattern
// races two near-simultaneous requests against each other).
var ErrDuplicateName = errors.New("a group with that name already exists in this workspace")

// Group is CAP-O07's intermediate role-assignment grouping (migrations/023)
// -- an indirection between users and user_application_roles: assign a role
// to a Group once, put people in and out of the Group, instead of touching
// every individual's own row. Workspace-scoped like Application, no RLS
// (same precedent as users/applications/user_application_roles -- an
// identity/metadata table, not row-level tenant business data).
type Group struct {
	ID          string
	WorkspaceID string
	Name        string
	CreatedAt   time.Time
}

type GroupStore struct {
	pool *pgxpool.Pool
}

func NewGroupStore(pool *pgxpool.Pool) *GroupStore {
	return &GroupStore{pool: pool}
}

func (s *GroupStore) db(ctx context.Context) querier {
	return dbFromContext(ctx, s.pool)
}

func (s *GroupStore) Create(ctx context.Context, workspaceID, name string) (*Group, error) {
	g := &Group{}
	err := s.db(ctx).QueryRow(ctx,
		`INSERT INTO groups (workspace_id, name) VALUES ($1, $2)
		 RETURNING id, workspace_id, name, created_at`,
		workspaceID, name).Scan(&g.ID, &g.WorkspaceID, &g.Name, &g.CreatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, ErrDuplicateName
		}
		return nil, fmt.Errorf("create group: %w", err)
	}
	return g, nil
}

// ListByWorkspace lists every Group in a workspace, for the admin
// (/admin/users) page -- ordered by name, same convention as
// UserStore.ListByWorkspace.
func (s *GroupStore) ListByWorkspace(ctx context.Context, workspaceID string) ([]*Group, error) {
	rows, err := s.db(ctx).Query(ctx,
		`SELECT id, workspace_id, name, created_at FROM groups WHERE workspace_id = $1 ORDER BY name`,
		workspaceID)
	if err != nil {
		return nil, fmt.Errorf("list groups: %w", err)
	}
	defer rows.Close()

	var out []*Group
	for rows.Next() {
		g := &Group{}
		if err := rows.Scan(&g.ID, &g.WorkspaceID, &g.Name, &g.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, g)
	}
	return out, rows.Err()
}

func (s *GroupStore) GetByID(ctx context.Context, id string) (*Group, error) {
	g := &Group{}
	err := s.db(ctx).QueryRow(ctx,
		`SELECT id, workspace_id, name, created_at FROM groups WHERE id = $1`, id).
		Scan(&g.ID, &g.WorkspaceID, &g.Name, &g.CreatedAt)
	if err != nil {
		return nil, err
	}
	return g, nil
}

// GetByName (CAP-F23) looks up a Group by its own name, not id -- metadata
// (a Field's options) has no way to know a Group's DB-generated UUID ahead
// of time the way it can hand-pick a Machine/Field's own string id, so a
// group-restricted picker has to resolve by name at request time instead.
// Relies on migrations/024's UNIQUE(workspace_id, name) to make this a
// real lookup, not a "first of several" guess.
func (s *GroupStore) GetByName(ctx context.Context, workspaceID, name string) (*Group, error) {
	g := &Group{}
	err := s.db(ctx).QueryRow(ctx,
		`SELECT id, workspace_id, name, created_at FROM groups WHERE workspace_id = $1 AND name = $2`,
		workspaceID, name).Scan(&g.ID, &g.WorkspaceID, &g.Name, &g.CreatedAt)
	if err != nil {
		return nil, err
	}
	return g, nil
}

// MemberIDs returns every user id currently in groupID -- the admin edit
// page's own "who's checked" state.
func (s *GroupStore) MemberIDs(ctx context.Context, groupID string) (map[string]bool, error) {
	rows, err := s.db(ctx).Query(ctx, `SELECT user_id FROM group_members WHERE group_id = $1`, groupID)
	if err != nil {
		return nil, fmt.Errorf("list group members: %w", err)
	}
	defer rows.Close()

	out := map[string]bool{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out[id] = true
	}
	return out, rows.Err()
}

// SetMembers replaces groupID's entire membership with exactly userIDs, in
// one transaction -- the admin edit page always submits the full current
// checked set (a checkbox list), not a delta, same "full value, not a
// delta" convention SetApplicationRole's own doc comment already
// establishes for the per-user role selects.
func (s *GroupStore) SetMembers(ctx context.Context, groupID string, userIDs []string) error {
	db := s.db(ctx)
	if _, err := db.Exec(ctx, `DELETE FROM group_members WHERE group_id = $1`, groupID); err != nil {
		return fmt.Errorf("clear group members: %w", err)
	}
	for _, uid := range userIDs {
		if _, err := db.Exec(ctx,
			`INSERT INTO group_members (group_id, user_id) VALUES ($1, $2)`, groupID, uid); err != nil {
			return fmt.Errorf("add group member: %w", err)
		}
	}
	return nil
}

// ApplicationRoles returns groupID's own (application_id -> role)
// assignments -- the admin edit page's per-Application role selects,
// mirroring UserStore.ApplicationRoles exactly.
func (s *GroupStore) ApplicationRoles(ctx context.Context, groupID string) (map[string]string, error) {
	rows, err := s.db(ctx).Query(ctx,
		`SELECT application_id, role FROM group_application_roles WHERE group_id = $1`, groupID)
	if err != nil {
		return nil, fmt.Errorf("list group application roles: %w", err)
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

// SetApplicationRole assigns (or reassigns) groupID's role for one
// Application -- upsert, same shape as UserStore.SetApplicationRole.
func (s *GroupStore) SetApplicationRole(ctx context.Context, groupID, applicationID, role string) error {
	_, err := s.db(ctx).Exec(ctx,
		`INSERT INTO group_application_roles (group_id, application_id, role) VALUES ($1, $2, $3)
		 ON CONFLICT (group_id, application_id) DO UPDATE SET role = EXCLUDED.role`,
		groupID, applicationID, role)
	if err != nil {
		return fmt.Errorf("set group application role: %w", err)
	}
	return nil
}

// RemoveApplicationRole revokes groupID's assignment for one Application.
func (s *GroupStore) RemoveApplicationRole(ctx context.Context, groupID, applicationID string) error {
	_, err := s.db(ctx).Exec(ctx,
		`DELETE FROM group_application_roles WHERE group_id = $1 AND application_id = $2`,
		groupID, applicationID)
	if err != nil {
		return fmt.Errorf("remove group application role: %w", err)
	}
	return nil
}

// RolesForUser returns every (application_id -> role) a user holds
// *through Group membership only* (not their own direct
// user_application_roles rows) -- the other half of the union
// sessionAuth (cmd/server/main.go) merges into a session's effective
// Auth.ApplicationRoles. Deliberately a plain join, not a stored/cached
// result: Group membership changes take effect on this user's next
// request/session-resolution, same "no separate cache to invalidate"
// discipline UserStore.ApplicationRoles already follows.
func (s *GroupStore) RolesForUser(ctx context.Context, userID string) (map[string][]string, error) {
	rows, err := s.db(ctx).Query(ctx,
		`SELECT gar.application_id, gar.role
		 FROM group_members gm
		 JOIN group_application_roles gar ON gar.group_id = gm.group_id
		 WHERE gm.user_id = $1`, userID)
	if err != nil {
		return nil, fmt.Errorf("list group-derived roles: %w", err)
	}
	defer rows.Close()

	out := map[string][]string{}
	for rows.Next() {
		var appID, role string
		if err := rows.Scan(&appID, &role); err != nil {
			return nil, err
		}
		out[appID] = append(out[appID], role)
	}
	return out, rows.Err()
}

// MemberNames returns groupID's own members' display names, sorted -- the
// admin list page's compact "who's in this group" summary.
func (s *GroupStore) MemberNames(ctx context.Context, groupID string) ([]string, error) {
	rows, err := s.db(ctx).Query(ctx,
		`SELECT u.name FROM group_members gm JOIN users u ON u.id = gm.user_id
		 WHERE gm.group_id = $1 ORDER BY u.name`, groupID)
	if err != nil {
		return nil, fmt.Errorf("list group member names: %w", err)
	}
	defer rows.Close()

	var out []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		out = append(out, name)
	}
	return out, rows.Err()
}
