# Portal GA v3 Code-Quality Benchmark

> Status: v0.3 — all six action items closed | Created: 2026-09-06 | Updated: 2026-09-06

> **Correction (2026-09-06):** the original v0.1 draft cited `app/ARCHITECTURE.md`'s "Named, not
> solved" gap table without first checking `app/README.md`'s own "Current status" section for
> whether those gaps were still open — exactly the check `app/CLAUDE.md` says never to skip.
> They were not: `ROADMAP.md`'s Phase 1 status update (predates this document) already closed the
> migrations gap (goose, `.up`/`.down` pairs, rollback-verified), and Phase 3/4 already closed the
> rate-limiting and API-versioning gaps. Action items #1 and #2 below are struck through as a
> result; #3–#6 were independently verified still open against the current `ROADMAP.md` Phase 5
> scope (mirrors 3 CI workflows + `govulncheck` only — no LOC/complexity budgets, no
> `internal/testing` scaffolding, no ADR index, no actor-first convention).

> **Status update (2026-09-06): items #3–#6 done.** `scripts/check-quality-gates.sh` (4 gates,
> baseline-ratchet shape), `internal/testing/{builders,fixtures,httptest,mocks,testdb}`,
> `app/docs/decisions/README.md`, and `internal/permission/doc.go`'s "Actor-parameter convention"
> section all landed — see commits `a11b093`/`978fa57`/`9fda25f`/`47acbfe`. Full detail in each
> item's own entry below, not restated here.
>
> **Reconciled against a decision this document didn't know about yet:** between this document's
> v0.2 correction and this update, a separate change (commit `4ef8e15`) set `ROADMAP.md` Phase 5's
> own test-coverage target, explicit that "the HTTP/store/handler orchestration layers are NOT
> targeted for unit-test growth — the 219-test conformance suite remains their real proof." Item
> #4's `internal/testing/httptest` and `internal/testing/mocks` scaffold exactly that layer, which
> could read as contradicting that target. It doesn't: the target governs *coverage-percentage
> growth* for handler/store, not whether test infrastructure may exist for it. `httptest.NewHandler`
> has exactly one caller today (`TestNewHandler_Boots`, a smoke test proving the wiring itself
> works against a real database — not a step toward a handler coverage number), and `mocks.Storage`
> has none yet. Neither should be read as this document quietly overriding that target; a future
> session reaching for either to grow `internal/handler`'s own coverage percentage should re-read
> `ROADMAP.md`'s Phase 5 status update first, not assume this document already cleared that.

## Purpose and scope

This document benchmarks `app/`'s own engineering practice — CI gates, migration tooling, test
organization, decision-record hygiene — against `portal-ga3` (`/root/projects/portal-ga3`), a
mature, actively-maintained Go codebase built by the same team.

This is a **different question** from `../../benchmarks/002-portal-ga-cross-domain-survey.md`
(Study 5), which benchmarked Portal GA to discover missing *runtime capabilities* (wizard views,
organizational scoping, domain-event subscription). This document instead asks: independent of
what capabilities the runtime exposes, is `app/`'s own Go code held to the same engineering rigor
Portal GA holds itself to? The answer matters directly because `app/ARCHITECTURE.md`'s own
"Named, not solved" table already lists production-readiness gaps that Portal GA has, in practice,
already solved for itself.

**Sources studied (portal-ga3 repo):** `CLAUDE.md` (Rules #1–#12, Architecture Pattern, Code
Organization), `Makefile` (`check-*` targets), `scripts/check-*.sh` (27 scripts), `docs/
explanation/architecture/decisions/` (ADR-0001–0030 index), `internal/testing/` layout.

---

## Boundary: what does not transfer, and why

Portal GA is a **hand-coded** application: 39 business domains, each with its own
`internal/domains/{domain}/{core,handler,query}` folder and hand-written `Policy`/`Recorder`/
`Resolver` types. `app/` is a **generic interpreter**: one runtime, `internal/interpreter` +
`internal/constraint` + `internal/executor`, realizes arbitrarily many applications from Runtime
Metadata it has never seen at compile time. These are opposite designs by intent, not by maturity
gap — `002-architecture.md`'s own "Runtime Independence" section states the runtime "should remain
independent from... programming languages... Only Runtime Metadata should determine application
behavior," and `002-architecture.md`'s opening line is blunter still: "Applications are
interpreted. Applications are not generated."

| Portal GA pattern | Why it does not transfer |
|---|---|
| 39 domain folders, each with its own `core`/`handler`/`query` | Business logic per domain lives in Go source. In `app/`, business logic lives in Runtime Metadata rows (`internal/metadata`, `internal/constraint`) interpreted by generic code. Copying this structure would mean hand-coding business rules again — the exact thing `app/` exists to avoid. |
| Decision-Based Architecture naming for business rules (`{domain}_policy.go`, `{domain}_recorder.go` per domain) | Same reason — one hand-written file per business concept has no equivalent when the concept is a metadata row, not a compiled type. |
| Domain Integration Constitutional Framework (PICA→AAR, Consumer-Driven Contracts, Context Map Matrix) | Governs *how Portal GA's own 39 hard-coded domains* talk to each other. `app/` has no fixed domain set to govern this way — cross-machine interaction is itself a metadata concern (`benchmarks/002-portal-ga-cross-domain-survey.md` Angle 2 already covers this from the capability side). |

The rest of this document only considers patterns that apply to `app/`'s own **generic** layers
(`handler`, `executor`, `interpreter`, `permission`, `storage`, `config`, `db`) — code that is,
and will remain, hand-written regardless of what metadata it interprets.

---

## What transfers: engineering-practice gates

Each row below maps a concrete Portal GA practice to a gap `app/ARCHITECTURE.md` has already
named against itself, under "Named, not solved — deferred to the development plan".

| Practice | Portal GA evidence | `app/` gap it closes | Where it would live |
|---|---|---|---|
| Migration tool with `.up.sql`/`.down.sql` pairs, tracked in a `schema_migrations` table, applied only via `go run cmd/migrate/main.go up` (never raw `psql -f`) | `portal-ga3/CLAUDE.md` "Migration file HARUS `.up.sql`/`.down.sql`"; `scripts/check-migration-naming.sh`, `check-migration-full-replay.sh`, `check-migration-down-chain.sh` | `app/ARCHITECTURE.md`: *"Migrations are flat numbered `.sql` files, no version-tracking table, no rollback"* | `internal/db` + a new `cmd/migrate` |
| Rate-limiting middleware on high-risk endpoints, checked in CI | `portal-ga3/Makefile` `check-rate-limiting` target (FF-013, OWASP API4:2023), `scripts/check-rate-limiting.sh` | `app/ARCHITECTURE.md`: *"No rate-limiting/DoS-protection middleware"* | `internal/handler`'s own middleware stack (already flagged as the integration point in `app/ARCHITECTURE.md`) |
| Grep-based CI scan rejecting raw `err.Error()` leaked into HTTP responses (CWE-209) | `portal-ga3/Makefile` `check-security-patterns` target; `portal-ga3/CLAUDE.md` "NO raw `err.Error()` in HTTP responses" | Not separately named in `app/ARCHITECTURE.md`, but is exactly the kind of gap the near-zero test coverage row leaves unguarded | `app/scripts/local-ci.sh`, as a new grep-based gate over `internal/handler` |
| File/function size and cyclomatic-complexity budgets enforced in CI (handler >800 LOC = error, function >80 LOC, `gocyclo` >10 = error) | `portal-ga3/CLAUDE.md` Rule #7 "God Code Prevention"; `Makefile` `check-handler-size`, `check-complexity` | Not named as a gap, but directly guards against `internal/metadata/loader.go` growing to 1,098 lines again (the exact restructuring `app/ARCHITECTURE.md`'s own "What's graduated, but restructured" section had to do after the fact) | `app/scripts/local-ci.sh` |
| `internal/testing/` package: shared builders, fixtures, `httptest` helpers, mocks, and a `testdb` harness, used by every test instead of ad hoc setup per test file | `portal-ga3/CLAUDE.md` "Testing: `internal/testing/` (builders, fixtures, httptest, mocks, testdb)" | `app/ARCHITECTURE.md`: *"Test coverage near-zero outside 3 pure-function files added 2026-09-05"* — this is the scaffolding a real coverage push would need first | New `internal/testing` package |
| A single ADR index page listing every decision by number and one-line summary | `portal-ga3/docs/explanation/architecture/decisions/README.md` ("ADR Index — 30 architectural decisions") | `app/docs/decisions/` currently holds one ADR with no index; `prototype/go/docs/decisions/` (8 ADRs) has none either | `app/docs/decisions/README.md` |
| Explicit "actor first" call convention: every service-layer method takes `(ctx context.Context, actor models.UserContext, ...)` as its first two parameters, so authorization context is never implicit | `portal-ga3/CLAUDE.md` "Actor Context Pattern - All service methods: `func (s *Service) Method(ctx context.Context, actor models.UserContext, ...)`" | Not a named gap, but formalizes a convention `internal/permission`/`internal/executor` already follow informally — writing it down and lint-checking it prevents silent drift as the port continues | `internal/permission`'s own `doc.go`, plus a grep-based CI check |

---

## Recommended action list

Priority ordered by which gap is both already named in `app/ARCHITECTURE.md` and cheapest to
close. This is a candidate list for `app/ROADMAP.md` sequencing, not a decision — placing these
into a specific phase is the roadmap owner's call, not this document's.

1. ~~**Migration tool**~~ — already done. `ROADMAP.md`'s Phase 1 chose goose; all 24 migrations
   are `.up`/`.down` pairs, verified to apply and roll back against a real Postgres database.
2. ~~**Rate-limiting middleware**~~ — already done. `ROADMAP.md`'s Phase 3 added a hand-rolled
   per-IP rate limiter (`cmd/server/ratelimit.go`), recalibrated against real traffic shape in
   Phase 4.
3. ~~**CI gates**~~ — done. `scripts/check-quality-gates.sh`, 4 gates (error-leak scan, handler-LOC
   ratchet, `gocyclo` ratchet, actor-parameter convention), wired into `local-ci.sh` and
   `app-vet-test.yml`. Baseline-ratchet, not flat thresholds — see the gate's own header comment
   for why a flat portal-ga3-style threshold would have failed immediately against already-shipped
   code (`record_crud.go` at 1009 LOC). Found and fixed one real CWE-209 leak in `admin.go` along
   the way (raw internal error on a 500 response); the other 9 occurrences were controlled,
   human-authored validation messages, marked `// errleak:allow: <reason>` rather than changed.
4. ~~**`internal/testing` package**~~ — done, scoped down from portal-ga3's own shape where this
   codebase can't support it (`internal/testing/doc.go`'s "Scope note on mocks": no mockable
   interfaces exist for store/session/notification, only `internal/storage.Store`). Verified
   against a real migrated+seeded Postgres, not just compiled — see this document's own
   "Reconciled against a decision..." note above for how this relates to Phase 5's coverage target.
5. ~~**ADR index**~~ — done, `app/docs/decisions/README.md`.
6. ~~**Actor-first convention**~~ — done, `internal/permission/doc.go`'s "Actor-parameter
   convention" section + Gate 4 above. Adapted from portal-ga3's literal shape (a single `actor`
   struct) to what this codebase actually does — separate `actorRole`/`actorIdentity` string
   parameters, ctx-first, role-before-identity — rather than copying a shape this codebase doesn't
   use.

---

## Patterns worth reading for inspiration only (not proposed for adoption)

- `portal-ga3`'s component-selection guide (`docs/reference/components/COMPONENT-SELECTION-GUIDE.md`)
  — useful shape (a table of "when to use X vs Y" for near-duplicate UI helpers) if `internal/ui`
  ever accumulates enough templ components to need one; not needed at `app/`'s current size.
- `portal-ga3`'s Diátaxis-style `docs/` split (`guides/`, `reference/`, `explanation/`,
  `tutorials/`) — a reasonable taxonomy, but `app/`'s own doc set is small enough that root
  `CLAUDE.md`'s existing Tier system already covers it without a parallel taxonomy inside `app/`.
