-- +goose Up
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
--
-- CURRENT_USER (whichever role actually runs this migration), not a
-- hardcoded role name: prototype/go's own migration named its own
-- menata_runtime_app role literally, which only ever worked here by
-- accident (this dev host's shared Postgres cluster happens to already
-- have that role from prototype/go's own setup) -- a fresh database
-- anywhere else (this app's own CI, a real deployment under
-- menata_app_owner, DEVELOPMENT.md's own "Database role" section) has no
-- such role, and REVOKE/GRANT against a role that doesn't exist is a hard
-- error, not a silent no-op. REVOKE against a superuser (e.g. CI's plain
-- `postgres`) is itself a harmless no-op -- superusers bypass grants
-- entirely -- so this degrades safely there too.
REVOKE UPDATE, DELETE, TRUNCATE ON record_events FROM CURRENT_USER;

-- +goose Down
GRANT UPDATE, DELETE, TRUNCATE ON record_events TO CURRENT_USER;
ALTER TABLE record_events DROP COLUMN IF EXISTS correlation_id;
-- Best-effort only: the up migration deliberately widened performed_by from
-- UUID to TEXT so it could hold non-UUID identity/role strings (this
-- codebase's real identity model, see this file's own header) -- rolling
-- back the column type only succeeds if every current value still happens
-- to parse as a UUID. On a database that has ever recorded a non-UUID
-- performed_by (expected in real use), this cast fails; that failure is the
-- correct outcome, not a bug in this down migration -- there is no lossless
-- reverse for data this shape change already accepted.
ALTER TABLE record_events ALTER COLUMN performed_by TYPE UUID USING performed_by::uuid;
ALTER TABLE record_events ADD CONSTRAINT record_events_performed_by_fkey
    FOREIGN KEY (performed_by) REFERENCES users(id);
