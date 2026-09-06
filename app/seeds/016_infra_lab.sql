-- seeds/016_infra_lab.sql
-- Proof harness for Batch 10 (Infra): CAP-X12 (cross-record write
-- atomicity) and CAP-X13 (webhook idempotency). CAP-X07 (auto-generated
-- JSON API) and CAP-X08 (metadata export) need no seed of their own --
-- they're proven directly against machines/records that already exist
-- from earlier batches.

INSERT INTO applications (id, workspace_id, name) VALUES
    ('app_infra_lab', 'ws_default', 'Infra Lab')
ON CONFLICT (id) DO NOTHING;

INSERT INTO machines (id, application_id, name, config) VALUES
    ('mch_x12_ledger', 'app_infra_lab', 'Ledger', NULL),
    ('mch_x12_entry',  'app_infra_lab', 'Entry',  NULL),
    -- Deliberately NO machines row for the id evt_x12_commit's second
    -- create_record action names ("mch_x12_ghost") -- that's the point:
    -- records.machine_id REFERENCES machines(id), so an INSERT against a
    -- dangling machine id fails with a real foreign-key violation,
    -- CAP-X12's proof mechanism. create_record's own params are trusted,
    -- unvalidated metadata (see executor.go's doCreateRecord doc comment)
    -- -- this dangling reference loads fine, only fails at trigger time.
    ('mch_x13_source', 'app_infra_lab', 'Webhook Source', '{"webhook_secret":"infra-lab-secret-2026"}'),
    ('mch_x13_log',    'app_infra_lab', 'Webhook Log',    NULL)
ON CONFLICT (id) DO NOTHING;

INSERT INTO fields (id, machine_id, name, type, position, required, options) VALUES
    ('fld_x12l_status', 'mch_x12_ledger', 'Status', 'value_list', 0, false, '{"values":["Draft","Posted"]}'),

    ('fld_x12e_note', 'mch_x12_entry', 'Note', 'text', 0, false, '{}'),

    ('fld_x13s_amount', 'mch_x13_source', 'Amount', 'number', 0, false, '{}'),

    ('fld_x13g_note', 'mch_x13_log', 'Note', 'text', 0, false, '{}')
ON CONFLICT (id) DO NOTHING;

-- CAP-X12: three actions spanning three different machine ids -- the
-- record's own set_field, a create_record against a REAL machine
-- (mch_x12_entry, would succeed on its own), and a create_record against a
-- DANGLING machine id (fails). Before this batch's fix, actions 0 and 1
-- would have silently committed while action 2 only logged an error --
-- this event exists specifically to prove they now all roll back together.
INSERT INTO events (id, machine_id, name, position, condition) VALUES
    ('evt_x12_commit', 'mch_x12_ledger', 'Commit', 0, NULL)
ON CONFLICT (id) DO NOTHING;

INSERT INTO event_actions (event_id, type, position, params) VALUES
    ('evt_x12_commit', 'set_field',     0, '{"field":"fld_x12l_status","value":"Posted"}'),
    ('evt_x12_commit', 'create_record', 1, '{"machine":"mch_x12_entry","fields":{"fld_x12e_note":"Logged"}}'),
    ('evt_x12_commit', 'create_record', 2, '{"machine":"mch_x12_ghost","fields":{"note":"Should never persist"}}');

-- CAP-X13: a webhook-triggered event that creates a real Log record each
-- time it actually runs -- a repeat delivery with the SAME
-- X-Idempotency-Key must NOT double this, so counting mch_x13_log's own
-- records is a direct, observable proof (unlike CAP-E04's own T102/T103,
-- whose action is idempotent-looking regardless -- setting Status=Paid
-- twice looks the same as once).
INSERT INTO events (id, machine_id, name, position, condition) VALUES
    ('evt_x13_log', 'mch_x13_source', 'Log', 0, NULL)
ON CONFLICT (id) DO NOTHING;

INSERT INTO event_actions (event_id, type, position, params) VALUES
    ('evt_x13_log', 'create_record', 0, '{"machine":"mch_x13_log","fields":{"fld_x13g_note":"Logged"}}');

INSERT INTO permissions (id, machine_id, role, events, can_read, can_create, can_edit, can_delete) VALUES
    ('perm_x12l_member', 'mch_x12_ledger', 'Member', ARRAY['evt_x12_commit'], true, true, true, false),
    ('perm_x12e_member', 'mch_x12_entry',  'Member', ARRAY[]::TEXT[],         true, true, true, false),
    ('perm_x13s_member', 'mch_x13_source', 'Member', ARRAY['evt_x13_log'],    true, true, true, false),
    ('perm_x13g_member', 'mch_x13_log',    'Member', ARRAY[]::TEXT[],         true, true, true, false)
ON CONFLICT (id) DO NOTHING;

INSERT INTO views (id, machine_id, name, type, position, config) VALUES
    ('vw_x12l_form',   'mch_x12_ledger', 'Ledger Form',   'form',   0, '{"fields":["fld_x12l_status"]}'),
    ('vw_x12l_list',   'mch_x12_ledger', 'Ledgers',       'list',   1, '{"columns":["fld_x12l_status"]}'),
    ('vw_x12l_detail', 'mch_x12_ledger', 'Ledger Detail', 'detail', 2, '{}'),

    ('vw_x12e_form',   'mch_x12_entry', 'Entry Form',   'form',   0, '{"fields":["fld_x12e_note"]}'),
    ('vw_x12e_list',   'mch_x12_entry', 'Entries',      'list',   1, '{"columns":["fld_x12e_note"]}'),
    ('vw_x12e_detail', 'mch_x12_entry', 'Entry Detail', 'detail', 2, '{}'),

    ('vw_x13s_form',   'mch_x13_source', 'Source Form',   'form',   0, '{"fields":["fld_x13s_amount"]}'),
    ('vw_x13s_list',   'mch_x13_source', 'Sources',       'list',   1, '{"columns":["fld_x13s_amount"]}'),
    ('vw_x13s_detail', 'mch_x13_source', 'Source Detail', 'detail', 2, '{}'),

    ('vw_x13g_form',   'mch_x13_log', 'Log Form',   'form',   0, '{"fields":["fld_x13g_note"]}'),
    ('vw_x13g_list',   'mch_x13_log', 'Logs',       'list',   1, '{"columns":["fld_x13g_note"]}'),
    ('vw_x13g_detail', 'mch_x13_log', 'Log Detail', 'detail', 2, '{}')
ON CONFLICT (id) DO NOTHING;

-- Account (self-contained, same reasoning as prior lab seed files).
INSERT INTO users (workspace_id, name, email, password_hash, workspace_role) VALUES
    ('ws_default', 'Zara', 'zara@example.com', '$2a$10$moxxOcZzSu3ILTzJlLF2Q.9vxiGNnSXPl7kY1pT3t5o1FoDjqC8aK', 'Admin')
ON CONFLICT (workspace_id, email) DO NOTHING;

INSERT INTO user_application_roles (user_id, application_id, role)
SELECT u.id, 'app_infra_lab', 'Member' FROM users u WHERE u.email = 'zara@example.com'
ON CONFLICT (user_id, application_id) DO UPDATE SET role = EXCLUDED.role;
