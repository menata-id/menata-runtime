# Gap Analysis and Recommendations — Reconciling the Symfony Essay

> Part of Study 39 (`../README.md`). The owner supplied an essay (an AI session's own analysis of
> `app/` against Symfony's architecture, twelve numbered gap areas, a priority table, and a closing
> "combine both strengths" architecture sketch) and asked which parts are correct, which duplicate
> ground Study 37 (ObjectStack) already covered, and which should actually feed the roadmap.
> This document is that reconciliation, verified against `capability-registry.md`, `capability-
> lifecycle.md` §4, `app/` source, and (for Symfony-side facts) `symfony.com/doc` and
> `github.com/symfony/symfony`.
> Status: v1.0 | Created: 2026-09-07 | Updated: 2026-09-07

---

## 1. Framing

Study 37's framing question was "does a case need it, can it be said in business language, can it
be built as metadata the existing Go runtime interprets, and does it stay cheap at the data volumes
Study 8 measured?" That question still applies here, with one addition specific to comparing
against a general-purpose framework rather than a peer metadata runtime: **Symfony solves problems
that come from being a library used by thousands of unrelated applications and third-party
bundles inside one dynamically-typed runtime.** Menata Runtime is one application, one codebase,
one compiled Go binary, with no third-party extension ecosystem to serve. Several of Symfony's
most celebrated primitives (the DI container, the Bundle system, event-listener priority
ordering) exist specifically to make *that* problem tractable. Importing the primitive without
importing the problem it solves is exactly the "declared-but-unenforced complexity" failure mode
Study 37 spent 31 gaps and 6 rejected-architecture rows warning against — so each of the twelve
essay points below is checked against two questions in order: *is this gap real*, and *if real, is
a Symfony-shaped mechanism the cheapest way to close it, or does Menata already close it another
way.*

## 2. Verdict table

One row per essay section (essay's own numbering, §1–§12). "Registry state" is the row as it
stands in `capability-registry.md` today, not as of any earlier study snapshot.

| Essay § | Claim | Verdict | Registry state / evidence |
|---|---|---|---|
| §1 | Composable View is a Menata strength Symfony lacks | **Correct, already independently proven in more depth** | Study 38 (`benchmarks/029`) derived this from 14 real mockups + 2 proof mockups, not assertion; it also already rejected the SPA-shaped Component/Slot *ontology* the second ObjectStack reviewer separately proposed (`objectstack/docs/composable-view-proposal-reconciliation.md`). No action — confirmation, not a new finding |
| §2 | DI / Service Container is the #1 gap | **Real problem the essay names (extension/wiring), wrong mechanism** | See §3.1 below |
| §3 | Generic application Event system (`kernel.*`-style) | **The effect already exists via a different, cheaper grammar** | See §3.2 |
| §4 | HTTP Kernel / middleware lifecycle | **Partially real (one named gap), mostly already met idiomatically** | See §3.3 |
| §5 | Validation as a generic runtime service | **Already built** | `CAP-C13` ✅ (expression operator, CEL-shaped, usable in constraint/event conditions today) |
| §6 | Serialization / API / OpenAPI | **Already registered, not yet built** | `CAP-X07` row's own "Study 37 note" — Tier 2 (PUT/PATCH/DELETE, filters, perform-Event endpoint, OpenAPI) proposed by Study 37's R26, still open |
| §7 | Form abstraction (metadata → HTML + API + validation + transform) | **Already built, essay describes it without recognizing it** | Field (`CAP-F*`) + Form View (`CAP-V01`) + Constraint (`CAP-C*`) triple already is this; see §3.4 |
| §8 | Security policy engine (Voters, ABAC-style conditions) | **Already registered as three separate, already-admitted rows** | `CAP-P08` (scope depth), `CAP-P09` (approver resolvers), `CAP-O12` (unit tree) — all ❌ Proposed since Study 37; `CAP-C13` (✅) is the expression mechanism a Voter-style condition would use. CSRF + password hashing already ✅ (`CAP-X02`) |
| §9 | Three-tier cache (metadata / view-compile / data) | **Two-thirds already tracked, one-third a false premise** | `CAP-X04` (metadata reload) ⚠️, `CAP-X11` (multi-process cache) ❌ deferred by design (Study 23); "view compilation cache" doesn't apply — `templ` compiles to Go at build time, there is no per-request template-compile cost in this architecture to cache |
| §10 | Plugin / Bundle / Capability-Package system | **The reuse problem is real; already has a lighter answer than a plugin kernel** | See §3.1 (same rejection as ObjectStack's plugin kernel) and `CAP-X08` ⚠️ (metadata package export/import — the actual Menata answer to "a distributable business capability"), `CAP-V28` ❌ (category-keyed template, the closest analog to a pre-built capability bundle) |
| §11 | Observability / tracing | **Already registered, not yet built** | `CAP-I04` ⚠️ — Study 37's R12 (trace View over one correlation id) already proposed exactly this |
| §12 | Don't chase Symfony feature parity; use it as inspiration for the infra layer under the metadata layer | **Correct, and already the standing discipline** | Same conclusion as Study 37 §4/§5 and `capability-lifecycle.md` §4, independently re-derived a third time |

## 3. The four points that need real explanation

### 3.1 DI/Service Container (§2) and Plugin/Bundle system (§10) — same rejection, doubled

The essay is right that `app/` has no dependency-injection container and no plugin/bundle
mechanism. It is wrong that this is a gap Menata should close by building one, for a reason
already on record: **Study 37 §2 rejected this exact shape for ObjectStack's own microkernel +
runtime plugin DI**, verbatim:

> "A runtime plugin bus on a single Go binary with no third-party plugin ecosystem buys
> plugin-order, signature, health and DI machinery … for nothing a case needs."
> — `prototype/objectstack/docs/second-opinion-reconciliation.md` §2

Symfony's DI container is the same kind of machinery solving a *harder* version of the same
problem: not just "one app has many internal services" but "thousands of unrelated applications
each combine a different set of independently-versioned third-party bundles inside one PHP
process, and the container has to autowire and configure whichever combination shows up." Menata
Runtime has exactly one application, one codebase, compiled as one static binary — there is no
combinatorial wiring problem for a container to solve. `capability-lifecycle.md` §4 names the
actual target extension architecture for this codebase (registry seams — `fieldtype.Register`,
`action.Register`, `operator.Register`, `eventsource.Register`, `viewtype.Register`), and is
explicit that **this is a target, not yet the real code** — today it's ordinary Go `switch`
statements in `internal/metadata/loader.go`, `internal/executor/executor.go`, and
`internal/handler/*.go`. That is, if anything, a stronger argument against a DI container than a
weaker one: even the planned extension mechanism is a compile-time table, not a runtime container,
because Go's own type system and the small, fixed number of extension points make a container
solve a problem that doesn't exist here.

The essay's own supporting example — `services: invoice.approver: capability: approve_invoice`,
with the runtime resolving it — is not a DI container use case at all. It's an actor/approver
*resolution* declaration, which is precisely `CAP-P09` (`manager_of`/`unit_of` resolvers on
approver pickers), already registered ❌ from Study 37. The essay independently rediscovered the
need and misattributed the mechanism.

The Bundle/Capability-Package point (§10) gets one more piece of corroborating evidence this
study added specifically: Symfony's own field experience argues against the essay's framing.
Verified against current `symfony.com/doc` (`bundles.html`): since Symfony 4 and the Symfony Flex
tooling, organizing an application's *own* code into bundles is explicitly not recommended —
bundles now exist only to share code *across* multiple applications, and ordinary app code lives
directly in `src/` with plain service configuration. Symfony's own maintainers moved away from
"everything is a bundle" for exactly the reason Menata should not adopt it: a bundle/plugin
system is overhead when there's no ecosystem of independently-authored, cross-application
packages to manage. What Menata actually needs from §10 — a way to package and reuse a
business capability — already exists at the metadata layer, which is the right layer for it: `CAP-
X08` (metadata package export/import, ⚠️ built with one named limitation around compiler-generated
Process Overlay content) and `CAP-V28` (❌ proposed — a category-keyed saved configuration
template, the closest real analog to "install the Document Approval package"). Both are lighter
than a Go-level plugin interface because the unit being packaged is metadata, not code — exactly
the Metadata First principle a plugin kernel would cut against.

### 3.2 Generic application event system (§3)

The essay proposes a `BeforeCreate`/`AfterCreate`/`BeforeAction`/`AfterAction` dispatcher with
metadata like:

```yaml
events:
  invoice.created:
    handlers:
      - update_accounting
      - notify_approver
      - create_activity
```

This is not a new capability — it is, field for field, the Event → Action grammar Menata already
has: `CAP-E01` (business activity event), `CAP-E05` (internal/system-triggered event), and the
existing action types (`CAP-A06` `create_record` in another Machine, `CAP-A07`/`CAP-A08`
`aggregate_status`, `CAP-A10` in-app notification). "When Invoice is Created, do
`update_accounting`, `notify_approver`, `create_activity`" is precisely the shape
`Machine.Events[].Actions[]` already declares, phrased as business language rather than a
listener registration. The one real difference is that Symfony's `kernel.*` events are
*framework-lifecycle* events (a request arrived, a controller was resolved, a response is about to
be sent) — infrastructure-level hooks with no business meaning — which is a different axis than
Menata's business events entirely, and one with no case pressure: nothing in `case-portfolio.md`
needs code to run "before every HTTP response is sent" in a way the existing chi middleware chain
(session auth, CSRF, workspace-slug resolution — see §3.3) doesn't already cover.

### 3.3 HTTP Kernel / middleware lifecycle (§4)

`app/`'s router (`internal/router`) already runs a middleware chain (`RequireWorkspaceSlug`,
session/CSRF checks, request-ID tagging feeding `CAP-I04`) using chi's ordinary middleware
pattern — this is the idiomatic Go equivalent of Symfony's kernel-event pipeline, minus the
event-object indirection Symfony needs because *it* has to let arbitrary third-party bundles
insert themselves into the pipeline at arbitrary priorities. Menata's middleware chain is a fixed,
short, hand-ordered list in one file, because there is no third party inserting anything into it.
Building a generic, metadata-declarable middleware/hook pipeline on top of that would be
solving Symfony's bundle-composition problem for a codebase that doesn't have it.

The one concrete, already-named gap survives scrutiny: `app/ARCHITECTURE.md` already lists **"No
rate-limiting/DoS-protection middleware"** as a gap, independent of this essay. Symfony's own
`RateLimiter` component (token-bucket/sliding-window, core since Symfony 5.2) is real industry
confirmation that this is a standard, proven primitive — worth naming as a connection this study
adds: `CAP-P07` (public/unauthenticated access, ✅, Case 13's blog `Visitor` role) is exactly the
surface a rate limiter protects, and today it has none. This is not proposed as an admitted
capability here — no case has exercised it under real anonymous traffic yet, per
`capability-lifecycle.md`'s own A1 evidence bar — but it is a real, specific, previously-unlinked
terrain fact worth recording for whenever that admission test is run.

### 3.4 Form abstraction (§7)

The essay's own recommended target architecture for Menata is:

```
Metadata Form
      ↓
Form Runtime
      ├── UI Component
      ├── Validation
      ├── Data Binding
      ├── Transformation
      └── Action
```

...explicitly proposed as an alternative to adopting Symfony's Form component wholesale. This
already exists: `CAP-V01` (Form View) renders from `CAP-F*` Field metadata, `CAP-C*` Constraints
validate the same fields the form declares, and Create/Update actions bind submitted values back
onto the record — one metadata declaration driving UI, validation, and persistence, with no
separate DTO/transformer layer, which is the same "metadata source of truth" principle the essay
argues for. There is no gap here to close; the essay reconstructed Menata's own existing design
from first principles without checking that it already exists in `app/`.

## 4. Where Symfony is genuinely ahead and Menata should not close the gap

Consistent with Study 37 §5's "what not to copy, and why" — the same discipline applies here:

| Symfony strength | Why Menata doesn't need the equivalent |
|---|---|
| DependencyInjection container with autowiring | No multi-package wiring problem exists in a single static binary (§3.1) |
| Bundle system | Symfony's own current guidance is *against* using this for application code (§3.1); Menata's metadata packaging (`CAP-X08`) is the lighter equivalent for the actual reuse need |
| `kernel.*` event pipeline with listener priorities | Fixed, short, hand-ordered middleware chain already covers it; no third-party bundles compete for insertion order (§3.3) |
| Form component (Type/DataMapper/Transformer/Event/submission lifecycle) | Metadata Field+View+Constraint triple already gives the same end-to-end result with one declaration instead of five collaborating classes (§3.4) |
| Web Profiler / debug toolbar | Dev-only tooling with no case pressure; same "no speculative substrate" call Study 37 §5 made for a realtime bus "from the start" |
| Console component (CLI commands) | No case names an operator CLI need beyond the existing `goose` migrations and admin HTTP endpoints |

## 5. What this study adds to the roadmap

**No new capability ID.** Every real gap the essay surfaces already has a registry row from Study
37/38, at one of these statuses: `CAP-C13` ✅ built; `CAP-X07` Tier 2, `CAP-P08`, `CAP-P09`,
`CAP-O12`, `CAP-X17`, `CAP-I04` all ❌/⚠️ registered, awaiting build order. This study's
contribution is convergent evidence (a third independent source, after ObjectStack and the
second-opinion review, landing on the same short list from a completely different technology
family) and two specific corrections/connections not previously on record:

1. The Symfony Bundle/Flex field lesson (§3.1) as additional evidence *against* ever building a
   Go-level plugin kernel here, beyond ObjectStack's own architecture cost argument.
2. The `CAP-P07` ↔ rate-limiting connection (§3.3) — naming the terrain that would satisfy A1
   evidence for a future rate-limiting capability, not proposing one yet.

**Recommendation for `roadmap.md`:** record this study as a data point supporting the existing
Study 37/second-opinion priority order (`prototype/objectstack/docs/second-opinion-
reconciliation.md` §5) rather than adding a new order of its own — nothing here changes that
sequencing. Suggested single addition to that order's own list: when `CAP-P08`/`CAP-P09` come up
for build, confirm `CAP-C13` expressions are usable inside Permission conditions specifically, not
only Constraint/Event conditions — the essay's Voter-style example (`if: all: [user.role ==
approver, document.status == submitted]`) is exactly what `CAP-P08`/`CAP-P09` need and `CAP-C13`'s
own row doesn't explicitly say Permission is one of its consuming surfaces yet.
