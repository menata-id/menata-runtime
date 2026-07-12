-- seeds/014_integration_lab.sql
-- Proof harness for Batch 8 (2026-07-12): CAP-I01 (cross-machine event
-- subscription), CAP-I02 (event schema declaration), CAP-I03 (integration
-- contract), CAP-I05 (cross-cutting contribution). CAP-I04 (correlation
-- trace) shipped earlier, not part of this batch.
--
-- Order and Referral are the PUBLISHERS -- neither one's own metadata
-- names Audit Log or Points Ledger at all (Pattern C's whole point).
-- Audit Log/Points Ledger are the SUBSCRIBERS, declaring interest in
-- events elsewhere. Points Ledger receiving contributions from BOTH Order
-- Placed (large orders only, CAP-I03's own Contract) and Referral
-- Completed is CAP-I05's own proof: one shared KPI machine, fed by two
-- unrelated event sources, decoupled from each one's own definition.

INSERT INTO applications (id, workspace_id, name) VALUES
    ('app_integration_lab', 'ws_default', 'Integration Lab')
ON CONFLICT (id) DO NOTHING;

INSERT INTO machines (id, application_id, name) VALUES
    ('mch_int_order',     'app_integration_lab', 'Order'),
    ('mch_int_referral',  'app_integration_lab', 'Referral'),
    ('mch_int_audit_log', 'app_integration_lab', 'Audit Log'),
    ('mch_int_points',    'app_integration_lab', 'Points Ledger')
ON CONFLICT (id) DO NOTHING;

-- Fields
INSERT INTO fields (id, machine_id, name, type, position, required, options) VALUES
    ('fld_into_customer', 'mch_int_order', 'Customer', 'text',       0, true,  '{}'),
    ('fld_into_total',    'mch_int_order', 'Total',    'number',     1, true,  '{}'),
    ('fld_into_status',   'mch_int_order', 'Status',   'value_list', 2, false, '{"values":["New","Placed"]}'),

    ('fld_intr_referrer', 'mch_int_referral', 'Referrer', 'text',       0, true,  '{}'),
    ('fld_intr_status',   'mch_int_referral', 'Status',   'value_list', 1, false, '{"values":["Pending","Completed"]}'),

    ('fld_ial_message', 'mch_int_audit_log', 'Message', 'text', 0, true,  '{}'),
    ('fld_ial_source',  'mch_int_audit_log', 'Source',  'text', 1, false, '{}'),

    ('fld_ipl_member', 'mch_int_points', 'Member', 'text',   0, true, '{}'),
    ('fld_ipl_points',  'mch_int_points', 'Points', 'number', 1, true, '{}')
ON CONFLICT (id) DO NOTHING;

-- Events. category/schema_version/deprecated_message are CAP-I02's own
-- purely-declarative columns -- evt_into_legacy_notify still functions
-- (backward compat) but logs a warning every time (handler.triggerEvent)
-- and shows a "Deprecated" badge on its own trigger button
-- (internal/ui/detail.templ).
INSERT INTO events (id, machine_id, name, position, condition, category, schema_version, deprecated_message) VALUES
    ('evt_into_placed', 'mch_int_order', 'Place Order', 0, NULL, 'commerce', 'v1', NULL),
    ('evt_into_legacy_notify', 'mch_int_order', 'Legacy Notify', 1, NULL, 'commerce', 'v1',
     'Use Place Order instead -- this event will be removed in a future version.'),
    ('evt_intr_completed', 'mch_int_referral', 'Complete Referral', 0, NULL, 'growth', 'v1', NULL)
ON CONFLICT (id) DO NOTHING;

INSERT INTO event_actions (event_id, type, position, params) VALUES
    ('evt_into_placed', 'set_field', 0, '{"field":"fld_into_status","value":"Placed"}'),
    ('evt_into_legacy_notify', 'set_field', 0, '{"field":"fld_into_status","value":"Placed"}'),
    ('evt_intr_completed', 'set_field', 0, '{"field":"fld_intr_status","value":"Completed"}');

-- Subscriptions (CAP-I01). Audit Log subscribes to Order Placed
-- unconditionally. Points Ledger subscribes to BOTH Order Placed (CAP-I03:
-- gated by a Contract -- only orders >= 100 contribute points, skipped
-- otherwise) and Referral Completed (unconditionally) -- CAP-I05's own
-- proof, two different publishers, one shared machine.
INSERT INTO event_subscriptions (id, machine_id, publisher_event_id, fields, contract, on_violation, position) VALUES
    ('sub_audit_order_placed', 'mch_int_audit_log', 'evt_into_placed',
     '{"fld_ial_message":"Order placed","fld_ial_source":"field:fld_into_customer"}', NULL, 'skip', 0),
    ('sub_points_order_placed', 'mch_int_points', 'evt_into_placed',
     '{"fld_ipl_member":"field:fld_into_customer","fld_ipl_points":"10"}',
     '[{"field":"fld_into_total","operator":"greater_than_or_equal","value":"100"}]', 'skip', 0),
    ('sub_points_referral_completed', 'mch_int_points', 'evt_intr_completed',
     '{"fld_ipl_member":"field:fld_intr_referrer","fld_ipl_points":"25"}', NULL, 'skip', 1)
ON CONFLICT (id) DO NOTHING;

-- Permissions
INSERT INTO permissions (id, machine_id, role, events) VALUES
    ('perm_into_member',  'mch_int_order',     'Member', ARRAY['evt_into_placed','evt_into_legacy_notify']),
    ('perm_intr_member',  'mch_int_referral',  'Member', ARRAY['evt_intr_completed']),
    ('perm_ial_member',   'mch_int_audit_log', 'Member', ARRAY[]::TEXT[]),
    ('perm_ipl_member',   'mch_int_points',    'Member', ARRAY[]::TEXT[])
ON CONFLICT (id) DO NOTHING;

-- Views
INSERT INTO views (id, machine_id, name, type, position, config) VALUES
    ('vw_into_form',   'mch_int_order', 'Order Form', 'form', 0, '{"fields":["fld_into_customer","fld_into_total"]}'),
    ('vw_into_list',   'mch_int_order', 'Orders',     'list', 1, '{"columns":["fld_into_customer","fld_into_total","fld_into_status"]}'),
    ('vw_into_detail', 'mch_int_order', 'Order Detail', 'detail', 2, '{}'),

    ('vw_intr_form',   'mch_int_referral', 'Referral Form', 'form', 0, '{"fields":["fld_intr_referrer"]}'),
    ('vw_intr_list',   'mch_int_referral', 'Referrals',     'list', 1, '{"columns":["fld_intr_referrer","fld_intr_status"]}'),
    ('vw_intr_detail', 'mch_int_referral', 'Referral Detail', 'detail', 2, '{}'),

    ('vw_ial_list',   'mch_int_audit_log', 'Audit Log',        'list', 0, '{"columns":["fld_ial_message","fld_ial_source"]}'),
    ('vw_ial_detail', 'mch_int_audit_log', 'Audit Log Detail', 'detail', 1, '{}'),

    ('vw_ipl_list',   'mch_int_points', 'Points Ledger',        'list', 0, '{"columns":["fld_ipl_member","fld_ipl_points"]}'),
    ('vw_ipl_detail', 'mch_int_points', 'Points Ledger Detail', 'detail', 1, '{}')
ON CONFLICT (id) DO NOTHING;

-- Account (self-contained, same reasoning as prior lab seed files).
INSERT INTO users (workspace_id, name, email, password_hash, workspace_role) VALUES
    ('ws_default', 'Theo', 'theo@example.com', '$2a$10$moxxOcZzSu3ILTzJlLF2Q.9vxiGNnSXPl7kY1pT3t5o1FoDjqC8aK', 'Member')
ON CONFLICT (workspace_id, email) DO NOTHING;

INSERT INTO user_application_roles (user_id, application_id, role)
SELECT u.id, 'app_integration_lab', 'Member' FROM users u WHERE u.email = 'theo@example.com'
ON CONFLICT (user_id, application_id) DO UPDATE SET role = EXCLUDED.role;
