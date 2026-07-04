BINARY     = bin/server
TEMPL      = $(shell go env GOPATH)/bin/templ
DB_URL    ?= $(shell grep DATABASE_URL .env 2>/dev/null | cut -d= -f2-)

.PHONY: build run dev generate build-css migrate-up seed test

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

migrate-up:
	psql "$(DB_URL)" -f migrations/001_metadata_schema.sql
	psql "$(DB_URL)" -f migrations/002_data_schema.sql

seed:
	psql "$(DB_URL)" -f seeds/001_design_request.sql

test:
	go test ./...
