-- seeds/058_case3_sla_wiring.sql
-- case-03-case-19-completion-checklist.md's own Stage A item 1
-- (composable-runtime-roadmap.md 17r): closes Case 3's "SLA chips not
-- wired" gap against document-approval.html (ui-sample, checked
-- directly) -- the mockup's own real card markup shows "OVERDUE"/
-- "SLA breached · Nh" badges keyed on a real due date, which
-- mch_approval_document never had a Field for at all (confirmed by
-- direct field check, also already named in seeds/048's own comment).
--
-- Pure metadata: CAP-V17 (SLA badge, sla_field/sla_warning_days) is
-- already Supported and already proven elsewhere (SLA Badge Lab,
-- seeds/029; V02T2/V09T2 Lab, seeds/048) -- this seed only points that
-- already-built mechanism at a real Field/View on this Machine. No code
-- change, no capability-lifecycle.md admission needed (reuses an
-- existing, already-Supported render-time mechanism verbatim).

INSERT INTO fields (id, machine_id, name, type, position, required, options) VALUES
    ('fld_ad_due_date', 'mch_approval_document', 'Due Date', 'date', 7, false, '{}')
ON CONFLICT (id) DO NOTHING;

-- List views: the SLA badge substitutes for the raw date cell only when
-- the Field is a declared column (record_crud.go's own List rendering,
-- confirmed by reading it directly) -- Detail needs no columns change,
-- composable_detail.go's own substitution runs against every Machine
-- Field generically.
UPDATE views
   SET config = jsonb_set(config || '{"sla_field":"fld_ad_due_date","sla_warning_days":3}'::jsonb,
         '{columns}', (config->'columns') || '["fld_ad_due_date"]'::jsonb)
 WHERE id = 'vw_ad_all'
   AND NOT (config ? 'sla_field');

UPDATE views
   SET config = jsonb_set(config || '{"sla_field":"fld_ad_due_date","sla_warning_days":3}'::jsonb,
         '{columns}', (config->'columns') || '["fld_ad_due_date"]'::jsonb)
 WHERE id = 'vw_ad_pending'
   AND NOT (config ? 'sla_field');

UPDATE views
   SET config = config || '{"sla_field":"fld_ad_due_date","sla_warning_days":3}'::jsonb
 WHERE id = 'vw_ad_detail'
   AND NOT (config ? 'sla_field');

-- Two real demo Documents, fixed ids so the badge has something stable
-- to point at on the live site (Case 3 has no prior fixed-id seed
-- record convention at all -- every existing Document comes from
-- conformance-test churn or live usage). Dates fixed far in the past/
-- future, same "unambiguous bucket only" precedent seeds/029's own
-- header comment already established -- a date near "today" would
-- silently drift out of its bucket days after this seed runs, on a
-- deployment meant to stay live for a long time.
SET app.workspace_id = 'ws_default';

INSERT INTO records (id, machine_id, workspace_id, data, created_at, updated_at)
SELECT '44444444-5555-6666-7777-000000000001', 'mch_approval_document', 'ws_default',
       jsonb_build_object(
         'fld_ad_title', 'Vendor Contract Q3', 'fld_ad_document_type', 'Contract',
         'fld_ad_file', 'vendor-contract-q3.pdf', 'fld_ad_submitted_by', id,
         'fld_ad_approval_mode', 'Sequential', 'fld_ad_status', 'In Review',
         'fld_ad_due_date', '2026-08-01'
       ), NOW(), NOW()
  FROM users WHERE email = 'alice@example.com'
ON CONFLICT (id) DO NOTHING;

INSERT INTO records (id, machine_id, workspace_id, data, created_at, updated_at)
SELECT '44444444-5555-6666-7777-000000000002', 'mch_approval_document', 'ws_default',
       jsonb_build_object(
         'fld_ad_title', 'Procurement SOP', 'fld_ad_document_type', 'SOP',
         'fld_ad_file', 'procurement-sop.pdf', 'fld_ad_submitted_by', id,
         'fld_ad_approval_mode', 'Parallel', 'fld_ad_status', 'In Review',
         'fld_ad_due_date', '2027-06-01'
       ), NOW(), NOW()
  FROM users WHERE email = 'alice@example.com'
ON CONFLICT (id) DO NOTHING;
