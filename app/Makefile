DB_URL     ?= $(shell grep DATABASE_URL .env 2>/dev/null | cut -d= -f2-)
GOOSE       = go run github.com/pressly/goose/v3/cmd/goose@v3.28.0

.PHONY: build run dev migrate-up migrate-down migrate-status migrate-rls-cutover migrate-rls-cutover-down seed test

build:
	go build -o bin/server ./cmd/server

run: build
	./bin/server

dev:
	go run ./cmd/server

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

# Every seeds/NNN_*.sql file, in order -- mirrors prototype/go's own Makefile.
# Grows as more seed files graduate (app/ROADMAP.md Phase 4 ports all 38).
seed:
	psql "$(DB_URL)" -f seeds/001_design_request.sql
	psql "$(DB_URL)" -f seeds/004_approval.sql

test:
	go test ./...
