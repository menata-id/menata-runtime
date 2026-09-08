-- seeds/045_declared_navigation_pilot.sql
-- CAP-O03 Tier 5 (declared navigation, Phase 1) -- the pilot named in this
-- capability's own registry row and benchmarks/009's implementation plan:
-- app_approval is the real application this whole capability was motivated
-- by (Case 3's own observed nav clutter -- Approval Step and Signature
-- showing as equal-weight links a Submitter/Approver never actually picks
-- from a menu, only ever reaching either through the Document's own flow).
--
-- Declaring these rows makes app_approval's sub-nav strip and app-launcher
-- card list show exactly:
--   Approval Document              (machine target -- the one real destination)
--   Dashboard                      (view target -- vw_ad_dashboard, seeds/044)
--   Reference › Signature          (group target, with one nested child)
--
-- Approval Step is deliberately NOT declared here -- absent from both, per
-- this capability's whole point. It stays fully readable/creatable/
-- reachable directly (via the Document's own embedded child_lines rows and
-- the Decision Progress stepper's own links into it) -- this only removes
-- it from the curated menu, never from access.
INSERT INTO navigation_entries (id, application_id, parent_id, position, label, target_type, target_machine, target_view) VALUES
    ('nav_ad_document',  'app_approval', NULL,               0, 'Approval Document', 'machine', 'mch_approval_document', NULL),
    ('nav_ad_dashboard', 'app_approval', NULL,               1, 'Dashboard',         'view',    NULL,                    'vw_ad_dashboard'),
    ('nav_ad_reference', 'app_approval', NULL,               2, 'Reference',         'group',   NULL,                    NULL),
    ('nav_ad_signature', 'app_approval', 'nav_ad_reference', 0, 'Signature',         'machine', 'mch_signature',         NULL)
ON CONFLICT (id) DO NOTHING;
