# 002. Runtime Architecture

> This document describes the conceptual architecture of Menata Runtime.
>
> It intentionally avoids implementation details.
>
> The architecture defines responsibilities rather than technologies.

---

# Purpose

Menata Runtime realizes Business Knowledge as living applications.

Business Knowledge itself is not executable.

Runtime Metadata bridges Business Knowledge and executable applications.

The runtime continuously interprets Runtime Metadata into running applications.

Applications are interpreted.

Applications are not generated.

---

# High-level Architecture

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
──────────────────────────────
Authoring Layer
──────────────────────────────

Menata Apps Builder

Visual Builder

CLI

Manual Editor

Compatible Tools

        │
        ▼

Runtime Metadata

──────────────────────────────
Runtime Layer
──────────────────────────────

Menata Runtime

        │
        ▼

Applications
```

---

# Layer Responsibilities

## Business Reality

Business Reality is where organizations operate.

It contains people, activities, assets, policies, workflows, and decisions.

Business Reality continuously changes.

---

## Business Knowledge

Business Knowledge explains Business Reality.

It captures:

- objects,
- rules,
- permissions,
- constraints,
- events,
- relationships,
- views.

Business Knowledge is implementation independent.

---

## Menata Language

Menata Language expresses Business Knowledge.

It is designed primarily for humans.

Its responsibility is to describe business intent.

It does not describe implementation.

---

## Authoring Layer

The Authoring Layer produces Runtime Metadata.

Runtime Metadata may be produced by:

- Menata Apps Builder,
- visual builders,
- command-line tools,
- manual editors,
- compatible third-party tools.

The runtime never depends on how Runtime Metadata is created.

---

## Runtime Metadata

Runtime Metadata describes how Business Knowledge should be realized.

Runtime Metadata is designed primarily for deterministic machine interpretation.

It contains application realization rather than Business Knowledge.

---

## Menata Runtime

Menata Runtime interprets Runtime Metadata.

Its responsibility is to realize executable applications.

The runtime owns:

- application lifecycle,
- routing,
- rendering,
- navigation,
- authentication,
- authorization,
- event execution,
- constraint enforcement,
- platform services.

---

## Applications

Applications are runtime realizations.

Applications exist because Runtime Metadata exists.

Applications continuously evolve as Runtime Metadata evolves.

Applications remain isolated from each other through metadata.

---

# Separation of Responsibilities

Each layer owns different responsibilities.

| Layer | Responsibility |
|--------|----------------|
| Business Reality | Organizational activities |
| Business Knowledge | Organizational knowledge |
| Menata Language | Business expression |
| Authoring Layer | Runtime Metadata authoring |
| Runtime Metadata | Application realization description |
| Menata Runtime | Application realization |
| Applications | User experience |

Responsibilities should not overlap.

---

# Runtime Boundary

Menata Runtime is responsible only for realizing Runtime Metadata.

The runtime does not own:

- Business Reality,
- Business Knowledge,
- Menata Language,
- metadata authoring.

The runtime only interprets Runtime Metadata.

---

# Metadata Flow

Applications evolve through metadata.

```text
Business changes
        │
        ▼
Business Knowledge changes
        │
        ▼
Menata Language changes
        │
        ▼
Runtime Metadata changes
        │
        ▼
Runtime interprets changes
        │
        ▼
Applications evolve
```

Business changes should not require rewriting application source code.

---

# Runtime Independence

The runtime should remain independent from:

- programming languages,
- rendering technologies,
- storage technologies,
- infrastructure,
- deployment environments.

Only Runtime Metadata should determine application behavior.

---

# Single Runtime

A single runtime may host:

- one application,
- dozens of applications,
- hundreds of applications,
- thousands of applications.

Applications remain independent through Runtime Metadata.

The runtime remains a single execution environment.

---

# Workspace Boundary

Workspace is the primary organizational boundary.

Each workspace owns:

- applications,
- metadata,
- permissions,
- governance,
- deployment configuration.

Cross-workspace interaction should always be explicit.

---

# Evolution

Business Reality evolves.

Business Knowledge evolves.

Runtime Metadata evolves.

Applications evolve.

The runtime itself also evolves.

Business Knowledge remains the long-term organizational asset.

---

# Related Research

The layered architecture above was informed by studying architecture patterns from other world-class systems — browser engines, Kubernetes, Terraform, React, VS Code, and others. See [architecture-benchmark.md](architecture-benchmark.md) for the full comparison and the design implications drawn from each.

---

# Summary

Menata Runtime follows a simple architectural philosophy.

Business Knowledge explains organizations.

Runtime Metadata explains applications.

Menata Runtime realizes Runtime Metadata.

Applications become living representations of Business Knowledge.
