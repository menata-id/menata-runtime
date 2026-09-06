-- seeds/008_journal_entry.sql
-- CAP-F16 (line items / child table) real forcing case -- Study 6's own
-- accounting benchmark named Journal Entry + Journal Entry Line as the
-- textbook header-detail document (Odoo account.move + account.move.line,
-- Frappe Journal Entry + child table). Deliberately minimal: Account is a
-- plain text field here, not a real Chart of Account reference (CAP-O02
-- master data designation doesn't exist yet) -- the point of this seed is
-- proving CAP-F16's atomic parent+children authoring, not a full accounting
-- module. Double-entry balance (sum(debit) = sum(credit), CAP-C10) is
-- deliberately NOT enforced here either -- CAP-C10 depends on this seed
-- existing first, and is its own separate capability, not bundled in.

INSERT INTO workspaces (id, name, slug) VALUES
    ('ws_default', 'Default Workspace', 'ws_default')
ON CONFLICT (id) DO NOTHING;

INSERT INTO applications (id, workspace_id, name) VALUES
    ('app_accounting', 'ws_default', 'Accounting')
ON CONFLICT (id) DO NOTHING;

INSERT INTO machines (id, application_id, name) VALUES
    ('mch_journal_entry',      'app_accounting', 'Journal Entry'),
    ('mch_journal_entry_line', 'app_accounting', 'Journal Entry Line')
ON CONFLICT (id) DO NOTHING;

-- Fields
INSERT INTO fields (id, machine_id, name, type, position, required, options) VALUES
    ('fld_je_date',   'mch_journal_entry', 'Date',   'date',       0, true,  '{}'),
    ('fld_je_memo',   'mch_journal_entry', 'Memo',   'text',       1, true,  '{}'),
    ('fld_je_status', 'mch_journal_entry', 'Status', 'value_list', 2, false, '{"values":["Draft","Posted"]}'),

    ('fld_jel_journal_entry', 'mch_journal_entry_line', 'Journal Entry', 'reference', 0, true,
     '{"target_machine":"mch_journal_entry"}'),
    ('fld_jel_account', 'mch_journal_entry_line', 'Account', 'text',   1, true, '{}'),
    ('fld_jel_debit',   'mch_journal_entry_line', 'Debit',   'number', 2, false, '{}'),
    ('fld_jel_credit',  'mch_journal_entry_line', 'Credit',  'number', 3, false, '{}')
ON CONFLICT (id) DO NOTHING;

-- Events
INSERT INTO events (id, machine_id, name, position, condition) VALUES
    ('evt_je_post', 'mch_journal_entry', 'Post', 0,
     '{"field":"fld_je_status","operator":"equals","value":"Draft"}')
ON CONFLICT (id) DO NOTHING;

INSERT INTO event_actions (event_id, type, position, params) VALUES
    ('evt_je_post', 'set_field', 0, '{"field":"fld_je_status","value":"Posted"}');

-- Constraints
INSERT INTO constraints (id, machine_id, rule, expression, condition, position) VALUES
    ('cst_je_memo_required', 'mch_journal_entry', 'Memo is required.',
     '{"field":"fld_je_memo","operator":"required"}', NULL, 0),
    ('cst_jel_account_required', 'mch_journal_entry_line', 'Account is required.',
     '{"field":"fld_jel_account","operator":"required"}', NULL, 0)
ON CONFLICT (id) DO NOTHING;

-- Permissions
INSERT INTO permissions (id, machine_id, role, events) VALUES
    ('perm_je_accountant',  'mch_journal_entry',      'Accountant', ARRAY['evt_je_post']),
    ('perm_jel_accountant', 'mch_journal_entry_line', 'Accountant', ARRAY[]::TEXT[])
ON CONFLICT (id) DO NOTHING;

-- Views
INSERT INTO views (id, machine_id, name, type, position, config) VALUES
    ('vw_je_form', 'mch_journal_entry', 'Journal Entry Form', 'form', 0,
     '{"fields":["fld_je_date","fld_je_memo"],"child_lines":{"machine":"mch_journal_entry_line","parent_field":"fld_jel_journal_entry","fields":["fld_jel_account","fld_jel_debit","fld_jel_credit"],"max_rows":5}}'),
    ('vw_je_list', 'mch_journal_entry', 'Journal Entries', 'list', 1,
     '{"columns":["fld_je_date","fld_je_memo","fld_je_status"]}'),
    ('vw_je_detail', 'mch_journal_entry', 'Journal Entry Detail', 'detail', 2, '{}'),
    ('vw_jel_list', 'mch_journal_entry_line', 'Journal Entry Lines', 'list', 0,
     '{"columns":["fld_jel_account","fld_jel_debit","fld_jel_credit"]}'),
    ('vw_jel_detail', 'mch_journal_entry_line', 'Journal Entry Line Detail', 'detail', 1, '{}')
ON CONFLICT (id) DO NOTHING;

-- Account (CAP-X02/CAP-O01, self-contained here rather than in
-- seeds/007_authentication.sql -- that file runs before this one in
-- `make seed`'s ordering, and its role-assignment INSERT would violate the
-- applications FK if it referenced app_accounting before this file creates
-- it). Same password convention ("password") as every other seeded account.
INSERT INTO users (workspace_id, name, email, password_hash, workspace_role) VALUES
    ('ws_default', 'Ivy', 'accountant@example.com',
     '$2a$10$moxxOcZzSu3ILTzJlLF2Q.9vxiGNnSXPl7kY1pT3t5o1FoDjqC8aK', 'Member')
ON CONFLICT (workspace_id, email) DO NOTHING;

INSERT INTO user_application_roles (user_id, application_id, role)
SELECT u.id, 'app_accounting', 'Accountant' FROM users u WHERE u.email = 'accountant@example.com'
ON CONFLICT (user_id, application_id) DO UPDATE SET role = EXCLUDED.role;
