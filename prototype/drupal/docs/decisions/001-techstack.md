# ADR-001 — Tech Stack Selection (Drupal Prototype)

Date: 2026-07-04

Status: Accepted

---

## Context

The Menata Runtime Drupal Prototype needs a concrete tech stack.

The Drupal prototype exists alongside the Go prototype to validate the Menata principle:

> The same Business Knowledge may be realized through different Machine Interpretations.

The tech stack chosen here is intentionally different from the Go prototype to prove implementation independence.

---

## Decision

| Concern | Technology | Rationale |
|---------|-----------|-----------|
| Runtime Engine | Drupal 10 + PHP 8.2 | Drupal was one of the architectural benchmarks cited in the Menata Runtime specification. It provides entity model, views, workflow, ECA, and permissions out of the box — all mapping directly to Menata Runtime concepts. |
| Database | PostgreSQL | Already available on the server. Consistent with the Go prototype. |
| Interpreter | Custom module `menata_runtime` | A thin Drupal module that reads Runtime Metadata YAML and realizes it using Drupal's built-in APIs. No modification to Drupal core required. |
| Content Modeling | Drupal Content Types + Fields API | Maps directly to Machine + Field in Runtime Metadata. |
| State Transitions | Drupal Workflows module | Maps directly to Events with status changes. |
| Event Handling | ECA module (Events, Conditions, Actions) | Maps directly to Event responses (notify, create record). |
| List Views | Drupal Views module | Maps directly to View type `list`. |
| Forms | Drupal Form API | Maps directly to View type `form`. |
| Permissions | Drupal Roles + Permissions | Maps directly to Permission in Runtime Metadata. |
| CLI | Drush | Standard Drupal CLI for automation. |

---

## Why Drupal

The Menata Runtime architecture benchmark document identified Drupal as an architectural reference specifically for:

- metadata-driven configuration
- reusable components
- plugin architecture
- entity model
- views abstraction

Drupal proves that an existing mature framework can act as a Menata Runtime without requiring a custom runtime to be built from scratch.

The `menata_runtime` module is intentionally thin.

It delegates all application realization to Drupal's existing systems.

---

## Consequences

### Positive

- Drupal's entity, workflow, views, and permission systems map naturally to Menata Runtime concepts.
- No custom rendering engine required.
- Proven, mature, battle-tested framework.
- Drupal's CMI (Configuration Management Interface) uses YAML natively — aligned with Runtime Metadata format.

### Constraints

- Drupal requires PHP. The production Menata Runtime may not use PHP.
- Drupal's interpretation of Runtime Metadata is constrained by what Drupal natively supports.
- Complex event responses may require custom ECA plugins.

---

## Principles Applied

- **Machine First** — Drupal config YAML is deterministic and machine-generated.
- **Runtime First** — Drupal owns application realization. The `menata_runtime` module only interprets.
- **Technology Adaptable** — The same Runtime Metadata YAML drives both the Go and Drupal prototypes.
- **Open Platform** — Drupal's plugin architecture extends runtime capabilities without modifying core.
