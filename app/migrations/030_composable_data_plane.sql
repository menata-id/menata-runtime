-- +goose Up
-- 030_composable_data_plane.sql (composable-runtime-roadmap.md 17k, CR-21)
-- Data-plane composable artifacts -- Dataset, Query -- real, loadable
-- metadata, not just internal_composable's own View-inferred shapes.
-- Same pattern views/fields already established: typed columns + one
-- JSONB config blob, no RLS (metadata tables never have one -- only
-- records-adjacent tables do, see migrations/009/022's own exception).
--
-- config JSONB shapes (runtime-metadata-schema.md's own "Composable
-- Schema Extensions" section, PROPOSED sketch, now real here):
--   datasets.config -> {"relations":[{"id":"rel_x","via":"fld_x"}],
--                       "dimensions":[{"id":"dim_x","field":"fld_x"}],
--                       "measures":[{"id":"mea_x","aggregate":"sum","field":"fld_x"}]}
--   queries.config  -> {"projection":["dim_x","mea_x"],
--                       "filter":{"field":"fld_x","operator":"eq","value":"y"},
--                       "sort":{"field":"fld_x","direction":"desc"}}
CREATE TABLE IF NOT EXISTS datasets (
    id              TEXT PRIMARY KEY,
    application_id  TEXT NOT NULL REFERENCES applications(id) ON DELETE CASCADE,
    base_machine_id TEXT NOT NULL REFERENCES machines(id) ON DELETE CASCADE,
    name            TEXT NOT NULL,
    position        INT NOT NULL DEFAULT 0,
    config          JSONB NOT NULL DEFAULT '{}',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS queries (
    id         TEXT PRIMARY KEY,
    dataset_id TEXT NOT NULL REFERENCES datasets(id) ON DELETE CASCADE,
    name       TEXT NOT NULL,
    position   INT NOT NULL DEFAULT 0,
    config     JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- +goose Down
DROP TABLE IF EXISTS queries;
DROP TABLE IF EXISTS datasets;
