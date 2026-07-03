# 003. Runtime Language

> Runtime Language defines how applications are described for Menata Runtime.
>
> Unlike Menata Language, which expresses Business Knowledge, Runtime Language expresses Application Realization.
>
> Runtime Language is designed primarily for deterministic machine interpretation.

---

# Purpose

Business Knowledge explains how organizations work.

Runtime Language explains how applications should behave.

Business Knowledge answers:

> What does the business know?

Runtime Language answers:

> How should the runtime realize that knowledge?

Business Knowledge and Runtime Language serve different purposes.

Neither replaces the other.

---

# Design Philosophy

Runtime Language is designed for machines.

Not for Business Analysts.

Human readability is desirable.

Machine correctness is mandatory.

The language should always prefer:

- determinism,
- consistency,
- explicitness,
- composability,
- predictability.

Runtime Language is not intended to become another programming language.

It is a declarative metadata language.

---

# Relationship with Menata Language

Menata Language and Runtime Language describe different concerns.

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
Runtime Metadata
        │
        ▼
Runtime Language
        │
        ▼
Menata Runtime
        │
        ▼
Applications
```

Menata Language describes business.

Runtime Language describes application realization.

---

# Runtime Metadata

Runtime Language is expressed through Runtime Metadata.

Runtime Metadata is the executable description interpreted by Menata Runtime.

Applications are realized from Runtime Metadata.

Applications are never generated.

---

# Runtime Responsibilities

Runtime Language may describe:

- applications,
- navigation,
- pages,
- layouts,
- components,
- views,
- forms,
- actions,
- workflows,
- services,
- APIs,
- events,
- permissions,
- constraints,
- routing,
- themes,
- integrations,
- background jobs.

Business Knowledge should not describe these implementation concerns.

---

# Declarative

Runtime Language describes intent.

It does not describe execution steps.

For example:

Instead of saying:

> Create HTML.

The language describes:

> Display customer information.

The runtime determines how that information is rendered.

---

# Machine First

Runtime Language should always prioritize deterministic execution.

Two identical Runtime Metadata definitions should always produce identical application behavior.

Runtime interpretation should never depend upon:

- authoring tools,
- editors,
- operating systems,
- programming languages.

---

# Runtime Independence

Runtime Language should remain independent from:

- HTML,
- HTMX,
- CSS,
- JavaScript,
- Go,
- PostgreSQL,
- REST,
- GraphQL,
- rendering engines.

These are runtime implementation concerns.

Not language concerns.

---

# Composable

Every runtime capability should be composable.

Applications should be assembled from reusable metadata.

For example:

- pages,
- views,
- forms,
- services,
- workflows,
- permissions,
- constraints,

should remain independently reusable.

---

# Infer Before Configure

Runtime Language should minimize unnecessary configuration.

Whenever behavior can be inferred safely, explicit metadata should not be required.

Configuration should describe intent.

Not implementation.

---

# Stable Identity

Every runtime element should possess a stable identity.

Names may change.

Labels may change.

Presentation may change.

Identity should remain stable.

Stable identity enables:

- application evolution,
- metadata versioning,
- runtime migration,
- compatibility.

---

# Workspace Awareness

Runtime Metadata always belongs to a workspace.

Workspace defines:

- ownership,
- governance,
- visibility,
- deployment,
- security boundaries.

Applications should never assume a global namespace.

---

# Open Authoring

Runtime Metadata should be implementation independent.

Menata Apps Builder is the reference authoring tool.

However, Runtime Metadata may also be produced by:

- compatible builders,
- visual designers,
- command-line tools,
- manual editors,
- automated generators,
- third-party platforms.

Menata Runtime only interprets Runtime Metadata.

The runtime never depends upon its authoring process.

---

# Evolution

Runtime Language is expected to evolve.

Backward compatibility should be preferred whenever reasonably possible.

Applications should evolve by changing Runtime Metadata.

Not by rewriting runtime implementation.

---

# Serialization

Runtime Language does not require a specific serialization format.

Runtime Metadata may be represented using:

- YAML,
- TOML,
- JSON,
- XML,
- binary formats,
- future representations.

Serialization is an implementation detail.

Runtime Language remains unchanged.

---

# Future Evolution

Future versions of Runtime Language may introduce additional capabilities.

However, every capability should remain consistent with the core principles:

- Machine First
- Runtime First
- Metadata First
- Declarative

No capability should compromise these principles.

---

# Summary

Menata Language explains Business Knowledge.

Runtime Language explains Application Realization.

Menata Runtime interprets Runtime Language.

Applications become living representations of Business Knowledge.

Runtime Language is designed for machines.

Business Knowledge remains designed for humans.
