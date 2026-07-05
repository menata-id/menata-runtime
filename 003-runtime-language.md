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

For what Runtime Metadata may describe, its hierarchy, and its lifecycle properties (versioning, stable identity, serialization), see [004-runtime-metadata.md](004-runtime-metadata.md).

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

# Governing Principles

Runtime Language is not a separate principle system. It is governed by the same architectural principles as the rest of Menata Runtime — Machine First, Runtime First, Metadata First, Declarative, Composable, Infer Before Configure, Workspace Isolation, Open Platform, Compatible Authoring, and the others.

See [001-design-principles.md](001-design-principles.md) for the authoritative statement of each. This document does not restate them.

---

# Summary

Menata Language explains Business Knowledge.

Runtime Language explains Application Realization.

Menata Runtime interprets Runtime Language.

Applications become living representations of Business Knowledge.

Runtime Language is designed for machines.

Business Knowledge remains designed for humans.
