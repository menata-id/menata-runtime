# 007. Composable Runtime Architecture

> This document defines the architectural direction for evolving Menata Runtime from a
> View-oriented metadata runtime into a **Composable Application Runtime**.
>
> It is an architectural specification, not a rewrite plan. Existing Runtime Model concepts,
> capability registry entries, conformance discipline, Process Overlay compilation, and the
> current server-rendered implementation remain valid. New mechanisms should extend and unify
> them rather than replace them.
>
> Status: Draft v0.1 | Created: 2026-09-11

---

# 1. Purpose

Menata Runtime already provides a metadata-driven, declarative runtime in which applications are
realized from Runtime Metadata. The Runtime Model is composable at the artifact level: Machines,
Views, Events, Constraints, Permissions, Pages, Services, APIs, Navigation, and Shared Resources
have distinct responsibilities and stable identities.

The next architectural problem is deeper composition:

```text
Data
  → Query
  → Projection
  → Context
  → Composition
  → Component / Layout / View
  → UI IR
  → Renderer
```

The objective is to make composition a **universal runtime abstraction** across three dimensions:

1. **Data composition** — reusable semantic datasets, queries, projections, relations, measures,
   dimensions, and expressions.
2. **UI composition** — generic layouts, components, views, static content, bindings, slots, and
   nested experiences.
3. **Behavior composition** — reusable events, actions, constraints, permissions, and process
   primitives.

The runtime then compiles these declarations into efficient execution and rendering plans.

The architectural goal is:

> **High composability without sacrificing deterministic execution, security, or performance.**

---

# 2. Scope

This document governs the architecture of:

- Runtime Model evolution related to composition;
- semantic data and query abstractions;
- data projection and View Model concepts;
- generic UI composition;
- component and layout resolution;
- context and binding;
- intermediate representations (IR);
- query and render planning;
- caching and physical execution choices;
- renderer independence;
- capability-admission rules for new UI/data constructs.

This document does not redefine:

- Business Knowledge;
- the Menata Language as a business-language specification;
- workspace isolation and governance rules;
- existing event/constraint/permission semantics;
- deployment topology;
- the existing capability registry and conformance ratchet.

Those mechanisms remain authoritative where they already define behavior.

---

# 3. Architectural Position

Menata Runtime should evolve from:

```text
Machine
  └── View
        └── Renderer
```

toward:

```text
                         Runtime Metadata
                                │
                                ▼
                           Normalizer
                                │
             ┌──────────────────┼──────────────────┐
             ▼                  ▼                  ▼
         Domain IR           Data IR             UI IR
             │                  │                  │
      Machine / Event      Dataset / Query      Page / Layout
      Constraint           Projection           Component
      Permission           Expression           Binding
             │                  │                  │
             └──────────────────┼──────────────────┘
                                ▼
                         Context / Scope
                                │
                ┌───────────────┴───────────────┐
                ▼                               ▼
          Execution Plan                    Render Plan
                │                               │
                ▼                               ▼
            Database                         Renderer
```

The key architectural principle is:

> **Metadata expresses semantic intent. Runtime compilation determines physical realization.**

A declarative component must not need to know whether its data is served by a JSONB query,
an expression index, a generated column, a cache, or a materialized representation.

Likewise, a Dataset must not know whether it is rendered as a table, card, chart, API response,
CSV export, or another supported representation.

---

# 4. Design Principles

## 4.1 Composition over Specialization

A new application requirement should first be attempted as a composition of existing primitives.

A new specialized capability is justified only when composition cannot express the requirement
without introducing semantic ambiguity, unsafe behavior, excessive complexity, or unacceptable
runtime cost.

```text
Existing primitives
      ↓
Can they compose the requirement?
      │
   yes │ no
      │   │
      ▼   ▼
 compose  introduce a generic primitive
             │
             ▼
      specialized capability
      only when semantically unique
```

## 4.2 Semantic over Visual

Metadata should express semantic intent rather than implementation-level styling.

Prefer:

```text
metric
record
collection
section
stack
split
status
person
money
```

over:

```text
<div>
CSS classes
pixel offsets
framework-specific widget names
```

## 4.3 Data over View Coupling

Presentation must consume a semantic data contract rather than directly owning a private query
language whenever the data is reusable.

## 4.4 Projection over Retrieval

Components should request the smallest projection needed for rendering or execution.

The runtime should avoid loading entire records when only a small projection is required.

## 4.5 Runtime-Owned Physical Execution

Metadata defines logical intent. The runtime chooses:

- SQL strategy;
- index usage;
- cache strategy;
- materialization strategy;
- batching;
- parallelism;
- renderer implementation.

## 4.6 Determinism

Equivalent metadata must compile to semantically equivalent plans.

Plan construction must not depend on map iteration order, incidental database ordering, renderer
side effects, or non-deterministic capability discovery.

## 4.7 Stable Identity

Datasets, projections, components, layouts, bindings, and other addressable runtime artifacts
must have stable identity where they are persisted or referenced.

## 4.8 Reference over Duplication

Reusable semantic definitions should be referenced by identity instead of copied into multiple
Views or Components.

## 4.9 Security Before Optimization

Permission scoping is part of the logical data plan. It must be applied before aggregation,
projection, caching, and rendering. A cache must never widen the scope of a result.

## 4.10 Server Economy

The composable architecture must remain compatible with Menata's single-binary, modest-server,
server-rendered operating model. Flexibility must not imply an always-on client framework or a
large mandatory dependency graph.

---

# 5. Composition Model

Composition is defined as a graph of reusable runtime elements, not only a tree of Views.

At minimum, the runtime should recognize these composition domains:

```text
Domain
 ├── Machine
 ├── Field
 ├── Event
 ├── Constraint
 └── Permission

Data
 ├── Dataset
 ├── Source
 ├── Relation
 ├── Dimension
 ├── Measure
 ├── Projection
 ├── Expression
 ├── Filter
 └── Sort

Experience
 ├── Page
 ├── Layout
 ├── Section
 ├── Component
 ├── View
 ├── Slot
 ├── Binding
 └── Static Content

Behavior
 ├── Action
 ├── Event
 ├── Trigger
 ├── Constraint
 └── Process
```

These are logical concepts. The concrete serialization and storage representation may evolve.

---

# 6. Domain Plane

The existing Domain Plane remains the source of business capability semantics.

```text
Machine
 ├── Fields
 ├── Events
 ├── Constraints
 ├── Permissions
 └── Views
```

The composable architecture does not replace Machine. Machine remains the primary realization
unit for a business capability.

A Machine is a valid source for Data definitions, but a Data definition may intentionally expose
only a projection of that Machine.

A Machine is therefore not itself a UI data model.

---

# 7. Data Plane

The Data Plane is the architectural layer that decouples data semantics from presentation.

## 7.1 DataSource

A DataSource identifies the logical origin of data.

Initial source kinds may include:

- Machine records;
- another Dataset;
- a declared relation;
- a runtime service source where explicitly supported.

Example:

```yaml
source:
  machine: mch_purchase_request
```

## 7.2 Dataset

A Dataset is a named, reusable semantic data definition.

It describes what data is available without specifying how a particular component renders it.

Example:

```yaml
dataset:
  id: purchase_requests
  source:
    machine: mch_purchase_request

  dimensions:
    - id: status
      field: fld_status

  measures:
    - id: request_count
      aggregate: count

    - id: total_amount
      aggregate: sum
      field: fld_amount
```

A Dataset is the preferred shared data contract for reports, dashboards, tables, cards, charts,
exports, APIs, and other consumers that need the same semantic data.

## 7.3 Dimension

A Dimension identifies a grouping, labeling, or categorical axis of a Dataset.

Examples:

- status;
- region;
- customer;
- month;
- category.

Dimensions should retain semantic identity independent of renderer.

## 7.4 Measure

A Measure is a named calculation over the Dataset.

Initial aggregate vocabulary may include:

- count;
- sum;
- avg;
- min;
- max;
- count_distinct.

Derived measures may use expressions such as ratio, difference, or percentage where the expression
engine can prove bounded and deterministic evaluation.

## 7.5 Relation

A Relation describes a reusable association between data sources.

Relations should reuse existing Machine reference semantics rather than inventing a second
relationship identity.

A relation may be compiled to a join, semi-join, lookup, or other physical strategy.

## 7.6 Projection

A Projection describes the exact semantic shape consumed by a renderer or downstream runtime
operation.

Example:

```yaml
projection:
  - id: title
    source: fld_number
    semantic_type: title

  - id: requester
    source: fld_requester
    semantic_type: person

  - id: amount
    source: fld_amount
    semantic_type: money

  - id: status
    source: fld_status
    semantic_type: status
```

Projection is not just a field selection. It establishes a stable semantic contract between Data
execution and presentation.

## 7.7 Filter

Filters select records or aggregate groups according to a bounded expression language.

Existing field/operator filter forms remain valid as syntax sugar and may compile to the common
expression representation.

## 7.8 Sort

Sort describes logical ordering. Physical execution determines whether an index, database sort,
or another strategy is used.

## 7.9 Pagination

Pagination is part of the Data Plane because it determines how much data is retrieved, not how it
is visually presented.

The runtime should preserve the current capability that prevents unbounded list retrieval.

---

# 8. Query Model

The Query Model is the executable logical representation of a data request.

A logical query should be representable as:

```go
DataQuery {
    Source
    Projection
    Filter
    GroupBy
    Measures
    Sort
    Limit
    Offset
    Parameters
}
```

This is an architectural abstraction, not a requirement that the current Go structs use this exact
shape.

## 8.1 Query Composition

A Query may be composed from reusable Dataset definitions plus local predicates, projections,
and parameters.

Example:

```text
Dataset: purchase_requests
        +
Filter: status = Submitted
        +
Projection: number, requester, amount
        +
Sort: amount desc
        ↓
Logical Query
```

## 8.2 Query Normalization

Before execution, the runtime normalizes equivalent forms into a canonical logical representation.

Normalization should:

- resolve references;
- merge compatible predicates;
- remove redundant projections;
- preserve permission predicates;
- validate data types;
- determine required fields;
- canonicalize parameter names;
- produce stable plan identity.

## 8.3 Query Pushdown

The runtime should push computation to the database when it is:

- semantically equivalent;
- supported by the target database;
- cheaper than in-process evaluation;
- safe under the permission model.

Simple filters and aggregates should normally execute in SQL rather than after full retrieval.

## 8.4 Query Cost Awareness

The planner should distinguish at least:

```text
P1 interactive read
P2 interactive write
P3 heavy read
P4 asynchronous
P5 boot/reload
```

The existing NFR performance classes remain the governing budget model.

---

# 9. Expression Layer

Expressions are a shared semantic primitive used by:

- computed values;
- constraints;
- event conditions;
- view filters;
- Dataset filters;
- conditional visibility;
- action values;
- derived measures.

A single bounded expression model should replace proliferation of independent mini-languages.

## 9.1 Expression Safety

Expressions must be:

- deterministic;
- bounded;
- side-effect free;
- incapable of I/O;
- incapable of arbitrary code execution;
- statically validated before execution.

## 9.2 Allowed Context

The initial runtime context may expose only explicitly declared values such as:

```text
record
old
current_user
today
now
parameters
```

Access outside this context must fail closed.

## 9.3 Evaluation Strategy

The same expression may be evaluated:

```text
in process
```

or pushed down:

```text
into SQL
```

when equivalence can be proven.

The expression compiler should therefore expose both a logical AST and an execution capability
assessment.

---

# 10. View Model and Binding

A component should not consume raw records by default.

Instead:

```text
Query
 ↓
Projection
 ↓
View Model
 ↓
Component
```

The View Model is the runtime materialization of the semantic projection plus contextual values
needed by a component.

Example:

```text
RecordSummary
 ├── title
 ├── subtitle
 ├── status
 ├── avatar
 ├── href
 └── actions
```

The component does not need to know whether `status` came from a field, an expression, or a
joined relation.

---

# 11. Context and Scope

Composable experiences require a formal context propagation model.

## 11.1 Context

Context is the set of values available to a node during compilation and rendering.

Initial context domains may include:

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

## 11.2 Scope

Context is resolved through lexical runtime scopes.

A conceptual scope chain is:

```text
Page Scope
   ↓
Section Scope
   ↓
Collection Scope
   ↓
Record Scope
   ↓
Field Scope
```

A child can consume parent scope values only when that binding is explicitly permitted.

## 11.3 Binding

Binding connects a component input to a context or semantic data value.

Examples:

```yaml
binding:
  value: context.record.name
```

or:

```yaml
binding:
  dataset: customer_summary
  value: measure.revenue
```

Bindings must be statically resolvable where possible.

---

# 12. Experience Plane

The Experience Plane expresses how a user experience is assembled.

## 12.1 Page

A Page is an experience root. It should not own business logic.

## 12.2 Layout

Layout is a generic spatial composition primitive.

Initial generic layouts may include:

```text
stack
row
columns
grid
split
tabs
panel
section
```

Layout should accept arbitrary compatible children.

The runtime should not require separate layout concepts such as `DashboardLayout`,
`DetailLayout`, or `FormLayout` when their behavior can be expressed through generic primitives.

## 12.3 Component

A Component is a reusable semantic presentation primitive.

Examples:

```text
RecordSummaryCard
ActivityFeed
Metric
Collection
Record
Field
ActionBar
StatusBadge
Avatar
```

Components should be named for reusable semantic or structural identity rather than one business
case.

## 12.4 View

View remains a supported abstraction but changes in architectural position.

A View should be understood as a **precomposed or domain-oriented experience/data presentation
contract**, not the only way UI is composed.

Examples:

```text
list
form
detail
calendar
board
report
```

Existing View types remain valid. New capabilities should prefer composition before introducing a
new View type.

## 12.5 Slot

A Slot is a named composition point into which compatible child nodes may be inserted.

Slots enable a component to expose controlled extension points without exposing its internal
implementation.

Example:

```text
RecordPage
 ├── header
 ├── summary
 ├── body
 └── actions
```

## 12.6 Static Content

Static content should be a first-class composable node when a page needs to combine explanatory
text or visual material with structured data.

Examples:

- heading;
- paragraph;
- image;
- link;
- callout;
- divider.

Static content does not imply arbitrary HTML or a code hatch.

---

# 13. Component Contract

A component should declare a contract similar to:

```text
Component
 ├── identity
 ├── inputs
 ├── data requirements
 ├── child slots
 ├── actions/events
 ├── accessibility semantics
 └── renderer implementation
```

A component must not silently execute unrelated business queries.

Its data requirements must be resolvable from declared bindings and datasets.

---

# 14. Component Registry

A runtime component registry is the preferred long-term dispatch mechanism.

Conceptually:

```text
component type
      ↓
contract / schema
      ↓
validator
      ↓
resolver
      ↓
renderer
```

The registry is responsible for discovering how a component type is implemented. It must not own
business authorization decisions.

This should eventually replace proliferation of `switch` statements across handlers/templates for
new component types, while retaining compatibility with current implementations during migration.

---

# 15. UI Intermediate Representation

The UI IR is the normalized internal representation of a composed experience.

Conceptually:

```go
UINode {
    Kind
    Identity
    Properties
    Bindings
    Children
    Slots
    Actions
    Conditions
}
```

The actual Go representation may differ.

## 15.1 UI Compilation

The UI compilation pipeline should be:

```text
Runtime Metadata
      ↓
Parse
      ↓
Validate
      ↓
Resolve references
      ↓
Normalize
      ↓
Build UI IR
      ↓
Bind context
      ↓
Resolve Data requirements
      ↓
Build Render Plan
      ↓
Renderer
```

## 15.2 UI IR Responsibilities

UI IR should encode:

- composition hierarchy;
- semantic component types;
- layout relationships;
- stable identities;
- resolved bindings;
- conditional visibility;
- action references;
- data requirements.

UI IR should not embed HTML, CSS framework classes, SQL, or database-specific implementation.

---

# 16. Data IR

Data IR is the normalized semantic representation of a data requirement.

Conceptually:

```text
Data IR
 ├── Source
 ├── Projection
 ├── Filter
 ├── Relation
 ├── GroupBy
 ├── Measure
 ├── Sort
 ├── Parameters
 └── Security Scope
```

The Query Planner consumes Data IR and produces a physical execution plan.

---

# 17. Execution Plan

The runtime should distinguish logical intent from physical execution.

```text
Data IR
   ↓
Logical Query Plan
   ↓
Cost / Capability Analysis
   ↓
Physical Query Plan
```

A physical plan may select:

```text
live SQL
indexed SQL
cached result
batched query
materialized dataset
in-process evaluation
```

The selected strategy must remain transparent to metadata authors.

---

# 18. Query and UI Plan Relationship

UI composition and query execution are different concerns but should be compiled together enough
to eliminate unnecessary work.

Example:

```text
Page
 ├── Metric revenue
 ├── Orders table
 └── Activity timeline
```

could compile to:

```text
UI IR
 ├── Metric → Dataset A
 ├── Table  → Dataset B
 └── Feed   → Dataset C
```

The runtime may then detect reusable sources, compatible filters, or opportunities for batching,
without changing the UI metadata.

---

# 19. Security Integration

Security must enter the logical data plan before physical optimization.

Conceptually:

```text
Component request
      ↓
Dataset
      ↓
Permission scope
      ↓
Logical Data Plan
      ↓
Optimization
      ↓
SQL / Cache / Materialization
```

Never:

```text
query all data
      ↓
render
      ↓
trim unauthorized rows
```

Permissions remain owned by the existing authorization architecture. Data Plane composition must
consume permission decisions; it must not redefine authorization semantics.

Cached results must include security scope in their identity.

Example conceptual cache key:

```text
dataset
+ normalized filter
+ normalized parameters
+ permission scope
+ interpreter/version
```

---

# 20. Performance Architecture

Composable UI must not imply unbounded data retrieval or client-side processing.

## 20.1 Projection Pushdown

Only requested fields should be retrieved.

## 20.2 Filter Pushdown

Filters should execute in PostgreSQL whenever safe and beneficial.

## 20.3 Aggregate Pushdown

Measures should execute in SQL rather than materializing raw records whenever the aggregation is
supported.

## 20.4 Index Awareness

Metadata that declares frequently filtered, sorted, grouped, or joined fields should be available
to index management.

This aligns with the existing metadata-driven index direction.

## 20.5 Cache Awareness

Reusable Dataset results may be cached when:

- the result is expensive;
- freshness requirements permit caching;
- permission scope is part of cache identity;
- invalidation is deterministic enough.

## 20.6 Materialization

Large, expensive, stable semantic datasets may eventually use materialized representations.

Materialization is a runtime optimization, not a new semantic data model.

## 20.7 Batch and Deduplicate

The planner should recognize multiple components that request compatible datasets and avoid
redundant queries.

## 20.8 No Mandatory Browser Data Engine

Server-side data filtering, grouping, and authorization remain the default. Browser virtualization
may be added later for rendering efficiency, but it is not a substitute for server-side query
planning.

---

# 21. Physical Storage Strategy

Logical metadata must remain independent from physical storage.

A logical field may be physically realized as:

```text
JSONB value
expression index
stored column
generated column
materialized aggregate
```

The runtime may evolve physical strategy as workload changes without changing business metadata.

This is the preferred path for keeping a metadata-based logical schema while achieving relational
database performance.

---

# 22. Renderer Architecture

A renderer consumes UI IR and resolved View Models rather than raw Runtime Metadata whenever
possible.

Conceptually:

```text
UI IR + View Model
        ↓
Renderer
```

Possible renderers include:

```text
HTML / Templ
REST / JSON
CSV
XLSX
future client renderer
future mobile renderer
```

The existence of multiple renderers must not duplicate the data semantics.

The current server-rendered HTML/Templ renderer remains the reference implementation.

---

# 23. View Types and the Specialization Rule

Existing View types are not deprecated.

However, new design follows this rule:

> **Do not introduce a new ViewType when the requirement can be represented by existing Data,
> Layout, Component, Binding, and Renderer primitives.**

Examples:

```text
calendar
= collection
+ date dimension
+ calendar renderer
```

```text
board
= collection
+ group dimension
+ board renderer
```

```text
card list
= collection
+ card renderer
```

```text
dashboard
= page
+ generic layout
+ metric/chart/collection components
```

The implementation may continue exposing convenience View types for compatibility or authoring
simplicity. The runtime should progressively lower them to generic primitives where practical.

---

# 24. Convenience Abstractions

Menata should support **semantic presets**.

A preset is a higher-level declaration that compiles into generic primitives.

Examples:

```text
List View
Detail View
Dashboard View
Approval Page
```

A preset is successful when the runtime can lower it into a common substrate.

This is the same architectural pattern already proven by Process Overlay, where a higher-level
process declaration compiles into Fields, Events, Permissions, Constraints, and Actions.

---

# 25. Compilation and Lowering

Composable Runtime should use a consistent lowering model:

```text
High-level declarative metadata
           ↓
      normalize / validate
           ↓
      generic intermediate model
           ↓
         lower
           ↓
 existing runtime primitives
```

This permits convenience syntax without creating independent execution engines for every feature.

Examples:

```text
Process
  ↓
Events + Permissions + Constraints
```

```text
Dashboard
  ↓
Page + Grid + Components + Datasets
```

```text
Card List
  ↓
Collection + Card Renderer + Projection
```

---

# 26. Capability Admission Gate

Before adding a new capability, the architecture review must ask:

1. Can an existing primitive express it?
2. Can a combination of existing primitives express it?
3. Would one new generic primitive unlock this and other cases?
4. Is the requirement semantically unique enough to justify a new capability?
5. What is the performance cost of its generic form?
6. What conformance proof will demonstrate it?

A capability should normally be admitted at the lowest useful abstraction level.

```text
composition
   ↓
new generic primitive
   ↓
new specialized capability
```

not:

```text
business case
   ↓
new ViewType
```

---

# 27. Backward Compatibility

The architecture must permit incremental adoption.

Existing metadata such as:

```yaml
view:
  type: list
  config:
    fields: [...]
    filter: [...]
    default_sort: ...
```

remains valid.

The compiler may internally lower it to:

```text
Page / Visit Surface
       ↓
Collection Component
       ↓
Dataset / Query
       ↓
Projection
       ↓
Table Renderer
```

No application author should need to rewrite existing metadata merely because the runtime gained
a more composable internal model.

---

# 28. Current Menata Features and Target Mapping

| Existing concept | Composable target |
|---|---|
| Machine | Domain source / capability |
| Field | Data attribute / semantic field |
| Reference | Relation source |
| View | Convenience experience/data preset |
| ViewConfig.filter | Expression-backed Query filter |
| ViewConfig.default_sort | Query sort |
| Report | Dataset + aggregation + presentation |
| Dashboard | Page + Layout + Components + Datasets |
| Child View composition | Component/Page composition |
| Process Overlay | High-level behavior preset lowered to primitives |
| Computed Field | Expression-backed projection value |
| UI components in `templ` | Renderer implementations |
| Capability Registry | Admission + evidence mechanism |
| Conformance suite | Proof of behavioral compatibility |

---

# 29. Known Gaps

The following gaps are architectural targets rather than claims of current implementation.

## 29.1 Data

- first-class Dataset runtime model;
- reusable dimensions and measures;
- generalized Projection;
- generic Query AST/Data IR;
- relation-aware data plans;
- common expression AST;
- query cost model.

## 29.2 UI

- generic Layout runtime model;
- generic Component composition;
- Slot model;
- first-class Static Content nodes;
- unified binding/context model;
- component registry seam;
- UI IR;
- generic render plan.

## 29.3 Performance

- metadata-driven expression indexes as a reconciled system;
- Dataset-level cache policy;
- query deduplication/batching;
- adaptive materialization;
- explicit query-plan benchmarks.

## 29.4 Governance

- composition architecture gate in capability admission;
- conformance tests for generic composition;
- complexity limits on nested composition;
- diagnostics for expensive or ambiguous plans.

---

# 30. Non-Goals

This architecture does not require:

- a React-like client framework;
- a general-purpose browser-side query engine;
- arbitrary JavaScript execution from metadata;
- arbitrary HTML injection;
- a DAG engine for UI;
- table-per-Machine storage;
- rewriting all existing View types;
- replacing Machine with Dataset;
- replacing the capability registry.

Composable Runtime is a runtime abstraction strategy, not a frontend framework replacement.

---

# 31. Validation and Conformance

Every generic composition mechanism must be backed by executable proof.

At minimum, tests should prove:

## 31.1 Data Reuse

The same Dataset can feed multiple consumers without redefining its semantics.

## 31.2 Projection Reuse

A Projection can feed multiple renderers.

## 31.3 UI Reuse

A generic Component can be embedded by multiple Pages or Views.

## 31.4 Layout Reuse

A Layout can host multiple compatible child types.

## 31.5 Context Safety

Nested bindings resolve only within declared scope.

## 31.6 Permission Safety

Composed views cannot widen the permission scope of the underlying data.

## 31.7 Query Efficiency

Composed UI does not create N+1 queries for a shared Dataset or repeated reference lookup.

## 31.8 Backward Compatibility

Existing View metadata renders identically or within a defined compatibility budget after lowering.

## 31.9 Performance

Composition depth and Dataset reuse must be benchmarked against the established NFR budgets.

---

# 32. Recommended Benchmark Program

A dedicated benchmark family should measure composability rather than only feature correctness.

## 32.1 Application Construction Ratio

```text
ACR = business-specific runtime code / total application realization
```

Lower is better.

## 32.2 Composition Reuse Ratio

```text
CRR = reused generic components / total rendered components
```

Higher is better.

## 32.3 View Independence

Measure how many presentation changes can occur without changing the Dataset or logical Query.

## 32.4 Data Independence

Measure how many changes in Projection or Dataset shape can occur without changing renderer code.

## 32.5 Runtime Extension Cost

Measure the number of runtime files/packages/handlers that must change to introduce a new
composable experience.

## 32.6 Query Reuse Ratio

Measure how many UI consumers use the same Dataset or normalized logical query.

## 32.7 Physical Query Efficiency

Compare:

```text
components
vs
logical datasets
vs
physical SQL queries
```

The goal is to detect query duplication introduced by composition.

## 32.8 Composition Depth Cost

Measure latency and memory against nesting depth.

Example matrix:

```text
1 level
2 levels
4 levels
8 levels
```

## 32.9 Storage Strategy Benchmark

Compare at representative scale:

```text
JSONB scan
GIN
expression index
generated column
stored hot column
materialized dataset
```

with p50/p95/p99 latency, CPU, IO, memory, and write amplification.

---

# 33. Reference Test Matrix

A future benchmark suite should combine:

```text
Data Shapes
 ├── single Machine
 ├── reference join
 ├── aggregate
 ├── time series
 └── multi-source composition

Presentation Shapes
 ├── table
 ├── cards
 ├── detail
 ├── metric
 ├── chart
 ├── timeline
 └── mixed page

Composition Shapes
 ├── flat
 ├── nested
 ├── repeated dataset
 ├── shared dataset
 └── mixed static + structured
```

Each test records:

- metadata size;
- compiled IR size;
- query count;
- query planning cost;
- database execution cost;
- render cost;
- response size;
- memory;
- cache hit rate.

---

# 34. Implementation Strategy

The architecture should be introduced incrementally.

## Phase 1 — Normalize the Existing View System

Create an internal normalized representation without changing public metadata.

```text
existing View metadata
      ↓
normalizer
      ↓
internal View/Data model
```

Goal: prove that current Views can lower into a generic substrate.

## Phase 2 — Introduce Expression IR

Unify field/operator predicates, computed values, and conditional logic behind one bounded
expression model.

## Phase 3 — Introduce Dataset + Projection

Implement a reusable semantic Dataset and Projection layer for the existing list, report, and
dashboard paths.

## Phase 4 — Introduce DataPlan

Compile Dataset + filter + projection + sorting into a common query plan and integrate permission
scope before physical execution.

## Phase 5 — Introduce Generic Layout + Component Tree

Move composed Pages from View-only children to generic composition nodes while preserving existing
`children` behavior.

## Phase 6 — Introduce UI IR

Compile generic composition to a renderer-independent IR.

## Phase 7 — Component Registry

Move new component resolution behind a registry seam. Existing templ functions can remain the
implementation behind registered components during transition.

## Phase 8 — Query Optimization

Add metadata-driven indexes, query deduplication, cache, batching, and cost-aware physical plans.

## Phase 9 — Adaptive Execution

Only after benchmarks justify it, introduce materialized datasets and other heavier physical
strategies.

---

# 35. Migration Rule

At no point should the migration require a big-bang replacement.

The preferred sequence is:

```text
Current implementation
        ↓
Adapter
        ↓
Normalized model
        ↓
Generic IR
        ↓
New execution path
```

Old and new paths should be compared under conformance and benchmark workloads before replacing
default execution.

---

# 36. Architectural Invariants

The following invariants should remain true as the architecture evolves.

1. **Machine remains the business capability boundary.**
2. **Dataset does not own presentation.**
3. **Component does not own arbitrary data access.**
4. **View does not own authorization.**
5. **Layout does not own business logic.**
6. **Permissions are resolved before result exposure.**
7. **Expressions cannot perform I/O.**
8. **Physical storage remains implementation detail.**
9. **Renderer does not redefine semantic data.**
10. **New specialization requires a composition review.**
11. **Every admitted capability needs executable proof.**
12. **Performance budgets apply to composed experiences, not only isolated endpoints.**

---

# 37. Architectural Decision Summary

The preferred Menata direction is:

```text
                    COMPOSABLE APPLICATION RUNTIME
                               │
          ┌────────────────────┼────────────────────┐
          ▼                    ▼                    ▼
        DOMAIN                DATA                   UI
          │                    │                    │
      Machine              Dataset                Page
      Field                Query                  Layout
      Event                Projection             Component
      Constraint           Expression              View
      Permission           Relation                Binding
                             │                     Context
          │                 │                    │
          └─────────────────┼────────────────────┘
                            ▼
                      Runtime Compiler
                            │
                ┌───────────┴───────────┐
                ▼                       ▼
             Data IR                  UI IR
                │                       │
                ▼                       ▼
          Execution Plan             Render Plan
                │                       │
                ▼                       ▼
             Database               Renderer
```

The intended architectural outcome is not a larger collection of View types.

It is a **small, composable substrate** from which richer application experiences can be built.

---

# 38. Relationship to Existing Menata Research

This document consolidates directions already discovered by the repository rather than replacing
those studies.

Relevant existing work includes:

- the design principle of composability and reference over duplication;
- Runtime Model separation of Machine, Page, View, Action, Service, API, Navigation, and Theme;
- architecture benchmarking around internal models, declarative interpretation, reconciliation,
  and independent renderers;
- Study 8's metadata-driven index and lazy metadata loading direction;
- the existing View configuration for filters, sorting, pagination, and composition;
- composed View and Dashboard capabilities;
- Process Overlay as an example of high-level metadata compiling into lower-level primitives;
- semantic Dataset direction identified in the ObjectStack comparison;
- expression-layer research;
- UI component decomposition and presentation-primitive research.

The architectural purpose of this document is to establish the common model connecting those
findings.

---

# 39. Open Research Questions

The following questions remain intentionally open and should be answered by executable studies:

1. What is the minimum Dataset model that covers CRUD, dashboard, report, and analytics cases?
2. How should Dataset inheritance or composition work without creating hidden semantic coupling?
3. Which expressions can always be pushed down safely to PostgreSQL?
4. What cost model is sufficient for a modest single-server deployment?
5. At what workload should Dataset caching become beneficial?
6. When does materialization outperform live queries under Menata's expected write/read mix?
7. What is the minimum generic Layout vocabulary that covers enterprise application patterns?
8. How should slots and bindings interact with authorization and conditional visibility?
9. How should client-side interactivity evolve without turning the runtime into a mandatory SPA?
10. How much composition depth is practical before metadata and runtime complexity dominate?
11. What generic component contracts can replace multiple specialized ViewTypes without losing
    domain expressiveness?
12. Which capabilities should be compile-time lowered versus runtime-resolved?

These questions should be answered through benchmarked prototypes and conformance tests rather
than architecture-by-assertion.

---

# 40. Recommended Next Studies

## Study 41 — Semantic Data Runtime

Prototype:

```text
Dataset
Dimension
Measure
Projection
Expression
Relation
```

and implement it for one existing list, one dashboard, and one report.

## Study 42 — DataPlan / Query Compiler

Build a common Data IR and compare it with the current handler/store query paths.

Measure correctness and planning overhead.

## Study 43 — Generic Composition Runtime

Prototype:

```text
Page
Stack
Grid
Split
Section
Collection
Record
Metric
Static Content
```

Use existing application cases and measure how many specialized View types become unnecessary.

## Study 44 — UI IR

Build the smallest renderer-independent UI IR capable of representing the current HTML/Templ
surface.

## Study 45 — Composition Performance

Measure query count, latency, memory, and render cost as composition depth and Dataset reuse grow.

## Study 46 — Adaptive Physical Execution

Compare live SQL, cached Dataset results, indexed JSONB access, generated columns, and materialized
representations at the established Study 8 scale.

---

# 41. Final Position

Menata Runtime should not abandon its current architecture in favor of a new frontend-oriented
composition framework.

The preferred evolution is:

```text
Metadata-driven Runtime
        ↓
Composable Metadata
        ↓
Composable Data
        ↓
Composable UI
        ↓
Composable Behavior
        ↓
Common Intermediate Representations
        ↓
Cost-aware Runtime Execution
```

The central architectural shift is:

```text
Machine → View → specialized renderer
```

toward:

```text
Data
 → Query
 → Projection
 → Context
 → Composition
 → Component / Layout / View
 → UI IR
 → Renderer
```

with behavior remaining independently composable:

```text
Event
 + Action
 + Constraint
 + Permission
 + Process
```

and with physical execution remaining runtime-owned:

```text
logical intent
      ↓
compiler / optimizer
      ↓
physical realization
```

The objective is therefore not "more metadata" or "more ViewTypes".

The objective is:

> **A small, deterministic, reusable runtime substrate capable of expressing many applications
> through composition while allowing the runtime to continuously choose efficient physical
> execution.**

That is the architectural direction for Menata Runtime's transition from a metadata-driven
application runtime into a general-purpose **Composable Application Runtime**.
