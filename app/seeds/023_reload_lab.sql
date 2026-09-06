-- seeds/023_reload_lab.sql
-- CAP-X04 proof (metadata live reload, Option A of
-- docs/decisions/002-metadata-loading.md): this Machine is DELIBERATELY NOT
-- part of `make seed`'s boot-time list (Makefile) -- the whole point of
-- T151 is proving it becomes visible WITHOUT a server restart, applied via
-- a direct `psql` call mid-conformance-run, then picked up by
-- `POST /admin/reload`.
--
-- FRANK (hr@example.com, seeds/007_authentication.sql) is reused rather
-- than a new account -- this file only needs one more role grant for an
-- account the suite already has a session for.

INSERT INTO workspaces (id, name) VALUES ('ws_default', 'Default Workspace')
ON CONFLICT (id) DO NOTHING;

INSERT INTO applications (id, workspace_id, name) VALUES
    ('app_reload_lab', 'ws_default', 'Reload Lab')
ON CONFLICT (id) DO NOTHING;

INSERT INTO machines (id, application_id, name) VALUES
    ('mch_reload_case', 'app_reload_lab', 'Reload Case')
ON CONFLICT (id) DO NOTHING;

INSERT INTO fields (id, machine_id, name, type, position, required, options) VALUES
    ('fld_rlc_title', 'mch_reload_case', 'Title', 'text', 0, true, '{}')
ON CONFLICT (id) DO NOTHING;

INSERT INTO permissions (id, machine_id, role, events) VALUES
    ('perm_rlc_reader', 'mch_reload_case', 'Reader', ARRAY[]::TEXT[])
ON CONFLICT (id) DO NOTHING;

INSERT INTO views (id, machine_id, name, type, position, config) VALUES
    ('vw_rlc_list', 'mch_reload_case', 'Reload Cases', 'list', 0, '{"columns":["fld_rlc_title"]}')
ON CONFLICT (id) DO NOTHING;

INSERT INTO user_application_roles (user_id, application_id, role)
SELECT u.id, 'app_reload_lab', 'Reader' FROM users u WHERE u.email = 'hr@example.com'
ON CONFLICT (user_id, application_id) DO UPDATE SET role = EXCLUDED.role;
