# Composable Runtime Architecture — Transformation Blueprint

> Status: v1.0 — first pass. Assembles `prototype/objectstack/docs/composable-view-proposal-
> reconciliation.md` §1–§10 (four external/owner passes on a "Composable Metadata Runtime"
> direction, 2026-09-07 → 2026-09-11), `capability-registry.md` (`CAP-V10`, `CAP-V22`, `CAP-C13`,
> `CAP-X10`), and `benchmarks/004-scale-architecture-study.md` (Study 8) into one target
> architecture and a phased, evidence-gated evolution plan | Created: 2026-09-11 |
> Updated: 2026-09-11

> **What this document is.** A synthesis and a sequencing plan, requested directly by the owner
> after five rounds of external/owner review converged on the same underlying architecture
> question: today's composition unit is `View`, and the runtime should evolve toward composing
> Data, UI, and Behavior independently. It organizes gaps already found and evidenced elsewhere in
> this repo into one coherent target shape, so a future session doesn't have to re-derive the
> relationships between `CAP-V10`, `CAP-V22`, `CAP-C13`, `CAP-X10`, and Study 8 from scratch.
>
> **What this document is not.** A capability admission. Nothing here is built by this document,
> and nothing here skips `capability-lifecycle.md` §2's A1–A5 test or this repo's own "declare
> targets first" discipline (`roadmap.md`, `capability-registry.md` passim). Every phase below
> names its own forcing condition — a real case, or a direct owner decision of the same
> evidentiary class already accepted for `CAP-V10 Tier 2`/`CAP-F24`/`CAP-V28`. Where no such
> condition exists yet, the phase is recorded as sequencing information only, not a green light to
> build ahead of need.

---

## 1. Where this comes from

| Source | What it contributed |
|---|---|
| `prototype/objectstack/docs/composable-view-proposal-reconciliation.md` §1–§4 | Settled that a dynamic client-side Component Registry/Canvas is SPA-shaped machinery this server-rendered runtime has no structural need for — confirmed again in every later pass, not re-litigated here |
| Same doc, §5, §7, §8 | `CAP-V10 Tier 2` (page composing Views + closed static content + one layout shape) scoped, then implemented and shipped 2026-09-09; recursive nesting depth and parent→child context named as real, deliberately deferred gaps |
| Same doc, §9 | A fourth, broader restatement checked against the shipped `CAP-V10 Tier 2` — confirmed most claims already closed, named three genuinely new framings: a Query/Projection layer independent of any one View, a UI Intermediate Representation, and composability-measuring benchmark KPIs |
| Same doc, §10 | The owner's own synthesis: their earlier verbal Dataset/Query-first framing merged with §9's UI-first framing into three composability planes (Data/UI/Behavior), `View` reframed as a composition preset rather than a primitive, and a request for this document |
| `capability-registry.md` `CAP-V22` | Semantic dataset (named aggregate reused by name across `report`/`dashboard`/chart Views) — already registered ❌ Proposed, Study 37 R2, "metric drift" the exact problem a Data plane exists to solve |
| `capability-registry.md` `CAP-C13` | Expression operator (CEL) — already ✅, the general compute/transform primitive a Projection layer would reuse rather than duplicate |
| `capability-registry.md` `CAP-X10` | Metadata-driven index management — already ❌, deliberately deferred (no measured scale pressure yet), per "Infer Before Configure" |
| `benchmarks/004-scale-architecture-study.md` (Study 8) | The existing scale/performance architecture study — index strategy, cache strategy, P95 targets — the Data-plane framing's own performance argument connects directly to this, not a new concern |
| `internal/metadata/compile.go` (Study 21, `CAP-W01` Process Overlay) | Proof that "declarative metadata compiles at load time into lower-level runtime primitives" already works in this codebase — the precedent a UI IR would extend, not invent |
| `capability-lifecycle.md` §2 | The existing A1–A5 admission gate, which every phase below still has to pass |

---

## 2. Target architecture

Three composability planes, meeting in one compiler discipline:

```text
                         MENATA RUNTIME
                              │
        ┌─────────────────────┼─────────────────────┐
        ▼                     ▼                     ▼
   DOMAIN PLANE           DATA PLANE           EXPERIENCE PLANE
   (built, stable)        (partial)            (partial)
        │                     │                     │
     Machine               Dataset                Page
     Field                 Query                   Layout
     Event                 Projection              Component
     Constraint            Expression (CAP-C13 ✅)  Binding
     Permission            Relation                Context
        │                     │                     │
        └─────────────────────┼─────────────────────┘
                              ▼
                    RUNTIME COMPILER / NORMALIZER
                              │
                   ┌──────────┴──────────┐
                   ▼                     ▼
             EXECUTION PLAN            UI IR
             (Query Planner /     (compile-time dispatch
              Cost Model,          today; no formal IR)
              → Postgres)                │
                   │                     ▼
                   ▼                 RENDERER
               POSTGRES            (Templ/HTMX, one
                                    target by design)
```

A transversal layer applies to all three planes: **Compiler / Validation / Security / Lifecycle**
— this already exists (`internal/metadata`, `capability-lifecycle.md`, the conformance ratchet) and
needs no new concept, only for new plane concepts to plug into it the same way Fields/Events do
today.

**Reframing `View`, not removing it.** `View` stops being the runtime's only composition unit and
becomes a **named preset** — a fixed, useful combination of Data-plane and Experience-plane pieces,
the same way a `list` View today is really "a Dataset (implicit: the Machine's own records) + a
Projection (implicit: `Columns`) + a Collection component + a table renderer," just not spelled out
as four separable things. This is evolution, not replacement: every existing `ViewType` stays valid
metadata; the planes describe what a `ViewType` is *made of* underneath, which is what stops new
UX needs from defaulting to a new `ViewType`.

---

## 3. Current-state inventory

One row per concept in the target architecture — what exists today, what's missing, and the
evidence behind each cell. This is the load-bearing table: every phase in §5 traces back to one row
here.

| Concept | Plane | Status | Evidence | Gap |
|---|---|---|---|---|
| Machine/Field/Event/Constraint/Permission | Domain | ✅ Built | `runtime-metadata-schema.md`, `app/internal/model/model.go` | None — this plane is the stable core, untouched by this blueprint |
| `View`/`ViewConfig` (query + presentation unified per-View) | Data + Experience | ✅ Built, but the *only* composition unit | `app/internal/model/model.go:621-786` | Reuse only happens by referencing a whole View by id, never a bare data shape (see Dataset row) |
| `page` composition (`Children`: `{view: id}` + closed static content) | Experience | ✅ Built (`CAP-V10 Tier 2`, 2026-09-09) | `capability-registry.md` `CAP-V10 Tier 2` row | Single-level only — a child cannot itself be a composing `page` (deliberately out of scope) |
| Layout (one shape: `main`/`aside` 2/3+1/3 grid pairing) | Experience | 🟡 Partial | `CAP-V10 Tier 2` row (`Layout` field); `gap-analysis-and-recommendations.md` G22/R19/R20 (form sections/columns, low priority) | No closed vocabulary yet (stack/grid/split/tabs) — one instance shipped, not a general primitive |
| Static content vocabulary (`heading`/`text`/`button`/`image`) | Experience | ✅ Built | `CAP-V10 Tier 2` row (`PageContent`) | Closed set by design — extending it is a Go change + registry-seam entry, never a declared arbitrary `type` |
| Compute/transform expression (CEL) | Data | ✅ Built | `capability-registry.md` `CAP-C13` | None — already general-purpose, usable in filters, computed Fields, constraints |
| Semantic dataset (named, reusable aggregate) | Data | ❌ Proposed | `capability-registry.md` `CAP-V22`, Study 37 R2 | No forcing case yet in `case-portfolio.md`; `roadmap.md` item 24 step 4 ("Analytics trio") already names Cases 9/15 as the candidate proof cases |
| Bare, reusable row-level Query/Projection (independent of any one View) | Data | ❌ Not designed | `composable-view-proposal-reconciliation.md` §9.2(a) | Genuinely new framing — narrower than a full "Dataset," would need its own admission pass, no case names it yet |
| Parent→child context/scope propagation | Experience | ❌ Not designed, but scoped | §5, §8(iii) above | Named mandatory the moment `CAP-V10 Tier 2`'s recursion or any parent-scoped child View is asked for; today's only dynamic token is `$current_user` (`CAP-V05`) |
| Dynamic Component Registry / Canvas | Experience | ⛔ Rejected, structurally | §4 above | Not a gap — the problem it solves (client-side runtime dispatch of arbitrary metadata) doesn't exist in a server-rendered, compile-time-dispatched runtime |
| UI Intermediate Representation / compile step | Experience | ❌ Not designed | §9.2(b) above; precedent exists for the *pattern* in `CAP-W01` (Process Overlay) | No forcing case — the concrete problem a UI IR solves (one representation, many renderers) doesn't apply while `app/ARCHITECTURE.md` commits to exactly one renderer |
| Query Planner / Cost Model / physical plan | Data | 🟡 Partial | Study 8 (index strategy, cache strategy, P95 targets); `CAP-X10` (❌, deliberately deferred, no scale pressure yet) | Study 8 already covers this ground for the Domain/Data plane generally — a Dataset-aware projection pushdown (§6 below) is additive to it, not a competing mechanism |
| Recursive `page`-composing-`page` | Experience | ⛔ Explicitly out of scope | `CAP-V10 Tier 2` row, "Explicitly still out of scope" | Deliberate deferral, not an oversight — revisit only with a real forcing case |

---

## 4. Component/primitive admission — one gate, not two

The owner's synthesis (§10 above) proposed a three-step gate: *can an existing primitive compose
it → can a new generic primitive compose it → only then is it a genuinely new capability.* Checked
against what already governs this registry:

| Synthesis's step | Already covered by |
|---|---|
| 1. Can an existing primitive compose it? | `capability-lifecycle.md` A4 — **Non-composability**: "Cannot be built by composing existing supported capabilities" (Study 5's own ADR-0012 Pattern A/B precedent) |
| 2. Can a new *generic* primitive compose it, rather than a business-specific one? | A3 (**Single responsibility within Grammar**) + A5 (**Business language exists**) together already push toward generic, reusable shapes — this is the same discipline `guides/breaking-down-ui-components-for-metadata.md` already applies at the presentation-primitive level (`RecordSummaryCard`, not `ApprovalListCard`) |
| 3. Is it truly semantically unique? | A1 (**Dual evidence**) + A2 (**Universality or declared verticality**) |

**Conclusion: no new gate is needed.** `capability-lifecycle.md` §2's existing A1–A5 test already
enforces exactly what the synthesis asks for. What *is* new, and worth naming explicitly: Data-
plane concepts (Dataset, Query, Projection) don't yet have a settled Grammar-area home. A3 currently
lists `Field/Event/Action/Constraint/Permission/View` as the Grammar areas a capability must map to
one of. Before `CAP-V22` (or any Data-plane candidate) is scoped past its current one-line registry
entry, an explicit owner decision is needed: does "Dataset" become a new Grammar area (a `D`
prefix, alongside `F/E/A/C/P/V/R/X/I/O`), or does it fold under `View` the way `ViewConfig` already
holds query fields today? This decision has no forcing case of its own — it only matters once
`CAP-V22` or a Query/Projection candidate is actually scoped for admission (Phase 1 below) — so it
is named here as a prerequisite, not resolved.

---

## 5. Phased evolution plan

Sequenced by dependency and risk, not by top-down architectural preference. **Every phase's own
"Forcing condition" column is the actual gate** — a phase does not start building until its
condition is met, exactly the discipline that already governed `CAP-V10 Tier 2`, `CAP-F24`, and
`CAP-V28`.

| Phase | What it does | Depends on | Forcing condition | If met today? |
|---|---|---|---|---|
| **0. Grammar-area decision** | Owner decides whether Dataset is a new Grammar area or folds under View (§4) | Nothing | A direct owner decision — no case needed, this is a taxonomy question | Open, not yet asked |
| **1. Semantic dataset** | Admit and build `CAP-V22` — named aggregate reused by `report`/`dashboard`/chart Views | Phase 0 | A real case needing the same aggregate shown two ways (candidates already named: Cases 9/15, `roadmap.md` item 24 step 4 "Analytics trio") | Not yet — no case has forced it |
| **2. Bare Query/Projection reuse** | A row-level data shape addressable independently of any View (§9.2(a)) — narrower than Phase 1, would extend it once real | Phase 1 | A case needing the *same rows*, not just the same aggregate, rendered by two different presentations (e.g. a Customer list as both Table and Kanban) | Not yet — no case names this |
| **3. Layout vocabulary** | Extend the one shipped shape (`main`/`aside`) into the closed set already tracked as G22/R19/R20 (stack/grid/columns) | Nothing (independent of Data-plane phases) | `gap-analysis-and-recommendations.md`'s own R20 recommendation: "bundle into a design pass" — next time a Form/composed-page mockup needs multi-column, do it as one pass, not per-case | Low priority, unforced, but already queued in `roadmap.md` |
| **4. Context/scope propagation** | Parent→child token(s) (`$context.*`-shaped) so a composed page's child View can scope itself to the page's current record/filter | `CAP-V10 Tier 2` (built) | Already named **mandatory, not deferrable**, the moment a real case needs a parent-scoped child View (§8(iii)) — not before | Not yet — `approval-dashboard.html` shipped without needing it |
| **5. Recursive composition depth** | A `page`'s own `{view: id}` child may itself be a composing `page` | `CAP-V10 Tier 2`, Phase 4 | A real case needing genuine multi-level nesting — explicitly out of scope until then (`CAP-V10 Tier 2` row) | Not yet, deliberately |
| **6. Query Planner / projection pushdown** | Dataset-aware column selection (only fetch the JSONB keys a bound Component actually needs) and index/cache hints derived from declared Filter/Sort/dimension usage | Phase 1/2 | Measured query cost pressure at real data volumes — the same trigger already governing `CAP-X10`'s own deferral; do together, not as two separate scale passes | Not yet — Study 8's own scale target (100 workspaces × 50 machines × 1M records) isn't reached |
| **7. UI Intermediate Representation** | A formal compile step for UI metadata (`Normalize → Resolve → Compile → Render Plan`), generalizing the Process Overlay's own compile-at-load-time pattern to UI | Phases 3–5 (needs real layout/composition variety to be worth formalizing) | A second rendering target (mobile, a visual builder) actually being pursued — `app/ARCHITECTURE.md`'s current one-renderer policy means this has no forcing condition today | Not met — explicitly against current architecture policy |
| **8. Composability benchmark** | A new study measuring Application Construction Ratio, Composition Reuse Ratio, View/Data Independence, Runtime Extension Cost, Query Reuse Ratio, Query Efficiency, Composition Execution Cost (synthesis of §9.2(c) above and the owner's own additions) | Enough of Phases 1–4 built to have something to measure | Scheduled the normal way, via `roadmap.md`'s "Recommended order for upcoming sessions", once Phases 1 and 3 have real code to benchmark | Not yet — nothing to measure until Phase 1/3 land |

**Reading order, if the owner wants to start now:** Phase 0 (a decision, not work) can happen
immediately since it blocks nothing else from being *designed*. Phases 1 and 3 are independent of
each other and can proceed whenever their own forcing conditions are met — Phase 3 already has a
queued trigger (`roadmap.md` R20), Phase 1 needs Cases 9/15 to actually need shared aggregates.
Phases 2, 4–8 are each gated by an earlier phase's completion plus their own real case, in the order
listed.

---

## 6. What does *not* change

Consistent with `composable-view-proposal-reconciliation.md` §4's own structural argument, reaffirmed by every subsequent pass (§9.1, §10):

- **No dynamic Component Registry, no Canvas.** The problem they solve (client-side runtime
  dispatch of arbitrary metadata) does not exist in this server-rendered, compile-time-dispatched
  runtime. Extension stays "add Go code + a registry-seam entry + recompile," per
  `capability-lifecycle.md` §4 — the same mechanism that already rejected ObjectStack's plugin
  kernel (`second-opinion-reconciliation.md` §2).
- **`View`, `Machine`, `Page`, and the capability registry are not removed or restructured.** The
  three-plane model in §2 describes what these concepts are made of, not a replacement ontology.
- **No phase here overrides `capability-lifecycle.md`'s admission gate or this repo's "declare
  targets first" discipline.** A phase's own forcing condition is the same kind of evidence
  (`case-portfolio.md` terrain, or a direct owner decision of the class already accepted for
  `CAP-V10 Tier 2`/`CAP-F24`/`CAP-V28`) this registry has always required.

---

## 7. Disposition

**No capability admitted, no registry row status changed, no code changed by this document.** This
is a planning artifact — each phase in §5 still needs its own A1–A5 pass (`capability-
lifecycle.md`) or direct owner decision before anything is built. What changed procedurally:
this document created; `roadmap.md`'s Study 37 log gets a dated addendum pointing here;
`capability-registry.md`'s status header gets a pointer; `README.md`'s Tier 3 table gets a row for
this document.
