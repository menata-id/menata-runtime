# Menata Runtime — Prototype

> One Business Knowledge. Many runtimes. Same result.

---

## Status

**Prototype** — active research and implementation.

These prototypes explore the feasibility of the Menata Runtime architecture across different implementation technologies.

---

## The Point

Business Knowledge is expressed once in Menata Language.

The same Business Knowledge can be realized by different runtimes.

```text
design-request.menata
        │
        ├──▶ Go Runtime          ──▶ Running Application (Go + HTMX)
        ├──▶ Drupal Runtime      ──▶ Running Application (Drupal)
        ├──▶ Frappe Runtime      ──▶ Running Application (Frappe)
        ├──▶ Directus Runtime    ──▶ Running Application (Directus)
        ├──▶ Budibase Runtime    ──▶ Running Application (Budibase)
        ├──▶ Salesforce Runtime  ──▶ Running Application (Salesforce)
        └──▶ Camunda Runtime     ──▶ Running Application (Camunda)
```

The Business Knowledge (`design-request.menata`) remains unchanged.

From it, Runtime Metadata is derived — a stable bridge that any interpreter can read.

Only the interpreter differs.

This validates the core Menata principle:

> Business Knowledge should remain independent from implementation technology.

---


## Metadata-Only Testing Goal

The metadata proof prototypes test one specific question:

> Can an existing framework realize a business application from metadata alone, without writing additional code?

The Menata Runtime interpreter (Go prototype) is a separate code layer. The metadata proofs are independent of it — they test whether each platform's own engine can interpret the metadata natively.

Coverage is scored against 16 features derived from design-request.menata (Machine, Fields, Events, Constraints, Permissions, Views).

| Prototype | Score | Coverage | Not Coverable Without Code |
|-----------|-------|----------|---------------------------|
| [salesforce/](salesforce/) | **100%** | 16/16 | — |
| [frappe/](frappe/) | **~85%** | 14/16 | Due date constraint, conditional attachment constraint |
| [drupal/](drupal/) | **~85%** | 14/16 | Due date constraint, conditional attachment constraint |
| [camunda/](camunda/) | **~80%** | 13/16 | Service Task notify (connector worker code) |
| [directus/](directus/) | **~70%** | 11/16 | State machine enforcement (no workflow module), 2 constraints not yet in proof |
| [budibase/](budibase/) | **~65%** | 10/16 | State machine enforcement, date constraint, conditional constraint |
| [go/](go/) | — | — | Custom runtime — not a metadata-only test |

Note: Camunda scores higher than Directus/Budibase because DMN covers all 4 constraints as pure metadata (including the hardest conditional one), even though its notifications need connector code.

---

## Prototypes

| Prototype | Technology | Type | Status |
|-----------|-----------|------|--------|
| [go/](go/) | Go + PostgreSQL + Templ + HTMX + Hyperscript | Custom runtime | In Progress |
| [drupal/](drupal/) | Drupal 10 + PHP + PostgreSQL | CMS framework | Metadata Proof |
| [frappe/](frappe/) | Frappe Framework + Python + PostgreSQL | Business framework | Metadata Proof |
| [directus/](directus/) | Directus 10 + Node.js + PostgreSQL | Database-first | Metadata Proof |
| [budibase/](budibase/) | Budibase + Node.js + PostgreSQL | Low-code platform | Metadata Proof |
| [salesforce/](salesforce/) | Salesforce Metadata API | Enterprise platform | Metadata Proof |
| [camunda/](camunda/) | Camunda 8 + BPMN + DMN | Process engine | Metadata Proof |

---

## From Business Knowledge to Application

The same `design-request.menata` is the source for all prototypes.

Each prototype derives Runtime Metadata (`design-request.yaml`) from that Business Knowledge, then interprets it into a running application using its own native format.

See examples in each prototype's `docs/examples/` directory.

---

## Scope

All prototypes focus on validating core runtime concepts:

- Runtime Metadata loading and validation
- Object and Field realization
- List, Detail, and Form views
- Event execution
- Constraint enforcement
- Permission enforcement

---

## Relationship to Specification

These prototypes implement concepts defined in the Menata Runtime specification.

| Specification | Document |
|--------------|---------|
| Design Principles | [../001-design-principles.md](../001-design-principles.md) |
| Architecture | [../002-architecture.md](../002-architecture.md) |
| Runtime Language | [../003-runtime-language.md](../003-runtime-language.md) |
| Runtime Metadata | [../004-runtime-metada.md](../004-runtime-metada.md) |
| Runtime Lifecycle | [../005-runtime-lifecycle.md](../005-runtime-lifecycle.md) |
| Runtime Model | [../006-runtime-model.md](../006-runtime-model.md) |
