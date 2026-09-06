-- seeds/024_change_policy_lab.sql
-- CAP-W07 proof (change_policy, effective-dated metadata evolution): this
-- Machine and its three baseline records exist at boot -- what T154-T158
-- actually prove live, without a restart, is the RULE CHANGE itself
-- (seeds/025_change_policy_activate.sql, applied mid-conformance-run via
-- psql + POST /admin/reload, reusing CAP-X04).
--
-- FRANK (hr@example.com, seeds/007_authentication.sql) is reused for
-- Create/Edit access -- the same account CAP-X04's own lab already granted
-- a role in a different Application.
--
-- "Compliance Note" and "Approval Reference" are deliberately NOT required
-- at the Field level (that only drives the UI's `required` attribute,
-- internal/ui/components.templ) -- server-side enforcement comes entirely
-- from the two change_policy-scoped Constraints added in 025, so their
-- absence/presence IS the thing under test. Two DIFFERENT fields, one per
-- change_policy mode, so T156/T157 can isolate the dimension each is
-- proving (state vs. date) by filling the field the OTHER constraint
-- gates -- otherwise a record that's simultaneously "not Draft" and
-- "newly created" would get blocked by whichever constraint the test
-- wasn't trying to exercise.

INSERT INTO workspaces (id, name, slug) VALUES ('ws_default', 'Default Workspace', 'ws_default')
ON CONFLICT (id) DO NOTHING;

INSERT INTO applications (id, workspace_id, name) VALUES
    ('app_policy_lab', 'ws_default', 'Change Policy Lab')
ON CONFLICT (id) DO NOTHING;

INSERT INTO machines (id, application_id, name) VALUES
    ('mch_policy_case', 'app_policy_lab', 'Policy Case')
ON CONFLICT (id) DO NOTHING;

INSERT INTO fields (id, machine_id, name, type, position, required, options) VALUES
    ('fld_pc_title',        'mch_policy_case', 'Title',              'text', 0, true,  '{}'),
    ('fld_pc_compliance',   'mch_policy_case', 'Compliance Note',    'text', 1, false, '{}'),
    ('fld_pc_approval_ref', 'mch_policy_case', 'Approval Reference', 'text', 2, false, '{}'),
    ('fld_pc_status',       'mch_policy_case', 'Status',             'value_list', 3, false,
     '{"values":["Draft","Submitted","Closed"]}')
ON CONFLICT (id) DO NOTHING;

INSERT INTO permissions (id, machine_id, role, events, can_read, can_create, can_edit, can_delete) VALUES
    ('perm_pc_editor', 'mch_policy_case', 'Editor', ARRAY[]::TEXT[], true, true, true, false)
ON CONFLICT (id) DO NOTHING;

INSERT INTO views (id, machine_id, name, type, position, config) VALUES
    ('vw_pc_list',   'mch_policy_case', 'Policy Cases',      'list',   0, '{"columns":["fld_pc_title","fld_pc_status"]}'),
    ('vw_pc_form',   'mch_policy_case', 'New Policy Case',   'form',   1, '{"fields":["fld_pc_title","fld_pc_compliance","fld_pc_approval_ref"]}'),
    ('vw_pc_detail', 'mch_policy_case', 'Policy Case Detail', 'detail', 2, '{}')
ON CONFLICT (id) DO NOTHING;

INSERT INTO user_application_roles (user_id, application_id, role)
SELECT u.id, 'app_policy_lab', 'Editor' FROM users u WHERE u.email = 'hr@example.com'
ON CONFLICT (user_id, application_id) DO UPDATE SET role = EXCLUDED.role;

-- Baseline records. RLS (migrations/009, applied on this shared dev
-- database even though it's deliberately excluded from a fresh `make
-- migrate-up`) forces every non-superuser connection to set
-- app.workspace_id before touching `records` -- set it explicitly rather
-- than relying on the connection's default role.
SET app.workspace_id = 'ws_default';

INSERT INTO records (id, machine_id, workspace_id, data, created_at, updated_at) VALUES
    -- currently Draft -- T154 (no rule yet) / T155 (records_in_states gates it)
    ('11111111-1111-1111-1111-111111111101', 'mch_policy_case', 'ws_default',
     '{"fld_pc_title":"Draft case","fld_pc_status":"Draft"}', NOW(), NOW()),
    -- already past Draft when 025's rule arrives -- T156 (grandfathered)
    ('11111111-1111-1111-1111-111111111102', 'mch_policy_case', 'ws_default',
     '{"fld_pc_title":"Submitted case","fld_pc_status":"Submitted"}', NOW(), NOW()),
    -- created well before 025's new_records effective_from (2026-01-01) --
    -- T157 (new_records policy doesn't reach it)
    ('11111111-1111-1111-1111-111111111103', 'mch_policy_case', 'ws_default',
     '{"fld_pc_title":"Old case","fld_pc_status":"Draft"}', '2025-01-01', '2025-01-01')
ON CONFLICT (id) DO NOTHING;
