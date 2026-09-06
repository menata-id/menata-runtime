-- seeds/009_action_lab.sql
-- Deliberately minimal proof harness for Batch 3's Actions cluster
-- (2026-07-12): CAP-A06 (create_record), CAP-A09 (conditional "if"
-- actions), CAP-A11 (date arithmetic), CAP-A12 (value_list "next"
-- stepping), CAP-A13 (cross-record set_field), CAP-A15 (batch/series
-- generation) all fire from ONE event (evt_al_task_complete) so one HTTP
-- trigger proves all six at once -- not a full project-management case,
-- the same "reduced slice, real forcing case" discipline as
-- seeds/005_complaint.sql. CAP-A14 (aggregate-conditioned action) is a
-- genuinely different pattern (a GATE, not an action), proven separately
-- below via Point Entry/Badge, mirroring the community-points.yaml case
-- this capability was originally discovered from.

INSERT INTO workspaces (id, name) VALUES
    ('ws_default', 'Default Workspace')
ON CONFLICT (id) DO NOTHING;

INSERT INTO applications (id, workspace_id, name) VALUES
    ('app_action_lab', 'ws_default', 'Action Lab')
ON CONFLICT (id) DO NOTHING;

INSERT INTO machines (id, application_id, name) VALUES
    ('mch_al_project',  'app_action_lab', 'Project'),
    ('mch_al_task',     'app_action_lab', 'Task'),
    ('mch_al_task_log', 'app_action_lab', 'Task Log'),
    ('mch_al_point_entry', 'app_action_lab', 'Point Entry'),
    ('mch_al_badge',    'app_action_lab', 'Badge')
ON CONFLICT (id) DO NOTHING;

-- Fields
INSERT INTO fields (id, machine_id, name, type, position, required, options) VALUES
    ('fld_alp_name',          'mch_al_project', 'Name',          'text', 0, true,  '{}'),
    ('fld_alp_last_activity', 'mch_al_project', 'Last Activity', 'date', 1, false, '{}'),

    ('fld_alt_title',    'mch_al_task', 'Title',    'text',       0, true,  '{}'),
    ('fld_alt_priority',  'mch_al_task', 'Priority', 'value_list', 1, false, '{"values":["Normal","Urgent"]}'),
    ('fld_alt_stage',     'mch_al_task', 'Stage',    'value_list', 2, false, '{"values":["Todo","Doing","Done"]}'),
    ('fld_alt_follow_up', 'mch_al_task', 'Follow Up Date', 'date', 3, false, '{}'),
    ('fld_alt_project',   'mch_al_task', 'Project',  'reference',  4, false,
     '{"target_machine":"mch_al_project"}'),

    ('fld_altl_title', 'mch_al_task_log', 'Title', 'text', 0, true, '{}'),
    ('fld_altl_note',  'mch_al_task_log', 'Note',  'text', 1, false, '{}'),

    ('fld_alpe_member', 'mch_al_point_entry', 'Member', 'text',   0, true, '{}'),
    ('fld_alpe_points',  'mch_al_point_entry', 'Points', 'number', 1, true, '{}'),

    ('fld_alb_member', 'mch_al_badge', 'Member', 'text', 0, true, '{}')
ON CONFLICT (id) DO NOTHING;

-- Events
-- evt_al_task_complete bundles CAP-A06/A09/A11/A12/A13/A15 -- see action list below.
INSERT INTO events (id, machine_id, name, position, condition) VALUES
    ('evt_al_task_complete', 'mch_al_task', 'Complete', 0, NULL),
    -- CAP-A14: this event's own "condition" column holds an AggregateCondition
    -- (detected by the aggregate_field key, metadata.Loader's loadEvents),
    -- not a plain CAP-E06 ConstraintExpression -- gates the whole trigger on
    -- SUM(fld_alpe_points) across THIS SAME Member's own other Point Entry
    -- rows (machine omitted -- defaults to this event's own Machine, the
    -- common case: aggregate condition and triggering event on the same
    -- Machine, exactly the community-points.yaml case this mirrors).
    -- Its own action creates the Badge (CAP-A06 again) once the gate passes.
    ('evt_pe_award', 'mch_al_point_entry', 'Award Badge', 0,
     '{"aggregate_field":"fld_alpe_points","scope_field":"fld_alpe_member","operator":"greater_than_or_equal","value":"100"}')
ON CONFLICT (id) DO NOTHING;

INSERT INTO event_actions (event_id, type, position, params) VALUES
    -- CAP-A12: Stage advances to the next declared value_list option.
    ('evt_al_task_complete', 'set_field', 0, '{"field":"fld_alt_stage","value":"next"}'),
    -- CAP-A11: Follow Up Date = today + 7 Days (flat date arithmetic).
    ('evt_al_task_complete', 'set_field', 1, '{"field":"fld_alt_follow_up","value":"today + 7 Days"}'),
    -- CAP-A09: only notifies when Priority is Urgent -- "if" gates the action,
    -- not the whole event.
    ('evt_al_task_complete', 'notify', 2, '{"role":"PM","if":{"field":"fld_alt_priority","operator":"equals","value":"Urgent"}}'),
    -- CAP-A06: creates a Task Log entry, copying this Task's own Title via
    -- "field:<id>" (not template interpolation -- see executor.go's own note).
    ('evt_al_task_complete', 'create_record', 3,
     '{"machine":"mch_al_task_log","fields":{"fld_altl_title":"field:fld_alt_title","fld_altl_note":"Completed"}}'),
    -- CAP-A13: stamps the linked Project's own Last Activity field --
    -- record_field names the Task's OWN reference field to the Project.
    ('evt_al_task_complete', 'cross_set_field', 4,
     '{"record_field":"fld_alt_project","field":"fld_alp_last_activity","value":"today"}'),
    -- CAP-A15: creates 2 follow-up Tasks from one action.
    ('evt_al_task_complete', 'batch_generate', 5,
     '{"machine":"mch_al_task","count":2,"fields":{"fld_alt_title":"Follow-up","fld_alt_stage":"Todo"}}'),

    -- CAP-A14's own action, once the aggregate gate above passes: create the
    -- Badge, copying Member from this Point Entry (CAP-A06 again).
    ('evt_pe_award', 'create_record', 0,
     '{"machine":"mch_al_badge","fields":{"fld_alb_member":"field:fld_alpe_member"}}');

-- Permissions
INSERT INTO permissions (id, machine_id, role, events) VALUES
    ('perm_alp_pm',  'mch_al_project',      'PM', ARRAY[]::TEXT[]),
    ('perm_alt_pm',  'mch_al_task',         'PM', ARRAY['evt_al_task_complete']),
    ('perm_altl_pm', 'mch_al_task_log',     'PM', ARRAY[]::TEXT[]),
    ('perm_alpe_pm', 'mch_al_point_entry',  'PM', ARRAY['evt_pe_award']),
    ('perm_alb_pm',  'mch_al_badge',        'PM', ARRAY[]::TEXT[])
ON CONFLICT (id) DO NOTHING;

-- Views
INSERT INTO views (id, machine_id, name, type, position, config) VALUES
    ('vw_alp_form',   'mch_al_project', 'Project Form', 'form', 0, '{"fields":["fld_alp_name"]}'),
    ('vw_alp_list',   'mch_al_project', 'Projects',     'list', 1, '{"columns":["fld_alp_name","fld_alp_last_activity"]}'),
    ('vw_alp_detail', 'mch_al_project', 'Project Detail', 'detail', 2, '{}'),

    ('vw_alt_form',   'mch_al_task', 'Task Form', 'form', 0,
     '{"fields":["fld_alt_title","fld_alt_priority","fld_alt_stage","fld_alt_project"]}'),
    ('vw_alt_list',   'mch_al_task', 'Tasks',     'list', 1,
     '{"columns":["fld_alt_title","fld_alt_priority","fld_alt_stage","fld_alt_follow_up"]}'),
    ('vw_alt_detail', 'mch_al_task', 'Task Detail', 'detail', 2, '{}'),

    ('vw_altl_list',   'mch_al_task_log', 'Task Logs', 'list', 0, '{"columns":["fld_altl_title","fld_altl_note"]}'),
    ('vw_altl_detail', 'mch_al_task_log', 'Task Log Detail', 'detail', 1, '{}'),

    ('vw_alpe_form',   'mch_al_point_entry', 'Point Entry Form', 'form', 0,
     '{"fields":["fld_alpe_member","fld_alpe_points"]}'),
    ('vw_alpe_list',   'mch_al_point_entry', 'Point Entries', 'list', 1,
     '{"columns":["fld_alpe_member","fld_alpe_points"]}'),
    ('vw_alpe_detail', 'mch_al_point_entry', 'Point Entry Detail', 'detail', 2, '{}'),

    ('vw_alb_form',   'mch_al_badge', 'Badge Form', 'form', 0, '{"fields":["fld_alb_member"]}'),
    ('vw_alb_list',   'mch_al_badge', 'Badges',     'list', 1, '{"columns":["fld_alb_member"]}'),
    ('vw_alb_detail', 'mch_al_badge', 'Badge Detail', 'detail', 2, '{}')
ON CONFLICT (id) DO NOTHING;

-- Account (self-contained, same reasoning as seeds/008's Ivy -- this file
-- runs after seeds/007_authentication.sql in `make seed`'s ordering, and
-- referencing app_action_lab there would violate the applications FK).
INSERT INTO users (workspace_id, name, email, password_hash, workspace_role) VALUES
    ('ws_default', 'Pam', 'pm@example.com',
     '$2a$10$moxxOcZzSu3ILTzJlLF2Q.9vxiGNnSXPl7kY1pT3t5o1FoDjqC8aK', 'Member')
ON CONFLICT (workspace_id, email) DO NOTHING;

INSERT INTO user_application_roles (user_id, application_id, role)
SELECT u.id, 'app_action_lab', 'PM' FROM users u WHERE u.email = 'pm@example.com'
ON CONFLICT (user_id, application_id) DO UPDATE SET role = EXCLUDED.role;
