-- seeds/051_dashboard_nav_supersede.sql
-- The declared "Dashboard" nav entry (seeds/045_declared_navigation_pilot.sql,
-- nav_ad_dashboard) still pointed at vw_ad_dashboard -- CAP-V10 Tier 1's own
-- tiles-only view, real and unchanged since before today, but no longer the
-- screen that actually matches approval-dashboard.html (the canonical
-- mockup reference, app/CLAUDE.md's own ui-sample index table). vw_ad_page
-- (seeds/050) is that screen now: it already composes vw_ad_dashboard's own
-- tiles as its own first section, so nothing about vw_ad_dashboard's data
-- is lost, only which screen a person actually lands on from the nav.
--
-- vw_ad_dashboard itself is NOT deleted, NOT deprecated as metadata -- it
-- stays a real, valid View (vw_ad_page's own Children references it BY ID,
-- Interpreter.GetView, same reasoning app/CLAUDE.md already documents for
-- superseded ui-sample mockups: "don't resurrect it as a design reference,
-- don't delete it either"). Its own standalone route
-- (GET /mch_approval_document/dashboard) still works exactly as before --
-- only the declared nav entry that used to point AT it now points at the
-- composed page instead.

UPDATE navigation_entries SET target_view = 'vw_ad_page' WHERE id = 'nav_ad_dashboard';
