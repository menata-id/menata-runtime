-- 009_workspace_isolation_rls.sql
-- CAP-X06 (part 2/2 -- the enforcement flip).
--
-- *** APPLY ONLY AT FINAL CUTOVER, IN THE SAME DEPLOY WINDOW AS RESTARTING
-- WITH THE BINARY THAT SETS app.workspace_id PER REQUEST. ***
--
-- Enabling RLS here makes every query against these tables return zero rows
-- for any connection that hasn't set app.workspace_id (current_setting with
-- missing_ok=true resolves to NULL, and NULL = anything is never true) --
-- fail closed, by design. If the currently-running server binary doesn't
-- yet set that GUC (see cmd/server/main.go's request-scoped transaction
-- middleware, internal/store/txctx.go), running this migration ahead of
-- deploying that binary breaks the live app immediately. Migration
-- 008_workspace_isolation_schema.sql (additive, already applied) is safe to
-- run any time; this one is not.

-- FORCE is required in addition to ENABLE, or the owning role
-- (menata_runtime_app -- see DEVELOPMENT.md's "Database role" section)
-- bypasses RLS by default, since table owners are exempt unless forced.
ALTER TABLE records ENABLE ROW LEVEL SECURITY;
ALTER TABLE records FORCE ROW LEVEL SECURITY;
ALTER TABLE record_events ENABLE ROW LEVEL SECURITY;
ALTER TABLE record_events FORCE ROW LEVEL SECURITY;
ALTER TABLE notifications ENABLE ROW LEVEL SECURITY;
ALTER TABLE notifications FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS ws_isolation ON records;
CREATE POLICY ws_isolation ON records
    USING (workspace_id = current_setting('app.workspace_id', true));

DROP POLICY IF EXISTS ws_isolation ON record_events;
CREATE POLICY ws_isolation ON record_events
    USING (workspace_id = current_setting('app.workspace_id', true));

DROP POLICY IF EXISTS ws_isolation ON notifications;
CREATE POLICY ws_isolation ON notifications
    USING (workspace_id = current_setting('app.workspace_id', true));
