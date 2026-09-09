-- seeds/047_approval_flow_template.sql
-- CAP-V28: category-keyed saved approval-flow template. A form View's own
-- child_lines section (Approval Document's embedded Approval Step rows,
-- seeds/044) pre-fills from a matching record of this small companion
-- Machine the instant Document Type is picked on Create -- e.g. Document
-- Type = Contract pre-fills the saved Contract approval flow. Still fully
-- editable per record afterward, never enforced. Depends on CAP-F24
-- (seeds/043) already existing -- the template's own Step rows carry the
-- same approver_type/approver_user/approver_group shape a real Approval
-- Step does, per this capability's own registry row.
--
-- Two Machines, the same parent/child shape mch_approval_document/
-- mch_approval_step already use one layer up: "Approval Flow Template"
-- (one per Document Type) and "Approval Flow Template Step" (its saved
-- rows, authored via the SAME child_lines mechanism CAP-F16 already is,
-- not a new authoring UI).

INSERT INTO machines (id, application_id, name, config) VALUES
    ('mch_approval_flow_template',      'app_approval', 'Approval Flow Template', NULL),
    ('mch_approval_flow_template_step', 'app_approval', 'Approval Flow Template Step', NULL)
ON CONFLICT (id) DO NOTHING;

-- Fields — Approval Flow Template
-- fld_aft_document_type shares fld_ad_document_type's own option set
-- verbatim (seeds/004) -- CAP-V28's own MatchField is only meaningful
-- against the same value space the trigger field offers.
INSERT INTO fields (id, machine_id, name, type, position, required, options) VALUES
    ('fld_aft_name',          'mch_approval_flow_template', 'Name',          'text',       0, true, '{}'),
    ('fld_aft_document_type', 'mch_approval_flow_template', 'Document Type', 'value_list', 1, true, '{"values":["SOP","Policy","Contract","Report","Other"]}')
ON CONFLICT (id) DO NOTHING;

-- Fields — Approval Flow Template Step
-- Mirrors Approval Step's own CAP-F24 toggle shape (seeds/043) exactly --
-- fld_afts_approver_type/_user/_group -- so a saved row maps cleanly onto
-- a real Step's own fields via the handler's ChildFieldMap. fld_afts_template
-- is this row's own reference back at the parent template (ChildParentField).
INSERT INTO fields (id, machine_id, name, type, position, required, options) VALUES
    ('fld_afts_template',       'mch_approval_flow_template_step', 'Template',       'reference',  0, true,  '{"target_machine":"mch_approval_flow_template"}'),
    ('fld_afts_approver_type',  'mch_approval_flow_template_step', 'Approver Type',  'value_list', 1, false, '{"values":["User","Group"]}'),
    ('fld_afts_approver_user',  'mch_approval_flow_template_step', 'Approver User',  'user',       2, false, '{"restrict_to_group":"Document Approvers"}'),
    ('fld_afts_approver_group', 'mch_approval_flow_template_step', 'Approver Group', 'group',      3, false, '{}'),
    ('fld_afts_sequence',       'mch_approval_flow_template_step', 'Sequence',       'number',     4, true,  '{}')
ON CONFLICT (id) DO NOTHING;

-- Constraints — same conditional-required shape CAP-F24 itself uses
-- (seeds/043) for the equivalent real-Step fields.
INSERT INTO constraints (id, machine_id, rule, expression, condition) VALUES
    ('cst_afts_template_required', 'mch_approval_flow_template_step', 'Template is required.',
        '{"field":"fld_afts_template","operator":"required"}', NULL),
    ('cst_afts_sequence_required', 'mch_approval_flow_template_step', 'Sequence is required.',
        '{"field":"fld_afts_sequence","operator":"required"}', NULL),
    ('cst_afts_approver_user_required', 'mch_approval_flow_template_step', 'Approver User is required when Approver Type is User.',
        '{"field":"fld_afts_approver_user","operator":"required"}',
        '{"field":"fld_afts_approver_type","operator":"equals","value":"User"}'),
    ('cst_afts_approver_group_required', 'mch_approval_flow_template_step', 'Approver Group is required when Approver Type is Group.',
        '{"field":"fld_afts_approver_group","operator":"required"}',
        '{"field":"fld_afts_approver_type","operator":"equals","value":"Group"}'),
    ('cst_aft_name_required', 'mch_approval_flow_template', 'Name is required.',
        '{"field":"fld_aft_name","operator":"required"}', NULL),
    ('cst_aft_document_type_required', 'mch_approval_flow_template', 'Document Type is required.',
        '{"field":"fld_aft_document_type","operator":"required"}', NULL)
ON CONFLICT (id) DO NOTHING;

-- Permissions — reuses the existing Submitter role (already the role
-- authoring a Document's own approval chain via perm_ad_submitter_steps,
-- seeds/004) rather than introducing a new role nothing else assigns yet.
INSERT INTO permissions (id, machine_id, role, events, can_create, can_read, can_edit) VALUES
    ('perm_aft_submitter',  'mch_approval_flow_template',      'Submitter', ARRAY[]::TEXT[], true, true, true),
    ('perm_afts_submitter', 'mch_approval_flow_template_step', 'Submitter', ARRAY[]::TEXT[], true, true, true)
ON CONFLICT (id) DO NOTHING;

-- Views
INSERT INTO views (id, machine_id, name, type, position, config) VALUES
    ('vw_aft_form',
     'mch_approval_flow_template', 'Template Form', 'form', 0,
     '{"fields":["fld_aft_name","fld_aft_document_type"],"child_lines":{"machine":"mch_approval_flow_template_step","parent_field":"fld_afts_template","fields":["fld_afts_approver_type","fld_afts_approver_user","fld_afts_approver_group","fld_afts_sequence"],"max_rows":5}}'),
    ('vw_aft_all',
     'mch_approval_flow_template', 'All Templates', 'list', 1,
     '{"columns":["fld_aft_name","fld_aft_document_type"],"default_sort":{"field":"created_at","direction":"desc"}}'),
    ('vw_aft_detail',
     'mch_approval_flow_template', 'Template Detail', 'detail', 2, '{}'),
    ('vw_afts_form',
     'mch_approval_flow_template_step', 'Template Step Form', 'form', 0,
     '{"fields":["fld_afts_template","fld_afts_approver_type","fld_afts_approver_user","fld_afts_approver_group","fld_afts_sequence"]}'),
    ('vw_afts_detail',
     'mch_approval_flow_template_step', 'Template Step Detail', 'detail', 1, '{}')
ON CONFLICT (id) DO NOTHING;

-- The actual wiring: Approval Document's own Submission Form gains
-- child_lines_template, naming Document Type as TriggerField and mapping
-- this file's new Machines/Fields as the saved-template source. ChildFieldMap
-- keys are Approval Document's own vw_ad_form.child_lines.fields (seeds/043);
-- values are this file's own parallel fields.
UPDATE views SET config = config || '{"child_lines_template":{
    "trigger_field":"fld_ad_document_type",
    "template_machine":"mch_approval_flow_template",
    "match_field":"fld_aft_document_type",
    "child_machine":"mch_approval_flow_template_step",
    "child_parent_field":"fld_afts_template",
    "child_sequence_field":"fld_afts_sequence",
    "child_field_map":{
        "fld_as_approver_type":"fld_afts_approver_type",
        "fld_as_approver_user":"fld_afts_approver_user",
        "fld_as_approver_group":"fld_afts_approver_group",
        "fld_as_sequence":"fld_afts_sequence"
    }
}}'::jsonb WHERE id = 'vw_ad_form';
