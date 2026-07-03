# 001. Design Principles

> Design Principles define the architectural philosophy of Menata Runtime.
>
> They guide every architectural decision while remaining independent from implementation technologies.
>
> As technologies evolve, these principles should remain stable.

---

# Core Principles

## 1. Machine First

Menata Runtime is designed primarily for deterministic machine interpretation.

Human readability is important.

Machine correctness is mandatory.

Runtime Metadata should always prioritize consistency, determinism, and correctness over human convenience.

---

## 2. Runtime First

The runtime owns application realization.

Metadata describes application intent.

The runtime determines how that intent is realized.

Application behavior belongs to the runtime.

---

## 3. Metadata First

Applications are defined by Runtime Metadata.

Application source code implements the runtime.

Runtime Metadata implements applications.

Application evolution should primarily occur by changing Runtime Metadata rather than application source code.

---

## 4. Declarative

Runtime Metadata describes **what** applications should become.

The runtime determines **how** applications are realized.

Implementation details belong to the runtime.

---

# Architecture Principles

## 5. Convention over Configuration

The runtime should provide intelligent defaults.

Configuration should only exist where application intent cannot be inferred safely.

Simple applications should require minimal configuration.

---

## 6. Infer Before Configure

Inference is preferred over explicit configuration.

Whenever application behavior can be inferred safely, explicit metadata should not be required.

Configuration should primarily describe exceptions.

---

## 7. Composable

Applications should be assembled from reusable metadata.

Objects.

Views.

Events.

Constraints.

Permissions.

Services.

Pages.

Each component should remain independently reusable.

---

## 8. Reference over Duplication

Relationships should be expressed through references.

Business Knowledge should have a single source of truth.

Duplicated metadata should be avoided whenever possible.

---

## 9. Workspace Isolation

Workspace is the primary execution boundary.

Applications belong to workspaces.

Isolation applies to:

- ownership,
- visibility,
- governance,
- security,
- deployment.

Cross-workspace interaction should always be explicit.

---

# Evolution Principles

## 10. Live Evolution

Applications should evolve continuously.

Changing Runtime Metadata changes application behavior.

Application evolution should not require application regeneration.

---

## 11. Data Preservation

Business data is more valuable than Runtime Metadata.

Runtime Metadata may evolve.

Applications may evolve.

The runtime may evolve.

Business data should remain preserved.

Potentially destructive changes should always require explicit migration decisions.

---

## 12. Long-term Compatibility

Business Knowledge should survive multiple runtime generations.

Runtime evolution should preserve compatibility whenever reasonably possible.

Organizations should not lose Business Knowledge because runtime implementation evolves.

---

## 13. Technology Adaptable

Implementation technologies will evolve.

Programming languages may change.

Rendering engines may change.

Databases may change.

Infrastructure may change.

Business Knowledge should remain stable across technological evolution.

---

# Platform Principles

## 14. Single Runtime

A single runtime should be capable of realizing one application or thousands of independent applications.

Applications are isolated by Runtime Metadata.

Not by runtime instances.

---

## 15. Open Platform

Menata Runtime should remain extensible.

Additional capabilities should be introduced through extension rather than modification.

The core runtime should remain stable.

---

## 16. Compatible Authoring

Runtime Metadata should be implementation independent.

Menata Apps Builder is the reference authoring tool.

However, Runtime Metadata may be produced by:

- Menata Apps Builder,
- visual builders,
- command-line tools,
- manual editors,
- compatible third-party builders,
- any implementation that follows the Runtime Language specification.

The runtime only interprets Runtime Metadata.

The runtime never depends on how Runtime Metadata was created.

---

# Vision

Applications should evolve at the pace of Business Knowledge.

Business Knowledge evolves.

Runtime Metadata evolves.

The runtime evolves.

Applications continuously evolve.

Business Knowledge remains the long-term organizational asset.

---

# Summary

Menata Runtime is built upon four fundamental beliefs.

- Machine First
- Runtime First
- Metadata First
- Declarative

Everything else supports these four principles.
