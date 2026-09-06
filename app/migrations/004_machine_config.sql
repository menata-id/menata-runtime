-- +goose Up
-- 004_machine_config.sql
-- CAP-X03: machine-level config block.
--
-- Some Runtime Metadata needs a workspace-authored value that isn't a Field
-- of any record and isn't a Constraint/Permission/View either — it configures
-- how the *Machine itself* behaves. First need: Approval Document's Sequential
-- vs Parallel approval mode wiring (which field holds the mode, which Machine
-- holds its steps, which field on the steps points back) — already sketched,
-- commented out, in prototype/go/docs/examples/approval-document.yaml.
--
-- config: {"approval_mode_field": "fld_ad_approval_mode",
--           "steps_machine": "mch_approval_step",
--           "steps_parent_field": "fld_as_document"}
-- NULL/absent = no config, the default for every Machine that doesn't need one.

ALTER TABLE machines ADD COLUMN IF NOT EXISTS config JSONB;

-- +goose Down
ALTER TABLE machines DROP COLUMN IF EXISTS config;
