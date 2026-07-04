# ADR-001 — Tech Stack Selection

Date: 2026-07-04

Status: Accepted

---

## Context

The Menata Runtime Prototype needs a concrete tech stack to validate the runtime architecture.

The stack must be chosen carefully.

The Menata Runtime specification explicitly states that the runtime should remain independent from implementation technology.

The tech stack chosen here applies to this prototype only.

A future production runtime may use a different tech stack while interpreting the same Runtime Metadata.

---

## Decision

| Concern | Technology | Rationale |
|---------|-----------|-----------|
| Runtime Engine | Go | Compiled, fast, strong standard library, excellent HTTP server, goroutines for concurrent event execution. Straightforward to reason about. |
| Database | PostgreSQL | Already available on the server. JSONB support for flexible Runtime Metadata storage. Reliable, battle-tested. |
| Templates | Templ | Type-safe HTML templates for Go. Compile-time template correctness aligns with "machine correctness is mandatory" principle. Works naturally with HTMX. |
| Partial Updates | HTMX | Server-driven partial page updates without JavaScript. Keeps business behavior on the server where it belongs. |
| Client Behavior | Hyperscript | Constrained, declarative client-side scripting for UI-only concerns (modals, toggles, inline feedback). More constrained than Alpine.js — discourages moving business logic to the client. |
| Styling | Tailwind CSS | Utility-first CSS. Minimal custom CSS. Consistent UI without a heavy design framework. |

---

## Alternatives Considered

### SQLite instead of PostgreSQL

SQLite is simpler to set up and requires no server.

Rejected because PostgreSQL is already available on the server.

PostgreSQL JSONB provides better query performance for Runtime Metadata than SQLite's text-based JSON functions.

### Alpine.js instead of Hyperscript

Alpine.js is more widely adopted and has a larger ecosystem.

Rejected because Alpine.js encourages reactive state management and client-side logic.

This conflicts with the Menata Runtime principle that the runtime owns application behavior, not the client.

Hyperscript is more constrained by design and discourages complex client-side logic.

### Standard `html/template` instead of Templ

Go's standard `html/template` is simpler and has no compilation step.

Rejected because Templ provides type safety at compile time, which reduces runtime template errors and aligns better with the prototype's goal of correctness.

---

## Consequences

The prototype tech stack is nearly identical to an existing production application (portal-ga3).

This has the following implications:

- The team is already familiar with all chosen technologies.
- Existing patterns can be referenced and reused.
- The stack is proven in a production environment.

The chosen stack does not represent a commitment for a production Menata Runtime.

Future runtime implementations may choose different technologies while remaining compatible with the same Runtime Metadata format.

---

## Principles Applied

- **Machine First** — Templ enforces template correctness at compile time.
- **Runtime First** — HTMX keeps behavior server-side.
- **Technology Adaptable** — Runtime Metadata format is independent from this tech stack.
- **Convention over Configuration** — all chosen tools favor conventions over boilerplate.
