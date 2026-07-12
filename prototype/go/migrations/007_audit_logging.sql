-- 007_audit_logging.sql
-- CAP-I04 (correlation trace) + CAP-R04 fix (actor attribution + DB-level
-- immutability), per nfr-standards.md's STRIDE table:
--   Repudiation: "record_events must always carry actor + correlation_id"
--   Tampering:   "append-only event log" / "append-only at the DB level
--                (no UPDATE/DELETE grants)"
--
-- performed_by was `UUID REFERENCES users(id)` but never actually populated
-- -- this prototype's real identity model (CAP-P02's menata_identity /
-- menata_role session cookies) was never backed by the `users` table; the
-- Login handler never queries it. Store the identity/role string directly,
-- the model actually in use, instead of a dead FK to an unused table.
ALTER TABLE record_events DROP CONSTRAINT IF EXISTS record_events_performed_by_fkey;
ALTER TABLE record_events ALTER COLUMN performed_by TYPE TEXT USING performed_by::text;

-- One correlation_id per HTTP request (chi's middleware.RequestID, threaded
-- through ctx -- see cmd/server/main.go), shared by every record_events row
-- a single request produces even when it cascades across records
-- (CAP-A08 aggregate_status, CAP-E05 trigger_event) -- lets "why did
-- Document X become Approved" be answered by one id, not guessed from
-- timestamps.
ALTER TABLE record_events ADD COLUMN IF NOT EXISTS correlation_id TEXT;

-- Append-only enforced at the DB level, not just application discipline --
-- a Tampering countermeasure, not tidiness. INSERT/SELECT unaffected.
-- CAP-R03 (record delete) isn't implemented yet and shouldn't ever write to
-- this table directly if it is -- deletes cascade via records' own FK
-- (record_events_record_id_fkey ON DELETE CASCADE), which Postgres performs
-- under the constraint's own privileges, not the calling role's DML grants,
-- so this REVOKE doesn't block that cascade.
REVOKE UPDATE, DELETE, TRUNCATE ON record_events FROM menata_runtime_app;
