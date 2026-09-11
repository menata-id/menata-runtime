# Composable Runtime Architecture — Transformation Blueprint

> Status: v1.2 — adds the **Composable Execution Planner** as a first-class transformation phase and performance proof boundary, cross-linked to `007-composable-runtime-architecture.md` v0.4. 007 defines *what the architecture must be*; this document defines *how the runtime gets there and proves it*. Previously v1.1 — cross-linked to 007, three composability planes, current-state inventory, and evidence-gated phased plan. Created: 2026-09-11.

> **What this document is.** The Tier 3 companion to **`007-composable-runtime-architecture.md`** (Tier 1). 007 defines the model, contracts, and invariants. This document defines the current-state gaps, sequencing, forcing conditions, and benchmark program needed to reach that target.

> **What this document is not.** A capability admission, and not itself the normative spec. Nothing here is built by this document, and nothing here skips `capability-lifecycle.md` §2's A1–A5 test or the repository's "declare targets first" discipline. Every phase names its own forcing condition; where no forcing condition exists yet, the phase is sequencing information only.

---

## 1. Where this comes from

| Source | What it contributed |
|---|---|
| `007-composable-runtime-architecture.md` | Normative three-plane architecture, Data/UI/Behavior separation, UI IR, Static Component Registry seam, physical execution boundary, and now the **Composable Execution Planner** plus performance invariants |
| `prototype/objectstack/docs/composable-view-proposal-reconciliation.md` §1–§10 | Settled that the target is composable metadata without a dynamic client-side plugin/canvas architecture; identified Query/Projection, UI IR, and composability benchmark gaps |
| `capability-registry.md` `CAP-V10` | Shipped page composition precedent; shows the experience composition substrate already exists in bounded form |
| `capability-registry.md` `CAP-V22` | Proposed semantic dataset; the primary candidate for reusable Data-plane semantics |
| `capability-registry.md` `CAP-C13` | Existing CEL expression primitive reusable by Projection, Query, constraints, and bindings |
| `capability-registry.md` `CAP-X10` | Proposed metadata-driven index management; physical optimization input for the planner |
| `benchmarks/004-scale-architecture-study.md` (Study 8) | Existing scale/performance evidence: lazy metadata loading, cache, singleflight, RLS, index strategy, pool fairness, pagination, analytics isolation, and P95 targets |
| `internal/metadata/compile.go` (`CAP-W01`) | Existing proof that declarative metadata can compile into lower-level runtime primitives at load time |
| `capability-lifecycle.md` §2 | Existing A1–A5 admission/evidence discipline |

---

## 2. Target architecture

Three composability planes meet in one compiler discipline, with the execution planner as the physical-economy boundary:

```text
                         MENATA RUNTIME
                              │
        ┌─────────────────────┼─────────────────────┐
        ▼                     ▼                     ▼
   DOMAIN PLANE           DATA PLANE           EXPERIENCE PLANE
        │                     │                     │
     Machine               Dataset                Page
     Field                 Query                   Layout
     Event                 Projection              Component
     Constraint            Expression              Binding
     Permission            Relation                 Context
        │                     │                     │
        └─────────────────────┼─────────────────────┘
                              ▼
                   RUNTIME COMPILER / NORMALIZER
                              │
                  Logical Composition / Data IR
                              │
                              ▼
                COMPOSABLE EXECUTION PLANNER
                              │
                ┌─────────────┼─────────────┐
                ▼             ▼             ▼
             shared        batched       bounded
             work           work           work
                └─────────────┼─────────────┘
                              ▼
                       Costed Execution Plan
                              │
                 ┌────────────┴────────────┐
                 ▼                         ▼
            Physical Data              Render Plan
             Operations                    │
                 │                        ▼
                 ▼                     Renderer
              Postgres
```

The target invariant is:

> **Many logical components, as few physical operations as semantics and correctness permit.**

---

## 3. Current-state inventory

| Concept | Plane | Status | Evidence | Gap |
|---|---|---|---|---|
| Machine/Field/Event/Constraint/Permission | Domain | ✅ Built | `runtime-metadata-schema.md`, model/runtime code | Stable core |
| `View`/`ViewConfig` | Data + Experience | ✅ Built, but only composition unit historically | `app/internal/model/model.go` | Reuse remains View-centric |
| `page` composition | Experience | ✅ Built (`CAP-V10 Tier 2`) | registry/conformance | Single-level today |
| Layout | Experience | 🟡 Partial | CAP-V10 | Closed vocabulary still limited |
| Static content | Experience | ✅ Built | CAP-V10 | Closed by design |
| CEL expression | Data/Behavior | ✅ Built (`CAP-C13`) | capability registry | Reusable substrate already exists |
| Semantic Dataset | Data | ❌ Proposed | `CAP-V22` | Needs forcing case |
| Bare Query/Projection reuse | Data | ❌ Not designed | reconciliation §9.2 | Needs forcing case |
| UI IR | Experience | ❌ Not designed | 007 target | One renderer means no immediate forcing case |
| Query Planner / Cost Model | Data | 🟡 Partial | Study 8 + CAP-X10 | Existing scale ideas not unified around composed dependency graph |
| **Composable Execution Planner** | Cross-plane | ❌ Not implemented as a first-class boundary | 007 v0.4; Study 8 provides pieces | Need dependency DAG, coalescing, batching, bounded concurrency, costing, and proof harness |
| Recursive composition | Experience | ⛔ Explicitly out of scope | CAP-V10 Tier 2 | Revisit only with real case |

---

## 4. Composition-performance problem statement

A composable application naturally produces an experience tree:

```text
Page
 ├── KPI Revenue
 ├── KPI Orders
 ├── Customer Summary
 ├── Recent Orders
 └── Approval Queue
```

A naive implementation may turn that into:

```text
5 logical components
→ 5 metadata resolutions
→ 5 query plans
→ 5 database round trips
```

The runtime must instead derive a dependency graph and optimize physical work:

```text
Composition Tree
      ↓
Dependency DAG
      ↓
shared / compatible dependencies
      ↓
coalescing + batching + bounded parallelism
      ↓
costed physical plan
      ↓
minimum reasonable physical work
```

This is the core reason the Composable Execution Planner is a first-class boundary rather than an implementation detail of the Query Planner.

---

## 5. Phased evolution plan

| Phase | What it does | Depends on | Forcing condition | If met today? |
|---|---|---|---|---|
| **0. Grammar-area decision** | Decide whether Dataset is a new Grammar area or folds under View | Nothing | Owner taxonomy decision | Open |
| **1. Semantic dataset** | Admit/build `CAP-V22` for reusable semantic aggregates | Phase 0 | Case needs same aggregate in multiple presentations | Not yet |
| **2. Bare Query/Projection reuse** | Make row-level data shape addressable independently of View | Phase 1 | Case needs same rows across different presentations | Not yet |
| **3. Layout vocabulary** | Extend stack/grid/columns/split/tabs where real composed-page work forces it | Nothing | Existing queued design trigger | Low priority |
| **4. Context/scope propagation** | Parent→child context for real parent-scoped composition | CAP-V10 | Real case requires scope propagation | Not yet |
| **5. Recursive composition depth** | Allow nested page composition if a real case requires it | Phase 4 | Real multi-level nesting case | Not yet |
| **6. Composable Execution Planner foundation** | Derive Dependency DAG from composition + Data/UI IR; define canonical dependency identity; add plan objects and execution budgets | Phases 1/2 | A composed case produces repeated logical dependencies or measurable fan-out | **Design now; implementation when forced by case/benchmark** |
| **7. Planner optimization** | Add dependency deduplication, query coalescing, batching, bounded concurrency, and cost-based strategy selection | Phase 6 | Benchmark shows meaningful physical-work reduction opportunity | Not yet |
| **8. Query/index pushdown** | Dataset-aware projection/filter pushdown and metadata-driven index hints | Phase 1/2/6 | Measured DB cost pressure at target scale | Not yet |
| **9. UI Intermediate Representation** | Formalize normalized UI compile step | Phases 3–5 | Second renderer/builder or other explicit forcing case | Not yet |
| **10. Composability benchmark** | Compare naive vs planner-enabled execution across shared and independent dependency patterns | Phases 6–8 | Enough implementation exists to measure | Design now; execute with planner prototype |

The planner is deliberately split into **foundation** and **optimization** so the execution boundary can be established without prematurely building an elaborate optimizer. The first useful milestone is a correct dependency DAG and hard execution budgets; optimization follows measured evidence.

---

## 6. Composable Execution Planner — implementation target

### 6.1 Planner input

The planner should consume the normalized result of composition and data binding:

```text
UI IR / Composition Tree
          +
Data IR / logical query requirements
          +
Context + permission scope
          ↓
Composable Execution Planner
```

### 6.2 Planner stages

```text
1. Collect declared dependencies
2. Resolve stable identities
3. Canonicalize equivalent requests
4. Build dependency DAG
5. Detect shared/coalescible work
6. Group compatible operations
7. Estimate cost
8. Apply execution budgets
9. Select physical strategies
10. Emit executable plan
```

### 6.3 Physical strategies

The planner may choose among:

```text
single live SQL
shared query execution
batched query
indexed SQL
cache hit
materialized representation
bounded parallel execution
in-process evaluation
async job
explicit rejection/degradation
```

The physical strategy must remain invisible to metadata authors.

### 6.4 Stable dependency identity

A planner dependency identity should be derived from canonical logical semantics, not renderer implementation. At minimum, the identity should account for:

```text
source
security scope
normalized filter
parameters
projection
sort/pagination
grouping/measures
metadata/interpreter version
```

This makes safe deduplication and plan reuse possible without conflating semantically different requests.

### 6.5 Coalescing rule

The planner should coalesce requests when one physical operation can satisfy all consumers without violating:

- projection requirements;
- row/cardinality semantics;
- sort/pagination semantics;
- security scope;
- freshness requirements;
- latency budgets.

A broader projection is not automatically cheaper than two narrow queries. The planner must decide using measured or bounded cost rather than a blanket "fewer queries is always better" rule.

### 6.6 Batching rule

Batch compatible operations when round-trip reduction dominates the additional complexity. Batch boundaries must preserve transaction, permission, parameter, and failure semantics.

### 6.7 Bounded concurrency

Independent nodes may execute concurrently, but concurrency is budgeted by request, workspace, and database capacity. The planner should work with the scale architecture's per-workspace semaphore and analytics/OLTP resource separation rather than bypassing them.

### 6.8 Interactive budget rule

A plan classified as interactive must have explicit budgets for at least:

```text
physical operations
estimated rows
estimated DB work
estimated memory
parallelism width
```

If the budget cannot be met, the planner must choose an explicit fallback such as cache, batching, lazy loading, async work, or clear rejection/diagnostic.

---

## 7. Relation to existing Scale Study

The planner is not a competing performance subsystem. It is the common boundary that lets existing performance mechanisms participate in composable execution.

| Existing mechanism | Planner responsibility |
|---|---|
| lazy per-workspace metadata loading | consume compiled immutable interpreter rather than reload metadata per component |
| singleflight | prevent duplicate cold compilation |
| batch metadata loading | avoid metadata N+1 while constructing the dependency graph |
| RLS/workspace scope | include security scope before deduplication/caching |
| expression indexes / CAP-X10 | choose indexed physical query when cost-effective |
| pagination | bound row retrieval and render work |
| workspace semaphore | bound execution concurrency |
| analytics pool | isolate heavy Dataset/report operations |
| cache | reuse results only when security/freshness semantics match |

The Scale Study's current target is 100 workspaces × 50 machines × 1M records on a modest server first; it also proposes p95 list < 200 ms, lazy boot under 5 s, and explicit cross-workspace RLS probes. The composability benchmark should extend those scale tests rather than replace them.

---

## 8. Benchmark program

The key experiment is not simply "does a component render faster?" It is:

> **As logical composition increases, does physical server work grow sublinearly through dependency reuse, batching, and bounded execution?**

### 8.1 Core scenarios

| Scenario | Logical shape | What it proves |
|---|---|---|
| A | 1 component → 1 dataset | baseline overhead |
| B | 10 components → 10 independent datasets | fan-out cost |
| C | 10 components → 3 shared datasets | dependency sharing |
| D | 5 components → same dataset, compatible projections | projection/query coalescing |
| E | dashboard with mixed OLTP + analytics | resource isolation |
| F | 100 concurrent workspaces, cold start | metadata/cache behavior |
| G | identical logical request, incompatible security scopes | safe non-coalescing |
| H | 30+ components with bounded budgets | admission/degradation behavior |

### 8.2 Compare two execution modes

```text
Mode 1: naive
component → independent physical operation

Mode 2: planner
composition → dependency DAG → shared/batched/bounded plan
```

The planner is only considered successful where it preserves semantics/security and improves or maintains relevant SLOs.

### 8.3 Metrics

```text
p50 / p95 / p99 latency
physical operations / request
queries / request
rows scanned / request
rows returned / request
CPU / request
memory / request
DB connection utilization
cache hit ratio
metadata/plan cache hit ratio
planner compile time
execution DAG width
estimated vs actual cost error
```

### 8.4 New composability KPIs

The existing benchmark direction can be extended with:

```text
Query Reuse Ratio
= logical data requests / physical executions

Physical Fan-out Ratio
= physical operations / top-level request

Composition Execution Cost
= normalized physical work / composed experience

Planner Benefit
= (baseline physical work - planner physical work)
  / baseline physical work
```

These are measurement definitions, not guarantees; normalization must be documented per benchmark.

---

## 9. Failure modes the planner must prevent

| Failure mode | Required control |
|---|---|
| one-query-per-component explosion | dependency DAG + deduplication |
| metadata N+1 | pre-resolved immutable graph + batch loading |
| unlimited goroutines | bounded concurrency budget |
| shared cache leaks | security scope in dependency/cache identity |
| huge projection | projection minimization |
| huge list retrieval | pagination/row budget |
| analytical query starving OLTP | workload class + separate resource pool |
| expensive composition accepted blindly | composition/execution cost budget |
| cache/reuse after metadata change | versioned identity + invalidation |
| broad query chosen merely to reduce query count | cost-based coalescing |

---

## 10. What does not change

- No dynamic Component Registry or client-side Canvas is introduced.
- `View`, `Machine`, `Page`, and the capability registry remain valid existing abstractions.
- Existing admission gates remain authoritative.
- Physical plans remain runtime-internal; business metadata does not encode PostgreSQL-specific plans.
- Server-side authorization, filtering, and aggregation remain authoritative.

The Composable Execution Planner changes the **execution boundary**, not the semantic ownership model.

---

## 11. Open questions

The following need benchmark evidence:

- when query coalescing becomes more expensive than multiple narrow queries;
- how to estimate JSONB/expression-index cost accurately enough for planning;
- how far batching should go before PostgreSQL planning overhead dominates;
- when a cache hit is preferable to a fresh shared execution;
- how much planner compilation cost is worth paying for a request;
- what execution budgets provide useful protection without rejecting legitimate complex applications;
- whether physical plan fragments should be cached persistently or only in-process.

---

## 12. Disposition

**The Composable Execution Planner is a PROPOSED architectural boundary.** It is not claimed as implemented by this document. The immediate architectural decision is to make the boundary explicit so that future composable capabilities cannot bypass performance governance.

The correct implementation order is:

```text
first: correct dependency graph
second: hard budgets / isolation
third: measured deduplication + batching
fourth: cost-based physical selection
fifth: benchmark at representative scale
```

This keeps performance as a property of the composable runtime architecture itself rather than a collection of after-the-fact optimizations.
