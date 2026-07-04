# Menata Runtime — Prototype

> One Business Knowledge. Two runtimes. Same result.

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
        ├──▶ Go Runtime     ──▶ Running Application (Go + HTMX)
        │
        └──▶ Drupal Runtime ──▶ Running Application (Drupal)
```

The Menata Language remains unchanged.

The Runtime Metadata format remains unchanged.

Only the interpreter differs.

This validates the core Menata principle:

> Business Knowledge should remain independent from implementation technology.

---

## Prototypes

| Prototype | Technology | Type | Status |
|-----------|-----------|------|--------|
| [go/](go/) | Go + PostgreSQL + Templ + HTMX + Hyperscript | Custom runtime | In Progress |
| [drupal/](drupal/) | Drupal 10 + PHP + PostgreSQL | CMS framework | Metadata Proof |
| [frappe/](frappe/) | Frappe Framework + Python + PostgreSQL | Business framework | Planned |
| [directus/](directus/) | Directus 10 + Node.js + PostgreSQL | Database-first | Metadata Proof |
| [budibase/](budibase/) | Budibase + Node.js + PostgreSQL | Low-code platform | Metadata Proof |
| [salesforce/](salesforce/) | Salesforce Metadata API | Enterprise platform | Metadata Proof |
| [camunda/](camunda/) | Camunda 8 + BPMN + DMN | Process engine | Metadata Proof |

---

## Shared Runtime Metadata

Both prototypes interpret the same Runtime Metadata format.

The same YAML file that describes a Design Request in the Go prototype is the same YAML file used by the Drupal prototype.

See examples in each prototype's `docs/examples/` directory.

---

## Scope

Both prototypes focus on validating core runtime concepts:

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
