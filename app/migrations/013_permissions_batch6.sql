-- +goose Up
-- 013_permissions_batch6.sql
-- Batch 6 (Permissions): CAP-P03/P04/P06/P07.
--
-- CAP-P06 (field-level visibility): which Fields a role's Permission row
-- excludes from what it can see -- "Salary visible only to HR." Distinct
-- from can_read (whole-Machine); this is per-Field, per-role, same
-- Permission row CAP-P02's owner_field already extends.
ALTER TABLE permissions ADD COLUMN IF NOT EXISTS hidden_fields TEXT[] NOT NULL DEFAULT '{}';

-- CAP-P04 (delegation): an Event can declare it needs fresh input at
-- trigger time (not just the record's own existing data) -- "hand this off
-- to a specific peer," where "which peer" can only be known at the moment
-- of delegating, not baked into the record beforehand. Rendered as an
-- inline picker alongside the trigger button (internal/ui); an
-- event_actions `set_field` action reads it back via a new "input:<field>"
-- value prefix, parallel to the existing "field:<id>" (reads the record's
-- own data) and "current_user"/"today" tokens.
ALTER TABLE events ADD COLUMN IF NOT EXISTS input_fields TEXT[] NOT NULL DEFAULT '{}';

-- CAP-P03 (separation of duties) and CAP-P07 (public/unauthenticated read
-- access) need no schema change: P03 is a Machine.Config pair
-- (sod_reference_field/sod_requester_field, same pattern CAP-R07/R08
-- already established for Machine-level behavior that isn't a Field of any
-- record); P07 is a Permission row with role = 'Visitor', which the
-- existing permissions table already supports without a new column.

-- +goose Down
ALTER TABLE events DROP COLUMN IF EXISTS input_fields;
ALTER TABLE permissions DROP COLUMN IF EXISTS hidden_fields;
