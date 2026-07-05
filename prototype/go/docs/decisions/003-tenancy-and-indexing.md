# ADR-003: Tenancy and Indexing Strategy

**Status:** Accepted (for the next runtime iteration)
**Date:** 2026-07-04
**Source study:** `runtime/benchmarks/004-scale-architecture-study.md` (Study 8)

## Context

The prototype loads all metadata eagerly at boot, stores records in a single
`records` table keyed only by `machine_id`, and has no workspace dimension on
data. Analysis identified the first walls at scale: boot-time metadata load,
JSONB filter sequential scans, unenforceable workspace isolation, and noisy
neighbors on the shared pool.

## Decision

1. **Tenancy: shared schema + `workspace_id` (Option A)**, hardened by
   PostgreSQL Row-Level Security and a `PARTITION BY HASH (workspace_id)`
   layout from the first migration of the next iteration.
   Database-per-workspace remains the documented escape hatch for heavy or
   regulated tenants (workspace → DSN routing map).

2. **Indexing:** composite base indexes
   (`workspace_id, machine_id, created_at` / `status`), GIN `jsonb_path_ops`
   on `data`, `status` promoted to a real column, and **metadata-driven
   expression indexes** — hot fields are read from view `filter` /
   `default_sort` / reference declarations and reconciled against
   `pg_indexes` (CAP-X10).

3. **Metadata loading:** lazy per-workspace load with per-workspace cached
   interpreters (`atomic.Pointer`, singleflight, LRU) invalidated via
   `LISTEN/NOTIFY` — unifying ADR-002's live-reload Option C with the scale
   cache (CAP-X11). The boot-time `LoadAll` is retired.

4. **Fairness:** per-workspace concurrency semaphore in front of the shared
   pool; reports run on a separate small pool with `statement_timeout`.

## Consequences

- Handlers stay unchanged; workspace resolution and RLS context live in middleware.
- Serving is stateless per replica — horizontal scaling is load-balancer-only.
- Migration fan-out, schema catalogs, and per-tenant connection storms (Options B/C) are avoided.
- Per-tenant backup/restore is harder under Option A — accepted; the escape hatch covers tenants who need it.
- Pass thresholds and the load-test matrix are defined in the source study; the decision is falsifiable against them.
