-- seeds/028_lift_lab.sql
-- B6 (decompile-lift, CAP-W05 backward direction) proof fixture. Lives in
-- the SAME app_overlay_lab Application as mch_ca_manual/mch_ca_overlay
-- (019_overlay_lab.sql), reusing those same seeded accounts/roles
-- (Supervisor/Worker/Reviewer, SURYA/WATI/RIAN/WINDA) with zero new setup.
--
-- Deliberately NOT part of `make seed`'s boot-time list, and its `process`
-- column is intentionally left unset here -- applied mid-conformance-run via
-- a second psql call (the actual JSON output of GET /mch_ca_manual/process-lift),
-- then picked up by POST /admin/reload (CAP-X04), same established
-- exclusion pattern as 023/025/026_*.sql.

INSERT INTO machines (id, application_id, name) VALUES
    ('mch_ca_lifted', 'app_overlay_lab', 'Corrective Action (Lifted)')
ON CONFLICT (id) DO NOTHING;

INSERT INTO fields (id, machine_id, name, type, position, required, options) VALUES
    ('fld_cal_title',    'mch_ca_lifted', 'Title',    'text', 0, true,  '{}'),
    ('fld_cal_assignee', 'mch_ca_lifted', 'Assignee', 'user', 1, false, '{}')
ON CONFLICT (id) DO NOTHING;

INSERT INTO views (id, machine_id, name, type, position, config) VALUES
    ('vw_cal_form', 'mch_ca_lifted', 'New Corrective Action', 'form', 0,
     '{"fields":["fld_cal_title","fld_cal_assignee"]}')
ON CONFLICT (id) DO NOTHING;
