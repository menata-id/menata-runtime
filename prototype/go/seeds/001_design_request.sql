-- seeds/001_design_request.sql
-- Seed Runtime Metadata for design-request.yaml example.
-- Safe to run multiple times (ON CONFLICT DO NOTHING).

-- Workspace
INSERT INTO workspaces (id, name) VALUES
    ('ws_default', 'Default Workspace')
ON CONFLICT (id) DO NOTHING;

-- Application
INSERT INTO applications (id, workspace_id, name) VALUES
    ('app_design', 'ws_default', 'Design')
ON CONFLICT (id) DO NOTHING;

-- Machine
INSERT INTO machines (id, application_id, name) VALUES
    ('mch_design_request', 'app_design', 'Design Request')
ON CONFLICT (id) DO NOTHING;

-- Fields
INSERT INTO fields (id, machine_id, name, type, position, options) VALUES
    ('fld_requester',    'mch_design_request', 'Requester',   'user',       0, '{}'),
    ('fld_design_type',  'mch_design_request', 'Design Type', 'value_list', 1, '{"values":["Poster","Thumbnail","Banner 2:1"]}'),
    ('fld_due_date',     'mch_design_request', 'Due Date',    'date',       2, '{}'),
    ('fld_title',        'mch_design_request', 'Title',       'text',       3, '{}'),
    ('fld_description',  'mch_design_request', 'Description', 'rich_text',  4, '{}'),
    ('fld_attachment',   'mch_design_request', 'Attachment',  'file',       5, '{}'),
    ('fld_status',       'mch_design_request', 'Status',      'value_list', 6, '{"values":["Draft","Submitted","Accepted","Rejected","In Progress","Completed"]}')
ON CONFLICT (id) DO NOTHING;

-- Events
INSERT INTO events (id, machine_id, name, position) VALUES
    ('evt_submit',   'mch_design_request', 'Submit',   0),
    ('evt_accept',   'mch_design_request', 'Accept',   1),
    ('evt_reject',   'mch_design_request', 'Reject',   2),
    ('evt_start',    'mch_design_request', 'Start',    3),
    ('evt_complete', 'mch_design_request', 'Complete', 4)
ON CONFLICT (id) DO NOTHING;

-- Event Actions
INSERT INTO event_actions (event_id, type, position, params) VALUES
    ('evt_submit',   'set_field', 0, '{"field":"fld_status","value":"Submitted"}'),
    ('evt_submit',   'notify',    1, '{"role":"Designer"}'),
    ('evt_accept',   'set_field', 0, '{"field":"fld_status","value":"Accepted"}'),
    ('evt_reject',   'set_field', 0, '{"field":"fld_status","value":"Rejected"}'),
    ('evt_start',    'set_field', 0, '{"field":"fld_status","value":"In Progress"}'),
    ('evt_complete', 'set_field', 0, '{"field":"fld_status","value":"Completed"}'),
    ('evt_complete', 'notify',    1, '{"role":"Requester"}');

-- Constraints
INSERT INTO constraints (id, machine_id, rule, expression, condition, position) VALUES
    ('cst_title_required',
     'mch_design_request',
     'Title is required.',
     '{"field":"fld_title","operator":"required"}',
     NULL, 0),
    ('cst_description_required',
     'mch_design_request',
     'Description is required.',
     '{"field":"fld_description","operator":"required"}',
     NULL, 1),
    ('cst_due_date_future',
     'mch_design_request',
     'Due Date must be after today.',
     '{"field":"fld_due_date","operator":"after","value":"today"}',
     NULL, 2),
    ('cst_attachment_required_for_banner',
     'mch_design_request',
     'Attachment is required for Banner design type.',
     '{"field":"fld_attachment","operator":"required"}',
     '{"field":"fld_design_type","operator":"equals","value":"Banner 2:1"}',
     3)
ON CONFLICT (id) DO NOTHING;

-- Permissions
INSERT INTO permissions (id, machine_id, role, events) VALUES
    ('perm_requester', 'mch_design_request', 'Requester', ARRAY['evt_submit']),
    ('perm_designer',  'mch_design_request', 'Designer',  ARRAY['evt_accept','evt_reject','evt_start','evt_complete'])
ON CONFLICT (id) DO NOTHING;

-- Views
INSERT INTO views (id, machine_id, name, type, position, config) VALUES
    ('vw_request_form',
     'mch_design_request', 'Request Form', 'form', 0,
     '{"fields":["fld_design_type","fld_due_date","fld_title","fld_description","fld_attachment"]}'),
    ('vw_my_requests',
     'mch_design_request', 'My Requests', 'list', 1,
     '{"columns":["fld_title","fld_design_type","fld_due_date","fld_status"],"default_sort":{"field":"created_at","direction":"desc"}}'),
    ('vw_design_queue',
     'mch_design_request', 'Design Queue', 'list', 2,
     '{"columns":["fld_title","fld_requester","fld_design_type","fld_due_date","fld_status"],"default_sort":{"field":"fld_due_date","direction":"asc"}}'),
    ('vw_request_detail',
     'mch_design_request', 'Request Detail', 'detail', 3,
     '{}')
ON CONFLICT (id) DO NOTHING;

-- Seed users for testing
-- password: 'password' hashed with bcrypt (cost 10)
INSERT INTO users (workspace_id, name, email, password_hash, role) VALUES
    ('ws_default', 'Alice Requester', 'alice@example.com',
     '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy', 'Requester'),
    ('ws_default', 'Bob Designer',    'bob@example.com',
     '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy', 'Designer')
ON CONFLICT (workspace_id, email) DO NOTHING;
