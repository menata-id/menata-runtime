# Composable Runtime Roadmap

> Structured implementation roadmap for turning `007-composable-runtime-architecture.md` into the next generation of Menata Runtime without breaking the proven `app/` baseline.
>
> This roadmap complements `composable-runtime-blueprint.md`. The blueprint explains the architectural transformation and current-state gaps; this document is the implementation sequence, deliverables, dependencies, and proof gates.
>
> **Status:** Active
> **Created:** 2026-09-11

---

# 1. Roadmap Position

The repository has two different roadmap concerns:

- root `roadmap.md` = capability discovery, evidence, and admission governance;
- `app/ROADMAP.md` = graduation/cutover history for the current production runtime;
- this document = next implementation roadmap for composable runtime architecture.

The composable work must not be mixed into the historical graduation phases. `app/` is already the live baseline; composability now evolves it incrementally behind conformance and benchmark gates.

---

# 2. Current Gap Register

The following are the important implementation gaps that must be explicitly tracked rather than left implicit in `007`:

| ID | Gap | Current state | Priority |
|---|---|---|---|
| CR-01 | Canonical Domain/Data/Experience model in code | conceptual only | P0 |
| CR-02 | Data IR | proposed | P0 |
| CR-03 | Dataset as reusable semantic contract | proposed / View-centric implementation | P0 |
| CR-04 | Projection model | proposed | P0 |
| CR-05 | Logical Query model independent from SQL/store APIs | partial concepts | P0 |
| CR-06 | Context / Scope / Binding | conceptual; partial embedded-view behavior | P0 |
| CR-07 | UI IR | proposed; renderer still View-oriented | P0 |
| CR-08 | Generic Component contract | proposed; specialized templates dominate | P1 |
| CR-09 | Static Component Registry seam | target; ordinary switches remain | P1 |
| CR-10 | View → generic primitive lowering | partial / not universal | P0 |
| CR-11 | Dependency DAG | proposed | P0 |
| CR-12 | CEP | proposed; not first-class code | P0 |
| CR-13 | CEP → Query Planner boundary | terminology/architecture needs enforcement | P0 |
| CR-14 | shared/batched execution | individual mechanisms exist; no common composable planner | P1 |
| CR-15 | bounded execution width / fan-out | individual limits exist; no composition-level budget | P1 |
| CR-16 | security-aware dependency identity | permission/RLS foundations exist; planner identity not implemented | P0 |
| CR-17 | plan identity / immutable plan reuse | partial metadata/interpreter caching foundations | P1 |
| CR-18 | inference diagnostics | principle exists; inspectability tooling missing | P1 |
| CR-19 | composability benchmark harness | benchmark design exists; executable planner benchmark not complete | P0 |
| CR-20 | trial applications using shared composable substrate | trial plan exists; runtime migration remains | P0 |
| CR-21 | metadata/schema representation for new composable artifacts | not yet universal | P0 |
| CR-22 | capability registry alignment | composable concepts span existing CAPs but are not yet one tracked implementation program | P1 |
| CR-23 | conformance model for composition validity | existing conformance is capability-oriented; composition proofs need expansion | P1 |
| CR-24 | failure isolation / partial rendering policy | architectural rule; no common composed-request implementation | P2 |
| CR-25 | renderer-neutral View Model / Render Input | proposed | P1 |
| CR-26 | migration compatibility for existing View handlers | planned | P0 |
| CR-27 | Grammar-area decision for `Dataset` (new Grammar area `D`, alongside `F/E/A/C/P/V/R/X/I/O`, vs. folding under `View`) | **undecided anywhere in writing** — `composable-runtime-blueprint.md` §5 names this its own Phase 0, highest priority, cost-free to resolve; `004`/`006` already use `Dataset`/`Relation`/`Projection`/`Dimension`/`Measure` as if this were settled, without the decision itself ever being recorded | P0 |
| CR-28 | `007-composable-runtime-architecture.md` §40 (claim-by-claim PROVEN/PROPOSED citation matrix) does not exist | **missing, pre-dates today's rewrite** — 007's own changelog (lines 13-16, 20-28) and `README.md`'s Tier 1 table both cite "§40" as the mechanism distinguishing implemented claims from architectural targets; the file's section numbering stops at "§34. Status of the Planner" (~line 1794) and ends ~line 1805 with no §40 present. Until this exists, no reader can verify which of 007's claims are safe to build against — blocks Phase 0 exit and makes Phase 1 (Canonical Semantic Model in code) premature | P0 |

---

# 3. Principles for Implementation

1. **Do not rewrite the proven runtime in one pass.** Introduce shared seams incrementally.
2. **View compatibility first.** Existing View metadata and handlers continue to work while lowering is introduced.
3. **Logical before physical.** Build semantic IR before optimizing SQL.
4. **Planner before proliferation.** Establish the dependency/planning boundary before adding many generic components.
5. **Security before coalescing.** Security scope is part of logical dependency identity.
6. **No one-component-one-query rule.** Logical component count must not become physical query count by architecture.
7. **No generic escape hatch.** Components remain bounded; arbitrary code/data access is not introduced as “generic composition.”
8. **Evidence before optimization.** Every coalescing/batching strategy is benchmarked against a naive baseline.
9. **Inference is inspectable.** Important inferred dependencies must be diagnosable.
10. **Trial cases are real consumers.** Document Approval and Project Management use the same substrate.

---

# 4. Phase 0 — Contract Freeze and Documentation Alignment

**Goal:** eliminate conceptual ambiguity before adding code.

### Deliverables

- [x] canonical vocabulary recorded in `composable-runtime-architecture-map.md`;
- [x] source-of-truth hierarchy recorded;
- [x] runtime compilation terminology clarified;
- [x] View / Component / Dataset distinction documented;
- [x] CEP vs Query Planner boundary documented;
- [x] current implementation gaps explicitly recorded;
- [x] inference-inspectability principle added to `001-design-principles.md`;
- [x] `002`, `003`, and `006` aligned with the composable model;
- [x] `004-runtime-metadata.md` aligned with the composable model (Domain/Data/Experience metadata, compilation boundary);
- [x] `005-runtime-lifecycle.md` aligned with compilation/normalization/dependency-analysis/planning terminology;
- [ ] add cross-links from any remaining composable-related benchmark/guide that still describes the old View-only model.

**Detailed backlog:** the deliverables above are the short checklist; the full item-by-item
backlog — including work not yet started, such as `runtime-metadata-schema.md` alignment,
capability-governance taxonomy, composition-level NFRs, and README/agent-guidance wording — is
tracked in `composable-runtime-roadmap-phase0-documentation-alignment.md` (DOC-01–DOC-10). That
addendum's own §4 Exit Criteria is the authoritative Phase 0 gate; the exit criteria line below is
a summary, not a substitute for it.

### Exit criteria

No Tier 1 document uses “interpreted” to imply “no internal compilation,” and no document presents View as the universal composition primitive. See `composable-runtime-roadmap-phase0-documentation-alignment.md` §4 for the complete, itemized gate — Phase 0 is not done until every item there is checked, not merely once `001`–`007` look consistent.

---

# 5. Phase 1 — Canonical Semantic Model

**Goal:** introduce the minimum internal Go model that can represent composition without forcing immediate database schema changes.

### Build

Introduce a package/boundary conceptually equivalent to:

```text
internal/composable/
  domain.go
  data.go
  experience.go
  context.go
  identity.go
```

The exact package name may change during implementation; the contract matters more than the path.

Initial types:

```text
NodeIdentity
DataSource
DatasetRef
Projection
LogicalQuery
Binding
Scope
ContextRef
ComponentRef
PageNode
LayoutNode
ComponentNode
ViewRef
```

Do not implement every proposed 007 object immediately. Start with the smallest model required by the two trial applications.

### Compatibility

Existing `View`, `ViewConfig`, Machine, Event, Permission, and Store structures remain intact. Adapters expose their semantic requirements into the new model.

### Proof

- unit tests for deterministic normalization;
- approval and project-management metadata can produce a canonical semantic representation;
- no change in existing 219+ capability conformance results.

### Exit criteria

The runtime can represent a composed page/data requirement without directly invoking a View handler as the only source of truth.

---

# 6. Phase 2 — Data IR, Dataset, and Projection

**Goal:** decouple semantic data requirements from ViewConfig and store-specific APIs.

### Build

Implement:

```text
DataSource
Dataset
RelationRef
Projection
Filter
Sort
Measure
LogicalQuery
DataIR
```

Start with the minimal vocabulary needed for:

- list/table;
- card collection;
- dashboard metric;
- board grouping;
- approval-step collection.

Existing field/operator filters remain syntax sugar and lower into the common expression/filter representation where practical.

### Important boundary

A Dataset is semantic. It must not contain:

- HTML;
- Templ syntax;
- CSS classes;
- SQL;
- PostgreSQL index names;
- cache implementation details.

### Proof

Create equivalent Data IR from:

1. existing List View;
2. existing Dashboard section;
3. Project Management collection/card;
4. Document Approval approval-step list.

### Exit criteria

At least two different experience consumers can use the same Dataset definition without copying renderer-specific configuration.

---

# 7. Phase 3 — Context, Scope, Binding

**Goal:** make nested composition semantically safe and predictable.

### Build

Implement:

```text
Context
Scope
ContextRef
Binding
```

with the initial domains:

```text
page
route
parameters
current_user
workspace
record
parent_record
selection
query_result
variables
```

### Rules

- child scopes cannot access undeclared parent values;
- record/collection context is explicit;
- bindings are validated before rendering;
- context does not become an arbitrary query API;
- security context is never downgraded by a child binding.

### Proof

- Project → List → Card → Checklist context;
- Approval Document → Approval Step → Approver context;
- invalid out-of-scope binding is rejected at load/compile time.

### Exit criteria

Nested trial UI can be expressed without business-specific handler branches for context propagation.

---

# 8. Phase 4 — UI IR and Generic Composition

**Goal:** make Experience composition independent from specialized View handlers.

### Build

Implement a minimal UI IR:

```text
UINode
 ├── identity
 ├── kind
 ├── properties
 ├── bindings
 ├── children
 ├── slots
 ├── actions
 └── conditions
```

Initial node kinds:

```text
page
layout
component
view_ref
static_content
```

### Lowering

```text
Page metadata
   ↓
UI IR
   ↓
render inputs
```

Existing View types are adapted as `view_ref`/preset nodes first. Do not delete existing handlers yet.

### Exit criteria

At least one complete page from each trial case is produced from UI IR while preserving existing rendered behavior.

---

# 9. Phase 5 — Component Contract and Static Registry Seam

**Goal:** stop specialized renderer dispatch from spreading into handlers.

### Build

Define a closed compile-time component contract and static registry seam:

```text
component type
   ↓
contract
   ↓
validator
   ↓
resolver
   ↓
renderer
```

Migrate only a small number of proven generic components first:

```text
Heading
Text
Collection
RecordSummaryCard
Metric
StatusBadge
ActionBar
```

### Do not do

- dynamic plugin loading;
- arbitrary component code in metadata;
- arbitrary SQL from components;
- client-side metadata interpreter;
- giant `GenericComponent` property bags.

### Exit criteria

New generic components can be added without adding business-specific branches to unrelated handlers.

---

# 10. Phase 6 — View Lowering and Compatibility Layer

**Goal:** make existing View types convenience presets over the composable substrate.

### Initial lowering targets

```text
List
 → Collection + Projection + table renderer

Card List
 → Collection + Projection + card renderer

Board
 → Collection + Group Dimension + board renderer

Dashboard
 → Page + Layout + Metric / Collection components

Calendar
 → Collection + Date Dimension + calendar renderer
```

### Compatibility rule

Existing metadata remains valid. Existing View handlers may remain behind the lowering layer until equivalence is proven.

### Exit criteria

At least three existing View types can lower to shared IR without changing conformance behavior.

---

# 11. Phase 7 — Dependency DAG

**Goal:** make all composed data/expression/security/render dependencies explicit before physical execution.

### Build

Derive:

```text
UI IR
Data IR
Bindings
Permissions
Expressions
Relations
Cache requirements
        ↓
Dependency DAG
```

Each dependency gets canonical identity including, where relevant:

```text
source
security scope
normalized filters
parameters
projection
grouping
measures
pagination
interpreter/metadata version
```

### Proof

The same Dataset consumed by multiple components produces one logical dependency node.

Security-incompatible consumers remain separate nodes.

### Exit criteria

A composed request can be inspected as a DAG before physical execution.

---

# 12. Phase 8 — Composable Execution Planner

**Goal:** make physical economy a first-class runtime boundary.

### Build in this order

1. execution groups;
2. dependency deduplication;
3. security-aware equivalence;
4. bounded concurrency;
5. batching;
6. shared execution;
7. cache opportunities;
8. async fallback;
9. cost estimation;
10. plan diagnostics.

### Planner contract

```text
Dependency DAG
      ↓
Execution Groups
      ↓
Cost / Capability Analysis
      ↓
Costed Execution Plan
      ↓
Physical operations
```

The planner does not replace the database query planner.

### Exit criteria

Planner-enabled execution passes all semantic/security conformance tests and demonstrates lower or equal physical work for the benchmark scenarios defined below.

---

# 13. Phase 9 — Query Planner / Physical Data Integration

**Goal:** connect logical execution units to efficient PostgreSQL execution without leaking physical details into metadata.

### Build

Integrate existing scale mechanisms:

- projection pushdown;
- filter pushdown;
- aggregate pushdown;
- index awareness;
- pagination;
- cache;
- workspace concurrency limits;
- separate analytics/report capacity;
- materialization when evidence justifies it.

### Boundary

```text
CEP
 ↓
logical data execution unit
 ↓
Query Planner
 ↓
SQL / cache / materialization
```

### Exit criteria

The runtime can demonstrate why a composed logical request produced a given bounded physical plan.

---

# 14. Phase 10 — Renderer / View Model Separation

**Goal:** ensure renderers consume semantic render inputs rather than raw metadata or raw database records.

### Build

```text
Data IR
 ↓
View Model / Render Input
 ↓
Renderer
```

Initial semantic view-model examples:

```text
RecordSummary
MetricValue
CollectionItem
StatusValue
ActionSet
```

The renderer must not know whether a value originated from a field, expression, or relation.

### Exit criteria

HTML/Templ remains the reference renderer, but the data semantics are not coupled to Templ.

---

# 15. Phase 11 — Composition Conformance

**Goal:** extend the existing capability-oriented proof system to composition semantics.

### New proof classes

```text
CMP-01 valid composition tree
CMP-02 cyclic composition rejected
CMP-03 unresolved reference rejected
CMP-04 slot/type mismatch rejected
CMP-05 scope violation rejected
CMP-06 dependency deduplication
CMP-07 security prevents unsafe coalescing
CMP-08 bounded fan-out
CMP-09 planner deterministic
CMP-10 View lowering equivalence
CMP-11 partial/failure isolation
CMP-12 trial-case cross-substrate reuse
```

The existing conformance suite remains authoritative for existing capabilities.

---

# 16. Phase 12 — Composability Performance Benchmark

**Goal:** prove that logical composability does not cause proportional physical work.

### Required scenarios

| Scenario | Expected question |
|---|---|
| 1 component → 1 Dataset | baseline overhead |
| 10 components → 10 independent Datasets | bounded concurrency / fan-out |
| 10 components → 3 shared Datasets | deduplication/coalescing |
| 1 Dataset → multiple projections | projection sharing vs narrower queries |
| mixed OLTP + analytics | pool isolation and class enforcement |
| 100 concurrent workspaces, cold/warm metadata | cache and workspace isolation |
| same logical request, different security scopes | security-aware identity |
| nested Project board | composition depth / context cost |
| Approval detail + stepper | embedded dependency sharing |

### Metrics

```text
p50 / p95 / p99 latency
logical nodes / request
DAG nodes / request
physical queries / request
physical operations / request
rows scanned
rows returned
CPU
memory
DB pool utilization
planner time
render time
cache hit ratio
execution width
```

### Acceptance rule

No planner optimization is accepted merely because query count falls. It must preserve semantics, security, and relevant latency/resource SLOs.

---

# 17. Phase 13 — Trial Migration

**Goal:** prove the architecture with real user workflows.

### Document Approval

Progressively lower:

```text
Approval Page
Approval Document
Approval Steps
Approver binding
Decision actions
Activity
```

into the shared composable substrate.

### Project Management

Progressively lower:

```text
Project Page
Board
List
Card
Checklist
Members
Activity
Ordering
```

into the same substrate.

### Success criterion

The two applications share the same:

- Page/Layout primitives;
- Dataset/Projection machinery;
- Component contract;
- Context/Binding system;
- permission-aware dependency planning;
- CEP;
- renderer boundary.

Case-specific business behavior may remain in Domain/Action/Event logic. It must not leak into generic composition infrastructure.

---

# 18. Phase 14 — Production Hardening

Only after real trial workloads:

- tune cost model thresholds;
- add persistent/longer-lived plan caching if evidence justifies it;
- tune batching vs parallelism;
- add materialization where warranted;
- add streaming/lazy loading where needed;
- expose inference and execution diagnostics;
- add operational dashboards for planner health;
- document fallback/degradation policy;
- ratchet performance benchmarks in CI where runtime cost is stable enough.

No optimization becomes a new semantic metadata concept merely because it is useful operationally.

---

# 19. Dependencies and Ordering

The critical dependency chain is:

```text
Documentation contract
        ↓
Canonical semantic model
        ↓
Data IR + Projection
        ↓
Context / Binding
        ↓
UI IR
        ↓
Component contract
        ↓
View lowering
        ↓
Dependency DAG
        ↓
CEP
        ↓
Query / physical integration
        ↓
Performance proof
        ↓
Trial migration
```

Do not implement CEP before Data IR/Dataset/Projection and dependency identity exist. A planner without a stable logical dependency model will optimize handler-specific behavior and recreate the architecture problem this roadmap is intended to solve.

---

# 20. Definition of Done for the Program

The composable runtime program is complete enough for production when:

- [ ] Domain/Data/Experience model exists in runtime code;
- [ ] Dataset and Projection are real semantic contracts;
- [ ] Context/Scope/Binding are validated;
- [ ] UI IR is the canonical input to the reference renderer for composed experiences;
- [ ] existing Views lower through the same substrate where applicable;
- [ ] generic Components have bounded contracts and a static dispatch seam;
- [ ] dependency DAG is inspectable;
- [ ] CEP is a first-class runtime boundary;
- [ ] physical execution is bounded and security-aware;
- [ ] Query Planner remains separate from CEP;
- [ ] inference affecting execution is inspectable;
- [ ] composition conformance is executable;
- [ ] performance benchmark demonstrates physical economy;
- [ ] Document Approval and Project Management run on the shared substrate;
- [ ] no existing capability conformance regression is introduced.

---

# 21. Relationship to Existing Plans

This roadmap intentionally reuses rather than replaces existing work:

- `007-composable-runtime-architecture.md` — normative target architecture;
- `composable-runtime-blueprint.md` — transformation blueprint/current-state inventory;
- `composable-runtime-architecture-map.md` — terminology and cross-document contract;
- root `roadmap.md` — capability discovery/admission evidence;
- `app/ROADMAP.md` — completed production graduation history;
- `capability-registry.md` — capability-level status and proof;
- `nfr-standards.md` — security/performance/architecture quality gates;
- `benchmarks/004-scale-architecture-study.md` and related scale studies — existing physical scale evidence;
- `composable-apps-trial.md` — trial applications and shared-substrate validation.

If implementation evidence contradicts a target assumption, update the blueprint/benchmark first and revise the target only through an explicit architecture decision. Do not silently change the meaning of a Tier 1 concept in implementation code.
