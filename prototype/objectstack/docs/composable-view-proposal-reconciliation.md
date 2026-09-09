# Composable-View Proposal Reconciliation — a Third External Document vs. Study 37's Existing Position

> Part of Study 37 (`../README.md`). On 2026-09-07 the owner brought a third document — an
> unsolicited architectural-direction memo titled "Architectural Direction: Composable Metadata
> Runtime," proposing "Everything is composable, View is the primary reusable composition unit" —
> and asked for it to be re-examined against `app/`'s actual code, then discussed further. This
> document is that reconciliation, in the same spirit as `second-opinion-reconciliation.md` (which
> reconciled an independently-produced second review the same day). Every verdict below was checked
> against `capability-registry.md` v0.58 rows and `app/` source, not taken from the proposal's own
> prose.
> Status: v1.1 — §8 added (2026-09-09): an owner Q&A session re-read this document and the
> proposal's own asks against live code a second time, found two scoping holes in §7's Tier 2
> wording (recursive nesting depth, composed-page layout) neither §5 nor §7 named, and assessed the
> §5 context-passing question's actual priority. No row admitted, no code changed | Previously v1.0
> | Created: 2026-09-07 | Updated: 2026-09-09

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
