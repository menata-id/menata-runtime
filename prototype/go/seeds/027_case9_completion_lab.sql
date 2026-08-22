-- seeds/027_case9_completion_lab.sql
-- CAP-C08 proof (general cross-record constraint), realized through CAP-C10
-- (aggregate compare: sum(debit) = sum(credit)) and CAP-C11 (reference-field
-- check: no posting into a closed Fiscal Period) -- case-portfolio.md's
-- Case 9 (Accounting).
--
-- A NEW, self-contained lab (new Application, new Machines) -- the existing
-- seeds/008_journal_entry.sql fixture (already conformance-tested) is
-- deliberately left untouched: adding an unconditionally-checked new
-- Constraint to an already-tested Machine risks breaking whatever existing
-- fixture data doesn't happen to balance debit=credit. Its own header
-- comment already disclaimed CAP-C10 for exactly this reason ("depends on
-- this seed existing first... not bundled in").

INSERT INTO workspaces (id, name) VALUES ('ws_default', 'Default Workspace')
ON CONFLICT (id) DO NOTHING;

INSERT INTO applications (id, workspace_id, name) VALUES
    ('app_case9_lab', 'ws_default', 'Case 9 Completion Lab')
ON CONFLICT (id) DO NOTHING;

INSERT INTO machines (id, application_id, name) VALUES
    ('mch_c9_fiscal_period', 'app_case9_lab', 'Fiscal Period'),
    ('mch_c9_journal_entry', 'app_case9_lab', 'Journal Entry'),
    ('mch_c9_journal_entry_line', 'app_case9_lab', 'Journal Entry Line')
ON CONFLICT (id) DO NOTHING;

INSERT INTO fields (id, machine_id, name, type, position, required, options) VALUES
    ('fld_c9fp_name',   'mch_c9_fiscal_period', 'Name',   'text', 0, true, '{}'),
    ('fld_c9fp_status', 'mch_c9_fiscal_period', 'Status', 'value_list', 1, false, '{"values":["Open","Closed"]}'),

    ('fld_c9je_memo',   'mch_c9_journal_entry', 'Memo',   'text', 0, true, '{}'),
    ('fld_c9je_period', 'mch_c9_journal_entry', 'Fiscal Period', 'reference', 1, true, '{"target_machine":"mch_c9_fiscal_period"}'),
    ('fld_c9je_status', 'mch_c9_journal_entry', 'Status', 'value_list', 2, false, '{"values":["Draft","Posted"]}'),

    ('fld_c9jel_entry',  'mch_c9_journal_entry_line', 'Journal Entry', 'reference', 0, true, '{"target_machine":"mch_c9_journal_entry"}'),
    ('fld_c9jel_debit',  'mch_c9_journal_entry_line', 'Debit',  'number', 1, false, '{}'),
    ('fld_c9jel_credit', 'mch_c9_journal_entry_line', 'Credit', 'number', 2, false, '{}')
ON CONFLICT (id) DO NOTHING;

INSERT INTO events (id, machine_id, name, position, condition) VALUES
    ('evt_c9je_post',   'mch_c9_journal_entry',  'Post',  0, '{"field":"fld_c9je_status","operator":"equals","value":"Draft"}'),
    ('evt_c9fp_close',  'mch_c9_fiscal_period',  'Close', 0, '{"field":"fld_c9fp_status","operator":"equals","value":"Open"}')
ON CONFLICT (id) DO NOTHING;

INSERT INTO event_actions (event_id, type, position, params) VALUES
    ('evt_c9je_post',  'set_field', 0, '{"field":"fld_c9je_status","value":"Posted"}'),
    ('evt_c9fp_close', 'set_field', 0, '{"field":"fld_c9fp_status","value":"Closed"}');

-- CAP-C10: sum(debit) = sum(credit) across every Line referencing this
-- Entry, checked only on Post (Condition gates it -- CAP-C09 re-validates
-- against the SIMULATED post-action data, so "status equals Posted" is
-- true exactly when this transition is the one being evaluated).
-- `expression` is a required NOT NULL column but is never evaluated for a
-- cross_record constraint (constraint.Engine.Violations skips any Constraint
-- with CrossRecord set) -- a harmless, inert placeholder, not a real rule.
INSERT INTO constraints (id, machine_id, rule, expression, position, cross_record, condition) VALUES
    ('cst_c9je_balance', 'mch_c9_journal_entry',
     'Total Debit must equal Total Credit before posting.',
     '{"field":"fld_c9je_status","operator":"required"}', 0,
     '{"kind":"aggregate","child_machine":"mch_c9_journal_entry_line","scope_field":"fld_c9jel_entry","field_a":"fld_c9jel_debit","field_b":"fld_c9jel_credit","operator":"equals"}',
     '{"field":"fld_c9je_status","operator":"equals","value":"Posted"}'),
    -- CAP-C11: the referenced Fiscal Period must not be Closed, checked the
    -- same way -- only on Post.
    ('cst_c9je_period_open', 'mch_c9_journal_entry',
     'Fiscal Period must be Open to post entries.',
     '{"field":"fld_c9je_status","operator":"required"}', 1,
     '{"kind":"reference_field","reference_field":"fld_c9je_period","target_field":"fld_c9fp_status","operator":"not_equals","value":"Closed"}',
     '{"field":"fld_c9je_status","operator":"equals","value":"Posted"}')
ON CONFLICT (id) DO NOTHING;

INSERT INTO permissions (id, machine_id, role, events, can_read, can_create, can_edit, can_delete) VALUES
    ('perm_c9fp_accountant',  'mch_c9_fiscal_period',      'Accountant', ARRAY['evt_c9fp_close'], true, true, true, false),
    ('perm_c9je_accountant',  'mch_c9_journal_entry',      'Accountant', ARRAY['evt_c9je_post'], true, true, true, false),
    ('perm_c9jel_accountant', 'mch_c9_journal_entry_line', 'Accountant', ARRAY[]::TEXT[], true, true, true, false)
ON CONFLICT (id) DO NOTHING;

INSERT INTO views (id, machine_id, name, type, position, config) VALUES
    ('vw_c9fp_list', 'mch_c9_fiscal_period', 'Fiscal Periods', 'list', 0, '{"columns":["fld_c9fp_name","fld_c9fp_status"]}'),
    ('vw_c9fp_form', 'mch_c9_fiscal_period', 'New Fiscal Period', 'form', 1, '{"fields":["fld_c9fp_name"]}'),
    ('vw_c9je_list', 'mch_c9_journal_entry', 'Journal Entries', 'list', 0, '{"columns":["fld_c9je_memo","fld_c9je_status"]}'),
    ('vw_c9je_form', 'mch_c9_journal_entry', 'New Journal Entry', 'form', 1, '{"fields":["fld_c9je_memo","fld_c9je_period"]}'),
    ('vw_c9jel_list', 'mch_c9_journal_entry_line', 'Journal Entry Lines', 'list', 0, '{"columns":["fld_c9jel_entry","fld_c9jel_debit","fld_c9jel_credit"]}'),
    ('vw_c9jel_form', 'mch_c9_journal_entry_line', 'New Journal Entry Line', 'form', 1, '{"fields":["fld_c9jel_entry","fld_c9jel_debit","fld_c9jel_credit"]}')
ON CONFLICT (id) DO NOTHING;

INSERT INTO users (workspace_id, name, email, password_hash, workspace_role) VALUES
    ('ws_default', 'Case9 Accountant', 'case9.accountant@example.com',
     '$2a$10$moxxOcZzSu3ILTzJlLF2Q.9vxiGNnSXPl7kY1pT3t5o1FoDjqC8aK', 'Member')
ON CONFLICT (workspace_id, email) DO NOTHING;

INSERT INTO user_application_roles (user_id, application_id, role)
SELECT u.id, 'app_case9_lab', 'Accountant' FROM users u WHERE u.email = 'case9.accountant@example.com'
ON CONFLICT (user_id, application_id) DO UPDATE SET role = EXCLUDED.role;
