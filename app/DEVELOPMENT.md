# Development Guide

> This document describes how to set up and run the real Menata Runtime application.
> **Current status: blueprint stage** — no server exists to run yet. This is a stub mirroring
> `prototype/go/DEVELOPMENT.md`'s own structure, filled in as the development plan ports real code.

---

## Prerequisites (target, once code is ported)

Same as `prototype/go`'s own proven stack — nothing here changes as part of graduating:

| Tool | Version | Purpose |
|------|---------|---------|
| Go | 1.25.0 | Runtime engine |
| PostgreSQL | 14+ | Runtime Metadata + Business Data storage |
| Node.js | 20+ | Tailwind CSS build |
| templ | v0.3.1020 | Type-safe HTML templates |

`go.mod` declares no dependencies yet — `prototype/go/go.mod`'s own 8 direct dependencies
(`github.com/a-h/templ`, `github.com/chai2010/webp`, `github.com/go-chi/chi/v5`,
`github.com/jackc/pgx/v5`, `github.com/joho/godotenv`, `github.com/pdfcpu/pdfcpu`,
`golang.org/x/crypto`, `golang.org/x/image`) get added via `go get` as each is actually needed
during the port, not declared speculatively ahead of it.

## No Docker/containers here either

Same real resource constraint as `prototype/go`'s own deployment (`prototype/go/DEVELOPMENT.md`'s
own note) — this host runs several other apps' production instances already; a container
runtime's overhead isn't free on a host this size. Deployment stays a plain compiled binary via
`server-manager.sh`, same pattern, once there's a binary to deploy.

## Installation

Not yet meaningful — no `cmd/server/main.go` exists yet. This section fills in as the development
plan ports `prototype/go/cmd/server/main.go`.

## Verification

`cd app && go build ./... && go vet ./...` — the only thing there is to verify right now, and it
passes clean against the current doc.go-only package skeleton.
