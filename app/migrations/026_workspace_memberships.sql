-- +goose Up
-- 026_workspace_memberships.sql
-- CAP-O11: separates identity from workspace membership -- `users` was a
-- (workspace_id, email) pair, not a person (UNIQUE(workspace_id, email)),
-- so the same email could back two disconnected rows in two workspaces
-- with no way for login to know which one (or which password) was meant.
-- `workspace_memberships` (mirrors migrations/023's `group_members` shape
-- one level up) is the real fix; the picker UI on top of it is the easy
-- part (see benchmarks/028-multi-workspace-identity-benchmark.md, Study 36).
--
-- Deliberate scope cut, named not silently dropped: `users.workspace_id`/
-- `workspace_role` are NOT dropped by this migration -- every seed file's
-- own `INSERT INTO users (workspace_id, ..., workspace_role)` stays
-- byte-identical (a real surgery risk across 24 files for a purely
-- cosmetic win); those two columns become unused by Go code after this
-- lands, cleanup deferred to a future pass, not silently left ambiguous.
CREATE TABLE IF NOT EXISTS workspace_memberships (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id        UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    workspace_id   TEXT NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    workspace_role TEXT NOT NULL,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, workspace_id)
);
CREATE INDEX IF NOT EXISTS idx_workspace_memberships_user ON workspace_memberships (user_id);

-- Backfill: one membership per existing users row, from its own
-- (soon-to-be-vestigial) workspace_id/workspace_role. On a fresh CI schema
-- this runs before any seed has inserted a user, so it's a no-op there --
-- seed files supply their own membership rows directly (see each seed
-- file's own new companion INSERT). ON CONFLICT DO NOTHING makes this safe
-- to re-run against a live database that already has some memberships.
INSERT INTO workspace_memberships (user_id, workspace_id, workspace_role)
SELECT id, workspace_id, workspace_role FROM users
WHERE workspace_id IS NOT NULL
ON CONFLICT (user_id, workspace_id) DO NOTHING;

-- users_workspace_id_email_key (confirmed via \d users before writing this
-- migration, not assumed) enforced per-workspace uniqueness -- exactly the
-- gap this capability closes. Guarded DO block, same convention
-- migrations/024 already established: Postgres constraint names aren't
-- guaranteed stable/predictable enough to assume blind.
-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'users_workspace_id_email_key'
          AND conrelid = 'users'::regclass
    ) THEN
        ALTER TABLE users DROP CONSTRAINT users_workspace_id_email_key;
    END IF;
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'users_email_key'
          AND conrelid = 'users'::regclass
    ) THEN
        ALTER TABLE users ADD CONSTRAINT users_email_key UNIQUE (email);
    END IF;
END $$;
-- +goose StatementEnd

-- sessions.workspace_id (CAP-O11): NULL means "authenticated, workspace not
-- yet chosen" -- a real intermediate state between login and the new
-- /choose-workspace picker, not an error state.
ALTER TABLE sessions ADD COLUMN IF NOT EXISTS workspace_id TEXT REFERENCES workspaces(id);

-- +goose Down
ALTER TABLE sessions DROP COLUMN IF EXISTS workspace_id;
-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'users_email_key'
          AND conrelid = 'users'::regclass
    ) THEN
        ALTER TABLE users DROP CONSTRAINT users_email_key;
    END IF;
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'users_workspace_id_email_key'
          AND conrelid = 'users'::regclass
    ) THEN
        ALTER TABLE users ADD CONSTRAINT users_workspace_id_email_key UNIQUE (workspace_id, email);
    END IF;
END $$;
-- +goose StatementEnd
DROP TABLE IF EXISTS workspace_memberships;
