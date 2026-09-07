# Composed-View UI Component Inventory

> Study 38 of the Capability Roadmap.
>
> Direct continuation of Study 37 (`prototype/objectstack/`) and its
> `composable-view-proposal-reconciliation.md`: that document scoped `CAP-V10` Tier 2 ("a `page`
> View composing any Views, alongside a small closed static-content vocabulary") but named its own
> content-vocabulary as asserted, not evidenced. This study corrects that — the component
> vocabulary a composed page actually needs is derived from real static UI-sample mockups this
> project has already built (`app/web/static/ui-sample/*.html`, the Process Overlay BRD v2
> exploration series plus Case 3's own extension mockups), the same "declare targets first, cite
> real evidence" discipline every other benchmark in this directory follows — not invented from
> imagination. One new mockup (`approval-dashboard.html`) was added this session specifically to
> close the one gap the existing 13 didn't cover: none of them was a page composing several Views.
>
> Status: v1.0 | Created: 2026-09-07 | Updated: 2026-09-07

---

# Method: cluster from real mockups, not case text

`case-portfolio.md`'s own declared targets are Business-Knowledge/capability statements (CAP-Fxx,
CAP-Axx, …), not screen descriptions — they don't say what a screen is *built from*. The actual
terrain evidence for a UI component vocabulary is the 14 static HTML mockups this project has
already produced as design exploration (`app/web/static/ui-sample/`, all explicitly labeled
"static mockup, not wired to real data — delete once validated"). Same cluster method as Study 28
(`benchmarks/020-ui-interaction-cluster-proof.md`) and its own predecessor Study 8
(`benchmarks/008-ui-workflow-interaction-benchmark.md`): find what recurs across the mockups
rather than treating each as an independent research pass, and admit a pattern to the registry
only when a real, cited need forces it — recurrence alone is not sufficient (`capability-
lifecycle.md` §2 A1–A5 still applies per pattern).

**Files examined** (14, `app/web/static/ui-sample/`): `index.html`, `worklist.html`,
`document-approval.html`, `document-submit.html`, `document-signature-placement.html`,
`quorum-approval.html`, `requirement-checklist.html`, `sla-badge.html`, `process-map.html`,
`groups.html`, `group-detail.html`, `group-approval.html`, `group-approval-detail.html`,
`approval-dashboard.html` (new, this session).
A fifteenth file, `component-proof.html` (new, this session), is not itself a case mockup — it
renders the two strongest clusters below from one shared shape each, side-by-side with their
originating markup, as the consolidation proof in its own section further down.

**Cross-checked against what's already real code**, not just other mockups: `internal/ui/
components.templ` already implements `StatusBadge`, `SlaBadge`, `TypeaheadPicker`,
`ChildLinesSection` as genuine reusable Go `templ` primitives — these are marked "already built"
below, not re-proposed.

---

# Clusters examined

## Cluster 1 — Section wrapper (title + source badge + optional action link, wraps any content)

**Evidence:** `approval-dashboard.html`, 3 occurrences (Summary / Pending Documents / Recent
Activity sections) — the only mockup in the set that embeds more than one independent View on one
page, because it is the only mockup built to test that shape.

**What it is:** a header row (`<h2>` + optional link, e.g. "View all →") sitting above arbitrary
content, used consistently to mark "here is where View X's own data begins."

**Why this matters more than the count suggests:** this is the literal missing mechanism
`composable-view-proposal-reconciliation.md` §5 named without specifying it — the thing that makes
"a page composing Views" visually legible instead of an unmarked wall of content. Without it,
`CAP-V10` Tier 2 has no way to show a viewer which section came from which View.

**Verdict:** not a new capability row — it is the definition of `CAP-V10` Tier 2's own `Children`
rendering, folded directly into that Tier 2's scope (updates the note already on `CAP-V10`'s row,
below). A3 (Cross-Cutting, View) ✅, A4 (not composable from something existing) ✅ — nothing
today wraps arbitrary content with a titled, linked header.

## Cluster 2 — Record Summary Card (avatar-initials + title + subtitle + status badge, one row)

**Evidence:** near-identical markup in 4 mockups — `quorum-approval.html`, `requirement-
checklist.html`, `sla-badge.html`, `process-map.html` — all render the same "CA-2044 · Line 3
Conveyor Sensor Fault / Corrective Action · Andi Prasetyo · 01 Sep 2026 / [status pill]" shape.

**Verdict:** presentation primitive, not a registry row — a reusable `templ` component
(`RecordSummaryCard`) supporting whatever View already renders a record's header (Detail, the
Process Overlay screens CAP-W01/W03/W04/W05 already own). No new data/behavior, same class as
`StatusBadge` already is. Implementation task, not a capability admission.

## Cluster 3 — Avatar / Avatar Stack (initials circle; overlapping-circle group variant)

**Evidence:** 8 of 14 mockups render a person as a 2-letter initials circle; `groups.html` and
`group-approval.html` additionally show the stacked/overlapping variant for "N members."

**Verdict:** presentation primitive, not a registry row — supports every existing `user`/
`reference` field rendering (CAP-F05/CAP-F13). No new row; a `templ` component task.

## Cluster 4 — Divided List (rows: avatar/icon + content + trailing status or action)

**Evidence:** `quorum-approval.html` (approver list), `requirement-checklist.html` (requirement
list), `group-detail.html` (member list) — 3 occurrences, same `<ul class="divide-y">` shape.

**Verdict:** presentation primitive generalizing `CAP-V02`'s own row rendering — not a new
capability. Distinct from Cluster 8 (Activity Feed) below by content shape: this is an *actionable*
row (has a trailing decision/status), the feed is a *narrative* row (actor did X to Y at time T).

## Cluster 5 — Stat Tile / KPI Card (large number + label + color accent + breakdown chips)

**Evidence:** `approval-dashboard.html`, 3 tiles (Total Pending / Overdue / Due Today) — new this
session; none of the 13 prior mockups render a large-number-focused tile, only small pills.

**Verdict:** this is the real visual form of `CAP-V10`'s existing ✅ mechanism (today's
`internal/ui/dashboard.templ` renders it plainer) **and** reinforcing evidence for the already-
proposed **`CAP-V23`** (chart section, "kpi-with-delta") — no new row, additional A1 evidence on
an existing candidate.

## Cluster 6 — Activity Feed / Timeline List (actor + verb + object + timestamp, narrative)

**Evidence:** `approval-dashboard.html`, 4 rows ("Rina Nur approved Vendor Contract Q3", etc.) —
new this session, distinct shape from Cluster 4 above (no trailing action, chronological narrative
not a decision queue).

**Verdict:** reinforcing evidence for the already-proposed **R28** (`CAP-R04` → ✅, a record
history timeline View, `second-opinion-reconciliation.md` §3) — that recommendation was scoped to
one record's own field-diff history; this mockup shows the same visual shape also serving a
cross-record dashboard activity feed. Worth noting on that row as a second use-case, not a second
capability — no new row.

## Cluster 7 — Sticky Action Bar (sticky footer: contextual info left, primary/secondary buttons right)

**Evidence:** identical shape in `quorum-approval.html`, `requirement-checklist.html`,
`document-approval.html` — 3 occurrences.

**Verdict:** presentation/layout primitive, not a registry row — supports any action-taking View
(Detail, the Process Overlay screens). Implementation task.

## Cluster 8 — Filter Chip / Segmented Control

**Evidence:** `worklist.html` and `document-approval.html` (status filter chips: "All/Overdue/Due
today"), `document-submit.html` (User/Group mode toggle) — 3 occurrences, two different uses
(filter vs. mode-select) sharing one visual shape.

**Verdict:** presentation of `CAP-V09`'s existing filter mechanism — not a new row.

## Cluster 9 — Removable Chip / Tag

**Evidence:** `group-detail.html`'s member chips (`× ` remove button) — 1 occurrence.

**Verdict:** reinforcing evidence for the already-proposed **R13** (`CAP-F03` Tier 2,
`multiple: true` rendered as removable tags) — no new row.

## Cluster 10 — Choice Card (radio input rendered as a bordered card with label + description)

**Evidence:** `document-submit.html`'s Sequential/Parallel mode picker — **1 occurrence only**.

**Verdict:** **not admitted** — a single instance is not a pattern (this study's own method
statement above). Named here so it isn't silently lost, same "silence is not a decision" posture
`capability-registry.md`'s own tracked-but-not-yet-studied rows use — revisit if a second mockup or
case independently produces the same shape.

## Cluster 11 — Two-column / Grid page layout (content 2/3 + sidebar 1/3; page-level section grid)

**Evidence:** `document-approval.html`'s detail section (Document + Approval Progress cards left,
Signature Position aside right) and `approval-dashboard.html`'s own layout (Pending Documents 2/3,
Recent Activity 1/3) — 2 occurrences, one at Detail-page level and one at composed-page level.

**Verdict:** reinforcing evidence (second independent instance) for the already-named gap in
`prototype/objectstack/docs/gap-analysis-and-recommendations.md` G22 (R19/R20, "form sections /
multi-column layout... presentation; low"). Still no dedicated case forcing it as its own
admission — stays a named, low-priority gap, not a new row.

---

# Consolidation proof — testing the claim, not asserting it

Naming a cluster ("this recurs 4×") is not the same as proving one shared component can actually
render all 4 occurrences with no visual loss. New mockup `component-proof.html` tests two clusters
directly: takes each occurrence's real markup from its origin file, renders it from one proposed
parameterized shape, and checks for divergence.

**Cluster 2 (Record Summary Card), 4 sources — proven clean.** Wrapper, avatar, and title markup
are byte-identical across `quorum-approval.html`, `requirement-checklist.html`, `sla-badge.html`,
`process-map.html`. Only `subtitle` and the trailing badge vary — and the badge slot is already
`StatusBadge`/`SlaBadge`, real code in `internal/ui/components.templ`. One real variance found
(`process-map.html`'s subtitle adds "submitted by", its badge reads "Currently: Review" not just
"Review") — both still fit as free-text params, not an enum, so the shape holds:
`RecordSummaryCard(avatarInitials, title, subtitle, badge templ.Component)`.

**Cluster 7 (Sticky Action Bar), 3 sources — proven, with a correction.** The outer bar (sticky
footer, label+message left) is reusable across all 3. The original assumption (a single
disabled-until-ready button) breaks on `document-approval.html`, which always shows two enabled
buttons (Reject + Approve) — not a state variant of one button. Corrected shape: `actions
[]templ.Component` (0..N), which still covers the single-button case as a 1-element slice.
**This is exactly the kind of finding worth testing for before writing the real Go component** —
assuming the narrower shape from only 2 of the 3 sources would have shipped a component that
silently couldn't render Document Approval's own decision bar.

**Method note for future clusters:** this comparison is cheap (it's a markup diff, not new code)
and caught a real structural gap Cluster 7's own one-line description didn't — worth running for
every cluster before it becomes a real `templ` component, not only the two done here as the proof
of the method itself.

---

# Summary — what actually changes in the registry

**Nothing is admitted.** Consistent with every prior Study 37/38 pass: this inventory adds
evidence to existing rows/notes, it does not register new capabilities on its own.

| Cluster | Registry impact |
|---|---|
| 1 (Section wrapper) | Folds into `CAP-V10` Tier 2's own scope — this *is* its rendering mechanism |
| 5 (Stat Tile/KPI) | Additional A1 evidence on `CAP-V23` |
| 6 (Activity Feed) | Additional use-case evidence on R28 (`CAP-R04` → ✅) |
| 9 (Removable Chip) | Additional A1 evidence on R13 (`CAP-F03` Tier 2) |
| 11 (Grid layout) | Second independent instance of the already-named G22/R19-R20 gap |
| 2, 3, 4, 7, 8 | Presentation primitives — implementation tasks (`templ` components), not registry rows |
| 10 (Choice Card) | Named, not admitted — single instance, no pattern yet |

---

# Case-to-mockup coverage — what's visually explored vs. what isn't

`case-portfolio.md`'s 21 cases, cross-checked against the 14 mockups above. A mockup gives real
component evidence; a case with no mockup and no real running View of a distinct shape is still
only evidenced at the capability-declaration level, not the presentation level.

| # | Case | View-relevant target(s) | Mockup? | Note |
|---|---|---|---|---|
| 1 | Design Request | CAP-V01–V03 (form/list/detail) | — | Foundational, exercised by every Machine in the real app itself; no dedicated mockup needed |
| 2 | Leave Request | same as 1 | — (appears as one row *inside* `worklist.html`) | Domain-portability case, not a new shape |
| 3 | Document Approval | CAP-F13/A07/A08, CAP-V20/V21, `CAP-V10` T2 | **✅ 6 files** + new `approval-dashboard.html` | Best-covered case by far |
| 4 | Maintenance Reminder | — | — | No distinct View shape declared |
| 5 | Inventory/Stock Movement | — | — | No distinct View shape declared |
| 6 | Petty Cash Ledger | — | — | No distinct View shape declared |
| 7 | Customer Complaint | CMMN/SLA/escalation | — (conceptually close to the generic "Corrective Action" example in `requirement-checklist.html`/`sla-badge.html`/`process-map.html`, not Case 7 itself by name) | The Process Overlay mockups use `mch_corrective_action`, `runtime-metadata-schema.md`'s own canonical Process Overlay example — not a Case 7 mockup |
| 8 | Payment Confirmation | — | — | No distinct View shape declared |
| 9 | Accounting (Trial Balance) | **CAP-V13** report view | **none** | A genuinely different visual shape (grouped totals, running balance) never mocked |
| 10 | Organization Composite | **CAP-V10** (cross-app dashboard) | **none** | The literal composition case — never visualized despite being CAP-V10's own cited evidence |
| 11 | Social App | CAP-V05 feed | — | Feed/infinite-scroll explicitly reviewed and NOT admitted (Study 8) — no pressure to mock |
| 12 | Community Site | CAP-A14 | — | No distinct View shape declared |
| 13 | Blog / One-Page Site | **CAP-V10** (public landing), CAP-P07 | **none** | **Highest-priority gap** — cited as CAP-V10 evidence in the registry itself, yet zero visual exploration; the literal case for "mix static content with structured Views" this whole discussion has been about |
| 14 | Lending Services | CAP-A15 (schedule) | — | A repayment-schedule table is a distinct shape, not yet explored |
| 15 | E-commerce | CAP-R08 (cart) | — | No distinct View shape declared |
| 16 | Point of Sale | composition only | — | No new shape expected |
| 17 | Helpdesk | reuses Case 7 | — | No new shape expected |
| 18 | HR Operations | CAP-F13 tree (org chart) | **none** | Pairs directly with proposed `CAP-V26` Tree lens — org-chart shape never explored |
| 19 | Project Management (Trello) | **CAP-V14 Tier 2** board | **real code, no mockup** (`internal/ui/board.templ` is ✅ built and running) | Better evidenced by screenshotting the real render than building a new static mockup |
| 20 | Hospital System | **CAP-V07/V18** resource calendar, CAP-P06 | **real code, no mockup** (`internal/ui/calendar.templ` is ✅ built and running) | Same — real code exists, review it directly |
| 21 | E-learning | CAP-F21 (certificate) | — | No distinct View shape declared |

**Recommended next mockups, in priority order** (maximizing new, non-redundant component
evidence — not redundant with Clusters 1–11 above):

1. **Case 13 (Blog one-page landing)** — closes the actual open question from this session's own
   earlier discussion (what, if anything, a closed static-content vocabulary should contain
   alongside `{view: id}` entries in `CAP-V10` Tier 2's `Children`) with real evidence instead of
   an invented list.
2. **Case 9 (Accounting Trial Balance)** — `CAP-V13`'s own report shape (grouped totals, running
   balance) has no visual exploration at all.
3. **Case 10 (Organization Composite dashboard)** — a second, differently-shaped composed-page
   case (cross-app nav + org-wide reporting) to stress-test `CAP-V10` Tier 2 beyond the
   single-application `approval-dashboard.html` example.
4. **Case 18 (HR org chart)** — direct visual precedent for `CAP-V26` (Tree lens).
5. Cases 19/20 — **skip new mockups**, review the real `board.templ`/`calendar.templ` renders
   instead; building a second, static, duplicate exploration would be lower-value than the four
   above.

No mockup has been built for 1–4 yet — this is a plan, not a claim of coverage. Owner's call on
which (if any) to build next, per this repo's own "declare targets first" discipline.
