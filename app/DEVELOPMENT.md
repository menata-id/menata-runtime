# Development Guide

> This document describes how to set up and run the real Menata Runtime application.
> **Current status: blueprint stage** — no server exists to run yet. This is a stub mirroring
> `prototype/go/DEVELOPMENT.md`'s own structure, filled in as the development plan ports real code.

---

## Prerequisites (target, once code is ported)

**Core tech stack** — the actual architecture-defining choices, unchanged from `prototype/go`'s
own proven stack (`prototype/go/docs/decisions/001-techstack.md`). See `ARCHITECTURE.md`'s
"Client-side JavaScript policy" for why HTMX's presence here is what keeps this list this short:

| Tool | Version | Purpose |
|------|---------|---------|
| Go | 1.25.0 | Runtime engine |
| PostgreSQL | 14+ | Runtime Metadata + Business Data storage |
| templ | v0.3.1020 | Type-safe HTML templates, compiled |
| HTMX | 2.0.4 | Server-driven partial page updates — the reason this stack needs almost no client-side JavaScript at all |

**Build-time tooling only — not part of the running application, never shipped to the browser:**

| Tool | Version | Purpose |
|------|---------|---------|
| Node.js | 20+ | Runs the Tailwind CSS CLI at build time (`make build-css`) only |

`go.mod` declares no dependencies yet — `prototype/go/go.mod`'s own 8 direct dependencies
(`github.com/a-h/templ`, `github.com/chai2010/webp`, `github.com/go-chi/chi/v5`,
`github.com/jackc/pgx/v5`, `github.com/joho/godotenv`, `github.com/pdfcpu/pdfcpu`,
`golang.org/x/crypto`, `golang.org/x/image`) get added via `go get` as each is actually needed
during the port, not declared speculatively ahead of it.

**Status update (2026-09-06):** Phase 0/1 landed `github.com/jackc/pgx/v5` and
`golang.org/x/crypto`, pinned to the same versions `prototype/go/go.mod` uses. The remaining six
land as Phase 2/3 need them.

## No Docker/containers here either

Same real resource constraint as `prototype/go`'s own deployment (`prototype/go/DEVELOPMENT.md`'s
own note) — this host runs several other apps' production instances already; a container
runtime's overhead isn't free on a host this size. Deployment stays a plain compiled binary via
`server-manager.sh`, same pattern, once there's a binary to deploy.

## Installation

**Status update (2026-09-06, Phase 3): a real server exists.** `cmd/server/main.go` is ported —
`make dev` (or `make build && make run`) boots it, same `.env` shape as `prototype/go`
(`DATABASE_URL`, `PORT`, `SECURE_COOKIES`; see below for `app/`'s own database). Uses a different
`PORT` than `prototype/go`'s own dev deployment (4000) to run both side by side during the port —
4001 is a reasonable default until Phase 6's cutover decision.

**Status update (2026-09-06): database setup (Phase 1).** `app/` uses its own database, separate
from `prototype/go`'s `menata_runtime` (never point `app/`'s tooling at that database — it's the
live `menata.app` dev deployment's own data). Local dev:

```bash
cd app
make migrate-up       # goose (see Makefile) -- applies migrations/*.sql, tracked in goose_db_version
make seed             # every seeds/NNN_*.sql file (all 37, per ROADMAP.md Phase 4) except
                       # 023/025/028, applied dynamically by the specific conformance tests
                       # that need them -- see Makefile's own seed target comment
```

**Status update (2026-09-06, Phase 4): dedicated database role, not `postgres`.** Phase 0–3's own
verification used the shared `postgres` superuser directly against a `menata_app` database created
ad hoc — expedient, but exactly the practice `prototype/go/DEVELOPMENT.md`'s own "Database role"
section warns against (that role is cluster-wide on this shared host, running other apps'
production databases too — it has already caused a real outage for another app once). Corrected
here, same pattern prototype/go already established for `menata_runtime`/`menata_runtime_app`:

```sql
CREATE ROLE menata_app_owner WITH LOGIN PASSWORD '<strong-random-password>' NOSUPERUSER NOCREATEDB NOCREATEROLE;
CREATE DATABASE menata_app OWNER menata_app_owner;
```

`.env`'s `DATABASE_URL` uses this role, never `postgres`:

```env
DATABASE_URL=postgres://menata_app_owner:<password>@localhost:5432/menata_app?sslmode=disable
PORT=4001
SECURE_COOKIES=false
```

Owning the database gives this role everything `migrate-up`/`seed` need (including
`CREATE EXTENSION pgcrypto`, a trusted extension installable without superuser) — verified by
re-running the full migrate-up → seed → boot → conformance cycle end to end under this role alone,
219/219 still passing. **Always use a dedicated role scoped to one database — never the shared
`postgres` account — on any host that isn't exclusively yours**, the same rule
`prototype/go/DEVELOPMENT.md` states for exactly this reason.

`make migrate-down` reverses every migration (verified round-trip clean during Phase 1).
`migrations/manual/009_workspace_isolation_rls.sql` is deliberately excluded from `migrate-up`
(own version table, `goose_manual_db_version`) — apply only via `make migrate-rls-cutover`, and
only at `ROADMAP.md`'s own Phase 6 cutover, same restriction `prototype/go`'s own
`migrate-rls-cutover` Makefile target already documents for its identical migration.

## Verification

`cd app && go build ./... && go vet ./...` — passes clean.

**Status update (2026-09-06):** `go test ./...` now also runs real tests: `internal/model`'s
ported `model_test.go` (Phase 0), and `internal/metadata`'s `TestLoadAllAgainstApprovalCase`
(Phase 1 — needs `DATABASE_URL` pointed at a database that already ran `make migrate-up && make
seed` above; skips itself otherwise).
