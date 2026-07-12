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
-- fld_lr_approved_date / fld_lr_approved_by: stamped by evt_lr_approve's dynamic
-- values (CAP-A02, "today" / "current_user") -- not set at Create, not in the form.
INSERT INTO fields (id, machine_id, name, type, position, required, options) VALUES
    ('fld_lr_employee',      'mch_leave_request', 'Employee',      'user',       0, true,  '{}'),
    ('fld_lr_leave_type',    'mch_leave_request', 'Leave Type',    'value_list', 1, true,  '{"values":["Annual Leave","Sick Leave","Emergency Leave","Unpaid Leave"]}'),
    ('fld_lr_start_date',    'mch_leave_request', 'Start Date',    'date',       2, true,  '{}'),
    ('fld_lr_end_date',      'mch_leave_request', 'End Date',      'date',       3, true,  '{}'),
    ('fld_lr_reason',        'mch_leave_request', 'Reason',        'rich_text',  4, true,  '{}'),
    ('fld_lr_status',        'mch_leave_request', 'Status',        'value_list', 5, false, '{"values":["Draft","Submitted","Approved","Rejected","Cancelled"]}'),
    ('fld_lr_approved_date', 'mch_leave_request', 'Approved Date', 'date',       6, false, '{}'),
    ('fld_lr_approved_by',   'mch_leave_request', 'Approved By',   'text',       7, false, '{}')
ON CONFLICT (id) DO NOTHING;

-- Events
-- condition (CAP-E06): the event may only fire from the given current Status --
-- e.g. Approve/Reject/Cancel all require Submitted, so an already-Approved
-- record can no longer be Rejected (the exact gap Study 1 named).
INSERT INTO events (id, machine_id, name, position, condition) VALUES
    ('evt_lr_submit',  'mch_leave_request', 'Submit',  0, '{"field":"fld_lr_status","operator":"equals","value":"Draft"}'),
    ('evt_lr_approve', 'mch_leave_request', 'Approve', 1, '{"field":"fld_lr_status","operator":"equals","value":"Submitted"}'),
    ('evt_lr_reject',  'mch_leave_request', 'Reject',  2, '{"field":"fld_lr_status","operator":"equals","value":"Submitted"}'),
    ('evt_lr_cancel',  'mch_leave_request', 'Cancel',  3, '{"field":"fld_lr_status","operator":"equals","value":"Submitted"}')
ON CONFLICT (id) DO NOTHING;

-- Event Actions
-- CAP-A02: evt_lr_approve stamps Approved Date ("today") and Approved By
-- ("current_user" -- resolves to the acting role; this prototype has no
-- per-user session, see internal/executor.Simulate's doc comment).
INSERT INTO event_actions (event_id, type, position, params) VALUES
    ('evt_lr_submit',  'set_field', 0, '{"field":"fld_lr_status","value":"Submitted"}'),
    ('evt_lr_approve', 'set_field', 0, '{"field":"fld_lr_status","value":"Approved"}'),
    ('evt_lr_approve', 'set_field', 1, '{"field":"fld_lr_approved_date","value":"today"}'),
    ('evt_lr_approve', 'set_field', 2, '{"field":"fld_lr_approved_by","value":"current_user"}'),
    ('evt_lr_approve', 'notify',    3, '{"role":"Employee"}'),
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
     NULL, 1),
    ('cst_lr_end_after_start',
     'mch_leave_request',
     'End Date must be after Start Date.',
     '{"field":"fld_lr_end_date","operator":"after","value_field":"fld_lr_start_date"}',
     NULL, 2)
ON CONFLICT (id) DO NOTHING;
-- CAP-C07 (cross-field comparison, 2026-07-12): the canonical "End Date
-- after Start Date" case this capability was named for -- compares against
-- another Field's own value (value_field), not a literal, so it stays
-- correct no matter what Start Date is, unlike a hardcoded date would.

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
