-- seeds/052_project_management.sql
-- Phase 13 (composable-runtime-roadmap.md §17, Trial Migration) -- Case 19
-- (Project Management), seeded at composable-substrate-proof scope, NOT a
-- faithful Trello-style board. Owner decision (2026-09-11, conversation):
-- a full realization needs new capability work capability-registry.md's
-- own CAP-V14 Tier 2 row already names as out of scope for the real board
-- view ("a deliberately narrower cut than Case 19's own literal
-- declaration... Case 19's own 'Lists' are separate, user-creatable,
-- freely-reordered records" -- the real board only groups by a fixed
-- value_list field, never a dynamic reference). That admission decision
-- is explicitly NOT made here.
--
-- Instead: Board -> List -> Card -> Checklist Item via CAP-F13 reference
-- fields, CAP-V06 child-lists (a Board's own Detail already lists every
-- List referencing it; a List's own Detail already lists every Card
-- referencing it -- the existing mechanism, no new code), and CAP-F16
-- ChildLines for Checklist Items embedded in a Card's own form (the
-- Journal-Entry-Lines precedent, verbatim). No drag-and-drop, no scoped
-- manual ordering.

INSERT INTO workspaces (id, name, slug) VALUES ('ws_default', 'Default Workspace', 'ws_default')
ON CONFLICT (id) DO NOTHING;

INSERT INTO applications (id, workspace_id, name) VALUES
    ('app_project_management', 'ws_default', 'Project Management')
ON CONFLICT (id) DO NOTHING;

INSERT INTO machines (id, application_id, name) VALUES
    ('mch_pm_board',           'app_project_management', 'Board'),
    ('mch_pm_list',            'app_project_management', 'List'),
    ('mch_pm_card',            'app_project_management', 'Card'),
    ('mch_pm_checklist_item',  'app_project_management', 'Checklist Item')
ON CONFLICT (id) DO NOTHING;

INSERT INTO fields (id, machine_id, name, type, position, required, options) VALUES
    ('fld_pmb_name',  'mch_pm_board', 'Name',  'text', 0, true,  '{}'),
    ('fld_pmb_owner', 'mch_pm_board', 'Owner', 'user', 1, false, '{}'),

    ('fld_pml_board', 'mch_pm_list', 'Board', 'reference', 0, true,  '{"target_machine":"mch_pm_board"}'),
    ('fld_pml_name',  'mch_pm_list', 'Name',  'text',      1, true,  '{}'),

    ('fld_pmc_list',        'mch_pm_card', 'List',        'reference', 0, true,  '{"target_machine":"mch_pm_list"}'),
    ('fld_pmc_title',       'mch_pm_card', 'Title',       'text',      1, true,  '{}'),
    ('fld_pmc_description', 'mch_pm_card', 'Description', 'rich_text', 2, false, '{}'),

    ('fld_pmci_card', 'mch_pm_checklist_item', 'Card', 'reference', 0, true,  '{"target_machine":"mch_pm_card"}'),
    ('fld_pmci_text', 'mch_pm_checklist_item', 'Text', 'text',      1, true,  '{}'),
    ('fld_pmci_done', 'mch_pm_checklist_item', 'Done', 'boolean',   2, false, '{}')
ON CONFLICT (id) DO NOTHING;

INSERT INTO permissions (id, machine_id, role, events, can_read, can_create, can_edit, can_delete) VALUES
    ('perm_pmb_member',  'mch_pm_board',          'Member', ARRAY[]::TEXT[], true, true, true, false),
    ('perm_pml_member',  'mch_pm_list',           'Member', ARRAY[]::TEXT[], true, true, true, false),
    ('perm_pmc_member',  'mch_pm_card',           'Member', ARRAY[]::TEXT[], true, true, true, false),
    ('perm_pmci_member', 'mch_pm_checklist_item', 'Member', ARRAY[]::TEXT[], true, true, true, false)
ON CONFLICT (id) DO NOTHING;

INSERT INTO views (id, machine_id, name, type, position, config) VALUES
    ('vw_pmb_form',   'mch_pm_board', 'Board Form',   'form', 0, '{"fields":["fld_pmb_name","fld_pmb_owner"]}'),
    ('vw_pmb_list',   'mch_pm_board', 'Boards',       'list', 1, '{"columns":["fld_pmb_name","fld_pmb_owner"]}'),
    ('vw_pmb_detail', 'mch_pm_board', 'Board Detail', 'detail', 2, '{}'),

    ('vw_pml_form',   'mch_pm_list', 'List Form',   'form', 0, '{"fields":["fld_pml_board","fld_pml_name"]}'),
    ('vw_pml_list',   'mch_pm_list', 'Lists',       'list', 1, '{"columns":["fld_pml_board","fld_pml_name"]}'),
    ('vw_pml_detail', 'mch_pm_list', 'List Detail', 'detail', 2, '{}'),

    -- CAP-F16: Card's own form atomically authors up to 5 Checklist Items
    -- alongside the Card itself, the Journal-Entry-Lines precedent.
    ('vw_pmc_form',   'mch_pm_card', 'Card Form',   'form', 0,
     '{"fields":["fld_pmc_list","fld_pmc_title","fld_pmc_description"],"child_lines":{"machine":"mch_pm_checklist_item","parent_field":"fld_pmci_card","fields":["fld_pmci_text","fld_pmci_done"],"max_rows":5}}'),
    ('vw_pmc_list',   'mch_pm_card', 'Cards',       'list', 1, '{"columns":["fld_pmc_list","fld_pmc_title"]}'),
    ('vw_pmc_detail', 'mch_pm_card', 'Card Detail', 'detail', 2, '{}'),
    -- CAP-V14 Tier 3: group_field is a `reference` Field (fld_pmc_list),
    -- not a fixed value_list -- one lane per real List record (dynamic,
    -- user-creatable lanes, Case 19's own literal declaration), the real
    -- proof case for the capability roadmap.md Study 41 registered.
    ('vw_pmc_board',  'mch_pm_card', 'Card Board',  'board', 3, '{"columns":["fld_pmc_title"],"group_field":"fld_pmc_list"}'),

    ('vw_pmci_form',   'mch_pm_checklist_item', 'Checklist Item Form',   'form', 0, '{"fields":["fld_pmci_card","fld_pmci_text","fld_pmci_done"]}'),
    ('vw_pmci_detail', 'mch_pm_checklist_item', 'Checklist Item Detail', 'detail', 1, '{}')
ON CONFLICT (id) DO NOTHING;

INSERT INTO users (workspace_id, name, email, password_hash, workspace_role) VALUES
    ('ws_default', 'Project Member', 'project.member@example.com',
     '$2a$10$moxxOcZzSu3ILTzJlLF2Q.9vxiGNnSXPl7kY1pT3t5o1FoDjqC8aK', 'Member')
ON CONFLICT (email) DO NOTHING;

-- CAP-O11: workspace_memberships (migrations/026) is what Login's own
-- memberships.ForUser check actually reads -- seeds/040_membership_
-- backfill.sql's own precedent for every seed added after that migration.
INSERT INTO workspace_memberships (user_id, workspace_id, workspace_role)
SELECT id, 'ws_default', 'Member' FROM users WHERE email = 'project.member@example.com'
ON CONFLICT (user_id, workspace_id) DO NOTHING;

INSERT INTO user_application_roles (user_id, application_id, role)
SELECT u.id, 'app_project_management', 'Member' FROM users u WHERE u.email = 'project.member@example.com'
ON CONFLICT (user_id, application_id) DO UPDATE SET role = EXCLUDED.role;

-- RLS (migrations/009) requires app.workspace_id set before touching
-- `records`, same pattern seeds/031/032 already established.
SET app.workspace_id = 'ws_default';

INSERT INTO records (id, machine_id, workspace_id, data, created_at, updated_at) VALUES
    ('33333333-4444-5555-6666-000000000001', 'mch_pm_board', 'ws_default',
     '{"fld_pmb_name":"Website Redesign"}', NOW(), NOW())
ON CONFLICT (id) DO NOTHING;

INSERT INTO records (id, machine_id, workspace_id, data, created_at, updated_at) VALUES
    ('33333333-4444-5555-6666-000000000011', 'mch_pm_list', 'ws_default',
     '{"fld_pml_board":"33333333-4444-5555-6666-000000000001","fld_pml_name":"To Do"}', NOW(), NOW()),
    ('33333333-4444-5555-6666-000000000012', 'mch_pm_list', 'ws_default',
     '{"fld_pml_board":"33333333-4444-5555-6666-000000000001","fld_pml_name":"Doing"}', NOW(), NOW()),
    ('33333333-4444-5555-6666-000000000013', 'mch_pm_list', 'ws_default',
     '{"fld_pml_board":"33333333-4444-5555-6666-000000000001","fld_pml_name":"Done"}', NOW(), NOW())
ON CONFLICT (id) DO NOTHING;

INSERT INTO records (id, machine_id, workspace_id, data, created_at, updated_at) VALUES
    ('33333333-4444-5555-6666-000000000021', 'mch_pm_card', 'ws_default',
     '{"fld_pmc_list":"33333333-4444-5555-6666-000000000011","fld_pmc_title":"Wireframe homepage"}', NOW(), NOW()),
    ('33333333-4444-5555-6666-000000000022', 'mch_pm_card', 'ws_default',
     '{"fld_pmc_list":"33333333-4444-5555-6666-000000000011","fld_pmc_title":"Collect brand assets"}', NOW(), NOW()),
    ('33333333-4444-5555-6666-000000000023', 'mch_pm_card', 'ws_default',
     '{"fld_pmc_list":"33333333-4444-5555-6666-000000000012","fld_pmc_title":"Build landing page"}', NOW(), NOW()),
    ('33333333-4444-5555-6666-000000000024', 'mch_pm_card', 'ws_default',
     '{"fld_pmc_list":"33333333-4444-5555-6666-000000000013","fld_pmc_title":"Kickoff meeting notes"}', NOW(), NOW())
ON CONFLICT (id) DO NOTHING;

INSERT INTO records (id, machine_id, workspace_id, data, created_at, updated_at) VALUES
    ('33333333-4444-5555-6666-000000000031', 'mch_pm_checklist_item', 'ws_default',
     '{"fld_pmci_card":"33333333-4444-5555-6666-000000000021","fld_pmci_text":"Desktop layout","fld_pmci_done":"true"}', NOW(), NOW()),
    ('33333333-4444-5555-6666-000000000032', 'mch_pm_checklist_item', 'ws_default',
     '{"fld_pmci_card":"33333333-4444-5555-6666-000000000021","fld_pmci_text":"Mobile layout","fld_pmci_done":"false"}', NOW(), NOW())
ON CONFLICT (id) DO NOTHING;
