package store

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Invitation is CAP-O10's own row (migrations/027) -- an Admin's invite of
// one email address into a Workspace at a given role, plus optional
// per-Application roles (reusing CAP-O01's own vocabulary, not a second
// mechanism). TokenHash is SHA-256(token) hex -- the raw token only ever
// lives in the emailed accept link, same "don't store the secret itself"
// discipline sessions.id already applies.
type Invitation struct {
	ID               string
	WorkspaceID      string
	Email            string
	WorkspaceRole    string
	ApplicationRoles map[string]string
	TokenHash        string
	Status           string // pending | accepted | revoked
	InvitedBy        string
	ExpiresAt        time.Time
	CreatedAt        time.Time
	AcceptedAt       *time.Time
}

// Expired reports whether this invitation's own window has lapsed --
// computed at read time from ExpiresAt, not a separate stored status: no
// background job flips a row to "expired," the same lazy-evaluation
// discipline CAP-X04's reload-on-demand already established (nothing here
// polls, it just checks "now" against a fixed deadline whenever a row is
// actually looked at).
func (inv *Invitation) Expired() bool {
	return time.Now().After(inv.ExpiresAt)
}

type InvitationStore struct {
	pool *pgxpool.Pool
}

func NewInvitationStore(pool *pgxpool.Pool) *InvitationStore {
	return &InvitationStore{pool: pool}
}

func (s *InvitationStore) db(ctx context.Context) querier {
	return dbFromContext(ctx, s.pool)
}

// Create inserts a new pending invitation. appRoles may be nil/empty (a
// workspace-role-only invite, no Application access yet).
func (s *InvitationStore) Create(ctx context.Context, workspaceID, email, workspaceRole string, appRoles map[string]string, tokenHash, invitedBy string, expiresAt time.Time) (*Invitation, error) {
	rolesJSON, err := json.Marshal(appRoles)
	if err != nil {
		return nil, fmt.Errorf("marshal application roles: %w", err)
	}
	inv := &Invitation{}
	var rawRoles []byte
	err = s.db(ctx).QueryRow(ctx,
		`INSERT INTO workspace_invitations (workspace_id, email, workspace_role, application_roles, token_hash, invited_by, expires_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 RETURNING id, workspace_id, email, workspace_role, application_roles, token_hash, status, invited_by, expires_at, created_at, accepted_at`,
		workspaceID, email, workspaceRole, string(rolesJSON), tokenHash, invitedBy, expiresAt).
		Scan(&inv.ID, &inv.WorkspaceID, &inv.Email, &inv.WorkspaceRole, &rawRoles, &inv.TokenHash, &inv.Status, &inv.InvitedBy, &inv.ExpiresAt, &inv.CreatedAt, &inv.AcceptedAt)
	if err != nil {
		return nil, fmt.Errorf("create invitation: %w", err)
	}
	if err := json.Unmarshal(rawRoles, &inv.ApplicationRoles); err != nil {
		return nil, fmt.Errorf("parse application roles: %w", err)
	}
	return inv, nil
}

// GetByTokenHash looks up an invitation by SHA-256(token) hex -- returns
// pgx.ErrNoRows (via the underlying Scan) when absent, exactly like
// SessionStore.Get, so the caller applies the same "not found" handling.
func (s *InvitationStore) GetByTokenHash(ctx context.Context, tokenHash string) (*Invitation, error) {
	inv := &Invitation{}
	var rawRoles []byte
	err := s.db(ctx).QueryRow(ctx,
		`SELECT id, workspace_id, email, workspace_role, application_roles, token_hash, status, invited_by, expires_at, created_at, accepted_at
		 FROM workspace_invitations WHERE token_hash = $1`, tokenHash).
		Scan(&inv.ID, &inv.WorkspaceID, &inv.Email, &inv.WorkspaceRole, &rawRoles, &inv.TokenHash, &inv.Status, &inv.InvitedBy, &inv.ExpiresAt, &inv.CreatedAt, &inv.AcceptedAt)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(rawRoles, &inv.ApplicationRoles); err != nil {
		return nil, fmt.Errorf("parse application roles: %w", err)
	}
	return inv, nil
}

// MarkAccepted flips a pending invitation to accepted, idempotently safe to
// call on an already-accepted row (e.g. the "already a member" branch of
// InviteAccept).
func (s *InvitationStore) MarkAccepted(ctx context.Context, id string) error {
	_, err := s.db(ctx).Exec(ctx,
		`UPDATE workspace_invitations SET status = 'accepted', accepted_at = NOW() WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("mark invitation accepted: %w", err)
	}
	return nil
}

// Revoke marks a pending invitation revoked -- its accept link stops
// working (GetByTokenHash still finds the row, but the caller's own status
// check rejects anything other than "pending").
func (s *InvitationStore) Revoke(ctx context.Context, id, workspaceID string) error {
	_, err := s.db(ctx).Exec(ctx,
		`UPDATE workspace_invitations SET status = 'revoked' WHERE id = $1 AND workspace_id = $2`, id, workspaceID)
	if err != nil {
		return fmt.Errorf("revoke invitation: %w", err)
	}
	return nil
}

// Reissue (CAP-O10 resend) replaces a pending invitation's token and
// expiry in place -- used when the original email failed to send or the
// window lapsed, without the Admin having to re-enter the role
// assignments from scratch.
func (s *InvitationStore) Reissue(ctx context.Context, id, tokenHash string, expiresAt time.Time) error {
	_, err := s.db(ctx).Exec(ctx,
		`UPDATE workspace_invitations SET token_hash = $2, expires_at = $3, status = 'pending' WHERE id = $1`,
		id, tokenHash, expiresAt)
	if err != nil {
		return fmt.Errorf("reissue invitation: %w", err)
	}
	return nil
}

// ListByWorkspace lists every invitation for the admin invitations page,
// newest first -- Expired() is computed by the caller/template from
// ExpiresAt, not filtered out here, so a lapsed pending invite is still
// visible (and resendable) rather than silently disappearing.
func (s *InvitationStore) ListByWorkspace(ctx context.Context, workspaceID string) ([]*Invitation, error) {
	rows, err := s.db(ctx).Query(ctx,
		`SELECT id, workspace_id, email, workspace_role, application_roles, token_hash, status, invited_by, expires_at, created_at, accepted_at
		 FROM workspace_invitations WHERE workspace_id = $1 ORDER BY created_at DESC`, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("list invitations: %w", err)
	}
	defer rows.Close()

	var out []*Invitation
	for rows.Next() {
		inv := &Invitation{}
		var rawRoles []byte
		if err := rows.Scan(&inv.ID, &inv.WorkspaceID, &inv.Email, &inv.WorkspaceRole, &rawRoles, &inv.TokenHash, &inv.Status, &inv.InvitedBy, &inv.ExpiresAt, &inv.CreatedAt, &inv.AcceptedAt); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(rawRoles, &inv.ApplicationRoles); err != nil {
			return nil, fmt.Errorf("parse application roles: %w", err)
		}
		out = append(out, inv)
	}
	return out, rows.Err()
}

// GetByID scoped to workspaceID -- same cross-workspace-404 convention
// GroupStore.GetByID's callers already apply, used by Revoke/Resend
// handlers before mutating.
func (s *InvitationStore) GetByID(ctx context.Context, id, workspaceID string) (*Invitation, error) {
	inv := &Invitation{}
	var rawRoles []byte
	err := s.db(ctx).QueryRow(ctx,
		`SELECT id, workspace_id, email, workspace_role, application_roles, token_hash, status, invited_by, expires_at, created_at, accepted_at
		 FROM workspace_invitations WHERE id = $1 AND workspace_id = $2`, id, workspaceID).
		Scan(&inv.ID, &inv.WorkspaceID, &inv.Email, &inv.WorkspaceRole, &rawRoles, &inv.TokenHash, &inv.Status, &inv.InvitedBy, &inv.ExpiresAt, &inv.CreatedAt, &inv.AcceptedAt)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(rawRoles, &inv.ApplicationRoles); err != nil {
		return nil, fmt.Errorf("parse application roles: %w", err)
	}
	return inv, nil
}
