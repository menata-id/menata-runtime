# UI Sub-Component Decomposition Criteria — Case 3 + Case 19 Inventory Update

> Study 40 of the Capability Roadmap.
>
> Direct continuation of Study 38 (`benchmarks/029-composed-view-component-inventory.md`), triggered
> by an owner request once both Case 3 (Document Approval, 4 mockups) and Case 19 (Project
> Management, 9 mockups + a flagship page) reached "mockup UI complete." The request had three
> parts, answered in three sections below: (1) break the now-complete mockups down into reusable
> sub-components and record the recurrence, (2) state the *criteria* for that split — referencing
> benchmarks from real, non-metadata applications, not house opinion — and (3) map each resulting
> sub-component onto how a Runtime Metadata author would actually invoke it, distinguishing what
> already has a real schema mechanism from what is still a named gap.
>
> Method inherited directly from Study 38 (cluster from real mockups, consolidation-proof before
> promoting a cluster, "recurrence alone is not sufficient") — this study does not re-derive that
> method, it applies it to evidence Study 38 didn't have (Case 19 didn't exist as static mockups
> yet) and adds the external-criteria and metadata-invocation layers Study 38 left implicit.
>
> Status: v1.0 | Created: 2026-09-10

---

# 1. What changed since Study 38

Study 38 (2026-09-07) explicitly recommended **against** building new Case 19 mockups: its own
"Recommended next mockups" table says *"Cases 19/20 — skip new mockups, review the real
`board.templ`/`calendar.templ` renders instead; building a second, static, duplicate exploration
would be lower-value than [Cases 13/9/10/18]."* The owner overrode that recommendation directly
(this repo's normal posture — a study is evidence for a decision, not a substitute for one) and had
9 Project Management mockups built anyway (`project-board.html`, `project-card.html`,
`project-timeline.html`, `project-calendar.html`, `project-dashboard.html`, `project-team.html`,
`project-automation.html`, `project-settings.html`, plus `case-19.html` as a flagship showcase),
alongside `approval-dashboard.html` already counted in Study 38 rounding out Case 3 to 4 mockups.
Recorded here, not silently absorbed, per this repo's own append-don't-rewrite convention — Study
38's recommendation stands as what it was: correct given the evidence-maximization goal it was
optimizing for, superseded by a direct instruction with a different goal (a flagship, clickable
Case 19 tour, not just component evidence).

**Files examined, beyond Study 38's original 14+1:** `document-submit.html`,
`document-signature-placement.html`, `document-approval.html`, `approval-dashboard.html` (Case 3,
all 4 already in Study 38) — re-read for cross-case confirmation — plus the 9 new Case 19 files
above, `case.html` (the 21-case generic template, Case 3/19 entries), and `index.html`'s own
"Workspace & Access screens" table (6 more mockups: `login.html`, `choose-workspace.html`,
`workspace-home.html`, `workspace-members.html`, `member-role-detail.html`,
`approval-role-matrix.html` — checked for cross-confirmation only, not a third case pass).

---

# 2. Decomposition criteria — what makes a "sub-component," per real (non-metadata) precedent

Study 38 used one working rule (recur ≥2×, then prove it with a consolidation diff) without stating
*why* that threshold, or distinguishing degrees of reuse. The owner asked directly for that
criterion to be grounded in benchmarks from comparable real applications, not asserted. Five
sources, all mainstream, well-documented design-system practice (no case-shaped product here is
metadata-driven — that asymmetry is itself relevant, see §2.6):

| Source | What it establishes |
|---|---|
| **Atomic Design** (Brad Frost, 2013) — the reference vocabulary most industry design systems cite | A strict small→large hierarchy: atoms (badge, avatar) → molecules (a labeled stat) → organisms (a data table, a nav bar) → templates → pages. Useful as *vocabulary*, not as an admission test — it says how to name a layer, not when a pattern earns one. |
| **Shopify Polaris** | Explicit split between **Structural components** (Card, Page, Layout, ResourceList — reused across any admin surface) and **merchant-specific patterns** that stay inline in one flow, never promoted. Promotion requires the "Rule of Three" — a pattern used independently in ≥3 places before it graduates into the shared library, the same threshold long-established in general refactoring practice (Fowler/GoF community) — stricter than Study 38's own ≥2. |
| **GitHub Primer** | A two-tier lifecycle — new patterns live in product-scoped "Labs," and only graduate to core Primer once **≥2 independent GitHub products** (not 2 screens of the same product) use the identical shape. This is a *cross-product* bar, not merely cross-screen — directly analogous to this repo's own Incubating→Supported lifecycle (`capability-lifecycle.md`), which already gates on independent evidence sources (A1) before a capability graduates. |
| **Atlassian Design System (ADS)** | Splits **Foundations** (color, type, Avatar, Badge/Lozenge — shared across Jira/Confluence/Trello) from **product-specific patterns** that live inside each product's own codebase (Jira's issue-type icon picker, Trello's card-cover-color picker) even when polished and reused *within* that one product. Directly relevant here because Case 19 is explicitly Trello-shaped: Trello's own card-cover-color picker is the real-world precedent for "this is genuinely this product's own thing, not a Foundation," the same judgment this study has to make about the kanban card and the sequential approval stepper below. |
| **Material Design 3** | A component enters the shared system only once **≥2 independent product surfaces** need behaviorally identical treatment; before that, it is scoped custom UI local to one surface. Converges with Primer's cross-product bar, not Polaris's raw occurrence count — recurrence *within* one product is necessary but not sufficient. |

## 2.1 Synthesis — the test this study actually applies

No single cited source is used as-is; capability-lifecycle.md §2's own five-criterion admission test
(A1–A5) is the closest existing structure in *this* repo, and the pattern above converges on an
almost line-for-line presentation-layer analogue of it. Stated explicitly, for the first time, as
this study's own deliverable:

| # | Criterion | Presentation-layer test | Real-world anchor |
|---|-----------|--------------------------|--------------------|
| P1 | **Cross-domain recurrence** | ≥2 independent occurrences, in genuinely different business domains (not two screens of the same case) | Primer (cross-*product*), Material (cross-*surface*) — stricter than same-case repetition |
| P2 | **Behavioral identity, proven not assumed** | A single parameterized shape must actually render every occurrence with no silent capability loss — Study 38's own consolidation-proof method, run as a markup diff before any `templ` code is written | Nothing cited names this explicitly, but it is what "Rule of Three" *protects against* in practice — premature promotion on visual similarity alone is the standard failure mode general refactoring literature warns about |
| P3 | **Foundation vs. product-specific** | Does correctness depend on domain-specific business rules (a sequence number, a coordinate pair, a drag-across-lane state machine), or purely on generic shape (a label, a count, a timestamp)? The former stays product-specific even after 10 occurrences; the latter is a Foundation candidate after 2 | ADS's Foundations/product-specific split, using Trello's own card-cover-picker as the concrete "stays specific" precedent |
| P4 | **Metadata-invocation reachability** — *specific to this runtime, no direct external analogue* | A shared internal `templ` helper is a weaker form of reuse than a component a metadata author can independently select via a real schema key (`display: cards`, a `ViewType`, a `content.type`). Rank every finding by which tier it reaches | This runtime's own Metadata First principle (`001-design-principles.md`) — no comparator here is metadata-driven, so this criterion has no borrowed precedent; it is this study's own addition, not imported |
| P5 | **Cost of premature promotion** | Recurrence alone does not force admission — same posture `capability-lifecycle.md` A4 already takes for capabilities generally | Study 38's own stated method: "admit a pattern to the registry only when a real, cited need forces it" |

## 2.2 What this changes about Study 38's own verdicts

Nothing is retroactively overturned — re-checked against P1–P5, every one of Study 38's 11
clusters still lands where Study 38 put it. What changes is *why*: Cluster 10 (Choice Card,
"not admitted — single instance") now has a stated reason beyond "only 1 occurrence" — it also
hadn't cleared P1's cross-*domain* bar even hypothetically. The new evidence below is graded
against P1–P5 directly, not against Study 38's looser recurrence-only rule.

---

# 3. Updated cluster inventory

## 3.1 Study 38 clusters reinforced by Case 19 (cross-domain confirmation — the strongest kind, P1)

**Cluster 5 (Stat Tile/KPI)** — was 1 case family (`approval-dashboard.html`'s 3 tiles). Now
**3 independent domains**: `project-dashboard.html` (Open cards / In progress / Blocked / Sprint
completion) and `project-team.html` (Team capacity / Allocated / Unallocated) reproduce the exact
same shape (`rounded-lg border bg-white p-4/5` → small uppercase label → large number → small
caption). This clears P1 decisively and was already real code before Case 19 existed —
`internal/ui/dashboard.templ`'s `StatTile` (cited directly in that file's own doc comment) is the
proof P2 (behavioral identity) doesn't even need re-running here; the mockups converged on a shape
the real component already implements. **Verdict: Foundation, already built, no action.**

**Cluster 6 (Activity Feed / Timeline List)** — was evidenced only inside Document Approval's own
`approval-dashboard.html`. `project-card.html`'s own "Activity" section (avatar + narrative
sentence with bold actor/object + relative timestamp: *"Raka moved this card from To Do to In
Progress · 2h ago"*) is markup-identical in shape to `approval-dashboard.html`'s *"Rina Nur approved
Vendor Contract Q3 · 10:42"*. This clears P1 (project management vs. document approval are genuinely
different domains) and is now the **strongest, best-evidenced gap in this entire study** — see §5,
it converges directly with an already-registered proposal (R28/`CAP-R04`) that the live, shipped
`/mch_approval_document/page` route is *already faking with a static placeholder* pending this
exact View type. Three independent sources now: Study 37's second-opinion review (R28's origin),
Study 38's `approval-dashboard.html` mockup, and this study's `project-card.html` mockup.
**Verdict: Foundation candidate, P1–P3 all clear, P4 is the one open question (see §5) — ready for
the admission test capability-lifecycle.md §2 would run on R28 directly.**

**Cluster 11 (Two-column / Grid page layout)** — was 2 instances (`document-approval.html`'s
Detail + `approval-dashboard.html`'s composed page). `project-card.html`'s own layout
(`grid gap-6 lg:grid-cols-[1fr_300px]` — content left, metadata sidebar right) is a third,
independent instance of the same content+aside shape, and clears P1 since it's a Detail-style page
in a domain neither prior instance touched. **Verdict: unchanged from Study 38 — still a named,
not-yet-admitted gap** (`prototype/objectstack/docs/gap-analysis-and-recommendations.md` G22/R19-R20),
now with a third confirming data point. Not promoted here — P5 (cost of premature promotion)
applies: three occurrences of a *layout* shape is real signal, but the runtime's own `page` View
already has a `layout: main|aside` mechanism (§4 below) covering the page-composition case;
whether Detail-type Views need their own two-column config is a narrower, still-open question this
study names but does not resolve.

## 3.2 New clusters, not in Study 38 at all

**New Cluster 12 — View-type switcher (tab bar linking a Machine's own auxiliary Views).**
Evidence: `project-board.html`, `project-timeline.html`, `project-calendar.html`,
`project-dashboard.html` — a `Board / Timeline / Calendar / Dashboard` bar, one page per view type.
**Different in kind from every other finding in this study**: this is not a new component
discovered bottom-up from the mockups — it is the mockups **catching up to an already-decided real
design standard**. `prototype/go/docs/decisions/008-mobile-ui-navigation-standard.md` (ADR-008,
Study 29) already specifies exactly this mechanism — "a small segmented pill under the page title
... linking between a Machine's own View types" — appearing automatically wherever a Machine
declares more than one collection-level View, driven by real data (`Handler.viewNavFor`,
`ui.viewNavPill`, both already shipped). The mockups (hand-authored static HTML, not generated from
metadata) simply hadn't kept the four Case 19 view-type pages consistent with each other — caught
and fixed this same session (only `project-board.html` had the bar, and even there as inert
`<button>`s with no navigation; now all four link to each other, correct tab highlighted).
**Verdict: not a new component to design — already a Foundation, already built (CAP-O03 Tier 3).
No metadata-invocation gap: this is automatic, driven by which auxiliary Views a Machine declares,
never something an author separately turns on.**

**New Cluster 13 — Checklist / progress-fraction widget.** Evidence: `project-card.html`'s
checklist section (progress bar + `<ul>` of checkbox rows, "6/8 complete") and the compact "☑ 6/8"
summary on `project-board.html`'s own cards. **Single case family (Project Management only) — fails
P1.** Named, not admitted, exactly Study 38's own Cluster 10 treatment ("so it isn't silently
lost... revisit if a second mockup or case independently produces the same shape").

**New Cluster 14 — Settings navigation rail.** Evidence: `project-settings.html`'s
`200px-fixed-nav + content` two-column shape (General / Lists & statuses / Labels / Custom fields /
Permissions). Visually adjacent to Cluster 11 but a different *purpose* (in-page section switcher,
not a content+metadata split) and a different ratio (fixed 200px, not a 2/3+1/3 grid pairing).
**Single instance — fails P1.** Flagged per Study 38's own method note: cheap to consolidation-proof
against Cluster 11 later, not worth doing on one instance now.

**New finding, not a reuse cluster — a drift, same class as the tab-bar bug in §3.2 Cluster 12.**
`workspace-members.html` renders its member list as a real `<table>`; `project-team.html` renders
its own member list as `<div class="grid grid-cols-[1.5fr_1fr_1fr_1fr]">` rows — same conceptual
shape (a columnar list of people), two different markup strategies, neither reconciled with the
other. This matters because the runtime already has a real, metadata-selectable choice for exactly
this decision — `display: cards` vs. the default table (§4) — and `project-team.html`'s grid-div
markup matches **neither** real rendering mode; it's a third, unauthorized shape that reached a
mockup with no deliberate metadata decision behind it. Recorded here as a mockup-fidelity note, not
a capability gap: when this mockup is next used as an implementation reference, it should be
normalized to one of the two real `display` values, not treated as license for inventing a third.

---

# 4. Metadata invocation — what's already real vs. still a gap

Direct answer to the follow-up question: *how does a metadata author command a View to render as
one of these sub-components, from the metadata itself, not from hand-written HTML.* Checked
against `runtime-metadata-schema.md`'s own "Views" section (current as of `CAP-V10` Tier 2's
2026-09-09 implementation) rather than assumed.

| Cluster / component | Metadata-invokable today? | How | Citation |
|---|---|---|---|
| Section wrapper (Study 38 Cluster 1) | ✅ Yes | Any `page` View's `children` entry — `title` sets the header, the entry's own content is the wrapped section | `runtime-metadata-schema.md` "View composition — `children`", `CAP-V10` Tier 2 registry row |
| Record Summary Card (Cluster 2) | ✅ Yes | `type: list`, `display: cards` — avatar/title/subtitle/badge resolved automatically from `columns` | `runtime-metadata-schema.md`'s `vw_list_cards` example, CAP-V02 Tier 2 (2026-09-09) |
| Stat Tile / KPI (Cluster 5) | ✅ Yes | `type: dashboard`, `sections: [{title, machine, group_field}]` | `runtime-metadata-schema.md`, `internal/ui/dashboard.templ`'s `StatTile` |
| Two-column page layout (Cluster 11, page-level) | ✅ Yes, page-level only | Two consecutive `children` entries with `layout: main` / `layout: aside` | `runtime-metadata-schema.md` "View composition — `children`" |
| Sequential decision stepper | ✅ Yes | `type: decision_stepper`, `{sequence_field, decision_field}` | `runtime-metadata-schema.md`, `CAP-V20` |
| Coordinate placement pin | ✅ Yes | `type: coord_placement`, `{reference_field, preview_field, page_field, x_field, y_field}` | `runtime-metadata-schema.md`, `CAP-V21` |
| View-type switcher tab bar (Cluster 12) | ✅ Yes, but automatic — not an author-set switch | Appears whenever a Machine declares >1 collection-level View; nothing to configure | ADR-008, `CAP-O03` Tier 3 |
| **Activity Feed / Timeline List (Cluster 6)** | ❌ **No — genuine gap** | Today's live `/mch_approval_document/page` fakes this section with a static `content` placeholder (`type: text`) because no View type renders `record_events` as a narrative feed | `capability-registry.md` CAP-R04 note, R28 (Study 37 origin) — this study is a third confirming source |
| Checklist / progress-fraction (Cluster 13) | ❌ No | `child_lines` embeds render as a field table today, not a checkbox list; no `display: checklist` value exists | Named here, not registered — fails P1 (single instance) |
| Detail-level two-column / sidebar (Cluster 11 sub-case) | ❌ No | `layout: main/aside` exists only on `page`-type `children`; a plain `detail` View has no equivalent | `prototype/objectstack/docs/gap-analysis-and-recommendations.md` G22/R19-R20, unchanged status |
| Settings navigation rail (Cluster 14) | ❌ No | No case pressure beyond one mockup | Named, not admitted |

## 4.1 On the one real gap worth acting on (Cluster 6 / R28)

Not decided here — this study names the evidence, it does not choose the implementation shape,
per this repo's own "declare targets first" discipline. Two shapes are visible from the schema as
it already exists, named so neither is silently assumed:

- **A new standalone `ViewType`** (e.g. `activity_log`), sourced automatically from `record_events`
  (actor, Event, timestamp, field diff) with no `Config` fields of its own — the same "opt-in,
  zero-config" posture `process_map` already has. Composed into a page the same way every other
  View is: `children: [{view: <id>, title: "Recent Activity", layout: aside}]` — no change to the
  composition mechanism itself, only a new leaf `ViewType` to reference.
- **Extending `page`'s own closed `content` vocabulary** (`heading`/`text`/`button`/`image`) with a
  fifth type, `activity_feed`, scoped to `page` Views only rather than a standalone reachable View.
  Narrower, but breaks the pattern that every other `content` type is genuinely static — an
  `activity_feed` entry would be the first `content` type that fetches live data, arguably
  belonging on the `children`/`view` side instead, which is the reason the first option above is
  the more consistent one.

Whoever picks this up should run `capability-lifecycle.md` §2's own five-criterion admission test
before building either shape — A1 (dual evidence) is already satisfied three times over (R28's own
origin, Study 38, this study); A3/A4/A5 were not re-checked here, that is real next-step work, not
assumed passed.

---

# 5. Docs impact — what is not done by this study

- **`guides/writing-runtime-metadata.md` is deliberately not touched.** That guide documents real,
  shipped schema only (`capability-lifecycle.md`'s own Definition-of-Done Layer 8 — a translation-
  guide row follows implementation, it doesn't precede it). Every already-invokable mechanism in §4
  is already documented there or in `runtime-metadata-schema.md` directly; the one real gap
  (Cluster 6) has no schema yet to document. Once R28 ships, its own translation-guide row is the
  correct place for "how to write metadata that invokes the Activity Feed component" — not before.
- **No capability admitted, no registry row created.** Consistent with Study 38's own posture:
  this inventory adds evidence to existing rows, it does not self-register new ones. `CAP-R04`'s
  existing row gains a short append-note citing this study as a third independent source for R28
  (see the row itself).
- **`benchmarks/029-composed-view-component-inventory.md`** gains a short append-note pointing
  forward to this study, recording that its own "skip Case 19 mockups" recommendation was
  overridden by direct owner instruction (§1 above) — not rewritten, per this repo's append
  convention.

---

# Correction (2026-09-10, same day) — two authoritative documents this study missed

Found only after this study was written and published, on a direct follow-up question asking for a
consolidated component list for roadmap planning: **`app/docs/ui-component-library.md`** (the real,
already-existing living catalog of every Study 38 primitive as actual `internal/ui/components.templ`
code, v1.7 as of 2026-09-09) and **`guides/breaking-down-ui-components-for-metadata.md`** (the
already-existing step-by-step methodology guide for exactly §2's own question — "how to write
metadata that invokes a decomposed component" — written 2026-09-07, the same day Study 38 landed).
Neither was checked before this study was written; both should have been, per this repo's own
"cite the concrete evidence" discipline. Two corrections follow from what they show:

1. **§4's table understates how much is already built.** `app/docs/ui-component-library.md`'s own
   catalog shows **7 of Study 38's 9 presentation primitives are already real `templ` code**
   (`Avatar`, `AvatarStack`, `StickyActionBar`, `SectionWrapper`, `StatTile`, `MemberChip` — wired to
   a real call site; `RecordSummaryCard`, `ActivityFeedItem`, `DividedList` — implemented but **not
   yet wired**, for named reasons). §4's "❌ No — genuine gap" verdict on Cluster 6 (Activity Feed)
   was too coarse: the **component** (`ActivityFeedItem` + `DividedList`) already exists as real
   code and was already proven against multiple sources by Study 38's own `component-proof.html` —
   the actual, narrower gap is only the **data source**: no `ViewType` yet resolves `record_events`
   into a feed for that component to render. Restated precisely: presentation layer done, metadata/
   View-type layer still missing — a `page` of "already built, wired" progress this study's original
   table didn't show.
2. **§5's "guide doesn't exist yet, write it once R28 ships" framing was wrong on the premise.** A
   methodology guide for exactly this question already existed (`guides/breaking-down-ui-components-
   for-metadata.md`, general method — 9 steps, not tied to any one capability) three days before this
   study was written. What's correctly still missing is not the *method* guide but a `writing-
   runtime-metadata.md` **translation-guide row** for the specific schema keys once R28 ships — that
   part of §5 stands.

`app/docs/ui-component-library.md` is updated in the same pass as this correction (Case 19's own new
named-not-admitted candidates — Checklist widget, Settings navigation rail — added to its own
"Not built" list, matching how it already carries `Choice Card`). This study's own §3/§4 content is
left as originally written per the append convention — read this correction alongside it, not as a
silent replacement.
