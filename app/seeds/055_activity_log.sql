-- seeds/055_activity_log.sql
-- CAP-R04 "R28" (composable-runtime-roadmap.md 17p): the read side of the
-- record_events audit trail, admitted this same session
-- (capability-registry.md's CAP-R04 row). One new activity_log View per
-- Machine, embedded in BOTH modes the admission decided on:
--   - record-scoped (CAP-V20 Children on the Detail View) -- one host
--     record's own history.
--   - cross-record (CAP-V10 Tier 2 Children on a composed page) --
--     recent activity across the whole Machine, no host record.
-- Which mode applies is decided entirely by which mechanism resolves the
-- Children entry (embed.go vs page.go), not by anything in Config here.
INSERT INTO views (id, machine_id, name, type, position, config) VALUES
    ('vw_ad_activity', 'mch_approval_document', 'Activity', 'activity_log', 3, '{}'),
    ('vw_pmc_activity', 'mch_pm_card', 'Activity', 'activity_log', 4, '{}')
ON CONFLICT (id) DO NOTHING;

-- Record-scoped mode: Document Detail's own history. vw_ad_detail's
-- config starts as '{}' (seeds/004_approval.sql) -- no prior Children to
-- disturb, so a plain jsonb_set (create_missing=true, the default) adds
-- the key fresh. Guarded so re-running this seed never double-appends.
UPDATE views
   SET config = jsonb_set(config, '{children}', '[{"view":"vw_ad_activity","title":"Activity"}]'::jsonb)
 WHERE id = 'vw_ad_detail'
   AND NOT (config ? 'children');

-- Record-scoped mode: Card Detail's own history (Case 19, Study 40's own
-- real target -- legitimately empty against today's real seeded data,
-- since mch_pm_card has zero declared Events; the mechanism itself is
-- real, named honestly in composable-runtime-roadmap.md's own 17p
-- section, not hidden).
UPDATE views
   SET config = jsonb_set(config, '{children}', '[{"view":"vw_pmc_activity","title":"Activity"}]'::jsonb)
 WHERE id = 'vw_pmc_detail'
   AND NOT (config ? 'children');

-- Cross-record mode: vw_ad_page's own "Recent Activity" section, a named
-- placeholder since 17k (seeds/050_composed_dashboard.sql) pending
-- exactly this capability -- replaced now with the real thing, same
-- index (2) and layout ("aside") so buildPageRows' own main+aside
-- pairing with vw_ad_pending (index 1) is unaffected. Append/patch, not
-- a rewrite of 050's own INSERT -- same precedent 053/054's own UPDATEs
-- already established for this exact view.
UPDATE views
   SET config = jsonb_set(config, '{children,2}', '{"view":"vw_ad_activity","title":"Recent Activity","layout":"aside"}'::jsonb)
 WHERE id = 'vw_ad_page'
   AND config->'children'->2->>'title' = 'Recent Activity'
   AND config->'children'->2->'content' IS NOT NULL;
