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

**Status update (2026-09-06):** Done — 219/219, full parity, via `./scripts/local-ci.sh`'s own
fully automated isolated-schema/throwaway-port cycle (migrate-up → seed → build → boot →
conformance), not just a manually-managed database.

**Ported:** all 37 `seeds/*.sql` files (the roadmap's own "38" was an overcount — 37 exist, no
`018`), the full `conformance/` suite (15 test files + `run.sh`/`lib.sh`/`README.md`),
`scripts/local-ci.sh` (adjusted for `app/`'s own `menata_app` database and doc framing) and
`scripts/check-css-classes.sh` (unchanged), and `web/` (`tailwind.config.js`, `package.json`,
`input.css` — `output.css` is a gitignored build artifact, regenerated via `make build-css`,
confirmed working with Node 20).

**Client-side JavaScript policy correction:** the four named capabilities (CAP-V16 typeahead,
CAP-V15 live-sum, CAP-V14 kanban drag-drop, CAP-V21 coordinate-placement) moved off
`layout.templ`'s page-global delegated `<script>` block onto per-element Hyperscript `_="..."`
attributes (`components.templ`, `board.templ`, `coordplace.templ`), each using Hyperscript's own
`js(...)` escape for the DOM/fetch logic that doesn't have a natural Hyperscript-native form —
colocated on the actual element, auto-reinitializing on an HTMX swap, not a growing shared file.
Hyperscript pinned via CDN (`hyperscript.org@0.9.93`, latest stable at port time), same pattern
`htmx.org@2.0.4` already uses. **Found, not fixed (flagged for a future pass):** CAP-V19
(reference-field live preview) sits in the exact same shared script block in an identical
situation, but was never one of the four this phase's own audit named — left as vanilla JS,
unconverted, since converting it wasn't in this phase's actual scope.

**Two real regressions found and fixed by the conformance run itself** (exactly what "ratchet,
never regress" and this phase's own verify step exist to catch):
1. **The rate limiter (Phase 3) was calibrated too aggressively for real use.** First full run: 121
   passed, 98 failed, the overwhelming majority on an unexpected 429 — the suite itself (all
   requests from one IP in this dev setup) peaked at 57 requests in a single wall-clock second, a
   legitimate-traffic shape (a real browser session issuing several HTMX/typeahead/asset requests
   in quick succession looks the same) the original 10 req/s-burst-30 defaults choked on. Reset to
   30 req/s / burst 120 (`cmd/server/ratelimit.go`) — comfortably above the measured peak. Second
   run: 216/219.
2. **`internal/handler/formfields.go`'s CAP-V19 `LivePreviewURL` still hardcoded the bare `/api/`
   prefix** Phase 3's own versioning change (`/api/v1/`) had already retired — a real dead-route
   bug in the live application, not just a stale test. Fixed to `/api/v1/`; the matching
   conformance assertion (`conformance/tests/060_ui_cluster.sh`) and the 9 hardcoded `/api/`
   occurrences testing CAP-X07 directly (`conformance/tests/040_batches10_11_subnav.sh`, T118–T120)
   updated to match. Third run: 219/219.

**Also found and corrected, unrelated to this phase's own changes:** Phase 0–3's own verification
had been using the shared `postgres` superuser directly against an ad hoc `menata_app` database —
exactly the practice `prototype/go/DEVELOPMENT.md`'s "Database role" section warns against (a real
outage for another app on this shared host, once already). Corrected to a dedicated
`menata_app_owner` role owning `menata_app`, no superuser/createdb bits — re-verified 219/219
under this role alone. See `DEVELOPMENT.md`'s own new section for the how-to.

`go build ./... && go vet ./... && go test ./...` clean; `make check-css` clean (44 color
utility classes, all present). Proceeding to Phase 5.

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

**Status update (2026-09-06):** Done. Three new GitHub Actions workflows at the repo root
(`.github/workflows/app-vet-test.yml`, `app-css-gate.yml`, `app-conformance.yml`), mirroring
`prototype/go`'s own three exactly (same `working-directory`/path-filter/service-container shape),
`app/**`-path-scoped so they don't fire on unrelated changes and don't collide with
`prototype/go`'s own identically-purposed workflows living in the same directory.

**govulncheck (new, not a port) found 37 real reachable vulnerabilities on its first run** —
exactly the kind of gap this step exists to close, not a clean pass to rubber-stamp. Triaged and
fixed:
- **`github.com/go-chi/chi/v5` v5.0.12 → v5.3.2**: two of the 37 (GO-2026-5777/5775) were
  `middleware.RealIP` itself — deprecated in v5.3.0 for being fundamentally spoofable (walks
  True-Client-IP/X-Real-IP/X-Forwarded-For unconditionally, mutates `r.RemoteAddr` in place).
  Migrated `cmd/server/main.go` to chi's own replacement, `middleware.ClientIPFromHeader("X-Real-IP")`
  — explicit about trusting exactly the one header `menata.app`'s Caddyfile unconditionally
  overwrites, read via `middleware.GetClientIP(ctx)` (new `clientIP()` helper, shared by
  `slogAccessLog` and `ratelimit.go`) instead of a mutated `r.RemoteAddr`. Verified live: a request
  carrying `X-Real-IP` resolves to that address in the access log; one without falls back to the
  raw TCP peer, same as before.
- **`golang.org/x/image` v0.44.0 → v0.45.0**: GO-2026-6222, excessive memory allocation decoding
  VP8L, reachable from `handler.compressImage`'s own image pipeline (CAP-F06).
- **`github.com/jackc/pgx/v5` v5.6.0 → v5.10.0**: GO-2026-5004, a SQL-injection-class bug (dollar-
  quoted string literal / placeholder confusion), reachable from `store.RecordStore.CountGroupedBy`.
  Fixed upstream at v5.9.2; took latest.
- **Go toolchain 1.25.0 → 1.25.14** (`go.mod`'s own `go` directive; same 1.25 line as `prototype/
  go`, no language-version jump): closed the remaining ~30, all Go standard-library CVEs
  (`crypto/tls`, `net/http`, `net/url`, `html/template`, `encoding/xml`, `encoding/asn1`,
  `crypto/x509`, `encoding/pem`) fixed across the 1.25.2–1.25.13 patch releases.
- Re-ran `govulncheck ./...` after each fix: 37 → 1 (pgx) → 0. Full 219/219 conformance + `go
  test ./...` re-verified after all four changes, including the ClientIPFromHeader migration.

**Test-coverage target, set against `app/`'s own real numbers** (`go test ./... -cover`, gathered
this phase): `internal/model` 88.9%, `internal/executor` 15.3%, `internal/handler` 0.6%,
`internal/metadata` 0.0% (its own test, `TestLoadAllAgainstApprovalCase`, is a DB-backed
integration check, not a pure-function unit test), every other package 0% (no test files). Given
`prototype/go/CLAUDE.md`'s own stated, deliberate testing philosophy — "there are no Go unit tests
in this codebase, the conformance suite IS the test suite" — a blanket module-wide percentage
target would fight that design, not extend it. **Target actually set:** pure-function/deterministic
logic (the shape `internal/model` and `internal/executor`'s own existing tests already cover —
`FindFieldByName`/`FindReferenceFieldTo`, `resolveDateArithmetic`/`addBusinessDays`) should reach
and hold 80%+ coverage over its own surface as new such functions are added; the HTTP/store/handler
orchestration layers are NOT targeted for unit-test growth — the 219-test conformance suite remains
their real proof, per `app-vet-test.yml`'s own "fast feedback loop the conformance suite alone
doesn't give" framing (`roadmap.md` item 14). `internal/metadata`'s `validateOperators`/
`validateReferences` (currently 0%, genuinely pure functions over `[]*model.Workspace`) are the
one concrete opportunity this baseline surfaces for a future session, not required by this phase.

**Status update (2026-09-06, taking up that named opportunity, not a new phase):**
`internal/metadata/validate_test.go` added — `validateOperators` at 100% statement coverage (7
cases), `validateReferences` at 48.6% (24 cases across reference fields, `owner_field`,
`cross_record` aggregate/reference_field, process requirement backrefs, money currency, computed
`source_field` — the headline CAP-F13/CAP-P02 checks plus the largest sub-checks; the View-config
checks (`child_lines`, list-view sort/filter/steps/report/coord_placement, dashboard sections) and
Subscription/Contract validation remain untested, a further opportunity for whoever picks this up
next, not silently claimed done here). Uses `internal/testing/builders` for every case. Full
219/219 conformance and `go build/vet/test ./...` re-verified; `scripts/check-quality-gates.sh`
unaffected (test-only change).

`go build ./... && go vet ./... && go test ./...`, `make check-css`, and the full conformance
suite (219/219, via `local-ci.sh`) all clean after every change in this phase. Proceeding to
Phase 6 (Cutover) whenever the owner decides to run it.

**Status update (2026-09-06, post-Phase-5 hardening, not a new phase):** `app/docs/
portal-ga3-code-quality-benchmark.md`'s action items #3–#6 closed — quality-ratchet CI gates,
`internal/testing` scaffolding, ADR index, actor-parameter convention. Full detail in that
document and in `README.md`'s own mirrored status update; not restated here. Does not change this
phase's own test-coverage target above — see the benchmark doc's own reconciliation note.

## Phase 6 — Cutover

**Decision point, not pre-decided here:** does `app/`'s server replace `prototype/go`'s own
`menata.app` deployment directly, or run alongside on a new port/domain during a transition
window? Whoever reaches this phase decides against the real state of things at the time.

Once confident: `server-manager.sh` points at `app/`'s own binary instead of `prototype/go`'s.

**Status update (2026-09-06): Done — direct full replacement, `menata.app` now serves `app/`.**
Owner decision: direct replacement, not a parallel transition window; RLS handled "at the real
cutover moment."

Pre-cutover investigation (read-only) found the decision was simpler than the roadmap's own
framing assumed: **RLS was already live on `menata_runtime`** (`prototype/go`'s own deployment
must have applied 009 at some earlier, untracked point in its history) — `records`/
`record_events`/`notifications`/`action_outbox` all already showed `relrowsecurity=t,
relforcerowsecurity=t` with the exact `ws_isolation` policy 009 declares, before this phase
touched anything. A full `information_schema.columns` diff between `menata_runtime` and `app/`'s
own `menata_app` found zero differences in application tables. So the cutover reduced to *which
binary serves the existing data*, not a data migration — `app/`'s `workspaceTx` middleware
(ported verbatim, Phase 3) already sets `app.workspace_id` per request exactly the way RLS
requires, independent of when 009 was actually applied.

**Sequence executed, each step verified before the next:**
1. `pg_dump menata_runtime` to `/root/backups/` before touching anything.
2. Copied `prototype/go/uploads/`'s 12 real files into `app/uploads/` (additive — `prototype/go`'s
   own copy untouched) — file-typed field values in existing records reference these by key.
3. Brought `menata_runtime` under goose tracking: `make migrate-up` (all 24, idempotent, zero
   errors) + `make migrate-rls-cutover` (009, confirmed still a no-op against the already-RLS'd
   tables) — `menata_runtime` now has both `goose_db_version` and `goose_manual_db_version`.
4. Built `app/`'s production binary (`make build`).
5. Pre-swap smoke test on a scratch port (4002) against the live `menata_runtime` database, while
   `prototype/go` was still the one actually serving traffic: `/health`, a real login (Alice), and
   `/files/{key}` serving a real uploaded PDF byte-identical to `prototype/go`'s own copy.
6. `server-manager.sh`'s `start_menata_runtime` (`/root/scripts/server-manager.sh`, not in this
   repo) now `cd`s into `app/` instead of `prototype/go/` — `stop_menata_runtime` needed no change
   (port-based). `app/.env` created with production values (`menata_runtime_app` role — reused
   as-is, already owns the right grants on this database — `menata_runtime` db, `PORT=4000`,
   `SECURE_COOKIES=true`).
7. Cutover: `server-manager.sh stop menata-runtime` then `start menata-runtime` — a few seconds of
   downtime for the swap itself, matching the direct-replacement decision.
8. Post-cutover verification against the real domain: `/health` → 200, a real login, a real write
   (a Design Request record, validated correctly by CAP-C09 constraints, persisted with a real
   `303` redirect), zero `ERROR`-level log lines, and confirmation the access log now attributes
   requests to the real external client IP (not Caddy's loopback address) — the Phase 5
   `ClientIPFromHeader` fix working correctly in actual production traffic, not just synthetic
   `curl -H` tests.

`prototype/go`'s own `bin/server` and `uploads/` are untouched — rollback is
`server-manager.sh`'s `start_menata_runtime` `cd` reverted to `prototype/go`, stop/start again.

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
