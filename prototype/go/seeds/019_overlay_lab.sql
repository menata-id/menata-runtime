-- seeds/019_overlay_lab.sql
-- Process Overlay B1 parity experiment (Study 21, brd-menata-runtime-v2.md
-- §6.6/§8): TWO Machines carrying the SAME Corrective Action process (the
-- comparator BRD's own worked example, reduced to the B1 subset -- no
-- requirements/SLA/quorum yet):
--
--   mch_ca_manual   -- hand-authored the v1 way: explicit Events with
--                      CAP-E06 guards, explicit Permissions, explicit
--                      Status field, explicit CAP-E05 trigger_event chain
--                      for the Submitted -> Review auto step.
--   mch_ca_overlay  -- ONLY a `process` declaration; Events/guards/
--                      Permissions/Status field are all compiled at load
--                      time (internal/metadata/compile.go).
--
-- Conformance T136-T139 drive both through the identical lifecycle and
-- assert identical behavior -- the falsifiable core claim of the v2
-- concept ("declared process, emergent execution", DoD §6.6).
--
-- Re-run note: like every seed, fields/machines/permissions/views use
-- ON CONFLICT DO NOTHING (the overlay machine's process column uses
-- DO UPDATE so declaration edits propagate on re-run); event_actions has
-- no natural key -- fresh-database or one-time apply only, same caveat as
-- the Makefile's own seed target.

INSERT INTO workspaces (id, name) VALUES ('ws_default', 'Default Workspace')
ON CONFLICT (id) DO NOTHING;

INSERT INTO applications (id, workspace_id, name) VALUES
    ('app_overlay_lab', 'ws_default', 'Overlay Lab')
ON CONFLICT (id) DO NOTHING;

-- ---------------------------------------------------------------------------
-- Machine 1: hand-authored (the v1 control arm)
-- ---------------------------------------------------------------------------

INSERT INTO machines (id, application_id, name) VALUES
    ('mch_ca_manual', 'app_overlay_lab', 'Corrective Action (Manual)')
ON CONFLICT (id) DO NOTHING;

INSERT INTO fields (id, machine_id, name, type, position, required, options) VALUES
    ('fld_cam_title',    'mch_ca_manual', 'Title',    'text', 0, true,  '{}'),
    ('fld_cam_assignee', 'mch_ca_manual', 'Assignee', 'user', 1, false, '{}'),
    ('fld_cam_status',   'mch_ca_manual', 'Status',   'value_list', 2, false,
     '{"values":["Open","Assigned","In_Progress","Submitted","Review","Verified","Closed"]}')
ON CONFLICT (id) DO NOTHING;

INSERT INTO events (id, machine_id, name, position, condition) VALUES
    ('evt_cam_assign',  'mch_ca_manual', 'Assign',  0, '{"field":"fld_cam_status","operator":"equals","value":"Open"}'),
    ('evt_cam_start',   'mch_ca_manual', 'Start',   1, '{"field":"fld_cam_status","operator":"equals","value":"Assigned"}'),
    ('evt_cam_submit',  'mch_ca_manual', 'Submit',  2, '{"field":"fld_cam_status","operator":"equals","value":"In_Progress"}'),
    ('evt_cam_approve', 'mch_ca_manual', 'Approve', 3, '{"field":"fld_cam_status","operator":"equals","value":"Review"}'),
    ('evt_cam_revise',  'mch_ca_manual', 'Revise',  4, '{"field":"fld_cam_status","operator":"equals","value":"Review"}'),
    ('evt_cam_close',   'mch_ca_manual', 'Close',   5, '{"field":"fld_cam_status","operator":"equals","value":"Verified"}'),
    -- the hand-rolled auto step: no Permission row lists it, so only the
    -- System-side trigger_event chain below can ever fire it
    ('evt_cam_auto',    'mch_ca_manual', 'Auto: Submitted → Review', 6,
     '{"field":"fld_cam_status","operator":"equals","value":"Submitted"}')
ON CONFLICT (id) DO NOTHING;

INSERT INTO event_actions (event_id, type, position, params) VALUES
    ('evt_cam_assign',  'set_field', 0, '{"field":"fld_cam_status","value":"Assigned"}'),
    ('evt_cam_start',   'set_field', 0, '{"field":"fld_cam_status","value":"In_Progress"}'),
    ('evt_cam_submit',  'set_field', 0, '{"field":"fld_cam_status","value":"Submitted"}'),
    ('evt_cam_submit',  'trigger_event', 1, '{"event":"evt_cam_auto"}'),
    ('evt_cam_auto',    'set_field', 0, '{"field":"fld_cam_status","value":"Review"}'),
    ('evt_cam_approve', 'set_field', 0, '{"field":"fld_cam_status","value":"Verified"}'),
    ('evt_cam_revise',  'set_field', 0, '{"field":"fld_cam_status","value":"In_Progress"}'),
    ('evt_cam_close',   'set_field', 0, '{"field":"fld_cam_status","value":"Closed"}');

INSERT INTO permissions (id, machine_id, role, events, owner_field) VALUES
    ('perm_cam_supervisor', 'mch_ca_manual', 'Supervisor',
     ARRAY['evt_cam_assign','evt_cam_close'], NULL),
    ('perm_cam_worker',     'mch_ca_manual', 'Worker',
     ARRAY['evt_cam_start','evt_cam_submit'], 'fld_cam_assignee'),
    ('perm_cam_reviewer',   'mch_ca_manual', 'Reviewer',
     ARRAY['evt_cam_approve','evt_cam_revise'], NULL)
ON CONFLICT (id) DO NOTHING;

INSERT INTO views (id, machine_id, name, type, position, config) VALUES
    ('vw_cam_form',   'mch_ca_manual', 'New Corrective Action', 'form', 0,
     '{"fields":["fld_cam_title","fld_cam_assignee"]}'),
    ('vw_cam_list',   'mch_ca_manual', 'Corrective Actions', 'list', 1,
     '{"columns":["fld_cam_title","fld_cam_status"]}'),
    ('vw_cam_detail', 'mch_ca_manual', 'Corrective Action Detail', 'detail', 2, '{}')
ON CONFLICT (id) DO NOTHING;

-- ---------------------------------------------------------------------------
-- Machine 2: overlay-declared (the experiment arm) -- NO events, NO
-- permissions, NO status field in this file; all compiled from `process`.
-- ---------------------------------------------------------------------------

INSERT INTO machines (id, application_id, name, process) VALUES
    ('mch_ca_overlay', 'app_overlay_lab', 'Corrective Action (Overlay)', '{
      "states": ["Open","Assigned","In_Progress","Submitted","Review","Verified","Closed"],
      "transitions": [
        {"name":"Assign",  "from":"Open",        "to":"Assigned",    "actor":{"role":"Supervisor"}},
        {"name":"Start",   "from":"Assigned",    "to":"In_Progress", "actor":{"role":"Worker","owner_field":"fld_cao_assignee"}},
        {"name":"Submit",  "from":"In_Progress", "to":"Submitted",   "actor":{"role":"Worker","owner_field":"fld_cao_assignee"}},
        {"name":"Approve", "from":"Review",      "to":"Verified",    "actor":{"role":"Reviewer"}},
        {"name":"Revise",  "from":"Review",      "to":"In_Progress", "actor":{"role":"Reviewer"}},
        {"name":"Close",   "from":"Verified",    "to":"Closed",      "actor":{"role":"Supervisor"}}
      ],
      "auto": [ {"from":"Submitted","to":"Review"} ]
    }'::jsonb)
ON CONFLICT (id) DO UPDATE SET process = EXCLUDED.process;

INSERT INTO fields (id, machine_id, name, type, position, required, options) VALUES
    ('fld_cao_title',    'mch_ca_overlay', 'Title',    'text', 0, true,  '{}'),
    ('fld_cao_assignee', 'mch_ca_overlay', 'Assignee', 'user', 1, false, '{}')
ON CONFLICT (id) DO NOTHING;

-- The status column names the COMPILED field id ("fld_<machine>_status" --
-- deterministic, see compile.go's resolveStatusField).
INSERT INTO views (id, machine_id, name, type, position, config) VALUES
    ('vw_cao_form',   'mch_ca_overlay', 'New Corrective Action', 'form', 0,
     '{"fields":["fld_cao_title","fld_cao_assignee"]}'),
    ('vw_cao_list',   'mch_ca_overlay', 'Corrective Actions', 'list', 1,
     '{"columns":["fld_cao_title","fld_mch_ca_overlay_status"]}'),
    ('vw_cao_detail', 'mch_ca_overlay', 'Corrective Action Detail', 'detail', 2, '{}')
ON CONFLICT (id) DO NOTHING;

-- ---------------------------------------------------------------------------
-- Accounts: one per role, plus a second Worker who is never the assignee
-- (the CAP-P02 ownership-parity probe). Password: "password", same bcrypt
-- hash convention as every other seeded account.
-- ---------------------------------------------------------------------------

INSERT INTO users (workspace_id, name, email, password_hash, workspace_role) VALUES
    ('ws_default', 'Surya Overlay', 'overlay.sup@example.com',
     '$2a$10$moxxOcZzSu3ILTzJlLF2Q.9vxiGNnSXPl7kY1pT3t5o1FoDjqC8aK', 'Member'),
    ('ws_default', 'Wati Overlay', 'overlay.worker@example.com',
     '$2a$10$moxxOcZzSu3ILTzJlLF2Q.9vxiGNnSXPl7kY1pT3t5o1FoDjqC8aK', 'Member'),
    ('ws_default', 'Winda Overlay', 'overlay.worker2@example.com',
     '$2a$10$moxxOcZzSu3ILTzJlLF2Q.9vxiGNnSXPl7kY1pT3t5o1FoDjqC8aK', 'Member'),
    ('ws_default', 'Rian Overlay', 'overlay.reviewer@example.com',
     '$2a$10$moxxOcZzSu3ILTzJlLF2Q.9vxiGNnSXPl7kY1pT3t5o1FoDjqC8aK', 'Member')
ON CONFLICT (workspace_id, email) DO NOTHING;

INSERT INTO user_application_roles (user_id, application_id, role)
SELECT u.id, 'app_overlay_lab',
       CASE u.email
           WHEN 'overlay.sup@example.com'      THEN 'Supervisor'
           WHEN 'overlay.reviewer@example.com' THEN 'Reviewer'
           ELSE 'Worker'
       END
FROM users u
WHERE u.email IN ('overlay.sup@example.com','overlay.worker@example.com',
                  'overlay.worker2@example.com','overlay.reviewer@example.com')
ON CONFLICT (user_id, application_id) DO UPDATE SET role = EXCLUDED.role;
