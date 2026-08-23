-- 023_groups.sql
-- CAP-O07: Groups/Teams as an intermediate role-assignment grouping --
-- assign a role to a Group, put users in Groups, rather than only ever
-- assigning roles directly to individuals (registry: "cheap to retrofit
-- later, an indirection table between users and user_application_roles").
--
-- Mirrors user_application_roles' own shape exactly, one level removed:
-- groups is workspace-scoped like users/applications themselves (no RLS
-- here, same precedent as users/applications/user_application_roles --
-- identity/metadata tables, not row-level tenant business data; workspace
-- isolation is an explicit WHERE filter at the query layer, same as
-- ApplicationsForWorkspace already does). group_application_roles mirrors
-- user_application_roles' UNIQUE(user_id, application_id) constraint with
-- UNIQUE(group_id, application_id) -- one role per group per Application;
-- a person can still end up holding several roles in one Application by
-- belonging to several Groups (or one direct assignment plus Group
-- membership) -- see UserStore's session-resolution merge, not this table.

CREATE TABLE IF NOT EXISTS groups (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id TEXT NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    name         TEXT NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS group_members (
    group_id   UUID NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (group_id, user_id)
);

CREATE TABLE IF NOT EXISTS group_application_roles (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    group_id       UUID NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
    application_id TEXT NOT NULL REFERENCES applications(id) ON DELETE CASCADE,
    role           TEXT NOT NULL,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (group_id, application_id)
);

CREATE INDEX IF NOT EXISTS idx_group_members_user ON group_members (user_id);
