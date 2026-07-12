-- 010_authentication.sql
-- CAP-X02 (real authentication) + CAP-O01 (two-tier identity/role registry).
--
-- users.role (a single global TEXT column) never fit the real model: role is
-- two-tier -- a Workspace role (Admin/Member, workspace-wide concerns) and a
-- separate Application role assigned per (user, application) pair (the same
-- person can be "Requester" in one Application and "Submitter" in another,
-- simultaneously, with no manual "switch role" step -- their role resolves
-- from which Application they're currently in). user_application_roles
-- replaces the flat column. Application role vocabulary stays implicit --
-- whatever role strings already appear in that Application's Machines'
-- `permissions` rows, exactly today's behavior, just now validated against
-- a real assignment too.

ALTER TABLE users DROP COLUMN IF EXISTS role;
ALTER TABLE users ADD COLUMN IF NOT EXISTS workspace_role TEXT NOT NULL DEFAULT 'Member';
ALTER TABLE users ALTER COLUMN password_hash SET NOT NULL;

CREATE TABLE IF NOT EXISTS user_application_roles (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id        UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    application_id TEXT NOT NULL REFERENCES applications(id) ON DELETE CASCADE,
    role           TEXT NOT NULL,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, application_id)
);

-- sessions: real server-side sessions. id is SHA-256(bearer token), hex --
-- not the token itself, so a leaked DB row alone can't be replayed as a
-- session (same "don't store the secret, store its hash" discipline as
-- password_hash already uses, applied to bearer tokens too).
CREATE TABLE IF NOT EXISTS sessions (
    id          TEXT PRIMARY KEY,
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    csrf_token  TEXT NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at  TIMESTAMPTZ NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_sessions_user_id ON sessions (user_id);

-- Not RLS-scoped (migrations/009's ws_isolation policy family): these three
-- tables are authentication infrastructure the app must read before a
-- workspace context even exists (to log in at all) -- same trusted-
-- infrastructure category as metadata.Loader's boot-time LoadAll, not
-- per-workspace business data like records/record_events/notifications.

-- Clean up the seed-era names that baked a role into the display name
-- ("Alice Requester") -- no longer fits once one person holds different
-- roles in different Applications (seeds/001_design_request.sql's own
-- INSERT is left alone; this is a one-time fixup for names already in a
-- database that ran it before this migration existed).
UPDATE users SET name = 'Alice' WHERE email = 'alice@example.com' AND name = 'Alice Requester';
UPDATE users SET name = 'Bob' WHERE email = 'bob@example.com' AND name = 'Bob Designer';
