# Menata Runtime — Drupal Prototype

> A metadata-driven application runtime built on Drupal 10.

---

## Status

**Prototype** — active development.

---

## How It Works

```text
Business Knowledge (.menata)
        │
        ▼
Runtime Metadata (.yaml)
        │
        ▼
Drupal Runtime (menata_runtime module)
        │
        ▼
Running Application (Drupal)
```

1. Business Knowledge is expressed using Menata Language.
2. Runtime Metadata describes how that knowledge should be realized.
3. The `menata_runtime` Drupal module interprets Runtime Metadata.
4. Drupal realizes the application using its built-in entity, views, workflow, and permission systems.

---

## The Same YAML. A Different Runtime.

The Runtime Metadata YAML used by this prototype is identical to the one used by the Go prototype.

The same `design-request.yaml` that produces a Go + HTMX application also produces a Drupal application.

This validates the Menata principle:

> Business Knowledge should remain independent from implementation technology.

---

## Tech Stack

| Concern | Technology |
|---------|-----------|
| Runtime Engine | Drupal 10 + PHP 8.2 |
| Database | PostgreSQL |
| Runtime Metadata Interpreter | Custom module: `menata_runtime` |
| Content Modeling | Drupal Content Types + Fields API |
| State Transitions | Drupal Workflows module |
| Event Handling | ECA module (Events, Conditions, Actions) |
| List Views | Drupal Views module |
| Forms | Drupal Form API |
| Permissions | Drupal Roles + Permissions |
| CLI | Drush |

See [docs/decisions/001-techstack.md](docs/decisions/001-techstack.md) for rationale.

---

## Prototype Scope

In scope:

- Runtime Metadata loading via `menata_runtime` module
- Machine → Content Type realization
- Field → Drupal Field realization
- Event → ECA + Workflow realization
- Constraint → Field validation realization
- Permission → Drupal Role + Permission realization
- View (list) → Drupal Views realization
- View (detail) → Node display mode realization
- View (form) → Drupal edit form realization

Out of scope (future iterations):

- Multi-workspace support
- External integrations
- Scheduling beyond Drupal cron
- API exposure (JSON:API is available but not configured by this prototype)

---

## Metadata Coverage

**Score: ~85% (14/16 features)**

The metadata files in `docs/examples/drupal-config/` are native Drupal configuration — importable via `drush config:import` without any custom code.

| # | Feature | File | Status |
|---|---------|------|--------|
| 1 | Machine definition | node.type.design_request.yml | ✅ Metadata only |
| 2 | All Fields with correct types | field.storage.node.*.yml | ✅ Metadata only |
| 3 | Status field + all states | field.storage.node.field_status.yml | ✅ Metadata only |
| 4 | State machine enforcement | workflows.workflow.design_request.yml (role per transition) | ✅ Metadata only |
| 5 | Event action: set status on transition | Workflow state update | ✅ Metadata only |
| 6 | Event action: notify Designer on Submit | eca.eca.design_request_notify.yml | ✅ Metadata only |
| 7 | Event action: notify Requester on Complete | eca.eca.design_request_notify.yml | ✅ Metadata only |
| 8 | Constraint: Title required | field config (required flag) | ✅ Metadata only |
| 9 | Constraint: Description required | field config (required flag) | ✅ Metadata only |
| 10 | Constraint: Due Date after today | — | ❌ Cannot be done without code — Drupal has no native "after today" date constraint configurable via YAML; requires PHP constraint plugin |
| 11 | Constraint: Attachment if Design Type = Banner | — | ❌ Cannot be done without code — needs PHP validation in a custom module |
| 12 | Permission: Requester role | user.role.requester.yml | ✅ Metadata only |
| 13 | Permission: Designer role | user.role.designer.yml | ✅ Metadata only |
| 14 | View: Form | Drupal edit form (auto-generated) | ✅ Metadata only |
| 15 | View: List | views.view.design_request_my_requests.yml | ✅ Metadata only |
| 16 | View: Detail | Node display mode (auto-generated) | ✅ Metadata only |

Note: The `menata_runtime` custom module described in the architecture is the Drupal implementation of the Menata Runtime interpreter — a separate code layer. The metadata proof tests only what native Drupal configuration can express without it.

## Getting Started

See [DEVELOPMENT.md](DEVELOPMENT.md).

---

## Documentation

- [ARCHITECTURE.md](ARCHITECTURE.md) — how Drupal concepts map to the Menata Runtime specification
- [DEVELOPMENT.md](DEVELOPMENT.md) — setup and local development
- [docs/drupal-mapping.md](docs/drupal-mapping.md) — Runtime Metadata → Drupal concept mapping
- [docs/examples/](docs/examples/) — example Business Knowledge and Runtime Metadata
- [docs/decisions/](docs/decisions/) — architectural decisions
