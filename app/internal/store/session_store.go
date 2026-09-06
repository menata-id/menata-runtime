package store

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Session is one real, server-side login session (CAP-X02). ID is
// SHA-256(bearer token), hex -- computed by internal/auth, never the raw
// token itself; this store never sees or stores the token a client actually
// holds, only its hash (see auth.HashSessionToken's doc comment for why).
//
// WorkspaceID (CAP-O11) is which ONE Workspace this session is currently
// in -- empty means "authenticated, workspace not yet chosen," a real
// intermediate state between Login and the /choose-workspace picker, not an
// error. Set at Create time when the identity has exactly one Workspace
// membership (the common case, zero added friction), or later via
// SetWorkspace once a multi-membership identity actually picks one.
type Session struct {
	ID          string
	UserID      string
	WorkspaceID string
	CSRFToken   string
	CreatedAt   time.Time
	ExpiresAt   time.Time
}

type SessionStore struct {
	pool *pgxpool.Pool
}

func NewSessionStore(pool *pgxpool.Pool) *SessionStore {
	return &SessionStore{pool: pool}
}

func (s *SessionStore) db(ctx context.Context) querier {
	return dbFromContext(ctx, s.pool)
}

// Create inserts a new session row at login. Login always mints a brand-new
// session rather than reusing/upgrading a pre-login one (session-fixation
// defense) -- callers never call this to "extend" an existing session, see
// Touch for that. workspaceID may be empty (CAP-O11: workspace not chosen
// yet -- a multi-membership identity, see SetWorkspace).
func (s *SessionStore) Create(ctx context.Context, tokenHash, userID, workspaceID, csrfToken string, expiresAt time.Time) error {
	var wsID any
	if workspaceID != "" {
		wsID = workspaceID
	}
	_, err := s.db(ctx).Exec(ctx,
		`INSERT INTO sessions (id, user_id, workspace_id, csrf_token, expires_at) VALUES ($1, $2, $3, $4, $5)`,
		tokenHash, userID, wsID, csrfToken, expiresAt)
	if err != nil {
		return fmt.Errorf("create session: %w", err)
	}
	return nil
}

// Get looks up a session by its token hash. Returns pgx.ErrNoRows (via the
// underlying Scan) when absent OR expired -- an expired session and a
// nonexistent one are deliberately indistinguishable to the caller, both
// just mean "not authenticated."
func (s *SessionStore) Get(ctx context.Context, tokenHash string) (*Session, error) {
	sess := &Session{}
	var wsID *string
	err := s.db(ctx).QueryRow(ctx,
		`SELECT id, user_id, workspace_id, csrf_token, created_at, expires_at
		 FROM sessions WHERE id = $1 AND expires_at > NOW()`, tokenHash).
		Scan(&sess.ID, &sess.UserID, &wsID, &sess.CSRFToken, &sess.CreatedAt, &sess.ExpiresAt)
	if err != nil {
		return nil, err
	}
	if wsID != nil {
		sess.WorkspaceID = *wsID
	}
	return sess, nil
}

// SetWorkspace (CAP-O11) records the /choose-workspace picker's own choice
// -- the caller (handler.ChooseWorkspace) has already verified the
// submitted workspace_id is one of this identity's own real memberships
// before calling this; this store method does not re-check that itself.
func (s *SessionStore) SetWorkspace(ctx context.Context, tokenHash, workspaceID string) error {
	_, err := s.db(ctx).Exec(ctx,
		`UPDATE sessions SET workspace_id = $2 WHERE id = $1`, tokenHash, workspaceID)
	if err != nil {
		return fmt.Errorf("set session workspace: %w", err)
	}
	return nil
}

// Touch extends a session's expiry (sliding expiration) -- called on every
// authenticated request so an active user's session doesn't expire mid-use.
func (s *SessionStore) Touch(ctx context.Context, tokenHash string, expiresAt time.Time) error {
	_, err := s.db(ctx).Exec(ctx,
		`UPDATE sessions SET expires_at = $2 WHERE id = $1`, tokenHash, expiresAt)
	if err != nil {
		return fmt.Errorf("touch session: %w", err)
	}
	return nil
}

// Delete removes a session -- logout.
func (s *SessionStore) Delete(ctx context.Context, tokenHash string) error {
	_, err := s.db(ctx).Exec(ctx, `DELETE FROM sessions WHERE id = $1`, tokenHash)
	if err != nil {
		return fmt.Errorf("delete session: %w", err)
	}
	return nil
}
