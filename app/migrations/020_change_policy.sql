-- +goose Up
-- CAP-W07: a Constraint may declare which in-flight records it applies to
-- (new_records / records_in_states / all_records), compiled into its own
-- Condition at load time (internal/metadata/compile.go's compileChangePolicies).
ALTER TABLE constraints ADD COLUMN IF NOT EXISTS change_policy JSONB;

-- +goose Down
ALTER TABLE constraints DROP COLUMN IF EXISTS change_policy;
