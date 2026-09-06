package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Membership is CAP-O11's real fix underneath the workspace picker: which
// Workspaces one identity (users row) belongs to, and their role in each --
// separated out because the same email can now legitimately belong to more
// than one Workspace (self-service founding, CAP-O09, made this reachable
// by real users, not just a hypothetical). Mirrors migrations/023's
// group_members shape one level up (identity, not Group, to Workspace).
type Membership struct {
	UserID        string
	WorkspaceID   string
	WorkspaceRole string
	CreatedAt     time.Time
}

type MembershipStore struct {
	pool *pgxpool.Pool
}

func NewMembershipStore(pool *pgxpool.Pool) *MembershipStore {
	return &MembershipStore{pool: pool}
}

func (s *MembershipStore) db(ctx context.Context) querier {
	return dbFromContext(ctx, s.pool)
}

// ForUser lists every Workspace an identity belongs to -- Login's own
// "0 = error, 1 = auto-enter, 2+ = show the picker" branch reads this
// directly.
func (s *MembershipStore) ForUser(ctx context.Context, userID string) ([]*Membership, error) {
	rows, err := s.db(ctx).Query(ctx,
		`SELECT user_id, workspace_id, workspace_role, created_at
		 FROM workspace_memberships WHERE user_id = $1 ORDER BY created_at`, userID)
	if err != nil {
		return nil, fmt.Errorf("list memberships: %w", err)
	}
	defer rows.Close()

	var out []*Membership
	for rows.Next() {
		m := &Membership{}
		if err := rows.Scan(&m.UserID, &m.WorkspaceID, &m.WorkspaceRole, &m.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// RoleFor resolves one identity's role in one specific Workspace --
// sessionAuth's own per-request resolution, now that a workspace_role isn't
// a fixed property of the users row anymore. ok is false when no such
// membership exists (e.g. a since-revoked membership a session's own
// workspace_id still names) -- treated as an invalid session by the caller,
// not a 500.
func (s *MembershipStore) RoleFor(ctx context.Context, userID, workspaceID string) (role string, ok bool, err error) {
	scanErr := s.db(ctx).QueryRow(ctx,
		`SELECT workspace_role FROM workspace_memberships WHERE user_id = $1 AND workspace_id = $2`,
		userID, workspaceID).Scan(&role)
	if scanErr != nil {
		if errors.Is(scanErr, pgx.ErrNoRows) {
			return "", false, nil
		}
		return "", false, fmt.Errorf("resolve membership role: %w", scanErr)
	}
	return role, true, nil
}

// Create adds one membership -- CAP-O09's founding flow (a brand-new
// Workspace's first Admin) is this store's first real caller.
func (s *MembershipStore) Create(ctx context.Context, userID, workspaceID, role string) (*Membership, error) {
	m := &Membership{}
	err := s.db(ctx).QueryRow(ctx,
		`INSERT INTO workspace_memberships (user_id, workspace_id, workspace_role)
		 VALUES ($1, $2, $3)
		 RETURNING user_id, workspace_id, workspace_role, created_at`,
		userID, workspaceID, role).Scan(&m.UserID, &m.WorkspaceID, &m.WorkspaceRole, &m.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("create membership: %w", err)
	}
	return m, nil
}

// SetRole changes an identity's role within one specific Workspace --
// replaces UserStore.SetWorkspaceRole now that role is per-membership, not
// a fixed property of the identity row.
func (s *MembershipStore) SetRole(ctx context.Context, userID, workspaceID, role string) error {
	_, err := s.db(ctx).Exec(ctx,
		`UPDATE workspace_memberships SET workspace_role = $3 WHERE user_id = $1 AND workspace_id = $2`,
		userID, workspaceID, role)
	if err != nil {
		return fmt.Errorf("set membership role: %w", err)
	}
	return nil
}
