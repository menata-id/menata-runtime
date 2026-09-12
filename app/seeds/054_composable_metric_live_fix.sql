-- seeds/054_composable_metric_live_fix.sql
-- CR-20/CR-21 (composable-runtime-roadmap.md 17l): 17k's own
-- seeds/053_composable_data_plane_lab.sql paired vw_ad_page's new
-- component:"Metric" Children entry with ds_ad_steps_by_document -- a
-- GROUPED Dataset (one Dimension, dim_decision). 17l's own fix to
-- internal/composable/ui.go (lowerComponentChild now routes Metric
-- through LowerMetric's existing semantic rule, "a grouped Dataset is a
-- breakdown, not a single metric") makes that pairing invalid.
-- ds_ad_steps_by_document itself stays exactly as declared -- a real,
-- legitimate Data-plane artifact, just not one appropriate for a Metric
-- binding. This seed adds a real, UNGROUPED, single-Measure Dataset
-- instead, and patches only the one Children entry's own dataset_id/
-- title -- append/patch, not a rewrite, same precedent
-- 051_dashboard_nav_supersede.sql's own UPDATE already set for correcting
-- already-seeded data.
INSERT INTO datasets (id, application_id, base_machine_id, name, position, config) VALUES
    ('ds_ad_total_steps', 'app_approval', 'mch_approval_step', 'Total Approval Steps', 1,
     '{"measures":[{"id":"mea_total_steps","aggregate":"count"}]}')
ON CONFLICT (id) DO NOTHING;

UPDATE views
   SET config = jsonb_set(
         jsonb_set(config, '{children,3,dataset_id}', '"ds_ad_total_steps"'),
         '{children,3,title}', '"Total Approval Steps"'
       )
 WHERE id = 'vw_ad_page'
   AND config->'children'->3->>'component' = 'Metric'
   AND config->'children'->3->>'dataset_id' = 'ds_ad_steps_by_document';
