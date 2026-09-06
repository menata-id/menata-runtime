-- seeds/033_follow_lab.sql
-- CAP-F20 proof: a many-to-many relationship (Follow) — neither side owns
-- the row, both sides are independently queryable, the (Follower, Followee)
-- pair must be unique. Case 11's own social-follow.yaml sketch (written
-- 2026-07-04, before CAP-F05 existed) originally called for both reference
-- fields to target a not-yet-built "$identity" flavor of CAP-F13 — CAP-F05
-- (`user` field, implemented 2026-07-12) already IS that real
-- reference-sugar over the `users` table, so this lab uses `type: user`
-- for both sides instead. Reading the actual code before writing this
-- (not guessing from the registry row's own one-line framing) found the
-- shape needs NO new mechanism at all: two `user` fields (CAP-F05) + a
-- composite `unique` Constraint (CAP-C12, RecordStore.ExistsWithFieldValues
-- and handler.uniquenessViolations are both completely field-type-agnostic
-- -- they read data[fieldID] as a plain string, never branch on the
-- field's declared Type) + two CAP-V05/V09 `$current_user`-filtered Views
-- (one per direction) are already sufficient, proven end-to-end here.

INSERT INTO workspaces (id, name, slug) VALUES ('ws_default', 'Default Workspace', 'ws_default')
ON CONFLICT (id) DO NOTHING;

INSERT INTO applications (id, workspace_id, name) VALUES
    ('app_follow_lab', 'ws_default', 'Follow Lab')
ON CONFLICT (id) DO NOTHING;

INSERT INTO machines (id, application_id, name) VALUES
    ('mch_f20_follow', 'app_follow_lab', 'Follow')
ON CONFLICT (id) DO NOTHING;

INSERT INTO fields (id, machine_id, name, type, position, required, options) VALUES
    ('fld_f20_follower',     'mch_f20_follow', 'Follower',     'user', 0, true, '{}'),
    ('fld_f20_followee',     'mch_f20_follow', 'Followee',     'user', 1, true, '{}'),
    ('fld_f20_followed_at',  'mch_f20_follow', 'Followed At',  'date', 2, false, '{}')
ON CONFLICT (id) DO NOTHING;

-- CAP-C12: the composite pair must be unique, but (Follower=A, Followee=B)
-- and (Follower=B, Followee=A) are DIFFERENT pairs -- following someone
-- doesn't collide with them following you back. ExistsWithFieldValues
-- compares both fields together, in the order given, so this is naturally
-- direction-sensitive -- proven, not just assumed, by T186/T187 below.
INSERT INTO constraints (id, machine_id, rule, expression, position) VALUES
    ('cst_f20_unique_pair', 'mch_f20_follow',
     'A Follower may follow a given Followee only once.',
     '{"operator":"unique","fields":["fld_f20_follower","fld_f20_followee"]}', 0)
ON CONFLICT (id) DO NOTHING;

INSERT INTO permissions (id, machine_id, role, events, can_read, can_create, can_edit, can_delete) VALUES
    ('perm_f20_member', 'mch_f20_follow', 'Member', ARRAY[]::TEXT[], true, true, false, false)
ON CONFLICT (id) DO NOTHING;

INSERT INTO views (id, machine_id, name, type, position, config) VALUES
    ('vw_f20_form', 'mch_f20_follow', 'New Follow', 'form', 0,
     '{"fields":["fld_f20_follower","fld_f20_followee","fld_f20_followed_at"]}'),
    ('vw_f20_list', 'mch_f20_follow', 'All Follows', 'list', 1,
     '{"columns":["fld_f20_follower","fld_f20_followee"]}'),
    -- CAP-V05/V09: "my" filter on Follower -- who I follow.
    ('vw_f20_following', 'mch_f20_follow', 'My Following', 'list', 2,
     '{"columns":["fld_f20_followee","fld_f20_followed_at"],"filter":[{"field":"fld_f20_follower","operator":"equals","value":"$current_user"}]}'),
    -- Same mechanism, the OTHER direction of the same join Machine -- who
    -- follows me. Distinct from V06's childLists (which only walks
    -- `reference`-typed fields, not `user`-typed ones) -- this is the real
    -- bidirectional-lookup path for a `user`-field-based join Machine.
    ('vw_f20_followers', 'mch_f20_follow', 'My Followers', 'list', 3,
     '{"columns":["fld_f20_follower","fld_f20_followed_at"],"filter":[{"field":"fld_f20_followee","operator":"equals","value":"$current_user"}]}')
ON CONFLICT (id) DO NOTHING;

INSERT INTO users (workspace_id, name, email, password_hash, workspace_role) VALUES
    ('ws_default', 'Follow Lab Amir', 'f20.amir@example.com',
     '$2a$10$moxxOcZzSu3ILTzJlLF2Q.9vxiGNnSXPl7kY1pT3t5o1FoDjqC8aK', 'Member'),
    ('ws_default', 'Follow Lab Budi', 'f20.budi@example.com',
     '$2a$10$moxxOcZzSu3ILTzJlLF2Q.9vxiGNnSXPl7kY1pT3t5o1FoDjqC8aK', 'Member')
ON CONFLICT (workspace_id, email) DO NOTHING;

INSERT INTO user_application_roles (user_id, application_id, role)
SELECT u.id, 'app_follow_lab', 'Member' FROM users u WHERE u.email IN ('f20.amir@example.com', 'f20.budi@example.com')
ON CONFLICT (user_id, application_id) DO UPDATE SET role = EXCLUDED.role;
