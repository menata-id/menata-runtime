# 006. Runtime Model

> Runtime Model defines the fundamental runtime concepts used by Menata Runtime.
>
> These concepts form the application model interpreted by the runtime.
>
> The Runtime Model is independent from serialization formats and implementation technologies.

---

# Purpose

Business Knowledge describes organizations.

Runtime Metadata describes applications.

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

Conceptually, every application follows the same hierarchy.

```text
Workspace
    │
    └── Application
            │
            ├── Machine
            │      ├── Page
            │      ├── View
            │      ├── Action
            │      ├── Service
            │      ├── Workflow
            │      └── API
            │
            ├── Navigation
            ├── Theme
            ├── Configuration
            └── Shared Resources
```

This hierarchy may evolve without changing its fundamental responsibilities.

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

Applications organize machines into a cohesive user experience.

An application belongs to exactly one workspace.

Applications may share runtime infrastructure.

Applications remain logically independent.

---

# Machine

Machine is the primary realization unit.

A machine realizes a single business capability.

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

# Page

A page represents a user interaction surface.

Pages organize user experience.

Pages do not own business logic.

Pages present business capabilities.

Examples include:

- List Page
- Detail Page
- Form Page
- Dashboard
- Report
- Calendar

---

# View

Views describe information presentation.

Views determine:

- visible data,
- ordering,
- grouping,
- formatting,
- filtering,
- visualization.

Views never determine business logic.

Rendering remains the responsibility of the runtime.

---

# Action

Actions describe user intent.

Examples include:

- Create
- Update
- Delete
- Approve
- Reject
- Submit
- Publish
- Cancel

Actions trigger runtime behavior.

Implementation belongs to the runtime.

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

Runtime Metadata defines:

- exposed endpoints,
- operations,
- permissions,
- serialization,
- behavior.

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

Navigation remains presentation independent.

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

- menus,
- views,
- templates,
- themes,
- services,
- integrations.

Shared resources reduce duplication.

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

Every runtime element possesses a stable identity.

Identity should survive:

- renaming,
- presentation changes,
- layout changes,
- implementation changes.

Stable identity enables:

- evolution,
- migration,
- compatibility,
- versioning.

---

# Composition

Applications are assembled through composition.

Large applications are combinations of smaller runtime elements.

Composition should always be preferred over duplication.

---

# Extensibility

New runtime concepts may be introduced.

Existing concepts should remain stable.

Extensions should not require redesigning the Runtime Model.

---

# Runtime Ownership

Each runtime concept owns exactly one responsibility.

For example:

| Concept | Responsibility |
|----------|----------------|
| Workspace | Organizational boundary |
| Application | Business solution |
| Machine | Business capability |
| Page | User interaction |
| View | Information presentation |
| Action | User intent |
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
- maintain deterministic interpretation.

---

# Summary

The Runtime Model defines the building blocks interpreted by Menata Runtime.

Applications are composed from Runtime Model elements.

The runtime realizes those elements into executable applications.

Business Knowledge remains independent.

Runtime Metadata describes realization.

Menata Runtime performs interpretation.
