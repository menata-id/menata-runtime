-- seeds/005_complaint.sql
-- Minimal slice of complaint.yaml (Case 7) — proves CAP-E05's trigger_event
-- mechanism, not the full case. Only the events needed to reach Investigating
-- (Triage, Add Investigation Note — both already [SUPPORTED]/[PARTIAL] in
-- complaint.yaml, seeded here verbatim minus the still-unsupported SLA Due
-- Date computation) plus Escalate and a NEW "Run SLA Check" event.
--
-- Run SLA Check is a manual stand-in for complaint.yaml's real
-- evt_cmp_sla_breach_check (schedule-triggered, CAP-E02, still [NOT YET],
-- untouched by this seed) — a Supervisor triggers it by hand instead of a
-- daily cron. Its condition is a single field check (Status = Investigating)
-- via events.condition (CAP-E06), not the real event's compound date+status
-- condition (CAP-A09, still [NOT YET]). Its one action, trigger_event,
-- fires Escalate on this same record — the CAP-E05 mechanism this seed
-- exists to prove.
--
-- Escalate's Assigned To is set to the static literal "Supervisor" here,
-- not the real case's dynamic "role:Supervisor" value token (no such
-- set_field value-resolution exists yet, distinct from CAP-A04's
-- recipient_field which only resolves notify recipients) -- and the
-- Priority raise_one_level() step is omitted entirely rather than faked.
-- Both stay [NOT YET], unchanged from complaint.yaml.
--
-- Delegate, Resolve, Close, Reopen, the Customer role, and every other
-- field/event/constraint/view in complaint.yaml are deliberately unseeded —
-- this is not a full Case 7 implementation. Safe to run multiple times
-- (ON CONFLICT DO NOTHING).

INSERT INTO workspaces (id, name) VALUES
    ('ws_default', 'Default Workspace')
ON CONFLICT (id) DO NOTHING;

INSERT INTO applications (id, workspace_id, name) VALUES
    ('app_customer_service', 'ws_default', 'Customer Service')
ON CONFLICT (id) DO NOTHING;

INSERT INTO machines (id, application_id, name) VALUES
    ('mch_complaint', 'app_customer_service', 'Complaint')
ON CONFLICT (id) DO NOTHING;

-- Fields (reduced set — see header)
INSERT INTO fields (id, machine_id, name, type, position, required, options) VALUES
    ('fld_cmp_complainant_name', 'mch_complaint', 'Complainant Name', 'text',       0, true,  '{}'),
    ('fld_cmp_status',           'mch_complaint', 'Status',           'value_list', 1, false, '{"values":["New","Triaged","Investigating"]}'),
    ('fld_cmp_assigned_to',      'mch_complaint', 'Assigned To',      'user',       2, false, '{}')
ON CONFLICT (id) DO NOTHING;

-- Events
-- evt_cmp_triage/evt_cmp_add_investigation_note: complaint.yaml's own
-- Status-transition actions, verbatim (their SLA Due Date derivation is
-- skipped, per header note).
-- evt_cmp_run_sla_check: new, this seed's own -- gated by events.condition
-- (Status = Investigating), fires trigger_event on Escalate.
INSERT INTO events (id, machine_id, name, position, condition) VALUES
    ('evt_cmp_triage',                  'mch_complaint', 'Triage',                  0, NULL),
    ('evt_cmp_add_investigation_note',  'mch_complaint', 'Add Investigation Note',  1, NULL),
    ('evt_cmp_escalate',                'mch_complaint', 'Escalate',                2, NULL),
    ('evt_cmp_run_sla_check',           'mch_complaint', 'Run SLA Check',           3,
     '{"field":"fld_cmp_status","operator":"equals","value":"Investigating"}')
ON CONFLICT (id) DO NOTHING;

INSERT INTO event_actions (event_id, type, position, params) VALUES
    ('evt_cmp_triage',                 'set_field',     0, '{"field":"fld_cmp_status","value":"Triaged"}'),
    ('evt_cmp_add_investigation_note', 'set_field',     0, '{"field":"fld_cmp_status","value":"Investigating"}'),
    ('evt_cmp_escalate',               'set_field',     0, '{"field":"fld_cmp_assigned_to","value":"Supervisor"}'),
    ('evt_cmp_run_sla_check',          'trigger_event', 0, '{"event":"evt_cmp_escalate"}');

-- Constraints
INSERT INTO constraints (id, machine_id, rule, expression, condition, position) VALUES
    ('cst_cmp_complainant_required',
     'mch_complaint',
     'Complainant Name is required.',
     '{"field":"fld_cmp_complainant_name","operator":"required"}',
     NULL, 0)
ON CONFLICT (id) DO NOTHING;

-- Permissions
-- Agent: logs complaints on a customer's behalf (Create) and works the
-- Triage/Investigation steps this slice covers. can_create/can_read default
-- true (migrations/006, CAP-P05).
-- Supervisor: Escalate + the new Run SLA Check, read-only otherwise.
INSERT INTO permissions (id, machine_id, role, events) VALUES
    ('perm_cmp_agent',      'mch_complaint', 'Agent',      ARRAY['evt_cmp_triage','evt_cmp_add_investigation_note']),
    ('perm_cmp_supervisor', 'mch_complaint', 'Supervisor', ARRAY['evt_cmp_escalate','evt_cmp_run_sla_check'])
ON CONFLICT (id) DO NOTHING;
UPDATE permissions SET can_create = false WHERE id = 'perm_cmp_supervisor';

-- Views
INSERT INTO views (id, machine_id, name, type, position, config) VALUES
    ('vw_cmp_form',
     'mch_complaint', 'Complaint Form', 'form', 0,
     '{"fields":["fld_cmp_complainant_name"]}'),
    ('vw_cmp_list',
     'mch_complaint', 'Complaints', 'list', 1,
     '{"columns":["fld_cmp_complainant_name","fld_cmp_status","fld_cmp_assigned_to"]}'),
    ('vw_cmp_detail',
     'mch_complaint', 'Complaint Detail', 'detail', 2, '{}')
ON CONFLICT (id) DO NOTHING;
