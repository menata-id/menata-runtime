# Multi-Workspace Scale & Performance Architecture

> Study 8 deliverable (`../capability-roadmap.md`).
>
> If all capabilities are used and workspaces multiply: what data structure and
> programming strategy delivers the best performance with optimal resources?
>
> Status: v0.1 | Created: 2026-07-04

**Reference scale target:** 100 workspaces × 50 machines × 1M records total, on a single modest server first, horizontal later.

---

# What breaks first (analysis of the current prototype)

Ordered by which wall is hit earliest:

| # | Breakage | Where | At roughly |
|---|----------|-------|-----------|
| 1 | **Eager metadata load at boot** — `LoadAll` walks every workspace → application → machine, issuing ~6 queries per machine in nested loops | `metadata/loader.go` | 5,000 machines ≈ 30k queries at startup; minutes of boot, all workspaces paying for each other |
| 2 | **JSONB filter scans** — any view filter/search over `records.data` fields is a sequential scan | `store/record_store.go` | noticeable at ~50k rows per machine, painful at 1M total |
| 3 | **No workspace dimension on data** — `records` has `machine_id` only; workspace isolation (CAP-X06) is unenforceable at the data layer | schema | first multi-tenant customer |
| 4 | **Single shared pool, no fairness** — one workspace's heavy report starves everyone (noisy neighbor) | `db/db.go` | first heavy tenant |
| 5 | Full-page renders of unbounded lists (no pagination — CAP-R05) | handler | 10k rows in one machine |

---

# Tenancy models

| | A — Shared schema + `workspace_id` | B — Schema per workspace | C — Database per workspace |
|---|---|---|---|
| Isolation | Logical (column + RLS) | Namespace | Physical |
| Resource efficiency | **Best** — shared buffers, one pool | Medium | Worst — N pools, N sets of buffers |
| Migration cost | 1 migration | N × migrations (fan-out, partial-failure states) | N × migrations + connection storm |
| Postgres catalog pressure | None | Catalog bloat at 1000s of schemas | Connection/instance limits |
| Noisy neighbor | Needs mitigation | Partial | Solved |
| Per-tenant backup/restore | Hard (row extraction) | Medium | Trivial |
| Ops complexity | **Lowest** | Medium | High |

**Decision: Option A** — shared schema with `workspace_id` on every data table, hardened by:

1. **PostgreSQL Row-Level Security** — `CREATE POLICY ws_isolation ON records USING (workspace_id = current_setting('app.workspace_id')::uuid)`. Isolation enforced by the database, not by developer discipline on every query. This *implements* CAP-X06 rather than auditing for it.
2. **Partition-ready layout** — `records` declared `PARTITION BY HASH (workspace_id)` from the first migration of the next iteration. Cheap when small; already correct when large.
3. **Hybrid escape hatch** — a heavy or regulated tenant can be moved to a dedicated database later (same schema, a `workspace → dsn` routing map in config). Stripe/Citus-style: shared by default, dedicated by exception.

Options B/C are rejected as *defaults* but C remains the documented escape hatch.

---

# Data structure strategy

## records table (next iteration)

```sql
CREATE TABLE records (
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id uuid NOT NULL,            -- denormalized: isolation + partitioning key
    machine_id   text NOT NULL,
    data         jsonb NOT NULL,
    status       text,                      -- promoted out of JSONB: hottest field everywhere
    created_at   timestamptz NOT NULL DEFAULT now(),
    updated_at   timestamptz NOT NULL DEFAULT now()
) PARTITION BY HASH (workspace_id);

CREATE INDEX ON records (workspace_id, machine_id, created_at DESC);  -- every list view
CREATE INDEX ON records (workspace_id, machine_id, status);            -- status filters
CREATE INDEX ON records USING GIN (data jsonb_path_ops);               -- containment queries
```

`status` is promoted to a real column because every case (1–10) filters or badges on it — the one universal hot field.

## Metadata-driven expression indexes `[SCALE FINDING]`

The runtime knows which JSONB fields are hot — **the metadata says so**: view `filter` blocks (CAP-V09), `default_sort` fields (CAP-V04), and reference fields (CAP-F13) name exactly the fields that will appear in WHERE/ORDER BY. Therefore index management can be metadata-driven:

```sql
-- derived automatically from vw_mt_overdue's filter on fld_mt_status,
-- or a sort on fld_lr_start_date:
CREATE INDEX CONCURRENTLY idx_mch_leave_fld_start
    ON records ((data->>'fld_lr_start_date'))
    WHERE machine_id = 'mch_leave_request';
```

The loader diffs declared hot fields against `pg_indexes` and creates/drops expression indexes accordingly — the same reconciliation idea as Kubernetes (architecture-benchmark.md), applied to indexes. Registered as **CAP-X10**.

## Line items (CAP-F16) and event log

- Line items: child rows in the same `records` table with a `parent_record_id` column (indexed), not nested JSONB arrays — keeps aggregate constraints (CAP-C10) and reports (CAP-V13) in SQL.
- `record_events` grows append-only forever: partition by month (`occurred_at`), retention policy per workspace.

---

# Programming strategy

## Metadata: lazy per-workspace loading + cache `[SCALE FINDING]`

Replace boot-time `LoadAll` with:

1. **Lazy load** — a workspace's Application Model is built on its first request.
2. **Per-workspace cache** — `map[workspaceID]*atomic.Pointer[Interpreter]`, LRU-evicted; `singleflight` so a cold workspace's concurrent requests trigger exactly one load.
3. **Batch the loader** — load a whole workspace's machines in ~6 set-based queries (`WHERE machine_id = ANY(...)`) instead of 6 × N nested queries.
4. **Invalidation via LISTEN/NOTIFY** (ADR-002 Option C) — `pg_notify('metadata_changed', workspace_id)` evicts exactly one workspace's cache entry. Live reload and scale caching are the same mechanism.

Registered as **CAP-X11**.

## Request path

- Middleware resolves workspace (subdomain/path) → sets `app.workspace_id` on the connection (RLS) → picks the cached interpreter. Handlers stay unchanged.
- Interpreter is immutable per version → **stateless serving** → horizontal scale is "run more replicas behind a load balancer"; only Postgres is shared.
- Pool fairness: per-workspace semaphore (max concurrent queries per tenant) in front of the shared `pgxpool` — cheap noisy-neighbor mitigation before anything fancier.
- Reports (CAP-V13) run on a second, smaller pool with `statement_timeout` — analytics can be slow, but it must never exhaust the OLTP pool.

---

# Load-test plan

Synthetic generator + measurement harness (extends the conformance suite's philosophy: claims need executable proof):

1. **Generator** — seed W workspaces × M machines (reuse Case 1/2 metadata as templates) × R records with realistic JSONB payloads. Parameterized: `W=1,10,100; M=10,50; R/machine=1k,10k,100k`.
2. **Workload mix** — 70% list (with filter), 20% detail, 8% create, 2% event trigger; driven by `vegeta` or `k6`; 10-minute sustained runs.
3. **Matrix** — each scale point run twice: baseline indexes vs metadata-driven expression indexes (CAP-X10 on/off).
4. **Metrics** — p50/p95/p99 latency per endpoint class, Postgres `pg_stat_statements` top queries, cache hit ratio, boot time, RSS.
5. **Pass thresholds (first iteration)** — p95 list < 200 ms at W=100/R=1M with X10 on; boot < 5 s (lazy loading); zero cross-workspace rows under RLS probe (isolation test: deliberately query with wrong `app.workspace_id`).

---

# Registry impact

| ID | Capability | Note |
|----|-----------|------|
| CAP-X10 | Metadata-driven index management (hot fields from view filters/sorts → expression indexes, reconciled) | `[SCALE FINDING]` |
| CAP-X11 | Lazy per-workspace metadata loading + cache (singleflight, LRU, LISTEN/NOTIFY eviction) | `[SCALE FINDING]` — supersedes the "load once at boot" model; same mechanism as ADR-002 live reload |

CAP-X06 (workspace isolation) gains its implementation strategy: RLS. CAP-R05 (pagination) is confirmed as a scale prerequisite, not just UX.

Decision recorded as ADR-003 in `prototype/go/docs/decisions/003-tenancy-and-indexing.md`.
