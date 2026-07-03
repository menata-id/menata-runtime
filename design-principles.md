# Menata Runtime Design Principles

> Menata Runtime is a product.
>
> These principles guide the architecture and evolution of the runtime.
>
> They are intentionally implementation independent and technology adaptable.

---

# 1. Machine First

Runtime Language is designed primarily for deterministic machine interpretation.

Human readability is desirable.

Machine correctness is mandatory.

The runtime should always prefer correctness, consistency, and determinism over syntactic convenience.

---

# 2. Runtime First

The runtime owns application realization.

Platform builders describe intent.

The runtime determines implementation.

Application behavior belongs to the runtime.

---

# 3. Metadata First

Applications are defined by metadata.

Application source code implements the runtime.

Metadata implements applications.

Applications should evolve by changing metadata rather than application source code.

---

# 4. Runtime Interpretation

Applications are interpreted.

Applications are not generated.

The runtime continuously interprets metadata into executable application behavior.

---

# 5. Declarative

Platform builders describe what should happen.

The runtime determines how it happens.

Implementation details should remain inside the runtime.

---

# 6. Platform Independence

Business Knowledge should remain independent from:

- programming languages,
- rendering technologies,
- databases,
- infrastructure,
- deployment models.

Technology serves Business Knowledge.

Never the opposite.

---

# 7. Separation of Responsibilities

Business Knowledge and Runtime Language have different responsibilities.

Menata Language explains business.

Runtime Language explains realization.

Neither should replace the other.

---

# 8. Convention over Configuration

Reasonable defaults should require little or no configuration.

Configuration should only exist where application intent cannot be inferred safely.

---

# 9. Infer Before Configure

The runtime should infer as much as possible.

Explicit configuration should only describe exceptions.

Inference reduces duplication while preserving explicit intent where necessary.

---

# 10. Composable

Applications should be assembled from reusable metadata.

Objects.

Views.

Events.

Constraints.

Permissions.

Services.

Pages.

Everything should remain composable.

---

# 11. Reference over Duplication

Relationships should be expressed through references.

Business Knowledge should have a single source of truth.

Duplicated metadata should be avoided whenever possible.

---

# 12. Workspace Isolation

Workspace is the primary execution boundary.

Applications belong to workspaces.

Isolation should apply to:

- ownership,
- visibility,
- governance,
- security,
- deployment.

Cross-workspace interaction should always be explicit.

---

# 13. Live Evolution

Applications should evolve continuously.

Changing metadata changes application behavior.

Application evolution should not require application regeneration.

---

# 14. Data Preservation

Business data is more valuable than runtime metadata.

Metadata may evolve.

Runtime may evolve.

Applications may evolve.

Business data should remain preserved.

Potentially destructive changes should always require explicit migration decisions.

---

# 15. Technology Adaptable

The runtime should freely adopt better technologies.

Renderers may change.

Databases may change.

Programming languages may change.

Infrastructure may change.

Business Knowledge should remain stable.

---

# 16. AI Native

Runtime metadata should be understandable by both runtime engines and AI.

AI should assist platform builders in:

- creating metadata,
- evolving metadata,
- optimizing applications,
- validating runtime configurations.

---

# 17. Open Platform

The runtime should remain extensible.

New capabilities should be added through extension.

Core architecture should remain stable.

---

# 18. Long-term Compatibility

Business Knowledge should survive multiple runtime generations.

Backward compatibility should be preferred whenever reasonably possible.

Organizational knowledge should outlive runtime implementation.

---

# 19. Single Runtime

A single runtime should be capable of realizing one application or thousands of independent applications.

Applications are isolated by metadata.

Not by runtime instances.

---

# 20. Applications Evolve at the Pace of Business Knowledge

Business evolves.

Business Knowledge evolves.

Platform builders refine application intent.

The runtime realizes those changes.

Applications should evolve at the pace of Business Knowledge.

---

# Summary

Menata Runtime is built around five fundamental beliefs.

- Machine First
- Runtime First
- Metadata First
- AI Native
- Applications evolve at the pace of Business Knowledge

Everything else is implementation.
