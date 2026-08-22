-- CAP-C08: a Constraint may declare which OTHER record(s) it depends on
-- (an aggregate over child records, or a field on a referenced record),
-- checked by handler.crossRecordViolations -- constraint.Eval never touches
-- storage.
ALTER TABLE constraints ADD COLUMN IF NOT EXISTS cross_record JSONB;
