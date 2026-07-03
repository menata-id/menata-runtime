# 005. Runtime Lifecycle

> Runtime Lifecycle describes how Menata Runtime continuously realizes Business Knowledge into running applications.
>
> Applications are not generated.
>
> Applications continuously evolve through Runtime Metadata interpretation.

---

# Purpose

Traditional application development follows a deployment lifecycle.

```text
Source Code
    │
    ▼
Compile
    │
    ▼
Build
    │
    ▼
Deploy
    │
    ▼
Run
```

Menata Runtime follows a metadata lifecycle.

```text
Business Knowledge
    │
    ▼
Runtime Metadata
    │
    ▼
Interpret
    │
    ▼
Running Application
```

Applications continuously evolve by interpreting Runtime Metadata.

---

# Lifecycle Overview

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
Validation
        │
        ▼
Interpretation
        │
        ▼
Application Realization
        │
        ▼
Running Application
        │
        ▼
User Interaction
        │
        ▼
Business Changes
        │
        └──────────────────────────────┐
                                       ▼
                             Runtime Metadata Updated
```

The lifecycle is continuous.

Applications evolve without regeneration.

---

# Phase 1 — Business Knowledge

Organizations continuously evolve.

Policies change.

Workflows change.

Objects change.

Permissions change.

Business Knowledge becomes the source for application evolution.

---

# Phase 2 — Runtime Metadata

Business Knowledge is realized as Runtime Metadata.

Runtime Metadata describes application intent.

Runtime Metadata becomes the executable description consumed by the runtime.

---

# Phase 3 — Validation

Before interpretation, Runtime Metadata should be validated.

Validation may include:

- structural validation,
- reference validation,
- constraint validation,
- permission validation,
- dependency validation,
- compatibility validation.

Invalid Runtime Metadata should never be interpreted.

---

# Phase 4 — Interpretation

Menata Runtime interprets Runtime Metadata.

Interpretation transforms metadata into executable runtime behavior.

Applications are interpreted.

Applications are never generated.

---

# Phase 5 — Application Realization

The runtime realizes:

- pages,
- navigation,
- views,
- forms,
- routing,
- APIs,
- services,
- workflows,
- permissions,
- constraints,
- platform services.

Everything is realized dynamically.

---

# Phase 6 — Running Applications

Applications become available to users.

Users interact with applications.

Business data changes.

Application behavior remains controlled by Runtime Metadata.

---

# Phase 7 — Continuous Evolution

Business evolves continuously.

Runtime Metadata evolves continuously.

Applications evolve continuously.

Application source code does not require modification.

---

# Runtime Responsibilities

Throughout the lifecycle the runtime is responsible for:

- metadata loading,
- metadata validation,
- metadata interpretation,
- application realization,
- routing,
- rendering,
- event execution,
- constraint enforcement,
- permission enforcement,
- platform services.

The runtime does not own Business Knowledge.

The runtime realizes Business Knowledge.

---

# Metadata Changes

Runtime Metadata may evolve by:

- adding applications,
- adding machines,
- adding pages,
- modifying views,
- modifying permissions,
- modifying constraints,
- modifying services,
- modifying routing,
- modifying navigation.

The runtime should realize changes without regenerating applications.

---

# Safe Evolution

Not every metadata change has the same impact.

Some changes are additive.

Some changes are behavioral.

Some changes affect existing business data.

The runtime should recognize these differences.

Potentially destructive changes should require explicit migration decisions.

Business data should always be preserved whenever reasonably possible.

---

# Versioning

Runtime Metadata should support versioning.

Versioning enables:

- compatibility,
- rollback,
- auditing,
- migration,
- safe evolution.

Business Knowledge should remain stable while Runtime Metadata evolves.

---

# Hot Reload

The runtime should support continuous interpretation.

Whenever reasonably possible:

- Runtime Metadata changes,
- runtime validates changes,
- runtime realizes changes,
- applications immediately reflect new behavior.

Application downtime should be minimized.

---

# Failure Handling

If Runtime Metadata cannot be interpreted:

- existing applications should remain stable,
- runtime should reject invalid metadata,
- validation errors should be reported clearly,
- business data should never be corrupted.

Safety is preferred over partial execution.

---

# Runtime Independence

The lifecycle should remain independent from:

- programming languages,
- databases,
- rendering engines,
- infrastructure,
- deployment models.

The lifecycle describes runtime behavior.

Implementation technologies remain replaceable.

---

# Lifecycle Principles

The Runtime Lifecycle follows several principles.

- Applications are interpreted.
- Applications are not generated.
- Metadata drives application evolution.
- Business Knowledge remains stable.
- Runtime Metadata evolves.
- Runtime realizes changes.
- Business data is preserved.

---

# Summary

Business Reality evolves.

Business Knowledge evolves.

Runtime Metadata evolves.

Menata Runtime continuously interprets Runtime Metadata.

Applications continuously evolve.

Business Knowledge remains the long-term organizational asset.
