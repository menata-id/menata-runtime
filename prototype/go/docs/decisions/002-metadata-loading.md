# ADR-002: Metadata Loading Strategy

**Status:** Option A implemented (see Status update below); Option C still deferred
**Date:** 2026-07-04

## Status update (2026-08-22)

Option A (admin-triggered reload) is implemented and conformance-proven — `CAP-X04` in
`../../../../capability-registry.md`, proof in `../../../../benchmarks/015-metadata-live-reload-
proof.md` (T151–T153). `POST /admin/reload` re-runs `loader.LoadAll` and swaps the interpreter via
`atomic.Pointer[Interpreter]` (`internal/interpreter/store.go`) exactly as the "Constraint on swap"
section below anticipated; a failed reload leaves the old interpreter serving untouched, surfaced
to the admin as a 500. This was prompted directly by CAP-W07 (`change_policy`) turning out to
depend architecturally on live reload existing first, not the other way around.

Option C (`LISTEN/NOTIFY`, still this ADR's own recommended long-term answer) remains unbuilt —
tracked as `CAP-X11` (❌), bundled with the lazy per-workspace loading it would naturally pair with.
Option B was never built and is still not recommended, unchanged from the original recommendation
below.

## Context

The runtime builds its Application Model (machines, fields, events, constraints, permissions, views)
by reading Runtime Metadata from PostgreSQL at startup. After that, the in-memory interpreter is
never refreshed until the process restarts.

```go
// cmd/server/main.go
workspaces, err := loader.LoadAll(context.Background())
interp := interpreter.New(workspaces)          // frozen for process lifetime
```

This means: **adding or changing metadata requires a server restart to take effect.**

## Decision (prototype)

Load once at startup. Restart required to pick up metadata changes.

Acceptable for the prototype phase — the friction is low (restart is fast) and the simplicity
avoids premature complexity.

## Options for production

| Option | Mechanism | Tradeoff |
|--------|-----------|----------|
| **A — Admin endpoint** | `POST /admin/reload` swaps the interpreter atomically | Manual trigger, explicit, no background overhead. Best for low-frequency changes (new machine = intentional deploy). |
| **B — Periodic poll** | Background goroutine re-loads every N seconds, swaps if changed | Automatic but adds constant DB queries and up-to-N-second lag. Simple to implement. |
| **C — PostgreSQL LISTEN/NOTIFY** | DB trigger fires `pg_notify('metadata_changed', '')` on any metadata table INSERT/UPDATE; server goroutine listens and reloads | Near-instant, zero polling overhead. Correct model: DB is the source of change, DB notifies. More infrastructure to wire up. |

## Recommendation for production

Option C (`LISTEN/NOTIFY`) is the right long-term answer. It treats the DB as the authority,
has no polling overhead, and reacts within milliseconds of a metadata change.

Option A is a useful addition regardless — useful for forced reloads and operational control
(e.g., after a bulk migration).

Option B alone is not recommended for production.

## Constraint on swap

Whichever option is chosen, the interpreter swap must be **atomic** to avoid serving a partially
built model mid-request. Use `sync/atomic` or a read-write mutex:

```go
// atomic pointer swap (Option A or C)
var interp atomic.Pointer[interpreter.Interpreter]
interp.Store(interpreter.New(workspaces))

// ... on reload:
newInterp := interpreter.New(newWorkspaces)
interp.Store(newInterp)   // atomic — in-flight requests finish on old interp
```
