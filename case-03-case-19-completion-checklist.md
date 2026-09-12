# Case 3 & Case 19 Completion Checklist — Real Status vs. `/ui-sample/`

> **Status:** Active tracking document
> **Created:** 2026-09-12
> **Scope:** ONLY the two trial applications named in `composable-apps-trial.md` §6 — Document
> Approval (Case 3) and Project Management (Case 19). No other seeded application (there are
> ~30 "lab" applications proving unrelated capabilities) is in scope here.

## 1. Purpose

This document exists because a real, owner-flagged problem kept recurring across sessions
(recorded in `app/CLAUDE.md`'s "ui-sample" section and `composable-runtime-roadmap.md`'s
Principle 11, both dated 2026-09-12): work kept treating the CURRENT, incomplete code rendering
as if it were the design target, instead of checking `app/web/static/ui-sample/` (served live at
`/ui-sample/`) first. This document is the concrete fix — a complete, per-screen inventory,
grounded directly against the real `ui-sample` mockup files (not against the current code), so
future work can proceed "bertahap" (staged) against a real checklist instead of an implied one
that turns out not to exist when checked.

**Two independent axes are tracked per screen, never conflated:**
- **Classic capability** — does the real Go runtime (`internal/model`, `internal/handler`,
  seeded metadata) even have the Fields/View/feature needed at all, regardless of architecture.
- **Composable status** — `not started` / `pilot only` (proven on an additive
  `/composable-preview` route) / `production` (the real route itself cut over,
  `composable-runtime-roadmap.md`'s own 17-series).

Closing "composable status" is impossible without "classic capability" existing first. Closing
"classic capability" does not by itself close "composable status." Every row below states both,
separately, with evidence.

## 2. Document Approval (Case 3)

Ground truth: `/ui-sample/case.html?case=3`'s own `screens` array (read directly from the file,
2026-09-12) — four screens, each with a description and a linked mockup file.

| # | Screen (ui-sample) | Mockup file | Maps to (real) | Classic capability | Composable status | Gap to close |
|---|---|---|---|---|---|---|
| 1 | Submit document (Wizard/Form) | `document-submit.html` | `vw_ad_form`, `POST /mch_approval_document` | **Built**, but structurally split: the mockup bundles metadata + PDF + mode + per-step assignee + step reordering + saved-flow selection into ONE wizard; real code splits submitting the Document (`vw_ad_form`) from creating each Approval Step (`vw_as_form`, one POST per step). A real, separate "Approval Flow Template" mechanism (`seeds/047_approval_flow_template.sql`, `mch_approval_flow_template`/`_step`) does cover reusable/saved flows by document type, confirmed — just not surfaced inside one screen. | Not composable — `model.ViewTypeForm` has a composable Dataset case (`BuildDatasetFromView`) but nothing renders a Form through it; this whole screen is untouched by any 17-series work. | Real UX-flow gap (one wizard vs. two separate submissions) independent of composable; no dedicated wizard screen exists. |
| 2 | Signature positions (Coordinate editor) | `document-signature-placement.html` | `CoordPlace`, `GET/POST /mch_approval_document/{id}/place` (CAP-V21) | **Built.** | **Not composable at all** — `internal/composable` has zero representation for `model.ViewTypeCoordPlacement`; no Dataset case, no Component. | Composable-representation gap only; visual/functional gap not checked here. |
| 3 | Approval inbox (Worklist + Detail) | `document-approval.html` | List (`vw_ad_all`/`vw_ad_pending` cards) + Detail (`vw_ad_detail`) + progress (`DecisionStepper`, `/mch_approval_document/{id}/progress`) | **Built**, with one confirmed gap: no `sla_field` is configured anywhere on this Machine's own views (grepped directly), so the mockup's own "SLA chips" are not wired for this case at all (CAP-V17 exists generically elsewhere, e.g. SLA Lab, just not applied here). The mockup's inline PDF preview alongside the action bar is also not confirmed to exist as one layout — the real file field only renders as a download link (`record_crud.go`'s `FieldTypeFile` case), not an inline viewer. | **List: production** (17g). **Detail: production** (17o closes `CR-29` — the real `/mch_approval_document/{id}` route now sources its own field set/order from `composable.BuildDatasetFromView`'s `ViewTypeDetail` case; per-field value formatting, which needs real store I/O, stays in the handler by architectural necessity). **DecisionStepper progress: not composable at all.** | SLA wiring for this Machine (classic gap); inline PDF preview layout (classic gap, unverified how large); DecisionStepper composable representation (composable gap, still open). |
| 4 | Approval dashboard (Composed page) | `approval-dashboard.html` | `GET /mch_approval_document/page` (`vw_ad_page`) | **Built** for all 4 of its own real sections (Summary/Pending Documents/Recent Activity/Total Approval Steps). **Recent Activity is real now** (17p, closing CAP-R04's own "R28" gap) — a real cross-record activity feed, not a placeholder. | **2 sections are production composable** — 17l's own "Total Approval Steps" Metric, and note that Recent Activity (17p) is classic-only, not composable (no `internal/composable` representation for `activity_log` yet — a separate, later increment). | Field-level diff between snapshots (17p's own named, deferred gap); composable representation for `activity_log` (separate, unforced work); the Metric section still doesn't map to anything ui-sample asked for, named so it's never mistaken for design-completeness progress. |

## 3. Project Management (Case 19)

Ground truth: `/ui-sample/case-19.html`'s own explicit screen list (read directly, 2026-09-12) —
its own words: "Ten screens are represented by the two existing core mockups plus these
composable view explorations." Two core + six exploratory, all linked from this one page (the
valid entry point per `app/CLAUDE.md`'s own ui-sample rule).

### Core (2)

| # | Screen | Mockup file | Maps to (real) | Classic capability | Composable status | Gap to close |
|---|---|---|---|---|---|---|
| 1 | Project board (Kanban) | `project-board.html` | `GET /mch_pm_card/board` (`vw_pmc_board`) | **Built**, CAP-V14 Tier 3, a deliberately narrower cut (`capability-registry.md`'s own row says so) — fixed lane grouping only. Confirmed by direct field check: `mch_pm_card` has exactly `fld_pmc_list` (reference), `fld_pmc_title` (text), `fld_pmc_description` (rich_text) — **no label field, no assignee/member field, no due-date field.** The mockup's own labels/member avatars/checklist-count badge/free-position drag have no backing data to render even if composable supported them. | **Production** for the lane-grouping itself (17n). | Missing Fields (label, assignee, due date) — classic, blocks the mockup regardless of architecture; richer drag semantics (reorder within a lane, not just move between lanes) — classic UI/JS work, unrelated to composable. |
| 2 | Card detail | `project-card.html` | `GET /mch_pm_card/{id}` (`vw_pmc_detail`) + embedded Checklist (CAP-F16 `ChildLines` on the Card's own Form, CAP-V06 reverse-reference list on Detail) + embedded Activity (17p) | **Built** for description + checklist + activity mechanism. **The activity feed itself always renders empty against today's real data** — `mch_pm_card` has zero declared Events (a fact about this Machine's own metadata, not a flaw in the mechanism). **Not built at all:** assignee (no such Field), position/due-date (no such Field). | **Production** (17o — `Detail` is one shared handler across every Machine, so this route was cut over in the same change as Case 3's own Detail page; `CR-29` closed). | New `user`/assignee Field on `mch_pm_card` (classic, not yet admitted); new date Field for "position"/due date (classic, not yet admitted); at least one real Event on `mch_pm_card` needed before its own activity feed ever shows anything. |
| — | Checklist (no mockup file — `case.html`'s own screens array lists this third entry with no linked file) | — | `mch_pm_checklist_item`, embedded via the Card's own Form/Detail | **Built.** | Not composable at all (no pilot, no production — untouched). | Composable representation only, if ever forced. |

### Exploratory — "composable view explorations" (case-19.html's own label; real, named, but not a hard requirement the way the two core screens are)

| # | Screen | Mockup file | Classic capability | Note |
|---|---|---|---|---|
| 3 | Timeline / Roadmap | `project-timeline.html` | **Not built.** No Timeline-type View declared for any PM Machine. | Needs a new View declaration at minimum; `model.ViewTypeTimeline` already exists generically (used elsewhere, e.g. Action Lab) but has never been wired to Project Management. |
| 4 | Calendar | `project-calendar.html` | **Not built** for PM. `model.ViewTypeCalendar` exists generically elsewhere. | Needs a date Field on a PM Machine + a real Calendar View declared. |
| 5 | Sprint Dashboard | `project-dashboard.html` | **Not built.** No Dashboard View declared for any PM Machine. | Needs a Dashboard View declaration; the underlying `DashboardSection` mechanism already exists generically. |
| 6 | Team Capacity | `project-team.html` | **Not built.** No member/allocation Field or aggregation exists. | Blocked on the same missing assignee Field named in Card Detail's own row above. |
| 7 | Workflow Automation | `project-automation.html` | **Partially built** — Events/Actions (CAP-E*/CAP-A*) exist as a generic mechanism; no PM-specific automation authoring screen exists. | Needs a UI surfacing trigger→condition→action specifically for PM, if ever admitted — not a data-model gap, a UI-surface one. |
| 8 | Board Settings | `project-settings.html` | **Not built.** No dedicated settings screen; Permissions exist generically (real `Permission` records) but with no UI matching this mockup's own statuses/labels/custom-fields/permissions grouping. | Needs a real settings screen; the underlying Permission mechanism it would configure already exists. |

## 4. What this means, stated plainly

- **Case 3:** 3 of 4 screens now have at least one real composable slice (List production, Detail
  production as of 17o, one Metric); the other screen (Submit) and one sub-part of screen 3
  (DecisionStepper progress) have never been touched by any composable work at all. Signature
  Positions (screen 2) also remains untouched. Every screen still has at least one real, classic
  (non-composable) gap against its own ui-sample mockup.
- **Case 19:** both core screens now have a real composable slice (Board's lane-grouping, Card
  Detail as of 17o). Both core screens are still missing real Fields (label, assignee, due date)
  that block matching the mockup regardless of architecture. All 6 exploratory screens are
  entirely unbuilt in classic code — composable status does not even apply to them yet.
- **Shared blocker, closed 2026-09-12 (17p):** CAP-R04 (activity/history timeline, "R28") is
  admitted and built — real for Case 3's own two screens (Approval inbox's Detail history,
  Approval dashboard's Recent Activity), mechanically real but legitimately empty for Case 19's
  own Card Detail (no declared Events on `mch_pm_card` yet — a separate, still-open gap).

## 5. Suggested staged order

Staged so each stage only depends on the one before it.

1. ~~**Composable cutover for what already has real code** — Detail (both cases), closing
   `CR-29`'s own deferred production cutover.~~ **Done, 2026-09-12 (17o)** — `Detail` is one
   shared handler across every Machine in the runtime, so this single change closed it for both
   trial cases (and every other seeded Machine) at once. Proven by the full 292-test conformance
   suite passing unchanged (Detail pages across many real Machines/field types).
2. ~~**Close the shared classic blocker** — CAP-R04 (activity timeline), since it blocks a named
   screen in both cases simultaneously.~~ **Done, 2026-09-12 (17p)** — admission test run and
   recorded (`capability-registry.md`), one new `activity_log` View Type built dual-mode (record-
   scoped via CAP-V20, cross-record via CAP-V10 Tier 2). Case 19's own feed stays honestly empty
   pending real Events on `mch_pm_card` (not a flaw in this capability). No field-diff between
   snapshots yet — named, deferred.
3. **Close Case 19's core-screen Field gaps** — assignee/member Field, label Field, due-date Field
   on `mch_pm_card` (also admission-gated). Unblocks Team Capacity (screen 6) as a side effect.
4. **New View types for Case 19's remaining exploratory screens** — Timeline, Calendar, Dashboard
   (screens 3-5), each admission-gated, each usable generically once built (not PM-specific code).
5. **Automation/Settings screens** (7-8) — UI-surface work over already-existing generic
   mechanisms; lowest technical risk, but a real, unscoped design/build effort on its own.
6. **Re-run composable cutovers** for whatever Stage 2-5 adds, same "prove additively, then cut
   over" discipline every 17-series increment already used — never build a new classic feature
   and its composable equivalent in the same step.

## 6. Cross-references

- `composable-runtime-roadmap.md` §2 (Gap Register) — `CR-29` (Detail primitive), `CR-05`
  (Logical Query execution semantics) are the composable-side blockers named above.
- `app/CLAUDE.md`'s "ui-sample" section, `composable-runtime-roadmap.md`'s Principle 11 — why
  this document exists and the rule it enforces going forward.
- `capability-lifecycle.md` §2 (A1-A5) — the admission test any NEW classic capability in §5
  above must pass before being built; `capability-registry.md`/`roadmap.md` are where that
  admission gets tracked once it happens.
