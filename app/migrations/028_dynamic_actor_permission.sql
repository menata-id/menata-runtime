-- +goose Up
-- 028_dynamic_actor_permission.sql
-- CAP-F24: per-record actor-type toggle. A Permission's existing
-- `owner_field` is a STATIC single-Field gate, always the same Field for
-- every record on the Machine. This adds three new, all-optional columns
-- so a Permission can instead declare a DYNAMIC gate: `actor_type_field`
-- names a value_list Field (e.g. "User"/"Group") whose value, per record,
-- selects whether `actor_user_field` or `actor_group_field` is this
-- record's own real approver gate -- resolved at Approve/Reject time, not
-- Machine-design time (capability-registry.md's CAP-F24 row).
--
-- Deliberately three flat nullable columns, not one JSONB blob, mirroring
-- `owner_field`'s own existing shape on this same table -- consistent with
-- this table's established convention, and lets `actor_user_field`/
-- `actor_group_field` carry real FK constraints to `fields(id)` the same
-- way `owner_field` already does (a dangling reference fails at the
-- database level, not just at metadata load time).
--
-- A Permission may declare BOTH `owner_field` and the three actor_*
-- columns at once -- internal/permission/guard.go's own CanTrigger treats
-- the dynamic gate as the PRIMARY check when `actor_type_field` resolves to
-- a real value on a given record, falling back to `owner_field` when it
-- doesn't (a record that predates this feature, or was never given a
-- value). This is what lets Approval Step's own `perm_as_approver` adopt
-- CAP-F24 without invalidating any already-seeded Step or already-written
-- conformance test that only ever set the original `fld_as_approver`.
ALTER TABLE permissions ADD COLUMN IF NOT EXISTS actor_type_field text REFERENCES fields(id);
ALTER TABLE permissions ADD COLUMN IF NOT EXISTS actor_user_field text REFERENCES fields(id);
ALTER TABLE permissions ADD COLUMN IF NOT EXISTS actor_group_field text REFERENCES fields(id);

-- +goose Down
ALTER TABLE permissions DROP COLUMN IF EXISTS actor_type_field;
ALTER TABLE permissions DROP COLUMN IF EXISTS actor_user_field;
ALTER TABLE permissions DROP COLUMN IF EXISTS actor_group_field;
