# ADR-001 — Tech Stack Selection (Drupal Prototype)

Date: 2026-07-04
Status: Accepted

---

## Context

This is a metadata proof: showing how design-request.yaml maps to native Drupal configuration.
The goal is to validate how much of Menata Runtime Metadata can be expressed as Drupal YAML config files,
without writing any custom code inside Drupal.

---

## Decision

| Concern | Technology | Rationale |
|---------|-----------|-----------|
| Runtime Engine | Drupal 10 + PHP 8.2 | Drupal was one of the architectural benchmarks cited in the Menata Runtime specification. Its native modules (Workflow, ECA, Views, Fields API, Roles) cover most Menata Runtime concepts out of the box. |
| Database | PostgreSQL | Already available on the server. Consistent with the Go prototype. |
| Content Modeling | Drupal Content Types + Fields API | Maps directly to Machine + Field in Runtime Metadata. Config expressed as YAML. |
| State Transitions | Drupal Workflows module | Maps directly to Events with status changes. Role-per-transition enforcement in YAML. |
| Event Handling | ECA module (Events, Conditions, Actions) | Maps directly to Event responses (notify, create record). Expressed as YAML config. |
| List Views | Drupal Views module | Maps directly to View type list. Expressed as views.view.*.yml. |
| Forms | Drupal Form API | Maps directly to View type form. Auto-generated from field config. |
| Permissions | Drupal Roles + Permissions | Maps directly to Permission in Runtime Metadata. Expressed as user.role.*.yml. |
| Config Import | Drush config:import | Standard Drupal CLI for importing YAML config. |

No custom Drupal module is required for this metadata proof.

---

## Why Drupal

The Menata Runtime architecture benchmark document identified Drupal as an architectural reference for:

- metadata-driven configuration (CMI — all config is YAML)
- reusable plugin architecture
- entity model
- views abstraction

Drupal proves that an existing mature framework can realize most of a Menata Runtime application
from pure YAML configuration — without requiring a custom interpreter module inside Drupal itself.

---

## Consequences

### Positive

- Drupal's entity, workflow, views, and permission systems map naturally to Menata Runtime concepts.
- No custom rendering engine required.
- Proven, mature, battle-tested framework.
- Drupal CMI uses YAML natively — aligned with Runtime Metadata format.
- State machine enforcement via Workflow module — something Directus and Budibase cannot do in metadata.

### Constraints

- Drupal cannot express "after today" date constraint or conditional required constraints in YAML alone.
- Complex event responses may require custom ECA plugins (PHP code).
- Drupal requires PHP. The production Menata Runtime interpreter may not use PHP.

---

## Principles Applied

- **Business Knowledge First** — Business Knowledge (.menata) is the source; Drupal config is a derived artifact.
- **Metadata First** — All Drupal realization is expressed as importable YAML config, not code.
- **Technology Adaptable** — The same Runtime Metadata YAML drives both the Go and Drupal prototypes.
