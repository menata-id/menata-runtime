-- seeds/039_multi_workspace_identity_lab.sql
-- CAP-O11: a real person with genuinely two Workspace memberships (one in
-- ws_default, one in ws_acme), to conformance-prove the /choose-workspace
-- picker end to end -- every other seeded account only ever has one
-- membership, so without this fixture the picker path would be entirely
-- untested. workspace_id/workspace_role on the users row itself are
-- vestigial post-CAP-O11 (see migrations/026's own header) -- 'ws_default'/
-- 'Member' here is an arbitrary satisfying value, not meaningful; the two
-- real memberships below are what actually matter.
INSERT INTO users (workspace_id, name, email, password_hash, workspace_role) VALUES
    ('ws_default', 'Yusuf', 'multiworkspace@example.com',
     '$2a$10$moxxOcZzSu3ILTzJlLF2Q.9vxiGNnSXPl7kY1pT3t5o1FoDjqC8aK', 'Member')
ON CONFLICT (email) DO NOTHING;

INSERT INTO workspace_memberships (user_id, workspace_id, workspace_role)
SELECT id, 'ws_default', 'Member' FROM users WHERE email = 'multiworkspace@example.com'
ON CONFLICT (user_id, workspace_id) DO NOTHING;

INSERT INTO workspace_memberships (user_id, workspace_id, workspace_role)
SELECT id, 'ws_acme', 'Member' FROM users WHERE email = 'multiworkspace@example.com'
ON CONFLICT (user_id, workspace_id) DO NOTHING;
