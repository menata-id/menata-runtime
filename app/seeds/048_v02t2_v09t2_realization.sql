-- seeds/048_v02t2_v09t2_realization.sql
-- Realizes the two capabilities admitted 2026-09-09 (roadmap.md item 25.3,
-- capability-registry.md's own CAP-V02 Tier 2 / CAP-V09 Tier 2 rows):
--
-- 1. CAP-V02 Tier 2 (card-per-record list display) -- Approval Document's
--    own "All Documents" list adopts `display: "cards"`, the real Case 3
--    consumer the admission note itself named. Same columns/filter/sort
--    underneath (vw_ad_all's own config, seeds/004); only the rendering
--    mode changes.
--
-- 2. CAP-V09 Tier 2 (computed-SLA-bucket list filter) -- a NEW, self-
--    contained lab (own Machine), not a second View bolted onto the
--    existing SLA Badge Lab (mch_sla_ticket, seeds/029): Interpreter.
--    DefaultListView (internal/interpreter/interpreter.go) resolves only
--    the FIRST `list`-type View per Machine -- "a Machine declaring more
--    than one View of the same auxiliary type is a case this prototype
--    doesn't need yet" (that function's own doc comment). A second `list`
--    View added to mch_sla_ticket would be real, valid, load-time-checked
--    metadata with NO route ever reaching it -- dead weight, not a proof.
--    A dedicated Machine sidesteps that limit entirely, same reasoning
--    seeds/029 itself already used (a new self-contained lab rather than
--    reusing an older one with a conflicting shape). Also: Approval
--    Document itself has no due-date Field at all (no CAP-V17 SlaField
--    declared anywhere on it) -- there is nothing for $sla_urgency to
--    compute from there yet; adding one would be unrelated scope creep,
--    named honestly rather than silently faked.

UPDATE views SET config = config || '{"display":"cards"}' WHERE id = 'vw_ad_all';

INSERT INTO applications (id, workspace_id, name) VALUES
    ('app_sla_filter_lab', 'ws_default', 'SLA Filter Lab')
ON CONFLICT (id) DO NOTHING;

INSERT INTO machines (id, application_id, name) VALUES
    ('mch_v09t2_ticket', 'app_sla_filter_lab', 'Ticket')
ON CONFLICT (id) DO NOTHING;

INSERT INTO fields (id, machine_id, name, type, position, required, options) VALUES
    ('fld_v09t2_title', 'mch_v09t2_ticket', 'Title', 'text', 0, true, '{}'),
    ('fld_v09t2_due',   'mch_v09t2_ticket', 'Due',   'date', 1, false, '{}')
ON CONFLICT (id) DO NOTHING;

INSERT INTO permissions (id, machine_id, role, events, can_read, can_create, can_edit, can_delete) VALUES
    ('perm_v09t2_agent', 'mch_v09t2_ticket', 'Agent', ARRAY[]::TEXT[], true, true, true, false)
ON CONFLICT (id) DO NOTHING;

-- The list View IS the overdue filter from the start -- no second List
-- View needed (see the file header above for why that would be
-- unreachable), so this lab proves the filter directly: only records
-- whose fld_v09t2_due computes to the "overdue" bucket ever appear here.
INSERT INTO views (id, machine_id, name, type, position, config) VALUES
    ('vw_v09t2_overdue', 'mch_v09t2_ticket', 'Overdue Tickets', 'list', 0,
     '{"columns":["fld_v09t2_title","fld_v09t2_due"],"sla_field":"fld_v09t2_due","sla_warning_days":3,"filter":[{"field":"$sla_urgency","operator":"equals","value":"overdue"}]}'),
    ('vw_v09t2_form', 'mch_v09t2_ticket', 'New Ticket', 'form', 1,
     '{"fields":["fld_v09t2_title","fld_v09t2_due"]}')
ON CONFLICT (id) DO NOTHING;

INSERT INTO users (workspace_id, name, email, password_hash, workspace_role) VALUES
    ('ws_default', 'SLA Filter Agent', 'sla.filter.agent@example.com',
     '$2a$10$moxxOcZzSu3ILTzJlLF2Q.9vxiGNnSXPl7kY1pT3t5o1FoDjqC8aK', 'Member')
ON CONFLICT (email) DO NOTHING;

-- CAP-O11: seeds/040_membership_backfill.sql runs LAST (Makefile's own seed
-- ordering) and only backfills workspace_memberships for users that already
-- existed when IT ran -- a numerically later seed (048 > 040) creating a
-- brand new user must insert its own membership row explicitly, the same
-- pattern seeds/039/041/046 already established, or login 403s with "no
-- workspace membership" (caught live running this file's own conformance
-- test before this fix).
INSERT INTO workspace_memberships (user_id, workspace_id, workspace_role)
SELECT id, 'ws_default', 'Member' FROM users WHERE email = 'sla.filter.agent@example.com'
ON CONFLICT (user_id, workspace_id) DO NOTHING;

INSERT INTO user_application_roles (user_id, application_id, role)
SELECT u.id, 'app_sla_filter_lab', 'Agent' FROM users u WHERE u.email = 'sla.filter.agent@example.com'
ON CONFLICT (user_id, application_id) DO UPDATE SET role = EXCLUDED.role;
