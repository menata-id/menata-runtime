-- seeds/026_quorum_declarative_lab.sql
-- CAP-W03 declarative-quorum proof: the exact same N-of-M shape as
-- seeds/022_quorum_lab.sql (2 of 3 votes), but the PARENT machine
-- (mch_ql2_request) declares it via `process.requirements[].type: approval`
-- instead of anything being hand-typed. The loader
-- (internal/metadata/loader.go's compileApprovalRequirements) injects the
-- `aggregate_status` action onto the CHILD machine's (mch_ql2_vote) own
-- Approve/Reject events -- notice there is no `aggregate_status` action
-- anywhere in this file, unlike 022's hand-authored event_actions.
--
-- The child stays hand-authored on purpose (a value_list field literally
-- named "Decision", two Events that set_field it) -- the declarative form
-- doesn't require the target machine to be process-overlay-declared too,
-- only the parent that's expressing the requirement.
--
-- A NEW, self-contained lab (new Application, new Machines) so 022's own
-- existing T149/T150 stay untouched -- T159/T160 below assert the exact
-- same two behaviors against this declaratively-wired pair, proving
-- "compiled equals hand-authored".

INSERT INTO workspaces (id, name, slug) VALUES ('ws_default', 'Default Workspace', 'ws_default')
ON CONFLICT (id) DO NOTHING;

INSERT INTO applications (id, workspace_id, name) VALUES
    ('app_quorum_decl_lab', 'ws_default', 'Quorum Declarative Lab')
ON CONFLICT (id) DO NOTHING;

-- Parent: process-overlay-declared. Submit carries the declarative approval
-- requirement; Approve/Reject are System-only outcomes the injected
-- aggregate_status action fires -- enforced at compile time (a human-role
-- actor here is a load-time error, not just a convention).
INSERT INTO machines (id, application_id, name, process) VALUES
    ('mch_ql2_request', 'app_quorum_decl_lab', 'Quorum Request (Declarative)', '{
      "states": ["Open", "PendingApproval", "Approved", "Rejected"],
      "transitions": [
        {"name": "Submit", "from": "Open", "to": "PendingApproval", "actor": {"role": "Requester"},
         "requirements": [
           {"type": "approval", "target": "mch_ql2_vote", "min_approvals": 2,
            "on_quorum_approved": "Approve", "on_quorum_rejected": "Reject"}
         ]},
        {"name": "Approve", "from": "PendingApproval", "to": "Approved", "actor": {"role": "System"}},
        {"name": "Reject",  "from": "PendingApproval", "to": "Rejected", "actor": {"role": "System"}}
      ]
    }'::jsonb)
ON CONFLICT (id) DO UPDATE SET process = EXCLUDED.process;

INSERT INTO fields (id, machine_id, name, type, position, required, options) VALUES
    ('fld_qr2_title', 'mch_ql2_request', 'Title', 'text', 0, true, '{}')
ON CONFLICT (id) DO NOTHING;

-- Child: hand-authored, identical shape to mch_ql_vote -- no aggregate_status
-- action written here; the loader injects it onto evt_ql2v_approve/reject.
INSERT INTO machines (id, application_id, name) VALUES
    ('mch_ql2_vote', 'app_quorum_decl_lab', 'Quorum Vote (Declarative)')
ON CONFLICT (id) DO NOTHING;

INSERT INTO fields (id, machine_id, name, type, position, required, options) VALUES
    ('fld_ql2v_request',  'mch_ql2_vote', 'Request',  'reference',  0, true,  '{"target_machine":"mch_ql2_request"}'),
    ('fld_ql2v_voter',    'mch_ql2_vote', 'Voter',    'user',       1, true,  '{}'),
    ('fld_ql2v_decision', 'mch_ql2_vote', 'Decision', 'value_list', 2, false, '{"values":["Pending","Approved","Rejected"]}')
ON CONFLICT (id) DO NOTHING;

INSERT INTO events (id, machine_id, name, position, condition) VALUES
    ('evt_ql2v_approve', 'mch_ql2_vote', 'Approve', 0, '{"field":"fld_ql2v_decision","operator":"equals","value":"Pending"}'),
    ('evt_ql2v_reject',  'mch_ql2_vote', 'Reject',  1, '{"field":"fld_ql2v_decision","operator":"equals","value":"Pending"}')
ON CONFLICT (id) DO NOTHING;

INSERT INTO event_actions (event_id, type, position, params) VALUES
    ('evt_ql2v_approve', 'set_field', 0, '{"field":"fld_ql2v_decision","value":"Approved"}'),
    ('evt_ql2v_reject',  'set_field', 0, '{"field":"fld_ql2v_decision","value":"Rejected"}');

INSERT INTO permissions (id, machine_id, role, events, owner_field) VALUES
    ('perm_qr2_requester', 'mch_ql2_request', 'Requester', ARRAY[]::TEXT[], NULL),
    ('perm_qv2_voter',     'mch_ql2_vote',    'Voter',     ARRAY['evt_ql2v_approve','evt_ql2v_reject'], 'fld_ql2v_voter'),
    ('perm_qv2_requester', 'mch_ql2_vote',    'Requester', ARRAY[]::TEXT[], NULL)
ON CONFLICT (id) DO NOTHING;

UPDATE permissions SET can_create = false, can_edit = false WHERE id = 'perm_qv2_requester';

INSERT INTO views (id, machine_id, name, type, position, config) VALUES
    ('vw_qr2_form',   'mch_ql2_request', 'New Request', 'form', 0, '{"fields":["fld_qr2_title"]}'),
    ('vw_qr2_list',   'mch_ql2_request', 'Requests', 'list', 1, '{"columns":["fld_qr2_title","fld_mch_ql2_request_status"]}'),
    ('vw_qr2_detail', 'mch_ql2_request', 'Request Detail', 'detail', 2, '{}'),
    ('vw_qv2_form',   'mch_ql2_vote', 'New Vote', 'form', 0, '{"fields":["fld_ql2v_request","fld_ql2v_voter"]}'),
    ('vw_qv2_list',   'mch_ql2_vote', 'Votes', 'list', 1, '{"columns":["fld_ql2v_request","fld_ql2v_voter","fld_ql2v_decision"]}'),
    ('vw_qv2_detail', 'mch_ql2_vote', 'Vote Detail', 'detail', 2, '{}')
ON CONFLICT (id) DO NOTHING;

INSERT INTO users (workspace_id, name, email, password_hash, workspace_role) VALUES
    ('ws_default', 'Rani Requester Two', 'quorumdecl.requester@example.com',
     '$2a$10$moxxOcZzSu3ILTzJlLF2Q.9vxiGNnSXPl7kY1pT3t5o1FoDjqC8aK', 'Member'),
    ('ws_default', 'Voter Decl One', 'quorumdecl.voter1@example.com',
     '$2a$10$moxxOcZzSu3ILTzJlLF2Q.9vxiGNnSXPl7kY1pT3t5o1FoDjqC8aK', 'Member'),
    ('ws_default', 'Voter Decl Two', 'quorumdecl.voter2@example.com',
     '$2a$10$moxxOcZzSu3ILTzJlLF2Q.9vxiGNnSXPl7kY1pT3t5o1FoDjqC8aK', 'Member'),
    ('ws_default', 'Voter Decl Three', 'quorumdecl.voter3@example.com',
     '$2a$10$moxxOcZzSu3ILTzJlLF2Q.9vxiGNnSXPl7kY1pT3t5o1FoDjqC8aK', 'Member')
ON CONFLICT (workspace_id, email) DO NOTHING;

INSERT INTO user_application_roles (user_id, application_id, role)
SELECT u.id, 'app_quorum_decl_lab',
       CASE u.email WHEN 'quorumdecl.requester@example.com' THEN 'Requester' ELSE 'Voter' END
FROM users u
WHERE u.email IN ('quorumdecl.requester@example.com', 'quorumdecl.voter1@example.com',
                  'quorumdecl.voter2@example.com', 'quorumdecl.voter3@example.com')
ON CONFLICT (user_id, application_id) DO UPDATE SET role = EXCLUDED.role;
