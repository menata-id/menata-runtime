-- seeds/056_activity_log_drift_fix.sql
-- Two real data-drift issues caught live while redeploying 17p, fixed
-- here rather than by editing already-shipped seed files' own INSERTs
-- (053_composable_data_plane_lab.sql's own idempotency-guard bug is
-- fixed in place, since that's a genuine correctness bug in an
-- operational script, not a documented historical claim -- but the
-- resulting bad DATA already written to a real, persistent database
-- needs its own corrective UPDATE, the same "append/patch, don't
-- silently assume a fresh state" precedent 051/053/054/055 already set).
--
-- 1. vw_ad_page's own Children array accumulated a stale duplicate
--    Metric entry (component=Metric, dataset_id=ds_ad_steps_by_document,
--    title="Steps by Decision") from a second `make seed` run before
--    053's own guard fix above landed. Removed here, guarded by
--    existence so re-running this is a safe no-op once fixed.
-- 2. vw_ad_detail already had a real Children entry
--    ({"view":"vw_ad_progress"}) that predates seeds/055_activity_log.sql
--    and isn't reflected in 004_approval.sql's own checked-in INSERT --
--    055's own guard (`NOT (config ? 'children')`) correctly declined to
--    touch it rather than silently overwrite it, but that also meant
--    vw_ad_activity never got appended there at all. Appended here
--    instead, preserving the pre-existing vw_ad_progress entry.
-- Status update (2026-09-12, caught live applying this very fix): the
-- first version of this UPDATE filtered with
-- `WHERE NOT (elem->>'component' = 'Metric' AND elem->>'dataset_id' = '...')`.
-- For every entry with no "component" key at all (Summary, Pending
-- Documents, Recent Activity), `elem->>'component' = 'Metric'` is SQL
-- NULL, not false -- three-valued logic, not the boolean two-valued logic
-- this looked like at a glance. `NOT (NULL AND ...)` is also NULL, and
-- Postgres's WHERE excludes NULL rows exactly like it excludes false
-- ones -- so the very first run of this fix, applied directly against
-- the real menata_runtime database, silently dropped every non-Metric
-- Children entry from vw_ad_page (confirmed live: the config collapsed
-- to just the one Metric entry). Restored by hand immediately, and
-- rewritten below with IS DISTINCT FROM, which never itself returns
-- NULL, before re-applying.
-- Repair step for the exact corruption the bug above produced when this
-- file's first version ran directly against the real menata_runtime
-- database: vw_ad_page's Children collapsed to just the surviving Metric
-- entry, losing Summary/Pending Documents/Recent Activity entirely.
-- Guarded on that precise signature (Summary's own vw_ad_dashboard entry
-- missing) so this is a safe no-op everywhere else -- a fresh schema, an
-- already-correct database, or a second run here.
UPDATE views
   SET config = '{"children":[
         {"view":"vw_ad_dashboard","title":"Summary"},
         {"view":"vw_ad_pending","title":"Pending Documents","layout":"main"},
         {"view":"vw_ad_activity","title":"Recent Activity","layout":"aside"},
         {"title":"Total Approval Steps","component":"Metric","dataset_id":"ds_ad_total_steps"}
       ]}'::jsonb
 WHERE id = 'vw_ad_page'
   AND NOT (config->'children' @> '[{"view":"vw_ad_dashboard"}]'::jsonb);

UPDATE views
   SET config = jsonb_set(
         config,
         '{children}',
         COALESCE(
           (SELECT jsonb_agg(elem) FROM jsonb_array_elements(config->'children') elem
            WHERE elem->>'component' IS DISTINCT FROM 'Metric'
               OR elem->>'dataset_id' IS DISTINCT FROM 'ds_ad_steps_by_document'),
           '[]'::jsonb
         )
       )
 WHERE id = 'vw_ad_page'
   AND EXISTS (
     SELECT 1 FROM jsonb_array_elements(config->'children') elem
     WHERE elem->>'component' = 'Metric' AND elem->>'dataset_id' = 'ds_ad_steps_by_document'
   );

UPDATE views
   SET config = jsonb_set(config, '{children}', (config->'children') || '[{"view":"vw_ad_activity","title":"Activity"}]'::jsonb)
 WHERE id = 'vw_ad_detail'
   AND NOT (config->'children' @> '[{"view":"vw_ad_activity"}]'::jsonb);
