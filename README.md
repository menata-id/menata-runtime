# Menata Runtime

> **Applications should evolve at the pace of Business Knowledge.**
>
> **A runtime that realizes Business Knowledge as living applications.**

## Overview

Business Knowledge explains how organizations work.

Menata provides a language for expressing that knowledge.

However, Business Knowledge alone does not become software.

Applications must be realized.

Pages.

Navigation.

Forms.

Dashboards.

Services.

Authentication.

Authorization.

Notifications.

Scheduling.

Integrations.

Search.

APIs.

These concerns belong to the runtime.

**Menata Runtime exists to realize Business Knowledge as living applications.**

Applications are not manually programmed.

Applications are interpreted from Runtime Metadata.

Applications evolve because Business Knowledge evolves.

---

> ⚠️ **Research Draft**
>
> Menata Runtime is an active research project.
>
> Runtime architecture, Runtime Language, Runtime Metadata, and the application engine are evolving continuously.
>
> Breaking changes are expected before version **1.0**.

---

# Why Menata Runtime?

Organizations continuously create new Business Knowledge.

Unfortunately, software rarely evolves at the same pace.

Every business change usually requires:

- redesign,
- implementation,
- testing,
- deployment,
- maintenance.

Over time, organizations accumulate far more Business Knowledge than they can realistically implement as software.

The problem is no longer capturing Business Knowledge.

The problem is realizing Business Knowledge into working applications.

Menata Runtime exists to solve that problem.

Instead of manually building every application, Runtime Metadata describes application intent.

The runtime continuously realizes that metadata into living applications.

---

# Vision

We believe every Business Knowledge deserves implementation.

Applications should evolve at the pace of Business Knowledge.

Business should drive software.

Not the other way around.

The runtime should:

- continuously realize Business Knowledge,
- minimize handwritten application code,
- maximize metadata reuse,
- remain independent from implementation technology,
- evolve without requiring applications to be rewritten.

Business Knowledge remains stable.

The runtime evolves.

Applications continuously evolve.

---

# Architecture

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
Any Compatible Tool

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

Business Reality explains what actually happens.

Business Knowledge explains why it happens.

Menata Language formally expresses Business Knowledge.

Runtime Metadata expresses how applications should be realized.

Menata Runtime realizes Runtime Metadata into executable applications.

---

# What is Menata Runtime?

Menata Runtime is a metadata-driven application runtime.

It interprets Runtime Metadata into complete applications.

A running application may include:

- pages,
- forms,
- tables,
- dashboards,
- workflows,
- navigation,
- menus,
- APIs,
- background jobs,
- notifications,
- authentication,
- authorization,
- search,
- integrations,
- platform services.

Applications are interpreted.

Applications are not generated.

Runtime Metadata plays a role similar to HTML in a web browser.

The runtime does not care how Runtime Metadata was created.

---

# Runtime Metadata

Runtime Metadata describes application realization.

It is designed primarily for deterministic machine interpretation.

Runtime Metadata may be produced by:

- Menata Apps Builder,
- Visual Builders,
- Command-line tools,
- Manual editors,
- AI-assisted tools,
- Any compatible implementation.

Menata Runtime only interprets Runtime Metadata.

It never depends on how the metadata was authored.

---

# Metadata-Driven Applications

Applications are described by Runtime Metadata rather than handwritten application source code.

Changing Runtime Metadata changes applications.

No code generation is required.

No scaffolding is required.

No duplicated CRUD implementation is required.

A single runtime may realize:

- one application,
- dozens of applications,
- hundreds of applications,
- thousands of applications.

All applications live inside the same runtime.

Applications are isolated by metadata.

Not by runtime instances.

---

# Runtime Responsibilities

Menata Runtime is responsible for:

- interpreting Runtime Metadata,
- realizing applications,
- application lifecycle,
- routing,
- navigation,
- rendering,
- authentication,
- authorization,
- event execution,
- constraint enforcement,
- search,
- scheduling,
- notifications,
- API exposure,
- application hosting,
- platform services.

Business Knowledge remains independent from runtime implementation.

---

# Design Principles

Menata Runtime is built upon the following principles.

## Core Principles

- Machine First
- Runtime First
- Metadata First
- Declarative

## Architecture Principles

- Convention over Configuration
- Infer Before Configure
- Composable
- Reference over Duplication
- Workspace Isolation

## Evolution Principles

- Live Evolution
- Data Preservation
- Long-term Compatibility
- Technology Adaptable

## Platform Principles

- Open Platform
- Extensible Runtime
- Single Runtime

## Vision

Applications evolve at the pace of Business Knowledge.

The complete rationale is available in:

`design-principles.md`

---

# Long-term Vision

Menata Runtime aims to become a universal metadata-driven application runtime.

A single runtime should realize one application or thousands of independent applications.

Applications should continuously evolve without rewriting application source code.

Business Knowledge should remain stable.

Runtime Metadata should evolve.

The runtime should evolve.

Applications should continuously reflect Business Knowledge.

---

# The Menata Ecosystem

The Menata ecosystem consists of independent open-source projects.

## Menata

Business Knowledge Language.

Designed for humans.

Defines Business Knowledge.

## Menata Runtime

Application Runtime.

Designed for machines.

Realizes Runtime Metadata into applications.

## Menata Apps Builder

Application Builder.

Designed for platform builders.

Produces Runtime Metadata.

The runtime does not depend on Menata Apps Builder.

Any tool capable of producing compatible Runtime Metadata can be used.

---

# Documentation

This directory mixes documents that change on different rhythms. To read the numbering correctly:

- **Numbered `001`–`006`** — the core specification. Stable, read in sequence, changes rarely (mirrors `specification/000`–`006` at the repo root — same convention, one level down).
- **Unnumbered, at this level** — supporting reference or governance documents. Two different kinds, distinguished below.
- **`benchmarks/` — numbered `000`–`004`** — evidence studies, numbered by production order (Study 1 → `000`, Study 2 → `001`, …), not a required reading sequence.

This stable/evolving split mirrors the pattern Portal GA v3 uses for its own domain-integration framework (`01-CONSTITUTIONAL-BRD.md` + `02-IMPLEMENTATION-GUIDE.md` as STABLE, `appendices/` as EVOLVING) — see `benchmarks/002-portal-ga-cross-domain-survey.md`.

## Tier 1 — Core Specification (stable, numbered)

| Document | Covers |
|----------|--------|
| [001-design-principles.md](001-design-principles.md) | Runtime architectural philosophy |
| [002-architecture.md](002-architecture.md) | Conceptual architecture, layer responsibilities |
| [003-runtime-language.md](003-runtime-language.md) | Runtime Language — how applications are described |
| [004-runtime-metadata.md](004-runtime-metadata.md) | Runtime Metadata — scope, hierarchy, versioning |
| [005-runtime-lifecycle.md](005-runtime-lifecycle.md) | How metadata continuously realizes running applications |
| [006-runtime-model.md](006-runtime-model.md) | Runtime object model — Workspace, Application, Machine, and beyond |

## Tier 2 — Supporting Reference (unnumbered, informs Tier 1)

Not part of the numbered reading sequence — each elaborates or grounds one Tier 1 document with concrete detail or external research, rather than adding new normative content.

| Document | Elaborates | Role |
|----------|-----------|------|
| [runtime-metadata-schema.md](runtime-metadata-schema.md) | §004 | Concrete YAML/DB schema for Runtime Metadata (shared by all prototypes) |
| [architecture-benchmark.md](architecture-benchmark.md) | §002 | Architecture references from other world-class systems (Chromium, Kubernetes, VS Code, …) — one-time research that informed the architecture, not part of the growing `benchmarks/` evidence series below |

## Practical Guides

| Document | Audience |
|----------|----------|
| [../guides/writing-menata.md](../guides/writing-menata.md) | Domain expert — how to write `.menata` |
| [guides/writing-runtime-metadata.md](guides/writing-runtime-metadata.md) | Developer — how to translate `.menata` into Runtime Metadata |

## Tier 3 — Capability Discovery & Governance (evolving)

The runtime's capability is being built and verified through a deliberate discovery process — cases, external benchmarks, and lifecycle governance — rather than ad hoc feature addition. These documents change continuously as each study/phase completes; unlike Tier 1, there is no expectation of stability.

| Document | Role |
|----------|------|
| [roadmap.md](roadmap.md) | The method + phased work plan (start here) |
| [capability-registry.md](capability-registry.md) | Single source of record — every known capability, its status, and priority |
| [case-portfolio.md](case-portfolio.md) | Deliberately chosen test cases and their declared targets |
| [capability-lifecycle.md](capability-lifecycle.md) | How a new capability is proposed, admitted, and completed |
| [nfr-standards.md](nfr-standards.md) | Architecture / performance / security standards per capability area |
| [benchmarks/](benchmarks/) | Tier 4 — external evidence studies (see below) |

## Tier 4 — Evidence Studies (`benchmarks/`, numbered by production order)

| Document | Study |
|----------|-------|
| [benchmarks/000-workflow-patterns-mapping.md](benchmarks/000-workflow-patterns-mapping.md) | Study 1 — Menata vs Workflow Patterns Initiative |
| [benchmarks/001-platform-capability-survey.md](benchmarks/001-platform-capability-survey.md) | Study 2 — cross-platform capability survey (Salesforce, Frappe, Drupal, Directus, Budibase, Camunda) |
| [benchmarks/002-portal-ga-cross-domain-survey.md](benchmarks/002-portal-ga-cross-domain-survey.md) | Study 5 — Portal GA v3 cross-domain integration patterns |
| [benchmarks/003-accounting-vertical-survey.md](benchmarks/003-accounting-vertical-survey.md) | Study 6 — accounting vertical (Odoo / ERPNext) |
| [benchmarks/004-scale-architecture-study.md](benchmarks/004-scale-architecture-study.md) | Study 8 — multi-workspace scale & performance architecture |
| [benchmarks/005-field-modeling-decision-framework.md](benchmarks/005-field-modeling-decision-framework.md) | Study 15 — field type selection: reference vs. value_list vs. primitive |

## Reference Implementation

| Location | What it is |
|----------|-----------|
| [prototype/go/](prototype/go/) | The working Go + PostgreSQL + Templ + HTMX prototype |
| [prototype/{salesforce,frappe,drupal,directus,budibase,camunda}/](prototype/) | Metadata-only proofs on other platforms — see [prototype/README.md](prototype/README.md) for the comparison scorecard |

---

# Contributing

Menata Runtime is currently focused on:

- Runtime Architecture,
- Runtime Language,
- Runtime Metadata,
- Metadata-driven Applications,
- Application Engine,
- Platform Services.

Ideas, discussions, critiques, implementation strategies, architectural proposals, and research references are highly appreciated.

---

# License

Licensed under the Apache License 2.0.

See the LICENSE file for details.
