-- seeds/031_typeahead_lab.sql
-- CAP-V16 proof: a reference field's candidate pool exceeding
-- handler.typeaheadThreshold (25) switches from an eager <select> to
-- TypeaheadPicker (HTMX search-as-you-type, GET /{machineID}/field-options).
-- 30 pre-seeded Product records so the threshold is crossed without a slow
-- 30-request boot-time seed.

INSERT INTO workspaces (id, name) VALUES ('ws_default', 'Default Workspace')
ON CONFLICT (id) DO NOTHING;

INSERT INTO applications (id, workspace_id, name) VALUES
    ('app_typeahead_lab', 'ws_default', 'Typeahead Lab')
ON CONFLICT (id) DO NOTHING;

INSERT INTO machines (id, application_id, name) VALUES
    ('mch_v16_product', 'app_typeahead_lab', 'Product'),
    ('mch_v16_order',   'app_typeahead_lab', 'Order')
ON CONFLICT (id) DO NOTHING;

INSERT INTO fields (id, machine_id, name, type, position, required, options) VALUES
    ('fld_v16p_name', 'mch_v16_product', 'Name', 'text', 0, true, '{}'),

    ('fld_v16o_title',   'mch_v16_order', 'Title',   'text', 0, true, '{}'),
    ('fld_v16o_product', 'mch_v16_order', 'Product', 'reference', 1, true, '{"target_machine":"mch_v16_product"}')
ON CONFLICT (id) DO NOTHING;

INSERT INTO permissions (id, machine_id, role, events, can_read, can_create, can_edit, can_delete) VALUES
    ('perm_v16p_clerk', 'mch_v16_product', 'Clerk', ARRAY[]::TEXT[], true, true, true, false),
    ('perm_v16o_clerk', 'mch_v16_order',   'Clerk', ARRAY[]::TEXT[], true, true, true, false)
ON CONFLICT (id) DO NOTHING;

INSERT INTO views (id, machine_id, name, type, position, config) VALUES
    ('vw_v16p_list', 'mch_v16_product', 'Products', 'list', 0, '{"columns":["fld_v16p_name"]}'),
    ('vw_v16o_form', 'mch_v16_order', 'New Order', 'form', 0, '{"fields":["fld_v16o_title","fld_v16o_product"]}'),
    ('vw_v16o_list', 'mch_v16_order', 'Orders', 'list', 1, '{"columns":["fld_v16o_title"]}')
ON CONFLICT (id) DO NOTHING;

INSERT INTO users (workspace_id, name, email, password_hash, workspace_role) VALUES
    ('ws_default', 'Typeahead Clerk', 'typeahead.clerk@example.com',
     '$2a$10$moxxOcZzSu3ILTzJlLF2Q.9vxiGNnSXPl7kY1pT3t5o1FoDjqC8aK', 'Member')
ON CONFLICT (workspace_id, email) DO NOTHING;

INSERT INTO user_application_roles (user_id, application_id, role)
SELECT u.id, 'app_typeahead_lab', 'Clerk' FROM users u WHERE u.email = 'typeahead.clerk@example.com'
ON CONFLICT (user_id, application_id) DO UPDATE SET role = EXCLUDED.role;

-- 30 products, well past the 25-record eager-<select> threshold. RLS
-- (migrations/009) forces every non-superuser connection to set
-- app.workspace_id before touching `records` -- same pattern
-- seeds/024_change_policy_lab.sql already established.
SET app.workspace_id = 'ws_default';

INSERT INTO records (id, machine_id, workspace_id, data, created_at, updated_at)
SELECT
    ('11111111-2222-3333-4444-' || lpad(n::text, 12, '0'))::uuid,
    'mch_v16_product',
    'ws_default',
    jsonb_build_object('fld_v16p_name', 'Widget ' || lpad(n::text, 3, '0')),
    NOW(), NOW()
FROM generate_series(1, 30) AS n
ON CONFLICT (id) DO NOTHING;
