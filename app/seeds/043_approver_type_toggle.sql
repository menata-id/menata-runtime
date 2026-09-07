-- seeds/043_approver_type_toggle.sql
-- CAP-F24: per-record approver-type toggle on Approval Step. Each Step now
-- picks, at submission (not Machine-design time), whether its own approver
-- gate is a specific person or an entire Group's membership --
-- document-submit.html's own "User/Group" toggle per approval-steps row.
--
-- fld_as_approver (the original, static single-approver field, seeds/
-- 004_approval.sql) is kept, unmodified in shape, but no longer required --
-- see the constraint swap below. This is deliberate backward compatibility,
-- not an oversight: every already-seeded Step, and every existing
-- conformance test that only ever sets fld_as_approver (010_case1_3_core.sh,
-- 020_notify_permissions_actionlab.sh, 120_pdf_signature.sh,
-- 140_decision_stepper.sh, 130_coord_placement.sh,
-- 150_group_approver_picker.sh), keeps working unchanged --
-- internal/permission/guard.go's own ResolveActorGate falls back to
-- fld_as_approver (via perm_as_approver's still-declared owner_field)
-- whenever fld_as_approver_type is unset on a given record. New Steps
-- created through the toggle set fld_as_approver_type instead, engaging
-- the new dynamic gate -- see capability-registry.md's own CAP-F24 row and
-- model.go's DynamicActorGate doc comment for the full reasoning.

INSERT INTO fields (id, machine_id, name, type, position, required, options) VALUES
    ('fld_as_approver_type',  'mch_approval_step', 'Approver Type',  'value_list', 6, false, '{"values":["User","Group"]}'),
    ('fld_as_approver_user',  'mch_approval_step', 'Approver User',  'user',       7, false, '{"restrict_to_group":"Document Approvers"}'),
    ('fld_as_approver_group', 'mch_approval_step', 'Approver Group', 'group',      8, false, '{}')
ON CONFLICT (id) DO NOTHING;

-- CAP-F23 carries forward onto the new field: the Approver User picker
-- stays narrowed to the same "Document Approvers" Group the legacy
-- fld_as_approver already used -- editing an already-seeded row needs a
-- companion UPDATE (see the file-level comment above), covers a database
-- that seeded fld_as_approver_user before this line existed.
UPDATE fields SET options = '{"restrict_to_group":"Document Approvers"}' WHERE id = 'fld_as_approver_user';

-- fld_as_approver stays a real Field (so old data/tests keep resolving
-- fine) but stops being unconditionally required -- a NEW Step using the
-- toggle never fills it in. Editing an already-seeded row's content needs
-- a companion UPDATE, not another INSERT (prototype/go/CLAUDE.md's own
-- "editing an already-seeded row" gotcha).
UPDATE fields SET required = false WHERE id = 'fld_as_approver';

-- The old unconditional "Approver is required" constraint is replaced by
-- two CONDITIONAL ones (CAP-C09's existing condition mechanism, same
-- pattern cst_as_notes_if_rejected already uses) -- each half of the
-- toggle is required only when that half was actually chosen. No
-- conformance test asserts the old unconditional constraint's own
-- rejection message, so this swap has no known negative-test impact.
DELETE FROM constraints WHERE id = 'cst_as_approver_required';

INSERT INTO constraints (id, machine_id, rule, expression, condition) VALUES
    ('cst_as_approver_user_required', 'mch_approval_step', 'Approver User is required when Approver Type is User.',
        '{"field":"fld_as_approver_user","operator":"required"}',
        '{"field":"fld_as_approver_type","operator":"equals","value":"User"}'),
    ('cst_as_approver_group_required', 'mch_approval_step', 'Approver Group is required when Approver Type is Group.',
        '{"field":"fld_as_approver_group","operator":"required"}',
        '{"field":"fld_as_approver_type","operator":"equals","value":"Group"}')
ON CONFLICT (id) DO NOTHING;

-- The actual gate: perm_as_approver keeps its existing owner_field
-- (fld_as_approver, the legacy fallback) and additionally declares the
-- three new dynamic-actor columns (migrations/028_dynamic_actor_permission.sql).
-- internal/permission/guard.go's ResolveActorGate tries the dynamic gate
-- FIRST, falling back to owner_field only when fld_as_approver_type is
-- unset on the record being checked.
UPDATE permissions SET
    actor_type_field = 'fld_as_approver_type',
    actor_user_field = 'fld_as_approver_user',
    actor_group_field = 'fld_as_approver_group'
    WHERE id = 'perm_as_approver';

-- Surface the toggle on the Step's own form IN PLACE of fld_as_approver --
-- not alongside it. Showing both on one form would render two independent
-- Approver pickers at once (found live, running conformance's own T217
-- against a fresh schema: fld_as_approver's picker and fld_as_approver_user's
-- own picker both narrow to "Document Approvers", so Bob legitimately
-- appears twice, once per field -- a real, confusing double-picker, not a
-- test-authoring quirk). fld_as_approver stays a real Field (still
-- resolvable by CanTrigger's legacy fallback, per model.go's own
-- DynamicActorGate doc comment) -- it simply isn't rendered on the form
-- that creates NEW Steps going forward. Every existing conformance test
-- that sets it does so via a raw POST (r.FormValue reads every Machine
-- Field regardless of the View's own `fields` list, record_crud.go's
-- Create), never through this rendered form -- unaffected either way.
UPDATE views SET config = jsonb_set(
    config, '{fields}',
    '["fld_as_document","fld_as_approver_type","fld_as_approver_user","fld_as_approver_group","fld_as_sequence","fld_as_signature_page","fld_as_signature_x","fld_as_signature_y"]'::jsonb
) WHERE id = 'vw_as_form';
