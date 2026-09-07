-- seeds/041_expression_lab.sql
-- CAP-C13 (expression operator, internal/expr, CEL) + CAP-F14 completion
-- (computed Field via a general expression, not just source*factor).
-- Study 37 item 2 (roadmap.md's own "Recommended order").
--
-- A NEW, self-contained lab -- same "new Machine, not a retrofit" ratchet
-- discipline every prior lab file states.
--
-- fld_eo_total (CAP-F14 completion) proves real arithmetic beyond one
-- multiply: quantity * unit_price * (1 - discount_pct/100).
--
-- cst_eo_no_draft_to_closed (CAP-C13, Constraint) proves `old` vs `record`:
-- a business rule ("must be Approved before Closed") expressed ONCE at the
-- Machine level, catching a direct Draft->Closed transition regardless of
-- WHICH event tried it -- evt_eo_close itself declares no "must come from
-- Approved" condition of its own, the Constraint is what actually blocks
-- it.
--
-- evt_eo_approve's own condition (CAP-C13, Event) proves real arithmetic in
-- a guard the old field/operator/value grammar could never express in one
-- clause: quantity * unit_price > 100.
--
-- vw_eo_list's own filter (CAP-C13, View) proves the same "expression"
-- operator flowing into CAP-V05/V09's declarative row filter for free, via
-- the one shared ConstraintExpression/FilterCondition.Expression field --
-- no new filtering mechanism. It's the Machine's only list-type View
-- (Interpreter.DefaultListView always picks the first "list" View and this
-- runtime has no per-View-instance route to pick a different one), so the
-- filter applies directly to plain GET /mch_expr_order, not a second URL.

INSERT INTO workspaces (id, name, slug) VALUES ('ws_default', 'Default Workspace', 'ws_default')
ON CONFLICT (id) DO NOTHING;

INSERT INTO applications (id, workspace_id, name) VALUES
    ('app_expression_lab', 'ws_default', 'Expression Lab')
ON CONFLICT (id) DO NOTHING;

INSERT INTO machines (id, application_id, name) VALUES
    ('mch_expr_order', 'app_expression_lab', 'Order')
ON CONFLICT (id) DO NOTHING;

INSERT INTO fields (id, machine_id, name, type, position, required, options) VALUES
    ('fld_eo_customer',     'mch_expr_order', 'Customer',      'text',       0, true,  '{}'),
    ('fld_eo_quantity',     'mch_expr_order', 'Quantity',      'number',     1, true,  '{}'),
    ('fld_eo_unit_price',   'mch_expr_order', 'Unit Price',    'number',     2, true,  '{}'),
    ('fld_eo_discount_pct', 'mch_expr_order', 'Discount %',    'number',     3, false, '{"default":"0"}'),
    ('fld_eo_status',       'mch_expr_order', 'Status',        'value_list', 4, false, '{"values":["Draft","Approved","Closed"]}'),
    ('fld_eo_total',        'mch_expr_order', 'Total',         'computed',   5, false,
        '{"expression":"double(record.fld_eo_quantity) * double(record.fld_eo_unit_price) * (1.0 - double(record.fld_eo_discount_pct) / 100.0)"}')
ON CONFLICT (id) DO NOTHING;

INSERT INTO constraints (id, machine_id, rule, expression, condition, position) VALUES
    ('cst_eo_no_draft_to_closed',
     'mch_expr_order',
     'An order cannot move directly from Draft to Closed -- it must be Approved first.',
     '{"operator":"expression","expression":"!(old.fld_eo_status == \"Draft\" && record.fld_eo_status == \"Closed\")"}',
     NULL, 0)
ON CONFLICT (id) DO NOTHING;

INSERT INTO events (id, machine_id, name, position, condition) VALUES
    ('evt_eo_approve', 'mch_expr_order', 'Approve', 0,
        '{"operator":"expression","expression":"double(record.fld_eo_quantity) * double(record.fld_eo_unit_price) > 100.0"}'),
    ('evt_eo_close',   'mch_expr_order', 'Close',   1, NULL)
ON CONFLICT (id) DO NOTHING;

INSERT INTO event_actions (event_id, type, position, params) VALUES
    ('evt_eo_approve', 'set_field', 0, '{"field":"fld_eo_status","value":"Approved"}'),
    ('evt_eo_close',   'set_field', 0, '{"field":"fld_eo_status","value":"Closed"}')
ON CONFLICT DO NOTHING;

INSERT INTO permissions (id, machine_id, role, events, can_read, can_create, can_edit, can_delete) VALUES
    ('perm_eo_sales',   'mch_expr_order', 'Sales',   ARRAY[]::TEXT[],                          true, true,  true,  false),
    ('perm_eo_manager', 'mch_expr_order', 'Manager', ARRAY['evt_eo_approve', 'evt_eo_close'], true, false, false, false)
ON CONFLICT (id) DO NOTHING;

INSERT INTO views (id, machine_id, name, type, position, config) VALUES
    ('vw_eo_form',   'mch_expr_order', 'New Order',  'form',   0,
        '{"fields":["fld_eo_customer","fld_eo_quantity","fld_eo_unit_price","fld_eo_discount_pct"]}'),
    ('vw_eo_list',   'mch_expr_order', 'Big Orders', 'list',   1,
        '{"columns":["fld_eo_customer","fld_eo_status","fld_eo_total"],"filter":[{"operator":"expression","expression":"double(record.fld_eo_quantity) * double(record.fld_eo_unit_price) > 500.0"}]}'),
    ('vw_eo_detail', 'mch_expr_order', 'Order Detail', 'detail', 2, '{}')
ON CONFLICT (id) DO NOTHING;

INSERT INTO users (workspace_id, name, email, password_hash, workspace_role) VALUES
    ('ws_default', 'Expression Lab Sales',   'expr.sales@example.com',
     '$2a$10$moxxOcZzSu3ILTzJlLF2Q.9vxiGNnSXPl7kY1pT3t5o1FoDjqC8aK', 'Member'),
    ('ws_default', 'Expression Lab Manager', 'expr.manager@example.com',
     '$2a$10$moxxOcZzSu3ILTzJlLF2Q.9vxiGNnSXPl7kY1pT3t5o1FoDjqC8aK', 'Member')
ON CONFLICT (email) DO NOTHING;

INSERT INTO workspace_memberships (user_id, workspace_id, workspace_role)
SELECT id, 'ws_default', 'Member' FROM users WHERE email IN ('expr.sales@example.com', 'expr.manager@example.com')
ON CONFLICT (user_id, workspace_id) DO NOTHING;

INSERT INTO user_application_roles (user_id, application_id, role)
SELECT u.id, 'app_expression_lab',
       CASE u.email WHEN 'expr.sales@example.com' THEN 'Sales' ELSE 'Manager' END
FROM users u WHERE u.email IN ('expr.sales@example.com', 'expr.manager@example.com')
ON CONFLICT (user_id, application_id) DO UPDATE SET role = EXCLUDED.role;
