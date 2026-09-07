-- +goose Up
-- 027_workspace_invitations.sql
-- CAP-O10: an existing workspace Admin invites someone in by email; the
-- runtime sends a real outbound email (internal/mailer) with a one-time
-- accept link. Study 35 §5.3's own shape, built after CAP-O11
-- (migrations/026) already separated identity from membership -- an
-- accepted invitation for an email that already has an identity elsewhere
-- just adds a workspace_memberships row, it doesn't need a second disjoint
-- users row the way Study 35 originally worried about (that ambiguity is
-- CAP-O11's fix, not re-solved here).
--
-- Workspace-scoped identity/metadata, same tier as groups (migrations/023)
-- -- no RLS (workspace isolation is an explicit WHERE filter at the query
-- layer, same precedent groups/group_members/user_application_roles
-- already established), not row-level tenant business data.
CREATE TABLE IF NOT EXISTS workspace_invitations (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id      TEXT NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    email             TEXT NOT NULL,
    workspace_role    TEXT NOT NULL,
    application_roles JSONB NOT NULL DEFAULT '{}'::jsonb,
    -- SHA-256(token) hex, same auth.HashSessionToken primitive
    -- sessions.id/CSRF tokens already use -- the raw token only ever lives
    -- in the emailed link, never at rest (same "don't store the secret
    -- itself" principle password_hash already applies).
    token_hash        TEXT NOT NULL UNIQUE,
    status            TEXT NOT NULL DEFAULT 'pending', -- pending | accepted | revoked
    invited_by        UUID NOT NULL REFERENCES users(id),
    expires_at        TIMESTAMPTZ NOT NULL,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    accepted_at       TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_workspace_invitations_workspace ON workspace_invitations (workspace_id);
CREATE INDEX IF NOT EXISTS idx_workspace_invitations_email ON workspace_invitations (email);

-- +goose Down
DROP TABLE IF EXISTS workspace_invitations;
