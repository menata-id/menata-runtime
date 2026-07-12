-- seeds/011_record_lifecycle_lab.sql
-- Proof harness for Batch 5 (2026-07-12): the Record Lifecycle cluster --
-- CAP-R03 (archive/restore), CAP-R05 (pagination), CAP-R06 (CSV
-- import/export), CAP-R07 (immutability after state), CAP-R08 (editable
-- scratch state). CAP-R04 (audit log) already shipped separately, not part
-- of this batch. Four machines, one Application -- each proof needs its
-- own Machine.Config shape (immutable_field/scratch_field are mutually
-- distinct settings), so one shared machine can't carry all five cases.
-- Business data (actual records, e.g. the 25+ Tickets CAP-R05's pagination
-- needs) is deliberately NOT seeded here -- every prior seed file only
-- declares metadata, never `records` rows (no natural key to guard a
-- re-run with ON CONFLICT); conformance/run.sh creates what it needs via
-- real HTTP POSTs, same as every other capability's proof data.

INSERT INTO applications (id, workspace_id, name) VALUES
    ('app_record_lifecycle', 'ws_default', 'Record Lifecycle Lab')
ON CONFLICT (id) DO NOTHING;

INSERT INTO machines (id, application_id, name, config) VALUES
    ('mch_rl_ticket', 'app_record_lifecycle', 'Ticket', NULL),
    ('mch_rl_document', 'app_record_lifecycle', 'Document', NULL),
    -- CAP-R07: frozen once fld_rlle_status is Posted -- guards Update AND
    -- Archive, every mutation path, not just CAP-E06's own event guard.
    ('mch_rl_ledger_entry', 'app_record_lifecycle', 'Ledger Entry',
     '{"immutable_field":"fld_rlle_status","immutable_values":"Posted"}'),
    -- CAP-R08: none of this Machine's Constraints apply while
    -- fld_rlc_status is Cart -- CAP-C09's own trigger-time re-check is the
    -- commit-point enforcement once Checkout moves it to Ordered.
    ('mch_rl_cart', 'app_record_lifecycle', 'Cart',
     '{"scratch_field":"fld_rlc_status","scratch_values":"Cart"}')
ON CONFLICT (id) DO NOTHING;

-- Fields
INSERT INTO fields (id, machine_id, name, type, position, required, options) VALUES
    ('fld_rlt_title', 'mch_rl_ticket', 'Title', 'text', 0, true, '{}'),

    ('fld_rld_title',  'mch_rl_document', 'Title',  'text',   0, true,  '{}'),
    ('fld_rld_amount', 'mch_rl_document', 'Amount', 'number', 1, false, '{}'),

    ('fld_rlle_memo',   'mch_rl_ledger_entry', 'Memo',   'text',       0, true, '{}'),
    ('fld_rlle_status', 'mch_rl_ledger_entry', 'Status', 'value_list', 1, false, '{"values":["Draft","Posted"]}'),

    ('fld_rlc_item',     'mch_rl_cart', 'Item',     'text',       0, true,  '{}'),
    ('fld_rlc_quantity', 'mch_rl_cart', 'Quantity', 'number',     1, true,  '{}'),
    ('fld_rlc_status',   'mch_rl_cart', 'Status',   'value_list', 2, false, '{"values":["Cart","Ordered"]}')
ON CONFLICT (id) DO NOTHING;

-- Constraints. mch_rl_document's is CAP-R06's own proof case (a required
-- Title, so an imported CSV row with it blank has something real to
-- reject) -- fields.required alone is informational in this runtime; only
-- an explicit Constraint row is actually enforced (constraint.Engine reads
-- machine.Constraints, never fields.required directly). mch_rl_cart's two
-- are CAP-R08's own proof: they apply once fld_rlc_status leaves Cart, not
-- before -- see mch_rl_cart's own Config above.
INSERT INTO constraints (id, machine_id, rule, expression, condition, position) VALUES
    ('cst_rld_title_required', 'mch_rl_document', 'Title is required',
     '{"field":"fld_rld_title","operator":"required"}', NULL, 0),
    ('cst_rlc_item_required', 'mch_rl_cart', 'Item is required',
     '{"field":"fld_rlc_item","operator":"required"}', NULL, 0),
    ('cst_rlc_qty_positive', 'mch_rl_cart', 'Quantity must be positive',
     '{"field":"fld_rlc_quantity","operator":"greater_than","value":"0"}', NULL, 1)
ON CONFLICT (id) DO NOTHING;

-- Events
INSERT INTO events (id, machine_id, name, position, condition) VALUES
    ('evt_rlle_post', 'mch_rl_ledger_entry', 'Post', 0, NULL),
    ('evt_rlc_checkout', 'mch_rl_cart', 'Checkout', 0, NULL)
ON CONFLICT (id) DO NOTHING;

INSERT INTO event_actions (event_id, type, position, params) VALUES
    ('evt_rlle_post', 'set_field', 0, '{"field":"fld_rlle_status","value":"Posted"}'),
    ('evt_rlc_checkout', 'set_field', 0, '{"field":"fld_rlc_status","value":"Ordered"}');

-- Permissions. can_delete is explicit true here (migrations/012 defaults
-- it false) -- Ticket/Ledger Entry both need CAP-R03 proof.
INSERT INTO permissions (id, machine_id, role, events, can_read, can_create, can_edit, can_delete) VALUES
    ('perm_rlt_member',   'mch_rl_ticket',       'Member', ARRAY[]::TEXT[],            true, true, true, true),
    ('perm_rld_member',   'mch_rl_document',     'Member', ARRAY[]::TEXT[],            true, true, true, false),
    ('perm_rlle_member',  'mch_rl_ledger_entry', 'Member', ARRAY['evt_rlle_post'],     true, true, true, true),
    ('perm_rlc_member',   'mch_rl_cart',         'Member', ARRAY['evt_rlc_checkout'],  true, true, true, false)
ON CONFLICT (id) DO NOTHING;

-- Views
INSERT INTO views (id, machine_id, name, type, position, config) VALUES
    ('vw_rlt_form',   'mch_rl_ticket', 'Ticket Form', 'form', 0, '{"fields":["fld_rlt_title"]}'),
    ('vw_rlt_list',   'mch_rl_ticket', 'Tickets',     'list', 1, '{"columns":["fld_rlt_title"]}'),
    ('vw_rlt_detail', 'mch_rl_ticket', 'Ticket Detail', 'detail', 2, '{}'),

    ('vw_rld_form',   'mch_rl_document', 'Document Form', 'form', 0, '{"fields":["fld_rld_title","fld_rld_amount"]}'),
    ('vw_rld_list',   'mch_rl_document', 'Documents',     'list', 1, '{"columns":["fld_rld_title","fld_rld_amount"]}'),
    ('vw_rld_detail', 'mch_rl_document', 'Document Detail', 'detail', 2, '{}'),

    ('vw_rlle_form',   'mch_rl_ledger_entry', 'Ledger Entry Form', 'form', 0, '{"fields":["fld_rlle_memo"]}'),
    ('vw_rlle_list',   'mch_rl_ledger_entry', 'Ledger Entries',    'list', 1, '{"columns":["fld_rlle_memo","fld_rlle_status"]}'),
    ('vw_rlle_detail', 'mch_rl_ledger_entry', 'Ledger Entry Detail', 'detail', 2, '{}'),

    ('vw_rlc_form',   'mch_rl_cart', 'Cart Form', 'form', 0, '{"fields":["fld_rlc_item","fld_rlc_quantity"]}'),
    ('vw_rlc_list',   'mch_rl_cart', 'Carts',     'list', 1, '{"columns":["fld_rlc_item","fld_rlc_quantity","fld_rlc_status"]}'),
    ('vw_rlc_detail', 'mch_rl_cart', 'Cart Detail', 'detail', 2, '{}')
ON CONFLICT (id) DO NOTHING;

-- Account (self-contained, same reasoning as prior lab seed files).
INSERT INTO users (workspace_id, name, email, password_hash, workspace_role) VALUES
    ('ws_default', 'Rex', 'rex@example.com',
     '$2a$10$moxxOcZzSu3ILTzJlLF2Q.9vxiGNnSXPl7kY1pT3t5o1FoDjqC8aK', 'Member')
ON CONFLICT (workspace_id, email) DO NOTHING;

INSERT INTO user_application_roles (user_id, application_id, role)
SELECT u.id, 'app_record_lifecycle', 'Member' FROM users u WHERE u.email = 'rex@example.com'
ON CONFLICT (user_id, application_id) DO UPDATE SET role = EXCLUDED.role;
