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
| 3 | Approval inbox (Worklist + Detail) | `document-approval.html` | List (`vw_ad_all`/`vw_ad_pending` cards) + Detail (`vw_ad_detail`) + progress (`DecisionStepper`, `/mch_approval_document/{id}/progress`) | **SLA wiring done, 2026-09-12 (17r, Stage A item 1).** `mch_approval_document` gained a real `fld_ad_due_date` Field; `sla_field`/`sla_warning_days` wired on `vw_ad_all`/`vw_ad_pending`/`vw_ad_detail` — real OVERDUE/"N day(s) left" badges now render on both List and Detail, matching the mockup directly. **Real bug found and fixed the same day** (not Case-3-specific — see `capability-registry.md`'s `CAP-V17` row): the real cards-mode List route (17g) never actually considered `SlaField` at all before this, and a per-record Status-badge fallback was needed so a Document with no due date (the common case) still shows its own Status badge instead of none. **Remaining, unrelated to SLA:** the mockup's inline PDF preview alongside the action bar is still not confirmed to exist as one layout — the real file field only renders as a download link (`record_crud.go`'s `FieldTypeFile` case), not an inline viewer. | **List: production** (17g, SLA badge now included). **Detail: production** (17o closes `CR-29`). **DecisionStepper progress: not composable at all.** | Inline PDF preview layout (classic gap, unverified how large — Stage A item 2); DecisionStepper composable representation (composable gap, Stage B); a small, named cosmetic gap in the SLA-fallback case (the fallback badge value can also still appear in the card's own subtitle text — see `CAP-V17`'s own row for the full account). |
| 4 | Approval dashboard (Composed page) | `approval-dashboard.html` | `GET /mch_approval_document/page` (`vw_ad_page`) | **Built** for all 4 of its own real sections (Summary/Pending Documents/Recent Activity/Total Approval Steps). **Recent Activity is real now** (17p, closing CAP-R04's own "R28" gap) — a real cross-record activity feed, not a placeholder. | **2 sections are production composable** — 17l's own "Total Approval Steps" Metric, and note that Recent Activity (17p) is classic-only, not composable (no `internal/composable` representation for `activity_log` yet — a separate, later increment). | Field-level diff between snapshots (17p's own named, deferred gap); composable representation for `activity_log` (separate, unforced work); the Metric section still doesn't map to anything ui-sample asked for, named so it's never mistaken for design-completeness progress. |

## 3. Project Management (Case 19)

Ground truth: `/ui-sample/case-19.html`'s own explicit screen list (read directly, 2026-09-12) —
its own words: "Ten screens are represented by the two existing core mockups plus these
composable view explorations." Two core + six exploratory, all linked from this one page (the
valid entry point per `app/CLAUDE.md`'s own ui-sample rule).

### Core (2)

| # | Screen | Mockup file | Maps to (real) | Classic capability | Composable status | Gap to close |
|---|---|---|---|---|---|---|
| 1 | Project board (Kanban) | `project-board.html` | `GET /mch_pm_card/board` (`vw_pmc_board`) | **Done, 2026-09-12 (17q).** Re-grounded directly against the mockup's own real card markup (not the earlier paraphrase above, kept struck through for the record): Labels are shared, reusable, colored (Design/blue, Frontend/emerald, High/rose, QA·Research·UX/violet, Planning/amber) and Members are multi-person per card — both real, both now built. `mch_pm_card` gained `fld_pmc_due_date` (date); two new join Machines (`mch_pm_card_member`, `mch_pm_card_label`) + a new master-data Machine (`mch_pm_label`) close the relationship gaps, composing entirely from already-supported Grammar (checked directly against `capability-lifecycle.md`'s A4 — no admission test needed, see below). Board's own card now renders real colored label chips, member avatars, checklist progress ("N/M"), and the due date (`seeds/057_case19_card_fields.sql`, conformance T291-T295). Still not built: richer drag semantics (reorder within a lane) — classic UI/JS work, unrelated to any of the above, still open. | **Production** for the lane-grouping (17n) AND the new per-card metadata (17q) — both render through the same composable Board route (`boardViaComposable`), the metadata via a new opt-in `CardMeta` Config key (`model.BoardCardMetaConfig`), same "explicit opt-in, not automatic detection" posture `CAP-V17`/`CAP-V18` already established. | Richer drag semantics only — everything else this row named is closed. ~~Missing Fields (label, assignee, due date) — classic, blocks the mockup regardless of architecture~~ (closed, 17q). |
| 2 | Card detail | `project-card.html` | `GET /mch_pm_card/{id}` (`vw_pmc_detail`) + embedded Checklist (CAP-F16 `ChildLines` on the Card's own Form, CAP-V06 reverse-reference list on Detail) + embedded Activity (17p) + embedded Members/Labels (17q) | **Done, 2026-09-12 (17q).** Due date renders automatically via Detail's own existing generic field loop (zero new Detail code, `CR-29`'s own discipline preserved). Members/Labels render automatically via CAP-V06's existing generic `childLists` — each its own titled reverse-reference section, resolved to REAL names ("Design", "Project Owner"), not raw ids — a real, generic gap in `childLists`' own label resolution was found and fixed the same day (`childListItemLabel`, `formfields.go`): a join Machine with no Text/Number Field used to fall back to the bare record id; it now also tries that row's OTHER `user`/`reference` Field, same "generic by construction" posture `childLists`' own doc comment already claims. **The activity feed itself still always renders empty against today's real data** — `mch_pm_card` has zero declared Events (unchanged, a fact about that Machine's own metadata, not a flaw). | **Production** (17o's Detail cutover, unchanged — Members/Labels/Due date all reach the page through mechanisms 17o/17p already proved, no new composable work needed for this row). | Nothing classic left for this screen. A cosmetic gap remains, named not hidden: Members/Labels render as titled reverse-reference blocks, not the mockup's own compact single `Members: Raka Aditya · Andi Nur` line — deliberately not special-cased into the shared generic Detail handler for one Machine's own layout. |
| — | Checklist (no mockup file — `case.html`'s own screens array lists this third entry with no linked file) | — | `mch_pm_checklist_item`, embedded via the Card's own Form/Detail | **Built.** | Not composable at all (no pilot, no production — untouched). | Composable representation only, if ever forced. |

### Exploratory — "composable view explorations" (case-19.html's own label; real, named, but not a hard requirement the way the two core screens are)

| # | Screen | Mockup file | Classic capability | Note |
|---|---|---|---|---|
| 3 | Timeline / Roadmap | `project-timeline.html` | **Not built.** No Timeline-type View declared for any PM Machine. | Needs a new View declaration at minimum; `model.ViewTypeTimeline` already exists generically (used elsewhere, e.g. Action Lab) but has never been wired to Project Management. |
| 4 | Calendar | `project-calendar.html` | **Not built** for PM. `model.ViewTypeCalendar` exists generically elsewhere. | Needs a date Field on a PM Machine + a real Calendar View declared. |
| 5 | Sprint Dashboard | `project-dashboard.html` | **Not built.** No Dashboard View declared for any PM Machine. | Needs a Dashboard View declaration; the underlying `DashboardSection` mechanism already exists generically. |
| 6 | Team Capacity | `project-team.html` | **Not built** — but its own blocker (a real Members relationship) closed 2026-09-12 (17q, see Core row #1/#2). Building the actual capacity/allocation aggregation view itself is separate, unscoped work — a real Members join existing is necessary, not sufficient. | No longer blocked on a missing Field; still needs a real Dashboard/aggregation View declared against `mch_pm_card_member`, plus admission if that aggregation shape is new. |
| 7 | Workflow Automation | `project-automation.html` | **Partially built** — Events/Actions (CAP-E*/CAP-A*) exist as a generic mechanism; no PM-specific automation authoring screen exists. | Needs a UI surfacing trigger→condition→action specifically for PM, if ever admitted — not a data-model gap, a UI-surface one. |
| 8 | Board Settings | `project-settings.html` | **Not built.** No dedicated settings screen; Permissions exist generically (real `Permission` records) but with no UI matching this mockup's own statuses/labels/custom-fields/permissions grouping. | Needs a real settings screen; the underlying Permission mechanism it would configure already exists. |

## 4. What this means, stated plainly

- **Case 3:** 3 of 4 screens now have at least one real composable slice (List production, Detail
  production as of 17o, one Metric); the other screen (Submit) and one sub-part of screen 3
  (DecisionStepper progress) have never been touched by any composable work at all. Signature
  Positions (screen 2) also remains untouched. Every screen still has at least one real, classic
  (non-composable) gap against its own ui-sample mockup.
- **Case 19:** both core screens now have a real composable slice (Board's lane-grouping, Card
  Detail as of 17o). **Status update (2026-09-12, 17q):** both core screens' own remaining Field
  gaps (label, member, due date) are closed too — Labels/Members/Due Date all render real, on both
  Board and Card Detail, matching the mockup's own real Trello-shaped multi-value relationships.
  Only richer drag semantics (screen 1) and the compact single-line Detail layout (screen 2,
  cosmetic) remain open for the two core screens. All 6 exploratory screens are still entirely
  unbuilt in classic code — composable status does not even apply to them yet (Team Capacity,
  screen 6, is unblocked as of 17q but still unbuilt itself).
- **Shared blocker, closed 2026-09-12 (17p):** CAP-R04 (activity/history timeline, "R28") is
  admitted and built — real for Case 3's own two screens (Approval inbox's Detail history,
  Approval dashboard's Recent Activity), mechanically real but legitimately empty for Case 19's
  own Card Detail (no declared Events on `mch_pm_card` yet — a separate, still-open gap).

## 5. Suggested staged order

Staged so each stage only depends on the one before it. **Status update (2026-09-12):** items 1-3
below were Case-19-only; after the owner asked directly whether everything is done (§7) and then
asked for a plan to finish the rest, this section's own remaining items were replaced with a
fuller staged plan covering BOTH cases' own outstanding work (full plan:
`/root/.claude/plans/goofy-puzzling-oasis.md`, approved this same conversation) — items 4+ below
supersede the old Case-19-only "New View types"/"Automation/Settings"/"Re-run composable cutovers"
wording, kept only in spirit, not verbatim (the old wording undersold Case 3's own remaining gaps
entirely).

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
3. ~~**Close Case 19's core-screen Field gaps** — assignee/member Field, label Field, due-date
   Field on `mch_pm_card` (also admission-gated).~~ **Done, 2026-09-12 (17q)** — re-grounded
   directly against `project-board.html`/`project-card.html`'s own real markup first (Members and
   Labels are both real MULTI-value, Trello-shaped relationships, not single-value Fields as this
   line originally assumed). **Correction on "admission-gated":** checked `capability-lifecycle.md`
   §2's A4 directly rather than assuming — every piece composes entirely from already-Supported
   Grammar (a plain date Field; two join Machines reusing CAP-V06's existing generic
   reverse-reference discovery; a master-data Machine reusing CAP-O02's existing pattern), which is
   exactly what A4 excludes from "new capability" — no admission test was actually run or needed.
   One real capability WAS extended (registered by citation on its own existing row, not a fresh
   one, same posture `CAP-V17` already used): `CAP-F16` (`ChildLines` → plural `ChildLinesGroups`,
   a Form can now embed more than one child-row block) and `CAP-V14` (Board gains an opt-in
   `CardMeta` rendering key). Unblocked Team Capacity (screen 6) as a side effect — the real
   relationship now exists, though the screen itself is still unbuilt.
4. **Stage A — small, low-risk, classic-only fixes, independent of each other:**
   ~~SLA wiring on Approval Document~~ **done, 2026-09-12 (17r)** — see row 3's own update above
   and `capability-registry.md`'s `CAP-V17` row for the real gap found and fixed along the way
   (the production cards-mode List route never actually considered `SlaField` at all until now).
   Remaining Stage A items, not yet started: inline PDF preview (Case 3 screen 3, ground exact
   size first); Board drag-reorder within one lane (Case 19 screen 1, client-side + a `sort_order`
   write, same shape `MoveToLane`/`Move` already use).
5. **Stage B — composable-representation-only gaps** (no new classic capability, confirmed by grep
   that `internal/composable` has zero references to any of these three today): `CoordPlacement`
   (Case 3 screen 2), `DecisionStepper` progress (Case 3 screen 3), `activity_log` (Case 3 screen
   4) — each a new Dataset/Component case, same shape already proven for other CAP-V20-style Views.
6. **Stage C — field-diff on Activity** (Case 3 screen 4): `record_events.snapshot` already
   stores the full pre-mutation record on every real Event fire (confirmed by reading
   `executor.go` directly) — this needs a diff computation over already-collected data, not new
   collection. Likely needs a real design decision (which fields, how to format a change) before
   building.
7. **Stage D — Case 19's remaining exploratory View types** (Timeline/Calendar/Dashboard, screens
   3-5, plus Team Capacity screen 6): re-check each against `capability-lifecycle.md`'s A4 before
   building anything, the same way Stage 3 (17q) corrected its own "admission-gated" guess — the
   underlying View types/mechanisms already exist and are already Supported elsewhere, so wiring
   them to Project Management is likely ordinary metadata, not a fresh capability, but this must
   be checked directly per screen, not assumed.
8. **Stage E — UI-surface work over existing generic mechanisms** (Automation screen 7, Board
   Settings screen 8): the one place in this whole remaining scope that may need a REAL
   `capability-lifecycle.md` A1-A5 run (a live trigger→condition→action AUTHORING UI has no
   precedent anywhere in this runtime — Events/Actions are always seed/metadata-time authored
   today) — ground that question first, don't assume either way. Lowest-confidence sizing of any
   stage; a real, unscoped design effort.
9. **Stage F — Submit-wizard unification** (Case 3 screen 1), sequenced last deliberately as the
   single largest, most uncertain item: CAP-V12's wizard creates exactly one record today: merging
   in a second Machine's own rows (Approval Steps) is a real mechanism question, not just a new
   View. May turn out to reuse 17q's own new `ChildLinesGroups` plural mechanism once grounded —
   benefits from every earlier stage's own findings first.

Composable cutovers for whatever Stage A/D/F add are folded into each of those stages directly
(build classic, prove composable, same step's own increment) rather than swept up at the very
end — Stage B above already covers the composable-only backlog that predates this update.

## 6. Cross-references

- `composable-runtime-roadmap.md` §2 (Gap Register) — `CR-29` (Detail primitive), `CR-05`
  (Logical Query execution semantics) are the composable-side blockers named above.
- `app/CLAUDE.md`'s "ui-sample" section, `composable-runtime-roadmap.md`'s Principle 11 — why
  this document exists and the rule it enforces going forward.
- `capability-lifecycle.md` §2 (A1-A5) — the admission test any NEW classic capability in §5
  above must pass before being built; `capability-registry.md`/`roadmap.md` are where that
  admission gets tracked once it happens.

## 7. Outstanding items (not yet done, 2026-09-12)

A consolidated, itemized answer to "is everything done yet" — each item below already has its own
evidence/citation in §2/§3/§5 above; this section exists so the answer doesn't require piecing
those table cells back together by hand.

**Document Approval (Case 3):**
- Submit document (screen 1) — the mockup's one wizard vs. the real code's two separate
  submissions (Document, then each Approval Step) is a real, unclosed UX-flow gap; this screen has
  never been touched by any composable work either.
- Signature positions (screen 2) — built classically, but has zero `internal/composable`
  representation at all.
- Approval inbox (screen 3) — SLA wiring done, 2026-09-12 (17r); still no inline PDF preview (file
  fields only render as a download link); `DecisionStepper` progress has no composable
  representation.
- Approval dashboard (screen 4) — no field-level diff between activity snapshots yet; no
  composable representation for `activity_log`.

**Project Management (Case 19):**
- Project board (screen 1) — richer drag semantics (reorder within one lane, not just move
  between lanes) still classic UI/JS work, not started.
- Card detail (screen 2) — cosmetic only: Members/Labels render as titled reverse-reference
  blocks, not the mockup's own compact single-line `Members: Raka Aditya · Andi Nur` format.
- Timeline / Roadmap (screen 3) — not built at all; no View declared.
- Calendar (screen 4) — not built at all; no View declared.
- Sprint Dashboard (screen 5) — not built at all; no View declared.
- Team Capacity (screen 6) — unblocked as of 17q (a real Members relationship now exists) but the
  actual aggregation/capacity View itself is still not built.
- Workflow Automation (screen 7) — the generic Event/Action mechanism exists; no PM-specific
  automation-authoring UI exists.
- Board Settings (screen 8) — not built at all; no dedicated settings screen.

**Net: Stages 1-3 of §5's own 6-stage staged order are done (17o/17p/17q); Stages 4-6 — the
exploratory View types, Automation/Settings screens, and their own eventual composable
cutovers — have not been started.**
