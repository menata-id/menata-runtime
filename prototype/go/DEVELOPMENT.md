# Development Guide

> This document describes how to set up and run the Menata Runtime Prototype locally.

---

## Prerequisites

| Tool | Version | Purpose |
|------|---------|---------|
| Go | 1.22+ | Runtime engine |
| PostgreSQL | 14+ | Runtime Metadata + Business Data storage |
| Node.js | 18+ | Tailwind CSS build |
| templ | latest | Type-safe HTML templates |

---

## Installation

### 1. Clone the repository

```bash
git clone https://github.com/menata-id/menata-runtime.git
cd menata-runtime/prototype/go
```

### 2. Install Go dependencies

```bash
go mod tidy
```

### 3. Install templ

```bash
go install github.com/a-h/templ/cmd/templ@latest
```

### 4. Install Node dependencies

```bash
npm install
```

### 5. Set up PostgreSQL

Create a database matching `.env.example`'s name (`menata_runtime`) — or edit `.env` in the next step
to match whatever name/credentials you actually created:

```sql
CREATE DATABASE menata_runtime;
```

### 6. Configure environment

Copy the example environment file:

```bash
cp .env.example .env
```

`.env.example` already has working defaults:

```env
DATABASE_URL=postgres://postgres:password@localhost:5432/menata_runtime?sslmode=disable
PORT=3100
```

### 7. Run database migrations

```bash
make migrate-up
```

### 8. Seed example Runtime Metadata

```bash
make seed
```

---

## Running the Prototype

### Build CSS

```bash
npm run build:css
```

### Generate Templ files

```bash
templ generate
```

### Build and run

```bash
make build
make run
```

Or in development mode with live reload:

```bash
make dev
```

The application will be available at `http://localhost:3100`.

---

## Project Structure

```
prototype/go/
├── cmd/
│   └── server/
│       └── main.go          ← entry point
├── internal/
│   ├── config/               ← env/config loading
│   ├── db/                   ← pgx pool setup
│   ├── interpreter/          ← indexed lookups over the Application Model
│   ├── router/                ← HTTP routing (chi), routes derived from metadata
│   ├── handler/               ← HTTP handlers + cross-record orchestration
│   │                            (reference lookups, workflow actions, guards)
│   ├── executor/               ← single-record event simulation + persistence
│   ├── constraint/             ← constraint/guard expression evaluation
│   ├── permission/             ← role → event authorization
│   ├── metadata/                ← Runtime Metadata loading + load-time validation
│   ├── model/                   ← Application Model (in-memory)
│   └── ui/                      ← Templ templates (.templ + generated _templ.go)
├── web/
│   └── static/
│       └── css/              ← Tailwind output (not built in every environment --
│                                see Troubleshooting)
├── migrations/               ← Database migrations, applied in numeric order
├── seeds/                    ← Example Runtime Metadata, one file per Case
├── conformance/               ← the real test suite (run.sh) -- see conformance/README.md
├── docs/                     ← ADRs (decisions/), Menata Language examples (examples/)
├── .env.example
├── Makefile
├── go.mod
└── go.sum
```

See `CLAUDE.md` for architectural patterns, conventions, and gotchas established while
implementing capabilities against this structure — read it before adding a new capability.

---

## Makefile Commands

| Command | Description |
|---------|-------------|
| `make build` | Build the server binary |
| `make run` | Run the server |
| `make dev` | Run with `go run` (no live reload — restart manually after a code change) |
| `make migrate-up` | Apply every migration, in order |
| `make seed` | Load every example seed, in order (fresh database only — see the Makefile's own comment on `event_actions` re-run safety) |
| `make conformance` | Run the real test suite against a running server |
| `make generate` | Run templ generate |
| `make build-css` | Build Tailwind CSS |
| `make test` | `go test ./...` — reports "no test files" today; use `make conformance` |

---

## Adding a New Machine

1. Define Business Knowledge using Menata Language (see `docs/examples/`)
2. Create Runtime Metadata (YAML) describing the machine
3. Load the metadata via `make seed` or the admin interface
4. The runtime automatically realizes the new machine as a running application

No Go code changes are required to add a new machine.

---

## Troubleshooting

**Runtime Metadata fails to load**

Check validation errors in the server log.

Invalid Runtime Metadata is rejected.

The server will report the specific validation failure.

**CSS changes not reflected — or `web/static/css/output.css` doesn't exist at all**

Run `npm install && npm run build:css`. Neither step runs automatically; a fresh checkout with
`node_modules` never installed will serve pages with a 404'd stylesheet (unstyled but otherwise
functional HTML) until this is done once.

**Templ changes not reflected**

Run `make generate` (or `templ generate` directly) to regenerate the `_templ.go` files, then rebuild.
Pin the CLI version to what `go.mod` already declares (`go run github.com/a-h/templ/cmd/templ@vX.Y.Z
generate`) to avoid a version-mismatch warning against `@latest`.

**Can't restart the server — "address already in use," but nothing is running**

`go run` compiles to a temp binary whose path is not stable (`/tmp/go-build*/.../exe/server` one run,
`~/.cache/go-build/.../server` the next) — `pkill -f` on a path pattern misses a server started under
a different path. Find and kill by the port instead:

```bash
kill -9 $(ss -ltnp 2>/dev/null | grep :4000 | grep -oP 'pid=\K[0-9]+')
```
