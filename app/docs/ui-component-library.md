# UI Component Library

> Status: v1.8 — `benchmarks/030-ui-subcomponent-decomposition-criteria.md` (Study 40, Case 19
> mockups) reinforces `ActivityFeedItem`/`DividedList` (Cluster 6) with a third independent source
> and names two new not-yet-admitted candidates (Checklist widget, Settings navigation rail) — added
> to the "Not built" list below. No primitive's wired status changed. | Created: 2026-09-07 |
> Updated: 2026-09-10
>
> Previously v1.7 — `CAP-V28` (category-keyed saved approval-flow template) implemented and verified
> live, closing a second row of the "Known gaps against real mockups" table (`CAP-F24` already
> closed as of v1.6) | Updated: 2026-09-09

> Status v1.6 — two of the "Known gaps against real mockups" table's own metadata-only pieces
> were wired for real onto the persistent `menata_runtime` database (the one actually serving
> menata.app), not just proven on the throwaway isolated schema the 2026-09-07 assembly proof used

> **Live-wiring pass (2026-09-08).** Owner asked to make the real running app's Detail/Submit/
> Dashboard pages actually match four `ui-sample` mockups "hanya dengan mengubah metadatanya" (by
> metadata change alone), as a live test of whether the capability the rest of this document
> catalogs really is metadata-configurable end to end — not a new capability, a verification pass.
> Checking the persistent `menata_runtime` DB directly (`psql`, RLS-scoped with `SET LOCAL
> app.workspace_id`) found the two Detail-page `children` entries (`seeds/042`) and the CAP-F24
> toggle fields (`seeds/043`) already live — confirms `capability-registry.md`'s own dated proof
> was against the real database this time, not stale. Two pieces from the gap table below were
> NOT live despite being described as "already metadata-only (proven live)": `document-submit.
> html`'s embedded "Approval steps" rows and `approval-dashboard.html`'s "Summary" tiles had only
> ever been verified on `CREATE SCHEMA verify_ui_components` (dropped after that session), never
> actually applied to `menata_runtime` itself — a real instance of this repo's own named "shared
> dev DB drift risk" (isolated-schema proof ≠ live state). Closed by `seeds/
> 044_document_submit_dashboard_live_wiring.sql`: `child_lines` (CAP-F16, already ✅) added to
> `vw_ad_form`, embedding Approval Step authoring (including the CAP-F24 User/Group toggle) in the
> real Document creation form; a new `vw_ad_dashboard` (CAP-V10, already ✅) grouping Approval
> Document by Status. Both applied directly to `menata_runtime` and picked up via `POST
> /{ws}/admin/reload` (CAP-X04, no restart) — verified end to end with real HTTP requests as Alice
> (create) and Bob (decide), not just a config read: the child-rows section renders with real
> Bob/"Document Approvers" options, the dashboard tile renders a real 2-document status
> breakdown. Also renamed `vw_ad_progress`/`vw_as_place` (pure `name` column edits, no config
> change) to `document-approval.html`'s own exact copy ("Approval Progress"/"Your Signature
> Position") — confirmed matching live on Bob's real Step Detail page.
>
> **What this did NOT close, named honestly rather than silently worked around:** CAP-V12 (wizard
> `steps`) and CAP-F16 (`child_lines`) turned out not to compose in code today — `ui.WizardForm`
> (`internal/ui/wizard.templ`) takes no `childLines` parameter — so the live form has the mockup's
> "Approval steps" content but not its 3-step pagination; fixing that is a code change (a new
> `WizardForm` parameter + `record_crud.go` wiring it through), not a metadata one, and wasn't
> attempted here. The new dashboard's tile is a real per-Status count, not the mockup's own
> cross-Machine "Total Pending" / SLA-computed "Overdue"/"Due Today" tiles — this app has no
> Leave/Corrective Action Machine and no SLA mechanism on Approval Document for those to source
> from; CAP-V10's Sections config has no way to express either even if the data existed. Every
> other gap the table below already named (inbox-as-cards, SLA-bucket filter, Choice Card,
> multi-pin overlay, embedding `pending_documents`/`recent_activity` on the dashboard) is
> unchanged — still real code work, not metadata, per the same table's own verdict.

> **Cleanup (2026-09-07), two unrelated fixes asked for together.**
>
> **1. More definitive logging.** Every silent-`nil` branch in `embed.go`/`decisionstepper.go`'s/
> `coordplace.go`'s new render-a-child functions used to fail exactly the same way regardless of
> WHY — a declared `children` entry naming a typo'd View id, one naming a View of a Type this
> runtime doesn't (yet) embed, a dangling parent/file reference, a parent Machine missing its own
> `steps_machine`/`steps_parent_field` config — all produced the identical outcome (the section
> just doesn't appear) with zero trace anywhere. Given this codebase's own real, dated incident with
> exactly this class of problem (`capability-registry.md`'s `CAP-V20` row, the `mch_ca_lifted`
> corrupted-`process`-column incident that blocked a clean boot with no warning beforehand), these
> are now split into two groups: ordinary, expected empty states (a record with no parent reference
> set yet; a role that can't read the parent) stay silent, since logging those would just be noise
> on every normal page load; genuine anomalies (a `children` reference that doesn't resolve or
> isn't an embeddable Type, a dangling record reference, a missing storage file, a parent Machine
> missing required config) now each get their own specific `slog.Warn` naming the exact view/record/
> field involved — not a generic catch-all message.
>
> **2. Stopped threading a derivable value in as its own parameter.** Asked directly: why did
> `CoordPlacePreview` take an `isPDF bool` parameter at all, when it already receives `previewKey`
> and `isPDF` is nothing but `previewKey`'s own file extension? The answer was "no good reason" —
> `internal/handler/coordplace.go`'s `coordPlacePreview` computed it once for its own internal
> reason (deciding whether to even attempt PDF page-counting) and then kept exporting it, and both
> callers kept re-forwarding it another two hops down into the render layer, which could have just
> asked `previewKey` the same question itself. Fixed: `coordPlacePreview` no longer returns `isPDF`
> (still computes it locally, since it still has its own real reason to); `CoordPlace`/
> `CoordPlacePreview` (`coordplace.templ`) dropped the parameter entirely and gained `isPDFPreview
> (previewKey)`, a one-line pure function, called at both of the two spots that used to read the
> parameter. The exact same smell existed one component over: `MemberChip`'s own `initials string`
> parameter was 100% `initials(name)`, computed by every call site right before calling it
> (`admin.templ`) — dropped the same way, `MemberChip` now takes `name` only and calls `initials
> (name)` itself. Neither fix changed any rendered output (confirmed live: Bob's chip still reads
> "BO", his embedded PDF preview still renders as an `<object>`) — both are pure "who computes this"
> moves, the render layer now owning facts it can derive from data it's already been given, instead
> of trusting a second, independently-computed copy the caller had no real need to hold.

> **Update (2026-09-07): `Children` now has a second real consumer.** v1.3's own honest weakness —
> generalizing a dispatch-by-Type mechanism when exactly one Type (`decision_stepper`) actually used
> it — is closed: Approval Step's own Detail page now *also* embeds its own `coord_placement` View
> (`vw_as_place`, "Set Signature Position") inline, next to the decision stepper and the
> `StickyActionBar` — the full `document-approval.html` mockup shape (Document fields → Decision
> Progress → Your Signature Position → Approve/Reject bar), not just the progress half of it.
>
> Unlike `decision_stepper` (a DIFFERENT Machine's own record, found via `steps_parent_field`),
> `coord_placement` is same-record: Approval Step's own `(page, x%, y%)` Fields, previewing a file
> on the Document it references. `internal/handler/coordplace.go`'s new `renderCoordPlacementChild`
> reuses `coordPlacePreview`/`coordPlaceOwnerOK` verbatim (the same resolution/ownership logic the
> standalone `/place` route already ran) against the host record directly. `internal/ui/
> coordplace.templ`'s own preview+pin body was extracted into `CoordPlacePreview`, the same
> "exported so embed.go can reuse it as a `templ.Component`" move `StepperList` already made for
> `decision_stepper`. `embed.go`'s dispatch (`renderChildView`) grew by exactly one `case` line, as
> designed. `EmbeddedSection` gained `ActionLabel`/`ActionHref` (reusing `SectionWrapper`'s
> pre-existing action-link slot, not a new one) so a multi-page embedded preview can link to its own
> full `/place` page, where page-switching (`?page=`) still works — the embedded copy can't safely
> support that itself, since a page-selector `<form>`'s `this.form.submit()` would re-GET the HOST
> page's own URL, not the coord_placement route's.
>
> `seeds/042_inline_view_composition.sql` (renamed from `042_inline_decision_stepper.sql` — its own
> scope grew, so did its name) now declares both: `{"children":[{"view":"vw_ad_progress"},
> {"view":"vw_as_place"}]}`. Verified live the same way as every prior pass: config reset to `{}`
> on the already-deployed binary (neither card shown), then the two-entry `UPDATE` + `POST /admin/
> reload` (both cards present, sticky bar unaffected, no restart) — against Bob's own real step on
> the live demo document, whose editable pin correctly carries his own record's `data-coordplace-*`
> attributes.

## Design rationale — `Children`/dispatch-by-Type vs. a single-purpose field

Asked directly after the second correction above: was generalizing worth it, compared to just
keeping a narrow, single-purpose `ParentStepperView`-shaped key? Answered honestly, including the
real cost, not just the win:

| | Single-purpose field (`ParentStepperView`) | `Children` + dispatch-by-Type |
|---|---|---|
| Consistency with the rest of this runtime | Exception — the one place deciding what to render from a config key's own *name* | Matches the router's own existing pattern everywhere else: resolve a View, read its `Type`, dispatch |
| Adding a second embeddable Type | A whole new field (`ParentReportView`, …) — the exact "single-purpose parameter per capability" shape `DetailLink`'s own doc comment already named as an anti-pattern here | One map entry (`EmbeddableChildViewTypes`) + one `case` (`embed.go`) — proven by `coord_placement` above, not just claimed |
| Embedding more than one View | Not possible — singular by construction | `[]ChildViewRef` — already a list, no change needed |
| Convergence with `CAP-V10` Tier 2 (still ❌ not admitted) | Dead end — a different mechanism would be needed later | Same `{view: <id>}` shape Study 37/38 already proposed for it |
| Code paths to trace for one feature | One field, one check, one function | Config → `GetView` → `switch Type` → a second, Type-specific function in a different file — more hops |
| Justified by real need today, or by speculation? | N/A | **Real**, not speculative: the generalization was motivated by an existing architectural inconsistency (every other View dispatch already worked this way), not by "might need it later" — but it only had ONE actual user (`decision_stepper`) until `coord_placement` above, which is the legitimate part of the "premature generalization" concern until that second case existed |
| A new invariant to maintain | None | `EmbeddableChildViewTypes` must stay in sync between `metadata/validate.go` (load-time) and `embed.go` (render-time) — mitigated with cross-referencing doc comments on both, not compiler-enforced |
| Error message specificity | Sharp: "must name a decision_stepper View, is X" | Slightly more generic: "not a Type this runtime knows how to embed as a child" (room for a future improvement: list the valid Types in the message) |

**Bottom line:** for this runtime specifically, the generalization was the right call — not because
generic is always better, but because the narrow version was the one place breaking an
architectural pattern held everywhere else, for a single capability. The honest cost (more
indirection, one more invariant to keep in sync) is real and worth naming, not papering over — and
`coord_placement` becoming a second real consumer the same day is what turns "justified by
principle" into "justified by principle AND practice."

> **Correction (2026-09-07), a second one, on top of the one below.** v1.2 fixed *that* the
> composition had to be metadata-declared; it didn't fully fix *how* the code was allowed to reason
> about that declaration. Its config key, `parent_stepper_view`, still baked "this must be a
> decision_stepper" into the key's own name — so the handler code was still checking "does this
> View want a stepper," a capability-specific question, instead of "what View is named here, and
> what does ITS OWN declared type say to do with it." Flagged directly by the owner: "coba baca
> kembali di dokumentasi, ui komponen dipecah menjadi kecil kecil tujuannya agar bisa diatur hanya
> di metadata... bukannya harusnya logika bukan cek untuk tipe tampilan ini. tapi cek dari view
> ini, tipenya apa. semua adalah view, kapabilitasnya ada di property view ini." Every other
> primary View this runtime serves already works this way (the router resolves a Machine's own
> View by its declared `type` and dispatches from there) — the inline-embedding mechanism was the
> one place still special-casing a specific type by name instead of dispatching generically.
>
> Corrected: `ViewConfig.Children []ChildViewRef{View string}` (`internal/model/model.go`) replaces
> `ParentStepperView` — a View names any other View by id, with **no opinion about what kind**.
> `internal/handler/embed.go`'s new `renderChildView` is the actual dispatch: resolve the named
> View, `switch` on its own `Type`, hand off to that Type's own renderer (today, only
> `decision_stepper` → `renderDecisionStepperChild`, `decisionstepper.go`). `Detail`
> (`detail.templ`) itself never learns what kind of View it embedded — it renders a generic
> `[]EmbeddedSection{Title, Content templ.Component}` inside `SectionWrapper`, exactly the
> "capability lives in the View's own property" shape asked for. `model.
> EmbeddableChildViewTypes` is the one place the load-time check (`metadata/validate.go`) and the
> render-time dispatch (`embed.go`) share, so the two can't silently drift apart — adding a second
> embeddable Type is one new map entry plus one new `case`, nothing else touched. Verified two ways
> against the real running app: the positive case again (config reset to empty on the already-
> deployed generalized binary, card absent; `seeds/042`'s new `{"children":[{"view":
> "vw_ad_progress"}]}` applied + `POST /admin/reload`, card present, no restart), and — new this
> pass — a **negative** case: an isolated schema with `children` deliberately pointing at
> `vw_ad_all` (a `list`-type View, not a stepper) failed to boot with a clear, specific error
> naming exactly which View and why, proving the "Unknown = explicit" validation actually rejects a
> wrong-type reference, not only a missing one. `capability-registry.md`'s `CAP-V20` row carries the
> full account as its own dated note, same as the first correction.

> **Correction (2026-09-07), superseding how the status update below describes the fix — read this
> one first.** The v1.1 fix was real (the inline progress card is the right screen), but *how* it
> decided to show that card was wrong: `parentStepperFor` scanned every Machine looking for one
> whose `steps_machine` config happened to match the page being rendered, then composed the two
> screens automatically whenever it found one. Flagged directly by the owner: "seharusnya
> pengaturan seperti itu hanya berdasarkan metadata saja" — and re-reading Study 38's own reasoning
> for decomposing the UI into `RecordSummaryCard`/`StickyActionBar`/`SectionWrapper`/etc. in the
> first place confirms the point: the decomposition exists so a *composition* can be declared in
> metadata, not so a Go function can silently reassemble it by pattern-matching Machine structure.
> A Go heuristic doing that composition is the exact thing this whole exercise was supposed to
> replace.
>
> Corrected the same day: `ViewConfig.ParentStepperView` (`internal/model/model.go`) is a new,
> explicit metadata key — a child Machine's own "detail" View (e.g. Approval Step's
> `vw_as_detail`) names the PARENT's own `decision_stepper` View (e.g. `vw_ad_progress`) it wants
> shown inline. `parentStepperFor` now only ever reads that declared key — no scanning, no
> inference. Load-time validated (`internal/metadata/validate.go`, same discipline as
> `coord_placement`): the named View must exist and actually be a `decision_stepper`. Resolving a
> View by id across Machines needed one new index, `Interpreter.GetView` (`interpreter.go`),
> mirroring `metadata/validate.go`'s own pre-existing `eventMachine` reverse index for the same
> class of cross-machine reference. `seeds/042_inline_view_composition.sql` is the actual metadata
> command — `UPDATE views SET config = config || '{"parent_stepper_view":"vw_ad_progress"}' WHERE
> id = 'vw_as_detail'` — and it was verified as the ONLY thing controlling this: deployed the
> binary first with the seed *not* yet applied (inline card absent, confirmed against the live
> Step Detail page), then ran that one `UPDATE` plus `POST /admin/reload` (CAP-X04, no restart, no
> new deploy) against the exact same running process — the card appeared immediately after.
> `capability-registry.md`'s `CAP-V20` row carries the full account, including the correction
> itself, as its own dated note.

> **Status update (2026-09-07): `SectionWrapper`'s "not yet wired" note below is now wrong for one
> case.** Trying the assembled Document Approval app end-to-end (as opposed to just verifying each
> primitive in isolation) surfaced a real UX gap `document-approval.html`'s own mockup doesn't
> have: the real app's Approve/Reject decision (an Approval Step's own Detail page,
> `StickyActionBar`) and its Decision Progress (`CAP-V20`, a separate `/progress` page) were two
> different page loads — deciding a step gave no visibility into where that step sits in the whole
> flow, unlike the mockup's single combined screen. Fixed: a Step's own Detail page now also shows
> the parent Document's decision-stepper progress **inline**, in a `SectionWrapper` card between
> the fields and the `StickyActionBar` — `SectionWrapper` turned out to have a real use *inside* a
> single Machine's own Detail page, not only across multiple Machines' Views the way `CAP-V10` Tier
> 2 needs it. `internal/handler/decisionstepper.go`'s `parentStepperFor` (reverse `steps_machine`
> lookup, degrades to nothing shown on any error or permission gap) and `computeStepperSteps`
> (extracted from `DecisionStepper` itself, now shared by both the full-page and inline paths) do
> the work; `stepperList` (`decisionstepper.templ`) is the shared row-rendering `templ` both paths
> render from — a second render call site was the actual proof this component decomposition works,
> not just that each piece compiles. `ui.Detail`'s own new `parentStepper *ParentStepper` param
> and `ui.ParentStepper` (`types.go`) carry the doc comments with the full detail. Verified against
> the real running app (both the Sequential demo document, and the Parallel one — Bob's own step
> now shows "Step 1 — Bob: Current" with its own inline Approve/Reject *and* "Step 2 — Carol:
> Current" with none, directly above the sticky bar), and against an unrelated Machine's Detail
> page (Design Request) to confirm nothing renders there — `parentStepperFor` correctly finds no
> parent for a Machine nothing else names as its `steps_machine`.

Catalog of the general-purpose `templ` presentation primitives extracted from Study 38
(`benchmarks/029-composed-view-component-inventory.md`, `app/web/static/ui-sample/component-
proof.html`) — recurring UI shapes found across the ui-sample mockups, most heavily in the
Document Approval + Group family (Case 3, `case-portfolio.md`), the best-covered case in that
inventory. Study 38's own verdict was that these are **presentation primitives, not new
capabilities** — no new registry row, no new metadata mechanism, pure `templ` extraction work.
This document is that extraction's own reference catalog: signature, source cluster, real call
site (file:line), and — for the ones not yet wired anywhere — why not, so a future session doesn't
have to re-derive that reasoning.

Every primitive lives in [`internal/ui/components.templ`](../internal/ui/components.templ), each
with its own doc comment carrying the same information in more detail. This document is the
cross-cutting index; the doc comment on each `templ` func is the authoritative source if the two
ever disagree (append a dated correction here rather than letting this file silently drift, per
root `CLAUDE.md`'s "append, don't rewrite" convention).

## Catalog

| # | Component | Study 38 cluster | Status | Real call site |
|---|---|---|---|---|
| 1 | `Avatar(initials string)` | Cluster 3 | Implemented, wired | `AvatarStack`, `RecordSummaryCard`, `ActivityFeedItem` (all below) |
| 2 | `AvatarStack(initials []string)` | Cluster 3 | Implemented, wired | `admin.templ`'s `AdminUsers` — Groups list member avatars |
| 3 | `RecordSummaryCard(avatarInitials, title, subtitle string, badge templ.Component)` | Cluster 2, ✓ proven by diff | Implemented, **not yet wired** | none — see below |
| 4 | `StickyActionBar(label, message string)` (children = actions) | Cluster 7, ✓ proven with a correction | Implemented, wired | `detail.templ`'s `Detail` — the Approve/Reject/Submit button row |
| 5 | `SectionWrapper(title, viewBadge, actionLabel, actionHref string)` | Cluster 1 | Implemented, wired (one call site; `CAP-V10` Tier 2 use still pending — see status update above) | `detail.templ`'s `Detail` — inline parent decision-stepper card |
| 6 | `StatTile(label string, value int)` | Cluster 5 | Implemented, wired | `dashboard.templ`'s `Dashboard` — CAP-V10 tile number |
| 7 | `ActivityFeedItem(actorInitials, line, meta string)` | Cluster 6 | **Wired, 2026-09-12 (composable-runtime-roadmap.md 17p)** | `components.templ`'s new `ActivityLogSection` — CAP-R04 "R28"'s own real activity feed, both record-scoped (Document/Card Detail) and cross-record (the composed page's own "Recent Activity") |
| 8 | `DividedList()` (children = rows) | Cluster 4 | **Wired, 2026-09-12 (17p)** | same `ActivityLogSection` call site as row 7 |
| 9 | `MemberChip(id, name, initials string, checked bool)` | Cluster 9 | Implemented, wired | `admin.templ`'s `GroupDetail` — Group membership editor |

Not built: **Choice Card** (Study 38 Cluster 10) — the radio-as-bordered-card shape from
`document-submit.html`'s Sequential/Parallel picker. Study 38's own verdict stands: one occurrence
is not a pattern. Revisit only if a second, independent mockup or real screen produces the same
shape. **Grid 2/3+1/3 layout** (Cluster 11) is a Tailwind class convention
(`grid grid-cols-3 gap-4` + `col-span-2`), not a component — already used ad hoc at both call
sites Study 38 named (`detail.templ`'s aside-based layouts, `approval-dashboard.html`'s mockup);
no `templ` wrapper needed for two CSS classes.

**Added 2026-09-10** (`benchmarks/030-ui-subcomponent-decomposition-criteria.md`, Study 40, Case 19
Project Management mockups), same "not built" reasoning — single occurrence each, no pattern yet:

- **Checklist / progress-fraction widget** — `project-card.html`'s checkbox list + progress bar
  ("6/8 complete"), and the compact "☑ 6/8" summary on `project-board.html`'s own kanban cards.
  Would need a `display: checklist` value on `child_lines` embeds, analogous to `display: cards` on
  `list` (CAP-V02 Tier 2) — not proposed as a capability yet, just named.
- **Settings navigation rail** — `project-settings.html`'s `200px fixed nav + content` two-column
  shape (General / Lists & statuses / Labels / Custom fields / Permissions). Visually adjacent to
  the Grid 2/3+1/3 layout above but a different ratio and purpose (in-page section switcher, not a
  content+metadata split) — Study 40 flagged it as worth a consolidation-proof check against that
  layout later, not proven either way yet.

## Why 4 primitives aren't wired to a real page yet

Each one's own doc comment in `components.templ` states this too; repeated here so the catalog is
readable without opening the source file.

- **`RecordSummaryCard`** — its 4 proven sources (`quorum-approval.html` etc.) are all Process
  Overlay mockups; that subsystem (CAP-W01/W03/W04/W05) isn't ported into `app/` yet. `List`'s own
  row rendering (`list.templ`) is a table, not a one-record-per-card layout — no current View type
  renders single records as cards. Ready the moment either lands.
- **`SectionWrapper`** — now has one real call site (`Detail`'s inline parent-stepper card, see the
  status update above), but its *original* intended use is still unwired: this is also the
  rendering mechanism `CAP-V10` Tier 2 (a page composing several independently-sourced Views)
  needs, per Study 38's own verdict. Tier 2 is still ❌ not admitted (`capability-registry.md`) —
  today's real Dashboard is one View's own Sections, not several Views' worth of content, so there
  is nothing yet for THAT use to mark.
- **`ActivityFeedItem`** — no capability in this runtime resolves a cross-record "who did what
  when" feed today (`CAP-R04` is scoped to one record's own field-diff history). Available the
  moment that feed exists. **Added 2026-09-10 (Study 40):** now a THIRD independent source
  (`project-card.html`, Case 19 — a genuinely different domain from Document Approval), and the
  live, shipped `/mch_approval_document/page` route's own "Recent Activity" section is confirmed
  already faking this exact content with a static placeholder pending it — the component is not
  the blocker, only the missing `ViewType`/data source is. This is the single most-evidenced
  not-yet-wired primitive in this catalog; see `capability-registry.md`'s `CAP-R04` row (R28) for
  the admission-test next step.
- **`DividedList`** — has no unwired use on its own; it's the wrapper `ActivityFeedItem` would sit
  inside once that has a real data source, and is available to any future "row: avatar + content +
  trailing status" list that isn't already `list.templ`'s own table.

Leaving these unwired instead of forcing a call site is deliberate — the same "don't compose a
page a real capability doesn't support yet" discipline `app/CLAUDE.md` asks for generally.

## Assembly proof — Document Approval + Group, end to end

The question this catalog exists to answer: **do these primitives actually compose into the real
running screens for Case 3 (Document Approval) and its Group extension, not just the static
`ui-sample` mockups?** Verified 2026-09-07 against a real, isolated Postgres schema
(`CREATE SCHEMA verify_ui_components`, migrated + seeded, dropped afterward — the established
safe-testing pattern, `prototype/go/CLAUDE.md`'s own "isolated-schema test server" section) with a
real compiled binary, real logins (`conformance/lib.sh`'s own seeded accounts), and real HTTP
requests — not just a `go build` pass.

1. **Approve/Reject sticky bar** — created a real Approval Document (Alice, Submitter) and
   Approval Step (assigned to Bob), submitted it, then loaded `/ws_default/mch_approval_step/
   {id}` as Bob. The rendered page's button row is `@StickyActionBar("", "")` wrapping both
   `<form>` actions — confirmed `position:sticky` now has a real effect: the bar was moved
   **outside** `Detail`'s own fields card (`detail.templ`), whose `overflow-hidden` (used for
   rounded corners over the field list) silently disables `position:sticky` on anything nested
   inside it. The original inline button row sat inside that card; it would have rendered with
   the sticky class but never actually stuck to the viewport.
2. **Group member avatars** — created a Group via the real admin flow (`POST /admin/groups`),
   confirmed the 0-member state (`AvatarStack` with an empty slice renders the "no members" empty
   state correctly), then added a member and confirmed `AvatarStack(memberInitials(g.MemberNames))`
   renders one 2-letter-initials circle per member on `/ws_default/admin/users`.
3. **Group membership chips** — same Group, opened `/ws_default/admin/groups/{id}`: a checked
   (current) member renders as `MemberChip`'s solid variant, every other workspace user as the
   dashed "+ add" variant — confirmed against the real conformance suite too, not just this manual
   pass: **T200** (`conformance/tests/110_groups_lab.sh`, "Wati appears as a checked member on the
   Group's edit page") passed unchanged against the refactored template, proving the visual
   restyle (checkbox grid → chips) didn't change the underlying `<input type="checkbox"
   name="member_id">` semantics `SetMembers` (CAP-O07) depends on.
4. **Dashboard stat tiles** — loaded a real `dashboard` View (`vw_alp_dashboard`) as a
   permission-holding session; `@StatTile(tile.Title, tile.Total)` renders byte-identical HTML to
   the hand-written markup it replaced (same classes, same values) — a pure extraction, confirmed
   by direct comparison, not just "it compiles."

Full conformance suite run in the same pass: 189/230 passed. All failures were in areas this
change never touches (`CAP-F14` expression evaluation, `CAP-C13` cross-record constraints,
`CAP-O11` multi-workspace login, and most of `CAP-O07`'s own auth/notification internals beyond
the two membership-rendering tests named above) — pre-existing drift on the shared, long-lived
dev database, not a regression from this change. One of those pre-existing issues (a stale,
malformed `machines.process` JSONB value on `mch_ca_lifted`, `app_overlay_lab` — leftover from
`seeds/028_lift_lab.sql`'s own documented "applied mid-conformance-run, not persisted" fixture,
accumulated across past sessions on this persistent schema) blocked a clean server boot entirely;
fixed by resetting that one column to `NULL`, its documented default state per that seed file's
own comment — unrelated to this session's UI work, but worth naming here since fixing it was part
of getting to a clean verification pass.

## Known gaps against real mockups (2026-09-07)

Cross-checked live (against `menata.app`, not just static reading) whether four more `ui-sample`
mockups — `document-submit.html`, `document-signature-placement.html`, `document-approval.html`,
`approval-dashboard.html` — can be produced by metadata configuration alone. Full table + evidence:
`roadmap.md`'s Study 38, second addendum. Short version: the `children` composition proven above
(Assembly proof) covers exactly the Detail-page pieces of `document-approval.html` and
`document-signature-placement.html`; everything else these four mockups show beyond that is a
named, tracked gap — `CAP-F24`/`CAP-V28` (❌ Proposed, not yet built), `Choice Card` (still not
admitted, one instance), `CAP-V21`'s own "multiple pins on one preview" deferral, `CAP-V10` Tier 2
(❌ not admitted), and two gaps this cross-check named for the first time: no `list` View renders
as cards (`RecordSummaryCard` above has no consumer — `capability-registry.md`'s `CAP-V02` row),
and no mechanism filters a list by a computed SLA-urgency bucket (`CAP-V09`/`CAP-V17` rows).
`roadmap.md`'s own item 25 has the priority order for closing these.

> **Status update (2026-09-09): two of this table's own gaps are closed.** `CAP-F24` (done
> 2026-09-07, already noted elsewhere in this document's own live-wiring pass above) and `CAP-V28`
> (done 2026-09-09) are both ✅ now, not ❌ Proposed — `vw_ad_form` pre-fills its embedded Approval
> Step rows from a saved "Approval Flow Template" record the instant Document Type is picked,
> verified end to end against `menata_runtime` (`capability-registry.md`'s own `CAP-V28` row has
> the full account). Still open, unchanged by this pass: `Choice Card`, `CAP-V21`'s multi-pin
> deferral, `CAP-V10` Tier 2 (admitted to the registry 2026-09-09, still not built), the
> card-per-record list rendering gap, and the SLA-bucket list filter gap.

The general method used to reach this verdict — how to tell a real metadata gap from something
that just needs a `templ` primitive, and how to avoid the two architecture mistakes this document's
own correction history (above) already walked back once each — is written up as its own guide:
`guides/breaking-down-ui-components-for-metadata.md`.

## Conventions for the next primitive

Follow `components.templ`'s existing pattern, established by `CSRFField`/`StatusBadge`/
`TypeaheadPicker` and extended by this batch:

- Doc comment states: which Study 38 cluster (or other named evidence) it comes from, whether
  Study 38's own consolidation proof (`component-proof.html`) tested it against multiple real
  sources, and its real call site(s) — or, honestly, that it has none yet and why.
- A children slot (`{ children... }`) for "wraps arbitrary content" shapes (`StickyActionBar`,
  `SectionWrapper`, `DividedList`), following `Page`'s own established pattern
  (`layout.templ`) — not a `[]templ.Component` param unless the slot needs to be iterated
  individually (Study 38's own Cluster 7 correction: `StickyActionBar`'s actions are multiple
  independent `<form>`s, which a plain children block already handles without a slice type).
- Interactivity is Hyperscript (`_="..."`) colocated on the element it affects, never a
  page-global script — the same client-side JavaScript policy `ARCHITECTURE.md` and
  `layout.templ`'s own comment already establish for this codebase, extended by `MemberChip`'s
  `on change` toggle.
- Name a known simplification in the doc comment rather than silently shipping it — see
  `MemberChip`'s own comment on why its avatar/× glyph reflects load-time state, not a live DOM
  swap, matching `shellBottomBar`'s existing "deliberately no active-tab highlighting yet"
  precedent (`layout.templ`).
- After editing, regenerate with the pinned `templ` version (`app/DEVELOPMENT.md`'s own command,
  not `make generate`), then `go build ./...`, then verify against a real running server — an
  isolated schema for anything beyond a read-only check, per `prototype/go/CLAUDE.md`'s
  established safe-testing pattern (reused for this batch's own verification, see above).
