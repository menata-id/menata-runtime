-- seeds/004_approval.sql
-- Case 3 (Document Approval) — the case that originally motivated CAP-A07,
-- CAP-A08, CAP-X03 (P1/P3/P4 in Study 1's original gap numbering).
-- Realizes approval-document.yaml + approval-step.yaml, now that all three
-- [NOT YET] mechanisms they already specified are implemented.
-- Safe to run multiple times (ON CONFLICT DO NOTHING) except event_actions,
-- which has no natural key -- do not re-run this file's event_actions block
-- against a database that already has it applied.

-- Application
INSERT INTO applications (id, workspace_id, name) VALUES
    ('app_approval', 'ws_default', 'Approval')
ON CONFLICT (id) DO NOTHING;

-- Machines
-- config (CAP-X03): Approval Document names which field holds its mode, which
-- Machine holds its steps, and which field on the steps points back -- exactly
-- the block approval-document.yaml already sketched, commented out.
INSERT INTO machines (id, application_id, name, config) VALUES
    ('mch_approval_document', 'app_approval', 'Approval Document',
     '{"approval_mode_field":"fld_ad_approval_mode","steps_machine":"mch_approval_step","steps_parent_field":"fld_as_document"}'),
    ('mch_approval_step', 'app_approval', 'Approval Step', NULL)
ON CONFLICT (id) DO NOTHING;

-- Fields — Approval Document
INSERT INTO fields (id, machine_id, name, type, position, required, options) VALUES
    ('fld_ad_title',          'mch_approval_document', 'Title',          'text',       0, true,  '{}'),
    ('fld_ad_document_type',  'mch_approval_document', 'Document Type',  'value_list', 1, true,  '{"values":["SOP","Policy","Contract","Report","Other"]}'),
    ('fld_ad_file',           'mch_approval_document', 'File',           'file',       2, true,  '{}'),
    ('fld_ad_submitted_by',   'mch_approval_document', 'Submitted By',   'user',       3, true,  '{}'),
    ('fld_ad_approval_mode',  'mch_approval_document', 'Approval Mode',  'value_list', 4, true,  '{"values":["Sequential","Parallel"]}'),
    ('fld_ad_notes',          'mch_approval_document', 'Notes',          'rich_text',  5, false, '{}'),
    ('fld_ad_status',         'mch_approval_document', 'Status',         'value_list', 6, false, '{"values":["Draft","In Review","Approved","Rejected","Withdrawn"]}')
ON CONFLICT (id) DO NOTHING;

-- Fields — Approval Step
-- fld_as_document (CAP-F13): the reference the whole workflow orchestration
-- resolves the parent through.
INSERT INTO fields (id, machine_id, name, type, position, required, options) VALUES
    ('fld_as_document',    'mch_approval_step', 'Document',    'reference',  0, true,  '{"target_machine":"mch_approval_document"}'),
    ('fld_as_approver',    'mch_approval_step', 'Approver',    'user',       1, true,  '{}'),
    ('fld_as_sequence',    'mch_approval_step', 'Sequence',    'number',     2, true,  '{}'),
    ('fld_as_decision',    'mch_approval_step', 'Decision',    'value_list', 3, false, '{"values":["Pending","Approved","Rejected"]}'),
    ('fld_as_notes',       'mch_approval_step', 'Notes',       'rich_text',  4, false, '{}'),
    ('fld_as_decided_at',  'mch_approval_step', 'Decided At',  'date_time',  5, false, '{}')
ON CONFLICT (id) DO NOTHING;

-- Events — Approval Document
-- condition (CAP-E06): evt_ad_approve/reject only fire from In Review -- they
-- are never pressed by a user (see Permissions: role System), only reached via
-- CAP-A08's aggregate_status, but the guard still applies uniformly (triggerEvent
-- has one path for both HTTP and internal callers).
INSERT INTO events (id, machine_id, name, position, condition) VALUES
    ('evt_ad_submit',   'mch_approval_document', 'Submit',   0, '{"field":"fld_ad_status","operator":"equals","value":"Draft"}'),
    ('evt_ad_approve',  'mch_approval_document', 'Approve',  1, '{"field":"fld_ad_status","operator":"equals","value":"In Review"}'),
    ('evt_ad_reject',   'mch_approval_document', 'Reject',   2, '{"field":"fld_ad_status","operator":"equals","value":"In Review"}'),
    ('evt_ad_withdraw', 'mch_approval_document', 'Withdraw', 3, '{"field":"fld_ad_status","operator":"equals","value":"Draft"}')
ON CONFLICT (id) DO NOTHING;

-- Events — Approval Step
INSERT INTO events (id, machine_id, name, position, condition) VALUES
    ('evt_as_approve', 'mch_approval_step', 'Approve', 0, '{"field":"fld_as_decision","operator":"equals","value":"Pending"}'),
    ('evt_as_reject',  'mch_approval_step', 'Reject',  1, '{"field":"fld_as_decision","operator":"equals","value":"Pending"}')
ON CONFLICT (id) DO NOTHING;

-- Event Actions — Approval Document
INSERT INTO event_actions (event_id, type, position, params) VALUES
    ('evt_ad_submit',   'set_field', 0, '{"field":"fld_ad_status","value":"In Review"}'),
    ('evt_ad_submit',   'notify',    1, '{"role":"Approver"}'),
    ('evt_ad_approve',  'set_field', 0, '{"field":"fld_ad_status","value":"Approved"}'),
    -- CAP-A04: recipient_field resolves to THIS Document's own Submitted By
    -- value (e.g. "Alice"), not every user holding the Submitter role --
    -- role is the fallback if that field is ever empty.
    ('evt_ad_approve',  'notify',    1, '{"recipient_field":"fld_ad_submitted_by","role":"Submitter"}'),
    ('evt_ad_reject',   'set_field', 0, '{"field":"fld_ad_status","value":"Rejected"}'),
    ('evt_ad_reject',   'notify',    1, '{"recipient_field":"fld_ad_submitted_by","role":"Submitter"}'),
    ('evt_ad_withdraw', 'set_field', 0, '{"field":"fld_ad_status","value":"Withdrawn"}');

-- Event Actions — Approval Step
-- CAP-A07: activate_next only on Approve (rejecting doesn't hand off -- it
-- cascades the whole Document via aggregate_status instead).
-- CAP-A08: aggregate_status on both -- any Rejected cascades immediately,
-- Approved only rolls up once every sibling has left Pending.
INSERT INTO event_actions (event_id, type, position, params) VALUES
    ('evt_as_approve', 'set_field', 0, '{"field":"fld_as_decision","value":"Approved"}'),
    ('evt_as_approve', 'set_field', 1, '{"field":"fld_as_decided_at","value":"now"}'),
    ('evt_as_approve', 'notify',    2, '{"role":"Submitter"}'),
    ('evt_as_approve', 'activate_next', 3, '{"mode_field":"fld_ad_approval_mode"}'),
    ('evt_as_approve', 'aggregate_status', 4,
     '{"parent_field":"fld_as_document","parent_event_if_all_approved":"evt_ad_approve","parent_event_if_any_rejected":"evt_ad_reject"}'),
    ('evt_as_reject',  'set_field', 0, '{"field":"fld_as_decision","value":"Rejected"}'),
    ('evt_as_reject',  'set_field', 1, '{"field":"fld_as_decided_at","value":"now"}'),
    ('evt_as_reject',  'notify',    2, '{"role":"Submitter"}'),
    ('evt_as_reject',  'aggregate_status', 3,
     '{"parent_field":"fld_as_document","parent_event_if_all_approved":"evt_ad_approve","parent_event_if_any_rejected":"evt_ad_reject"}');

-- Constraints
INSERT INTO constraints (id, machine_id, rule, expression, condition, position) VALUES
    ('cst_ad_title_required', 'mch_approval_document', 'Title is required.',
     '{"field":"fld_ad_title","operator":"required"}', NULL, 0),
    ('cst_ad_file_required',  'mch_approval_document', 'File is required.',
     '{"field":"fld_ad_file","operator":"required"}', NULL, 1),
    ('cst_ad_mode_required',  'mch_approval_document', 'Approval Mode is required.',
     '{"field":"fld_ad_approval_mode","operator":"required"}', NULL, 2),
    ('cst_as_notes_if_rejected', 'mch_approval_step', 'Notes is required.',
     '{"field":"fld_as_notes","operator":"required"}',
     '{"field":"fld_as_decision","operator":"equals","value":"Rejected"}', 0),
    ('cst_as_document_required', 'mch_approval_step', 'Document is required.',
     '{"field":"fld_as_document","operator":"required"}', NULL, 1),
    ('cst_as_approver_required', 'mch_approval_step', 'Approver is required.',
     '{"field":"fld_as_approver","operator":"required"}', NULL, 2),
    ('cst_as_sequence_required', 'mch_approval_step', 'Sequence is required.',
     '{"field":"fld_as_sequence","operator":"required"}', NULL, 3)
ON CONFLICT (id) DO NOTHING;

-- Permissions
-- System (CAP-A08): evt_ad_approve/reject are never triggered via this row --
-- doAggregateStatus calls triggerEvent directly, bypassing the HTTP permission
-- gate entirely, the same way a cron-fired event would (CAP-E05 territory).
-- The row is kept for documentation fidelity with approval-document.yaml, not
-- because anything currently checks it.
INSERT INTO permissions (id, machine_id, role, events) VALUES
    ('perm_ad_submitter', 'mch_approval_document', 'Submitter', ARRAY['evt_ad_submit','evt_ad_withdraw']),
    ('perm_ad_system',    'mch_approval_document', 'System',    ARRAY['evt_ad_approve','evt_ad_reject']),
    ('perm_as_approver',  'mch_approval_step',     'Approver',  ARRAY['evt_as_approve','evt_as_reject'])
ON CONFLICT (id) DO NOTHING;

-- Views
INSERT INTO views (id, machine_id, name, type, position, config) VALUES
    ('vw_ad_form',
     'mch_approval_document', 'Submission Form', 'form', 0,
     '{"fields":["fld_ad_title","fld_ad_document_type","fld_ad_file","fld_ad_submitted_by","fld_ad_approval_mode","fld_ad_notes"]}'),
    ('vw_ad_all',
     'mch_approval_document', 'All Documents', 'list', 1,
     '{"columns":["fld_ad_title","fld_ad_document_type","fld_ad_approval_mode","fld_ad_status"],"default_sort":{"field":"created_at","direction":"desc"}}'),
    ('vw_ad_detail',
     'mch_approval_document', 'Approval Status', 'detail', 2, '{}'),
    ('vw_as_form',
     'mch_approval_step', 'Step Form', 'form', 0,
     '{"fields":["fld_as_document","fld_as_approver","fld_as_sequence"]}'),
    ('vw_as_progress',
     'mch_approval_step', 'Approval Progress', 'list', 1,
     '{"columns":["fld_as_document","fld_as_approver","fld_as_sequence","fld_as_decision"],"default_sort":{"field":"fld_as_sequence","direction":"asc"}}'),
    ('vw_as_detail',
     'mch_approval_step', 'Step Detail', 'detail', 2, '{}')
ON CONFLICT (id) DO NOTHING;
