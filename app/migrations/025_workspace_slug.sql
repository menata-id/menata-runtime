-- +goose Up
-- 025_workspace_slug.sql
-- CAP-X14: every workspace-scoped route gains a `/{slug}/` prefix. `slug` is
-- a new column, deliberately kept separate from the existing `id` (Study 35
-- §5.5, benchmarks/027-workspace-self-service-provisioning-study.md) so a
-- future slug rename never ripples through every `workspace_id` FK that
-- already points at `id`. Backfilled from the existing `id` verbatim for
-- the two seeded workspaces (`ws_default`, `ws_acme`) -- not prettified, so
-- no seed data or conformance expectation changes shape.
ALTER TABLE workspaces ADD COLUMN IF NOT EXISTS slug TEXT;
UPDATE workspaces SET slug = id WHERE slug IS NULL;
ALTER TABLE workspaces ALTER COLUMN slug SET NOT NULL;
-- +goose StatementBegin
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'workspaces_slug_key'
          AND conrelid = 'workspaces'::regclass
    ) THEN
        ALTER TABLE workspaces ADD CONSTRAINT workspaces_slug_key UNIQUE (slug);
    END IF;
END $$;
-- +goose StatementEnd

-- +goose Down
ALTER TABLE workspaces DROP CONSTRAINT IF EXISTS workspaces_slug_key;
ALTER TABLE workspaces DROP COLUMN IF EXISTS slug;
