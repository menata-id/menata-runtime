# Composable Runtime Architecture Map

> **Purpose:** canonical cross-document integration contract for the Composable Application Runtime work.
>
> This document does not introduce a competing architecture. `007-composable-runtime-architecture.md` remains the normative target architecture. This document makes the relationships between the numbered specification, the existing metadata model, the implementation in `app/`, the capability registry, the blueprint, and the implementation roadmap explicit.
>
> **Status:** Active companion / integration contract
> **Last reviewed:** 2026-09-11

---

# 1. Canonical Terminology

The following definitions are the vocabulary to use consistently across the repository.

| Term | Canonical meaning | Not to be confused with |
|---|---|---|
| **Business Knowledge** | Human-oriented knowledge about how an organization works | Runtime Metadata |
| **Runtime Metadata** | Declarative machine-oriented description of application realization | physical execution plan |
| **Runtime Language** | The conceptual language expressed by Runtime Metadata | Menata Language |
| **Runtime Model** | Logical concepts used by the runtime to realize metadata | database schema |
| **Machine** | Primary realization unit for a business capability | UI data model |
| **DataSource** | Logical origin of data | physical table/query |
| **Dataset** | Reusable semantic data definition independent of renderer | cached result |
| **Projection** | Semantic output shape required by a consumer | merely a column list |
| **Query** | Executable logical data request | SQL statement |
| **Expression** | Bounded, deterministic computation primitive | arbitrary code |
| **Page** | Experience root / user interaction surface | business process |
| **Layout** | Generic spatial composition primitive | Dashboard-specific layout |
| **Component** | Reusable presentation primitive with a bounded contract | generic arbitrary property bag |
| **View** | Existing precomposed/domain-oriented presentation contract and compatibility abstraction | universal composition primitive |
| **Binding** | Explicit connection between component inputs and context/data | implicit arbitrary data access |
| **UI IR** | Normalized experience tree used by the runtime | HTML/CSS |
| **Data IR** | Normalized logical data requirement | SQL execution plan |
| **Dependency DAG** | Runtime graph of data/expression/security/cache/render dependencies | UI tree |
| **Composable Execution Planner (CEP)** | Cross-plane planning boundary that turns semantic dependencies into bounded physical execution | database-only query optimizer |
| **Query Planner** | Physical planning of an individual logical data operation | CEP |
| **Render Plan** | Runtime plan for producing renderer inputs from UI IR and resolved data | UI metadata |
| **Physical Execution Plan** | Runtime-internal choice of SQL, cache, batch, materialization, etc. | Runtime Metadata |
| **Static Component Registry** | Compile-time/static dispatch seam for a closed set of known component types | dynamic plugin system |

---

# 2. Source-of-Truth Hierarchy

When documents overlap, resolve them in this order:

```text
001 Design Principles
        ↓
002 Conceptual Runtime Architecture
        ↓
003 Runtime Language
        ↓
004 Runtime Metadata
        ↓
005 Runtime Lifecycle
        ↓
006 Runtime Model
        ↓
007 Composable Runtime Architecture
        ↓
Composable Architecture Map (this document)
        ↓
Capability Registry / NFR / Benchmarks
        ↓
Composable Runtime Blueprint
        ↓
Composable Runtime Roadmap
        ↓
Implementation (`app/`)
```

The practical meaning is:

1. Numbered documents define stable concepts and architectural constraints.
2. `007` defines the target composable architecture and its normative rules.
3. This map resolves terminology and cross-document relationships; it must not contradict `007`.
4. Benchmarks and capability documents provide evidence and admission status; they do not silently redefine Tier 1 architecture.
5. The blueprint explains current gaps and transformation strategy.
6. The roadmap defines implementation order and proof gates.
7. `app/` is the current implementation and may lag the target architecture while each migration phase is being completed.

---

# 3. Canonical Realization Pipeline

The phrase “interpreted application” remains valid at the product/authoring level: metadata defines the application and no application source code is generated.

The internal runtime mechanism should be described as **runtime compilation and execution**:

```text
Business Knowledge
      ↓
Menata Language
      ↓
Runtime Metadata
      ↓
Parse / Validate
      ↓
Resolve / Normalize
      ↓
┌──────────────┬───────────────┬────────────────┐
│ Domain IR    │ Data IR       │ Experience IR  │
└──────────────┴───────────────┴────────────────┘
                     ↓
              Dependency DAG
                     ↓
       Composable Execution Planner
                     ↓
             Physical execution
              ┌──────┴───────┐
              ↓              ↓
        Data / View       Render Plan
        Model results          ↓
              └──────────→ Renderer
```

**No source-code generation is implied.** Runtime compilation means compiling declarative metadata into runtime-internal representations.

---

# 4. Canonical Composition Model

Composition has two different structures and they must not be collapsed:

### Experience tree

```text
Page
 └── Layout
      ├── Component
      ├── View
      └── Component
```

The tree controls experience ownership, hierarchy, slots, and rendering order.

### Semantic dependency graph

```text
Component A ──→ Dataset X ──→ Relation R
Component B ──→ Dataset X
Component C ──→ Dataset Y
```

The graph controls data, expression, security, cache, and execution dependencies.

The CEP joins these concerns for planning, but the runtime must retain the distinction.

---

# 5. View Compatibility and Lowering

Existing Views remain supported. They are not deprecated by the composable architecture.

The intended direction is:

```text
Existing View
     ↓
implicit semantic requirements
     ↓
Dataset / Projection / Binding
     ↓
Experience primitives
     ↓
UI IR
```

Examples:

```text
List View
 = Collection + Projection + Table renderer

Card List
 = Collection + Projection + Card renderer

Board
 = Collection + Group Dimension + Board renderer

Dashboard
 = Page + Layout + Metric / Collection / Chart components
```

The implementation may keep specialized View handlers during migration. A new ViewType should not be added merely because an existing handler uses a type switch.

---

# 6. Component Contract

Every generic component introduced by the composable architecture should declare, conceptually:

```text
Component
 ├── identity
 ├── type
 ├── inputs
 ├── data requirements
 ├── bindings
 ├── child slots
 ├── actions/events
 ├── accessibility semantics
 └── renderer
```

Rules:

- A component has a bounded semantic contract.
- A component does not silently query arbitrary business data.
- A component does not own authorization semantics.
- Data requirements are resolvable before physical execution.
- Component contracts are versionable.
- Breaking changes require explicit compatibility handling.

The static component registry is only the dispatch seam for this closed set of compiled component types. It is not a runtime plugin mechanism.

---

# 7. Data Contract

A reusable Dataset is the preferred shared data contract when multiple consumers need the same semantic data.

```text
DataSource
    ↓
Dataset
    ↓
Filter / Relation / Group / Measure
    ↓
Projection
    ↓
Logical Query
    ↓
Data IR
```

A View-local query may remain local. It does not become a persisted Dataset merely because it contains a query.

A Dataset must not contain renderer-specific properties.

---

# 8. Context / Scope / Binding Contract

Context is explicit runtime input to composition.

Initial context domains:

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

Conceptual scope:

```text
Page
 ↓
Section
 ↓
Collection
 ↓
Record
 ↓
Field
```

Bindings must identify where their values come from and must respect scope boundaries. Context is not a backdoor for arbitrary queries.

This contract is a prerequisite for nested composition such as:

```text
Project
 └── List
      └── Card
           └── Checklist
```

and record-level compositions such as:

```text
Approval Document
 └── Approval Step
      └── Approver
```

---

# 9. CEP vs Query Planner

These terms must remain distinct.

### Composable Execution Planner

Answers:

- What logical work exists for the whole composed experience?
- Which dependencies are shared?
- Which operations can be batched?
- What execution width is safe?
- Which work should be deferred or made asynchronous?
- What security scopes prevent coalescing?

### Query Planner

Answers:

- How should one logical data operation be executed efficiently by the database/runtime?
- SQL shape, indexes, scans, joins, aggregates, and other physical data strategies.

Canonical relationship:

```text
UI/Data composition
        ↓
Data IR + dependency DAG
        ↓
Composable Execution Planner
        ↓
logical execution units
        ↓
Query Planner / cache / materialization
        ↓
physical operations
```

The CEP is therefore not a second SQL optimizer.

---

# 10. Security Ordering

Security is part of logical planning and precedes optimization that could widen visibility.

```text
Declared data requirement
        ↓
Permission / RLS scope
        ↓
Data IR
        ↓
Dependency DAG
        ↓
CEP optimization
        ↓
Physical execution
```

No cache, deduplication, batch, or shared execution may cross incompatible security scopes.

---

# 11. Current Implementation Reality

The current `app/` implementation is intentionally behind the target architecture.

Known facts:

- `View` remains the principal existing presentation abstraction.
- `ViewConfig` still contains many specialized presentation concerns.
- Page composition and embedded Views exist only in bounded forms.
- Generic Dataset / Projection / Data IR are architectural targets, not a complete current implementation.
- UI IR is an architectural target, not yet the universal renderer input.
- CEP is proposed infrastructure, not a claim that the full planner already exists.
- Static registry seams are target architecture; existing implementation still contains ordinary Go dispatch in places.

These are migration gaps, not contradictions, provided new work follows the lowering direction in this document.

The current-state inventory and phase-by-phase transformation remain documented in `composable-runtime-blueprint.md`.

---

# 12. Documentation Responsibilities

| Document | Responsibility | Should not become |
|---|---|---|
| `001-design-principles.md` | enduring philosophy | implementation plan |
| `002-architecture.md` | conceptual layers and runtime boundary | detailed component schema |
| `003-runtime-language.md` | language semantics and declarative realization | SQL / Go implementation |
| `004-runtime-metadata.md` | metadata artifact/schema semantics | physical execution plan |
| `005-runtime-lifecycle.md` | metadata lifecycle and live evolution | UI renderer specification |
| `006-runtime-model.md` | logical runtime concepts | database schema |
| `007-composable-runtime-architecture.md` | normative composable target | status report of every implementation detail |
| `capability-registry.md` | capability status/proof/priority | architectural vocabulary source |
| `nfr-standards.md` | quality/security/performance requirements | implementation design |
| `composable-runtime-blueprint.md` | current-state gaps and transformation blueprint | normative architecture |
| `composable-runtime-roadmap.md` | implementation sequence and proof gates | new architecture |
| `benchmarks/*` | external/empirical evidence | source of truth for semantics |
| `app/` docs | actual implementation status | target architecture |

---

# 13. Documentation Consistency Rules

When adding a composable capability:

1. Define or reuse the semantic primitive first.
2. Record the capability in `capability-registry.md` if it is a capability admission.
3. State whether it is **PROVEN**, **PARTIAL**, **PROPOSED**, or **DEFERRED**.
4. Define its lowering target.
5. Define its data dependencies and context requirements.
6. Define its security boundary.
7. Define its physical execution implications.
8. Add a conformance proof or explicitly record why proof is deferred.
9. Update the blueprint if the current-state gap changes.
10. Update the roadmap only if implementation order changes.
11. Do not create a new ViewType solely to bypass an existing composition boundary.
12. Do not introduce a new mini-language when the shared Expression model can express the requirement safely.

---

# 14. Definition of Done for Composable Runtime Work

A composable capability is not complete merely because its metadata validates or its UI renders.

It is complete when all applicable layers have a coherent contract:

```text
Semantic definition
      ↓
Metadata representation
      ↓
Validation
      ↓
Normalization / lowering
      ↓
IR / dependency representation
      ↓
Security scoping
      ↓
Execution planning
      ↓
Physical execution
      ↓
Rendering
      ↓
Conformance proof
      ↓
Performance proof
```

A capability may intentionally stop at a lower layer during research, but its status must say so explicitly.

---

# 15. Relationship to the Trial Applications

The Document Approval and Project Management trial applications are validation consumers of the same substrate, not separate architectures.

They should progressively demonstrate:

```text
Document Approval              Project Management
      │                              │
      ├── detail/page                ├── board/page
      ├── approval step              ├── list/card
      ├── actor binding              ├── nested composition
      └── process action             └── ordering/action
                 \                  /
                  \                /
                   shared runtime
                         │
                  Data + Experience
                         │
                        CEP
```

A primitive should be considered healthy when both cases can use it without case-specific forks in the composable substrate.

---

# 16. Final Architectural Rule

The long-term architecture is:

> **Business Knowledge defines intent; Runtime Metadata declares realization; Domain, Data, and Experience primitives express that realization; the runtime compiles and normalizes them; the Composable Execution Planner minimizes and bounds physical work; renderers realize the resulting experience.**

This is the single sentence that should remain consistent across the repository.
