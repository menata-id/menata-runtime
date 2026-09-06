# ADR-001: Graduate `prototype/go`'s codebase as `app/`'s foundation

**Status:** Accepted (blueprint stage) — the full decision record lives outside `app/`, at repo
root: [`benchmarks/026-runtime-graduation-decision.md`](../../../benchmarks/026-runtime-graduation-decision.md)
(Study 34). This ADR restates only what's necessary for someone reading `app/`'s own history
without needing to find that file first, and formalizes the technical shape of the graduation
itself. Numbering restarts at 001 — this is `app/`'s own decision log, not a continuation of
`prototype/go/docs/decisions/`'s numbering.

## Context

Seven platform prototypes (`prototype/{go,drupal,frappe,directus,budibase,salesforce,camunda}`)
were built to discover and prove which capabilities a Menata Runtime needs. One of them,
`prototype/go`, pulled far enough ahead (~90 capabilities ✅, a 219-test conformance suite, an
architecture independently vetted twice — ADR-004 and Study 33, both concluding its switch-based
dispatch and flat package layout are not a problem at this scale) that it stopped being a peer of
the other six. The owner decided (Study 34) that `prototype/`/`benchmarks/` have done their job and
the real application is built going forward in a new top-level folder, `app/`.

The remaining question — graduate `prototype/go`'s own codebase into `app/`, or start `app/` from
zero — was answered by weighing concrete evidence, not the generic shape of either argument:

- Every architectural concern ever seriously raised against `prototype/go` (registry-seam
  dispatch) has been re-examined twice and found not to be a real problem at this scale.
- `prototype/go`'s own "prototype-honest" heuristics (e.g. `displayLabel`'s Name-or-first-text-
  field guess) compensate for a Menata Language grammar gap, not a Go-implementation shortcut — a
  from-scratch rewrite would need them again unchanged.
- This host runs no containers (a real RAM/CPU constraint, not preference), which removes the two
  biggest generic "start fresh" appeals: container-native deployment and multi-instance-first
  design. Neither path gets to use them here.
- Because the architecture is already lean (Study 34 §7's own measurements: clean layered import
  graph, zero cycles, 3–6 files touched per past capability with diffs concentrated in one new
  file each), every remaining production-readiness gap (migrations tooling, storage abstraction,
  secrets, API versioning, rate limiting, test depth) is an *incremental addition* to the existing
  codebase, not a rewrite — starting fresh buys no head start on any of them.

## Decision

Graduate `prototype/go`'s codebase into `app/`, package by package, preserving the measured-clean
layer structure verbatim except where a named restructuring is cheaper to do once during the port
(`internal/metadata`'s split) or where a capability genuinely didn't exist before (`internal/
storage`). See `../../ARCHITECTURE.md` for the full package-by-package blueprint.

## Consequences

- `prototype/go` is not deleted, renamed, or further developed — it stays exactly as it is, the
  historical record of the capability-discovery process, per Study 34's own decision.
- `app/`'s `go.mod` uses a distinct module path (`menata.id/app`, not `prototype/go`'s own
  `menata.id/runtime`) so the two never collide and neither accidentally depends on the other.
- Every gap named in Study 34 §5/`ARCHITECTURE.md`'s own "Named, not solved" table is real,
  scoped, and deferred to a development plan — not silently dropped, not blocking this blueprint
  from existing.
- This decision itself is recorded twice, deliberately: in full at
  `benchmarks/026-runtime-graduation-decision.md` (the process, evidence, and owner's own words),
  and here in `app/` (the technical shape, for anyone reading this codebase's own history without
  needing the repo-root decision record first).
