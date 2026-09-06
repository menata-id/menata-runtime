-- +goose Up
-- 006_permission_extensions.sql
-- CAP-P02 (record-level ownership) + CAP-P05 (CRUD-level permissions).
--
-- owner_field: when set, this permission's Events additionally require the
-- acting identity (not just the acting role) to equal the record's own
-- owner_field value -- e.g. only the specific Approver named on an Approval
-- Step, not anyone holding the "Approver" role, may decide it (WRP-1 Direct
-- Allocation). NULL = role-only, the behavior every existing permission row
-- already has.
--
-- can_read/can_create/can_edit: CRUD-level permission, independent of Events.
-- Default true so every role that already has a permission row on a machine
-- (i.e. every role actually named in that case's business narrative) keeps
-- working unchanged. The real behavior change is structural: a role with NO
-- permission row at all on a machine is now denied, not implicitly allowed.

ALTER TABLE permissions ADD COLUMN IF NOT EXISTS owner_field TEXT REFERENCES fields(id);
ALTER TABLE permissions ADD COLUMN IF NOT EXISTS can_read   BOOLEAN NOT NULL DEFAULT true;
ALTER TABLE permissions ADD COLUMN IF NOT EXISTS can_create BOOLEAN NOT NULL DEFAULT true;
ALTER TABLE permissions ADD COLUMN IF NOT EXISTS can_edit   BOOLEAN NOT NULL DEFAULT true;

-- +goose Down
ALTER TABLE permissions DROP COLUMN IF EXISTS can_edit;
ALTER TABLE permissions DROP COLUMN IF EXISTS can_create;
ALTER TABLE permissions DROP COLUMN IF EXISTS can_read;
ALTER TABLE permissions DROP COLUMN IF EXISTS owner_field;
