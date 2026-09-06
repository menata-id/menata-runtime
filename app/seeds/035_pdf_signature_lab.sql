-- seeds/035_pdf_signature_lab.sql
-- CAP-F22: binary PDF signature compositing (Study 32,
-- ../../benchmarks/024-pdf-signature-approval-study.md), extending Case 3
-- (Document Approval, seeds/004_approval.sql) rather than replacing it.
--
-- Scoped additions only. 004_approval.sql's own event_actions block has
-- already run on shared/long-lived databases and event_actions has no
-- natural key (see that file's own header comment and prototype/go/
-- CLAUDE.md's documented convention) -- this file adds new fields/rows and
-- exactly ONE new scoped event_actions INSERT + a position-renumbering
-- UPDATE, never re-runs 004's own blocks.
--
-- Safe to run multiple times, same as 004_approval.sql, EXCEPT its own
-- event_actions INSERT + UPDATE pair -- do not re-run this file against a
-- database that already has it applied.

-- New fields — Approval Step: where a placement was decided, before
-- submission (Study 32 §4.2). Percentage of page, not absolute points, so a
-- placement holds regardless of render resolution.
--
-- NOT required at the schema/constraint level, deliberately, even though
-- Study 32's own business narrative says a placement is decided "before
-- submission" -- Approval Step is a pre-existing, already-✅ Machine
-- (CAP-F13/A07/A08 since 2026-07-11) that older Case 3 tests
-- (conformance/tests/010_case1_3_core.sh) already create Steps against
-- without these fields. Making them constraint-required broke that
-- existing, ratcheted coverage the first time this was tried (T62 failed --
-- a Step created without a placement is rejected outright, corrupting the
-- older sequential-guard test's own setup). CAP-F22 itself enforces the
-- real precondition at the point it actually matters: doCompositeSignature
-- (internal/executor/executor.go) fails cleanly (T208) if an approval
-- reaches it with no usable placement, rather than blocking every Step
-- ever created. CAP-V21 (coordinate-placement editor, not built yet) will
-- eventually replace these three plain number inputs with a drag-to-place
-- UI over the same fields -- no schema change needed then either.
INSERT INTO fields (id, machine_id, name, type, position, required, options) VALUES
    ('fld_as_signature_page', 'mch_approval_step', 'Signature Page', 'number', 6, false, '{}'),
    ('fld_as_signature_x',    'mch_approval_step', 'Signature X %',  'number', 7, false, '{}'),
    ('fld_as_signature_y',    'mch_approval_step', 'Signature Y %',  'number', 8, false, '{}')
ON CONFLICT (id) DO NOTHING;

-- New field — Approval Document: the progressively-stamped output.
-- doCompositeSignature (internal/executor/executor.go) reads whichever of
-- this or fld_ad_file already has a value (this one once any approval has
-- run, fld_ad_file -- the original upload -- on the very first), always
-- writes here.
INSERT INTO fields (id, machine_id, name, type, position, required, options) VALUES
    ('fld_ad_signed_file', 'mch_approval_document', 'Signed File', 'file', 7, false, '{}')
ON CONFLICT (id) DO NOTHING;

-- New Machine — Signature: one row per user, an ordinary CAP-F13-style
-- lookup (Study 32 §4.3), no new mechanism. Workspace-authored, not
-- system-managed -- a user uploads their own signature image once, reused
-- across every Document they approve.
INSERT INTO machines (id, application_id, name, config) VALUES
    ('mch_signature', 'app_approval', 'Signature', NULL)
ON CONFLICT (id) DO NOTHING;

INSERT INTO fields (id, machine_id, name, type, position, required, options) VALUES
    ('fld_sig_owner', 'mch_signature', 'Owner', 'user', 0, true, '{}'),
    ('fld_sig_image', 'mch_signature', 'Image', 'file', 1, true, '{}')
ON CONFLICT (id) DO NOTHING;

INSERT INTO constraints (id, machine_id, rule, expression, condition, position) VALUES
    ('cst_sig_owner_required', 'mch_signature', 'Owner is required.',
     '{"field":"fld_sig_owner","operator":"required"}', NULL, 0),
    ('cst_sig_image_required', 'mch_signature', 'Image is required.',
     '{"field":"fld_sig_image","operator":"required"}', NULL, 1),
    ('cst_sig_unique_owner', 'mch_signature', 'This user already has a Signature on file.',
     '{"fields":["fld_sig_owner"],"operator":"unique"}', NULL, 2)
ON CONFLICT (id) DO NOTHING;

-- Any Approval-app identity manages their OWN Signature row only --
-- record-level scoping via owner_field, same pattern perm_as_approver
-- already uses in 004_approval.sql.
INSERT INTO permissions (id, machine_id, role, events, owner_field, can_create, can_read, can_edit) VALUES
    ('perm_sig_self_submitter', 'mch_signature', 'Submitter', ARRAY[]::TEXT[], 'fld_sig_owner', true, true, true),
    ('perm_sig_self_approver',  'mch_signature', 'Approver',  ARRAY[]::TEXT[], 'fld_sig_owner', true, true, true)
ON CONFLICT (id) DO NOTHING;

INSERT INTO views (id, machine_id, name, type, position, config) VALUES
    ('vw_sig_form',   'mch_signature', 'My Signature',     'form',   0, '{"fields":["fld_sig_owner","fld_sig_image"]}'),
    ('vw_sig_all',    'mch_signature', 'Signatures',       'list',   1, '{"columns":["fld_sig_owner"],"default_sort":{"field":"created_at","direction":"desc"}}'),
    ('vw_sig_detail', 'mch_signature', 'Signature Detail', 'detail', 2, '{}')
ON CONFLICT (id) DO NOTHING;

-- Scoped event-action addition — Approval Step's Approve. Position 4, BEFORE
-- aggregate_status's cross-record cascade (renumbered 4->5 below): a
-- compositing failure should abort the whole Approve (Persist returns
-- error, the request's transaction rolls back) before any parent-Document
-- rollup fires, not after.
INSERT INTO event_actions (event_id, type, position, params) VALUES
    ('evt_as_approve', 'composite_pdf_signature', 4,
     '{"document_field":"fld_as_document","source_file_field":"fld_ad_file","output_file_field":"fld_ad_signed_file","page_field":"fld_as_signature_page","x_field":"fld_as_signature_x","y_field":"fld_as_signature_y","signature_machine":"mch_signature","signature_owner_field":"fld_sig_owner","signature_image_field":"fld_sig_image"}');

UPDATE event_actions SET position = 5 WHERE event_id = 'evt_as_approve' AND type = 'aggregate_status';

-- vw_as_form's own content is edited in place (not just appended to) --
-- ON CONFLICT DO NOTHING won't pick this up on a database where the row
-- already exists from 004_approval.sql, so a companion UPDATE is required,
-- per CLAUDE.md's documented "editing an already-seeded row" gotcha.
UPDATE views SET config = '{"fields":["fld_as_document","fld_as_approver","fld_as_sequence","fld_as_signature_page","fld_as_signature_x","fld_as_signature_y"]}'
    WHERE id = 'vw_as_form';
