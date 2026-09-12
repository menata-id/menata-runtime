-- seeds/057_case19_card_fields.sql
-- case-03-case-19-completion-checklist.md's own Stage 3 (composable-
-- runtime-roadmap.md 17q): closes Case 19's real Field gaps against
-- project-board.html/project-card.html (ui-sample, checked directly, not
-- the checklist's own older paraphrase) -- Members and Labels are BOTH
-- real multi-value, Trello-shaped relationships ("Raka Aditya · Andi
-- Nur", "Frontend · Sprint 4"), not single-value fields. Composes
-- entirely from already-supported Grammar (reference/user/value_list/date
-- Fields, CAP-O02 master-data, CAP-V06 reverse-reference discovery) -- no
-- capability-lifecycle.md admission needed, see this increment's own
-- roadmap section for the A4 finding.

INSERT INTO fields (id, machine_id, name, type, position, required, options) VALUES
    ('fld_pmc_due_date', 'mch_pm_card', 'Due Date', 'date', 3, false, '{}')
ON CONFLICT (id) DO NOTHING;

-- mch_pm_label: a real, reusable, colored master-data catalog (CAP-O02) --
-- project-board.html's own real card markup shows the SAME label name
-- rendering in the SAME color on every card that carries it (Design/blue,
-- Frontend/emerald, High/rose, QA·Research·UX/violet, Planning/amber),
-- confirming a shared named+colored entity, not free text per card.
INSERT INTO machines (id, application_id, name, config) VALUES
    ('mch_pm_label', 'app_project_management', 'Label', '{"master_data":"true"}')
ON CONFLICT (id) DO NOTHING;

-- mch_pm_card_label / mch_pm_card_member: many-to-many join Machines --
-- same shape mch_pm_checklist_item already uses for Card's own child
-- rows (a `reference` field pointing back at the Card), discovered
-- automatically by both CAP-V06's childLists (Detail) and this
-- increment's own Board card-meta rendering (composable_board.go's
-- boardReverseParentField) -- neither needed a NEW discovery mechanism.
INSERT INTO machines (id, application_id, name) VALUES
    ('mch_pm_card_label',  'app_project_management', 'Card Label'),
    ('mch_pm_card_member', 'app_project_management', 'Card Member')
ON CONFLICT (id) DO NOTHING;

INSERT INTO fields (id, machine_id, name, type, position, required, options) VALUES
    ('fld_pmlb_name',  'mch_pm_label', 'Name',  'text',       0, true, '{}'),
    ('fld_pmlb_color', 'mch_pm_label', 'Color', 'value_list', 1, true,
     '{"values":["slate","blue","emerald","amber","rose","violet"]}'),

    ('fld_pmcl_card',  'mch_pm_card_label', 'Card',  'reference', 0, true, '{"target_machine":"mch_pm_card"}'),
    ('fld_pmcl_label', 'mch_pm_card_label', 'Label', 'reference', 1, true, '{"target_machine":"mch_pm_label"}'),

    ('fld_pmcm_card', 'mch_pm_card_member', 'Card',   'reference', 0, true, '{"target_machine":"mch_pm_card"}'),
    ('fld_pmcm_user', 'mch_pm_card_member', 'Member', 'user',      1, true, '{}')
ON CONFLICT (id) DO NOTHING;

INSERT INTO permissions (id, machine_id, role, events, can_read, can_create, can_edit, can_delete) VALUES
    ('perm_pmlb_member', 'mch_pm_label',       'Member', ARRAY[]::TEXT[], true, true, true, false),
    ('perm_pmcl_member', 'mch_pm_card_label',  'Member', ARRAY[]::TEXT[], true, true, true, false),
    ('perm_pmcm_member', 'mch_pm_card_member', 'Member', ARRAY[]::TEXT[], true, true, true, false)
ON CONFLICT (id) DO NOTHING;

INSERT INTO views (id, machine_id, name, type, position, config) VALUES
    ('vw_pmlb_form',   'mch_pm_label', 'Label Form',   'form', 0, '{"fields":["fld_pmlb_name","fld_pmlb_color"]}'),
    ('vw_pmlb_list',   'mch_pm_label', 'Labels',       'list', 1, '{"columns":["fld_pmlb_name","fld_pmlb_color"]}'),
    ('vw_pmlb_detail', 'mch_pm_label', 'Label Detail', 'detail', 2, '{}'),

    -- Form + Detail only, no List -- same shape mch_pm_checklist_item
    -- already uses (seeds/052): a join row's own List has nothing
    -- meaningful to browse beyond its two reference fields, already
    -- visible wherever CAP-V06 renders it as a reverse-reference section.
    ('vw_pmcl_form',   'mch_pm_card_label', 'Card Label Form',   'form', 0, '{"fields":["fld_pmcl_card","fld_pmcl_label"]}'),
    ('vw_pmcl_detail', 'mch_pm_card_label', 'Card Label Detail', 'detail', 1, '{}'),

    ('vw_pmcm_form',   'mch_pm_card_member', 'Card Member Form',   'form', 0, '{"fields":["fld_pmcm_card","fld_pmcm_user"]}'),
    ('vw_pmcm_detail', 'mch_pm_card_member', 'Card Member Detail', 'detail', 1, '{}')
ON CONFLICT (id) DO NOTHING;

-- 17q Part B: vw_pmc_form's own Checklist ChildLines block (seeds/052)
-- stays in the singular `child_lines` slot, untouched -- Members/Labels
-- are added as two NEW ChildLinesGroups entries (model.ViewConfig's own
-- plural sibling field), so a Card's own form now authors Checklist rows
-- AND Member rows AND Label rows atomically, same CREATE-time-only
-- pattern all three share.
UPDATE views
   SET config = jsonb_set(config, '{child_lines_groups}',
         '[{"machine":"mch_pm_card_member","parent_field":"fld_pmcm_card","fields":["fld_pmcm_user"],"max_rows":6},
           {"machine":"mch_pm_card_label","parent_field":"fld_pmcl_card","fields":["fld_pmcl_label"],"max_rows":6}]'::jsonb)
 WHERE id = 'vw_pmc_form'
   AND NOT (config ? 'child_lines_groups');

-- 17q Part D: vw_pmc_board's own opt-in CardMeta -- reverse-reference
-- Machines are named, their own back-reference field is discovered
-- automatically (composable_board.go's boardReverseParentField, the same
-- CAP-V06 convention), matching Board's own already-established "explicit
-- opt-in Config key, not automatic detection" posture (CAP-V17/CAP-V18).
UPDATE views
   SET config = jsonb_set(
         jsonb_set(config, '{columns}', '["fld_pmc_title"]'::jsonb),
         '{card_meta}',
         '{"labels_machine":"mch_pm_card_label","labels_ref_field":"fld_pmcl_label",
           "labels_name_field":"fld_pmlb_name","labels_color_field":"fld_pmlb_color",
           "members_machine":"mch_pm_card_member","members_user_field":"fld_pmcm_user",
           "progress_machine":"mch_pm_checklist_item","progress_done_field":"fld_pmci_done",
           "due_date_field":"fld_pmc_due_date"}'::jsonb
       )
 WHERE id = 'vw_pmc_board'
   AND NOT (config ? 'card_meta');

-- A second real Project Management user -- Case 19's own seed (052) has
-- exactly one ("Project Member"), too few to demonstrate Members as a
-- genuinely MULTI-person relationship end to end. Same hash as every
-- other seeded account (password "password").
INSERT INTO users (workspace_id, name, email, password_hash, workspace_role) VALUES
    ('ws_default', 'Project Owner', 'project.owner@example.com',
     '$2a$10$moxxOcZzSu3ILTzJlLF2Q.9vxiGNnSXPl7kY1pT3t5o1FoDjqC8aK', 'Member')
ON CONFLICT (email) DO NOTHING;

INSERT INTO workspace_memberships (user_id, workspace_id, workspace_role)
SELECT id, 'ws_default', 'Member' FROM users WHERE email = 'project.owner@example.com'
ON CONFLICT (user_id, workspace_id) DO NOTHING;

INSERT INTO user_application_roles (user_id, application_id, role)
SELECT u.id, 'app_project_management', 'Member' FROM users u WHERE u.email = 'project.owner@example.com'
ON CONFLICT (user_id, application_id) DO UPDATE SET role = EXCLUDED.role;

SET app.workspace_id = 'ws_default';

-- Real due dates on 3 of the 4 already-seeded Cards (seeds/052) -- the
-- 4th (Kickoff meeting notes) deliberately left blank, proving the "no
-- due date" case renders cleanly too, not just the happy path.
UPDATE records SET data = data || '{"fld_pmc_due_date":"2026-09-14"}'::jsonb
 WHERE id = '33333333-4444-5555-6666-000000000021' AND machine_id = 'mch_pm_card'
   AND NOT (data ? 'fld_pmc_due_date');
UPDATE records SET data = data || '{"fld_pmc_due_date":"2026-09-18"}'::jsonb
 WHERE id = '33333333-4444-5555-6666-000000000022' AND machine_id = 'mch_pm_card'
   AND NOT (data ? 'fld_pmc_due_date');
UPDATE records SET data = data || '{"fld_pmc_due_date":"2026-09-20"}'::jsonb
 WHERE id = '33333333-4444-5555-6666-000000000024' AND machine_id = 'mch_pm_card'
   AND NOT (data ? 'fld_pmc_due_date');

-- Real Labels, named and colored exactly as project-board.html's own real
-- card markup shows them.
INSERT INTO records (id, machine_id, workspace_id, data, created_at, updated_at) VALUES
    ('33333333-4444-5555-6666-000000000041', 'mch_pm_label', 'ws_default', '{"fld_pmlb_name":"Design",   "fld_pmlb_color":"blue"}',    NOW(), NOW()),
    ('33333333-4444-5555-6666-000000000042', 'mch_pm_label', 'ws_default', '{"fld_pmlb_name":"Frontend", "fld_pmlb_color":"emerald"}', NOW(), NOW()),
    ('33333333-4444-5555-6666-000000000043', 'mch_pm_label', 'ws_default', '{"fld_pmlb_name":"High",     "fld_pmlb_color":"rose"}',    NOW(), NOW()),
    ('33333333-4444-5555-6666-000000000044', 'mch_pm_label', 'ws_default', '{"fld_pmlb_name":"QA",       "fld_pmlb_color":"violet"}',  NOW(), NOW()),
    ('33333333-4444-5555-6666-000000000045', 'mch_pm_label', 'ws_default', '{"fld_pmlb_name":"Research", "fld_pmlb_color":"violet"}',  NOW(), NOW()),
    ('33333333-4444-5555-6666-000000000046', 'mch_pm_label', 'ws_default', '{"fld_pmlb_name":"UX",       "fld_pmlb_color":"violet"}',  NOW(), NOW()),
    ('33333333-4444-5555-6666-000000000047', 'mch_pm_label', 'ws_default', '{"fld_pmlb_name":"Planning", "fld_pmlb_color":"amber"}',   NOW(), NOW()),
    ('33333333-4444-5555-6666-000000000048', 'mch_pm_label', 'ws_default', '{"fld_pmlb_name":"Complete", "fld_pmlb_color":"emerald"}', NOW(), NOW())
ON CONFLICT (id) DO NOTHING;

-- Real Card<->Label joins -- Wireframe homepage carries two labels (Design
-- + High), matching project-board.html's own multi-label card shape.
INSERT INTO records (id, machine_id, workspace_id, data, created_at, updated_at) VALUES
    ('33333333-4444-5555-6666-000000000051', 'mch_pm_card_label', 'ws_default',
     '{"fld_pmcl_card":"33333333-4444-5555-6666-000000000021","fld_pmcl_label":"33333333-4444-5555-6666-000000000041"}', NOW(), NOW()),
    ('33333333-4444-5555-6666-000000000052', 'mch_pm_card_label', 'ws_default',
     '{"fld_pmcl_card":"33333333-4444-5555-6666-000000000021","fld_pmcl_label":"33333333-4444-5555-6666-000000000043"}', NOW(), NOW()),
    ('33333333-4444-5555-6666-000000000053', 'mch_pm_card_label', 'ws_default',
     '{"fld_pmcl_card":"33333333-4444-5555-6666-000000000022","fld_pmcl_label":"33333333-4444-5555-6666-000000000045"}', NOW(), NOW()),
    ('33333333-4444-5555-6666-000000000054', 'mch_pm_card_label', 'ws_default',
     '{"fld_pmcl_card":"33333333-4444-5555-6666-000000000023","fld_pmcl_label":"33333333-4444-5555-6666-000000000042"}', NOW(), NOW()),
    ('33333333-4444-5555-6666-000000000055', 'mch_pm_card_label', 'ws_default',
     '{"fld_pmcl_card":"33333333-4444-5555-6666-000000000024","fld_pmcl_label":"33333333-4444-5555-6666-000000000047"}', NOW(), NOW())
ON CONFLICT (id) DO NOTHING;

-- Real Card<->Member joins -- Wireframe homepage carries two members
-- (Project Member + Project Owner), matching project-card.html's own
-- "Raka Aditya · Andi Nur" multi-person shape. Kickoff meeting notes
-- deliberately gets none, proving the "no members" case renders cleanly.
INSERT INTO records (id, machine_id, workspace_id, data, created_at, updated_at)
SELECT '33333333-4444-5555-6666-000000000061', 'mch_pm_card_member', 'ws_default',
       jsonb_build_object('fld_pmcm_card', '33333333-4444-5555-6666-000000000021', 'fld_pmcm_user', id), NOW(), NOW()
  FROM users WHERE email = 'project.member@example.com'
ON CONFLICT (id) DO NOTHING;
INSERT INTO records (id, machine_id, workspace_id, data, created_at, updated_at)
SELECT '33333333-4444-5555-6666-000000000062', 'mch_pm_card_member', 'ws_default',
       jsonb_build_object('fld_pmcm_card', '33333333-4444-5555-6666-000000000021', 'fld_pmcm_user', id), NOW(), NOW()
  FROM users WHERE email = 'project.owner@example.com'
ON CONFLICT (id) DO NOTHING;
INSERT INTO records (id, machine_id, workspace_id, data, created_at, updated_at)
SELECT '33333333-4444-5555-6666-000000000063', 'mch_pm_card_member', 'ws_default',
       jsonb_build_object('fld_pmcm_card', '33333333-4444-5555-6666-000000000022', 'fld_pmcm_user', id), NOW(), NOW()
  FROM users WHERE email = 'project.member@example.com'
ON CONFLICT (id) DO NOTHING;
INSERT INTO records (id, machine_id, workspace_id, data, created_at, updated_at)
SELECT '33333333-4444-5555-6666-000000000064', 'mch_pm_card_member', 'ws_default',
       jsonb_build_object('fld_pmcm_card', '33333333-4444-5555-6666-000000000023', 'fld_pmcm_user', id), NOW(), NOW()
  FROM users WHERE email = 'project.owner@example.com'
ON CONFLICT (id) DO NOTHING;
