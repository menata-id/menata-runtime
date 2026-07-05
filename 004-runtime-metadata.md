# 004. Runtime Metadata

> Runtime Metadata is the executable description interpreted by Menata Runtime.
>
> It defines how applications are realized.
>
> Runtime Metadata is designed primarily for deterministic machine interpretation.

---

# Purpose

Business Knowledge explains organizations.

Runtime Metadata explains applications.

Business Knowledge answers:

> What does the business know?

Runtime Metadata answers:

> How should the runtime realize that knowledge?

Runtime Metadata is the bridge between Business Knowledge and executable applications.

---

# Position in the Architecture

```text
Business Reality
        │
        ▼
Business Knowledge
        │
        ▼
Menata Language
        │
        ▼
Authoring Layer
        │
        ▼
Runtime Metadata
        │
        ▼
Menata Runtime
        │
        ▼
Applications
```

Business Knowledge remains implementation independent.

Runtime Metadata contains implementation intent.

Menata Runtime interprets Runtime Metadata into running applications.

---

# Runtime Metadata is not Business Knowledge

Business Knowledge should never describe runtime implementation.

Runtime Metadata should never redefine Business Knowledge.

Business Knowledge remains the source of truth.

Runtime Metadata realizes that knowledge.

---

# Runtime Metadata Characteristics

Runtime Metadata should be:

- deterministic,
- declarative,
- composable,
- versionable,
- machine-readable,
- implementation independent,
- extensible.

Runtime Metadata should avoid ambiguity.

---

# Runtime Metadata Scope

Runtime Metadata may describe:

- applications,
- machines,
- pages,
- navigation,
- layouts,
- views,
- forms,
- actions,
- workflows,
- services,
- APIs,
- routing,
- permissions,
- constraints,
- themes,
- notifications,
- integrations,
- scheduling,
- platform configuration.

Business Knowledge should not contain these runtime concerns.

---

# Runtime Metadata Hierarchy

The exact hierarchy may evolve.

Conceptually, Runtime Metadata is expected to be organized as:

```text
Workspace
    └── Application
            └── Machine
                    ├── Page
                    ├── View
                    ├── Service
                    ├── Workflow
                    ├── Navigation
                    ├── API
                    └── Configuration
```

Each level owns its own responsibility.

---

# Workspace

Workspace is the highest organizational boundary.

Workspace owns:

- applications,
- permissions,
- governance,
- deployment configuration,
- shared resources.

Applications should not assume a global namespace.

---

# Application

Application is an independently realizable solution.

An application consists of one or more machines.

Applications remain isolated through Runtime Metadata.

---

# Machine

Machine is the primary realization unit.

A machine realizes one business capability.

Examples include:

- Customer Management
- Purchase Request
- Purchase Order
- Attendance
- Asset Registration

Machines may interact through references, events, permissions, and constraints.

---

# Page

Pages define user experiences.

Pages organize application interaction.

Pages do not own Business Knowledge.

Pages realize Business Knowledge.

---

# View

Views determine how information is presented.

Views describe presentation intent.

Rendering remains the responsibility of the runtime.

---

# Service

Services expose application capabilities.

Services may include:

- business services,
- background jobs,
- scheduled execution,
- messaging,
- integration,
- external communication.

Service implementation belongs to the runtime.

---

# Workflow

Workflows coordinate application behavior.

Workflow behavior should emerge from:

- events,
- constraints,
- permissions,
- actions.

Business processes should remain declarative.

---

# Navigation

Navigation describes how users move through applications.

Navigation should remain independent from rendering technology.

---

# Configuration

Configuration customizes runtime behavior.

Configuration should be minimized.

The runtime should infer behavior whenever safely possible.

---

# Stable Identity

Every metadata element should possess a stable identity.

Labels may change.

Presentation may change.

Identity should remain stable.

Stable identity enables:

- application evolution,
- metadata versioning,
- migration,
- compatibility.

---

# Metadata Evolution

Runtime Metadata is expected to evolve.

Applications evolve by changing Runtime Metadata.

The runtime continuously realizes those changes.

Application regeneration should not be required.

---

# Metadata Versioning

Runtime Metadata should support versioning.

Versioning enables:

- application evolution,
- compatibility,
- rollback,
- migration,
- auditing.

Versioning belongs to Runtime Metadata.

Not Business Knowledge.

---

# Metadata Independence

Runtime Metadata should remain independent from:

- programming languages,
- rendering engines,
- databases,
- infrastructure,
- deployment environments.

Runtime implementation may evolve.

Runtime Metadata should remain stable.

---

# Serialization

Runtime Metadata does not require a specific serialization format.

Possible representations include:

- YAML,
- TOML,
- JSON,
- XML,
- binary formats,
- future representations.

Serialization is an implementation detail.

Runtime Metadata remains conceptually unchanged.

---

# Future Evolution

Runtime Metadata is expected to expand over time.

New capabilities should extend existing concepts rather than introduce incompatible models.

Evolution should preserve compatibility whenever reasonably possible.

---

# Summary

Business Knowledge explains organizations.

Runtime Metadata explains applications.

Menata Runtime interprets Runtime Metadata.

Applications become executable realizations of Business Knowledge.

Business Knowledge remains the long-term organizational asset.

Runtime Metadata continuously evolves to realize that knowledge.
