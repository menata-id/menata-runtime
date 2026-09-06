# Menata Runtime (Application)

**This is the real Menata Runtime application** — not another prototype. It is being built by
graduating [`prototype/go`](../prototype/go/)'s own proven codebase (~90 capabilities ✅, a
219-test conformance suite, an architecture independently vetted twice) rather than starting from
zero, per the decision recorded in
[`../benchmarks/026-runtime-graduation-decision.md`](../benchmarks/026-runtime-graduation-decision.md)
(Study 34).

`../prototype/` (all seven platform prototypes) and `../benchmarks/` (the capability-discovery
evidence series) have done their job — proving which capabilities a runtime needs and that a real
implementation of them is possible. They stay exactly as they are, unrenamed, unrestructured, kept
as the historical record of that process. This folder is what that process was building toward.

## Where to start

- **[`ARCHITECTURE.md`](ARCHITECTURE.md)** — the blueprint: package layout, what's graduated as-is,
  what's new, what's deferred and why.
- **[`ROADMAP.md`](ROADMAP.md)** — the development plan: what order the port happens in, and what
  "done" means at each phase.
- **[`docs/decisions/001-graduation-from-prototype.md`](docs/decisions/001-graduation-from-prototype.md)**
  — the ADR formalizing this move.
- **[`DEVELOPMENT.md`](DEVELOPMENT.md)** — setup, once real code lands here.
- **[`CLAUDE.md`](CLAUDE.md)** — dev patterns and gotchas (starts as a pointer back to
  `prototype/go/CLAUDE.md`'s own accumulated catalog; entries graduate here as their code does).

## Current status

Blueprint stage: real package skeleton (`internal/*/doc.go`, one per package, each stating its
graduated source and any architectural change), `go.mod`, and this documentation set exist.
No business logic has been ported yet — that starts at `ROADMAP.md`'s own Phase 0. `go build ./...`
and `go vet ./...` both pass clean against the current doc.go-only skeleton.

**Status update (2026-09-06):** `ROADMAP.md`'s Phase 0 is done. `model`, `auth`, `db`, `config`,
`store` ported verbatim (pure move, diffed byte-identical against `prototype/go` apart from
stripping two now-redundant package-doc comments already carried by `doc.go`) under
`menata.id/app`'s own import path; `internal/storage` (new) implements the `Store` interface with
a `LocalDisk` backend, extracted from `prototype/go/internal/handler/upload.go`'s
`storeFile`/`ServeFile` logic. `go build ./... && go vet ./...` clean, `internal/model/
model_test.go` ported unchanged and passing. `menata.app` still serves `prototype/go` — no Caddy
change yet, that's `ROADMAP.md`'s own Phase 6 (Cutover) decision point, not before. Next: Phase 1
(metadata pipeline).

**Status update (2026-09-06, Phase 1):** `internal/metadata` (split loader.go/validate.go/
compile.go + materialize.go) and `internal/interpreter` ported. Migration tool chosen: goose — all
24 `prototype/go/migrations/*.sql` converted to Up/Down pairs, every one verified to apply and roll
back cleanly against a real Postgres database. `internal/metadata`'s new
`TestLoadAllAgainstApprovalCase` confirms `Loader.LoadAll` builds a correct Application Model from
a freshly migrated + seeded database. See `ROADMAP.md`'s own Phase 1 status update and
`DEVELOPMENT.md`'s new "database setup" section for the how-to. `menata.app` still serves
`prototype/go` — still Phase 6's decision, not before. Next: Phase 2 (business logic layer).

**Status update (2026-09-06, Phase 2):** `constraint`/`permission`/`executor` ported verbatim
(diffed against `prototype/go`, only import-path lines differ). `internal/executor/executor_test.go`
ported unchanged, all 14 cases pass. Two new pinned dependencies: `github.com/go-chi/chi/v5` and
`github.com/pdfcpu/pdfcpu`, matching `prototype/go/go.mod`'s own versions. `go build ./... && go
vet ./... && go test ./...` clean. `menata.app` still serves `prototype/go`. Next: Phase 3 (HTTP
layer).

**Status update (2026-09-06, Phase 3):** `internal/ui`, `internal/router`, `internal/handler`, and
`cmd/server/main.go` all ported — a real, bootable server exists. All three named gaps closed:
`/api/` routes versioned under `/api/v1/`, a new hand-rolled per-IP rate limiter
(`cmd/server/ratelimit.go`), and a deliberate (not silently deferred) secrets decision — plain env
vars stay, no vault introduced yet. Verified live against `menata_app`: server boots, `/health`
responds, `/login` renders. See `ROADMAP.md`'s own Phase 3 status update for the full detail,
including the one extra storage-abstraction fix (`handler/coordplace.go`'s PDF preview) beyond
Phase 0's original scope. `menata.app` still serves `prototype/go` — still Phase 6's decision.
Next: Phase 4 (conformance parity).

**Status update (2026-09-06, Phase 4): full conformance parity, 219/219.** All 37 seed files, the
full conformance suite, `web/` (Tailwind), and the client-side JavaScript policy correction
(CAP-V16/V15/V14/V21 moved to Hyperscript `_="..."` attributes; CAP-V19 found in the same
situation but flagged, not fixed — out of this phase's own named scope) all ported. Verified via
`./scripts/local-ci.sh`'s fully automated isolated-schema cycle, not just manual runs.

The conformance run itself caught two real regressions before they could ship: the Phase 3 rate
limiter was calibrated far too aggressively for real traffic shape (reset from 10 req/s/burst 30
to 30/120 after measuring the suite's own peak), and `formfields.go`'s CAP-V19 live-preview URL
still pointed at the pre-versioning `/api/` path (fixed to `/api/v1/`) — both are exactly what this
phase's own verify step exists to catch. Separately, Phase 0–3's own dev-database access was
corrected from the shared `postgres` superuser to a dedicated `menata_app_owner` role, matching
`prototype/go/DEVELOPMENT.md`'s own documented "why not `postgres`" reasoning. See `ROADMAP.md`'s
own Phase 4 status update and `DEVELOPMENT.md`'s database sections for full detail. `go build/vet/
test` and `make check-css` all clean. `menata.app` still serves `prototype/go`. Next: Phase 5
(CI + operational hardening).

**Status update (2026-09-06, Phase 5): CI live on GitHub, 0 reachable vulnerabilities.** Three new
workflows (`.github/workflows/app-{vet-test,css-gate,conformance}.yml`) mirror `prototype/go`'s
own three, scoped to `app/**`. `govulncheck` (new, not a port — the one gap `prototype/go` itself
never closed) found 37 real reachable vulnerabilities on its first run: chi's `middleware.RealIP`
(deprecated, spoofable — migrated to `middleware.ClientIPFromHeader("X-Real-IP")`), an `x/image`
VP8L-decode issue, a pgx SQL-injection-class bug, and ~30 Go-stdlib CVEs closed by bumping the
toolchain to the latest 1.25.x patch. All four fixed; 37 → 0. A real, evidence-based test-coverage
target is set (see `ROADMAP.md`'s own Phase 5 status update) rather than a guessed blanket
percentage, consistent with this project's own "conformance suite IS the test suite" philosophy.
Full 219/219 conformance and `go test ./...` re-verified after every fix. `menata.app` still
serves `prototype/go` — Phase 6 (Cutover) is next, whenever the owner decides to run it.

**Status update (2026-09-06, post-Phase-5 engineering-practice hardening):** not a new phase —
`app/docs/portal-ga3-code-quality-benchmark.md`'s own action items #3–#6 closed: 4 new CI quality
gates (`scripts/check-quality-gates.sh`: error-leak scan, handler-LOC ratchet, `gocyclo` ratchet,
actor-parameter convention), `internal/testing` scaffolding (builders/fixtures/httptest/mocks/
testdb, verified against a real database), an ADR index (`docs/decisions/README.md`), and the
actor-parameter convention written into `internal/permission/doc.go`. One real CWE-209 error leak
found and fixed along the way (`admin.go`'s Reload handler). See that document's own "Reconciled
against a decision this document didn't know about yet" note for how the new `internal/testing`
scaffolding relates to this same status block's test-coverage target above — it doesn't change
that target. Full 219/219 conformance and `go build/vet/test ./...` re-verified. `menata.app`
still serves `prototype/go` — Phase 6 (Cutover) unchanged, still next whenever the owner decides.

**Status update (2026-09-06, Phase 6): cutover done — `menata.app` now serves `app/`.** Owner
decision: direct full replacement (not a parallel transition window). Pre-cutover investigation
found RLS was already live on `menata_runtime` from `prototype/go`'s own earlier history, and a
full schema diff against `app/`'s own database found zero differences — the cutover reduced to
swapping which binary serves the existing data, backed by a `pg_dump` taken first. `server-
manager.sh` (not in this repo) now starts `app/`'s binary on the same port or database; `prototype/
go`'s own binary/uploads stay untouched for rollback. Verified live: real login, a real write
persisted and validated correctly, zero error-level logs, and the access log confirming real
client IPs resolve correctly through Caddy (the Phase 5 `ClientIPFromHeader` fix, proven in actual
production traffic). Full detail in `ROADMAP.md`'s own Phase 6 status update. This is the last
phase in the original graduation plan — `app/` is now the live Menata Runtime application.

**Status update (2026-09-06, post-cutover, not a new phase):** CAP-V19 (reference-field live
preview), flagged but left as vanilla JS in the Phase 4 client-side JavaScript policy correction
above, is now converted to Hyperscript — the last of the five capabilities that policy named.
`internal/ui/components.templ`'s `FieldInput` reference `<select>` carries an `on change`
Hyperscript handler in place of `layout.templ`'s old page-global delegated listener, which is now
removed entirely (no vanilla `<script>` remains on the page). Full detail in `ROADMAP.md`'s own
Phase 4 section. `go build/vet/test ./...` clean, full conformance re-verified 219/219.
