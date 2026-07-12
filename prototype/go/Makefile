BINARY     = bin/server
TEMPL      = $(shell go env GOPATH)/bin/templ
DB_URL    ?= $(shell grep DATABASE_URL .env 2>/dev/null | cut -d= -f2-)

.PHONY: build run dev generate build-css migrate-up migrate-rls-cutover seed conformance test

build:
	go build -o $(BINARY) ./cmd/server

run: build
	./$(BINARY)

dev:
	go run ./cmd/server

generate:
	$(TEMPL) generate

build-css:
	npm run build:css

# Every migrations/NNN_*.sql file, in order. Add new ones here as they're
# created -- there is no auto-discovery.
migrate-up:
	psql "$(DB_URL)" -f migrations/001_metadata_schema.sql
	psql "$(DB_URL)" -f migrations/002_data_schema.sql
	psql "$(DB_URL)" -f migrations/003_event_conditions.sql
	psql "$(DB_URL)" -f migrations/004_machine_config.sql
	psql "$(DB_URL)" -f migrations/005_notifications.sql
	psql "$(DB_URL)" -f migrations/006_permission_extensions.sql
	psql "$(DB_URL)" -f migrations/007_audit_logging.sql
	psql "$(DB_URL)" -f migrations/008_workspace_isolation_schema.sql
	psql "$(DB_URL)" -f migrations/010_authentication.sql

# CAP-X06 enforcement flip -- deliberately NOT part of migrate-up. Apply only
# at final cutover, in the same deploy window as restarting with the binary
# that sets app.workspace_id per request (see migrations/009's own header).
# Running this ahead of that binary being live breaks every query against
# records/record_events/notifications immediately (RLS fails closed).
migrate-rls-cutover:
	psql "$(DB_URL)" -f migrations/009_workspace_isolation_rls.sql

# Every seeds/NNN_*.sql file, in order. Fields/constraints/permissions/views
# use ON CONFLICT DO NOTHING and are safe to re-run; event_actions has no
# natural key and is NOT -- re-running this against an already-seeded
# database duplicates every event_action row. Safe on a fresh database only.
seed:
	psql "$(DB_URL)" -f seeds/001_design_request.sql
	psql "$(DB_URL)" -f seeds/002_leave_request.sql
	psql "$(DB_URL)" -f seeds/003_hr_employee.sql
	psql "$(DB_URL)" -f seeds/004_approval.sql
	psql "$(DB_URL)" -f seeds/005_complaint.sql
	psql "$(DB_URL)" -f seeds/006_second_workspace.sql
	psql "$(DB_URL)" -f seeds/007_authentication.sql
	psql "$(DB_URL)" -f seeds/008_journal_entry.sql

# The real test suite (there are no Go unit tests in this codebase -- every
# capability's proof is here, per capability-registry.md's Proof column).
# Requires: migrate-up + seed already applied, server already running
# (`make dev` in another terminal). Export DATABASE_URL to also run T19,
# which needs it for a documented test-fixture exception -- see
# conformance/README.md.
conformance:
	./conformance/run.sh

# No Go unit tests exist yet -- this reports "no test files", not a pass.
# Use `make conformance` for the real suite.
test:
	go test ./...
