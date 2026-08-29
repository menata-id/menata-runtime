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
--
-- pg_constraint.conname is NOT globally unique across schemas -- a naive
-- `WHERE conname = '...'` with no table qualification matches ANY schema's
-- same-named constraint, not just the one on the `groups` table this
-- migration's own search_path resolves to. Caught live: this migration
-- "passed" (DO, no error) against a fresh isolated test schema while
-- silently adding NO constraint at all, because an EARLIER run against a
-- different schema had already created a same-named constraint there, and
-- the unqualified check found that one instead. 'groups'::regclass
-- resolves through the CURRENT search_path, so conrelid correctly scopes
-- this to THIS schema's own groups table.
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'groups_workspace_name_unique'
          AND conrelid = 'groups'::regclass
    ) THEN
        ALTER TABLE groups ADD CONSTRAINT groups_workspace_name_unique UNIQUE (workspace_id, name);
    END IF;
END $$;
