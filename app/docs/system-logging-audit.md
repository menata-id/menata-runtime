# System Logging Audit

> Status: v0.1 — draft, all action items open | Created: 2026-09-07 | Updated: 2026-09-07

## Purpose and scope

This document audits `app/`'s own process-level logging (`log/slog` calls in `cmd/server` and
`internal/*`, the HTTP access log, and how both relate to the `record_events` business audit
trail) against common structured-logging / observability practice (12-factor app logging, OWASP
Logging Cheat Sheet, the correlation/SLO shape `capability-registry.md`'s CAP-I04 row already
names). It answers one question: **is what's already built ("Status: adequate" or "needs work")
close to done, and if not, what's the concrete gap list** — not a redesign proposal.

This is a different question from `record_events` itself (CAP-R04, the append-only, DB-level-
immutable *business* audit trail) — that mechanism is sound and out of scope for corrections here.
This document is about the **operational** log stream: what a person watching `stdout` (or
whatever ships it) sees during normal operation and during an incident.

**Sources read:** `cmd/server/main.go` (the process-wide `slog.NewJSONHandler`, `slogAccessLog`,
`workspaceTx`/`sessionAuth`/`csrfProtect` middleware), every `slog.*` call site under
`internal/handler`, `internal/executor`, `internal/mailer` (145 call sites total, see grep below),
`migrations/007_audit_logging.sql`, `capability-registry.md`'s CAP-I04 row, `nfr-standards.md`'s
STRIDE table and Repudiation/Tampering rows, `internal/config/config.go`, `internal/router/
router.go`.

---

## What's already solid

| Practice | Evidence |
|---|---|
| One structured JSON stream for the whole process, not two differently-shaped outputs | `cmd/server/main.go:39`, `slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))`, set before anything else logs; replaced chi's stdlib-`log` `middleware.Logger` with a custom `slogAccessLog` that writes through the same handler (main.go's own header comment; CAP-I04 registry row, "same day, format unified too") |
| Correlation id threading HTTP request → security event → business audit row | chi's `middleware.RequestID` generates one id per request; `slogAccessLog` attaches it as `correlation_id` to the access-log line; `Executor.Persist` writes the same id into every `record_events` row a request produces, including cascades (CAP-A08 `aggregate_status`, CAP-E05 `trigger_event`) — proven by conformance test T43 (CAP-I04 registry row) |
| Append-only, DB-level-immutable business audit trail, separate from the process log | `migrations/007_audit_logging.sql` — `REVOKE UPDATE, DELETE, TRUNCATE ON record_events FROM CURRENT_USER`; every row carries `performed_by` (actor) + `correlation_id` (CAP-R04, closes the STRIDE Repudiation/Tampering rows in `nfr-standards.md`) |
| Baseline access-log fields per request | `slogAccessLog` (`cmd/server/main.go:421-436`): `method`, `path`, `status`, `bytes`, `duration_ms`, `remote_addr` (via `clientIP`, which correctly prefers `middleware.ClientIPFromHeader("X-Real-IP")` over the raw TCP peer — Phase 5's govulncheck fix) |
| Security-relevant events actually logged, not silently swallowed | Login success/failure (`auth.go`), CSRF token mismatch (`main.go:724`), permission denial + rule violation (`events.go:213,232`), admin role/group changes with actor attribution (`admin.go:115,320,376`), invitation lifecycle (`invitations.go`) |
| No secrets found logged in cleartext | Spot-checked every `auth.go`/`invitations.go` call site touching a password or session token — only hash/verify *failures* are logged (`"hash password", "error", err`), never the raw value; session/CSRF tokens are never passed to a log call anywhere in the 145 call sites checked |

---

## Gaps found

Ordered by how directly each one undercuts the correlation/traceability goal `capability-
registry.md`'s CAP-I04 row already claims, or by operational risk if left alone.

| # | Gap | Evidence | Why it matters |
|---|---|---|---|
| 1 | **`correlation_id` is attached inconsistently, not on "every log line" as CAP-I04's registry row currently states.** A minority of call sites (mostly `auth.go`, `invitations.go`, `admin.go`'s reload/role-change lines, `api.go`'s import) attach it; the majority — nearly every `"render X"` error across `record_crud.go`, `views.go`, `formfields.go`, `csv.go`, `coordplace.go`, `decisionstepper.go`, `processmap.go`, `handler.go`, all of `executor.go`'s `slog.Info("notify", ...)` / enqueue-failure lines, `scheduler.go`, `mailer.go` — do not. | `grep -rn "slog\." internal cmd` (145 hits); cross-referenced against `"correlation_id"` substring — well under half carry it | An incident that starts from a `"render form"` or `"enqueue notify outbox row"` error line cannot be joined back to the request or the `record_events` rows it produced except by timestamp proximity — the exact class of problem CAP-I04 exists to solve, just not solved for this majority of call sites |
| 2 | **No configurable log level.** `slog.NewJSONHandler(os.Stdout, nil)` passes `nil` options — fixed at `Info`, no `slog.LevelVar`, no `LOG_LEVEL` env var (`internal/config/config.go` has no such field). `Debug` can never be enabled in production for troubleshooting; `Info` can never be quieted. | `cmd/server/main.go:39`; `internal/config/config.go` field list | Standard 12-factor expectation; without it, diagnosing a live issue means redeploying with code changes, not flipping an env var |
| 3 | **No source location on log lines.** Same `nil` `HandlerOptions` — `AddSource: true` is never set. | `cmd/server/main.go:39` | An `Error` line like `"render list", "error", err` (37+ near-identical messages across handlers) gives no file:line to jump to; message text alone is not always unique enough to grep back to one call site |
| 4 | **Panic recovery reverts to an unstructured, uncorrelated stream.** `middleware.Recoverer` (chi's stock middleware, registered `main.go:125`) writes its panic report through its own internal logger — plain text to stderr, not through the process's `slog` JSON handler — for exactly the highest-value moment (an unhandled panic) to have a correlation id and be machine-parseable. | `main.go:125`, `r.Use(middleware.Recoverer)`; chi's `middleware.Recoverer` source has no `slog` integration | Reintroduces the "two differently-shaped outputs" problem the JSON-unification work (main.go's own header comment) was meant to eliminate, at the one moment (a crash) where a clean structured log matters most |
| 5 | **No documented log retention/shipping for the production process.** Logs go to `stdout` only; nothing in `DEVELOPMENT.md`, `ROADMAP.md`, or this repo states how `server-manager.sh` (referenced from `ROADMAP.md`'s Phase 6, lives outside this repo) captures, rotates, or retains that stream in production. | `ROADMAP.md` Phase 6 cutover sequence; no `logrotate`/systemd-journald reference found in this repo | `nfr-standards.md` states an explicit retention policy for `record_events` ("partitioned by month, retention per workspace") but the process log stream — which is where gaps #1–#4 above actually surface during an incident — has no analogous statement anywhere |
| 6 | **No metrics/SLO layer — the other named half of CAP-I04.** Per-request `duration_ms` exists on each access-log line, useful for inspecting one request, not for aggregate error-rate/latency-percentile tracking without external log aggregation. | `capability-registry.md` CAP-I04 row: "**SLO registry half — not started**"; Study 37 (2026-09-07) proposes an admin trace view (R12) built from `record_events` + outbox, which is a UI over the audit trail, not a metrics/SLO answer | Named already in the registry; restated here because gaps #1–#4 make even the correlation half less complete than the row currently claims |
| 7 | **PII (email addresses, real names) logged with no stated retention/redaction policy for the process log stream.** `auth.go:193,215` logs the raw `email` on login failure / no-membership; `identity`/`actor` (real names) appear throughout. | `auth.go:193`, `"login failed", ..., "email", email` | Reasonable for audit purposes on its own, but unlike `record_events` (which has a stated per-workspace retention policy), the process log stream carrying the same personal data has none — worth a stated policy, not silent acceptance |
| 8 | **Background-loop errors have no backoff/dedup.** `runScheduler` (1 min tick) and `runOutboxDispatcher` (2 sec tick) `slog.Error` unconditionally on every failing tick, per workspace, forever. | `cmd/server/main.go:276,281,286` (scheduler), `322,327,333,350,356,358,363,367,371` (outbox dispatcher) | A single workspace with a persistent problem (e.g. a bad DB constraint) logs an `Error` every 2 seconds indefinitely — floods the stream and any alerting built on "an `Error` line pages someone" |
| 9 | **`/health` is logged like real traffic.** `slogAccessLog` is a global `r.Use()` (`main.go:124`) with no exclusion list; a monitoring probe polling `/health` every few seconds produces one `"http request"` JSON line per poll forever. | `main.go:124,451` (`isPublicPath` exempts `/health` from *auth*, not from the access logger) | Minor, but pure noise inflation in the one stream everything else in this audit is trying to make more signal-dense |

---

## Recommended action list

Priority ordered by (a) how directly the gap undercuts CAP-I04's own stated claim and (b) cost to
close. This is a candidate list for `app/ROADMAP.md` sequencing, not a decision — which phase (or
whether a dedicated one is warranted) is the roadmap owner's call, same convention
`portal-ga3-code-quality-benchmark.md`'s own action list used.

1. **Attach `correlation_id` to every `slog` call site inside a request or background-tick
   context**, not just the current minority — closes gap #1, the one that most directly
   contradicts CAP-I04's registry claim. Mechanical: thread `middleware.GetReqID(ctx)` (or the
   workspace-tick equivalent) through call sites that already have `ctx` in scope; a couple
   (`format.go`'s computed-field warnings) currently don't receive one and would need it plumbed
   in.
2. **Make log level configurable** (`LOG_LEVEL` env var → `slog.LevelVar`, defaulting to `Info`)
   and **add `AddSource: true`** to the `HandlerOptions` — closes gaps #2 and #3 together, since
   both are the same `nil`-options call site (`main.go:39`).
3. **Route panic recovery through the same `slog` JSON handler.** Either a small custom recovery
   middleware (mirroring how `slogAccessLog` already replaced chi's stock access logger) or
   `middleware.RecovererWithOptions` if chi's version supports a compatible custom logger — closes
   gap #4.
4. **State a retention/shipping policy for the production stdout stream** — even if the answer is
   "systemd-journald with its own default retention, no additional rotation needed," write that
   down where `ROADMAP.md`'s Phase 6 cutover sequence or `DEVELOPMENT.md` already documents
   production operation, so it's a stated decision, not an unknown — closes gap #5.
5. **Add backoff or dedup to the two background-loop error paths** (`runScheduler`,
   `runOutboxDispatcher`) — e.g. log at `Warn` after the first failure per workspace and only
   re-escalate to `Error` after N consecutive failures, or a simple per-workspace exponential
   backoff before the next attempt — closes gap #8.
6. **Exclude `/health` from `slogAccessLog`** (or log it at `Debug` once level is configurable per
   item 2) — closes gap #9, cheap once item 2 lands.
7. **State a PII-in-logs policy** (what's acceptable to log for audit purposes — email, name — and
   for how long the process log stream itself is retained) — closes gap #7. Lowest priority: no
   incident or compliance requirement has forced this yet, per this project's own "Infer Before
   Configure" posture, but it should be a named, not silent, gap the way `nfr-standards.md`
   already names it for `record_events`.
8. **SLO/metrics layer** (gap #6) — largest single item, already tracked at the registry level as
   CAP-I04's unstarted half; not re-scoped here, just re-flagged as still open and now visibly
   larger than the row's "⚠️ partial" status suggests once items 1–7 are weighed against it.
