-- seeds/030_resource_calendar_lab.sql
-- CAP-V18 proof: a calendar View grouped by a second dimension (a resource
-- reference field) on top of CAP-V07's existing date_field grouping --
-- extends the same "sort then linear-scan flush on change" algorithm with
-- one more key, no new View type.

INSERT INTO workspaces (id, name) VALUES ('ws_default', 'Default Workspace')
ON CONFLICT (id) DO NOTHING;

INSERT INTO applications (id, workspace_id, name) VALUES
    ('app_resource_cal_lab', 'ws_default', 'Resource Calendar Lab')
ON CONFLICT (id) DO NOTHING;

INSERT INTO machines (id, application_id, name) VALUES
    ('mch_v18_staff',       'app_resource_cal_lab', 'Staff'),
    ('mch_v18_appointment', 'app_resource_cal_lab', 'Appointment')
ON CONFLICT (id) DO NOTHING;

INSERT INTO fields (id, machine_id, name, type, position, required, options) VALUES
    ('fld_v18s_name', 'mch_v18_staff', 'Name', 'text', 0, true, '{}'),

    ('fld_v18a_title', 'mch_v18_appointment', 'Title', 'text', 0, true, '{}'),
    ('fld_v18a_staff', 'mch_v18_appointment', 'Staff', 'reference', 1, true, '{"target_machine":"mch_v18_staff"}'),
    ('fld_v18a_date',  'mch_v18_appointment', 'Date',  'date', 2, true, '{}')
ON CONFLICT (id) DO NOTHING;

INSERT INTO permissions (id, machine_id, role, events, can_read, can_create, can_edit, can_delete) VALUES
    ('perm_v18s_scheduler', 'mch_v18_staff',       'Scheduler', ARRAY[]::TEXT[], true, true, true, false),
    ('perm_v18a_scheduler', 'mch_v18_appointment',  'Scheduler', ARRAY[]::TEXT[], true, true, true, false)
ON CONFLICT (id) DO NOTHING;

INSERT INTO views (id, machine_id, name, type, position, config) VALUES
    ('vw_v18s_list', 'mch_v18_staff', 'Staff', 'list', 0, '{"columns":["fld_v18s_name"]}'),
    ('vw_v18s_form', 'mch_v18_staff', 'New Staff', 'form', 1, '{"fields":["fld_v18s_name"]}'),
    ('vw_v18a_form', 'mch_v18_appointment', 'New Appointment', 'form', 0, '{"fields":["fld_v18a_title","fld_v18a_staff","fld_v18a_date"]}'),
    ('vw_v18a_calendar', 'mch_v18_appointment', 'Schedule', 'calendar', 1,
     '{"columns":["fld_v18a_title"],"date_field":"fld_v18a_date","resource_field":"fld_v18a_staff"}')
ON CONFLICT (id) DO NOTHING;

INSERT INTO users (workspace_id, name, email, password_hash, workspace_role) VALUES
    ('ws_default', 'Resource Scheduler', 'resourcecal.scheduler@example.com',
     '$2a$10$moxxOcZzSu3ILTzJlLF2Q.9vxiGNnSXPl7kY1pT3t5o1FoDjqC8aK', 'Member')
ON CONFLICT (workspace_id, email) DO NOTHING;

INSERT INTO user_application_roles (user_id, application_id, role)
SELECT u.id, 'app_resource_cal_lab', 'Scheduler' FROM users u WHERE u.email = 'resourcecal.scheduler@example.com'
ON CONFLICT (user_id, application_id) DO UPDATE SET role = EXCLUDED.role;
