# Menata Runtime

> **A runtime that interprets Menata Language into living applications.**

Business Knowledge already exists.

Organizations know how work is performed.

They know the rules.

They know the decisions.

They know the workflows.

Menata provides a language for expressing that Business Knowledge.

However, Business Knowledge alone does not create applications.

Applications require interpretation.

Pages.

Navigation.

Interaction.

Services.

Application lifecycle.

**Menata Runtime exists to realize Business Knowledge as running applications.**

Applications are not manually programmed.

Applications are interpreted.

Applications evolve because Business Knowledge evolves.

---

> ⚠️ **Research Draft**
>
> Menata Runtime is an active research project.
>
> The runtime architecture, runtime language, and metadata model are evolving and **breaking changes are expected** before version **1.0**.
>
> The current focus of this repository is runtime architecture and platform design.
>
> Feedback, discussions, critiques, and implementation ideas are highly appreciated.

---

# Why Menata Runtime?

Business Knowledge should not depend on implementation technology.

The same Business Knowledge should be capable of becoming many different applications.

The challenge is no longer describing Business Knowledge.

The challenge is realizing that knowledge consistently.

Menata Runtime bridges the gap between Business Knowledge and running applications.

---

# Vision

We believe every Business Knowledge deserves implementation.

Applications should:

- evolve because Business Knowledge evolves,
- remain independent from implementation technology,
- be interpreted rather than manually programmed,
- remain adaptable as technologies evolve,
- become long-term organizational assets.

Business Knowledge should remain stable.

The runtime should continue evolving.

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
Menata Runtime
        │
        ▼
Applications
```

Business Reality is where work actually happens.

Business Knowledge explains how that work is performed.

Menata Language formally expresses that knowledge.

Menata Runtime interprets the language into running applications.

Applications become living realizations of Business Knowledge.

---

# Runtime Responsibilities

Menata Runtime is responsible for interpreting Business Knowledge into executable platform behavior.

Responsibilities include:

- application lifecycle
- page rendering
- navigation
- routing
- user interaction
- authentication
- authorization
- business event execution
- constraint enforcement
- notification
- background jobs
- search
- APIs
- AI integration
- application hosting

Business Knowledge remains independent from these runtime concerns.

---

# Runtime Language

Menata Runtime introduces its own language.

Unlike Menata Language, which expresses Business Knowledge, Runtime Language expresses Platform Intent.

Platform builders use Runtime Language to describe how Business Knowledge should be realized as applications.

Business Knowledge remains independent.

Platform behavior remains declarative.

The runtime interprets both into complete applications.

---

# Metadata-Driven Applications

Applications are defined by metadata rather than handwritten application code.

Changing metadata changes the application.

A single runtime may host:

- one application,
- dozens of applications,
- hundreds of applications,
- thousands of applications.

Applications are interpreted continuously.

No application regeneration is required.

---

# Design Principles

Menata Runtime is designed around several fundamental principles.

- Metadata First
- Runtime Interpretation
- Declarative
- Platform First
- Convention over Configuration
- Infer Before Configure
- Composable
- Runtime Native
- Live Evolution
- Technology Adaptable
- AI Ready
- Open Platform
- Long-term Compatibility

The complete rationale is available in `design-principles.md`.

---

# Long-term Vision

The Menata ecosystem is expected to include:

- Menata Language
- Menata Runtime
- Runtime Language
- AI-assisted Platform Builder
- Development Tools
- Reference Applications

Business Knowledge comes first.

The runtime realizes that knowledge.

Applications become one possible realization of Business Knowledge.

---

# Contributing

Menata Runtime is currently focused on runtime architecture, runtime language, and metadata-driven application design.

Ideas, critiques, implementation strategies, and architectural discussions are highly appreciated.

A formal RFC (Request for Comments) process will be introduced as the runtime matures.

---

# License

Licensed under the Apache License 2.0.

See the LICENSE file for details.
