-- seeds/053_composable_data_plane_lab.sql
-- CR-21 (composable-runtime-roadmap.md 17k): the first REAL, loadable
-- Data-plane Dataset/Query rows -- metadata/loader.go's loadDatasets/
-- loadQueries load these into model.Dataset/model.Query, validated by
-- metadata/validate.go's validateDatasets/validateQueries, and lowered by
-- internal/composable's BuildDatasetFromDeclaredDataset into the exact
-- same composable.Dataset shape the existing View/Report/Dashboard-
-- inferred adapters already produce.
--
-- ds_ad_steps_by_document declares a Dataset over the existing Approval
-- Step machine (mch_approval_step, seeds/004_approval.sql) -- one real
-- Relation (rel_document, via fld_as_document, the reference already
-- pointing back at mch_approval_document), one Dimension (fld_as_decision)
-- and one count Measure. qry_ad_steps_by_decision is a real Query
-- projecting both, sorted by count descending -- the shape
-- runtime-metadata-schema.md's own "Composable Schema Extensions" section
-- sketched (ds_revenue_by_region/qry_top_regions) but on real seeded ids
-- instead of an illustrative one.
INSERT INTO datasets (id, application_id, base_machine_id, name, position, config) VALUES
    ('ds_ad_steps_by_document', 'app_approval', 'mch_approval_step', 'Approval Steps by Decision', 0,
     '{"relations":[{"id":"rel_document","via":"fld_as_document"}],
       "dimensions":[{"id":"dim_decision","field":"fld_as_decision"}],
       "measures":[{"id":"mea_count","aggregate":"count"}]}')
ON CONFLICT (id) DO NOTHING;

INSERT INTO queries (id, dataset_id, name, position, config) VALUES
    ('qry_ad_steps_by_decision', 'ds_ad_steps_by_document', 'Steps by Decision', 0,
     '{"projection":["dim_decision","mea_count"],
       "sort":{"field":"mea_count","direction":"desc"}}')
ON CONFLICT (id) DO NOTHING;

-- Experience-plane closure (CR-21, 17k Part B): one additional Children
-- entry on the existing vw_ad_page (seeds/050_composed_dashboard.sql),
-- appended via jsonb concatenation rather than editing that file's own
-- INSERT -- "append, don't rewrite" applied to seed data the same way
-- 051_dashboard_nav_supersede.sql's own UPDATE already does. Additive
-- slot only: vw_ad_page's existing three Children entries (Summary,
-- Pending Documents, Recent Activity) are untouched, so
-- 250_composed_dashboard.sh's own existing assertions about them still
-- hold. component:"Metric" (RequiresDataset, no required Properties,
-- internal/composable/component.go's own componentRegistry) bound to
-- ds_ad_steps_by_document -- the first real metadata proving a Children
-- entry can declare "render Component X bound to Dataset Y" directly,
-- instead of only ever a View or static Content.
-- Status update (2026-09-12, caught live redeploying 17p): the original
-- guard below matched on this entry's own dataset_id
-- (ds_ad_steps_by_document) -- but seeds/054_composable_metric_live_fix.sql
-- (the very next seed in sequence) RENAMES that same entry's dataset_id
-- to ds_ad_total_steps. On any SECOND full `make seed` run against a
-- database that already went through 053+054 once, this guard no longer
-- recognized its own prior work (the dataset_id it was checking for was
-- already renamed away) and appended a second, stale duplicate --
-- confirmed live on menata_runtime's own vw_ad_page after a second
-- deployment. An isolated throwaway schema (this repo's own standard
-- test pattern) never catches this class of bug: each one only ever
-- runs `make seed` once before being torn down. Fixed to guard on "any
-- Metric component entry already exists at all", which survives 054's
-- own later rename.
UPDATE views
   SET config = jsonb_set(
         config,
         '{children}',
         (config->'children') || '[{"component":"Metric","dataset_id":"ds_ad_steps_by_document","title":"Steps by Decision"}]'::jsonb
       )
 WHERE id = 'vw_ad_page'
   AND NOT EXISTS (
     SELECT 1 FROM jsonb_array_elements(config->'children') AS c
     WHERE c->>'component' = 'Metric'
   );
