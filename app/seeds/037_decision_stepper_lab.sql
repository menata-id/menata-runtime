-- seeds/037_decision_stepper_lab.sql
-- CAP-V20: sequential decision stepper (Study 32,
-- ../../benchmarks/024-pdf-signature-approval-study.md §3's Decision
-- screen). Adds ONE new View to Approval Document -- no new fields, no new
-- Machine.Config keys: steps_machine/steps_parent_field already exist on
-- mch_approval_document (seeds/004_approval.sql), CAP-V20's handler is
-- simply the first code to actually read them. Safe to run multiple times
-- (ON CONFLICT DO NOTHING).

INSERT INTO views (id, machine_id, name, type, position, config) VALUES
    ('vw_ad_progress', 'mch_approval_document', 'Decision Progress', 'decision_stepper', 3,
     '{"decision_stepper":{"sequence_field":"fld_as_sequence","decision_field":"fld_as_decision"}}')
ON CONFLICT (id) DO NOTHING;
