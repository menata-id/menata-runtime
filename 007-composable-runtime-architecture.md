# 007. Composable Runtime Architecture

> This document defines the architectural direction for evolving Menata Runtime from a
> View-oriented metadata runtime into a **Composable Application Runtime**.
>
> It is an architectural specification, not a rewrite plan. Existing Runtime Model concepts,
> capability registry entries, conformance discipline, Process Overlay compilation, and the
> current server-rendered implementation remain valid. New mechanisms should extend and unify
> them rather than replace them.
>
> Status: Draft v0.3 — evidence chain and terminology reconciliation (2026-09-11): §14 renamed to
> Static Component Registry with an explicit non-dynamic-dispatch statement; §40 replaced with a
> claim-by-claim citation matrix distinguishing PROVEN (implemented, conformance-cited) from
> PROPOSED (an architectural target, not yet built); a Tier-1-semantics note added below;
> cross-linked to `composable-runtime-blueprint.md` | Previously v0.2 — bounded-component rule,
> composition-cycle rejection, composition cost budget, two admission-gate questions added |
> Previously v0.1 | Created: 2026-09-11 | Updated: 2026-09-11

> **What Tier 1 status means for this document.** Tier 1 in this repository means *normative
> architectural direction and constraint*, not a certification that every mechanism described here
> is built or proven — §41 (Open Research Questions) and §42 (Recommended Next Studies) already
> concede that much of this is still a hypothesis. Every substantive claim in §40 below is marked
> **PROVEN** (implemented and conformance-cited — a `CAP-` row, test ID, or file/line) or
> **PROPOSED** (an architectural target awaiting the study or case that would validate it). A
> PROPOSED claim carries no conformance backing and must still pass `capability-lifecycle.md` §2's
> A1–A5 admission gate before it becomes a capability — this document does not admit anything by
> itself.
>
> **Relationship to `composable-runtime-blueprint.md`.** This document (Tier 1) answers *what the
> architecture must be* — the model, contracts, invariants, and boundaries. `composable-runtime-
> blueprint.md` (Tier 3) answers *how the runtime gets there and proves it* — current-state gaps,
> phased sequencing, each phase's own forcing condition, and the benchmark program. Read this
> document for the target shape; read the blueprint for what's actually built today and what has to happen, in what order, before the rest is.

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

Composition is defined across a **tree of experience nodes plus a graph of semantic/data
references**. The distinction is intentional:

```text
Experience structure                 Semantic dependency
--------------------                 -------------------
Page                                 Component A ──→ Dataset X
  └── Layout                         Component B ──→ Dataset X
       ├── Component                 Component C ──→ Dataset Y
       └── View                      Dataset Y ──→ Relation Z
```

The UI composition tree determines ownership and rendering order. References create reusable
connections between independently-owned runtime artifacts. Cycles in the experience tree are
invalid; semantic reference cycles are valid only where the referenced capability explicitly
supports them and must not cause unbounded evaluation.

At minimum, the runtime should recognize these composition domains:

```text
Domain
 ├── Machine
 ├── Field
 ├── Event
 ├── Constraint
 └── Permission

Data
 ├── DataSource
 ├── Dataset
 ├── Relation
 ├── Dimension
 ├── Measure
 ├── Projection
 ├── Expression
 ├── Filter
 ├── Sort
 └── Query

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

### Dataset ownership and scope

Reusable Datasets must have an explicit ownership/scope model. The initial architecture recognizes
three useful scopes:

```text
Workspace Dataset
      ↓
Application Dataset
      ↓
Machine-local / View-local query
```

A reusable Dataset should be promoted to the narrowest scope that satisfies all consumers. A
View-local query does not need a persisted Dataset identity merely because it happens to contain a
query. Conversely, a Dataset that is shared across Views, APIs, reports, or applications should
have stable identity and explicit ownership.

Dataset definitions must not contain renderer-specific properties. Renderer choices belong to the
Experience Plane.

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

### Component boundedness

A Component MUST expose a bounded semantic contract. It MUST NOT become "generic" merely by
accepting arbitrary properties or silently performing arbitrary data access.

A component may declare:

- inputs;
- data requirements;
- child slots;
- bindings;
- actions/events;
- accessibility semantics;
- renderer implementation.

A component MUST NOT silently acquire additional business data that is not represented by its
contract.

Generic composition is achieved by combining bounded components, not by creating an unbounded
`GenericComponent` escape hatch.

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

**Normative rule:** a View MUST NOT be required as the universal composition primitive. A requirement
that can be expressed using Page, Layout, Component, Binding, Dataset, Projection, and Static Content
should not require a new ViewType solely because the current renderer dispatch is View-oriented.

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

Component contracts should be versionable. Breaking contract changes require explicit compatibility
handling rather than silently changing the meaning of existing metadata.

---

# 14. Static Component Registry (Component Registry Seam)

> **Not a dynamic mechanism.** The Component Registry is a compile-time or statically assembled
> registry seam. It is **not** a client-side interpreter, a dynamic plugin loader, or a runtime
> extension mechanism. `prototype/objectstack/docs/composable-view-proposal-reconciliation.md` §4
> already rejected that shape structurally — a dynamic runtime dispatcher exists to let a client
> receive and safely interpret *arbitrary* metadata at runtime, the exact situation ObjectStack's
> React console is in and this server-rendered runtime is not (`app/ARCHITECTURE.md`'s client-side
> JS policy). §9.1/§10 of the same document reaffirmed that verdict twice more. What follows names
> a *single identifiable seam* for a fixed, closed set of component types — the same shape
> `capability-lifecycle.md` §4 already calls a "compile-time registry seam" for field/action/view
> types — not a second, competing extension mechanism.

A static component registry is the preferred long-term dispatch seam for new generic components,
replacing accumulating business-specific `switch` statements scattered across handlers with one
identifiable resolution point.

Conceptually:

```text
component type (closed, known at compile time)
      ↓
contract / schema
      ↓
validator
      ↓
resolver
      ↓
renderer
```

The first implementation may use a statically compiled registry (a Go map or compiler-checked
switch, resolved and validated at load/compile time); dynamic plugin loading is not a requirement
and is not the target — extension still means adding Go code and recompiling, per
`capability-lifecycle.md` §4, exactly as it already does for field/action/view types today.

The registry is responsible for discovering how a component type is implemented. It must not own
business authorization decisions, and it must not accept a component `type` string that wasn't
compiled into the runtime.

Existing `templ` functions may remain behind registered components during migration.

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

UI IR represents an **experience tree**. Shared datasets, projections, actions, and other semantic
artifacts remain references and are not recursively copied into every node.

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

## 15.3 Composition validity

The compiler MUST reject:

- cyclic experience trees;
- unresolved required child references;
- slot/type mismatches;
- bindings outside the permitted scope;
- recursion that exceeds the configured composition depth limit.

The compiler SHOULD detect duplicate or unnecessary data dependencies before render planning.

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

Data IR is logical. It MUST NOT expose database-specific storage or index choices to metadata
authors.

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

Physical plans are runtime-internal artifacts. They MUST NOT become portable Runtime Metadata.

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

## 20.9 Composition Cost Budget

The runtime should measure or bound at least:

- composition depth;
- number of resolved nodes;
- number of distinct Dataset requests;
- number of physical queries;
- estimated render work;
- metadata/IR size.

A composed experience that exceeds configured budgets should fail clearly or degrade through an
explicit runtime policy. It must not silently produce unbounded work.

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
7. What existing abstraction should the new capability lower into?
8. Will the capability introduce a new business-specific UI primitive or query mini-language?

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

A proposal that bypasses composition review must be explicitly justified in the capability
registry.

---

# 26A. Capability Boundary

The composable runtime must define a clear boundary between **declarative composition** and
**general-purpose programming**. The purpose of this boundary is not to limit composition to a
small number of UI levels. Atomic components are valid and useful. The boundary is about
computational power and ownership of implementation.

## 26A.1 Atomic Components

Atomic Components are the terminal, reusable vocabulary of the Experience Plane. They represent a
small, closed semantic capability whose implementation belongs to the runtime.

Examples:

```text
Text
Image
Icon
Badge
Avatar
Money
Date
Button
Link
```

An Atomic Component may accept bounded inputs, bindings, accessibility attributes, visibility
conditions, and other explicitly declared properties. It does not contain arbitrary executable
logic supplied by Runtime Metadata.

Conceptually:

```text
Metadata
   ↓
select component capability
   ↓
bind declared inputs
   ↓
runtime-owned implementation
```

An Atomic Component is therefore **not** a mini program. It is a registered runtime capability.

## 26A.2 Composite Components

Composite Components combine bounded components into a reusable semantic or structural unit.

Examples:

```text
RecordSummaryCard
ActionBar
FormSection
ActivityFeed
Collection
```

A Composite Component may expose controlled Slots and child contracts, but its children remain
bounded capabilities. A composite must not become an unbounded container that can execute arbitrary
metadata-defined code.

The preferred rule is:

> **Compose capabilities; do not implement capabilities inside metadata.**

## 26A.3 Experience Components

Experience Components assemble reusable components into application-facing surfaces.

Examples:

```text
Page
Dashboard
Detail Experience
Approval Experience
```

Experience Components may contain Layouts, Components, Views, Static Content, Datasets, Bindings,
and Actions, but they remain declarative composition structures. Business behavior remains owned by
Events, Actions, Constraints, Permissions, and Process primitives.

An Experience Component should not become a second programming runtime.

## 26A.4 Bounded Expressions

Expressions are permitted where the system needs limited computation for semantic decisions or
value derivation.

Examples:

```text
status == "Approved"
amount * quantity
 due_date < today()
```

Expressions MAY provide:

- predicates;
- calculated values;
- conditional visibility;
- derived measures;
- data transformation;
- routing decisions within an explicitly bounded domain.

Expressions MUST NOT provide:

- arbitrary loops;
- recursion;
- arbitrary function invocation;
- I/O;
- database access outside declared runtime planning;
- filesystem access;
- network calls;
- process execution;
- dynamic code loading;
- mutation of unrelated runtime state.

The expression layer is therefore a **bounded declarative language**, not a general-purpose
programming language.

## 26A.5 What Metadata May Express

Runtime Metadata MAY:

```text
select registered capabilities
configure bounded inputs
bind data to declared component inputs
compose components and layouts
compose datasets and queries
filter / sort / group / aggregate through supported semantics
express bounded conditions and formulas
select actions and events that already exist
reference stable runtime identities
provide static content through approved content nodes
```

In particular, metadata may express **intent and composition**, while the runtime retains ownership
of implementation.

## 26A.6 What Metadata Must Never Express

Portable Runtime Metadata MUST NOT become an arbitrary implementation language.

It must not express:

```text
arbitrary source code
arbitrary HTML execution
arbitrary JavaScript execution
SQL strings as a general data-access escape hatch
filesystem/network/process operations
unbounded loops or recursion
runtime reflection over unspecified capabilities
hidden database queries
hidden cross-workspace access
arbitrary mutation outside declared Actions
```

A proposed feature that requires one of these mechanisms is not automatically impossible, but it is
**outside the Composable Runtime metadata boundary**. It must instead be implemented as a runtime
capability with an explicit contract, security model, performance budget, and conformance proof.

## 26A.7 Capability Admission Gate

The capability-admission process MUST distinguish three outcomes:

```text
1. Existing composition is sufficient
       ↓
   use composition

2. Existing composition is insufficient, but one bounded generic primitive is missing
       ↓
   add the smallest reusable primitive

3. The requirement has a genuinely distinct semantic / behavioral identity
       ↓
   admit a new runtime capability
```

Before admitting a new primitive, reviewers should answer:

1. Is it expressible using existing components, layouts, datasets, bindings, actions, and expressions?
2. If not, is there a smaller generic primitive that enables this and other use cases?
3. Does the proposal add computational freedom rather than semantic capability?
4. Does it introduce a new mini-language, arbitrary property bag, or hidden data access path?
5. Can its contract be finite, documented, validated, versioned, and benchmarked?
6. What existing runtime substrate will implement or lower it?
7. What security and permission boundaries apply?
8. What is its worst-case composition/execution cost?
9. What conformance test proves the behavior?
10. What evidence justifies its promotion from application composition to platform capability?

A proposal that cannot answer these questions should remain application-specific configuration or be
rejected rather than silently becoming a new generic capability.

## 26A.8 When a Composition Becomes a New Capability

Composition becomes a new runtime capability when at least one of the following is true:

- it has a distinct semantic contract used by multiple independent applications or domains;
- the behavior cannot be expressed clearly as a composition of existing capabilities;
- keeping it as raw composition would create repeated, error-prone or ambiguous declarations;
- runtime-owned optimization or security enforcement is materially different from ordinary
  composition;
- the capability provides a stable abstraction that prevents multiple ViewType/component-specific
  implementations from diverging.

The following are **not**, by themselves, sufficient reasons for a new capability:

```text
"the mockup looks different"
"the page needs another arrangement"
"a developer wants custom HTML"
"one case needs one special property"
"the existing composition syntax is inconvenient"
```

These should first trigger decomposition and composition review.

The architectural litmus test is:

> **Does the new abstraction contribute a reusable semantic capability, or merely encode one
> implementation's preferred arrangement?**

Only the former should normally become a runtime capability.

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

No application author should need to rewrite existing metadata merely because the runtime gained a
more composable internal model.

The lowering result must preserve existing observable behavior within an explicitly defined
compatibility budget.

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

## 31.10 Composition-cycle Safety

Experience-tree cycles, invalid slot recursion, and reference patterns capable of causing unbounded
resolution must be rejected or bounded before execution.

## 31.11 Component Contract Safety

A component must consume only declared inputs, bindings, and data requirements. Hidden business
queries are a conformance failure.

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

## 32.10 Component Contract Cost

Measure the resolution and render overhead of registry-based generic components versus the current
statically dispatched `templ` implementation. The registry must not add material latency merely to
provide composability.

## 32.11 Composition Budget Enforcement

Measure behavior at and beyond configured limits for:

- node count;
- nesting depth;
- Dataset count;
- physical query count;
- metadata size.

The benchmark must prove that over-budget compositions fail predictably rather than degrading into
unbounded work.

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

# 36. Architectural Contracts

The following rules are **normative architectural invariants**. They apply to new implementation
work even before the target abstraction is fully implemented. When current code does not yet satisfy
an invariant, the gap belongs in the roadmap rather than being treated as permission to introduce a
second incompatible pattern.

### AC-01 — Machine remains the business capability boundary

Machine remains the primary business-capability boundary. Composition must not turn Machine into a
mere storage table or make Page/View the owner of business semantics.

### AC-02 — Dataset is presentation-independent

A Dataset MUST NOT contain renderer-specific behavior. It may describe source, relations,
projection, filtering, grouping, measures, parameters, and other semantic data requirements.

### AC-03 — Component data access is explicit

A Component MUST NOT perform undeclared business data access. Required data must enter through a
Dataset, Projection, Binding, or another explicitly declared runtime contract.

### AC-04 — Layout is semantically neutral

Layout MUST NOT own business rules, business queries, authorization decisions, or domain-specific
state transitions.

### AC-05 — View is not the universal composition primitive

A View MUST NOT be required solely because the runtime currently dispatches rendering by ViewType.
New composition should prefer generic Page/Layout/Component/View/Binding primitives.

### AC-06 — UI composition is a tree

The experience hierarchy MUST be acyclic and tree-shaped. Reusable semantic artifacts are linked by
references rather than recursively duplicated into the tree.

### AC-07 — Semantic dependencies form explicit references

Datasets, projections, actions, components, and other reusable artifacts MAY be shared by multiple
nodes through stable references. Reference resolution MUST remain explicit and bounded.

### AC-08 — Metadata is logical, not physical

Portable Runtime Metadata MUST NOT encode PostgreSQL-specific indexes, join algorithms, cache
internals, file paths, generated SQL, renderer markup, or equivalent physical implementation
choices.

### AC-09 — Physical plans are runtime-internal

Logical plans and physical execution plans MUST remain separate. Physical plans MUST NOT become the
portable contract between authoring tools and the runtime.

### AC-10 — Component contracts are bounded

A generic Component MUST have a finite, documented contract. Arbitrary property bags, hidden data
loads, and code-execution escape hatches are not valid substitutes for composability.

### AC-11 — Slots are controlled extension points

A Slot MUST declare what class of children it accepts. Composition through Slots must not bypass
permission checks, data requirements, or component contracts.

### AC-12 — Bindings respect scope

Bindings MUST resolve only against explicitly permitted context/scope. A child must not implicitly
read unrelated parent state merely because it is reachable in the object graph.

### AC-13 — Security precedes data optimization

Authorization scope MUST be part of the logical Data Plan before aggregation, caching, materialization,
or result exposure.

### AC-14 — Expressions are side-effect free

Expressions MUST remain deterministic, bounded, side-effect free, and incapable of I/O or arbitrary
code execution.

### AC-15 — Generic first, specialized second

A new specialized ViewType, Component, or Query language MUST NOT be the first solution when existing
primitives or one new generic primitive can express the requirement cleanly.

### AC-16 — Convenience abstractions lower to common substrate

High-level presets such as List, Detail, Dashboard, Approval Page, or Process overlays SHOULD lower
to shared intermediate/runtime primitives instead of creating separate execution models.

### AC-17 — New capabilities require composition review

Every new capability MUST record whether it was expressible through composition, why a new primitive
was necessary, and what reusable substrate it uses.

### AC-18 — Generic capabilities require executable proof

A capability MUST have conformance tests and, where relevant, benchmark evidence before being treated
as a stable generic primitive.

### AC-19 — Composition complexity is bounded

The runtime MUST define enforceable limits or cost policies for composition depth, node count, data
requirements, and physical query generation. "Composable" MUST NOT mean "unbounded."

### AC-20 — Backward compatibility is a first-class constraint

Existing Runtime Metadata remains valid unless an explicit compatibility decision is recorded.
New generic infrastructure must be introduced through adapters/lowering where practical.

---

# 37. Architectural Review Gate

Before merging a change that adds or materially changes a composable runtime capability, reviewers
should answer the following questions:

```text
1. What semantic capability is being added?
2. Can existing primitives compose it?
3. If not, what is the smallest generic primitive that unlocks it?
4. Does the proposal create a new business-specific component or ViewType?
5. What are the input/data/slot/binding contracts?
6. Can the requirement be represented in the common Data IR or UI IR?
7. Where does authorization enter the data plan?
8. What does the capability lower into?
9. What is the bounded worst-case composition cost?
10. What conformance and benchmark evidence proves the design?
```

A "yes" to business-specific specialization is not automatically rejected, but it requires an
explicit architectural justification and must not silently establish a new pattern for future work.

---

# 38. Architectural Invariants

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
13. **Experience trees are acyclic.**
14. **Reusable data and behavior are referenced, not silently duplicated.**
15. **Physical plans remain runtime-internal.**
16. **Component contracts are bounded and explicit.**

---

# 39. Architectural Decision Summary

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

# 40. Relationship to Existing Menata Research

This document consolidates directions already discovered by the repository rather than replacing
those studies. Per the Tier-1-semantics note at the top of this document, every claim below is
marked **PROVEN** (implemented, conformance-cited) or **PROPOSED** (an architectural target, not
yet built) — Tier 1 status is not a claim that the proposed rows are already true.

| §007 claim | Evidence | Status |
|---|---|---|
| Composition over specialization; a new requirement should compose before it becomes a new `ViewType` | `capability-lifecycle.md` §2 A4 (non-composability test, Study 5's ADR-0012 Pattern A/B precedent) | **PROVEN** — the discipline already governs every admission |
| High-level metadata can compile into lower-level runtime primitives at load time | Process Overlay, Study 21, `CAP-W01`, `internal/metadata/compile.go`, conformance T136–T139 | **PROVEN** — the concrete precedent §15's UI IR pipeline generalizes from |
| A page can compose multiple Views plus a small closed static-content vocabulary and one layout shape | `CAP-V10 Tier 2`, implemented 2026-09-09, conformance T253–T258 (`conformance/tests/230_composed_page.sh`) | **PROVEN** — this is the only shipped instance of the Experience-plane composition tree §12 describes; recursive nesting (§12.4) is still explicitly out of scope |
| A compile-time expression layer (CEL-shaped) is usable in filters/computed Fields/constraints | `CAP-C13`, ✅ implemented, Study 37 R1 | **PROVEN** |
| A named, reusable semantic Dataset (§7.2) prevents metric drift across report/dashboard/chart consumers | `CAP-V22`, ❌ Proposed, Study 37 R2 (`prototype/objectstack/`, ObjectStack ADR-0021's own "revenue defined three times" lesson) | **PROPOSED** — no forcing case yet in `case-portfolio.md`; candidate proof cases already named (Cases 9/15, `roadmap.md` item 24 step 4) |
| Metadata-driven index management / query cost awareness (§8, §20) | `CAP-X10`, ❌, Study 8 (`benchmarks/004-scale-architecture-study.md`), deliberately deferred per "Infer Before Configure" | **PROPOSED** — no measured scale pressure yet at this prototype's data volumes |
| Generic, semantically-named presentation primitives (`RecordSummaryCard`, not `ApprovalListCard`) reduce ViewType/business-specific proliferation | Study 38 (`benchmarks/029-composed-view-component-inventory.md`) and Study 40 (`benchmarks/030-ui-subcomponent-decomposition-criteria.md`); 7 of 9 primitives already real `templ` code (`app/docs/ui-component-library.md`) | **PROVEN** at the presentation-primitive level; **PROPOSED** as a formal, registry-dispatched Component contract (§13–§14) — the generalization from "primitives exist" to "a Component Registry governs them" hasn't been built |
| A static/compile-time Component Registry seam is the right dispatch shape; a dynamic client-side/plugin registry is not | `capability-lifecycle.md` §4 (existing compile-time registry-seam pattern for field/action/view types) proves the static half; `composable-view-proposal-reconciliation.md` §4, reaffirmed §9.1/§10, proves the dynamic half is structurally rejected, not merely deferred | **PROVEN** (both halves — one is already-existing practice, the other is a settled negative verdict) |
| Parent→child context/scope propagation is undesigned but will be mandatory once composed pages need parent-scoped children | `composable-view-proposal-reconciliation.md` §5, §8(iii) | **PROPOSED** — named, prioritized low today (no forcing case), mandatory the moment `CAP-V10 Tier 2` recursion or an equivalent case arrives |
| A UI Intermediate Representation (§15) is a useful compile target | No prior study — new to this document and to `composable-view-proposal-reconciliation.md` §9.2(b) | **PROPOSED / research question** — §41 Q13–Q14, §42 Study 44; the concrete problem an IR solves (one representation, many renderers) has no forcing case while `app/ARCHITECTURE.md` commits to exactly one renderer |
| Composability-measuring benchmarks (Application Construction Ratio et al., §32) | `composable-view-proposal-reconciliation.md` §9.2(c), §10 | **PROPOSED** — a candidate future study, not run yet |

The architectural purpose of this document is to establish the common model connecting those
findings — proven and proposed alike — not to claim the proposed rows are already realized.

---

# 41. Open Research Questions

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
13. What is the minimal component registry contract that provides generic resolution without imposing
    a dynamic plugin system?
14. What is the minimal Data IR that can unify existing list/report/dashboard query paths without
    over-generalizing the data model?

These questions should be answered through benchmarked prototypes and conformance tests rather
than architecture-by-assertion.

---

# 42. Recommended Next Studies

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

## Study 47 — Architectural Contract Conformance

Build a machine-checkable or test-backed suite for AC-01 through AC-20, prioritizing:

- View-independent composition;
- Dataset/presentation separation;
- explicit component data requirements;
- bounded context/binding;
- permission-before-optimization;
- acyclic experience trees;
- backward-compatible lowering.

---

# 43. Final Position

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
