-- seeds/010_views_lab.sql
-- Proof harness for Batch 4 (2026-07-12): the Views cluster --
-- CAP-V04 (default_sort honored), CAP-V05 (filter by current user),
-- CAP-V07 (calendar/timeline), CAP-V08 (search -- generic, no seed data of
-- its own needed, proven directly against an existing list view),
-- CAP-V09 (declarative filter), CAP-V10 (composed dashboard), CAP-V12
-- (multi-step wizard form), CAP-V13 (aggregate report), CAP-V14 (manual
-- ordering). V11 (channel-independent rendering) is deliberately excluded
-- -- capability-registry.md already HOLDs it at Proposed pending a second
-- independent source, and this batch doesn't override that.
--
-- V04/V05/V09 are combined into ONE view ("My Overdue Tasks" -- filtered to
-- the acting user AND due before today, sorted soonest-due-first) rather
-- than three separate machines: only the FIRST list-type View on a Machine
-- is actually reachable (router.go's /{machineID} -> DefaultListView), so
-- proving three distinct list-view behaviors needs either three Machines or
-- one realistic view that genuinely needs all three together -- the latter
-- is also just a more honest example of how these compose in practice.
-- V14 needs its own Machine (ManualOrder replaces DefaultSort/Filter
-- entirely when set, handler.go's List). V07/V10/V13 reuse Batch 2/3's
-- existing Machines (Journal Entry Line, Task, Project) rather than
-- inventing new ones -- a report/calendar/dashboard is only interesting
-- over data that already looks like a real case.

INSERT INTO applications (id, workspace_id, name) VALUES
    ('app_views_lab', 'ws_default', 'Views Lab')
ON CONFLICT (id) DO NOTHING;

INSERT INTO machines (id, application_id, name) VALUES
    ('mch_vl_task',       'app_views_lab', 'Task'),
    ('mch_vl_backlog',    'app_views_lab', 'Backlog Item'),
    ('mch_vl_onboarding', 'app_views_lab', 'Onboarding Request')
ON CONFLICT (id) DO NOTHING;

-- Fields
INSERT INTO fields (id, machine_id, name, type, position, required, options) VALUES
    ('fld_vlt_title',    'mch_vl_task', 'Title',    'text', 0, true,  '{}'),
    ('fld_vlt_assignee', 'mch_vl_task', 'Assignee', 'user', 1, false, '{}'),
    ('fld_vlt_due',      'mch_vl_task', 'Due Date', 'date', 2, false, '{}'),

    ('fld_vlb_title', 'mch_vl_backlog', 'Title', 'text', 0, true, '{}'),

    ('fld_vlo_name',  'mch_vl_onboarding', 'Full Name', 'text',       0, true, '{}'),
    ('fld_vlo_email', 'mch_vl_onboarding', 'Email',     'text',       1, true, '{}'),
    ('fld_vlo_team',  'mch_vl_onboarding', 'Team',      'value_list', 2, true, '{"values":["Engineering","Sales","Support"]}'),
    ('fld_vlo_start', 'mch_vl_onboarding', 'Start Date','date',       3, true, '{}')
ON CONFLICT (id) DO NOTHING;

-- Permissions (can_read/can_create/can_edit default true, migrations/006).
INSERT INTO permissions (id, machine_id, role, events) VALUES
    ('perm_vlt_member', 'mch_vl_task',       'Member', ARRAY[]::TEXT[]),
    ('perm_vlb_member', 'mch_vl_backlog',    'Member', ARRAY[]::TEXT[]),
    ('perm_vlo_member', 'mch_vl_onboarding', 'Member', ARRAY[]::TEXT[])
ON CONFLICT (id) DO NOTHING;

-- Views
INSERT INTO views (id, machine_id, name, type, position, config) VALUES
    ('vw_vlt_form', 'mch_vl_task', 'Task Form', 'form', 0,
     '{"fields":["fld_vlt_title","fld_vlt_assignee","fld_vlt_due"]}'),
    -- CAP-V04+V05+V09: "My Overdue Tasks" -- filter AND-combines a
    -- $current_user ownership clause (V05) with a plain declarative clause
    -- (V09, due before today), sorted soonest-due-first (V04).
    ('vw_vlt_list', 'mch_vl_task', 'My Overdue Tasks', 'list', 1,
     '{"columns":["fld_vlt_title","fld_vlt_assignee","fld_vlt_due"],"default_sort":{"field":"fld_vlt_due","direction":"asc"},"filter":[{"field":"fld_vlt_assignee","operator":"equals","value":"$current_user"},{"field":"fld_vlt_due","operator":"before","value":"today"}]}'),
    ('vw_vlt_detail', 'mch_vl_task', 'Task Detail', 'detail', 2, '{}'),

    ('vw_vlb_form', 'mch_vl_backlog', 'Backlog Item Form', 'form', 0,
     '{"fields":["fld_vlb_title"]}'),
    -- CAP-V14: manual free ordering -- Up/Down controls, sort_order column.
    ('vw_vlb_list', 'mch_vl_backlog', 'Backlog', 'list', 1,
     '{"columns":["fld_vlb_title"],"manual_order":true}'),
    ('vw_vlb_detail', 'mch_vl_backlog', 'Backlog Item Detail', 'detail', 2, '{}'),

    -- CAP-V12: a 2-step wizard -- Steps replaces Fields; each step shows
    -- only its own subset, the final POST creates like any other form.
    ('vw_vlo_form', 'mch_vl_onboarding', 'Onboarding Form', 'form', 0,
     '{"steps":[["fld_vlo_name","fld_vlo_email"],["fld_vlo_team","fld_vlo_start"]]}'),
    ('vw_vlo_list', 'mch_vl_onboarding', 'Onboarding Requests', 'list', 1,
     '{"columns":["fld_vlo_name","fld_vlo_team","fld_vlo_start"]}'),
    ('vw_vlo_detail', 'mch_vl_onboarding', 'Onboarding Request Detail', 'detail', 2, '{}'),

    -- CAP-V07: Task's own Follow Up Date (seeds/009) grouped/ordered on.
    ('vw_alt_calendar', 'mch_al_task', 'Task Calendar', 'calendar', 3,
     '{"columns":["fld_alt_title","fld_alt_priority"],"date_field":"fld_alt_follow_up"}'),
    ('vw_alt_timeline', 'mch_al_task', 'Task Timeline', 'timeline', 4,
     '{"columns":["fld_alt_title","fld_alt_priority"],"date_field":"fld_alt_follow_up"}'),

    -- CAP-V10: composed dashboard over TWO different Action Lab Machines
    -- (seeds/009) -- the actual point of "composed," not a single
    -- Machine's own summary.
    ('vw_alp_dashboard', 'mch_al_project', 'Action Lab Dashboard', 'dashboard', 3,
     '{"sections":[{"title":"Tasks by Stage","machine":"mch_al_task","group_field":"fld_alt_stage"},{"title":"Projects","machine":"mch_al_project"}]}'),

    -- CAP-V13: Trial-Balance-style report over Journal Entry Line
    -- (seeds/008) -- grouped by Account, summing Debit/Credit.
    ('vw_jel_report', 'mch_journal_entry_line', 'Trial Balance', 'report', 2,
     '{"report":{"machine":"mch_journal_entry_line","group_field":"fld_jel_account","sum_fields":["fld_jel_debit","fld_jel_credit"]}}')
ON CONFLICT (id) DO NOTHING;

-- Account (self-contained, same reasoning as seeds/008/009's own accounts).
INSERT INTO users (workspace_id, name, email, password_hash, workspace_role) VALUES
    ('ws_default', 'Vera', 'vera@example.com',
     '$2a$10$moxxOcZzSu3ILTzJlLF2Q.9vxiGNnSXPl7kY1pT3t5o1FoDjqC8aK', 'Member')
ON CONFLICT (workspace_id, email) DO NOTHING;

INSERT INTO user_application_roles (user_id, application_id, role)
SELECT u.id, 'app_views_lab', 'Member' FROM users u WHERE u.email = 'vera@example.com'
ON CONFLICT (user_id, application_id) DO UPDATE SET role = EXCLUDED.role;
