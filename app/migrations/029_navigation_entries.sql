-- +goose Up
-- 029_navigation_entries.sql
-- CAP-O03 Tier 5 (declared navigation, Phase 1): an Application MAY declare
-- real, ordered, nestable navigation metadata -- when it does, this table's
-- rows entirely replace CAP-O03's own inferred sub-nav/app-launcher listing
-- for that Application (internal/handler/handler.go's subNavFor/
-- AppMachines). An Application with zero rows here keeps today's exact
-- inferred behavior unchanged -- fully additive, no existing seed needs to
-- change. See benchmarks/009-in-app-navigation-benchmark.md's own
-- implementation-plan follow-on finding for the full design.
--
-- Phase 1 scope only: target_type 'machine' | 'view' | 'group'.
-- target_url exists as a column now so Phase 2 (external links) needs no
-- further migration, only a loader change -- deliberately NOT validated or
-- rendered yet (internal/metadata/validate.go rejects a 'url' entry
-- explicitly, per "Unknown = explicit").
--
-- Flat columns, not a JSONB config -- nothing here varies by target_type
-- enough to need one, matching this table's own closest sibling (fields/
-- permissions' flat-column shape) rather than views' config-blob shape.
CREATE TABLE navigation_entries (
    id              text PRIMARY KEY,
    application_id  text NOT NULL REFERENCES applications(id),
    parent_id       text REFERENCES navigation_entries(id),
    position        int NOT NULL DEFAULT 0,
    label           text NOT NULL,
    target_type     text NOT NULL CHECK (target_type IN ('machine', 'view', 'url', 'group')),
    target_machine  text REFERENCES machines(id),
    target_view     text REFERENCES views(id),
    target_url      text
);

CREATE INDEX idx_navigation_entries_application_id ON navigation_entries(application_id);
CREATE INDEX idx_navigation_entries_parent_id ON navigation_entries(parent_id);

-- +goose Down
DROP TABLE IF EXISTS navigation_entries;
