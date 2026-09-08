-- seeds/044_document_submit_dashboard_live_wiring.sql
-- Closes two of the metadata-only gaps `app/docs/ui-component-library.md`'s
-- "Known gaps against real mockups (2026-09-07)" table named against the
-- live app but never actually wired into the persistent `menata_runtime`
-- database that serves menata.app -- the prior verification for these two
-- pieces ran against an isolated throwaway schema (`CREATE SCHEMA
-- verify_ui_components`, dropped afterward), so the capability was proven
-- in general but this Case's own live metadata was never updated to use it.
--
-- 1. `document-submit.html`'s "Approval steps" section -- CAP-F16
--    (`child_lines`) applied to the real Document Submission form for the
--    first time, embedding Approval Step authoring (including the CAP-F24
--    User/Group toggle fields) directly in the Document create form. NOT
--    combined with CAP-V12 `steps` (wizard) on the same View: `ui.
--    WizardForm` (`internal/ui/wizard.templ`) has no `childLines` parameter
--    today, so a wizard-paginated form cannot also render embedded child
--    rows -- a real code-level gap (not a metadata omission), named here
--    rather than silently worked around. Choosing child_lines over the
--    wizard keeps the functionally novel part of the mockup (the approver
--    rows, CAP-F24) working for real, at the cost of the mockup's own
--    3-step pagination, which stays unmet by this seed.
--
-- 2. `approval-dashboard.html`'s "Summary" section -- a real CAP-V10
--    `dashboard` View for Approval Document, grouped by Status. Scoped
--    down from the mockup's own literal tiles ("Total Pending" broken down
--    by a Document/Leave/Corrective Action cross-Machine split, "Overdue"/
--    "Due Today" computed from an SLA threshold) -- this app has no Leave
--    or Corrective Action Machine, and no SLA field/mechanism is declared
--    on Approval Document, so those exact tiles have no real data to
--    source from. A per-Status breakdown is what CAP-V10's existing
--    Sections mechanism (`{title, machine, group_field}`) actually
--    supports without new code -- a smaller, honest application of the
--    same already-✅ capability, not a pixel match of the mockup's own
--    exploratory tile set.

UPDATE views SET config = config || '{"child_lines":{"machine":"mch_approval_step","parent_field":"fld_as_document","fields":["fld_as_approver_type","fld_as_approver_user","fld_as_approver_group","fld_as_sequence"],"max_rows":5}}'
    WHERE id = 'vw_ad_form';

INSERT INTO views (id, machine_id, name, type, position, config) VALUES
    ('vw_ad_dashboard', 'mch_approval_document', 'Approval Dashboard', 'dashboard', 4,
     '{"sections":[{"title":"Approval Documents by Status","machine":"mch_approval_document","group_field":"fld_ad_status"}]}')
ON CONFLICT (id) DO NOTHING;

-- Label alignment with document-approval.html's own section headings
-- (`children`-composed inline sections on the Step Detail page, seeds/
-- 042_inline_view_composition.sql) -- these two Views already render the
-- right content, this only renames them to match the mockup's own copy
-- exactly ("Approval Progress" / "Your Signature Position" rather than
-- "Decision Progress" / "Set Signature Position").
UPDATE views SET name = 'Approval Progress' WHERE id = 'vw_ad_progress';
UPDATE views SET name = 'Your Signature Position' WHERE id = 'vw_as_place';
