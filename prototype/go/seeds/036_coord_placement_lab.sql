-- seeds/036_coord_placement_lab.sql
-- CAP-V21: coordinate-placement editor (Study 32,
-- ../../benchmarks/024-pdf-signature-approval-study.md §5's Signature
-- Placement screen). Adds ONE new View to Approval Step, reusing the
-- (page, x%, y%) fields CAP-F22 already reads (seeds/035_pdf_signature_
-- lab.sql) and the Document's original PDF (fld_ad_file, seeds/
-- 004_approval.sql) as the preview source. Safe to run multiple times
-- (ON CONFLICT DO NOTHING).

INSERT INTO views (id, machine_id, name, type, position, config) VALUES
    ('vw_as_place', 'mch_approval_step', 'Set Signature Position', 'coord_placement', 3,
     '{"coord_placement":{"reference_field":"fld_as_document","preview_field":"fld_ad_file","page_field":"fld_as_signature_page","x_field":"fld_as_signature_x","y_field":"fld_as_signature_y"}}')
ON CONFLICT (id) DO NOTHING;
