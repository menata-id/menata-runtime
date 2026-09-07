# Menata Runtime — Symfony Comparator Study

> Status: v1.0 — Study 39, first pass: an owner-supplied essay (an AI session's own comparison of
> `app/` against Symfony's architecture) reconciled against Study 37/38's already-settled findings
> and the live `capability-registry.md`, with independent verification of the Symfony-side claims
> (`symfony.com/doc`, `symfony/symfony` source). No row admitted, no code changed.
> Created: 2026-09-07 | Updated: 2026-09-07

---

> **Not a metadata-proof prototype.** Same genre as [`prototype/objectstack/`](../objectstack/)
> (Study 37) — a **comparator study**, not one of the 16-feature metadata-proof scorecards the
> other `prototype/*` folders carry. Placed under `prototype/` rather than `benchmarks/` (where
> root [`README.md`](../../README.md)'s "Where does a new document go?" rule would otherwise put
> it) at the owner's own suggestion, mirroring the ObjectStack precedent exactly. Root `README.md`
> Tier 4, [`prototype/README.md`](../README.md), and `roadmap.md` Study 39 all point here.

## Why Symfony, and why this is a different kind of comparison than ObjectStack

ObjectStack (Study 37) is a peer in the same genre as Menata Runtime — a metadata-interpreting
business-app runtime — so that study compared *capability coverage* area by area (field types,
views, workflow, approvals, permissions, analytics) and found Menata ahead on the metadata-first
identity and behind on a specific, named list of business-capability gaps.

Symfony is not a peer in that sense. It is a general-purpose PHP web framework with no metadata
layer, no business-application ontology, and no opinion about Fields/Machines/Events at all — a
developer writes application code on top of it. So the honest comparison here is not "does
Symfony have the capability" (it usually doesn't — it has no domain model to have it in) but
*does Symfony's infrastructure layer — the plumbing underneath any web application, metadata-driven
or not — contain a proven primitive that `app/` is missing or has built weaker than it needs to
be.* That is the question the owner-supplied essay (reproduced in full context in the conversation
that produced this study) actually asks, correctly, even though its own framing sometimes drifts
toward "Symfony has X as a concept, so Menata needs an X" without checking whether Menata already
has X's *effect* through a different, cheaper mechanism. Sorting that out — which of the essay's
twelve claims are real infrastructure debt vs. which are solved differently on purpose — is this
study's job.

## Sources studied

| Source | How it was read | Snapshot |
|---|---|---|
| Owner-supplied essay | Read in full from the conversation that requested this study (an AI session's own `app/`-vs-Symfony analysis, twelve numbered gap areas + priority table) | 2026-09-07 |
| `symfony.com/doc` | Fetched directly for two claims that needed current-state verification, not assumed from training: the Bundle/Flex status for application code, and the exact `HttpKernel` event list | `symfony/doc` current + `6.4` |
| `github.com/symfony/symfony` | `KernelEvents.php` at tag `7.4`, cited for the kernel-event names | tag `7.4` |
| Everything else about Symfony (DependencyInjection, Form, Validator, Security/Voter, Cache PSR-6/16, Console) | Not independently re-verified — these are long-stable, canonical component shapes; the essay's descriptions of them are accurate and are treated as given, with one correction (see below) | — |
| Menata Runtime side | `app/` (live codebase — `internal/router`, `internal/handler`, `capability-lifecycle.md` §4, `capability-registry.md`, `app/ARCHITECTURE.md`), `prototype/objectstack/` (Study 37/38, all five documents), `roadmap.md` Studies 37–38 | repo `main`, 2026-09-07 |

The one correction worth flagging up front: the essay treats Symfony's **Bundle system** as a
live, central organizing concept ("Symfony mempunyai konsep bundle/component ecosystem"). Current
Symfony docs say the opposite for application code — since Symfony 4 / Symfony Flex, organizing
*your own* application code into bundles is explicitly **not recommended**; bundles are for
sharing code *across* multiple applications, and a default Flex application keeps everything in
`src/` with plain service configuration. That matters here: Symfony's own field experience is a
data point *against* a bundle-shaped plugin/packaging system being necessary for a single
application's own code — which is exactly Menata Runtime's shape (one application, one codebase,
no third-party bundle ecosystem to serve). See `docs/gap-analysis-and-recommendations.md` §4 (R10
in the essay's own numbering).

## Documents in this study

| Document | Covers |
|---|---|
| [docs/gap-analysis-and-recommendations.md](docs/gap-analysis-and-recommendations.md) | Point-by-point verdict on the essay's twelve gap areas: which are already covered (by name, by a different mechanism, or by an already-registered Study 37/38 candidate), which are correctly identified as real and should feed the roadmap, and which should be declined and why — each grounded in `capability-registry.md` rows, `capability-lifecycle.md` §4, and `app/` source, not the essay's own prose |

## Executive summary

The essay's own top-line conclusion is right, and independently reached before: *Menata Runtime is
already more metadata-native than Symfony (which has no metadata layer at all — that comparison
is closer to Study 37's ObjectStack framing than the essay realizes), and the real question is
runtime-infrastructure maturity underneath the metadata layer, not capability breadth.* That is
also Study 37 §4/§5's conclusion in different words, and Study 38's conclusion about UI
composability specifically.

Once checked against the registry, most of the essay's twelve areas fall into one of three buckets:

1. **Already registered and prioritized** — the essay independently re-derives ground Study 37/38
   already covered and the owner already admitted as candidates: expression/validation
   (`CAP-C13`, ✅ *already built*), API completeness + OpenAPI (`CAP-X07` Tier 2), permission scope
   depth + approver resolution (`CAP-P08`/`CAP-P09`/`CAP-O12`), observability trace
   (`CAP-I04`), metadata versioning (`CAP-X17`). A third independent source converging on the same
   short list is useful evidence, not a new gap.
2. **Already solved by a different, cheaper mechanism than the one the essay names** — general
   application events (already the Event/Action metadata grammar, `CAP-E*`/`CAP-A*`), form
   abstraction (already the Field + Form View + Constraint metadata triple), view-compilation
   caching (based on a false premise — Go's `templ` compiles views to native code at *build* time,
   there is no per-request template-compile cost to cache), metadata packaging (`CAP-X08`, already
   ⚠️ built, partially).
3. **Correctly identified as a real gap, but rejected as a runtime *mechanism*** — a
   Symfony-style Dependency Injection container and a Symfony-style Bundle/plugin kernel are both
   solutions to a problem (wiring hundreds of independently-versioned, dynamically-loaded
   third-party packages inside one long-running PHP process) that a single static Go binary with
   no plugin ecosystem does not have. This is the exact rejection Study 37 §2 already made for
   ObjectStack's own microkernel-plus-DI design, for the same reason, and it applies with even more
   force to Symfony's DI container, which exists to solve a *harder* version of the same problem
   than ObjectStack's.

No genuinely new capability survives this study that Study 37/38 hadn't already found — one
concrete, previously-unstated *connection* is worth recording: the essay's rate-limiting point
lines up with `app/ARCHITECTURE.md`'s own already-named gap ("no rate-limiting/DoS-protection
middleware") and with `CAP-P07`'s public/unauthenticated-access surface, which is exactly the
surface a rate limiter protects — worth an admission-test pass whenever a case exercises `CAP-P07`
under real anonymous traffic. See the gap-analysis document §5 for the full accounting.
