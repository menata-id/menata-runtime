-- seeds/050_composed_dashboard.sql
-- CAP-V10 Tier 2: a `page` View composing three sections onto one real
-- screen, realizing approval-dashboard.html's own composed shape (Summary
-- tiles + Pending Documents + Recent Activity, the latter two side by side)
-- as far as today's real data sources allow -- not a mockup, a real route
-- (GET /mch_approval_document/page).
--
-- Section 1 (Summary): the EXISTING vw_ad_dashboard, unchanged -- CAP-V10
-- as it already was.
--
-- Section 2 (Pending Documents): a NEW list View, vw_ad_pending, filtered
-- to Status = "In Review", display: cards. Deliberately NOT vw_ad_all
-- (the Machine's own default/reachable list, seeds/004 position 0) --
-- `page` composition resolves a Children entry's View by ID directly
-- (Interpreter.GetView), never through DefaultListView's own "first list
-- View by position" lookup that only governs the standalone GET
-- /mch_approval_document route. That's what makes this View reachable at
-- all despite the Machine already having vw_ad_all as its one
-- DefaultListView-resolved list -- a second `list`-type View is only ever
-- dead metadata if nothing ELSE references it by id; here the page's own
-- Children does. This is the real, structural answer to the "one
-- reachable list View" limitation found and documented earlier today
-- (guides/runtime-metadata-gotchas.md) -- composition-by-id was the way
-- through it, not a capability gap needing new code.
--
-- Section 3 (Recent Activity): a static `content` entry, not a real View --
-- named honestly as a placeholder. CAP-R04 (a record-history/activity
-- timeline View) does not exist yet; faking one here with invented data
-- would misrepresent what's actually composable today. This section
-- exists specifically to prove the LAYOUT mechanism itself (paired
-- main+aside 2/3+1/3 row) works even when one side has no real data
-- source -- View composition is a presentation-layer concern, independent
-- of whether every composed piece's own underlying data happens to exist.

INSERT INTO views (id, machine_id, name, type, position, config) VALUES
    ('vw_ad_pending', 'mch_approval_document', 'Pending Documents', 'list', 5,
     '{"columns":["fld_ad_title","fld_ad_submitted_by","fld_ad_status"],"display":"cards","filter":[{"field":"fld_ad_status","operator":"equals","value":"In Review"}]}')
ON CONFLICT (id) DO NOTHING;

INSERT INTO views (id, machine_id, name, type, position, config) VALUES
    ('vw_ad_page', 'mch_approval_document', 'Approval Dashboard (composed)', 'page', 6,
     '{"children":[
        {"view":"vw_ad_dashboard","title":"Summary"},
        {"view":"vw_ad_pending","title":"Pending Documents","layout":"main"},
        {"content":{"type":"text","text":"Recent Activity requires a record-history/activity-timeline View (CAP-R04), not built yet -- this section is a named placeholder, not invented data. See capability-registry.md CAP-R04 and CAP-V10 Tier 2 rows."},"title":"Recent Activity","layout":"aside"}
     ]}')
ON CONFLICT (id) DO NOTHING;
