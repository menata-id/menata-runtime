-- seeds/038_group_approver_lab.sql
-- CAP-F23: group-restricted user-field picker. Approval Step's own
-- fld_as_approver field is already seeded (seeds/004_approval.sql) --
-- editing its existing content needs a companion UPDATE, not another
-- INSERT ... ON CONFLICT DO NOTHING (that would silently no-op on any
-- database that already ran 004_approval.sql, per this session's own
-- "editing an already-seeded row" gotcha). No Group named "Document
-- Approvers" is seeded here -- Groups are only ever created at runtime via
-- /admin/groups (CAP-O07's own established precedent, seeds/034_groups_
-- lab.sql seeds none either); conformance creates it live through the
-- real admin UI, same as CAP-O07's own T195 already does.
UPDATE fields SET options = '{"restrict_to_group":"Document Approvers"}'
    WHERE id = 'fld_as_approver';
