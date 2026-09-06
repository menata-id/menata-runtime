-- seeds/003_hr_employee.sql
-- Seed Runtime Metadata for hr-employee.yaml (Case 18).
-- CAP-F13 proof case: fld_emp_manager is a self-reference (Employee -> Employee),
-- the tree/hierarchy option's second case instance after Case 9's Chart of Account.
-- Safe to run multiple times (ON CONFLICT DO NOTHING).

-- Workspace
INSERT INTO workspaces (id, name) VALUES
    ('ws_default', 'Default Workspace')
ON CONFLICT (id) DO NOTHING;

-- Application
INSERT INTO applications (id, workspace_id, name) VALUES
    ('app_hr', 'ws_default', 'HR')
ON CONFLICT (id) DO NOTHING;

-- Machine
INSERT INTO machines (id, application_id, name) VALUES
    ('mch_employee', 'app_hr', 'Employee')
ON CONFLICT (id) DO NOTHING;

-- Fields
-- fld_emp_id: CAP-F18 auto-numbering [NOT YET] -- entered manually for now.
-- fld_emp_manager: CAP-F13 reference, target_machine = mch_employee (self-reference).
INSERT INTO fields (id, machine_id, name, type, position, required, options) VALUES
    ('fld_emp_id',         'mch_employee', 'Employee ID', 'text',       0, true,  '{}'),
    ('fld_emp_name',       'mch_employee', 'Name',        'text',       1, true,  '{}'),
    ('fld_emp_department', 'mch_employee', 'Department',  'text',       2, false, '{}'),
    ('fld_emp_position',   'mch_employee', 'Position',    'text',       3, false, '{}'),
    ('fld_emp_manager',    'mch_employee', 'Manager',     'reference',  4, false, '{"target_machine":"mch_employee"}'),
    ('fld_emp_hire_date',  'mch_employee', 'Hire Date',   'date',       5, true,  '{}'),
    ('fld_emp_status',     'mch_employee', 'Status',      'value_list', 6, false, '{"values":["Active","Terminated"]}')
ON CONFLICT (id) DO NOTHING;

-- No Events -- Employee has none in hr-employee.menata (not every Object needs one).

-- Constraints
INSERT INTO constraints (id, machine_id, rule, expression, condition, position) VALUES
    ('cst_emp_id_required',
     'mch_employee',
     'Employee ID is required.',
     '{"field":"fld_emp_id","operator":"required"}',
     NULL, 0),
    ('cst_emp_name_required',
     'mch_employee',
     'Name is required.',
     '{"field":"fld_emp_name","operator":"required"}',
     NULL, 1),
    ('cst_emp_hire_date_required',
     'mch_employee',
     'Hire Date is required.',
     '{"field":"fld_emp_hire_date","operator":"required"}',
     NULL, 2)
ON CONFLICT (id) DO NOTHING;

-- Permissions
INSERT INTO permissions (id, machine_id, role, events) VALUES
    ('perm_hr', 'mch_employee', 'HR', ARRAY[]::TEXT[])
ON CONFLICT (id) DO NOTHING;

-- Views
INSERT INTO views (id, machine_id, name, type, position, config) VALUES
    ('vw_emp_form',
     'mch_employee', 'Employee Form', 'form', 0,
     '{"fields":["fld_emp_id","fld_emp_name","fld_emp_department","fld_emp_position","fld_emp_manager","fld_emp_hire_date"]}'),
    ('vw_emp_directory',
     'mch_employee', 'Employee Directory', 'list', 1,
     '{"columns":["fld_emp_id","fld_emp_name","fld_emp_department","fld_emp_manager","fld_emp_status"],"default_sort":{"field":"created_at","direction":"desc"}}'),
    ('vw_emp_detail',
     'mch_employee', 'Employee Detail', 'detail', 2,
     '{}')
ON CONFLICT (id) DO NOTHING;
