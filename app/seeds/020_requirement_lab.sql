-- seeds/020_requirement_lab.sql
-- Process Overlay B3 proof (Study 21 addendum, CAP-W01): generic Requirement
-- with evidence cardinality -- the comparator BRD's own sharpest named gap
-- (Study 19 §4.1: "no generic check exists for evidence-count"), and Study
-- 20 §6.3's "write-time fan-in, read-time O(1)" performance keystone.
--
-- A NEW, self-contained lab (new Machines, new Application) rather than an
-- edit to seeds/019 -- this codebase's own "ratchet rule — new Machines,
-- not a retrofit" (see CAP-P03's note): 019's mch_ca_overlay/mch_ca_manual
-- are already driven through every transition (T136-T143), and adding a
-- cardinality requirement to their existing Submit would break those.
--
-- mch_req_case declares ONE requirement: Submit (Open -> Submitted) needs
-- at least 2 mch_req_photo records referencing it. mch_req_photo is a
-- plain data-entry Machine with no Events of its own -- attaching one is
-- an ordinary Create, and that Create is what stamps mch_req_case's
-- generated counter field (fld_mch_req_case_mch_req_photo_count) via
-- handler.stampRequirementCounters.

INSERT INTO workspaces (id, name) VALUES ('ws_default', 'Default Workspace')
ON CONFLICT (id) DO NOTHING;

INSERT INTO applications (id, workspace_id, name) VALUES
    ('app_req_lab', 'ws_default', 'Requirement Lab')
ON CONFLICT (id) DO NOTHING;

INSERT INTO machines (id, application_id, name, process) VALUES
    ('mch_req_case', 'app_req_lab', 'Requirement Case', '{
      "states": ["Open", "Submitted"],
      "transitions": [
        {"name": "Submit", "from": "Open", "to": "Submitted", "actor": {"role": "Worker"},
         "requirements": [
           {"type": "evidence", "target": "mch_req_photo", "cardinality": "2..*"}
         ]}
      ]
    }'::jsonb)
ON CONFLICT (id) DO UPDATE SET process = EXCLUDED.process;

INSERT INTO machines (id, application_id, name) VALUES
    ('mch_req_photo', 'app_req_lab', 'Requirement Photo')
ON CONFLICT (id) DO NOTHING;

INSERT INTO fields (id, machine_id, name, type, position, required, options) VALUES
    ('fld_rc_title', 'mch_req_case', 'Title', 'text', 0, true, '{}'),
    ('fld_rp_case',    'mch_req_photo', 'Case',    'reference', 0, true,  '{"target_machine":"mch_req_case"}'),
    ('fld_rp_caption', 'mch_req_photo', 'Caption', 'text',      1, false, '{}')
ON CONFLICT (id) DO NOTHING;

-- mch_req_photo has no Events -- attaching one is a plain Create, same
-- posture as seeds/003_hr_employee.sql's own "not every Object needs one".
INSERT INTO permissions (id, machine_id, role, events, can_read, can_create) VALUES
    ('perm_rp_worker', 'mch_req_photo', 'Worker', ARRAY[]::TEXT[], true, true)
ON CONFLICT (id) DO NOTHING;

INSERT INTO views (id, machine_id, name, type, position, config) VALUES
    ('vw_rc_form',   'mch_req_case', 'New Case', 'form', 0, '{"fields":["fld_rc_title"]}'),
    ('vw_rc_list',   'mch_req_case', 'Cases', 'list', 1, '{"columns":["fld_rc_title"]}'),
    ('vw_rc_detail', 'mch_req_case', 'Case Detail', 'detail', 2, '{}'),
    ('vw_rp_form',   'mch_req_photo', 'Attach Photo', 'form', 0, '{"fields":["fld_rp_case","fld_rp_caption"]}'),
    ('vw_rp_list',   'mch_req_photo', 'Photos', 'list', 1, '{"columns":["fld_rp_caption"]}'),
    ('vw_rp_detail', 'mch_req_photo', 'Photo Detail', 'detail', 2, '{}')
ON CONFLICT (id) DO NOTHING;

INSERT INTO users (workspace_id, name, email, password_hash, workspace_role) VALUES
    ('ws_default', 'Wira Requirement', 'req.worker@example.com',
     '$2a$10$moxxOcZzSu3ILTzJlLF2Q.9vxiGNnSXPl7kY1pT3t5o1FoDjqC8aK', 'Member')
ON CONFLICT (workspace_id, email) DO NOTHING;

INSERT INTO user_application_roles (user_id, application_id, role)
SELECT u.id, 'app_req_lab', 'Worker' FROM users u WHERE u.email = 'req.worker@example.com'
ON CONFLICT (user_id, application_id) DO UPDATE SET role = EXCLUDED.role;
