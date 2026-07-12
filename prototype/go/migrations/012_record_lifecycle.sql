-- 012_record_lifecycle.sql
-- CAP-R03 (delete/archive): soft delete, not a hard DELETE -- consistent
-- with CAP-R04's append-only record_events audit trail (a hard-deleted
-- record would leave audit rows pointing at nothing). can_delete defaults
-- to false, unlike can_read/can_create/can_edit's historical `true`
-- default (migrations/006) -- deletion is materially more dangerous and
-- deserves an explicit opt-in per Permission row, not a blanket grant,
-- matching CAP-P05's deny-by-default posture better than the other three
-- ever did.

ALTER TABLE permissions ADD COLUMN IF NOT EXISTS can_delete BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE records ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ;
CREATE INDEX IF NOT EXISTS idx_records_deleted_at ON records(machine_id, deleted_at);
