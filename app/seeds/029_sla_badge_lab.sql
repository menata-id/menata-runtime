-- seeds/029_sla_badge_lab.sql
-- CAP-V17 proof: a countdown badge rendered at request time
-- (handler.slaUrgency), same "computed at render time, nothing stored"
-- precedent as CAP-F14/CAP-V13 -- no new mechanism, just a new View config
-- key (ViewConfig.SlaField/SlaWarningDays) read at List/Detail render time.
--
-- A NEW, self-contained lab (new Application, new Machine) with three
-- records fixed relative to the seed's own load time: one clearly overdue
-- (a date far in the past), one clearly not (a date far in the future).
-- Warning-bucket boundary testing is deliberately not exercised here (a
-- date-relative-to-NOW() fixture would be flaky against a fixed
-- sla_warning_days threshold run months apart) -- the two unambiguous
-- buckets (overdue / not-overdue) are what's proven.

INSERT INTO workspaces (id, name, slug) VALUES ('ws_default', 'Default Workspace', 'ws_default')
ON CONFLICT (id) DO NOTHING;

INSERT INTO applications (id, workspace_id, name) VALUES
    ('app_sla_badge_lab', 'ws_default', 'SLA Badge Lab')
ON CONFLICT (id) DO NOTHING;

INSERT INTO machines (id, application_id, name) VALUES
    ('mch_sla_ticket', 'app_sla_badge_lab', 'Ticket')
ON CONFLICT (id) DO NOTHING;

INSERT INTO fields (id, machine_id, name, type, position, required, options) VALUES
    ('fld_slat_title', 'mch_sla_ticket', 'Title', 'text', 0, true, '{}'),
    ('fld_slat_due',   'mch_sla_ticket', 'Due',   'date', 1, false, '{}')
ON CONFLICT (id) DO NOTHING;

INSERT INTO permissions (id, machine_id, role, events, can_read, can_create, can_edit, can_delete) VALUES
    ('perm_slat_agent', 'mch_sla_ticket', 'Agent', ARRAY[]::TEXT[], true, true, true, false)
ON CONFLICT (id) DO NOTHING;

INSERT INTO views (id, machine_id, name, type, position, config) VALUES
    ('vw_slat_list', 'mch_sla_ticket', 'Tickets', 'list', 0,
     '{"columns":["fld_slat_title","fld_slat_due"],"sla_field":"fld_slat_due","sla_warning_days":3}'),
    ('vw_slat_detail', 'mch_sla_ticket', 'Ticket Detail', 'detail', 1,
     '{"sla_field":"fld_slat_due","sla_warning_days":3}'),
    ('vw_slat_form', 'mch_sla_ticket', 'New Ticket', 'form', 2,
     '{"fields":["fld_slat_title","fld_slat_due"]}')
ON CONFLICT (id) DO NOTHING;

INSERT INTO users (workspace_id, name, email, password_hash, workspace_role) VALUES
    ('ws_default', 'SLA Agent', 'sla.agent@example.com',
     '$2a$10$moxxOcZzSu3ILTzJlLF2Q.9vxiGNnSXPl7kY1pT3t5o1FoDjqC8aK', 'Member')
ON CONFLICT (workspace_id, email) DO NOTHING;

INSERT INTO user_application_roles (user_id, application_id, role)
SELECT u.id, 'app_sla_badge_lab', 'Agent' FROM users u WHERE u.email = 'sla.agent@example.com'
ON CONFLICT (user_id, application_id) DO UPDATE SET role = EXCLUDED.role;
