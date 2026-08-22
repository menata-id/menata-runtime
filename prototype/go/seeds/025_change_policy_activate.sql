-- seeds/025_change_policy_activate.sql
-- CAP-W07 proof: the actual metadata CHANGE -- two new Constraint rows on
-- the already-live mch_policy_case, each gating a DIFFERENT field so
-- conformance/run.sh's T156/T157 can isolate the dimension each
-- change_policy is proving (state vs. date) rather than the two rules
-- tripping over each other on a shared field. Deliberately NOT part of
-- `make seed`'s boot-time list -- applied via a direct `psql` call
-- mid-conformance-run, then picked up by POST /admin/reload (CAP-X04),
-- same established pattern as seeds/023_reload_lab.sql/T151.

INSERT INTO constraints (id, machine_id, rule, expression, position, change_policy) VALUES
    ('cst_pc_states', 'mch_policy_case',
     'Compliance Note is required for cases still in Draft',
     '{"field":"fld_pc_compliance","operator":"required"}', 0,
     '{"applies_to":"records_in_states","states":["Draft"]}'),
    ('cst_pc_new', 'mch_policy_case',
     'Approval Reference is required for cases opened under the policy effective 2026-01-01',
     '{"field":"fld_pc_approval_ref","operator":"required"}', 1,
     '{"applies_to":"new_records","effective_from":"2026-01-01"}')
ON CONFLICT (id) DO NOTHING;
