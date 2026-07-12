-- 016_workspace_services.sql
-- Batch 9 (Workspace services): CAP-O02 (master data designation), CAP-O04
-- (workspace-wide search -- no schema change, reuses CAP-V08's own
-- per-machine search logic across every readable Machine), CAP-O05
-- (unified notification center), CAP-O06 (business calendar).

-- CAP-O06: a Workspace's own declared non-working days, consumed by
-- CAP-A11's date arithmetic ("today + N Business Days" skips weekends and
-- these dates) -- loaded once at boot into the Interpreter, the same
-- "in-memory index, no DB access at request time" posture every other
-- piece of Runtime Metadata already gets.
CREATE TABLE IF NOT EXISTS workspace_holidays (
    workspace_id TEXT NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    holiday_date DATE NOT NULL,
    name         TEXT,
    PRIMARY KEY (workspace_id, holiday_date)
);

-- CAP-O05: a per-user notification preference -- "immediate" (CAP-A10's
-- existing flat inbox, unchanged default) or "digest" (the SAME inbox,
-- grouped by day instead of listed flat). No new delivery channel (no
-- email infrastructure exists in this prototype) -- a real, working
-- extension of the one channel that already exists, not a stub for one
-- that doesn't.
ALTER TABLE users ADD COLUMN IF NOT EXISTS notification_preference TEXT NOT NULL DEFAULT 'immediate';

-- CAP-O02: master data designation lives on machines.config (CAP-X03's
-- existing generic settings, no new column) -- "master_data": "true" flags
-- a Machine as canonical/cross-app-referenced. The one new behavior this
-- unlocks (protected deactivation -- CAP-R03's Archive is blocked while
-- another record, on ANY Machine, still references this one) needs no
-- schema change either, just a query at archive time.
