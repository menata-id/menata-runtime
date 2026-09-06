-- seeds/021_sla_lab.sql
-- Process Overlay B4 proof, Part 1 (Study 21 addendum, CAP-W04): a single
-- `sla` declaration compiles to a due-date Field, a due-date-stamping action
-- on every transition landing on the SLA-bound state, and a scheduled
-- breach Event -- entirely within ONE Machine (internal/metadata/
-- compile.go's compileSLA), no cross-machine wiring.
--
-- A NEW, self-contained lab (new Application, new Machine) rather than an
-- edit to 019/020 -- this codebase's own "ratchet rule — new Machines, not
-- a retrofit" (same reasoning as seeds/020's own header).
--
-- duration "0 Days" is a deliberate fast-test trick (due = today, so the
-- very next scheduler tick already sees it overdue) -- the same shape
-- T99/T100's own real-scheduler wait already relies on, no multi-day wait
-- needed.

INSERT INTO workspaces (id, name, slug) VALUES ('ws_default', 'Default Workspace', 'ws_default')
ON CONFLICT (id) DO NOTHING;

INSERT INTO applications (id, workspace_id, name) VALUES
    ('app_sla_lab', 'ws_default', 'SLA Lab')
ON CONFLICT (id) DO NOTHING;

INSERT INTO machines (id, application_id, name, process) VALUES
    ('mch_sla_case', 'app_sla_lab', 'SLA Case', '{
      "states": ["Open", "Review", "Escalated", "Closed"],
      "transitions": [
        {"name": "Submit",  "from": "Open",      "to": "Review",    "actor": {"role": "Worker"}},
        {"name": "Close",   "from": "Review",    "to": "Closed",    "actor": {"role": "Manager"}},
        {"name": "Resolve", "from": "Escalated", "to": "Closed",    "actor": {"role": "Manager"}}
      ],
      "sla": [
        {"state": "Review", "duration": "0 Days",
         "on_breach": {"notify": {"role": "Manager"}, "escalate_to": "Escalated"}}
      ]
    }'::jsonb)
ON CONFLICT (id) DO UPDATE SET process = EXCLUDED.process;

INSERT INTO fields (id, machine_id, name, type, position, required, options) VALUES
    ('fld_sc_title', 'mch_sla_case', 'Title', 'text', 0, true, '{}')
ON CONFLICT (id) DO NOTHING;

INSERT INTO views (id, machine_id, name, type, position, config) VALUES
    ('vw_sc_form',   'mch_sla_case', 'New Case', 'form', 0, '{"fields":["fld_sc_title"]}'),
    ('vw_sc_list',   'mch_sla_case', 'Cases', 'list', 1, '{"columns":["fld_sc_title"]}'),
    ('vw_sc_detail', 'mch_sla_case', 'Case Detail', 'detail', 2, '{}')
ON CONFLICT (id) DO NOTHING;

INSERT INTO users (workspace_id, name, email, password_hash, workspace_role) VALUES
    ('ws_default', 'Wisnu SLA', 'sla.worker@example.com',
     '$2a$10$moxxOcZzSu3ILTzJlLF2Q.9vxiGNnSXPl7kY1pT3t5o1FoDjqC8aK', 'Member'),
    ('ws_default', 'Maya SLA', 'sla.manager@example.com',
     '$2a$10$moxxOcZzSu3ILTzJlLF2Q.9vxiGNnSXPl7kY1pT3t5o1FoDjqC8aK', 'Member')
ON CONFLICT (workspace_id, email) DO NOTHING;

INSERT INTO user_application_roles (user_id, application_id, role)
SELECT u.id, 'app_sla_lab',
       CASE u.email WHEN 'sla.worker@example.com' THEN 'Worker' ELSE 'Manager' END
FROM users u WHERE u.email IN ('sla.worker@example.com', 'sla.manager@example.com')
ON CONFLICT (user_id, application_id) DO UPDATE SET role = EXCLUDED.role;
