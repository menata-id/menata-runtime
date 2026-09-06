# Development Plan & Roadmap

The blueprint (`ARCHITECTURE.md`) says what `app/` is built from and why. This document says
**in what order** it actually gets built, and what "done" means at each step — phased so every
step ends with something real to verify, not a big-bang port of ~12,753 lines followed by one
giant test at the end.

Full decision record behind all of this: `../benchmarks/026-runtime-graduation-decision.md`
(Study 34).

**Not the same document as root [`roadmap.md`](../roadmap.md).** That file is the
capability-*discovery* method and evidence log (why a capability exists, Studies 1–34); this file
is the actual build sequence for `app/`'s own code. The filenames differ only by case
(`roadmap.md` vs `ROADMAP.md`) — match by full path, not basename.

---

## Principles carried forward from `prototype/go`'s own discipline

- **Ratchet, never regress** — once a capability is ported and conformance-proven here, it must
  never regress. Same rule `roadmap.md`'s own Principles section already states for the discovery
  process; it applies to this port too.
- **Prove, don't assume** — each phase below ends with something runnable or testable, not just
  "the code compiles."
- **Pure move where the blueprint doesn't call for a change** — most packages graduate verbatim
  (import path change only: `menata.id/runtime` → `menata.id/app`). Only `internal/metadata`
  (the loader.go split) and `internal/store`/`internal/storage` (the file-upload split) are
  restructured; everything else is copied, not rewritten.
- **Named gaps get addressed at their cheapest moment, not ignored or front-loaded.** Each
  Study 34 §5 gap lands in the specific phase below where fixing it is cheapest — before real data
  or consumers exist to migrate away from, not as a bolt-on afterward and not solved prematurely
  before there's real code to attach it to.

---

## Phase 0 — Foundation (leaf packages, no server yet)

**Port:** `model`, `auth`, `db`, `config` verbatim (0 internal dependencies, Study 34 §7 Method 2)
— import path change only. `store` ports too, but with its file-upload byte-handling split out
into the new `internal/storage` package (interface first, local-disk backend behaviorally
identical to `prototype/go`'s own `uploads/`) — the one deliberate restructuring at this phase,
per `ARCHITECTURE.md`'s "New package" section.

**Verify:** `go build ./... && go vet ./...` clean. Port `internal/model/model_test.go` unchanged
and confirm it still passes — the first ported test, and the cheapest possible regression check.

**Milestone:** `app/` compiles as a real module with 5 working leaf-layer packages and a working
storage abstraction. No server yet.

**Status update (2026-09-06):** Done. `model`/`auth`/`db`/`config`/`store` ported verbatim
(byte-identical diff against `prototype/go` apart from two redundant package-doc comments already
covered by `doc.go`), `internal/storage`'s `Store` interface + `LocalDisk` backend written new.
`go build ./... && go vet ./...` clean; `internal/model/model_test.go` passes. Caddy/`menata.app`
untouched, per this roadmap's own Phase 6. Proceeding to Phase 1.

## Phase 1 — Metadata pipeline

**Port:** `internal/metadata`, already split into `loader.go` (per-table `Load*`), `validate.go`
(`validateOperators`/`validateReferences`), `compile.go` (unchanged, joins the split) —
`internal/interpreter`.

**Gap addressed here:** migrations. `prototype/go`'s own flat numbered `.sql` files (no
version-tracking, no rollback — Study 34 §5) convert to a real migration tool's up/down pairs
(golang-migrate, goose, or Atlas — pick one when this phase is actually run, no case forces the
choice yet) here, before any real seeded data exists in `app/`'s own database to migrate away
from. This is the cheapest point in the whole port to make this change; deferring it means
converting a database with real rows in it later instead of an empty one now.

**Verify:** a fresh migrate-up + one real seed file (`seeds/004_approval.sql`'s own Case 3,
Approval — not all 38 yet) followed by `Loader.LoadAll` producing a correct in-memory Application
Model. A small verification script or test is enough; no HTTP server needed yet.

**Milestone:** metadata loads from Postgres into memory correctly, on the new migration tooling.

**Status update (2026-09-06):** Done. `internal/metadata` split as planned (loader.go/validate.go/
compile.go, function-inventory-verified identical to `prototype/go`'s pre-split content, plus
`materialize.go` ported unchanged); `internal/interpreter` ported verbatim. Migration tool: goose
(chosen over golang-migrate/Atlas — pure-Go CLI via `go run`, same no-added-go.mod-dependency
pattern this project's `templ generate` already uses). All 24 `prototype/go/migrations/*.sql`
converted to goose's Up/Down format (23 in `migrations/`, `009_workspace_isolation_rls.sql` kept
in `migrations/manual/` with its own version table -- same "not part of routine migrate-up"
exclusion `prototype/go`'s Makefile already enforced for it); every Up and Down verified to run
clean round-trip against a real Postgres database (`app/DEVELOPMENT.md`'s new "database setup"
section). Verify step done against `app/`'s own `menata_app` database (never `prototype/go`'s
`menata_runtime`): `make migrate-up && make seed` (seeds/001 + 004, Case 3 Approval) followed by
`internal/metadata`'s new `TestLoadAllAgainstApprovalCase` -- confirms `ws_default`/`app_approval`/
both Approval machines load correctly. `go build ./... && go vet ./... && go test ./...` clean.
Proceeding to Phase 2.

## Phase 2 — Business logic layer

**Port:** `internal/constraint`, `internal/permission`, `internal/executor` — all depend only on
already-ported packages (Study 34 §7 Method 2), verbatim.

**Verify:** port `internal/executor/executor_test.go` unchanged (the `resolveDateArithmetic`/
`addBusinessDays` tests, including the "-" operator case fixed 2026-09-05) and confirm it passes.

**Milestone:** the full business-logic layer is portable and unit-testable in isolation, still no
HTTP server.

**Status update (2026-09-06):** Done. `constraint`/`permission`/`executor` ported verbatim (diffed
against `prototype/go`, only the expected import-path lines differ). Two new go.mod dependencies
pinned to `prototype/go/go.mod`'s own versions: `github.com/go-chi/chi/v5 v5.0.12` (executor uses
its `middleware` package), `github.com/pdfcpu/pdfcpu v0.15.0` (CAP-F22 composite PDF signature).
`internal/executor/executor_test.go` ported unchanged — all 14 cases pass, including the "-"
operator fix (2026-09-05). `go build ./... && go vet ./... && go test ./...` clean. Proceeding to
Phase 3.

## Phase 3 — HTTP layer

**Port:** `internal/ui` (including the `.templ` sources and generated `_templ.go` files),
`internal/router`, `internal/handler`, `cmd/server/main.go`.

**Gaps addressed here:**
- **API versioning** — introduce an `/api/v1/` prefix in `router.go`'s own port, from the start.
  This is the one moment before any real API consumer exists that renaming the surface is free;
  every day after this phase, it's a breaking change instead.
- **Rate limiting** — add a rate-limiting middleware (a chi-compatible token-bucket, e.g.
  per-IP/per-session) while the middleware stack is being wired fresh anyway.
- **Secrets** — `internal/config`'s own integration point (its `doc.go`) gets exercised for real
  here. Whether a full secrets-manager integration is actually justified yet, or plain env vars
  remain fine a while longer, is a call for whoever runs this phase to make against the real
  deployment target at the time — not pre-decided here, per this project's own "Infer Before
  Configure" principle.

**Verify:** `go build`, boot the server for the first time, `/health` responds.

**Milestone:** a real, bootable `app/` server exists.

**Status update (2026-09-06):** Done. `internal/ui` (.templ sources ported, regenerated via `templ
generate` rather than porting stale codegen), `internal/router`, `internal/handler` (14 files +
format_test.go), and `cmd/server/main.go` all ported. All three named gaps closed:
- **API versioning**: `/api/{machineID}` moved under `r.Route("/api/v1", ...)` in router.go.
  Verified: the old bare path no longer matches the API route family (falls through to the
  generic `/{machineID}` HTML route instead, confirmed by curl against a live instance).
- **Rate limiting**: new `cmd/server/ratelimit.go`, a hand-rolled per-IP token bucket
  (`golang.org/x/time/rate`, 10 req/s, burst 30, idle-eviction sweep) — matches this codebase's
  own house style of hand-writing every middleware rather than importing a framework. Wired after
  `middleware.RealIP` (new — corrects `r.RemoteAddr` from Caddy's own `X-Real-IP` header, without
  it every request would appear to come from Caddy's loopback address) and before
  `sessionAuth`/`workspaceTx`, so an over-budget IP is rejected before DB work runs for it.
  Verified: 40 rapid requests from one IP produced exactly 30×200 then 10×429.
- **Secrets**: decided, not deferred silently — plain env vars via `.env` stay the mechanism
  (`internal/config`, unchanged). Single VPS, single process, no multi-host secret-distribution
  need and no case forcing a vault integration yet, per "Infer Before Configure." Documented
  in `cmd/server/main.go` at `config.Load()`'s own call site.

One additional restructuring beyond upload.go's storage split (Phase 0's own scope): `internal/
handler/coordplace.go`'s PDF page-count preview read `uploadsDir` directly
(`pdfapi.PageCountFile`) — moved to `h.storage.Get` + `pdfapi.PageCount` (a byte-reader form of
the same pdfcpu call), keeping the whole `handler` package storage-abstracted. `internal/
executor/executor.go`'s own CAP-F22 composite-signature action (ported unchanged in Phase 2) still
reads/writes local disk directly via its own `uploadsDir` constant — out of Phase 3's port list
and not touched; both `handler`'s `storage.LocalDisk` and executor's hardcoded path point at the
same `uploads/` directory, so this is a known, harmless inconsistency (the abstraction isn't used
everywhere yet), not a regression.

Verified live: `go build -o bin/server ./cmd/server`, booted against `menata_app`
(migrated + seeded), `/health` → 200, `/` → 303 to `/login`, `/login` renders full HTML (templ
pipeline confirmed working end to end). `go build ./... && go vet ./... && go test ./...` clean.
Proceeding to Phase 4.

## Phase 4 — Conformance parity

**Port:** all 38 `seeds/*.sql` files, the full `conformance/` suite (15 test files + `run.sh`/
`lib.sh`), `scripts/local-ci.sh` and `scripts/check-css-classes.sh`, `web/` (Tailwind config +
static assets), and the client-side JavaScript policy correction from `ARCHITECTURE.md` — the four
cases that shipped as undocumented vanilla `<script>` in `prototype/go` (CAP-V16 typeahead, CAP-V15
live-sum preview, CAP-V14 kanban drag-drop, CAP-V21 coordinate-placement) port to real Hyperscript
here, not carried over as vanilla JS.

**Verify:** run the full conformance suite against `app/` for the first time. Any failure here
reveals either a porting mistake or a place a Phase 0-3 restructuring (metadata split, storage
abstraction, Hyperscript instead of vanilla JS) broke something real — fix before moving on, per
"ratchet, never regress." Also run `make test` (the ported unit tests from Phases 0/2) one more
time together with conformance, not as a substitute for it.

**Milestone:** `app/`'s own conformance suite passes at full parity with whatever
`prototype/go`'s own count is at the time this phase runs (219 as of this roadmap's writing).

## Phase 5 — CI + operational hardening

**Port:** mirror `prototype/go`'s three GitHub Actions workflows
(`conformance.yml`/`css-gate.yml`/`vet-test.yml`) for `app/`'s own path.

**New, not a port** — the one Study 34 §5 gap not addressed by an earlier phase: dependency
vulnerability scanning (`govulncheck`), added as a new CI step. Not previously done even for
`prototype/go` — a genuinely new addition here, not a carried-over gap.

**Test-coverage target:** the near-zero coverage outside pure-function packages (Study 34 §5) gets
a real, stated target at this point — set by whoever runs this phase against `app/`'s own actual
coverage numbers, not guessed in advance here.

**Milestone:** `app/` has the same CI maturity `prototype/go` reached this session, plus the
dependency-scanning gap `prototype/go` itself never closed.

## Phase 6 — Cutover

**Decision point, not pre-decided here:** does `app/`'s server replace `prototype/go`'s own
`menata.app` deployment directly, or run alongside on a new port/domain during a transition
window? Whoever reaches this phase decides against the real state of things at the time.

Once confident: `server-manager.sh` points at `app/`'s own binary instead of `prototype/go`'s.

---

## Explicitly out of scope for this roadmap

- **Object storage (S3-compatible) backend for `internal/storage`** — the interface exists from
  Phase 0; a real S3 backend is only built when a real multi-host need actually appears, per
  "Infer Before Configure." Building it speculatively now would be exactly the premature
  configuration this project's own principle warns against.
- **Multi-instance/`LISTEN`-`NOTIFY` cache invalidation (CAP-X11)** — no deployment target needs
  it; this host runs no containers, no multi-instance deployment (`ARCHITECTURE.md`'s own
  "Deployment target" section).
- **`internal/handler`'s team-scale API-boundary question** (Study 34 §5's last, most speculative
  finding) — no second engineer exists yet to make this concrete. Revisit only if that changes.
