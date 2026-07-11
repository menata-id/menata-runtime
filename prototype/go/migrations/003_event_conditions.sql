-- 003_event_conditions.sql
-- CAP-E06: state-conditional event availability.
--
-- An Event may declare a guard condition (same shape as a Constraint's
-- condition): the event may only be triggered when the record's CURRENT
-- data satisfies it. Realizes the `if` condition Menata Language already
-- allows on Events (specification/003-event.md §Conditions) -- until now
-- the runtime only stored `When <Event>` and ignored any guard on it,
-- which is why an Approved record could still be Rejected (Study 1's
-- headline CAP-E06 finding).
--
-- condition: {"field": "fld_status", "operator": "equals", "value": "Submitted"}
-- NULL = no guard, always allowed (same convention as constraints.condition).

ALTER TABLE events ADD COLUMN IF NOT EXISTS condition JSONB;
