-- seeds/013_event_sources_lab.sql
-- Proof harness for Batch 7 (2026-07-12): CAP-E02 (time-driven, "Every Day
-- 08:00"), CAP-E03 (date-driven, "When Due Date - 1 Day"), CAP-E04
-- (external event / webhook).

INSERT INTO applications (id, workspace_id, name) VALUES
    ('app_event_sources', 'ws_default', 'Event Sources Lab')
ON CONFLICT (id) DO NOTHING;

INSERT INTO machines (id, application_id, name, config) VALUES
    ('mch_es_reminder', 'app_event_sources', 'Reminder', NULL),
    ('mch_es_task', 'app_event_sources', 'Scheduled Task', NULL),
    -- CAP-E04: webhook_secret is the ONLY credential handler.Webhook
    -- checks -- no user session, no role. Deliberately a fixed value in a
    -- committed seed file (this is a demo/lab machine, not a real
    -- production integration) -- a real deployment would set this per
    -- Machine to a generated secret, not commit one.
    ('mch_es_payment', 'app_event_sources', 'Payment',
     '{"webhook_secret":"demo-webhook-secret-2026"}')
ON CONFLICT (id) DO NOTHING;

-- Fields
INSERT INTO fields (id, machine_id, name, type, position, required, options) VALUES
    ('fld_esr_title', 'mch_es_reminder', 'Title', 'text',       0, true,  '{}'),
    ('fld_esr_sent',  'mch_es_reminder', 'Sent',  'value_list', 1, false, '{"values":["No","Yes"]}'),

    ('fld_est_title',    'mch_es_task', 'Title',    'text',       0, true,  '{}'),
    ('fld_est_due',      'mch_es_task', 'Due Date', 'date',       1, true,  '{}'),
    ('fld_est_reminded', 'mch_es_task', 'Reminded', 'value_list', 2, false, '{"values":["No","Yes"]}'),

    ('fld_esp_amount',    'mch_es_payment', 'Amount',    'number',     0, true,  '{}'),
    ('fld_esp_status',    'mch_es_payment', 'Status',    'value_list', 1, false, '{"values":["Pending","Paid"]}'),
    ('fld_esp_reference', 'mch_es_payment', 'Reference', 'text',       2, false, '{}')
ON CONFLICT (id) DO NOTHING;

-- Events. CAP-E02/E03's schedule column disambiguates by which key is
-- present (time vs date_field), the same way CAP-A14's condition column
-- already disambiguates aggregate vs ordinary by key (metadata/loader.go).
INSERT INTO events (id, machine_id, name, position, condition, input_fields, schedule) VALUES
    -- CAP-E02: "00:00" (any time of day satisfies >= 00:00) so this fires
    -- on the very first scheduler tick after a Reminder is created,
    -- regardless of what wall-clock time this suite happens to run at --
    -- the same reasoning T66's own date-arithmetic test already used
    -- (deterministic proof, not "only passes before/after a specific
    -- hour").
    ('evt_esr_daily', 'mch_es_reminder', 'Daily Sweep', 0,
     '{"field":"fld_esr_sent","operator":"equals","value":"No"}', ARRAY[]::TEXT[], '{"time":"00:00"}'),
    ('evt_est_remind', 'mch_es_task', 'Send Reminder', 0,
     '{"field":"fld_est_reminded","operator":"equals","value":"No"}', ARRAY[]::TEXT[], '{"date_field":"fld_est_due","offset_days":-1}'),
    -- CAP-E04: no schedule, no condition -- fires only via
    -- POST /webhooks/mch_es_payment/{recordID}/evt_esp_confirm, secret-
    -- authenticated, carrying the payment reference as input.
    ('evt_esp_confirm', 'mch_es_payment', 'Confirm Payment', 0, NULL, ARRAY['fld_esp_reference'], NULL)
ON CONFLICT (id) DO NOTHING;

INSERT INTO event_actions (event_id, type, position, params) VALUES
    ('evt_esr_daily', 'set_field', 0, '{"field":"fld_esr_sent","value":"Yes"}'),
    ('evt_est_remind', 'set_field', 0, '{"field":"fld_est_reminded","value":"Yes"}'),
    ('evt_esp_confirm', 'set_field', 0, '{"field":"fld_esp_status","value":"Paid"}'),
    ('evt_esp_confirm', 'set_field', 1, '{"field":"fld_esp_reference","value":"input:fld_esp_reference"}');

-- Permissions. evt_esr_daily/evt_est_remind never appear in any role's
-- events array -- they're System/scheduler-fired only, never user-
-- triggered (handler.RunScheduledEvents calls triggerEvent directly,
-- bypassing Guard.CanTrigger the same way CAP-A08's aggregate_status
-- cascade already does). evt_esp_confirm likewise fires only via the
-- webhook's own secret, not a role.
INSERT INTO permissions (id, machine_id, role, events) VALUES
    ('perm_esr_member', 'mch_es_reminder', 'Member', ARRAY[]::TEXT[]),
    ('perm_est_member', 'mch_es_task',     'Member', ARRAY[]::TEXT[]),
    ('perm_esp_member', 'mch_es_payment',  'Member', ARRAY[]::TEXT[])
ON CONFLICT (id) DO NOTHING;

-- Views
INSERT INTO views (id, machine_id, name, type, position, config) VALUES
    ('vw_esr_form',   'mch_es_reminder', 'Reminder Form', 'form', 0, '{"fields":["fld_esr_title"]}'),
    ('vw_esr_list',   'mch_es_reminder', 'Reminders',     'list', 1, '{"columns":["fld_esr_title","fld_esr_sent"]}'),
    ('vw_esr_detail', 'mch_es_reminder', 'Reminder Detail', 'detail', 2, '{}'),

    ('vw_est_form',   'mch_es_task', 'Scheduled Task Form', 'form', 0, '{"fields":["fld_est_title","fld_est_due"]}'),
    ('vw_est_list',   'mch_es_task', 'Scheduled Tasks',     'list', 1, '{"columns":["fld_est_title","fld_est_due","fld_est_reminded"]}'),
    ('vw_est_detail', 'mch_es_task', 'Scheduled Task Detail', 'detail', 2, '{}'),

    ('vw_esp_form',   'mch_es_payment', 'Payment Form', 'form', 0, '{"fields":["fld_esp_amount"]}'),
    ('vw_esp_list',   'mch_es_payment', 'Payments',     'list', 1, '{"columns":["fld_esp_amount","fld_esp_status","fld_esp_reference"]}'),
    ('vw_esp_detail', 'mch_es_payment', 'Payment Detail', 'detail', 2, '{}')
ON CONFLICT (id) DO NOTHING;

-- Account (self-contained, same reasoning as prior lab seed files).
INSERT INTO users (workspace_id, name, email, password_hash, workspace_role) VALUES
    ('ws_default', 'Sam', 'sam@example.com', '$2a$10$moxxOcZzSu3ILTzJlLF2Q.9vxiGNnSXPl7kY1pT3t5o1FoDjqC8aK', 'Member')
ON CONFLICT (workspace_id, email) DO NOTHING;

INSERT INTO user_application_roles (user_id, application_id, role)
SELECT u.id, 'app_event_sources', 'Member' FROM users u WHERE u.email = 'sam@example.com'
ON CONFLICT (user_id, application_id) DO UPDATE SET role = EXCLUDED.role;
