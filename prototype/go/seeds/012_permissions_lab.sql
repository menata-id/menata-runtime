-- seeds/012_permissions_lab.sql
-- Proof harness for Batch 6 (2026-07-12): CAP-P03 (separation of duties),
-- CAP-P04 (delegation), CAP-P06 (field-level visibility), CAP-P07
-- (public/unauthenticated read access). CAP-P05 (deny-by-default CRUD)
-- already shipped earlier, not part of this batch.

INSERT INTO applications (id, workspace_id, name) VALUES
    ('app_permissions_lab', 'ws_default', 'Permissions Lab')
ON CONFLICT (id) DO NOTHING;

INSERT INTO machines (id, application_id, name, config) VALUES
    ('mch_pl_expense', 'app_permissions_lab', 'Expense Report', NULL),
    -- CAP-P03: sod_reference_field/sod_requester_field -- the person who
    -- submitted the referenced Expense Report may not also decide its own
    -- Approval, even when they hold the Approver role and are the
    -- assigned owner_field (CAP-P02) too.
    ('mch_pl_expense_approval', 'app_permissions_lab', 'Expense Approval',
     '{"sod_reference_field":"fld_plea_expense","sod_requester_field":"fld_ple_submitted_by"}'),
    ('mch_pl_employee', 'app_permissions_lab', 'Employee', NULL),
    ('mch_pl_post', 'app_permissions_lab', 'Blog Post', NULL)
ON CONFLICT (id) DO NOTHING;

-- Fields
INSERT INTO fields (id, machine_id, name, type, position, required, options) VALUES
    ('fld_ple_title',         'mch_pl_expense', 'Title',         'text',       0, true,  '{}'),
    ('fld_ple_submitted_by',  'mch_pl_expense', 'Submitted By',  'user',       1, false, '{}'),
    ('fld_ple_status',        'mch_pl_expense', 'Status',        'value_list', 2, false, '{"values":["Draft","Approved"]}'),

    ('fld_plea_expense',      'mch_pl_expense_approval', 'Expense',      'reference',  0, true,  '{"target_machine":"mch_pl_expense"}'),
    ('fld_plea_approver',     'mch_pl_expense_approval', 'Approver',     'user',       1, false, '{}'),
    ('fld_plea_decision',     'mch_pl_expense_approval', 'Decision',     'value_list', 2, false, '{"values":["Pending","Approved"]}'),
    -- CAP-P04: stamped by evt_plea_delegate, distinct from Approver itself
    -- so a delegation leaves an accountability trail, not just a silent
    -- reassignment.
    ('fld_plea_delegated_by', 'mch_pl_expense_approval', 'Delegated By', 'user',       3, false, '{}'),

    ('fld_ple2_name',   'mch_pl_employee', 'Name',   'text',   0, true,  '{}'),
    -- CAP-P06's own proof field -- hidden from Staff, visible to HR.
    ('fld_ple2_salary', 'mch_pl_employee', 'Salary', 'number', 1, false, '{}'),

    ('fld_plp_title',  'mch_pl_post', 'Title',  'text',       0, true,  '{}'),
    ('fld_plp_body',   'mch_pl_post', 'Body',   'text',       1, false, '{}'),
    ('fld_plp_status', 'mch_pl_post', 'Status', 'value_list', 2, false, '{"values":["Draft","Published"]}')
ON CONFLICT (id) DO NOTHING;

-- Events
INSERT INTO events (id, machine_id, name, position, condition, input_fields) VALUES
    ('evt_plea_approve', 'mch_pl_expense_approval', 'Approve', 0, NULL, ARRAY[]::TEXT[]),
    -- CAP-P04: the delegator re-submits fld_plea_approver's OWN field id
    -- with a NEW value (who to hand off to) -- collected fresh at trigger
    -- time (handler.TriggerEvent), not read from the record's existing
    -- data, since "who to delegate to" can't be known beforehand.
    ('evt_plea_delegate', 'mch_pl_expense_approval', 'Delegate', 1, NULL, ARRAY['fld_plea_approver'])
ON CONFLICT (id) DO NOTHING;

INSERT INTO event_actions (event_id, type, position, params) VALUES
    ('evt_plea_approve', 'set_field', 0, '{"field":"fld_plea_decision","value":"Approved"}'),
    ('evt_plea_delegate', 'set_field', 0, '{"field":"fld_plea_delegated_by","value":"current_user"}'),
    ('evt_plea_delegate', 'set_field', 1, '{"field":"fld_plea_approver","value":"input:fld_plea_approver"}');

-- Permissions. can_read/can_create/can_edit default true (migrations/006),
-- can_delete defaults false (migrations/012) -- explicit here where it
-- matters for this batch's own proofs.
INSERT INTO permissions (id, machine_id, role, events, owner_field, hidden_fields) VALUES
    ('perm_ple_member',  'mch_pl_expense', 'Member', ARRAY[]::TEXT[], NULL, ARRAY[]::TEXT[]),
    -- CAP-P02's own owner_field (only the assigned Approver may decide)
    -- composes with CAP-P03's separation-of-duties Machine.Config check
    -- above -- both must pass, not either/or.
    ('perm_plea_member', 'mch_pl_expense_approval', 'Member',
     ARRAY['evt_plea_approve','evt_plea_delegate'], 'fld_plea_approver', ARRAY[]::TEXT[]),
    ('perm_ple2_hr',    'mch_pl_employee', 'HR',    ARRAY[]::TEXT[], NULL, ARRAY[]::TEXT[]),
    -- CAP-P06: Staff can read/create/edit Employee records at all (CAP-P05
    -- CRUD tier), but never sees fld_ple2_salary -- List columns, Detail
    -- fields, both.
    ('perm_ple2_staff', 'mch_pl_employee', 'Staff', ARRAY[]::TEXT[], NULL, ARRAY['fld_ple2_salary'])
ON CONFLICT (id) DO NOTHING;

INSERT INTO permissions (id, machine_id, role, events, can_read, can_create, can_edit, can_delete) VALUES
    ('perm_plp_member', 'mch_pl_post', 'Member', ARRAY[]::TEXT[], true, true, true, false),
    -- CAP-P07: role = 'Visitor' is not a restricted account -- it's the
    -- ABSENCE of one (cmd/server/main.go's visitorAuth). Read-only,
    -- explicit false on the other three even though a Visitor session can
    -- never reach a POST route anyway (visitorAuth only ever grants a GET) --
    -- defense in depth, not load-bearing on its own.
    ('perm_plp_visitor', 'mch_pl_post', 'Visitor', ARRAY[]::TEXT[], true, false, false, false)
ON CONFLICT (id) DO NOTHING;

-- Views
INSERT INTO views (id, machine_id, name, type, position, config) VALUES
    ('vw_ple_form',   'mch_pl_expense', 'Expense Report Form', 'form', 0,
     '{"fields":["fld_ple_title","fld_ple_submitted_by"]}'),
    ('vw_ple_list',   'mch_pl_expense', 'Expense Reports',     'list', 1,
     '{"columns":["fld_ple_title","fld_ple_submitted_by","fld_ple_status"]}'),
    ('vw_ple_detail', 'mch_pl_expense', 'Expense Report Detail', 'detail', 2, '{}'),

    ('vw_plea_form',   'mch_pl_expense_approval', 'Expense Approval Form', 'form', 0,
     '{"fields":["fld_plea_expense","fld_plea_approver"]}'),
    ('vw_plea_list',   'mch_pl_expense_approval', 'Expense Approvals',     'list', 1,
     '{"columns":["fld_plea_expense","fld_plea_approver","fld_plea_decision","fld_plea_delegated_by"]}'),
    ('vw_plea_detail', 'mch_pl_expense_approval', 'Expense Approval Detail', 'detail', 2, '{}'),

    ('vw_ple2_form',   'mch_pl_employee', 'Employee Form', 'form', 0,
     '{"fields":["fld_ple2_name","fld_ple2_salary"]}'),
    ('vw_ple2_list',   'mch_pl_employee', 'Employees',     'list', 1,
     '{"columns":["fld_ple2_name","fld_ple2_salary"]}'),
    ('vw_ple2_detail', 'mch_pl_employee', 'Employee Detail', 'detail', 2, '{}'),

    ('vw_plp_form',   'mch_pl_post', 'Blog Post Form', 'form', 0,
     '{"fields":["fld_plp_title","fld_plp_body","fld_plp_status"]}'),
    ('vw_plp_list',   'mch_pl_post', 'Blog Posts',     'list', 1,
     '{"columns":["fld_plp_title","fld_plp_status"]}'),
    ('vw_plp_detail', 'mch_pl_post', 'Blog Post Detail', 'detail', 2, '{}')
ON CONFLICT (id) DO NOTHING;

-- Accounts (self-contained, same reasoning as prior lab seed files).
INSERT INTO users (workspace_id, name, email, password_hash, workspace_role) VALUES
    ('ws_default', 'Nora', 'nora@example.com', '$2a$10$moxxOcZzSu3ILTzJlLF2Q.9vxiGNnSXPl7kY1pT3t5o1FoDjqC8aK', 'Member'),
    ('ws_default', 'Omar', 'omar@example.com', '$2a$10$moxxOcZzSu3ILTzJlLF2Q.9vxiGNnSXPl7kY1pT3t5o1FoDjqC8aK', 'Member'),
    ('ws_default', 'Hana', 'hana@example.com', '$2a$10$moxxOcZzSu3ILTzJlLF2Q.9vxiGNnSXPl7kY1pT3t5o1FoDjqC8aK', 'Member'),
    ('ws_default', 'Iris', 'iris@example.com', '$2a$10$moxxOcZzSu3ILTzJlLF2Q.9vxiGNnSXPl7kY1pT3t5o1FoDjqC8aK', 'Member')
ON CONFLICT (workspace_id, email) DO NOTHING;

INSERT INTO user_application_roles (user_id, application_id, role)
SELECT u.id, 'app_permissions_lab', r.role
FROM users u JOIN (VALUES
    ('nora@example.com', 'Member'),
    ('omar@example.com', 'Member'),
    ('hana@example.com', 'HR'),
    -- CAP-P06's own proof account -- Staff can read mch_pl_employee at
    -- all (CAP-P05's CRUD tier) but never sees fld_ple2_salary.
    ('iris@example.com', 'Staff')
) AS r(email, role) ON u.email = r.email
ON CONFLICT (user_id, application_id) DO UPDATE SET role = EXCLUDED.role;
