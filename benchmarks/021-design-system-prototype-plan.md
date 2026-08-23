# Design-System Prototype Plan — clustering all 21 cases for reusable components

> Study 29 of the Capability Roadmap.
>
> This is the plan Track G asked for (`roadmap.md`: *"produce the design prototype itself —
> mobile layout first — mockups/wireframes and a written component-placement standard covering
> navigation controls generally... before writing any `internal/ui/*.templ` code"*), broadened
> per direct owner request from "just navigation" to **every existing case in
> `case-portfolio.md`**, with an explicit second goal: find which UI shapes repeat across cases,
> so the eventual mockups prove component *reuse*, not 21 one-off screens. Distinct from
> `009-in-app-navigation-benchmark.md` (named the gap, did not design the fix) and
> `008-ui-workflow-interaction-benchmark.md` (clustered *interaction* patterns already shipped
> server-side; this study clusters *visual/layout* shape, not yet designed at all, mobile-first).
>
> Status: v0.1 — plan only, no mockups produced yet | Created: 2026-08-23

---

# Why cluster instead of designing 21 cases

`case-portfolio.md` Rule 2 ("one dominant cluster per case") and `008`'s own method
("cluster, not case-by-case") both apply here for the same reason: designing all 21 cases
independently would produce 21 one-off screens, the opposite of the owner's stated goal
("komponen bisa dipakai berulang... pola style yang bisa dipakai bersama" — components usable
repeatedly, shared style patterns). The right unit of design work is a **UI shape**, not a case;
a case is admitted as a *representative* of a shape only if it's the richest or most demanding
instance of it, the same selection logic `009` already used to name Case 9 "the strongest
evidence" for sideways nav rather than designing all 11 qualifying cases.

---

# Cluster table — every case sorted by UI shape, not business domain

Baseline shape every one of the 21 cases has and needs no separate design pass for: List +
Detail + Form (CAP-F01–F04), already implemented, already sharing `internal/ui/components.templ`.
The clusters below are the shapes *on top of* that baseline.

| Cluster | Capability | Cases (all instances) | Richest representative |
|---|---|---|---|
| Cross-Machine sideways nav (`subNavBar`) | CAP-O03 Tier 2 ✅ | 3, 5, 6, 8, 9, 11, 12, 14, 15, 19, 20, 21 (11 of 21, per `009`) | **Case 9** — 4 Machines, "strongest evidence" per 009's own table |
| Within-Machine auxiliary-view nav | CAP-O03 Tier 3 ❌ (this study feeds it) | any case below with ≥1 auxiliary view | same cases as those rows |
| Board / kanban | CAP-V14 Tier 2 ✅ | 19 | **Case 19** |
| Manual ordering (Up/Down) | CAP-V14 ✅ | 19 | **Case 19** |
| Calendar (single-dim) | CAP-V07 ✅ | 20 | **Case 20** |
| Resource-grouped calendar (2-dim) | CAP-V18 ✅ | 20 | **Case 20** (same case, richer cut) |
| Aggregate report (group-by) | CAP-V13 ✅ | 9 (Trial Balance) | **Case 9** (same case as sideways nav — deliberate overlap, see below) |
| Composed dashboard | CAP-V10 ✅ | 10, 12, 13 (public variant) | **Case 12** (concrete tiles: Group/Event/Points/Badge) + **Case 13** (public/unauthenticated variant, materially different chrome) |
| SLA/deadline countdown badge | CAP-V17 ✅ | 7, 17 | **Case 7** |
| Live aggregate on child-line rows (Form) | CAP-V15 ✅ | 9 (debit=credit) | **Case 9** (third reuse of the same case) |
| Live cross-record balance preview (Form) | CAP-V19 ✅ | 5, 6 | **Case 6** (Fund balance — smaller, cleaner than Case 5's UoM conversion) |
| Typeahead reference/user picker | CAP-V16 ✅ | most cases with a `reference` field | **Case 6** (same case, reused) |
| Public/unauthenticated read access | CAP-P07 ✅ | 13 | **Case 13** (same case as public dashboard) |
| Sensitive-field visibility (per-role redaction) | CAP-P06 ✅ | 20 | **Case 20** (same case as calendar) |
| "Document" auxiliary view | — (named in `009`'s follow-on list; **no `ViewType`, no `.templ`, no CAP row exists for it** — checked directly against `internal/ui/*.templ` and `capability-registry.md` for this study) | none | **not designable yet — flagged below, not silently designed around** |

Seven cases cover every real cluster at least once: **9, 19, 20, 7, 6, 12, 13**. That is the
representative set — not all 21, not just 1.

**Correction (2026-08-23): now eight.** Case 3 (Document Approval) added on direct owner
request — see this file's own 2026-08-23 Phase 2 update below for why (a sequential
multi-approver decision stepper, a UI shape none of the original seven's clusters names).

**Named exclusion, not an oversight:** "Document" appears in `009`'s own follow-on-finding list
of auxiliary View types alongside Calendar/Timeline/Report/Dashboard/Board, but grepping
`internal/ui/*.templ` and `capability-registry.md` for it (this study) finds no `ViewType`, no
`.templ` file, and no CAP row — it was named there as a category label, not a shipped or even
registered concept. CAP-F21 (certificate file rendering, Case 21) is the nearest real thing, but
it's a *field* rendering (a generated-file link on a Detail page), not a distinct auxiliary View
with its own route the way Board/Calendar/Report/Dashboard are. This plan does not invent a
Document View to design around — if a future case forces one into existence per the registry's
own admission process, it becomes an eighth cluster then, not now.

---

# Why Case 9 carries three clusters

Not a shortcut — Case 9 (Accounting) is independently the richest representative for sideways
nav, aggregate report, *and* live child-line aggregation, the same way `009` already flagged it as
"the case the original UI-workflow question itself was raised against." Designing it once and
reading off all three answers is the reuse the owner is asking for actually happening, not
avoided: the sideways-nav strip, the Report page, and the Form's live debit=credit total need to
visually cohere on **the same case's own screens**, which is a stronger reuse proof than three
separate cases that never have to agree with each other.

---

# Phased plan

Mobile-first per direct owner instruction (`009`'s follow-on finding, `roadmap.md` Track G) —
every phase designs the phone viewport first, tablet/desktop second, informed by what the phone
pass establishes rather than a retrofit.

## Phase 1 — Core chrome, on Case 9

Answers CAP-O03 Tier 3's own open questions (`capability-registry.md` CAP-O03 Tier 3 row):
global nav bar, `subNavBar`'s existing sideways-nav strip (today desktop-only, unchecked at phone
width), and the new within-Machine auxiliary-view control (List ↔ Report, since Case 9 has both).
Deliverable: List/Detail/Form/Report screens for Journal Entry + Chart of Account, phone width
first, with the two nav axes resolved into a standard (not stacked into two rows if it can be
helped) and named (bottom tab bar / hamburger / sheet — per `009`'s own question list).

## Phase 2 — Auxiliary view shapes, applying Phase 1's standard

Board (Case 19), Calendar + resource-grouped calendar (Case 20), composed Dashboard both
authenticated (Case 12) and public/unauthenticated (Case 13). Each must reuse Phase 1's nav
standard, not invent its own — the point of this phase is proving the standard holds under a
board's own toolbar, a calendar's own date-nav controls, and a dashboard's section tiles, all at
phone width. A phase that needs to bend the standard is itself a finding, written up not silently
absorbed.

## Phase 3 — Form/List decoration patterns

SLA badge (Case 7), live balance preview + typeahead picker (Case 6). Smaller in scope than
Phases 1–2 (these decorate existing List/Form rows, they don't add a route) but still unchecked
at phone width today — badge/picker touch targets and truncation behave differently on a small
screen than the current desktop-table layout assumes.

## Phase 4 — Extract the shared vocabulary

Write up, from what Phases 1–3 actually produced (not decided in advance): spacing scale,
breakpoint(s), icon-vs-label rule for nav controls, active-state styling, touch-target sizing.
Deliberately **not** touching color/branding — `capability-registry.md`'s Theme row is closed
("zero cases show per-workspace visual branding evidence") and this study finds nothing to
reopen it with; reuse the existing single Tailwind theme as-is.

## Phase 5 — Owner review, then implementation

Each phase's artboards go up as a `design`-skill canvas Artifact — click-to-select, comment,
iterate directly, not a round-trip through prose. Only after the owner has reviewed/refined the
full set does Track G's actual implementation begin (CAP-O03 Tier 3 build) against the resulting
written standard, per the roadmap's own ordering.

---

# Tooling decision

Mockups are built as `design`-skill canvas Artifacts, one per phase, mobile artboard(s) first —
not static text wireframes. Reasoning: the owner's own refine loop (click-to-select, properties
panel, inline edit) is a better fit for iterating on a **shared component standard** than
re-describing each change in prose; it also makes reuse visible directly — the same nav-strip
artboard component literally duplicated across Case 9/19/20/12/13's mockups is stronger proof of
reuse than four separately-written descriptions that merely claim to agree.

---

# Where the resulting standard document lives

Not decided by this plan alone — flagged now so it isn't silently dropped once Phase 4 lands.
The clustering/plan (this document) stays in `benchmarks/` at root, same as `008`/`009`/`020`,
because it's discovery/governance work over the portfolio. The **output** — a written
component-placement standard expressed in this prototype's own terms (`internal/ui/*.templ`
names, Tailwind classes, `subNavBar` extensions) — is Go-prototype implementation guidance, not
cross-prototype governance, so per `CLAUDE.md`'s doc-placement rule it belongs under
`prototype/go/docs/` (likely as a new ADR, `prototype/go/docs/decisions/008-*.md`, following the
existing `001`–`007` numbering) rather than at root. Confirm this when Phase 4 actually has
content to place.

---

# Explicit non-goals

- Not redesigning all 21 cases — 7 representative cases cover every real cluster; the other 14
  compose the same shapes (`case-portfolio.md`'s own novelty column already says as much for
  16–18, 21).
- Not a Theme/branding pass — closed, no case evidence, not reopened by this study.
- Not the harder half of CAP-V18 (live drag-to-reschedule) — already named and deferred in `008`.
- Not implementation — every phase produces mockups/standard only; `internal/ui/*.templ` is not
  touched until Phase 5 hands off, per the owner's own instruction in `009`'s follow-on finding.

---

# Next step

Phase 1 (Case 9 core chrome, phone-width-first) is ready to start — no open questions block it.
Whoever (or whichever session) picks this up next should produce that Artifact first, not the
written standard — the standard in Phase 4 is derived from what the mockups settle, not decided
ahead of them.

---

# Update (2026-08-23) — Phase 1 done: direction chosen, nav pattern resolved

Three genuinely different directions were sketched for Case 9's mobile List screen (owner asked
directly for a stated target look, not just a mechanical nav fix) — **A: Utilitarian** (matches
today's real Tailwind app closely), **B: Calm Structured** (bottom sheet combining both nav axes),
**C: Technical Ledger** (monospace figures, one chip-strip combining both axes). **Owner picked
Direction A.** B/C kept on the canvas, unchosen, for reference.

Built out into the full Phase 1 deliverable — Case 9 List/Detail/Form/Report, mobile (390×844),
in Direction A's visual language (`internal/ui`'s own slate/blue Tailwind tokens, lifted from
`layout.templ`/`list.templ`/`report.templ`/`components.templ` directly, not invented). This
answers CAP-O03 Tier 3's own open questions concretely:

- **Bottom tab bar** = cross-Machine sideways nav (CAP-O03 Tier 2's mobile form) — sibling
  Machines in the Application (Chart of Account / Journal Entry / Fiscal Period for Case 9).
- **A small segmented pill under the page title** = within-Machine view-type nav (CAP-O03 Tier 3)
  — shown on List and Report (a Machine's own collection-level pages), each linking to the other.
- **Detail keeps the tab bar, drops the pill** — a single record isn't a "view" to switch between,
  the pill's job doesn't apply there.
- **Form drops the tab bar entirely**, replaced by explicit Cancel/Save — a focused task,
  deliberately not a place to navigate sideways away from mid-entry.
- Two nav rows never stack on top of each other on any one screen — the open question that most
  worried `009`'s follow-on finding resolves cleanly once Detail/Form are read as not needing the
  within-Machine pill at all.

Also carries the live CAP-V15 aggregate footer into both its balanced (Detail) and unbalanced
(Form) states, and CAP-V13's Trial Balance restacked from the desktop 3-column table into grouped
mobile cards.

Canvas: `https://claude.ai/code/artifact/d8285b9f-6689-44c2-a8a0-6692ec724ab1`, page "Runtime UI —
Case 9" (added to the existing Menata Apps Builder canvas per owner's direction to continue there,
not a separate artifact — the Authoring Layer page concepts stay on their own page, unchanged).

**Not yet done**: Phase 2 (Board/Calendar/Dashboard on Cases 19/20/12/13, applying this same
resolved standard), Phase 3 (SLA badge/live balance preview on Cases 7/6), Phase 4 (write the
extracted standard up as a doc, per this file's own "Where the resulting standard document lives"
section). This update covers Phase 1 only.

---

# Update (2026-08-23) — Phase 2 done, Case 3 added on direct request

**Case 3 (Document Approval) added as an 8th representative case**, on direct owner request —
a real user need right now, not just cluster-coverage completeness. Its own UI shape wasn't
covered by the original 7: none of List/Detail/Form/Board/Calendar/Report/Dashboard names a
**sequential multi-approver decision stepper** (`approval-step.menata`'s own `Sequence` +
`Decision` fields, `Activate Next Step` action) — a genuinely new component, not a re-skin of
Case 9's Detail.

Five Phase 2 screens built, all mobile, Direction A:

| Case | Screen | Standard from Phase 1 | Result |
|---|---|---|---|
| 3 | Approval — vertical Approval Progress stepper (done/current/pending steps), sticky Approve/Reject | tab bar + no pill (record-level, same rule Case 9's Detail already established) | Holds. New component: the stepper itself, nothing in Phase 1 named it |
| 19 | Board | tab bar + List/Board pill | Holds. Lanes reuse `board.templ`'s own horizontal-scroll shape directly — already mobile-native, no adaptation needed |
| 20 | Resource Calendar (CAP-V18) | tab bar + List/Calendar pill | Holds, but the tab bar grows to 4 items (Patient/Visit/Record/Rx) — still fits at 390px; a 5th sibling Machine in some future case is the thing to watch |
| 12 | Dashboard | — | **Deviation, named**: no within-Machine pill — a composed dashboard isn't List/Report on one Machine, so Phase 1's pill has nothing to bind to. Tiles link out to each Machine's own List instead |
| 13 | Public landing (CAP-P07) | — | **Deviation, named**: no tab bar, no avatar, no notifications — a visitor has no "app" to switch inside, so both nav axes are genuinely absent, not merely hidden |

Two real, load-bearing findings for Phase 4's eventual standard write-up: (1) the within-Machine
pill is a List/Report-only device, not universal — Dashboard and any future non-collection view
type won't have one; (2) authenticated chrome (tab bar, avatar, notifications) is conditional on
CAP-P07's own authenticated/public distinction, not just a themeable toggle — a public page is a
structurally different shell, not the same shell with elements hidden.

Canvas updated in place (same artifact, page "Runtime UI — Case 9", two new rows). **Not yet
done**: Phase 3 (Cases 7/6 — SLA badge, live balance preview), Phase 4 (write the standard up,
now with 8 cases' worth of evidence instead of 1).

---

# Update (2026-08-23) — Phase 3 done

Two decoration-level screens, smaller in scope than Phases 1–2 (neither adds a route, both
decorate an existing List/Form):

| Case | Screen | Result |
|---|---|---|
| 7 | List with SLA badge (CAP-V17) | **Confirms an existing rule, doesn't add one**: Complaint has no sibling Machine (`case-portfolio.md`'s own note — "one Machine"), so the real `subNavFor` already returns `nil` and no tab bar renders; no within-Machine pill either, since Complaint declares no auxiliary View beyond List. The mockup's chrome is bare on purpose |
| 6 | Form with live cross-record balance preview (CAP-V19) + typeahead Fund picker (CAP-V16) | Shown deliberately PAST the limit it warns about (Amount > Fund's Current Balance) — proves the preview surfaces the CAP-C08 violation before submit, not just the happy path. Same focused-task Form rule as Phase 1: no tab bar, Cancel/Save |

No new deviations from the standard — both screens land exactly where Phase 1's own rules
already predicted (single-Machine → no tab bar; Form → no tab bar regardless). All 8
representative cases (3, 6, 7, 9, 12, 13, 19, 20) now have at least one mobile mockup on the
canvas.

**Not yet done**: Phase 4 — write the standard up as a doc (this file's own "Where the resulting
standard document lives" section already names the likely location, `prototype/go/docs/decisions/`
as a new ADR). Phases 1–3 are the design work; Phase 4 is the only remaining step before Track G's
CAP-O03 Tier 3 implementation can start.
