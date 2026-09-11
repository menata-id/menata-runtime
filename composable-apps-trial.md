# Composable Apps Trial

> Status: Draft v0.1 — Trial plan for the first real applications running on the Composable Application Runtime.
>
> This document is the implementation contract for the trial phase. It does not introduce a separate
> trial architecture. **Document Approval and Project Management are the first real applications of the
> Composable Application Runtime itself.** Their real user workloads are part of the evidence used to
> validate composability, execution planning, performance, and capability reuse.

## 1. Purpose

The trial exists to prove that Menata Runtime can build useful applications by composing reusable
runtime primitives across the data, experience, and behavior planes while keeping physical execution
bounded and efficient.

The trial therefore validates two things at the same time:

```text
application usefulness
        +
composable runtime correctness
        +
physical execution economy
```

The trial is successful only when all three hold together.

The architecture baseline is defined by:

- `007-composable-runtime-architecture.md` — normative architectural direction;
- `composable-runtime-blueprint.md` — transformation and benchmark plan;
- `capability-registry.md` — capability admission authority;
- the existing case portfolio and conformance suites — evidence and regression boundary.

## 2. Trial Principle

> **There is no trial-only application architecture.**
>
> Document Approval and Project Management are the first real applications running on the Composable
> Application Runtime. Any capability required by either application MUST be implemented as a reusable
> runtime primitive, composition mechanism, or explicitly admitted capability — never as application-
> specific execution logic.

This rule is stronger than "the applications use metadata". The applications must exercise the same
runtime path that future applications will use:

```text
Runtime Metadata
      ↓
Normalization / Compilation
      ↓
Domain + Data + Experience composition
      ↓
Dependency graph
      ↓
Composable Execution Planner
      ↓
Physical execution plan
      ↓
Data operations / actions
      ↓
Render plan
      ↓
Server-rendered experience
```

An application-specific shortcut from UI to database, an application-specific query executor, or an
application-specific renderer is outside the trial architecture.

## 3. What "Composable App" Means Here

For this trial, an application is considered composable when its behavior and experience can be
expressed through reusable runtime artifacts and composition rather than bespoke code paths.

The trial application surface may contain, where required by the case:

```text
Application
 ├── Workspace context
 ├── Navigation
 ├── Machines
 │    ├── Fields
 │    ├── Relations
 │    ├── Constraints
 │    ├── Events / Actions
 │    └── Permissions
 ├── Data
 │    ├── Views
 │    ├── Query / Projection requirements
 │    ├── Dataset semantics where admitted
 │    └── Filters / Sort / Pagination
 └── Experience
      ├── Pages
      ├── Layouts / Sections
      ├── Components
      ├── Bindings
      └── View composition
```

The exact metadata syntax may remain implementation-specific. The architectural requirement is that
these concerns have distinct semantic responsibilities and can be reused across both trial cases.

## 4. Common Runtime Path

Both applications MUST use the same runtime substrate.

```text
             DOCUMENT APPROVAL
                     │
                     │
                     ▼
             ┌───────────────┐
             │ Composable    │
             │ Application   │
             │ Runtime       │
             └──────┬────────┘
                    │
             ┌──────┴──────┐
             ▼             ▼
     execution plan     render plan
             │             │
             ▼             ▼
          database      UI renderer

             PROJECT MANAGEMENT
                     │
                     │
                     ▼
             ┌───────────────┐
             │ same runtime  │
             │ substrate     │
             └───────────────┘
```

There must not be:

```text
Document Approval → special workflow executor
Project Management → special board executor
```

when the required behavior can be expressed through common runtime primitives.

## 5. Trial Objectives

### 5.1 Application objectives

The two applications must each provide a usable end-to-end vertical slice for real work.

### 5.2 Runtime objectives

The trial must demonstrate:

1. metadata-driven business behavior;
2. reusable relations, fields, views, events, constraints, and permissions;
3. composable page and UI structures;
4. reusable data and presentation primitives across distinct domains;
5. predictable permission enforcement before optimization or caching;
6. a first-class execution-planning boundary;
7. bounded physical database work despite increasing logical composition;
8. no application-specific dispatch path to physical execution;
9. conformance and regression protection for admitted capabilities.

### 5.3 Evidence objectives

Real user work performed in the trial applications becomes benchmark evidence.

The benchmark must measure not only feature correctness but also the relationship between:

```text
logical composition
        ↕
physical operations
```

The expected property is:

> **Logical composition may grow faster than physical work when dependencies can be shared, batched,
> cached, or otherwise executed more efficiently.**

## 6. Trial Applications

## 6.1 Case 3 — Document Approval

Document Approval is the primary workflow/composition case. Existing repository evidence already
covers a substantial part of its composed experience, including dashboard composition, approval
progress, approver selection, ordering, and related document/step relationships.

### Trial v1 vertical slice

```text
Workspace
  ↓
Document Approval
  ↓
Approval Dashboard
  ├── KPI / status summary
  ├── pending approvals
  └── recent activity / related summary where supported
  ↓
Document List
  ↓
Document Detail
  ├── document metadata
  ├── approval progress
  └── current action(s)
  ↓
Submit Document
  ├── choose approver user / group
  ├── define or reuse approval steps
  └── preserve step ordering
  ↓
Approval Execution
  ├── sequential / parallel behavior as admitted
  ├── approve
  ├── reject
  └── status transition
```

The trial should focus on the smallest workflow surface that proves the runtime composition model.
Advanced PDF placement and template scenarios may be included when useful, but they are not a reason
to create a Document-Approval-specific runtime architecture.

### Composition evidence to preserve

Document Approval should exercise reusable primitives for:

- record detail composition;
- status and progress presentation;
- people/group references;
- collections of child records;
- activity/timeline-style presentation when admitted;
- action bars / decision controls;
- conditional action visibility;
- permission-aware presentation;
- workflow state and transition behavior.

The dashboard is especially important because it is a real composed page rather than a single View.

## 6.2 Case 19 — Project Management

Project Management is the cross-domain composition test. It intentionally uses a different user model
and visual interaction pattern from Document Approval while relying on the same runtime substrate.

### Trial v1 vertical slice

```text
Workspace
  ↓
Project
  ↓
Board
  ├── List A
  │    ├── Card
  │    │    └── Checklist items
  │    └── ...
  ├── List B
  │    └── ...
  └── ...
  ↓
Card Detail
  ├── title / description
  ├── member references
  ├── due date
  ├── checklist
  └── activity / actions where supported
```

Interactive behavior required for the trial includes, where already admitted or admitted through the
normal capability process:

- create/edit cards;
- move a card between lists;
- reorder cards within a list;
- open card detail;
- assign members;
- set due dates;
- manage checklist items;
- board representation of the same underlying records.

The board is a **composition of data and interaction**, not a separate persistence model merely because
its rendering differs from a list.

### Important runtime constraint

Manual card ordering must be expressed as reusable ordering semantics. It must not introduce a
Project-Management-specific storage or execution path merely to support drag-and-drop.

## 7. Shared Primitive Test

The two applications are intentionally different enough to expose accidental specialization.

The trial should actively look for opportunities to reuse the same runtime primitives across both.

| Concern | Document Approval | Project Management | Reuse target |
|---|---|---|---|
| record identity | document | card | generic record identity |
| person/group reference | approver | member | reference / people semantic |
| status | approval state | card/list state | status semantic |
| collection | approval steps | checklist / cards / lists | generic collection semantics |
| ordering | approval step order | card order | generic ordering semantics |
| detail page | document detail | card detail | record-detail composition |
| action control | approve/reject/submit | move/edit/complete | generic action/event model |
| conditional visibility | allowed decision | contextual card actions | expression + permission |
| page composition | approval dashboard | board | page/layout composition |
| activity | approval history | card activity | activity/timeline presentation |

This table is a trial hypothesis, not an automatic capability admission. A row becomes a reusable
capability only through the repository's capability lifecycle and evidence process.

## 8. Composable UI Requirements

The trial must prove that composability applies to the experience plane, not only to database access.

### 8.1 Page composition

A page may be assembled from independent runtime-owned pieces.

Example:

```text
Approval Dashboard
 ├── Status Summary
 ├── Pending Approval Collection
 └── Activity Collection
```

and:

```text
Project Board
 ├── Board header / controls
 ├── List collection
 │    └── Card collection
 └── contextual actions
```

These are logical compositions. The renderer remains responsible for turning normalized UI structures
into HTML/server-rendered output.

### 8.2 Semantic components

The trial should prefer semantic primitives such as:

```text
record summary
stat / metric
person / avatar
collection
section
activity / timeline
status
action
filter
sticky action area
board
```

over application-specific template fragments.

A visual pattern shared between the two applications should first be considered a reusable presentation
primitive before introducing another specialized component.

### 8.3 Composition boundaries

The runtime must preserve explicit boundaries between:

```text
Data meaning
UI composition
Behavior
Physical execution
```

A UI component must not become the owner of database strategy.

## 9. Composable Data Requirements

The trial should use the lightest data abstraction that is sufficient for the case.

Not every query needs to become a persisted Dataset.

The intended progression is:

```text
Machine/View-local query
        ↓
reusable projection / query semantics when required
        ↓
shared Dataset when multiple consumers need one semantic contract
```

The trial must demonstrate that a presentation can consume a semantic data requirement without
encoding renderer-specific SQL or storage details.

Examples include:

```text
approval dashboard
  → counts grouped by approval status

pending approval list
  → document + requester + current step + status

project board
  → ordered lists + ordered cards + selected card projection

card detail
  → card + members + checklist + due date
```

Security scope, filters, projection requirements, sort, and pagination are logical inputs to execution.

## 10. Composable Behavior Requirements

Business behavior in both applications should use the same runtime mechanisms for:

- actions;
- event dispatch;
- constraints;
- state transition validation;
- permissions;
- expressions;
- conditional behavior.

Examples:

```text
Document Approval
  Submit → validate → initialize approval state → activate required step(s)
  Approve → validate actor → transition step → recalculate status
  Reject → validate actor → transition document

Project Management
  Move Card → validate target list → change parent/order → renumber where required
  Complete Checklist Item → validate actor → update item state
```

The application metadata expresses the intent. Shared runtime behavior determines execution semantics.

## 11. Composable Execution Planner in the Trial

The Composable Execution Planner is part of the trial architecture from the beginning.

It may start with a deliberately small implementation, but the boundary must exist before the
applications become complex.

### 11.1 Minimum planner contract

```text
composition + data requirements + context
                ↓
          dependency graph
                ↓
       canonical dependencies
                ↓
    shared / batched / bounded plan
                ↓
         physical execution
```

Minimum v1 responsibilities:

1. collect dependencies;
2. canonicalize equivalent logical requests;
3. build a dependency DAG;
4. deduplicate exact-equivalent dependencies;
5. preserve permission/security scope;
6. enforce bounded concurrency;
7. expose plan/operation metrics.

The planner does not need an elaborate cost optimizer on day one. Cost-based selection can evolve
incrementally as the trial supplies real evidence.

### 11.2 Planned evolution

```text
V1  dependency graph + exact deduplication
V2  compatible query batching / coalescing
V3  cost estimation and strategy selection
V4  cache-aware planning
V5  adaptive planning from measured execution
```

The execution boundary must remain stable while implementation sophistication increases.

### 11.3 Physical-work invariant

The trial must detect and reject designs that produce:

```text
one component
  → one metadata load
  → one query
  → one goroutine
  → one round trip
```

for every logical node without examining opportunities for reuse.

The target is:

```text
many logical nodes
      ↓
shared dependency graph
      ↓
minimum safe physical operations
```

## 12. Performance and Benchmark Program

The applications are not only feature demonstrations. They are real benchmark sources.

### 12.1 Baseline dimensions

At minimum measure:

```text
request latency: p50 / p95 / p99
physical operations / request
SQL queries / request
rows scanned / request
rows returned / request
planner compilation time
plan-cache hit ratio where present
metadata-cache hit ratio
CPU / request
memory / request
database connection utilization
execution DAG width
```

### 12.2 Scenario families

The trial should produce at least these benchmark classes:

| Class | Example | Purpose |
|---|---|---|
| simple | single detail page | runtime overhead baseline |
| composed | dashboard with several sections | composition overhead |
| shared | several UI pieces reading overlapping data | dependency reuse |
| nested | board → lists → cards | multi-level dependency growth |
| interactive | move/reorder/approve | mutation execution |
| concurrent | multiple active users/workspaces | bounded concurrency |
| permission-sensitive | same logical request, different scopes | safe non-coalescing |
| heavy | analytics-style composed summary | isolation / degradation |

### 12.3 Naive-vs-planner comparison

Where practical, benchmark both:

```text
Mode A: independent logical execution
Mode B: planner-mediated execution
```

Compare physical work as well as latency. A faster response achieved by unsafe broad caching or
permission widening is a failure, not a success.

### 12.4 Real-workload evidence

User activity in the trial should be sampled into representative benchmark fixtures and scenarios.
The goal is not synthetic perfection; the goal is to prove that actual application compositions can
remain within the runtime's physical execution budgets.

## 13. Capability Admission During the Trial

The capability registry remains the authority. The trial does not bypass it.

For every new requirement:

```text
Trial requirement
      ↓
Is an existing capability sufficient?
      │
   yes│        no
      ▼         ▼
   compose   Can generic runtime composition express it?
                │
             yes│        no
                ▼          ▼
             compose   define/admit new capability
                           ↓
                       implement
                           ↓
                       conformance
```

The trial applications must not accumulate silent one-off behavior under the assumption that it will
be generalized later.

A temporary experiment may exist outside the production runtime path during research, but it must not
be presented as the trial implementation of the Composable Application Runtime.

## 14. Anti-Patterns Explicitly Out of Scope

The trial will not use any of the following as its architectural shortcut:

### 14.1 Application-specific executors

```text
if app == "approval" { ... }
if app == "project-management" { ... }
```

for behavior that should be represented through shared runtime semantics.

### 14.2 Application-specific database contracts

A trial application must not bypass the common metadata/data execution path with handwritten SQL
that becomes part of the application's architectural contract.

Handwritten SQL may still exist where an explicitly admitted infrastructure boundary requires it, but
such cases remain runtime-owned rather than application-owned.

### 14.3 Renderer-owned semantics

A board renderer must not define what a Card is. A workflow template must not define what Approval
Step semantics are. Those meanings belong to the Domain/Data planes.

### 14.4 Trial-only component registries

The trial must not introduce a dynamic application-local component registry merely to make pages easy
to assemble. The renderer should consume normalized runtime composition according to the architecture
specified in `007-composable-runtime-architecture.md`.

### 14.5 "Build first, compose later"

Do not first ship either case as a conventional specialized application and retrofit composability
afterwards. That would test migration rather than the intended runtime architecture.

## 15. Definition of Done

The trial is complete when all of the following are true.

### 15.1 Document Approval

- end-to-end core approval flow is usable by real users;
- core workflow behavior is metadata/runtime driven;
- composed dashboard/detail experiences run through the common runtime;
- permissions are enforced through shared runtime mechanisms;
- no Document-Approval-specific physical executor is required.

### 15.2 Project Management

- end-to-end core board workflow is usable by real users;
- board, lists, cards, and detail views use common runtime composition;
- move/reorder behavior uses generic ordering/behavior semantics;
- no Project-Management-specific physical executor is required.

### 15.3 Cross-case proof

- both applications share runtime primitives;
- at least one meaningful primitive is demonstrated across both domains;
- new presentation composition can be introduced without rewriting the data/execution layer;
- data can be consumed by multiple presentation forms without duplicating business semantics;
- execution planning is present on the live runtime path;
- physical-work metrics are collected;
- benchmark results show bounded execution under representative composition and concurrency.

### 15.4 Governance proof

- capabilities introduced by the trial are represented in the capability registry;
- conformance tests cover admitted behavior;
- trial-specific hacks are either removed or explicitly converted into admitted generic primitives;
- architectural claims are marked PROVEN only when implementation/conformance evidence exists.

## 16. Implementation Streams

Work should proceed in parallel but remain coupled through the common runtime path.

### Stream A — Runtime substrate

Strengthen and stabilize:

- metadata compilation/normalization;
- relations;
- views and data requirements;
- events/actions;
- constraints;
- permissions;
- page composition.

### Stream B — Composable UI

Implement the minimum reusable experience primitives demanded by both cases:

- sections;
- record summaries;
- collections/lists;
- detail composition;
- status/progress;
- people/avatar presentation;
- activity/timeline where admitted;
- board composition;
- action areas;
- context-aware bindings.

### Stream C — Composable Execution

Implement:

- dependency model;
- dependency DAG;
- canonical dependency identity;
- exact deduplication;
- bounded concurrency;
- physical operation accounting;
- query batching/coalescing as evidence demands;
- cost-aware planning as evidence demands.

### Stream D — Trial Applications

Build Document Approval and Project Management directly on Streams A–C.

There is no separate "application framework" layer between the cases and the runtime.

## 17. Suggested Sequence

```text
1. Freeze the common runtime contract for the trial
2. Confirm the minimum reusable primitives already available
3. Build Document Approval vertical slice on the common runtime
4. Build Project Management vertical slice on the same runtime
5. Add missing generic primitives only where the cases force them
6. Run conformance after each admitted capability
7. Enable planner dependency graph on all composed requests
8. Measure real workloads
9. Optimize shared/batched execution based on evidence
10. Reconcile proven capabilities into the registry and roadmap
```

The important sequencing rule is:

> **The architecture boundary comes first; planner sophistication comes later.**

The trial must never depend on a future planner that does not exist. A minimal correct planner seam is
part of the initial runtime, while optimizations can evolve from measured workload evidence.

## 18. Exit Criteria for the Next Phase

After the first real user trial, the repository should be able to answer with evidence:

1. Which runtime primitives were sufficient across both cases?
2. Which new capabilities were genuinely generic and admitted?
3. Which composition patterns caused physical fan-out?
4. How much dependency reuse did the planner achieve?
5. Where did batching improve execution, and where did it hurt?
6. Which UI composition patterns deserve reusable primitives?
7. Which remaining gaps are architectural, and which are simply missing capabilities?
8. Can a third application be built without introducing an application-specific execution path?

The final question is the strongest trial signal:

> **Can Menata Runtime build the next application by composing what the first two applications proved,
> instead of rebuilding the runtime around the third application's special case?**

## 19. Related Documents

- `007-composable-runtime-architecture.md`
- `composable-runtime-blueprint.md`
- `capability-registry.md`
- `capability-lifecycle.md`
- `case-portfolio.md`
- `benchmarks/029-composed-view-component-inventory.md`
- `benchmarks/030-ui-subcomponent-decomposition-criteria.md`
- `benchmarks/020-ui-interaction-cluster-proof.md`
- `prototype/go/docs/examples/README.md`
- `prototype/go/docs/examples/pm-list.yaml`

## 20. Architectural Statement

The trial is not a demonstration application sitting beside the Composable Application Runtime.

It is the first production-shaped workload **of** that runtime.

```text
                 Composable Application Runtime
                              │
                 ┌────────────┴────────────┐
                 ▼                         ▼
         Document Approval         Project Management
                 │                         │
                 └────────────┬────────────┘
                              ▼
                    real user workloads
                              │
                              ▼
                composability + performance evidence
                              │
                              ▼
                    next runtime evolution
```

That is the purpose of the trial: not merely to show that the two applications work, but to prove that
they work **because the runtime is composable** and that the same composability can remain operationally
efficient under real use.
