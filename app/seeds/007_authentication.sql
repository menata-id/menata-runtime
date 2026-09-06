-- 007_authentication.sql
-- CAP-X02/CAP-O01: one real account per business role introduced across
-- Cases 1-6's seeds, each with an explicit per-Application role assignment
-- (user_application_roles) -- role is no longer a single flat column, see
-- migrations/010_authentication.sql's header for why. Password for every
-- account below is "password" (the same bcrypt hash Case 1's Alice/Bob
-- already use) -- a seed convention, not a production credential.
--
-- Two workspace Admins are seeded, one per workspace, so /admin/users is
-- exercisable by a real seeded account in each: hr@example.com (ws_default),
-- staff@example.com (ws_acme).
--
-- Wendy (submitter2@example.com) is a second, distinct Submitter in
-- app_approval, alongside Alice -- needed by conformance/run.sh's CAP-A04
-- negative case (a `notify: {recipient_field: ..., role: Submitter}` action
-- must reach only the record's own submitter, not broadcast to everyone who
-- holds the Submitter role) -- indistinguishable from Alice without a real
-- second Submitter account to check against.

INSERT INTO users (workspace_id, name, email, password_hash, workspace_role) VALUES
    ('ws_default', 'Carol', 'carol@example.com',
     '$2a$10$moxxOcZzSu3ILTzJlLF2Q.9vxiGNnSXPl7kY1pT3t5o1FoDjqC8aK', 'Member'),
    ('ws_default', 'Dave', 'employee@example.com',
     '$2a$10$moxxOcZzSu3ILTzJlLF2Q.9vxiGNnSXPl7kY1pT3t5o1FoDjqC8aK', 'Member'),
    ('ws_default', 'Eve', 'manager@example.com',
     '$2a$10$moxxOcZzSu3ILTzJlLF2Q.9vxiGNnSXPl7kY1pT3t5o1FoDjqC8aK', 'Member'),
    ('ws_default', 'Frank', 'hr@example.com',
     '$2a$10$moxxOcZzSu3ILTzJlLF2Q.9vxiGNnSXPl7kY1pT3t5o1FoDjqC8aK', 'Admin'),
    ('ws_default', 'Grace', 'agent@example.com',
     '$2a$10$moxxOcZzSu3ILTzJlLF2Q.9vxiGNnSXPl7kY1pT3t5o1FoDjqC8aK', 'Member'),
    ('ws_default', 'Henry', 'supervisor@example.com',
     '$2a$10$moxxOcZzSu3ILTzJlLF2Q.9vxiGNnSXPl7kY1pT3t5o1FoDjqC8aK', 'Member'),
    ('ws_acme', 'Ivan', 'staff@example.com',
     '$2a$10$moxxOcZzSu3ILTzJlLF2Q.9vxiGNnSXPl7kY1pT3t5o1FoDjqC8aK', 'Admin'),
    ('ws_default', 'Wendy', 'submitter2@example.com',
     '$2a$10$moxxOcZzSu3ILTzJlLF2Q.9vxiGNnSXPl7kY1pT3t5o1FoDjqC8aK', 'Member')
ON CONFLICT (workspace_id, email) DO NOTHING;

-- Per-Application role assignments (CAP-O01) for every account across every
-- Case, including Case 1's Alice/Bob -- looked up by email rather than a
-- hardcoded user id, since gen_random_uuid() means the id isn't known ahead
-- of time. ON CONFLICT DO UPDATE (not DO NOTHING) so re-running this file
-- against a database that already has these rows still converges to the
-- assignments below, the same "safe to re-run" guarantee fields/
-- constraints/permissions/views already have -- unlike event_actions, this
-- has a real natural key (user_id, application_id) to conflict on.
INSERT INTO user_application_roles (user_id, application_id, role)
SELECT u.id, x.application_id, x.role
FROM (VALUES
    ('alice@example.com',      'app_design',           'Requester'),
    ('alice@example.com',      'app_approval',          'Submitter'),
    ('submitter2@example.com', 'app_approval',          'Submitter'),
    ('bob@example.com',        'app_design',           'Designer'),
    ('bob@example.com',        'app_approval',          'Approver'),
    ('carol@example.com',      'app_approval',          'Approver'),
    ('employee@example.com',   'app_hr',               'Employee'),
    ('manager@example.com',    'app_hr',               'Manager'),
    ('hr@example.com',         'app_hr',               'HR'),
    ('agent@example.com',      'app_customer_service', 'Agent'),
    ('supervisor@example.com', 'app_customer_service', 'Supervisor'),
    ('staff@example.com',      'app_ops',              'Staff')
) AS x(email, application_id, role)
JOIN users u ON u.email = x.email
ON CONFLICT (user_id, application_id) DO UPDATE SET role = EXCLUDED.role;
