# UI Workflow / Interaction Benchmark

> Runs the study queued in `../capability-registry.md` §"Tracked but Not Yet Studied" row
> **UI Workflow / Interaction Patterns** (queued 2026-07-12, run same day per direct request).
>
> Scope: task-*interaction* patterns in Form and View page components — does what this runtime
> renders actually help a user work the way world-class software in the same vertical does.
> Explicitly **not** visual style/theme (`capability-registry.md`'s Theme row — checked
> 2026-07-12, zero case evidence, closed) and **not** Page-as-composition-of-Views (Page row —
> same outcome, closed). Cross-referenced against all 21 cases in `../case-portfolio.md`, not a
> single vertical, per the direct request ("untuk setiap case").
>
> Status: v0.1 | Created: 2026-07-12

---

# Method: cluster, not case-by-case

The portfolio's own Rule 2 is "one dominant cluster per case" — most of the 21 cases share a UI
*shape* with several others (Cases 16/17 are explicitly "pure composition, no new capability
expected" for exactly this reason). Treating this as 21 independent research passes would mostly
re-derive the same handful of interaction patterns 21 times. Instead: identify the interaction
*clusters* that recur across the portfolio, name the world-class reference for each, and run
`capability-lifecycle.md` §2's five-criterion admission test per cluster — same discipline as
every other benchmark in this directory, just organized by pattern instead of by vertical.

**A cluster is only admitted if a real case's own declared targets need it** — matching
`benchmarks/003`'s discipline (Study 6's original accounting declaration was corrected against
GAAP/SOX directly, not against what two platforms merely offer) and this study's own negative
results below are just as much the point as the positive ones. A world-class platform having a
feature is a benchmark, not a case — admission still requires both, per Rule A1.

---

# Clusters examined

## Cluster A1 — Live aggregate total on embedded child-line rows

**World-class reference:** Odoo/QuickBooks/Xero/NetSuite journal/invoice entry — a running
total (or, for double-entry, a running debit/credit imbalance) updates as each line is typed,
*before* the user attempts to submit — the error is visible while it's still cheap to fix, not
after a round-trip.

**Case evidence:** Case 9 (Accounting) is the primary, explicit case — `CAP-F16` (Journal Entry
Lines) plus `CAP-C10` (`sum(Debit) = sum(Credit)` before Post) are both declared targets, and a
double-entry form with no visible running balance is a well-documented real-world error source
(the exact invariant the control exists to protect). Case 15 (E-commerce) is secondary,
supporting evidence — an Order's line items (`CAP-F16`) implicitly need a visible running total
for a checkout to be usable at all, though Case 15's own declaration doesn't spell this out as
explicitly as Case 9 does.

**Admission test:**
| # | Criterion | Result |
|---|-----------|--------|
| A1 | Dual evidence | ✅ Case 9 (primary) + Case 15 (secondary) + 4 named platforms |
| A2–A5 | (see `capability-lifecycle.md` §2 for full text) | ✅ — computed client-side from data already rendered in the form, no new server round-trip, no conflict with existing capabilities |

**Architecture check:** summing numbers already present in rendered `<input>` elements on
`input` events is a small, framework-free `<script>` block — the same class of progressive
enhancement as an HTML5 `datetime-local` input, not a violation of this prototype's no-SPA
posture. Critically, it is **presentation only** — the authoritative check stays CAP-C10 at
submit time server-side; the live total is a courtesy, never trusted, same "client is advisory,
server enforces" principle CAP-C09/CAP-F06 already apply elsewhere in this registry.

**Verdict: ADMITTED — CAP-V15**, see registry candidates below.

---

## Cluster A2 — Live cross-record balance preview on create forms

**World-class reference:** Expensify/Concur ("remaining budget" shown while filling an expense
line against a report) — distinct from Cluster A1: here the aggregate being checked lives on a
*different, already-existing* record (a Fund, an Item), not on sibling rows being typed in the
same form.

**Case evidence:** Case 6 (Petty Cash) — `Voucher Amount <= Fund Current Balance` (`CAP-C08`) is
a declared cross-record constraint; a Custodian filling a Voucher with no visibility into the
Fund's remaining balance can only discover the rejection after submitting. Case 5 (Inventory) —
same shape, `Item.Stock On Hand >= Normalized Quantity` (`CAP-C08`) before an Out movement.

**Admission test:** A1 ✅ (two independent cases, both already-registered `CAP-C08` cross-record
constraints, no new benchmark platform needed beyond A1's own — the pattern is well-established
enough it doesn't need a named external product, the two cases alone clear the bar the same way
CAP-C08's own third instance, Case 5, was accepted as reinforcement not novelty).

**Verdict: ADMITTED — CAP-V19**, see registry candidates below (numbered after the calendar/SLA
clusters since it surfaced later in this pass, not because it is lower-priority).

---

## Cluster B — Typeahead search on `reference`/`user` pickers

**World-class reference:** Salesforce Lookup field, ServiceNow reference qualifier UI, Frappe
Link field, Jira/Linear user assignment picker — search-as-you-type against a filtered candidate
list, not a single long `<select>` a user has to scroll.

**Case evidence:** Case 9's Chart of Account (`CAP-F13` self-reference, a real business's COA
routinely runs to 100+ accounts), Case 18's Employee (`CAP-F13` Manager self-reference, grows
with headcount, no upper bound), Case 15's Product on Order Line (`CAP-F13`, a real catalog can
run to thousands of SKUs) — a plain `<select>` degrades badly well before any of these reach
real-world scale, and this runtime's own `reference`/`user` field rendering (CAP-F13/CAP-F05) is
today exactly that plain `<select>` for every case.

**Admission test:** A1 ✅ (3 named platforms + 3 independent cases, each already-registered
`CAP-F13`/`CAP-F05` fields whose picker this would enhance, not replace).

**Architecture check:** genuinely needs one new thing this prototype didn't have before —
`CAP-X07`'s own `GET /api/{machine}` route (already live) has no query-filtering parameter
today; a typeahead widget needs a debounced fetch against a *filtered* candidate list, not the
full list. This is a small, additive extension to an already-shipped route (`?q=` matching the
same columns CAP-V08's own list search already uses), not a new subsystem.

**Verdict: ADMITTED — CAP-V16**, see registry candidates below.

---

## Cluster C — Board (kanban) view: cross-column drag, not just within-list reorder

**World-class reference:** Trello, Jira board, Linear, Asana — cards dragged between columns
(status/list change) and reordered within a column, both without a page reload.

**Case evidence:** Case 19 (Project Management) is the case CAP-V14 itself was named for —
re-reading its own declared targets: `List.Reorder`, `Card.Move` (**both** explicitly named as
the `CAP-V14 (new)` target), plus `Card.Move changing List | CAP-F13 + CAP-A13 | Reused`. What
shipped (`capability-registry.md`'s CAP-V14 row, ✅) is honest about the gap already: "scoped to
Up/Down, not drag-and-drop... `Card.Move`'s cross-record reference-write composition... is out
of scope — no case forcing it yet." That sentence is no longer accurate — Case 19 forced it from
the start; it just wasn't picked up in CAP-V14's first implementation pass.

**Admission test:** A1 ✅ — this is unusually strong evidence: the SAME case that already forced
CAP-V14 into existence names this specific gap directly, not a new case being stretched to fit.

**Architecture check, and an honest tradeoff, not a one-sided "buttons are worse":** true
drag-and-drop needs client-side JS (native HTML5 Drag and Drop API — no framework required, same
progressive-enhancement posture as Cluster A1). But CAP-V14's existing Up/Down-button mechanism
has a real property drag-and-drop does not: it's fully keyboard/screen-reader accessible with no
extra work. The honest conclusion is **both**, not a replacement — drag-and-drop as the primary
interaction, buttons/keyboard as the always-present accessible fallback, not a JS-only feature
that silently breaks for anyone not using a mouse.

**Verdict: ADMITTED — CAP-V14 Tier 2** (an extension of the already-✅ capability, not a fresh
CAP-V number — same "escalate the existing mechanism" pattern CAP-F19's own Tier 1/2/3 framing
uses), see registry candidates below.

---

## Cluster D — SLA/deadline countdown, computed at render time

**World-class reference:** Zendesk/Freshdesk/ServiceNow ticket queues — a color-coded "time
remaining" badge on every ticket in the list, not just a binary overdue/not-overdue split.

**Case evidence:** Case 7 (Customer Complaint) declares `SLA Due Date` (`CAP-A11`, priority-keyed
offset) and auto-escalation on breach (`CAP-E02`+`CAP-A09`+`CAP-E05`) as explicit targets, and
already has `Overdue Complaints (compound filter) | CAP-V09`. What's missing between "the data
exists" and "the user can act on it at a glance" is the countdown itself — CAP-V09 today can
only render a binary filtered list (in the overdue view, or not), not "6 hours remaining" vs. "2
days remaining" as a visible, sortable signal. Case 17 (Helpdesk) reinforces this as a second,
independent instance (explicitly a domain-portability case, same SLA shape aimed at employees).

**Admission test:** A1 ✅ (3 named platforms + 2 independent cases).

**Architecture check:** trivially server-renderable — `time_remaining = due_date - now()`
computed at render time from a field that already exists (`CAP-A11`'s own offset-computed date),
the exact same "computed at render time, nothing stored" precedent `CAP-F14`/`CAP-V13` already
established. No JS required at all for the MVP (a live-ticking countdown that updates without a
page refresh would need JS or a meta-refresh, but a *value accurate as of page load* — which is
what every one of the named platforms actually shows in a list view, not a literal ticking
clock — is enough and needs none).

**Verdict: ADMITTED — CAP-V17**, see registry candidates below.

---

## Cluster E — Resource/staff-grouped calendar (two-dimension grouping)

**World-class reference:** Google Calendar resource view, Calendly team scheduling, Epic/Cerner
scheduling grids — a grid with **resource** (doctor, room, staff member) as columns and time
slots as rows, not a single person's flat date-grouped list.

**Case evidence:** Case 20 (Hospital System)'s own declared target is explicit and specific:
"Doctor Calendar | **CAP-V07 (first real case evidence)** | A flat filtered list cannot serve
'what does Dr. X's Tuesday look like'" — and what shipped for CAP-V07 (`capability-registry.md`,
✅) is records grouped by ONE dimension (a date field), server-rendered as date-sections. That
answers "what's due this week" but not "what does Dr. X's Tuesday look like specifically, next
to Dr. Y's" — CAP-V07 as built doesn't have a resource dimension at all. This is the same class
of gap Cluster C found for CAP-V14: the case that forced the capability into existence named a
sharper shape than what the first implementation pass delivered.

**Admission test:** A1 ✅ (3 named platforms + the capability's own originating case, stated
explicitly, not inferred).

**Architecture check, with an honest split:** the **static** half (a table: resource columns ×
date/time rows, computed from existing records grouped by TWO fields instead of one) is
comfortably server-renderable, no new architecture risk. **Drag-to-reschedule** is a materially
bigger lift — real-time, needs conflict detection server-side (two appointments can't land on
the same resource+slot) *and* client-side JS for the drag interaction itself. Flagging this
split, not silently picking the harder half or silently dropping the whole cluster, is the same
move `benchmarks/006`'s own inventory study made for reservation/allocation ("real, but doubles
the case's dominant cluster... candidate for a future case, not this one").

**Verdict: ADMITTED (static grid) — CAP-V18.** Drag-to-reschedule is named, not built — a
genuinely separate, larger capability (real-time conflict detection + a client interaction
layer this prototype has nothing like today), queued below, not silently absorbed into CAP-V18.

---

## Reviewed, NOT admitted (named, not silently dropped)

The same rigor that admitted five clusters above has to be applied evenly — a benchmark alone,
without a case actually demanding the interaction, is not enough (Rule A1), exactly the standard
Page and Theme were already held to.

| Pattern | World-class reference | Why it fails admission |
|---------|------------------------|--------------------------|
| **Infinite scroll + optimistic UI** (like/comment count updates before server confirms) | Instagram/Facebook/Twitter feed | Case 11/12's own declared targets are about the *data model* (`CAP-F20` join Machines, `CAP-C12` composite uniqueness) and correct feed *filtering* (`CAP-V05` extended) — neither case names infinite-scroll or optimistic-update interaction as a need; they only need the feed to show the right posts. Also independently conflicts with this prototype's own no-SPA/no-realtime posture (`CAP-V07`/`CAP-V14`'s own stated precedent) — two separate reasons, not one |
| **Faceted multi-dimension search/browse** (category × price × brand, live per-facet counts) | Amazon/Shopify | Neither Case 15 (E-commerce, whose own declared targets are about Cart/Checkout mechanics — `CAP-R08`) nor Case 13 (Blog) names a multi-dimension browse need. Case 13's own real, already-registered gap is narrower and different: `value_list` being single-select only (`CAP-F03` scope note) — a data-model gap, not this interaction pattern |
| **Keystroke-level autosave/draft** | Gmail draft autosave, Notion instant-save | No case in the portfolio declares this. Case 15's Cart is the closest analog and already solves "don't lose progress" architecturally via `CAP-R08` (each add-to-cart is its own persisted, editable scratch record) without needing per-keystroke saving |
| **SEO / Open Graph / social-share polish** | WordPress/Ghost/Medium | Case 13 (the only public-facing case) declares `CAP-P07` (unauthenticated access control) as its target — about *who* can reach the page, not about search-engine/social-share presentation. No case asks for this |
| **Live drag-to-reschedule** (Cluster E's harder half) | Calendly, Epic/Cerner | Real need per Case 20, but a materially larger, separate lift (real-time conflict detection + client drag interaction) — queued as its own future item, not bundled into CAP-V18's static-grid admission |

---

# New / extended registry candidates surfaced by this benchmark

| Candidate | Description | Evidence | Status |
|-----------|--------------|----------|--------|
| **CAP-V15** | Live aggregate total/balance preview on a form's own embedded child-line rows (`CAP-F16`), computed client-side from already-rendered inputs, purely presentational — the real check stays the existing server-side aggregate Constraint (e.g. `CAP-C10`) at submit | Case 9 (primary), Case 15 (secondary) + Odoo/QuickBooks/Xero/NetSuite | Proposed |
| **CAP-V16** | Typeahead/autocomplete search on `reference` (`CAP-F13`) and `user` (`CAP-F05`) pickers, replacing a plain `<select>` once the candidate pool is large — extends `CAP-X07`'s own JSON API with a `?q=` filter parameter (reusing `CAP-V08`'s existing column-match logic), no new subsystem | Case 9, Case 18, Case 15 + Salesforce/ServiceNow/Frappe/Jira | Proposed |
| **CAP-V14 Tier 2** | Board (kanban) view: cross-column drag-and-drop move (updates both `sort_order` and the grouping/list field in one action) plus within-column reorder, with the existing Up/Down buttons kept as the accessible/keyboard fallback, not replaced | Case 19 — the SAME case that originally forced CAP-V14 into existence, re-read against its own full declared target | Proposed, as an extension of CAP-V14 (already ✅) |
| **CAP-V17** | SLA/deadline countdown badge on list and detail views, computed at render time from a declared date field + an urgency threshold — no JS required for an accurate-as-of-page-load value | Case 7 (primary, the case CAP-V09's own Overdue filter was built for), Case 17 (reinforced) + Zendesk/Freshdesk/ServiceNow | Proposed |
| **CAP-V18** | Resource/staff-grouped calendar — records grouped by TWO dimensions (a resource-reference field AND a date field) as a grid, extending `CAP-V07`'s existing single-dimension date grouping. Drag-to-reschedule explicitly NOT included — queued as a separate, larger future item | Case 20 — the case CAP-V07 was itself built for, re-read against its own literal "what does Dr. X's Tuesday look like" declaration | Proposed, as an extension of CAP-V07 (already ✅) |
| **CAP-V19** | Live cross-record balance/remaining-capacity preview on a create form, when that form is gated by a `CAP-C08` cross-record constraint against another, already-existing record (a Fund's balance, an Item's stock) | Case 6, Case 5 — both already-registered `CAP-C08` instances | Proposed |

None of the above are implemented by this study — per the Case Portfolio's own process
(`case-portfolio.md` §Process per case, step 4: "register new findings... " precedes
implementation as a separate step everywhere else in this registry) and per the explicit
instruction this study itself was queued under ("do not implement in the same pass, registration
and implementation stay separate steps here as everywhere else").

---

# What this benchmark deliberately did NOT re-litigate

Page and Theme (`capability-registry.md` §Tracked but Not Yet Studied) were checked 2026-07-12,
independently of this study, and are unaffected by it — neither is about task-interaction
patterns, and both already concluded "no case evidence" on their own terms. This study's own
five "reviewed, not admitted" findings above use that exact same standard, applied to a
different, narrower question (interaction, not composition or branding).
