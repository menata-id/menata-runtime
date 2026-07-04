-- 001_metadata_schema.sql
-- Runtime Metadata tables: Workspace, Application, Machine, Field, Event,
-- Constraint, Permission, View.

CREATE TABLE IF NOT EXISTS workspaces (
    id         TEXT PRIMARY KEY,
    name       TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS applications (
    id           TEXT PRIMARY KEY,
    workspace_id TEXT NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    name         TEXT NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS machines (
    id             TEXT PRIMARY KEY,
    application_id TEXT NOT NULL REFERENCES applications(id) ON DELETE CASCADE,
    name           TEXT NOT NULL,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- options JSONB stores type-specific config:
--   value_list -> {"values": ["Draft","Submitted",...]}
--   reference  -> {"machine_id": "mch_some_machine"}
CREATE TABLE IF NOT EXISTS fields (
    id         TEXT PRIMARY KEY,
    machine_id TEXT NOT NULL REFERENCES machines(id) ON DELETE CASCADE,
    name       TEXT NOT NULL,
    type       TEXT NOT NULL,
    position   INT NOT NULL DEFAULT 0,
    required   BOOLEAN NOT NULL DEFAULT FALSE,
    options    JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS events (
    id         TEXT PRIMARY KEY,
    machine_id TEXT NOT NULL REFERENCES machines(id) ON DELETE CASCADE,
    name       TEXT NOT NULL,
    position   INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- type: set_field | notify | create_record
-- params JSONB:
--   set_field  -> {"field": "fld_status", "value": "Submitted"}
--   notify     -> {"role": "Designer"}
CREATE TABLE IF NOT EXISTS event_actions (
    id       BIGSERIAL PRIMARY KEY,
    event_id TEXT NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    type     TEXT NOT NULL,
    position INT NOT NULL DEFAULT 0,
    params   JSONB NOT NULL DEFAULT '{}'
);

-- expression: {"field": "fld_title", "operator": "required"}
-- condition (optional): {"field": "fld_design_type", "operator": "equals", "value": "Banner 2:1"}
CREATE TABLE IF NOT EXISTS constraints (
    id         TEXT PRIMARY KEY,
    machine_id TEXT NOT NULL REFERENCES machines(id) ON DELETE CASCADE,
    rule       TEXT NOT NULL,
    expression JSONB NOT NULL,
    condition  JSONB,
    position   INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- events TEXT[] stores event ids this role may trigger
CREATE TABLE IF NOT EXISTS permissions (
    id         TEXT PRIMARY KEY,
    machine_id TEXT NOT NULL REFERENCES machines(id) ON DELETE CASCADE,
    role       TEXT NOT NULL,
    events     TEXT[] NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(machine_id, role)
);

-- type: form | list | detail | dashboard | calendar | timeline
-- config JSONB:
--   list   -> {"columns": [...], "default_sort": {"field":"created_at","direction":"desc"}}
--   form   -> {"fields": [...]}
--   detail -> {}
CREATE TABLE IF NOT EXISTS views (
    id         TEXT PRIMARY KEY,
    machine_id TEXT NOT NULL REFERENCES machines(id) ON DELETE CASCADE,
    name       TEXT NOT NULL,
    type       TEXT NOT NULL,
    position   INT NOT NULL DEFAULT 0,
    config     JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
