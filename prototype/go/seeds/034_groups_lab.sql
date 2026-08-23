-- seeds/034_groups_lab.sql
-- CAP-O07: Groups/Teams as an intermediate role-assignment grouping --
-- proof that a role granted to a Group (not a direct user_application_roles
-- row) composes with a person's own direct assignment (union semantics),
-- takes effect without a restart, and revokes just as live when membership
-- changes. Group data itself (groups/group_application_roles) is
-- deliberately NOT seeded here -- the whole point is proving the
-- /admin/groups UI + GroupStore end to end, the same "prove the mechanism,
-- not just the data" discipline seeds/019_overlay_lab.sql already
-- established for the Process Overlay compiler.
--
-- Wati holds "Editor" DIRECTLY (can create/edit, but her Permission row
-- grants no Events) -- proves the direct half is unaffected by this
-- change. "Approver" (grants the Approve event) is only ever reached
-- through Group membership in the test itself -- proves the union: Wati
-- ends up able to both edit (direct) and approve (Group) at once, neither
-- role alone would be enough.

INSERT INTO workspaces (id, name) VALUES ('ws_default', 'Default Workspace')
ON CONFLICT (id) DO NOTHING;

INSERT INTO applications (id, workspace_id, name) VALUES
    ('app_groups_lab', 'ws_default', 'Groups Lab')
ON CONFLICT (id) DO NOTHING;

INSERT INTO machines (id, application_id, name) VALUES
    ('mch_gl_ticket', 'app_groups_lab', 'Ticket')
ON CONFLICT (id) DO NOTHING;

INSERT INTO fields (id, machine_id, name, type, position, required, options) VALUES
    ('fld_gl_title',  'mch_gl_ticket', 'Title',  'text',       0, true,  '{}'),
    ('fld_gl_status', 'mch_gl_ticket', 'Status', 'value_list', 1, false, '{"values":["Draft","Approved"]}')
ON CONFLICT (id) DO NOTHING;

INSERT INTO events (id, machine_id, name, position, condition) VALUES
    ('evt_gl_approve', 'mch_gl_ticket', 'Approve', 0, '{"field":"fld_gl_status","operator":"equals","value":"Draft"}')
ON CONFLICT (id) DO NOTHING;

-- notify role="Approver" (CAP-A03) -- T203 proves this reaches Wati's inbox
-- through her Group membership alone (recipientMatch's new OR clause), not
-- a direct user_application_roles row -- she never gets one for "Approver".
INSERT INTO event_actions (event_id, type, position, params) VALUES
    ('evt_gl_approve', 'set_field', 0, '{"field":"fld_gl_status","value":"Approved"}'),
    ('evt_gl_approve', 'notify',    1, '{"role":"Approver"}')
ON CONFLICT DO NOTHING;

INSERT INTO permissions (id, machine_id, role, events, can_read, can_create, can_edit, can_delete) VALUES
    ('perm_gl_editor',   'mch_gl_ticket', 'Editor',   ARRAY[]::TEXT[],              true, true,  true,  false),
    ('perm_gl_approver', 'mch_gl_ticket', 'Approver', ARRAY['evt_gl_approve'], true, false, false, false)
ON CONFLICT (id) DO NOTHING;

INSERT INTO views (id, machine_id, name, type, position, config) VALUES
    ('vw_gl_form', 'mch_gl_ticket', 'New Ticket', 'form', 0, '{"fields":["fld_gl_title"]}'),
    ('vw_gl_list', 'mch_gl_ticket', 'All Tickets', 'list', 1, '{"columns":["fld_gl_title","fld_gl_status"]}')
ON CONFLICT (id) DO NOTHING;

INSERT INTO users (workspace_id, name, email, password_hash, workspace_role) VALUES
    ('ws_default', 'Groups Lab Wati', 'wati.gl@example.com',
     '$2a$10$moxxOcZzSu3ILTzJlLF2Q.9vxiGNnSXPl7kY1pT3t5o1FoDjqC8aK', 'Member'),
    ('ws_default', 'Groups Lab Yuda', 'yuda.gl@example.com',
     '$2a$10$moxxOcZzSu3ILTzJlLF2Q.9vxiGNnSXPl7kY1pT3t5o1FoDjqC8aK', 'Member')
ON CONFLICT (workspace_id, email) DO NOTHING;

-- Wati's DIRECT assignment -- "Editor" only. "Approver" is deliberately
-- absent here; the test grants it exclusively via Group membership.
INSERT INTO user_application_roles (user_id, application_id, role)
SELECT u.id, 'app_groups_lab', 'Editor' FROM users u WHERE u.email = 'wati.gl@example.com'
ON CONFLICT (user_id, application_id) DO UPDATE SET role = EXCLUDED.role;
