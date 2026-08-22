-- seeds/032_kanban_lab.sql
-- CAP-V14 Tier 2 proof: a "board" View groups records into lanes from an
-- existing value_list Field (ViewConfig.GroupField) -- narrower than Case
-- 19's own "user-creatable Lists" model, a deliberate scope cut named in
-- capability-registry.md's own CAP-V14 row. Two lanes ("Todo", "Doing")
-- start with records, "Done" starts empty -- proving every declared option
-- gets a lane even with nothing in it (BoardLane's own doc comment).

INSERT INTO workspaces (id, name) VALUES ('ws_default', 'Default Workspace')
ON CONFLICT (id) DO NOTHING;

INSERT INTO applications (id, workspace_id, name) VALUES
    ('app_kanban_lab', 'ws_default', 'Kanban Lab')
ON CONFLICT (id) DO NOTHING;

INSERT INTO machines (id, application_id, name) VALUES
    ('mch_kanban_task', 'app_kanban_lab', 'Task')
ON CONFLICT (id) DO NOTHING;

INSERT INTO fields (id, machine_id, name, type, position, required, options) VALUES
    ('fld_kbt_title',  'mch_kanban_task', 'Title',  'text',       0, true,  '{}'),
    ('fld_kbt_status', 'mch_kanban_task', 'Status', 'value_list', 1, false, '{"values":["Todo","Doing","Done"]}')
ON CONFLICT (id) DO NOTHING;

INSERT INTO permissions (id, machine_id, role, events, can_read, can_create, can_edit, can_delete) VALUES
    ('perm_kbt_lead', 'mch_kanban_task', 'Lead', ARRAY[]::TEXT[], true, true, true, false)
ON CONFLICT (id) DO NOTHING;

INSERT INTO views (id, machine_id, name, type, position, config) VALUES
    ('vw_kbt_board', 'mch_kanban_task', 'Task Board', 'board', 0,
     '{"columns":["fld_kbt_title"],"group_field":"fld_kbt_status"}'),
    ('vw_kbt_form', 'mch_kanban_task', 'New Task', 'form', 1,
     '{"fields":["fld_kbt_title","fld_kbt_status"]}')
ON CONFLICT (id) DO NOTHING;

INSERT INTO users (workspace_id, name, email, password_hash, workspace_role) VALUES
    ('ws_default', 'Kanban Lead', 'kanban.lead@example.com',
     '$2a$10$moxxOcZzSu3ILTzJlLF2Q.9vxiGNnSXPl7kY1pT3t5o1FoDjqC8aK', 'Member')
ON CONFLICT (workspace_id, email) DO NOTHING;

INSERT INTO user_application_roles (user_id, application_id, role)
SELECT u.id, 'app_kanban_lab', 'Lead' FROM users u WHERE u.email = 'kanban.lead@example.com'
ON CONFLICT (user_id, application_id) DO UPDATE SET role = EXCLUDED.role;

-- Two Todo, one Doing, zero Done -- RLS (migrations/009) requires
-- app.workspace_id set before touching `records`, same pattern
-- seeds/031_typeahead_lab.sql already established.
SET app.workspace_id = 'ws_default';

INSERT INTO records (id, machine_id, workspace_id, data, created_at, updated_at) VALUES
    ('22222222-3333-4444-5555-000000000001', 'mch_kanban_task', 'ws_default',
     '{"fld_kbt_title":"Write proposal","fld_kbt_status":"Todo"}', NOW(), NOW()),
    ('22222222-3333-4444-5555-000000000002', 'mch_kanban_task', 'ws_default',
     '{"fld_kbt_title":"Review budget","fld_kbt_status":"Todo"}', NOW(), NOW()),
    ('22222222-3333-4444-5555-000000000003', 'mch_kanban_task', 'ws_default',
     '{"fld_kbt_title":"Draft contract","fld_kbt_status":"Doing"}', NOW(), NOW())
ON CONFLICT (id) DO NOTHING;
