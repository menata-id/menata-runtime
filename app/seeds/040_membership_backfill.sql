-- seeds/040_membership_backfill.sql
-- CAP-O11: run LAST (see Makefile's own seed target ordering), after every
-- other seed file has had its own chance to insert into `users`. Backfills
-- one workspace_memberships row per user from its own (vestigial, post-
-- CAP-O11 -- see migrations/026's header) workspace_id/workspace_role,
-- exactly mirroring that migration's own backfill statement -- this is the
-- fresh-schema-CI equivalent of it, since on a fresh schema migrations run
-- BEFORE any seed has inserted a single user, so that migration's own
-- backfill is a no-op there. Deliberately generic (every user, not scoped
-- to one seed file) rather than a companion INSERT duplicated 24 times
-- after every individual seed file's own users insert -- ON CONFLICT DO
-- NOTHING makes running it once, last, exactly equivalent and far simpler.
-- seeds/039_multi_workspace_identity_lab.sql's own two explicit membership
-- rows are unaffected (already present, this is a no-op for that identity).
INSERT INTO workspace_memberships (user_id, workspace_id, workspace_role)
SELECT id, workspace_id, workspace_role FROM users
WHERE workspace_id IS NOT NULL
ON CONFLICT (user_id, workspace_id) DO NOTHING;
