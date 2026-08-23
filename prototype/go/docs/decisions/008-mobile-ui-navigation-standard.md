# ADR-008: Mobile-First UI Navigation & Component Standard

**Status:** Accepted and implemented, with one decision corrected before it shipped — see
"Correction (2026-08-23)" below before reading Decision #1 as originally written.
**Date:** 2026-08-23

**Update (2026-08-23, same day, before implementation): Decision #1 corrected.** The owner asked
directly, mid-implementation: if an organization runs several Applications (Document Approval,
Design Request, Project Management, ...) under one system, does a bottom tab bar that renames
itself every time a user crosses an Application boundary actually feel like one system? Checked
against world-class practice (`../../../../benchmarks/022-bottom-nav-consistency-benchmark.md`,
Study 30) — Material Design, Apple HIG, and Slack all hold a bottom bar's own identity **fixed**;
Salesforce's one partial exception is gated behind an explicit App Launcher action, not a passive
side effect of ordinary navigation. **The bottom tab bar does NOT carry `subNavFor`'s
per-Application data.** It is a fixed, global 3-item set — Home / Search / Notifications —
identical on every page, mirroring `navBar`'s own existing links; "Home" is this system's own App
Launcher (the pre-existing workspace home, `handler.Apps`). `subNavFor`'s cross-Machine data stays
exactly where it already worked: the existing top `subNavBar` strip, unchanged in scope. Every
other decision below (the within-Machine view pill, Detail/Form rules, Dashboard's exception, the
public-page exception) is unaffected — implemented as designed. **Implementation status: done**
— `Handler.viewNavFor` (`internal/handler/views.go`) + `ui.viewNavPill`
(`internal/ui/components.templ`) for the pill, `ui.shellBottomBar` (`internal/ui/layout.templ`)
for the corrected global bar. Conformance T186–T188
(`../../conformance/tests/090_mobile_nav.sh`).

## Context

`CAP-O03 Tier 3` (within-Machine navigation to a Machine's own auxiliary Views — Calendar,
Timeline, Report, Dashboard, Board) was registered as a real, owner-confirmed gap on 2026-08-22,
but the owner explicitly withheld implementation: UI/layout work needed a design-prototype pass
first, done **mobile-first** (phone viewport designed from scratch, not a retrofit of the
existing desktop chrome), and the standard needed to be derived from the whole
`case-portfolio.md`, not just the one gap that surfaced it — see
`../../../../roadmap.md`'s Track G and `../../../../capability-registry.md`'s CAP-O03 Tier 3 row
for the full requirement history.

`../../../../benchmarks/021-design-system-prototype-plan.md` (Study 29) is the plan that
followed: it clustered all 21 portfolio cases by UI shape rather than designing each
independently, selected **8 representative cases** (9, 19, 20, 7, 6, 12, 13, plus Case 3 added
mid-study on direct request — a real approval-management need, not just cluster completeness),
and phased the actual design work. That phased work — three owner-reviewed aesthetic/nav-pattern
directions, then mobile mockups for all 8 cases — is complete, published as a canvas Artifact
(`https://claude.ai/code/artifact/d8285b9f-6689-44c2-a8a0-6692ec724ab1`, page "Runtime UI — Case
9"). This document is that study's Phase 4: the standard extracted from what those 9 mockups
actually settled, written in this prototype's own terms so it can be implemented against
directly.

## Decision

**Two independent navigation axes, never stacked in the same chrome row.**

1. **Cross-Machine sideways nav** (existing `CAP-O03 Tier 2`, `subNavBar`/`subNavFor`) becomes a
   **bottom tab bar** at phone width, listing the current Application's sibling Machines exactly
   as `subNavFor` already resolves them (`Interpreter.ScopeFor`/`MachinesForApplication`,
   permission-trimmed via `Guard.CanRead`) — no new data source, a new rendering target for data
   `subNavFor` already produces. Already correctly suppressed below 2 machines (Case 7's own
   mockup — Complaint has no sibling Machine — confirms this rule needs no change).
2. **Within-Machine view-type nav** (`CAP-O03 Tier 3`, new) is a **small segmented pill** under
   the page title (e.g. `List | Report`, `List | Board`, `List | Calendar`) linking between a
   Machine's own View types.
3. **The pill appears only on a Machine's own collection-level pages** — List, Report, Board,
   Calendar/Timeline — **never** on Detail or Form. A single record isn't a "view" to switch
   between (Case 9's Detail, Case 3's Approval both keep the tab bar, drop the pill); a Form is a
   focused task and drops **both** axes, replaced by explicit `Cancel` / `Save` header actions
   (Case 9's Form, Case 6's Form) — deliberately not a place to navigate away from mid-entry.
4. **Composed Dashboard (`CAP-V10`) gets no pill** — it isn't List/Report on one Machine, so
   Tier 3's pill has nothing to bind to (Case 12). Its own tile grid, each tile linking out to
   that tile's Machine, is its navigation; this is a genuine gap in the two-axis model above, not
   an oversight — Dashboard sits outside both axes by construction.
5. **Public/unauthenticated pages (`CAP-P07`) are a structurally different shell, not the
   authenticated shell with elements hidden** (Case 13): no tab bar, no within-Machine pill, no
   avatar, no notifications bell. A visitor has no "app" to switch inside and no identity to show.
6. **Mobile list rendering**: List/Report/Calendar's desktop `<table>` (`list.templ`,
   `report.templ`, `calendar.templ`) restacks at phone width into bordered, rounded (`10px`),
   white **row cards** — one card per record, fields stacked vertically inside — rather than a
   horizontally-scrolled or truncated table. Board's existing horizontal-scroll lane layout
   (`board.templ`) needed no mobile adaptation at all — it was already phone-appropriate.
7. **Visual language is the existing app's, not a new one** — every color, badge pill shape
   (`StatusBadge`/`SlaBadge`'s own convention), border, and shadow in the mockups is lifted
   directly from `internal/ui/layout.templ`/`list.templ`/`report.templ`/`components.templ` and
   `tailwind.config.js`, confirmed against three deliberately different alternative directions
   (canvas page "Runtime UI — Case 9", "not chosen" row) before settling on this one. No new
   palette, font, or spacing scale is introduced by this standard.
8. **New component, not yet a capability**: Case 3's **Approval Progress stepper** (a vertical
   timeline over `Approval Step` records — done/current/pending, connected by a line, sticky
   Approve/Reject for the current approver) has no existing `ViewType` or Detail-page convention
   behind it. This ADR records its shape for implementation reference but does **not** itself
   register a new `CAP-V*` row — that admission (dual evidence: this case + a platform survey)
   is `capability-registry.md`'s own process, a separate step from this design standard.

## Implementation Strategy

**Superseded by the correction above — `subNav` does NOT feed the bottom tab bar.** What was
actually built: `subNavBar` (top strip) unchanged; a new, separate, unconditional
`shellBottomBar(unreadCount)` — Home/Search/Notifications, no `[]SubNavLink` input at all. Left
below for the record of what was originally planned.

- `ui.Page`'s existing `subNav []SubNavLink` parameter is the right data source for the new
  bottom tab bar — no new query, a responsive rendering change: render `subNavBar`'s current
  desktop strip above `sm:` (Tailwind's stock breakpoint, no custom breakpoint introduced) and a
  new fixed-bottom tab bar below it, from the same slice.
- A new, small `viewNavPill([]ViewLink, active)` component renders the segmented pill; each of
  the 8 Machine-scoped renderers that has more than one real View type on the same Machine passes
  it (List/Report/Board/Calendar renderers only, per Decision #3 — Detail/Form/WizardForm/
  ImportCSV pass nothing).
- List/Report/Calendar's row-card mobile layout is a CSS/breakpoint change to the existing
  `<table>` markup (a `<div>`-based card row shown `<sm:`, the current `<table>` shown `sm:` and
  up), not a second template — same discipline `CAP-V14 Tier 2`'s board already set (server-
  rendered, no JS framework).
- **Known gap to close during implementation, not carried forward from the mockups as-is**: the
  mockups' own icon-only buttons (top-bar search/bell, the `+` new-record button) render at
  30×30px for visual density — below this skill's own 44px minimum tap target. Real markup should
  size the actual tappable element (button box, not just the icon glyph) to at least 40px within
  the 48px header, not literally 30px.

## Consequences

- Two clean, independently-testable nav mechanisms instead of one that would have had to serve
  both jobs — avoids the "three stacked nav rows" failure mode `009`'s own follow-on finding
  worried about.
- Dashboard and public pages are named exceptions with a stated reason (Decision #4, #5), not
  silent special-casing discovered later during implementation.
- The mobile row-card list pattern is a real, if small, rendering fork from the desktop table —
  `list.templ`/`report.templ`/`calendar.templ` each need both branches maintained, the same
  tradeoff `CAP-V07`'s calendar/timeline dual-purpose templ already accepted for a similar reason.
- No `internal/ui/*.templ` file is touched by this ADR itself — it is design record only, same
  as `case-portfolio.md`'s own "declare targets first" discipline applied to UI instead of
  Business Knowledge.

## Compliance

Same HTTP black-box discipline the rest of `conformance/run.sh` already uses (`CAP-O03 Tier 2`'s
own T135 precedent): once implemented, a conformance test can assert the bottom-tab-bar markup is
present/absent by machine-count and page type, and that the view-nav pill is present only on
List/Report/Board/Calendar responses — a markup assertion, not a rendered-viewport screenshot
(this suite has no headless-browser tooling, the same limitation
`benchmarks/020-ui-interaction-cluster-proof.md` already named for CAP-V14 Tier 2/V15/V19's own
client-side pieces). Visual/responsive correctness itself was verified by the design-prototype
mockups this ADR describes, not by an automated test — that is what a design-prototype pass is
*for*.
