-- 018_field_types.sql
-- Remaining field types: CAP-F18 (auto-numbering) is the only one needing a
-- schema change. CAP-F06 (file) stores uploaded bytes on local disk
-- (prototype/go/uploads/), keyed by an unguessable UUID -- no new table.
-- CAP-F07/F08/F09/F10/F14/F15/F17/F19/F21 are pure rendering/validation/
-- composition changes, no schema at all.

-- CAP-F18: a per-(machine, field) monotonic counter, incremented atomically
-- via INSERT ... ON CONFLICT DO UPDATE ... RETURNING (never a
-- SELECT-then-UPDATE, which races two concurrent Creates against each
-- other the same way CAP-X13's webhook claim would if it weren't atomic).
CREATE TABLE IF NOT EXISTS field_sequences (
    machine_id  TEXT NOT NULL REFERENCES machines(id) ON DELETE CASCADE,
    field_id    TEXT NOT NULL,
    next_value  BIGINT NOT NULL DEFAULT 1,
    PRIMARY KEY (machine_id, field_id)
);
