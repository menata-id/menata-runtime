-- seeds/015_workspace_services_lab.sql
-- Proof harness for Batch 9 (2026-07-12): CAP-O02 (master data
-- designation), CAP-O04 (workspace-wide search), CAP-O05 (unified
-- notification center), CAP-O06 (business calendar). CAP-O01/CAP-O03
-- shipped earlier, not part of this batch.
--
-- Employee (app_workspace_lab_hr) and Project (app_workspace_lab_ops) are
-- deliberately TWO SEPARATE Applications -- CAP-O02's own case is
-- specifically cross-app referenceability (Case 10), not same-app
-- reference (already fully covered by CAP-F13 alone).

INSERT INTO applications (id, workspace_id, name) VALUES
    ('app_workspace_lab_hr',  'ws_default', 'Workspace Lab HR'),
    ('app_workspace_lab_ops', 'ws_default', 'Workspace Lab Ops')
ON CONFLICT (id) DO NOTHING;

INSERT INTO machines (id, application_id, name, config) VALUES
    -- CAP-O02: master_data flags this Machine as canonical/cross-app --
    -- Archive is blocked while any OTHER record, on ANY Machine, still
    -- references it (handler.setDeleted).
    ('mch_wsx_employee', 'app_workspace_lab_hr', 'Employee', '{"master_data":"true"}'),
    ('mch_wsx_task',     'app_workspace_lab_hr', 'Scheduled Task', NULL),
    ('mch_wsx_project',  'app_workspace_lab_ops', 'Project', NULL)
ON CONFLICT (id) DO NOTHING;

-- Fields
INSERT INTO fields (id, machine_id, name, type, position, required, options) VALUES
    ('fld_wsxe_name', 'mch_wsx_employee', 'Name', 'text', 0, true, '{}'),

    ('fld_wsxt_title',       'mch_wsx_task', 'Title',       'text', 0, true,  '{}'),
    ('fld_wsxt_target_date', 'mch_wsx_task', 'Target Date', 'date', 1, false, '{}'),

    ('fld_wsxp_title', 'mch_wsx_project', 'Title', 'text',      0, true,  '{}'),
    -- CAP-O02's own proof: a `reference` field on a Machine in a
    -- DIFFERENT Application, targeting the master-data Machine --
    -- cross-app referenceability already works via CAP-F13 alone, this
    -- just proves it deliberately rather than assuming it.
    ('fld_wsxp_lead',  'mch_wsx_project', 'Lead',  'reference', 1, false, '{"target_machine":"mch_wsx_employee"}')
ON CONFLICT (id) DO NOTHING;

-- Events. CAP-O06: "N Business Days" skips weekends (and any date in
-- workspace_holidays, migrations/016) -- the flat date-arithmetic family
-- CAP-A11 already established, one more unit.
INSERT INTO events (id, machine_id, name, position, condition) VALUES
    ('evt_wsxt_schedule', 'mch_wsx_task', 'Schedule', 0, NULL)
ON CONFLICT (id) DO NOTHING;

INSERT INTO event_actions (event_id, type, position, params) VALUES
    ('evt_wsxt_schedule', 'set_field', 0, '{"field":"fld_wsxt_target_date","value":"today + 5 Business Days"}');

-- Permissions. can_delete explicit true on Employee -- CAP-O02's own
-- Archive-block proof needs it.
INSERT INTO permissions (id, machine_id, role, events, can_read, can_create, can_edit, can_delete) VALUES
    ('perm_wsxe_member', 'mch_wsx_employee', 'Member', ARRAY[]::TEXT[],              true, true, true, true),
    ('perm_wsxt_member', 'mch_wsx_task',     'Member', ARRAY['evt_wsxt_schedule'],   true, true, true, false),
    ('perm_wsxp_member', 'mch_wsx_project',  'Member', ARRAY[]::TEXT[],              true, true, true, false)
ON CONFLICT (id) DO NOTHING;

-- Views
INSERT INTO views (id, machine_id, name, type, position, config) VALUES
    ('vw_wsxe_form',   'mch_wsx_employee', 'Employee Form', 'form', 0, '{"fields":["fld_wsxe_name"]}'),
    ('vw_wsxe_list',   'mch_wsx_employee', 'Employees',     'list', 1, '{"columns":["fld_wsxe_name"]}'),
    ('vw_wsxe_detail', 'mch_wsx_employee', 'Employee Detail', 'detail', 2, '{}'),

    ('vw_wsxt_form',   'mch_wsx_task', 'Scheduled Task Form', 'form', 0, '{"fields":["fld_wsxt_title"]}'),
    ('vw_wsxt_list',   'mch_wsx_task', 'Scheduled Tasks',     'list', 1, '{"columns":["fld_wsxt_title","fld_wsxt_target_date"]}'),
    ('vw_wsxt_detail', 'mch_wsx_task', 'Scheduled Task Detail', 'detail', 2, '{}'),

    ('vw_wsxp_form',   'mch_wsx_project', 'Project Form', 'form', 0, '{"fields":["fld_wsxp_title","fld_wsxp_lead"]}'),
    ('vw_wsxp_list',   'mch_wsx_project', 'Projects',     'list', 1, '{"columns":["fld_wsxp_title","fld_wsxp_lead"]}'),
    ('vw_wsxp_detail', 'mch_wsx_project', 'Project Detail', 'detail', 2, '{}')
ON CONFLICT (id) DO NOTHING;

-- Account (self-contained, same reasoning as prior lab seed files).
INSERT INTO users (workspace_id, name, email, password_hash, workspace_role) VALUES
    ('ws_default', 'Yara', 'yara@example.com', '$2a$10$moxxOcZzSu3ILTzJlLF2Q.9vxiGNnSXPl7kY1pT3t5o1FoDjqC8aK', 'Member')
ON CONFLICT (workspace_id, email) DO NOTHING;

INSERT INTO user_application_roles (user_id, application_id, role)
SELECT u.id, app.id, 'Member' FROM users u, (VALUES ('app_workspace_lab_hr'), ('app_workspace_lab_ops')) AS app(id)
WHERE u.email = 'yara@example.com'
ON CONFLICT (user_id, application_id) DO UPDATE SET role = EXCLUDED.role;
