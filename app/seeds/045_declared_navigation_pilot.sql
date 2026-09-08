-- seeds/045_declared_navigation_pilot.sql
-- CAP-O03 Tier 5 (declared navigation, Phase 1) -- the pilot named in this
-- capability's own registry row and benchmarks/009's implementation plan:
-- app_approval is the real application this whole capability was motivated
-- by (Case 3's own observed nav clutter -- Approval Step showing as an
-- equal-weight link a Submitter/Approver never actually picks from a menu,
-- only ever reaching it through the Document's own flow).
--
-- Declaring these rows makes app_approval's sub-nav strip and app-launcher
-- card list show exactly:
--   Approval Document              (machine target -- the one real destination)
--   Dashboard                      (view target -- vw_ad_dashboard, seeds/044)
--
-- Approval Step is deliberately NOT declared here -- absent from both, per
-- this capability's whole point. It stays fully readable/creatable/
-- reachable directly (via the Document's own embedded child_lines rows and
-- the Decision Progress stepper's own links into it) -- this only removes
-- it from the curated menu, never from access.
INSERT INTO navigation_entries (id, application_id, parent_id, position, label, target_type, target_machine, target_view) VALUES
    ('nav_ad_document',  'app_approval', NULL, 0, 'Approval Document', 'machine', 'mch_approval_document', NULL),
    ('nav_ad_dashboard', 'app_approval', NULL, 1, 'Dashboard',         'view',    NULL,                    'vw_ad_dashboard')
ON CONFLICT (id) DO NOTHING;

-- Correction (2026-09-08, same day): the first version of this pilot also
-- declared a "Reference" group nesting Signature, purely to demonstrate
-- Phase 1's group/nesting mechanism in the same pilot that proves the
-- rest of it. Real mistake, caught from a live screenshot: (1) Signature
-- is NOT actually a real menu destination either, by this same capability's
-- own Case 3 note's own reasoning (reached only through the signature
-- registration flow, never browsed to directly) -- putting it back in the
-- curated menu contradicted that; (2) the extra group label's own width
-- (uppercase + letter-spacing) pushed the strip past a narrow viewport's
-- available space, cutting "Signature" off entirely with no visible
-- scroll affordance. Removing it both restores this row's own stated
-- reasoning and fixes the overflow. A database that already ran the
-- INSERT above from its first version needs this DELETE too --
-- ON CONFLICT DO NOTHING on the INSERT above does not undo it.
DELETE FROM navigation_entries WHERE id IN ('nav_ad_signature', 'nav_ad_reference');
