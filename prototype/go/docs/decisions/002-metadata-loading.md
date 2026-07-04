# ADR-002: Metadata Loading Strategy

**Status:** Prototype decision — deferred  
**Date:** 2026-07-04

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
