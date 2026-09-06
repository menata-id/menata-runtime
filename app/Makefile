DB_URL     ?= $(shell grep DATABASE_URL .env 2>/dev/null | cut -d= -f2-)
GOOSE       = go run github.com/pressly/goose/v3/cmd/goose@v3.28.0
TEMPL       = go run github.com/a-h/templ/cmd/templ@v0.3.1020

.PHONY: build run dev generate build-css check-css migrate-up migrate-down migrate-status migrate-rls-cutover migrate-rls-cutover-down seed conformance test local-ci

build:
	go build -o bin/server ./cmd/server

run: build
	./bin/server

dev:
	go run ./cmd/server

generate:
	$(TEMPL) generate

build-css:
	npm run build:css

# Verifies every color utility class referenced in internal/**/*.templ,
# internal/**/*.go, cmd/**/*.go actually exists in the just-built
# output.css -- run build-css first, this only checks (see
# scripts/check-css-classes.sh's own header for why).
check-css: build-css
	./scripts/check-css-classes.sh

# Real migration tooling (app/ROADMAP.md Phase 1 gap fix -- prototype/go's own
# migrations/ are flat numbered .sql files with no version-tracking table and
# no rollback). goose tracks applied versions in goose_db_version and supports
# up/down for every file below migrations/ -- see each file's own
# "-- +goose Down" section.
migrate-up:
	$(GOOSE) -dir migrations postgres "$(DB_URL)" up

migrate-down:
	$(GOOSE) -dir migrations postgres "$(DB_URL)" down

migrate-status:
	$(GOOSE) -dir migrations postgres "$(DB_URL)" status

# CAP-X06 enforcement flip -- deliberately kept out of migrations/ (a plain
# `migrate-up` must never apply it) and tracked in its own version table
# (goose_manual_db_version), same reasoning as prototype/go's own
# migrate-rls-cutover Makefile target: apply only at final cutover, in the
# same deploy window as restarting with the binary that sets
# app.workspace_id per request. See migrations/manual/009_workspace_isolation_rls.sql's
# own header.
migrate-rls-cutover:
	$(GOOSE) -dir migrations/manual -table goose_manual_db_version postgres "$(DB_URL)" up

migrate-rls-cutover-down:
	$(GOOSE) -dir migrations/manual -table goose_manual_db_version postgres "$(DB_URL)" down

# Every seeds/NNN_*.sql file, in order -- mirrors prototype/go's own Makefile
# exactly. Fields/constraints/permissions/views use ON CONFLICT DO NOTHING
# and are safe to re-run; event_actions has no natural key and is NOT --
# re-running this against an already-seeded database duplicates every
# event_action row. Safe on a fresh database only.
#
# seeds/023_reload_lab.sql, 025_change_policy_activate.sql, and
# 028_lift_lab.sql are deliberately NOT here -- applied dynamically by the
# specific conformance tests that need them (conformance/tests/
# 050_process_overlay.sh, 060_ui_cluster.sh), not at boot time.
seed:
	psql "$(DB_URL)" -f seeds/001_design_request.sql
	psql "$(DB_URL)" -f seeds/002_leave_request.sql
	psql "$(DB_URL)" -f seeds/003_hr_employee.sql
	psql "$(DB_URL)" -f seeds/004_approval.sql
	psql "$(DB_URL)" -f seeds/005_complaint.sql
	psql "$(DB_URL)" -f seeds/006_second_workspace.sql
	psql "$(DB_URL)" -f seeds/007_authentication.sql
	psql "$(DB_URL)" -f seeds/008_journal_entry.sql
	psql "$(DB_URL)" -f seeds/009_action_lab.sql
	psql "$(DB_URL)" -f seeds/010_views_lab.sql
	psql "$(DB_URL)" -f seeds/011_record_lifecycle_lab.sql
	psql "$(DB_URL)" -f seeds/012_permissions_lab.sql
	psql "$(DB_URL)" -f seeds/013_event_sources_lab.sql
	psql "$(DB_URL)" -f seeds/014_integration_lab.sql
	psql "$(DB_URL)" -f seeds/015_workspace_services_lab.sql
	psql "$(DB_URL)" -f seeds/016_infra_lab.sql
	psql "$(DB_URL)" -f seeds/017_field_types_lab.sql
	psql "$(DB_URL)" -f seeds/019_overlay_lab.sql
	psql "$(DB_URL)" -f seeds/020_requirement_lab.sql
	psql "$(DB_URL)" -f seeds/021_sla_lab.sql
	psql "$(DB_URL)" -f seeds/022_quorum_lab.sql
	psql "$(DB_URL)" -f seeds/024_change_policy_lab.sql
	psql "$(DB_URL)" -f seeds/026_quorum_declarative_lab.sql
	psql "$(DB_URL)" -f seeds/027_case9_completion_lab.sql
	psql "$(DB_URL)" -f seeds/029_sla_badge_lab.sql
	psql "$(DB_URL)" -f seeds/030_resource_calendar_lab.sql
	psql "$(DB_URL)" -f seeds/031_typeahead_lab.sql
	psql "$(DB_URL)" -f seeds/032_kanban_lab.sql
	psql "$(DB_URL)" -f seeds/033_follow_lab.sql
	psql "$(DB_URL)" -f seeds/034_groups_lab.sql
	psql "$(DB_URL)" -f seeds/035_pdf_signature_lab.sql
	psql "$(DB_URL)" -f seeds/036_coord_placement_lab.sql
	psql "$(DB_URL)" -f seeds/037_decision_stepper_lab.sql
	psql "$(DB_URL)" -f seeds/038_group_approver_lab.sql

# The real test suite for HTTP/behavioral correctness -- requires migrate-up
# + seed already applied and the server running (`make dev` in another
# terminal). Export DATABASE_URL to also run T19 (see conformance/README.md).
conformance:
	./conformance/run.sh

# Go unit tests -- the ported Phase 0/2 tests (internal/model, internal/
# executor) plus internal/metadata's TestLoadAllAgainstApprovalCase (needs
# DATABASE_URL, skips itself otherwise). Not a substitute for conformance --
# this codebase's real proof is the black-box HTTP suite above.
test:
	go test ./...

# Zero-cost local CI gate -- see scripts/local-ci.sh's own header.
local-ci:
	./scripts/local-ci.sh
