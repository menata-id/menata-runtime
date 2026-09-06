-- +goose Up
-- 008_workspace_isolation_schema.sql
-- CAP-X06 (part 1/2 -- schema only, safe to run immediately).
--
-- Adds and backfills workspace_id on records/record_events/notifications --
-- purely additive, does not change any existing query's behavior (nothing
-- reads this column yet). Deliberately split from the RLS-enabling half
-- (migrations/009_workspace_isolation_rls.sql): enabling RLS before the
-- application-layer code that sets app.workspace_id per request exists
-- would make every query against these tables return zero rows for the
-- live server immediately -- current_setting('app.workspace_id', true)
-- resolves to NULL for any connection that never set it, and RLS fails
-- closed. Migration 009 is applied only at final cutover, in the same
-- deploy window as restarting with the binary that sets the GUC.
--
-- See docs/decisions/003-tenancy-and-indexing.md (ADR-003) and
-- benchmarks/004-scale-architecture-study.md (Study 8) for the design this
-- implements, and the explicit boundary on what's deliberately deferred
-- (PARTITION BY HASH, lazy per-workspace loading, RLS on metadata tables).

ALTER TABLE records ADD COLUMN IF NOT EXISTS workspace_id TEXT REFERENCES workspaces(id);
ALTER TABLE record_events ADD COLUMN IF NOT EXISTS workspace_id TEXT REFERENCES workspaces(id);
ALTER TABLE notifications ADD COLUMN IF NOT EXISTS workspace_id TEXT REFERENCES workspaces(id);

-- Backfill from the existing machine_id -> application_id -> workspace_id
-- chain (workspace_id is denormalized onto these tables for RLS/indexing,
-- not a new source of truth -- the chain above still defines it).
UPDATE records r SET workspace_id = a.workspace_id
    FROM machines m JOIN applications a ON a.id = m.application_id
    WHERE m.id = r.machine_id AND r.workspace_id IS NULL;

-- record_events is append-only by design (migrations/007 REVOKEs UPDATE
-- from menata_runtime_app) -- this one-time backfill of a genuinely new
-- column on pre-existing rows needs it back, then re-revokes immediately.
-- The table owner can GRANT/REVOKE its own privileges (ownership includes
-- managing grants), so this doesn't need a separate migration role.
GRANT UPDATE ON record_events TO menata_runtime_app;
UPDATE record_events e SET workspace_id = r.workspace_id
    FROM records r
    WHERE r.id = e.record_id AND e.workspace_id IS NULL;
REVOKE UPDATE ON record_events FROM menata_runtime_app;

UPDATE notifications n SET workspace_id = a.workspace_id
    FROM machines m JOIN applications a ON a.id = m.application_id
    WHERE m.id = n.machine_id AND n.workspace_id IS NULL AND n.machine_id IS NOT NULL;

-- records/record_events always have a resolvable machine -> workspace chain
-- (machine_id/record_id are NOT NULL FKs already) -- workspace_id can be
-- required. notifications.machine_id is itself nullable (a notify action
-- doesn't strictly need one), so workspace_id stays nullable there too.
ALTER TABLE records ALTER COLUMN workspace_id SET NOT NULL;
ALTER TABLE record_events ALTER COLUMN workspace_id SET NOT NULL;

CREATE INDEX IF NOT EXISTS idx_records_workspace_id ON records (workspace_id);
CREATE INDEX IF NOT EXISTS idx_record_events_workspace_id ON record_events (workspace_id);
CREATE INDEX IF NOT EXISTS idx_notifications_workspace_id ON notifications (workspace_id);

-- +goose Down
-- Dropping each column also drops its own index (idx_records_workspace_id
-- etc.) and the backfilled data with it -- there is no separate DROP INDEX
-- needed.
ALTER TABLE notifications DROP COLUMN IF EXISTS workspace_id;
ALTER TABLE record_events DROP COLUMN IF EXISTS workspace_id;
ALTER TABLE records DROP COLUMN IF EXISTS workspace_id;
