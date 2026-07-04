# Menata Runtime — Go Prototype

> A metadata-driven application runtime built with Go, PostgreSQL, Templ, HTMX, and Hyperscript.

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
Go Runtime (HTTP Server)
        │
        ▼
Running Application (HTML + HTMX)
```

1. Business Knowledge is expressed using Menata Language.
2. Runtime Metadata describes how that knowledge should be realized.
3. The Go runtime interprets Runtime Metadata into a running application.
4. Users interact with the application through a browser.

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

## Prototype Scope

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

## Getting Started

See [DEVELOPMENT.md](DEVELOPMENT.md).

---

## Documentation

- [ARCHITECTURE.md](ARCHITECTURE.md) — how this prototype maps to the runtime specification
- [DEVELOPMENT.md](DEVELOPMENT.md) — setup and local development
- [docs/runtime-metadata-schema.md](docs/runtime-metadata-schema.md) — Runtime Metadata format reference
- [docs/examples/](docs/examples/) — example Business Knowledge and Runtime Metadata
- [docs/decisions/](docs/decisions/) — architectural decisions
