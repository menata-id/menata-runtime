# 006. Runtime Model

> Runtime Model defines the fundamental runtime concepts used by Menata Runtime.
>
> These concepts form the application model realized by the runtime.
>
> The Runtime Model is independent from serialization formats and implementation technologies.

---

# Purpose

Business Knowledge describes organizations.

Runtime Metadata describes application realization.

The Runtime Model defines the runtime concepts used to realize those applications.

Every Runtime Metadata document is composed from Runtime Model elements.

---

# Design Goals

The Runtime Model should be:

- deterministic,
- composable,
- reusable,
- extensible,
- implementation independent,
- machine friendly.

Every concept should have a single responsibility.

---

# Runtime Hierarchy

Conceptually, an application has an organizational hierarchy and a composable realization model. These should not be confused.

```text
Workspace
    │
    └── Application
            │
            ├── Domain capabilities
            │      └── Machine
            │           ├── Field
            │           ├── Event
            │           ├── Constraint
            │           └── Permission
            │
            ├── Data definitions
            │      ├── DataSource
            │      ├── Dataset
            │      ├── Relation
            │      ├── Projection
            │      ├── Expression
            │      └── Query
            │
            └── Experience
                   ├── Page
                   ├── Layout
                   ├── Component
                   ├── View
                   ├── Slot
                   └── Binding
```

This is a logical model, not a requirement that all elements become separate database tables immediately. Existing metadata structures may continue to realize multiple logical concepts together while the runtime incrementally introduces explicit IR and registry boundaries.

The canonical relationship between this model and the composable execution pipeline is defined in [007-composable-runtime-architecture.md](007-composable-runtime-architecture.md) and [composable-runtime-architecture-map.md](composable-runtime-architecture-map.md).

---

# Workspace

Workspace is the highest organizational boundary.

A workspace represents an independent organizational environment.

A workspace owns:

- applications,
- permissions,
- governance,
- deployment,
- shared resources,
- runtime configuration.

Workspace isolation should always be maintained.

---

# Application

Application represents an independently realizable business solution.

Applications organize domain capabilities and experience into a cohesive user experience.

An application belongs to exactly one workspace.

Applications may share runtime infrastructure.

Applications remain logically independent.

---

# Machine

Machine is the primary realization unit for a business capability.

A Machine is the runtime realization of an **Object** as defined in the Menata Language Specification (`specification/001-object.md`). The Object names the Business Concept; the Machine is how this runtime executes it — carrying the Object's Fields, Events, Constraints, Permissions, and Views as Runtime Metadata. See `specification/000-language-spec.md` §Object and Machine.

Examples include:

- Purchase Request
- Purchase Order
- Customer
- Employee
- Asset
- Attendance

Machines may collaborate with other machines through:

- references,
- events,
- permissions,
- constraints.

Machines should remain independently understandable.

---

# Data Concepts

## DataSource

DataSource identifies the logical origin of data, such as Machine records or another reusable data source.

## Dataset

Dataset is a reusable semantic data definition. It describes available data without specifying how a particular component renders it.

## Relation

Relation describes a reusable association between data sources and reuses existing Machine reference semantics rather than inventing a second relationship identity.

## Projection

Projection defines the semantic shape consumed by a renderer or downstream operation. It is more than a raw field list because it may establish semantic roles such as title, person, money, or status.

## Query

Query is the executable logical representation of a data request, assembled from data source, projection, filters, grouping, measures, sorting, pagination, and parameters.

## Expression

Expression is the shared bounded semantic primitive used where deterministic computation is needed. Expressions are validated and cannot perform arbitrary code execution or I/O.

These concepts form the Data Plane. Their normalized runtime representation is Data IR; physical database choices remain runtime-owned.

---

# Experience Concepts

## Page

A Page represents a user interaction surface and experience root.

Pages organize user experience but do not own business logic.

## Layout

Layout is a generic spatial composition primitive such as stack, row, columns, grid, split, tabs, panel, or section.

## Component

A Component is a reusable semantic presentation primitive with a bounded contract. A component may declare inputs, data requirements, child slots, bindings, actions/events, accessibility semantics, and renderer implementation.

A component must not silently perform unrelated data access or authorization.

## View

A View describes a precomposed or domain-oriented presentation/data contract. Existing View types remain supported for compatibility and authoring simplicity.

A View is **not** the universal composition primitive. Requirements expressible through Page, Layout, Component, Dataset, Projection, Binding, and Static Content should not require a new ViewType merely because current dispatch is View-oriented.

## Slot

A Slot is a named composition point into which compatible child nodes may be inserted.

## Binding

Binding connects a component input to a context value or semantic data value. Binding is resolved against an explicit runtime scope.

The Experience concepts form the Experience Plane. Their normalized runtime representation is UI IR.

---

# Context and Scope

Composable experiences require explicit context propagation.

Typical context domains include:

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

Child bindings may consume parent values only when explicitly permitted. Context is semantic input, not an implicit mechanism for arbitrary data access.

---

# Action

Actions describe user or system intent.

Examples include:

- Create
- Update
- Delete
- Approve
- Reject
- Submit
- Publish
- Cancel

Actions trigger runtime behavior. Implementation belongs to the runtime.

---

# Workflow

Workflow coordinates application behavior.

Workflow emerges from:

- events,
- actions,
- permissions,
- constraints.

Workflow should remain declarative.

Business processes should not require imperative programming.

Workflow appears as a responsibility rather than a mandatory stored runtime artifact. A higher-level declarative process form may compile into Events, Constraints, Permissions, and Actions rather than becoming a second execution engine — see [004-runtime-metadata.md](004-runtime-metadata.md) and the process overlay material.

---

# Service

Services expose runtime capabilities.

Examples include:

- background processing,
- notification,
- scheduling,
- messaging,
- integration,
- document generation,
- email,
- AI service,
- external communication.

Service execution belongs to the runtime.

---

# API

API exposes application capabilities.

Runtime Metadata defines exposed endpoints, operations, permissions, serialization, and behavior.

Runtime determines implementation.

---

# Navigation

Navigation describes movement between application experiences.

Navigation may include:

- menus,
- breadcrumbs,
- tabs,
- shortcuts,
- quick actions.

Navigation remains separate from business execution and data semantics.

---

# Theme

Theme defines visual identity.

Theme may describe:

- colors,
- typography,
- spacing,
- icons,
- branding,
- layout preferences.

Rendering remains runtime dependent.

---

# Shared Resources

Applications may reuse shared resources.

Examples include:

- datasets,
- views,
- templates,
- themes,
- services,
- integrations.

Shared resources reduce duplication and should be referenced by stable identity.

---

# References

References connect runtime elements.

References establish relationships.

References should:

- remain stable,
- avoid duplication,
- preserve identity.

References should never redefine ownership.

---

# Identity

Every persisted or addressable runtime element possesses a stable identity.

Identity should survive:

- renaming,
- presentation changes,
- layout changes,
- implementation changes.

Stable identity enables:

- evolution,
- migration,
- compatibility,
- versioning,
- dependency sharing.

---

# Composition

Applications are assembled through composition.

The Experience Plane uses a tree for ownership and rendering order. Data and behavior dependencies form separate graphs. The runtime may combine these graphs for execution planning but must not collapse them into a single business/UI abstraction.

Composition should always be preferred over duplication.

---

# Extensibility

New runtime capabilities may be introduced through bounded extension seams.

A static/compile-time registry may dispatch a closed set of known component, field, action, or view types. It is not a dynamic plugin loader and does not permit arbitrary runtime code execution.

Existing concepts should remain stable and extensions should not require redesigning unrelated Runtime Model responsibilities.

---

# Runtime Ownership

| Concept | Responsibility |
|----------|----------------|
| Workspace | Organizational boundary |
| Application | Business solution |
| Machine | Business capability |
| DataSource / Dataset | Semantic data source/contract |
| Projection | Semantic data shape |
| Query | Logical data request |
| Page | User experience root |
| Layout | Spatial composition |
| Component | Reusable presentation primitive |
| View | Precomposed/domain-oriented presentation contract |
| Binding | Context/data connection |
| Action | User/system intent |
| Workflow | Behavioral coordination |
| Service | Runtime capability |
| API | External access |
| Navigation | User movement |
| Theme | Visual identity |

Responsibilities should never overlap.

---

# Evolution

The Runtime Model is expected to evolve.

Evolution should:

- preserve compatibility,
- minimize disruption,
- encourage reuse,
- avoid duplication,
- maintain deterministic realization.

---

# Summary

The Runtime Model defines the building blocks realized by Menata Runtime.

Applications are composed from Domain, Data, Experience, and Behavioral elements.

The runtime compiles those elements into internal representations, plans physical work, executes it, and renders responses.

Business Knowledge remains independent.

Runtime Metadata describes application realization.
