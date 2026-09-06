-- seeds/022_quorum_lab.sql
-- Process Overlay B4 proof, Part 2 (Study 21 addendum, CAP-W03): CAP-A08's
-- `aggregate_status` action generalized with an optional `min_approvals`
-- quorum parameter -- real N-of-M semantics (2 of 3 votes is enough,
-- doesn't wait for the 3rd; a minority of rejections that still leaves
-- enough headroom does not cancel early).
--
-- Deliberately hand-authored, same style as Case 3 (seeds/004_approval.sql)
-- -- NOT declared via `process`. This proves the mechanism itself works
-- independent of the Process Overlay; expressing quorum *declaratively*
-- through `process.requirements` is a separate, harder cross-machine
-- compile problem, named as explicit future work (see roadmap.md's B4
-- note), not attempted this pass.
--
-- A NEW, self-contained lab (new Application, new Machines) so Case 3's
-- own existing ALL-required tests (T22-T26) stay untouched -- min_approvals
-- omitted/zero falls through to that exact unchanged behavior.

INSERT INTO workspaces (id, name, slug) VALUES ('ws_default', 'Default Workspace', 'ws_default')
ON CONFLICT (id) DO NOTHING;

INSERT INTO applications (id, workspace_id, name) VALUES
    ('app_quorum_lab', 'ws_default', 'Quorum Lab')
ON CONFLICT (id) DO NOTHING;

INSERT INTO machines (id, application_id, name) VALUES
    ('mch_ql_request', 'app_quorum_lab', 'Quorum Request'),
    ('mch_ql_vote',    'app_quorum_lab', 'Quorum Vote')
ON CONFLICT (id) DO NOTHING;

INSERT INTO fields (id, machine_id, name, type, position, required, options) VALUES
    ('fld_qr_title',  'mch_ql_request', 'Title',  'text',       0, true,  '{}'),
    ('fld_qr_status', 'mch_ql_request', 'Status', 'value_list', 1, false,
     '{"values":["Open","Approved","Rejected"]}'),

    ('fld_qv_request',  'mch_ql_vote', 'Request',  'reference',  0, true,  '{"target_machine":"mch_ql_request"}'),
    ('fld_qv_voter',    'mch_ql_vote', 'Voter',    'user',       1, true,  '{}'),
    ('fld_qv_decision', 'mch_ql_vote', 'Decision', 'value_list', 2, false, '{"values":["Pending","Approved","Rejected"]}')
ON CONFLICT (id) DO NOTHING;

-- Request itself is never triggered by a role -- only System (via
-- aggregate_status) reaches evt_qr_approve/evt_qr_reject, same posture as
-- Case 3's own approval-document.yaml.
INSERT INTO events (id, machine_id, name, position, condition) VALUES
    ('evt_qr_approve', 'mch_ql_request', 'Approve', 0, '{"field":"fld_qr_status","operator":"equals","value":"Open"}'),
    ('evt_qr_reject',  'mch_ql_request', 'Reject',  1, '{"field":"fld_qr_status","operator":"equals","value":"Open"}'),

    ('evt_qv_approve', 'mch_ql_vote', 'Approve', 0, '{"field":"fld_qv_decision","operator":"equals","value":"Pending"}'),
    ('evt_qv_reject',  'mch_ql_vote', 'Reject',  1, '{"field":"fld_qv_decision","operator":"equals","value":"Pending"}')
ON CONFLICT (id) DO NOTHING;

INSERT INTO event_actions (event_id, type, position, params) VALUES
    ('evt_qr_approve', 'set_field', 0, '{"field":"fld_qr_status","value":"Approved"}'),
    ('evt_qr_reject',  'set_field', 0, '{"field":"fld_qr_status","value":"Rejected"}'),

    ('evt_qv_approve', 'set_field', 0, '{"field":"fld_qv_decision","value":"Approved"}'),
    ('evt_qv_approve', 'aggregate_status', 1,
     '{"parent_field":"fld_qv_request","parent_event_if_all_approved":"evt_qr_approve","parent_event_if_any_rejected":"evt_qr_reject","min_approvals":2}'),
    ('evt_qv_reject',  'set_field', 0, '{"field":"fld_qv_decision","value":"Rejected"}'),
    ('evt_qv_reject',  'aggregate_status', 1,
     '{"parent_field":"fld_qv_request","parent_event_if_all_approved":"evt_qr_approve","parent_event_if_any_rejected":"evt_qr_reject","min_approvals":2}');

-- Requester creates Requests (can_create/can_read default true) but only
-- READS Votes -- Voters are the ones who create/decide their own vote.
INSERT INTO permissions (id, machine_id, role, events, owner_field) VALUES
    ('perm_qr_system',    'mch_ql_request', 'System',    ARRAY['evt_qr_approve','evt_qr_reject'], NULL),
    ('perm_qr_requester', 'mch_ql_request', 'Requester', ARRAY[]::TEXT[], NULL),
    ('perm_qv_voter',     'mch_ql_vote',    'Voter',     ARRAY['evt_qv_approve','evt_qv_reject'], 'fld_qv_voter'),
    ('perm_qv_requester', 'mch_ql_vote',    'Requester', ARRAY[]::TEXT[], NULL)
ON CONFLICT (id) DO NOTHING;

UPDATE permissions SET can_create = false, can_edit = false WHERE id = 'perm_qv_requester';

INSERT INTO views (id, machine_id, name, type, position, config) VALUES
    ('vw_qr_form',   'mch_ql_request', 'New Request', 'form', 0, '{"fields":["fld_qr_title"]}'),
    ('vw_qr_list',   'mch_ql_request', 'Requests', 'list', 1, '{"columns":["fld_qr_title","fld_qr_status"]}'),
    ('vw_qr_detail', 'mch_ql_request', 'Request Detail', 'detail', 2, '{}'),
    ('vw_qv_form',   'mch_ql_vote', 'New Vote', 'form', 0, '{"fields":["fld_qv_request","fld_qv_voter"]}'),
    ('vw_qv_list',   'mch_ql_vote', 'Votes', 'list', 1, '{"columns":["fld_qv_request","fld_qv_voter","fld_qv_decision"]}'),
    ('vw_qv_detail', 'mch_ql_vote', 'Vote Detail', 'detail', 2, '{}')
ON CONFLICT (id) DO NOTHING;

INSERT INTO users (workspace_id, name, email, password_hash, workspace_role) VALUES
    ('ws_default', 'Rani Requester', 'quorum.requester@example.com',
     '$2a$10$moxxOcZzSu3ILTzJlLF2Q.9vxiGNnSXPl7kY1pT3t5o1FoDjqC8aK', 'Member'),
    ('ws_default', 'Voter One', 'quorum.voter1@example.com',
     '$2a$10$moxxOcZzSu3ILTzJlLF2Q.9vxiGNnSXPl7kY1pT3t5o1FoDjqC8aK', 'Member'),
    ('ws_default', 'Voter Two', 'quorum.voter2@example.com',
     '$2a$10$moxxOcZzSu3ILTzJlLF2Q.9vxiGNnSXPl7kY1pT3t5o1FoDjqC8aK', 'Member'),
    ('ws_default', 'Voter Three', 'quorum.voter3@example.com',
     '$2a$10$moxxOcZzSu3ILTzJlLF2Q.9vxiGNnSXPl7kY1pT3t5o1FoDjqC8aK', 'Member')
ON CONFLICT (workspace_id, email) DO NOTHING;

INSERT INTO user_application_roles (user_id, application_id, role)
SELECT u.id, 'app_quorum_lab',
       CASE u.email WHEN 'quorum.requester@example.com' THEN 'Requester' ELSE 'Voter' END
FROM users u
WHERE u.email IN ('quorum.requester@example.com', 'quorum.voter1@example.com',
                  'quorum.voter2@example.com', 'quorum.voter3@example.com')
ON CONFLICT (user_id, application_id) DO UPDATE SET role = EXCLUDED.role;
