# Prototype Architecture

> This document describes how the Menata Runtime Prototype maps to the Menata Runtime specification.
>
> It describes responsibilities, not implementation details.

---

## Overview

The prototype follows the layered architecture defined in the runtime specification.

```text
Browser
    │  HTML + HTMX requests
    ▼
Menata Runtime (Go HTTP Server)
    │
    ├── Router          — maps URLs to page handlers
    ├── Interpreter     — resolves Runtime Metadata into application behavior
    ├── Renderer        — produces HTML responses via Templ
    ├── Event Executor  — executes business events
    ├── Constraint Engine — enforces business rules
    └── Permission Guard  — enforces business authority
    │
    ▼
PostgreSQL
    │
    ├── metadata schema   — Runtime Metadata (Workspace, Application, Machine, etc.)
    └── data schema       — Business Data produced by running applications
```

---

## Layers

### Runtime Metadata Layer

Runtime Metadata is stored in PostgreSQL as structured records.

It describes applications, machines, pages, views, fields, events, constraints, and permissions.

Runtime Metadata is loaded at startup and reloaded on change.

The runtime interprets Runtime Metadata.

The runtime never hardcodes application behavior.

---

### Interpreter

The Interpreter is the core of the runtime.

It reads Runtime Metadata and builds an Application Model in memory.

The Application Model is what the runtime operates on.

It is never exposed directly to users.

```text
Runtime Metadata (PostgreSQL)
        │
        ▼
Interpreter
        │
        ▼
Application Model (in-memory)
        │
        ▼
Router / Renderer / Executor
```

This pattern is inspired by how browser engines work:

```text
HTML → Parser → DOM → Renderer → Browser
```

Similarly:

```text
Runtime Metadata → Interpreter → Application Model → Renderer → Running Application
```

---

### Router

The Router maps incoming HTTP requests to the correct application, machine, and page.

Routes are derived from Runtime Metadata.

Routes are never hardcoded.

---

### Renderer

The Renderer produces HTML responses from the Application Model.

It uses Templ templates.

HTMX handles partial page updates without full page reloads.

Hyperscript handles simple client-side behavior (modals, toggles, inline feedback).

The Renderer is independent from the Interpreter.

Multiple renderers may be added in the future (e.g., JSON API renderer).

---

### Event Executor

The Event Executor runs business events defined in Runtime Metadata.

Examples:

- `When Submit` → sets status, sends notification
- `Every Day` → triggers scheduled behavior

Event execution is entirely driven by Runtime Metadata.

---

### Constraint Engine

The Constraint Engine enforces business rules before and after events.

Constraints are defined in Runtime Metadata.

The engine never hardcodes business rules.

---

### Permission Guard

The Permission Guard enforces which roles may perform which events (CAP-P01), further narrowed
to the specific identity a Permission's `owner_field` names, not just the role class (CAP-P02) —
and, independently of Events, which roles may read/create/edit a Machine's records at all
(CAP-P05, deny-by-default: no Permission row on a Machine means no access).

Permissions are defined in Runtime Metadata.

---

## Runtime Model Mapping

The prototype implements the Runtime Model hierarchy defined in the specification.

```text
Workspace
    └── Application
            └── Machine
                    ├── Field
                    ├── Event
                    ├── Constraint
                    ├── Permission
                    └── View (List | Detail | Form)
```

| Runtime Model Concept | Prototype Realization |
|-----------------------|----------------------|
| Workspace | PostgreSQL schema boundary |
| Application | Group of machines with shared navigation — the workspace home (CAP-O03) lists Applications, drilling into one lists its own Machines, both role-scoped |
| Machine | Core realization unit — owns fields, events, constraints, permissions, views |
| Field | Typed business information (Text, Number, Date, User, Reference, Value List) |
| Event | Business occurrence trigger — executes actions on state change |
| Constraint | Business rule enforced before/after event execution |
| Permission | Role-based event authorization (CAP-P01), narrowed to a specific identity where a Permission declares one (CAP-P02), plus independent CRUD-level read/create/edit grants (CAP-P05) |
| View (List) | Table or card presentation of multiple records |
| View (Detail) | Full record presentation |
| View (Form) | Input surface for creating or updating records |

---

## Data Separation

Runtime Metadata and Business Data are stored separately.

```text
PostgreSQL
    ├── metadata.*     — Runtime Metadata tables
    └── data.*         — Business Data tables (generated from Machine definitions)
```

Runtime Metadata defines structure.

Business Data holds actual organizational records.

Business Data should always be preserved even when Runtime Metadata evolves.

---

## Lifecycle

The prototype follows the Runtime Lifecycle defined in the specification.

```text
Startup
    │
    ▼
Load Runtime Metadata from PostgreSQL
    │
    ▼
Validate Runtime Metadata
    │
    ▼
Build Application Model (in-memory)
    │
    ▼
Start HTTP Server
    │
    ▼
Serve Requests
    │
    ▼
On Metadata Change → Reload → Rebuild Model → Continue Serving
```

Invalid Runtime Metadata is rejected.

Running applications remain stable during reload failures.

---

## Prototype Constraints

**Corrected 2026-08-22** — this section described the prototype's very first cut (Cases 1–2) and
had gone stale: a real background scheduler (CAP-E02/E03), external webhook ingestion (CAP-E04),
and an auto-generated JSON API (CAP-X07) have all existed and been conformance-proven since
2026-07-12. Kept here, corrected, rather than deleted, so the history of what this prototype
originally scoped out (and later built) stays visible.

Remaining genuine simplifications, as of this correction:

- No object storage — uploaded files (CAP-F06) live on local disk, not S3/GCS/etc.
- No message queue / async job runner — every action (including slow ones: notification
  fan-out, cross-machine subscription dispatch, batch generation) still runs inline inside the
  triggering request's own transaction. Named as a real, not-yet-closed gap — CAP-W06 in
  `../capability-registry.md`, motivated by `../brd-menata-runtime-v2.md`'s Concept C analysis.
- No metadata live-reload — a metadata/seed change needs a server restart to take effect
  (CAP-X04).
- Single-process, single-host deployment — no horizontal scaling story built yet (Study 8,
  `../benchmarks/004-scale-architecture-study.md`, designs one; not implemented).

These remaining limitations exist to keep the prototype focused on validating the core
interpretation model before taking on production-operations concerns.

---

## Package Structure

The current `internal/` layout is flat — one package per concern (`config`, `constraint`, `db`,
`executor`, `handler`, `interpreter`, `metadata`, `model`, `permission`, `router`, `store`, `ui`).
This is correct for what the prototype validates today (Cases 1–2). It is not the final shape.

As the runtime grows to support the capabilities tracked in `../capability-registry.md` — reference
fields, pluggable actions, time-driven events, report views, workspace services — the package
structure evolves into layered seams (`core/`, `engine/`, `metadata/`, `store/`, `security/`,
`web/`, `platform/`) so that new capabilities are additive (register a plugin) rather than edits to
existing `switch` statements.

The target layout, the reasoning, and the capability-triggered migration plan (no big-bang refactor)
are recorded in [docs/decisions/004-internal-package-architecture.md](docs/decisions/004-internal-package-architecture.md).
