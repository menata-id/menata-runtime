-- seeds/006_second_workspace.sql
-- CAP-X06: a second, genuinely separate Workspace -- without one actually
-- existing, "multi-workspace isolation works" cannot be verified at all,
-- only asserted. Deliberately minimal (one application, one machine) --
-- this exists to prove isolation (conformance T-series, RLS probe), not as
-- a real business case like Cases 1-21. Same shape as seeds/001's Design
-- Request, smaller.

INSERT INTO workspaces (id, name, slug) VALUES
    ('ws_acme', 'Acme Corp', 'ws_acme')
ON CONFLICT (id) DO NOTHING;

INSERT INTO applications (id, workspace_id, name) VALUES
    ('app_ops', 'ws_acme', 'Operations')
ON CONFLICT (id) DO NOTHING;

INSERT INTO machines (id, application_id, name) VALUES
    ('mch_task', 'app_ops', 'Task')
ON CONFLICT (id) DO NOTHING;

INSERT INTO fields (id, machine_id, name, type, position, required, options) VALUES
    ('fld_task_title',  'mch_task', 'Title',  'text',       0, true,  '{}'),
    ('fld_task_status', 'mch_task', 'Status', 'value_list', 1, false, '{"values":["Open","Done"]}')
ON CONFLICT (id) DO NOTHING;

INSERT INTO events (id, machine_id, name, position) VALUES
    ('evt_task_complete', 'mch_task', 'Complete', 0)
ON CONFLICT (id) DO NOTHING;

INSERT INTO event_actions (event_id, type, position, params) VALUES
    ('evt_task_complete', 'set_field', 0, '{"field":"fld_task_status","value":"Done"}');

INSERT INTO constraints (id, machine_id, rule, expression, condition, position) VALUES
    ('cst_task_title_required', 'mch_task', 'Title is required.',
     '{"field":"fld_task_title","operator":"required"}', NULL, 0)
ON CONFLICT (id) DO NOTHING;

-- Staff: can_read/can_create/can_edit default true (migrations/006).
INSERT INTO permissions (id, machine_id, role, events) VALUES
    ('perm_task_staff', 'mch_task', 'Staff', ARRAY['evt_task_complete'])
ON CONFLICT (id) DO NOTHING;

INSERT INTO views (id, machine_id, name, type, position, config) VALUES
    ('vw_task_form',   'mch_task', 'Task Form',   'form',   0, '{"fields":["fld_task_title"]}'),
    ('vw_task_list',   'mch_task', 'Tasks',       'list',   1, '{"columns":["fld_task_title","fld_task_status"]}'),
    ('vw_task_detail', 'mch_task', 'Task Detail', 'detail', 2, '{}')
ON CONFLICT (id) DO NOTHING;
