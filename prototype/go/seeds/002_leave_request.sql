-- seeds/002_leave_request.sql
-- Machine ke-2: Leave Request — domain HR.
-- Dimasukkan ke workspace yang sama (ws_default) tapi application berbeda (app_hr).
-- Tidak ada perubahan kode — pure metadata.

-- Application baru: HR
INSERT INTO applications (id, workspace_id, name) VALUES
    ('app_hr', 'ws_default', 'HR')
ON CONFLICT (id) DO NOTHING;

-- Machine: Leave Request
INSERT INTO machines (id, application_id, name) VALUES
    ('mch_leave_request', 'app_hr', 'Leave Request')
ON CONFLICT (id) DO NOTHING;

-- Fields
INSERT INTO fields (id, machine_id, name, type, position, required, options) VALUES
    ('fld_lr_employee',   'mch_leave_request', 'Employee',   'user',       0, true,  '{}'),
    ('fld_lr_leave_type', 'mch_leave_request', 'Leave Type', 'value_list', 1, true,  '{"values":["Annual Leave","Sick Leave","Emergency Leave","Unpaid Leave"]}'),
    ('fld_lr_start_date', 'mch_leave_request', 'Start Date', 'date',       2, true,  '{}'),
    ('fld_lr_end_date',   'mch_leave_request', 'End Date',   'date',       3, true,  '{}'),
    ('fld_lr_reason',     'mch_leave_request', 'Reason',     'rich_text',  4, true,  '{}'),
    ('fld_lr_status',     'mch_leave_request', 'Status',     'value_list', 5, false, '{"values":["Draft","Submitted","Approved","Rejected","Cancelled"]}')
ON CONFLICT (id) DO NOTHING;

-- Events
INSERT INTO events (id, machine_id, name, position) VALUES
    ('evt_lr_submit',  'mch_leave_request', 'Submit',  0),
    ('evt_lr_approve', 'mch_leave_request', 'Approve', 1),
    ('evt_lr_reject',  'mch_leave_request', 'Reject',  2),
    ('evt_lr_cancel',  'mch_leave_request', 'Cancel',  3)
ON CONFLICT (id) DO NOTHING;

-- Event Actions
INSERT INTO event_actions (event_id, type, position, params) VALUES
    ('evt_lr_submit',  'set_field', 0, '{"field":"fld_lr_status","value":"Submitted"}'),
    ('evt_lr_approve', 'set_field', 0, '{"field":"fld_lr_status","value":"Approved"}'),
    ('evt_lr_approve', 'notify',    1, '{"role":"Employee"}'),
    ('evt_lr_reject',  'set_field', 0, '{"field":"fld_lr_status","value":"Rejected"}'),
    ('evt_lr_reject',  'notify',    1, '{"role":"Employee"}'),
    ('evt_lr_cancel',  'set_field', 0, '{"field":"fld_lr_status","value":"Cancelled"}');

-- Constraints
INSERT INTO constraints (id, machine_id, rule, expression, condition, position) VALUES
    ('cst_lr_reason_required',
     'mch_leave_request',
     'Reason is required.',
     '{"field":"fld_lr_reason","operator":"required"}',
     NULL, 0),
    ('cst_lr_start_future',
     'mch_leave_request',
     'Start Date must be after today.',
     '{"field":"fld_lr_start_date","operator":"after","value":"today"}',
     NULL, 1)
ON CONFLICT (id) DO NOTHING;

-- Permissions
INSERT INTO permissions (id, machine_id, role, events) VALUES
    ('perm_lr_employee', 'mch_leave_request', 'Employee', ARRAY['evt_lr_submit','evt_lr_cancel']),
    ('perm_lr_manager',  'mch_leave_request', 'Manager',  ARRAY['evt_lr_approve','evt_lr_reject'])
ON CONFLICT (id) DO NOTHING;

-- Views
INSERT INTO views (id, machine_id, name, type, position, config) VALUES
    ('vw_lr_form',
     'mch_leave_request', 'Leave Request Form', 'form', 0,
     '{"fields":["fld_lr_employee","fld_lr_leave_type","fld_lr_start_date","fld_lr_end_date","fld_lr_reason"]}'),
    ('vw_lr_my_requests',
     'mch_leave_request', 'My Requests', 'list', 1,
     '{"columns":["fld_lr_leave_type","fld_lr_start_date","fld_lr_end_date","fld_lr_status"],"default_sort":{"field":"fld_lr_start_date","direction":"asc"}}'),
    ('vw_lr_pending',
     'mch_leave_request', 'Pending Approvals', 'list', 2,
     '{"columns":["fld_lr_employee","fld_lr_leave_type","fld_lr_start_date","fld_lr_end_date","fld_lr_status"],"default_sort":{"field":"fld_lr_start_date","direction":"asc"}}'),
    ('vw_lr_detail',
     'mch_leave_request', 'Leave Request Detail', 'detail', 3,
     '{}')
ON CONFLICT (id) DO NOTHING;
