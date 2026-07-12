-- 011_manual_ordering.sql
-- CAP-V14: adds a manual, free-ordering column to records. Existing rows get
-- an initial value matching their created_at ordering (oldest = lowest, same
-- order the runtime already showed them in) so a manual-order View's first
-- render isn't a visible reshuffle; new rows default to "now" (fractional
-- seconds), placing them at the end. Reordering (RecordStore.Move) swaps two
-- rows' sort_order values -- a plain DOUBLE PRECISION, not an integer
-- sequence, so no renumbering pass is ever needed.

ALTER TABLE records ADD COLUMN IF NOT EXISTS sort_order DOUBLE PRECISION;

-- Backfill per workspace, not one blind UPDATE. On a deployment where
-- CAP-X06's RLS cutover (migrations/009) already ran, records is under
-- FORCE ROW LEVEL SECURITY -- a query with no app.workspace_id set matches
-- zero rows (RLS fails closed, by design), which would silently no-op this
-- backfill and then fail the NOT NULL constraint below. set_config(...,
-- true) here is the SQL-level equivalent of SET LOCAL, scoped for the rest
-- of this transaction, same GUC cmd/server/main.go's request-scoped
-- transaction sets per request. Harmless on a fresh install where 009
-- hasn't run yet -- RLS isn't enforced either way, so each iteration just
-- finds zero remaining NULL rows after the first.
DO $$
DECLARE
    ws RECORD;
BEGIN
    FOR ws IN SELECT id FROM workspaces LOOP
        PERFORM set_config('app.workspace_id', ws.id, true);
        UPDATE records SET sort_order = extract(epoch from created_at) WHERE sort_order IS NULL;
    END LOOP;
END $$;

ALTER TABLE records ALTER COLUMN sort_order SET DEFAULT extract(epoch from clock_timestamp());
ALTER TABLE records ALTER COLUMN sort_order SET NOT NULL;

CREATE INDEX IF NOT EXISTS idx_records_sort_order ON records(machine_id, sort_order);
