-- seeds/046_navigation_group_lab.sql
-- CAP-O03 Tier 5, Phase 1: a dedicated, tiny fixture for the group/nesting
-- half of declared navigation -- deliberately NOT app_approval. That real
-- pilot (seeds/045) was simplified to drop its own "Reference"/Signature
-- group after a live screenshot showed it overflowing a narrow viewport
-- (and, independently, contradicting this capability's own Case 3
-- reasoning that Signature isn't a real menu destination either) -- see
-- capability-registry.md's CAP-O03 Tier 5 row for the full correction.
-- Grouping itself is still real, implemented, and worth its own
-- conformance coverage; this fixture exists so that coverage doesn't
-- depend on app_approval's own actual curation choices staying a
-- particular shape.
--
-- Visitor (CAP-P07, anonymous read) covers subNavFor's own strip fine (any
-- GET straight to a Machine), but AppMachines (`/{ws}/apps/{id}`) is
-- deliberately NOT one of visitorAuth's own resolvable paths (its own doc
-- comment in cmd/server/main.go names `/{wsSlug}/apps/...` explicitly as a
-- path that never resolves to a Machine id) -- CAP-P07 is per-Machine, not
-- general anonymous navigation. A real seeded account is needed to check
-- the app-launcher card list, same pattern seeds/032_kanban_lab.sql's own
-- kanban.lead@example.com already established for its own lab fixture.
INSERT INTO applications (id, workspace_id, name) VALUES
    ('app_nav_lab', 'ws_default', 'Navigation Lab')
ON CONFLICT (id) DO NOTHING;

INSERT INTO machines (id, application_id, name) VALUES
    ('mch_nav_primary',   'app_nav_lab', 'Primary'),
    ('mch_nav_secondary', 'app_nav_lab', 'Secondary')
ON CONFLICT (id) DO NOTHING;

INSERT INTO fields (id, machine_id, name, type, position, required, options) VALUES
    ('fld_navp_title', 'mch_nav_primary',   'Title', 'text', 0, true, '{}'),
    ('fld_navs_title', 'mch_nav_secondary', 'Title', 'text', 0, true, '{}')
ON CONFLICT (id) DO NOTHING;

INSERT INTO permissions (id, machine_id, role, events, can_read, can_create, can_edit, can_delete) VALUES
    ('perm_navp_visitor', 'mch_nav_primary',   'Visitor', ARRAY[]::TEXT[], true, false, false, false),
    ('perm_navs_visitor', 'mch_nav_secondary', 'Visitor', ARRAY[]::TEXT[], true, false, false, false),
    ('perm_navp_reader',  'mch_nav_primary',   'Reader',  ARRAY[]::TEXT[], true, false, false, false),
    ('perm_navs_reader',  'mch_nav_secondary', 'Reader',  ARRAY[]::TEXT[], true, false, false, false)
ON CONFLICT (id) DO NOTHING;

INSERT INTO users (workspace_id, name, email, password_hash, workspace_role) VALUES
    ('ws_default', 'Nav Lab Reader', 'nav.lab@example.com',
     '$2a$10$moxxOcZzSu3ILTzJlLF2Q.9vxiGNnSXPl7kY1pT3t5o1FoDjqC8aK', 'Member')
ON CONFLICT (email) DO NOTHING;

-- CAP-O11: seeds/040_membership_backfill.sql runs LAST (Makefile's own seed
-- ordering) and only backfills workspace_memberships for users that already
-- existed when IT ran -- a numerically later seed (046 > 040) creating a
-- brand new user must insert its own membership row explicitly, the same
-- pattern seeds/039/041 already established, or login 403s with "no
-- workspace membership" (caught live running this file's own conformance
-- test before this fix).
INSERT INTO workspace_memberships (user_id, workspace_id, workspace_role)
SELECT id, 'ws_default', 'Member' FROM users WHERE email = 'nav.lab@example.com'
ON CONFLICT (user_id, workspace_id) DO NOTHING;

INSERT INTO user_application_roles (user_id, application_id, role)
SELECT u.id, 'app_nav_lab', 'Reader' FROM users u WHERE u.email = 'nav.lab@example.com'
ON CONFLICT (user_id, application_id) DO UPDATE SET role = EXCLUDED.role;

-- Primary (top-level machine target) + a "Secondary Group" nesting the
-- Secondary machine one level under it -- proves parent_id nesting and
-- group-flattening rendering (SubNavLink.IsGroup / Card.IsHeading)
-- independent of app_approval's own real curation.
INSERT INTO navigation_entries (id, application_id, parent_id, position, label, target_type, target_machine, target_view) VALUES
    ('nav_lab_primary',   'app_nav_lab', NULL,             0, 'Primary',         'machine', 'mch_nav_primary',   NULL),
    ('nav_lab_group',     'app_nav_lab', NULL,             1, 'Secondary Group', 'group',   NULL,                NULL),
    ('nav_lab_secondary', 'app_nav_lab', 'nav_lab_group',  0, 'Secondary',       'machine', 'mch_nav_secondary', NULL)
ON CONFLICT (id) DO NOTHING;
