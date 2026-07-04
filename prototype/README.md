# Menata Runtime — Prototype

> A metadata-driven application runtime that realizes Business Knowledge as living applications.

---

## Status

**Prototype** — active research and implementation.

This is an early-stage prototype exploring the feasibility of the Menata Runtime architecture.

Breaking changes are expected.

---

## What This Is

Menata Runtime Prototype is an implementation of the [Menata Runtime specification](../README.md).

It interprets Runtime Metadata into running applications.

Applications are not generated.

Applications are not manually programmed.

Applications are realized from Runtime Metadata.

---

## What This Is Not

This prototype does not define Business Knowledge.

Business Knowledge is defined by the [Menata Language](../../specification/).

This prototype realizes Business Knowledge expressed as Runtime Metadata.

---

## How It Works

```text
Business Knowledge
        │
        ▼
Runtime Metadata (YAML)
        │
        ▼
Menata Runtime (Go)
        │
        ▼
Running Application (HTML + HTMX)
```

1. Business Knowledge is expressed using Menata Language.
2. Runtime Metadata describes how that knowledge should be realized.
3. Menata Runtime interprets Runtime Metadata into a running application.
4. Users interact with the application through a browser.

---

## Prototype Scope

This prototype focuses on validating the core runtime concepts.

In scope:

- Runtime Metadata loading and validation
- Object and Field realization
- List, Detail, and Form views
- Event execution
- Constraint enforcement
- Permission enforcement
- Basic navigation

Out of scope (future iterations):

- Multi-workspace support
- External integrations
- Scheduling and background jobs
- API exposure
- Notifications

---

## Tech Stack

| Concern | Technology |
|---------|-----------|
| Runtime Engine | Go |
| Database | PostgreSQL |
| Templates | Templ |
| Interactivity | HTMX + Hyperscript |
| Styling | Tailwind CSS |

See [docs/decisions/001-techstack.md](docs/decisions/001-techstack.md) for rationale.

---

## Getting Started

See [DEVELOPMENT.md](DEVELOPMENT.md).

---

## Documentation

- [ARCHITECTURE.md](ARCHITECTURE.md) — how the prototype maps to the runtime specification
- [DEVELOPMENT.md](DEVELOPMENT.md) — setup and local development
- [docs/runtime-metadata-schema.md](docs/runtime-metadata-schema.md) — Runtime Metadata format reference
- [docs/examples/](docs/examples/) — example Business Knowledge and Runtime Metadata
- [docs/decisions/](docs/decisions/) — architectural decisions

---

## Relationship to Specification

This prototype implements concepts defined in the Menata Runtime specification.

| Specification | Document |
|--------------|---------|
| Design Principles | [001-design-principles.md](../001-design-principles.md) |
| Architecture | [002-architecture.md](../002-architecture.md) |
| Runtime Language | [003-runtime-language.md](../003-runtime-language.md) |
| Runtime Metadata | [004-runtime-metada.md](../004-runtime-metada.md) |
| Runtime Lifecycle | [005-runtime-lifecycle.md](../005-runtime-lifecycle.md) |
| Runtime Model | [006-runtime-model.md](../006-runtime-model.md) |

---

## License

Licensed under the Apache License 2.0.
