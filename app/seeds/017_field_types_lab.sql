-- seeds/017_field_types_lab.sql
-- Proof harness for the final batch: the 11 remaining field types
-- (CAP-F06/F07/F08/F09/F10/F14/F15/F17/F18/F19/F21). CAP-F17 (multi-currency
-- money) and CAP-F19 (quantity/UoM) are proven by COMPOSITION -- no new
-- code beyond CAP-F08/F14/F07/value_list, matching the registry's own
-- framing of both as "compose, don't add a mechanism".

INSERT INTO applications (id, workspace_id, name) VALUES
    ('app_field_types_lab', 'ws_default', 'Field Types Lab')
ON CONFLICT (id) DO NOTHING;

INSERT INTO machines (id, application_id, name, config) VALUES
    ('mch_ft_product',  'app_field_types_lab', 'Product',  NULL),
    ('mch_ft_invoice',  'app_field_types_lab', 'Invoice',  NULL),
    ('mch_ft_shipment', 'app_field_types_lab', 'Shipment', NULL)
ON CONFLICT (id) DO NOTHING;

-- mch_ft_product: CAP-F06 (file+image pipeline), F07 (number), F08 (money),
-- F09 (boolean), F10 (time/date_time/duration), F14 (computed: total =
-- price * qty), F15 (default on a non-value_list field), F18 (auto-number).
INSERT INTO fields (id, machine_id, name, type, position, required, options) VALUES
    ('fld_ftp_sku', 'mch_ft_product', 'SKU', 'text', 0, false,
        '{"auto_number_prefix":"SKU-","auto_number_padding":4}'),
    ('fld_ftp_title', 'mch_ft_product', 'Title', 'text', 1, true, '{}'),
    ('fld_ftp_priority', 'mch_ft_product', 'Priority', 'text', 2, false,
        '{"default":"Normal"}'),
    ('fld_ftp_price', 'mch_ft_product', 'Price', 'money', 3, false,
        '{"currency":"IDR"}'),
    ('fld_ftp_qty', 'mch_ft_product', 'Quantity', 'number', 4, false, '{}'),
    ('fld_ftp_total', 'mch_ft_product', 'Total', 'computed', 5, false,
        '{"source_field":"fld_ftp_price","factor_field":"fld_ftp_qty"}'),
    ('fld_ftp_active', 'mch_ft_product', 'Active', 'boolean', 6, false, '{}'),
    ('fld_ftp_release_time', 'mch_ft_product', 'Release Time', 'time', 7, false, '{}'),
    ('fld_ftp_launched_at', 'mch_ft_product', 'Launched At', 'date_time', 8, false, '{}'),
    ('fld_ftp_prep_duration', 'mch_ft_product', 'Prep Duration', 'duration', 9, false, '{}'),
    ('fld_ftp_photo', 'mch_ft_product', 'Photo', 'file', 10, false,
        '{"accept":"image/*","compress":true,"max_dimension":200,"format":"webp"}')
ON CONFLICT (id) DO NOTHING;

-- mch_ft_invoice: CAP-F17 (multi-currency money via reference sugar
-- composition -- money's own currency_field option + a computed base
-- mirror, rather than a dedicated new field type).
INSERT INTO fields (id, machine_id, name, type, position, required, options) VALUES
    ('fld_fti_currency', 'mch_ft_invoice', 'Currency', 'value_list', 0, true,
        '{"values":["IDR","USD","EUR"]}'),
    ('fld_fti_amount', 'mch_ft_invoice', 'Amount', 'money', 1, false,
        '{"currency_field":"fld_fti_currency"}'),
    ('fld_fti_rate', 'mch_ft_invoice', 'Rate to IDR', 'number', 2, false, '{}'),
    ('fld_fti_base_amount', 'mch_ft_invoice', 'Base Amount (IDR)', 'computed', 3, false,
        '{"source_field":"fld_fti_amount","factor_field":"fld_fti_rate"}')
ON CONFLICT (id) DO NOTHING;

-- mch_ft_shipment: CAP-F19 (Quantity/UoM Tier 1 -- flat factor pair,
-- composed from number + value_list + computed, no new field type).
INSERT INTO fields (id, machine_id, name, type, position, required, options) VALUES
    ('fld_fts_quantity', 'mch_ft_shipment', 'Quantity', 'number', 0, false, '{}'),
    ('fld_fts_unit', 'mch_ft_shipment', 'Unit', 'value_list', 1, false,
        '{"values":["Kg","G"]}'),
    ('fld_fts_factor', 'mch_ft_shipment', 'Factor to Grams', 'number', 2, false, '{}'),
    ('fld_fts_base_qty', 'mch_ft_shipment', 'Quantity (Grams)', 'computed', 3, false,
        '{"source_field":"fld_fts_quantity","factor_field":"fld_fts_factor"}')
ON CONFLICT (id) DO NOTHING;

INSERT INTO permissions (id, machine_id, role, events, can_read, can_create, can_edit, can_delete) VALUES
    ('perm_ftp_member', 'mch_ft_product',  'Member', ARRAY[]::TEXT[], true, true, true, false),
    ('perm_fti_member', 'mch_ft_invoice',  'Member', ARRAY[]::TEXT[], true, true, true, false),
    ('perm_fts_member', 'mch_ft_shipment', 'Member', ARRAY[]::TEXT[], true, true, true, false)
ON CONFLICT (id) DO NOTHING;

INSERT INTO views (id, machine_id, name, type, position, config) VALUES
    ('vw_ftp_form', 'mch_ft_product', 'Product Form', 'form', 0,
        '{"fields":["fld_ftp_title","fld_ftp_priority","fld_ftp_price","fld_ftp_qty","fld_ftp_active","fld_ftp_release_time","fld_ftp_launched_at","fld_ftp_prep_duration","fld_ftp_photo"]}'),
    ('vw_ftp_list', 'mch_ft_product', 'Products', 'list', 1,
        '{"columns":["fld_ftp_sku","fld_ftp_title","fld_ftp_price","fld_ftp_total","fld_ftp_active"]}'),
    ('vw_ftp_detail', 'mch_ft_product', 'Product Detail', 'detail', 2, '{}'),
    -- CAP-F21: a `document` View -- html/template source, {{.fld_x}}
    -- placeholders resolved against one record's own Data at render time.
    ('vw_ftp_document', 'mch_ft_product', 'Product Certificate', 'document', 3,
        '{"template":"<html><body><h1>Certificate</h1><p>SKU: {{.fld_ftp_sku}}</p><p>Title: {{.fld_ftp_title}}</p></body></html>"}'),

    ('vw_fti_form', 'mch_ft_invoice', 'Invoice Form', 'form', 0,
        '{"fields":["fld_fti_currency","fld_fti_amount","fld_fti_rate"]}'),
    ('vw_fti_list', 'mch_ft_invoice', 'Invoices', 'list', 1,
        '{"columns":["fld_fti_currency","fld_fti_amount","fld_fti_base_amount"]}'),
    ('vw_fti_detail', 'mch_ft_invoice', 'Invoice Detail', 'detail', 2, '{}'),

    ('vw_fts_form', 'mch_ft_shipment', 'Shipment Form', 'form', 0,
        '{"fields":["fld_fts_quantity","fld_fts_unit","fld_fts_factor"]}'),
    ('vw_fts_list', 'mch_ft_shipment', 'Shipments', 'list', 1,
        '{"columns":["fld_fts_quantity","fld_fts_unit","fld_fts_base_qty"]}'),
    ('vw_fts_detail', 'mch_ft_shipment', 'Shipment Detail', 'detail', 2, '{}')
ON CONFLICT (id) DO NOTHING;

-- Account (self-contained, same reasoning as prior lab seed files).
INSERT INTO users (workspace_id, name, email, password_hash, workspace_role) VALUES
    ('ws_default', 'Wira', 'wira@example.com', '$2a$10$moxxOcZzSu3ILTzJlLF2Q.9vxiGNnSXPl7kY1pT3t5o1FoDjqC8aK', 'Member')
ON CONFLICT (workspace_id, email) DO NOTHING;

INSERT INTO user_application_roles (user_id, application_id, role)
SELECT u.id, 'app_field_types_lab', 'Member' FROM users u WHERE u.email = 'wira@example.com'
ON CONFLICT (user_id, application_id) DO UPDATE SET role = EXCLUDED.role;
