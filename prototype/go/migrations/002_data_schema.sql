-- 002_data_schema.sql
-- Business Data tables: users, records, record_events.

CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE IF NOT EXISTS users (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id  TEXT REFERENCES workspaces(id),
    name          TEXT NOT NULL,
    email         TEXT NOT NULL,
    password_hash TEXT NOT NULL,
    role          TEXT NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(workspace_id, email)
);

-- records stores all business data produced by running applications.
-- data JSONB holds field values keyed by field id.
-- e.g. {"fld_title": "My Request", "fld_status": "Draft"}
CREATE TABLE IF NOT EXISTS records (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    machine_id TEXT NOT NULL REFERENCES machines(id),
    data       JSONB NOT NULL DEFAULT '{}',
    created_by UUID REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_records_machine_id ON records(machine_id);
CREATE INDEX IF NOT EXISTS idx_records_data ON records USING gin(data);

-- record_events is an append-only log of every event performed on a record.
-- snapshot captures the record.data state before the event was applied.
CREATE TABLE IF NOT EXISTS record_events (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    record_id    UUID NOT NULL REFERENCES records(id) ON DELETE CASCADE,
    event_id     TEXT NOT NULL,
    performed_by UUID REFERENCES users(id),
    performed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    snapshot     JSONB
);

CREATE INDEX IF NOT EXISTS idx_record_events_record_id ON record_events(record_id);
