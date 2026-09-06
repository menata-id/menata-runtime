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
to match whatever name/credentials you actually created. **Create a role scoped to this database,
not the shared `postgres` superuser** — see "Database role" below for why.

```sql
CREATE ROLE menata_runtime_app WITH LOGIN PASSWORD '<strong-random-password>';
CREATE DATABASE menata_runtime OWNER menata_runtime_app;
```

#### Database role — why not `postgres`

On a box that only ever runs this prototype, connecting as the `postgres` superuser is harmless.
On a **shared host running other apps**, the `postgres` role is a single cluster-wide account —
its password is not per-database. Setting it here (`ALTER ROLE postgres WITH PASSWORD ...`, or a
`DATABASE_URL` that happens to use it) changes it for every app on the host, and has caused a
production outage for another app before. **Always use a dedicated role scoped to one database —
never the shared `postgres` account — on any host that isn't exclusively yours.** Ops/deployment
specifics for any shared host this runs on live outside this repo — check with whoever manages
that host.

### 6. Configure environment

Copy the example environment file:

```bash
cp .env.example .env
```

Edit `.env` to use the role/password you created in step 5:

```env
DATABASE_URL=postgres://menata_runtime_app:<password>@localhost:5432/menata_runtime?sslmode=disable
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

### 9. Install the local pre-push CI gate (2026-08-29, Study 33)

No GitHub Actions here — this repo runs on a free GitHub plan with Actions minutes exhausted, so
the "nothing broken reaches `main`" check runs locally instead, at push time, against a throwaway
isolated Postgres schema + throwaway server port (never touches the database or port this
prototype's own dev deployment uses). The gate script is tracked at `scripts/pre-push`
(repo root); git's own hook slot (`.git/hooks/`) is not tracked, so it needs a one-time copy after
every fresh clone:

```bash
cp scripts/pre-push .git/hooks/pre-push && chmod +x .git/hooks/pre-push
```

It only runs (`make local-ci` under the hood — see `prototype/go/scripts/local-ci.sh`) when a push
actually touches `prototype/go/`; a docs-only push skips it. Bypass with `git push --no-verify`
(standard git) or `SKIP_LOCAL_CI=1 git push` when you deliberately need to. Full rationale:
`../../benchmarks/025-architecture-worldclass-audit.md`.

**Correction (2026-09-05):** "no GitHub Actions here" above was written under a private-repo
billing assumption that doesn't hold for this repo specifically — `menata-runtime` is **public**,
and GitHub's own billing docs state standard GitHub-hosted runner minutes are free and unbounded
for public repositories regardless of account plan (the 2,000 min/month Free-plan quota only
applies to private repos). One GitHub Actions workflow now exists on that basis:
`.github/workflows/css-gate.yml` (`make check-css` — rebuilds Tailwind CSS and fails if any color
utility class referenced in `.templ`/`.go` source didn't make it into the compiled output, the
class of bug conformance's own HTTP-black-box scope can never catch), triggered on push/PR to
`prototype/go/`. The heavier full conformance suite stays on the local pre-push hook above, not
moved to Actions — nothing forces that move, and the local gate already works.

**Follow-up (2026-09-05, same day):** it moved anyway, additively — `.github/workflows/
conformance.yml` now runs the exact same `local-ci.sh` this section describes, against a real
Postgres service container instead of this machine's own local one. Gives a visible commit-
status/PR-check/run-history on GitHub, which a purely local hook never could; the hook itself is
unchanged and still installed for fast pre-push feedback before a push even leaves the machine.
Verified live twice: 219/219 passed each run, ~4 minutes. `roadmap.md` item 18's own follow-up
note and `benchmarks/025-architecture-worldclass-audit.md`'s F1 correction carry the full story.

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

## Dev Deployment (`menata.app`)

**Correction (2026-08-22):** earlier revisions of this section called this a "Production
Deployment." It is not — `menata.app` is a **development** deployment; there is no production
instance of this runtime yet. The operational cautions below (restart discipline, port
awareness, no-auto-restart policy) remain real and unchanged, though: the same host runs other
apps' genuine production instances (`/root/scripts/server-manager.sh status` lists them), so
carelessness here still risks THEIR production traffic even though this app has none of its own.

This prototype is live at **`https://menata.app`** (reassigned to this port 2026-07-12 as
`aksi.menata.id`; domain changed to `menata.app` the same day — the old domain permanently
redirects to the new one — see `/root/projects/MULTI-APP-GUIDE.md` for the full multi-app
picture on that host). On that host specifically:

- Runs from `/root/projects/menata-runtime/prototype/go` (no separate `/root/production/`
  copy — unlike some other apps on that host, dev path *is* the deployed path).
- `PORT=4000` in `.env` (not the `3100` default above), proxied by Caddy
  (`menata.app { reverse_proxy localhost:4000 ... }` in `/etc/caddy/Caddyfile`; `aksi.menata.id`
  is a separate `redir` block to `menata.app`).
- Restart via the sanctioned script, never a bare `go run` left in the background — a bare
  dev-mode process on port 4000 will squat on production traffic:
  ```bash
  cd /root/projects/menata-runtime/prototype/go
  go build -o bin/server ./cmd/server
  /root/scripts/server-manager.sh restart menata-runtime
  ```
- **Before running anything on port 4000 on that host** (or any port), check
  `/root/projects/MULTI-APP-GUIDE.md`'s port allocation map first — this host runs several
  unrelated apps, and a plain `kill`/`pkill` without checking what's actually listening has
  taken down another app's production instance before.

**No Docker/containers anywhere in this deployment, deliberately.** The host is a single shared
VPS with several other apps' own production instances already running on it (server-manager.sh
status lists them) — resources (RAM/CPU) are a real, named constraint, not a preference; a
container runtime's own overhead on top of that is not free on a host this size. Deployment stays
a plain compiled binary (`go build` + `nohup`/`server-manager.sh`), same as every other app
sharing this host. See `benchmarks/026-runtime-graduation-decision.md`'s 2026-09-06 addendum for
how this constraint feeds directly into the prototype-vs-fresh-start decision for `app/`.
- No auto-restart of any kind (systemd `Restart=`, cron, watchdog) — see
  `/root/docs/server-policies/NO-AUTO-RESTART-POLICY.md`. Manual restart only.
- **RLS is live on this database** (CAP-X06, `migrations/009_workspace_isolation_rls.sql`) —
  `records`/`record_events`/`notifications` all `FORCE ROW LEVEL SECURITY`. A regular `make
  migrate-up` does *not* re-run `009` (it's a deliberate one-time cutover, not part of that
  target — see the migration's own header) and doesn't need to; it's already applied here.
  Any *new* migration that touches these three tables must account for RLS already being on
  (e.g. a raw `psql` backfill needs `SELECT set_config('app.workspace_id', ..., true)` first,
  or it fails closed — `conformance/run.sh`'s T19/T42/T43/T52 needed this same fix, see their
  comments for the working pattern, including why a naive FROM-clause subquery version of it
  is unreliable).
- **`SECURE_COOKIES=true` is required in this `.env`** (CAP-X02) — the session cookie's
  `Secure` attribute must be set once traffic is real HTTPS (via Caddy); the `false` default
  in `.env.example` is for local `http://localhost` dev only, where a `Secure` cookie would be
  silently refused by the browser.

---

## Project Structure

```
prototype/go/
├── cmd/
│   └── server/
│       └── main.go          ← entry point
├── internal/
│   ├── auth/                  ← password hashing, session tokens, CSRF (CAP-X02)
│   ├── config/               ← env/config loading
│   ├── db/                   ← pgx pool setup
│   ├── interpreter/          ← indexed lookups over the Application Model;
│   │                            atomic swap on admin reload (CAP-X04)
│   ├── router/                ← HTTP routing (chi), routes derived from metadata
│   ├── handler/               ← HTTP handlers + cross-record orchestration
│   │                            (reference lookups, workflow actions, guards) --
│   │                            split into domain-scoped files, see ADR-006
│   ├── executor/               ← single-record event simulation + persistence
│   ├── constraint/             ← constraint/guard expression evaluation
│   ├── permission/             ← role → event authorization
│   ├── metadata/                ← Runtime Metadata loading + load-time validation
│   ├── model/                   ← Application Model (in-memory)
│   ├── store/                   ← Postgres-backed record/notification/user stores
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
| `make check-css` | Build Tailwind CSS, then fail if any color utility class referenced in `.templ`/`.go` source didn't make it into the compiled output (also runs as a GitHub Actions check, `.github/workflows/css-gate.yml`, on every push/PR touching `prototype/go/`) |
| `make test` | `go test ./...` — real unit tests as of 2026-09-05 (`internal/model`, `internal/handler`, `internal/executor`: `displayLabel`, `FindFieldByName`, `FindReferenceFieldTo`, CAP-A11 date arithmetic — the documented pure-function heuristics, fast feedback the HTTP black-box suite below can't give). Most packages still report "no test files" — this covers the heuristics named above, not general coverage; use `make conformance` for everything else |

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
