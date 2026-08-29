-- 024_group_name_unique.sql
-- CAP-F23: a group-restricted user-field picker needs to look Groups up by
-- NAME (metadata can't reference a Group's DB-generated UUID -- see
-- capability-registry.md's CAP-F23 row for why), which only works reliably
-- if a name is unique per workspace. migrations/023_groups.sql never added
-- this constraint -- closed here as a byproduct of needing the lookup, not
-- scope creep. Verified against the live dev database first (0 groups
-- existed at the time this migration was written, so no backfill/dedupe
-- step was needed).
-- Postgres has no ADD CONSTRAINT IF NOT EXISTS (unlike ADD COLUMN) -- guard
-- manually so this migration stays safe to re-run, same convention every
-- other migration in this repo already follows for its own statements.
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'groups_workspace_name_unique'
    ) THEN
        ALTER TABLE groups ADD CONSTRAINT groups_workspace_name_unique UNIQUE (workspace_id, name);
    END IF;
END $$;
