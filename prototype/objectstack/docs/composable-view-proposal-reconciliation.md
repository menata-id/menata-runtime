# Composable-View Proposal Reconciliation — a Third External Document vs. Study 37's Existing Position

> Part of Study 37 (`../README.md`). On 2026-09-07 the owner brought a third document — an
> unsolicited architectural-direction memo titled "Architectural Direction: Composable Metadata
> Runtime," proposing "Everything is composable, View is the primary reusable composition unit" —
> and asked for it to be re-examined against `app/`'s actual code, then discussed further. This
> document is that reconciliation, in the same spirit as `second-opinion-reconciliation.md` (which
> reconciled an independently-produced second review the same day). Every verdict below was checked
> against `capability-registry.md` v0.58 rows and `app/` source, not taken from the proposal's own
> prose.
> Status: v1.3 — §10 added (2026-09-11): a fifth document, the owner's own synthesis reply to §9,
> merging their earlier (unrecorded, verbal) Data/Query-first proposal with §9's UI-composition
> framing into one three-plane model (Data/UI/Behavior composability) and requesting a dedicated
> blueprint document — produced as `../../../composable-runtime-blueprint.md`. No row admitted, no
> code changed | Previously v1.2 — §9 added (2026-09-11): a fourth external document, a full
> top-to-bottom
> restatement of the same "Composable Metadata Runtime" direction (owner-authored, Bahasa
> Indonesia), reconciled against the repo as it stands *after* `CAP-V10` Tier 2 shipped
> (2026-09-09) — most of its View-composition claims are already settled by §1–§8 or already
> closed by the ship; three framings (Query/Projection as a named layer, a UI Intermediate
> Representation, and a composability-measuring benchmark method) are genuinely new and not
> previously named anywhere in this study. No row admitted, no code changed | Previously v1.1 —
> §8 added (2026-09-09): an owner Q&A session re-read this document and the
> proposal's own asks against live code a second time, found two scoping holes in §7's Tier 2
> wording (recursive nesting depth, composed-page layout) neither §5 nor §7 named, and assessed the
> §5 context-passing question's actual priority. No row admitted, no code changed | Previously v1.0
> | Created: 2026-09-07 | Updated: 2026-09-11

---

## 1. What the proposal asked for, and why it isn't virgin territory

The proposal's own 20 sections ask, in short: make View a universal recursive composition
primitive (data + presentation + components + content + nested Views), demote Page to a thin
route→View resolver, add a Component Registry (capability → schema → validation → renderer) so
arbitrary metadata can safely reach a renderer, and add a Canvas composition surface mixing
content, components, and Views freely. It closes with its own required deliverable shape: (A)
Current Architecture, (B) Gap Analysis, (C) Proposed Architecture, (D) Compatibility Strategy, (E)
Implementation Plan.

This overlaps heavily — not coincidentally, since both are reacting to the same terrain — with
§12 of the second review already reconciled in `second-opinion-reconciliation.md`: *"UI metadata
'far behind' — a Page/Section/Component/Slot ontology."* That verdict, reached the same day against
the same code, was:

> "Rejected in that form; the concrete gaps are already R3/R17–R20. A Component/Slot vocabulary is
> what a client-rendered SPA needs to compose widgets; Menata's runtime renders server-side and
> `app/ARCHITECTURE.md`'s JS policy is a core principle, not a gap. What the review actually
> wants — pages composing several Views — exists for dashboards (`CAP-V10` sections) and the
> honest addition is a Tier 2 on `CAP-V10` ('a `page` composing any Views, not only summary
> tiles'), not a new ontology."

This document does not re-derive that verdict from scratch. It (§2–4) confirms the verdict still
holds against the proposal's fuller framing, then (§5) adds one genuine refinement the existing
`CAP-V10` Tier 2 note left open.

## 2. Current Architecture (evidence)

| Element | State | Evidence |
|---|---|---|
| Metadata hierarchy | `Workspace → Application → Machine → {fields, events, constraints, permissions, views}`. No `pages:` key exists anywhere in the schema | `runtime-metadata-schema.md` |
| Page concept | Designed (`006-runtime-model.md` lists Page as a Machine child) but **never implemented** — no `Page` struct in code | `app/internal/model/model.go` (no `type Page`) |
| View | `View{ID, MachineID, Name, Type, Config}` — every View belongs to exactly one Machine; 12 `ViewType`s | `app/internal/model/model.go:621-688` |
| Query + presentation | Already unified in one `ViewConfig` struct (`Fields`/`Columns`/`Filter` alongside `Steps`/`Template`) | `app/internal/model/model.go:688-786` |
| Dashboard composition | `Sections []DashboardSection{Title, Machine, GroupField}` — each section names a **Machine**, not a View; renders as a permission-scoped record count only | `app/internal/handler/views.go:264-315` |
| Rendering pipeline | One dedicated Go handler + one dedicated `.templ` file per `ViewType` (`list.templ`, `detail.templ`, `dashboard.templ`, `board.templ`, `report.templ`, …) | `app/internal/ui/`, `app/internal/handler/views.go` |
| Routing | Hard-coded `{machineID}/{view-type-suffix}` — a route names a Machine + a View type, never a View by id | `app/internal/router/router.go:123-148` |
| Component model | No registry, no schema/validation layer; field-input widgets are fixed `.templ` functions | `app/internal/ui/components.templ` |
| Client-side model | Server-rendered Templ + HTMX (partial-page swap) + Hyperscript (declarative, non-reactive); no client framework, by explicit policy | `app/ARCHITECTURE.md` §"Client-side JavaScript policy" |

## 3. Gap Analysis, mapped to the proposal's own sections

| Proposal section | Current state | Verdict |
|---|---|---|
| §5 query+presentation in one View | Already true | **No gap** |
| §4/§11 recursive View-in-View, parent→child context | Dashboard sections name a Machine, not a View; `CAP-V06` reverse-reference sub-lists are automatic, not declarative composition; no nesting exists | **Real gap** — see `CAP-V10` Tier 2 below |
| §8 Page as thin context layer over View | Page doesn't exist in code; routing hard-codes Machine+ViewType | **Real gap**, but no proposal to merge Page and View — they stay separate concepts per `006-runtime-model.md`'s own "Runtime Ownership" table |
| §7 Component Registry (capability/schema/validation/renderer) | Doesn't exist; not needed structurally — see §4 | **Confirmed as already-rejected**, reasoning below |
| §9 Canvas / freeform content+component+View mixing | Doesn't exist | **Confirmed as already-rejected**, same reasoning, with one carve-out — see §5 |
| §10 semantic layout (stack/grid/tabs) | Forms are single-column only | **Real gap, low priority, unclaimed** — `gap-analysis-and-recommendations.md` G22, tiered into R19/R20 ("presentation; low") |
| §12 declarative interaction (`on_click`) | Events already are the sole declarative-action mechanism, deliberately not duplicated as a second "Action" concept | **Already satisfied in spirit** — `CAP-P04` Tier 2 note (R29) covers presentation options for triggering an Event, not a new action grammar |
| §6 one data definition, many presentations | Not reusable across Views today | Narrower capability already proposed as `CAP-V22` semantic dataset (❌ Proposed) |

## 4. Why §7 (Component Registry) and §9 (Canvas) are "SPA-shaped," confirmed structurally

A Component Registry (capability → schema → validation → renderer) exists to solve one specific
problem: **a client receiving arbitrary metadata at runtime and needing a safe, dynamic way to
turn a type string into a live UI component without recompiling.** That is the literal situation
ObjectStack's React console is in — a `{type: "button", properties: {...}}` payload has to be
validated and dispatched *in the browser, at runtime*, or a bad payload from the server can crash
or misrender the client. The registry is the guardrail for that.

`app/` does not have that problem, structurally. `internal/ui/*.templ` compiles to ordinary Go
functions at build time. Dispatch from `ViewType` to its renderer is a compiler-checked Go call,
never an interpretation of a metadata string at runtime. There is no client-side renderer
receiving arbitrary component metadata to validate — HTMX only swaps in HTML the server already
rendered and validated at compile time. The problem the Registry solves does not exist here, so
building the machinery to solve it would be importing ObjectStack's constraint, not closing a gap
of Menata's own.

Canvas, as a freeform composition surface, exists to serve the same target: a visual builder
whose output is interpreted by a client-side renderer with open-ended composition (Webflow,
Builder.io, Retool-class tools). With a server-rendered target and no client framework
(`app/ARCHITECTURE.md`'s policy), there is no runtime on the other end that needs a Canvas-shaped
input.

This also settles the Open Platform tension the proposal implicitly raises (§7's registry framed
as enabling extension): Menata's actual extension mechanism is **compile-time registry seams**
(`capability-lifecycle.md` §4 — the field-type/action-type/view-type tables in `internal/model` +
`internal/executor`) — extension means adding Go code and recompiling, the same reasoning that
already rejected ObjectStack's runtime plugin kernel (`second-opinion-reconciliation.md` §2, first
row). A dynamic Component Registry would be a second, redundant extension mechanism solving a
problem the first one doesn't have.

## 5. What this pass adds: two asks the proposal bundled together, only one of which is closed

Re-reading the proposal's own "Mixed View" example —

```yaml
children:
  - component: { type: heading, properties: { text: "Approval Dashboard" } }
  - view: approval_summary
  - component: { type: button, properties: { label: "Create Document" } }
  - view: pending_documents
```

— shows it is actually asking for two different things under one word ("Component"):

**(A) Dynamic runtime component dispatch** — an open-ended type vocabulary resolved and validated
at runtime. This is the SPA-shaped machinery §4 rejects, and the rejection stands.

**(B) A small, closed set of static content primitives (`heading`, `text`, `button`, `image`) mixed
into a page's `children` alongside View references.** This needs no dynamic dispatch at all — each
primitive is one more fixed `.templ` function, the same shape `ViewType`/`ViewConfig` already use
(a fixed enum + a config struct, validated at load time, dispatched by a compiler-checked switch).
It is not a Component Registry; it is a small, bounded extension of the *already-proposed*
`CAP-V10` Tier 2 note itself, which currently only says a page View may compose "any Views" —
it does not yet say whether a small set of static content entries may sit alongside them.

This is a genuine gap in the existing Tier 2 note, not something `second-opinion-reconciliation.md`
§2/§4 considered and declined — worth folding into that note's scope when `CAP-V10` Tier 2 is
formally scoped for admission, appended below (§7).

**A second open question, also not previously closed:** if `CAP-V10` Tier 2 ships (a page View
referencing other Views by id, rendered recursively), the proposal's §11 context-passing need
(`$context.document.id` — a child View scoped to its parent's current record/filter) has no
declared mechanism yet. `CAP-V06`'s reverse-reference sub-lists solve one narrow instance
(child-of-record) automatically; a page-View embedding an arbitrary `pending_documents` list
scoped to "this dashboard's current filters" has no analog. Named here as a sub-item for whoever
scopes `CAP-V10` Tier 2 for real, not a blocker to raise before then.

## 6. Disposition

**No capability admitted, no row changed in status.** Consistent with this study's own discipline
(`gap-analysis-and-recommendations.md` §1, `second-opinion-reconciliation.md` final line): `CAP-V10`
Tier 2 has no forcing case yet in `case-portfolio.md` — Case 3's dashboard need is served today by
the count-tile form. This document only refines the *scope* of a Tier 2 note that already exists,
so that when a real case does force `CAP-V10` Tier 2's admission, the scoping question in §5 isn't
rediscovered from zero.

**What changed, procedurally:**

- `capability-registry.md`'s `CAP-V10` row — its Study 37 Tier 2 note gets a dated append
  narrowing "composing *any* Views" to explicitly name the two-entry-kind distinction (§5) and the
  open context-passing question, without changing the row's ✅ status or admitting a new one.
- `roadmap.md`'s Study 37 log (item 24) gets a dated note pointing here.
- `prototype/objectstack/README.md`'s documents table and status header get this document added.

## 7. Suggested wording for `CAP-V10` Tier 2, once a forcing case admits it

Not proposed as a new registry row — recorded here so the wording exists when a case forces the
decision, per this repo's "declare targets first" discipline:

> `CAP-V10` Tier 2 — a `page` `ViewType` whose `Config.Children` is an ordered list of entries,
> each either `{view: <view_id>}` (an existing View, rendered recursively, permission-checked
> independently the same way `Sections` are today) or `{content: {type, properties}}` drawn from a
> **fixed, closed** vocabulary (`heading`, `text`, `button`, `image` — extended only by adding a Go
> `.templ` function and a registry-seam entry, never by declaring an arbitrary `type` string).
> Context passing from a parent page to a child View (§5's second open question) is a prerequisite
> for the `pending_documents`-style worked example and must be scoped in the same admission pass,
> not deferred again.

**Data resolution mechanism (no new mechanism needed).** A `{view: <id>}` child entry names a View
that already owns its own data query (`ViewConfig.Filter`/`Columns`/Machine — the same fields every
`list`/`detail`/`dashboard` View already declares). Resolving the page means calling, per child,
the same handler logic `handler.List`/`handler.Detail`/etc. already run today (permission-scoped
fetch against Postgres, then that View's own `.templ` render), and stitching the resulting
fragments into the page layout — one request, entirely server-side, no new query layer. Live
refresh without a full page reload (should a dashboard tile need it) reuses the existing HTMX
polling pattern already proven by `CAP-V15`/`CAP-V16` (`hx-get` + `hx-trigger="every Ns"` against a
small per-child fragment endpoint) — no client-side reactive state, the server still owns the data
on every refresh. This closes the "how does dynamic data actually flow" question the proposal's own
§13 render pipeline left at the concept level.

## 8. Follow-up (2026-09-09): two more scoping holes in §7, and a necessity call on §5's open question

An owner Q&A session re-read this document and the original proposal's own §4/§9/§10/§11 against
`app/` a second time, two days after §5–§7 were written. Two things §7's Tier 2 wording does not
yet cover, and one priority judgment on the question §5 already named but left open:

**(i) Recursive depth is undesigned, not just unbuilt.** `CAP-V20` Tier 2's own
`EmbeddableChildViewTypes` (`app/internal/model/model.go:859-862`) lists exactly two Types today,
`decision_stepper` and `coord_placement` — both leaves, neither itself composable. Nothing in §7's
wording says whether a Tier 2 page's own `{view: id}` child may itself be a `page` (or any other
composing Type) with children of its own. The proposal's own ask (§4/§11: "View → Component → View
→ Component → View," arbitrary depth) is not a smaller version of what §7 scopes — it is a
different, larger question (cycle detection, a depth limit or none, whether permission-checking
composes correctly at depth >1) that has not been asked yet, let alone answered. Flag it explicitly
whenever `CAP-V10` Tier 2 is formally scoped, rather than assume single-level composition
generalizes for free.

**(ii) Composed-page layout is a real gap in §7's own evidence, not just the pre-existing G22 Form
gap.** §7 describes `Children` as an *ordered list* — stacked, single-column. But
`benchmarks/029-composed-view-component-inventory.md`'s "Two-column / Grid Layout" component
(`app/web/static/ui-sample/component-proof.html` §11) is evidenced twice: once at Detail-page level
(`document-approval.html`, already tracked as G22 → R19/R20) and once at **composed-page level**
(`approval-dashboard.html` itself — the same mockup this Tier 2's own §5/Study-38 evidence is
drawn from). Building Tier 2 exactly as §7 words it today — a flat stacked list — would not
reproduce the layout of the very mockup used to justify it. This needs folding into §7's wording
(a layout hint per entry, or a fixed set of composed-page layout shapes, closed-vocabulary the same
way §7's `{content: {type, properties}}` already is) at scoping time, not discovered again then.

**(iii) Context-passing's priority, assessed rather than left as an open flag.** §5 named
`$context.document.id`-style parent→child scoping as undeclared; this pass checked what actually
exists today and what forcing pressure exists now. What exists: exactly one dynamic Filter token,
`$current_user` (`CAP-V05`; a real literal substitution, `app/internal/expr/expr.go:15,27,91`,
resolved from the request's own identity) — general-purpose, but answers only "who is logged in,"
never "what record is this page currently about." `CAP-V06`'s reverse-reference sub-lists cover one
narrow instance of parent-scoping automatically (child rows whose own `reference` field equals the
parent's id) but are computed implicitly, not a declarable Filter token usable in an arbitrary
composition. **Priority: low today** — no forcing case in `case-portfolio.md`, and the one mockup
that looked like it would need this (`approval-dashboard.html`) already shipped live
(`app/seeds/044_document_submit_dashboard_live_wiring.sql`) using plain `CAP-V10` Sections, needing
no context-passing at all. **But mandatory, not deferrable, in the same pass that admits `CAP-V10`
Tier 2 itself** — without a parent→child context token, a composed page can only embed
context-free Views (global summaries, exactly what already-✅ `CAP-V10` gives today), never the
parent-record-scoped child View (e.g., "items related to *this* document") that is the proposal's
own actual motivating example. Admitting Tier 2 without also closing this would ship a capability
that cannot serve the case that justified it.

**Disposition, unchanged from §6:** no capability admitted, no row status changed. This section
only adds detail to the Tier 2 scope-in-waiting so items (i)–(iii) aren't rediscovered from zero
whenever a real case finally forces `CAP-V10` Tier 2's admission.

## 9. Fourth pass (2026-09-11): a full restatement of the same proposal, checked against the now-shipped `CAP-V10` Tier 2

Two days after §8, the owner brought a fourth document on the same proposal — a full top-to-bottom
architectural review (22 numbered sections, Bahasa Indonesia, no code citations of its own),
re-running the "Composable Metadata Runtime" direction from first principles against the repo as
it reads today. Its own framing (§17 of that document) explicitly treats `CAP-V10` Tier 2 as
already live and as "Version 1 of a Composition Model, not the final form" — so, unlike §1–§8,
this pass starts from *after* the 2026-09-09 ship, not before it. Every verdict below was checked
against `capability-registry.md`'s `CAP-V10` / `CAP-V10 Tier 2` rows and `app/` source, the same
discipline §2 used, not taken from the document's own prose.

### 9.1 Claims already settled or already closed by the ship

| The document's claim | Verdict | Evidence |
|---|---|---|
| Composition is still "Page → View → View," with no Component or Layout in the tree | **Partly outdated.** `CAP-V10` Tier 2 (implemented 2026-09-09) already mixes `{view: <id>}` entries with a closed-vocabulary `{content: {type, properties}}` set (`heading`/`text`/`button`/`image`) in the same `Children` list, plus a `Layout` field (`""`/`"main"`/`"aside"`, pairing two entries into a 2/3+1/3 grid row) — i.e. Page → {View \| Content}, with one real layout shape, not zero | `capability-registry.md` `CAP-V10 Tier 2` row, "Implemented 2026-09-09" paragraph; `app/internal/model/model.go` (`PageContent`, `Layout` fields) |
| Static content + structured View "belum jadi primitive universal," Case 13 still unexplored | **Primitive is real; the Case-13 application of it is what's still open.** The same closed static-content vocabulary is live today on `seeds/050_composed_dashboard.sql`'s own Recent Activity section (a `content` entry). What Study 38 actually found still true: Case 13 (Blog landing) and Case 10 (Organization Composite) have zero visual exploration using it — a mockup-coverage gap, not a missing mechanism | `benchmarks/029-composed-view-component-inventory.md`; `capability-registry.md` `CAP-V10` row's Study 38 note |
| Component Registry is needed so metadata can reach a renderer without a fixed `ViewType` switch | **Reaffirms §4, no new argument.** The document's own §16/§17 explicitly declines to recommend removing Machine/View/the registry-seam model — it argues for evolution, not the dynamic-dispatch machinery §4 already rejected structurally (no client-side interpreter exists to validate against at runtime) | §4 above, unchanged |
| Generic Layout model (Stack/Grid/Split/Tabs/Panel) is missing | **Real, but already tracked, narrower than described.** `CAP-V10` Tier 2 ships exactly one shape (`main`/`aside` pairing); a fuller vocabulary is the same gap already named as `G22`/`R19`/`R20` (form sections/columns, low priority) in `gap-analysis-and-recommendations.md`, now with one more confirming data point (composed-page layout, §8(ii) above) | `prototype/objectstack/docs/gap-analysis-and-recommendations.md` rows G22/R19/R20 |
| Recursive View-in-View nesting (arbitrary depth) is a real gap | **Already named, unchanged.** §8(i) above named this exact gap two days before this document arrived — `CAP-V10` Tier 2's own admitted scope explicitly excludes a `page` composing another `page` | §8(i) above; `CAP-V10 Tier 2` row, "Explicitly still out of scope" |
| A unified binding/context model (route, params, user, selected record, parent record) is missing | **Already tracked as an open, prioritized question**, narrower framing than the document's. §5/§8(iii) above cover the parent→child scoping case specifically (the one with real forcing pressure); the document's broader "page context" (route/params as declared tokens) has no forcing case named anywhere in `case-portfolio.md` today. `$current_user` is the one general-purpose dynamic Filter token that exists (`CAP-V05`, `app/internal/expr/expr.go`) | §5, §8(iii) above |

### 9.2 Framings genuinely new to this study

Three of the document's asks are not restatements — they don't appear anywhere in Study 37/38/39/40
or this reconciliation's own §1–§8. None are admitted here; each is recorded so it isn't
rediscovered from zero if a case ever forces the question.

**(a) Query/Projection as a named layer independent of any one View** (the document's §5/§6 —
"one Customer query, rendered as Table, Kanban, or Card"). Partly pre-covered: `ViewConfig` already
unifies Filter/Sort/Group *per View* (§2 table row above); `CAP-C13` (✅, the CEL expression
operator) is already a general compute/transform primitive usable in filters and computed Fields;
`CAP-V22` (❌ Proposed, "semantic dataset," Study 37 R2) already proposes the reusable-named-
*aggregate* half of this ask. What is genuinely uncovered: a bare, reusable row-level data shape
addressable independently of any View — today reuse only happens by referencing a whole View by id
(`{view: <id>}` in `Children`), never a data shape alone rendered by more than one presentation. No
forcing case named in `case-portfolio.md`. Not a candidate registry row — the existing `CAP-V22`
row is the nearest neighbor and should carry a pointer here, not a new row, until a case actually
needs the same rows shown two ways.

**(b) A UI Intermediate Representation / compile step for UI metadata** (the document's §12/§13 —
metadata → normalize → resolve → compile → render plan, as a formal stage before rendering). The
underlying *pattern* — declarative metadata compiled at load time into lower-level runtime
primitives — is already proven, just not for UI: the Process Overlay (Study 21, `CAP-W01`,
`internal/metadata/compile.go`) does exactly this for `process` blocks. Extending the same
discipline to UI metadata is a reasonable analogy, but the concrete problem a UI IR exists to solve
— one compiled representation serving *multiple* renderers (web, mobile, a visual builder) — doesn't
apply today: `app/ARCHITECTURE.md`'s client-side JS policy commits this runtime to exactly one
server-rendered target, by design, not many. Recorded as an idea worth revisiting only if/when a
second rendering target is ever pursued; no forcing case, no candidate registry row.

**(c) Composability-measuring benchmark KPIs** (the document's §20/§21 — Application Construction
Ratio, Composition Reuse Ratio, View/Data Independence, Runtime Extension Cost). A new measurement
method, not used by any prior study in this series (which measure capability coverage and
conformance-test count, not composition ratios). Not adopted here — a reconciliation document
doesn't schedule a new study; recorded as a candidate method for whichever future study takes on
Query/Projection or UI-composition depth next, per `roadmap.md`'s own "Recommended order"
mechanism.

The document's own closing ask — three follow-on artifacts (a Composable Runtime Architecture
Spec, a Gap-to-Roadmap Matrix, and a UI-IR benchmark/prototype) — is recorded here as owner-
suggested next steps, decision pending, the same disposition Study 37 itself closed with.

### 9.3 Disposition

**No capability admitted, no row status changed, no code changed.** `CAP-V10 Tier 2` stays ✅ at
its already-shipped scope — nothing in this pass asks for a scope change to what's live, only for
(a) and (b) above to be tracked as separate, unadmitted framings the moment a case forces either
one. Procedural updates only: this section; `capability-registry.md`'s `CAP-V10 Tier 2` row gets a
short dated pointer (not a scope change); `roadmap.md`'s Study 37 log gets a dated addendum
pointing here; `prototype/objectstack/README.md`'s status header and documents table.

## 10. Fifth pass (2026-09-11): the owner's own synthesis — Data-first and UI-first framings merged

The owner replied to §9 with their own synthesis, comparing it against a proposal they had raised
verbally in an earlier, unrecorded exchange (a Dataset/Query/Projection-first framing, not
previously written into this study). Their own conclusion: the two are not competing directions —
§9's document is stronger on *UI decomposition* (identifying `ViewType` proliferation as an
architectural tax and proposing Page→Layout→Component→Binding→Primitive), their own is stronger on
*data/query composability as a performance foundation* (Dataset→Query Planner→Physical Plan, tying
composability to Study 8's own scale-architecture concerns) — and Menata needs both, meeting in the
middle. Full synthesis: three composability dimensions (Data / UI / Behavior) converging through
one compiler discipline into an Execution Plan (database) and a UI IR (renderer); `View` demoted
from "fundamental UI primitive" to "specialized composition preset" (a `list` View ≈ a Collection
component + table renderer, a `dashboard` View ≈ a Page + grid layout + metric/collection
components); a Query Planner/Cost Model named as the missing link between `CAP-V22`'s already-
proposed semantic dataset and Study 8's already-registered index/cache scale findings; and a
sharpened admission gate ("can an existing primitive compose it? can a new *generic* primitive
compose it? only then is it a real capability") offered as a refinement of, not a replacement for,
`capability-lifecycle.md` §2's existing A1–A5 test.

**Disposition:** no capability admitted, no row changed, no code changed — this pass is a
synthesis and a request, not a new technical claim to verify against code. Per the owner's own
explicit ask, its output is a dedicated blueprint document rather than a further reconciliation
section here: **[`composable-runtime-blueprint.md`](../../../composable-runtime-blueprint.md)**
(root level, Tier 3), which organizes everything §1–§10 of this document, `capability-registry.md`
(`CAP-V10`/`CAP-V22`/`CAP-C13`/`CAP-X10`), and `benchmarks/004-scale-architecture-study.md` (Study
8) already established into one target architecture and a phased, evidence-gated evolution plan.
This document (`composable-view-proposal-reconciliation.md`) stays the historical record of how
each individual claim was checked against code; the blueprint is where they were assembled into a
plan.
