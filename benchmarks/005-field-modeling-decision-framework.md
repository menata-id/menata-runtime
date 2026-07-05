# Field Modeling Decision Framework

> Study 15 deliverable (`../roadmap.md`, Phase 5).
>
> A repeatable procedure for choosing a field's type — primitive, `value_list`,
> `reference`, or `user` — instead of ad hoc, case-by-case judgment.
>
> Triggered by a direct question during CAP-F13 (reference field) scoping:
> "some of these fields look like they should be reference — is there a
> world-class framework for deciding this?" There wasn't one; this is it.
>
> Status: v0.5 — 5 adversarial passes; consistently scoped to the metadata-author/runtime perspective, never the resulting application's end-user | Created: 2026-07-05

---

# Why this matters

Every case so far (1–10) chose field types by intuition. That worked while the vocabulary was small
(text, date, value_list, user, file). It stops working the moment `reference` (CAP-F13) enters the
picture, because now every field with any relational flavor faces a real choice — and getting it
wrong is expensive: a field modeled as `value_list` that should have been `reference` cannot grow
without a metadata change; a field modeled as `reference` where `value_list` would do adds
unnecessary machine authoring and query cost.

This framework exists so the choice is made once, consistently, by a procedure — not re-litigated
per field.

---

# External Yardsticks

| Source | What it contributes |
|--------|---------------------|
| **Codd's normalization theory (3NF)** | The functional-dependency test: does this value depend on something with its own identity, or is it an independent attribute of this record? |
| **DDD — Entity vs. Value Object** (Evans) | Entity = has identity and a lifecycle independent of any one record referencing it. Value Object = fully described by its attributes, no identity of its own. Maps directly onto reference-vs-primitive. |
| **Master Data Management (MDM)** | Formal criteria for what qualifies as *master data*: shared across multiple business processes, has its own governance/lifecycle, referenced rather than duplicated. |
| **Platform conventions** (already surveyed — `001-platform-capability-survey.md`, `003-accounting-vertical-survey.md`) | Frappe Link vs. Select, Salesforce Lookup vs. Picklist, Odoo Many2one vs. Selection, Directus M2O vs. dropdown — all draw this same line in practice. |

---

# The Decision Tree

```text
Does this field's value name a THING with its own identity and lifecycle,
independent of this record?
│
├─ NO ──► Is it one of a small, FIXED, stable set of options
│         (won't grow without a metadata change)?
│         │
│         ├─ YES ──► value_list
│         └─ NO  ──► primitive (text / rich_text / number / date / ...)
│
└─ YES ─► Is the referenced thing a workspace-authored Machine
          (has its own Fields / Events / Constraints / Views)?
          │
          ├─ YES ──► reference (target_machine: ...)
          └─ NO ───► Is it platform identity (a person / user)?
                      │
                      ├─ YES ──► user (sugar over reference-to-identity — CAP-F05)
                      └─ NO ───► UNRESOLVED — master data candidate (needs CAP-O02)
```

## Four supporting tests

Use these when the top-level question ("has its own identity and lifecycle?") is not immediately
obvious:

| Test | Question | Reference-leaning answer |
|------|----------|--------------------------|
| **Growth test** | Will new values be added by an admin, without a metadata/deployment change? | Yes → reference/master data. No → `value_list`. |
| **Identity test** | Does the value have its own ID, separate from being just a display label? | Yes → reference. No → `value_list`/text. |
| **Reuse test** | Is the same entity referenced by more than one Machine? | Yes → reference. No (used only here) → local field. |
| **Cardinality test** | Could two different records show the same label but mean different things (two people named "John")? | Yes → needs reference with a real ID, not free text. |

---

# The Two Failure Modes

Following the tree honestly surfaces two *different* kinds of "doesn't resolve cleanly" — they have
different causes and different fixes. Conflating them leads to fixing the wrong thing.

## Failure Mode 1 — Modeling gap (the tree has no answer yet)

The rightmost leaf, "Is it platform identity? NO", terminates in **UNRESOLVED**, not a usable type.
A field lands here when:

1. It passes the identity/lifecycle test (it names a real, reusable thing), **and**
2. That thing is not yet authored as a workspace Machine, **and**
3. It is not platform identity (not a `user`).

**Root cause:** the tree assumes a target exists to reference — either a workspace Machine or a
recognized system entity. When neither exists, there is nothing to point `reference` at yet. This
is not a flaw in the tree; the tree is correctly detecting that the field's *true* model (a shared,
governed entity) has no Machine and no master-data designation (CAP-O02) behind it.

**Fix:** author the missing Machine, and/or implement CAP-O02 (master data designation) so the
entity can be formally marked shared/governed. Not a field-type fix — a capability-implementation fix.

## Failure Mode 2 — Execution gap (the tree resolves; the runtime can't realize it yet)

The field resolves cleanly to `user`, `reference`, `number`, `date_time`, or `file` — but the
prototype's current implementation of that type is incomplete (renders as free text, no validation,
no storage). This is not a modeling problem. The type choice is correct; only its runtime
realization is unfinished, and it is already tracked in `capability-registry.md` under CAP-F05
(user), CAP-F06 (file), CAP-F07 (number), CAP-F10 (date_time), CAP-F13 (reference).

## Why the distinction matters

| | Failure Mode 1 | Failure Mode 2 |
|---|---|---|
| Nature | Modeling gap — the right type doesn't exist in usable form yet | Execution gap — the right type is chosen, its implementation isn't done |
| Cause | No Machine / no master-data governance behind the entity | Runtime hasn't built the renderer/validator/storage for an already-correct type |
| Fixed by | CAP-O02 + authoring the missing Machine | CAP-F05 / F06 / F07 / F10 / F13 (already prioritized) |
| Frequency observed | Rare — one case so far | Common — most non-primitive fields today |

---

# Retrofit Calibration — running the tree against every field in Cases 1–10

| Field | Case | Tree answer | Outcome |
|-------|------|-------------|---------|
| `Title` | Design Request, Approval Document | Fails identity + growth test | `text` — correct as-is |
| `Design Type` / `Leave Type` / `Approval Mode` / `Decision` / `Frequency` | 1, 2, 3, 4 | Fixed, small, no independent identity | `value_list` — correct as-is |
| `Due Date` / `Start Date` / `End Date` / `Next Due Date` / `Last Completed` | 1, 2, 4 | Primitive value, no identity | `date` — correct as-is |
| `Description` / `Reason` / `Notes` / `Completion Notes` | 1, 2, 3, 4 | Primitive value, no identity | `rich_text` — correct as-is |
| `Requester` / `Employee` / `Approver` / `Assignee` / `Supervisor` / `Submitted By` | 1, 2, 3, 4 | Identity ✓, not a workspace Machine, is platform identity | `user` — **correct type, Failure Mode 2** (CAP-F05) |
| `Document` (Approval Step → Approval Document) | 3 | Identity ✓, is a workspace Machine | `reference` — **correct type, Failure Mode 2** (CAP-F13) — the original case that discovered CAP-F13 |
| `Sequence` | 3 | Primitive value, no identity | `number` — **correct type, Failure Mode 2** (CAP-F07) |
| `Decided At` | 3 | Primitive value, no identity | `date_time` — **correct type, Failure Mode 2** (CAP-F10) |
| `File` / `Attachment` | 1, 3 | **Identity ✓** (own storage key, versioning, lifecycle) | **Reclassified to reference sugar** (was scored Failure Mode 2 / plain primitive in the first pass — see "Second-Pass Corrections" and "Final Recap" below for the settled answer) |
| `Equipment` (used only within Case 4's own application, Facility) | 4 | Identity ✓ (reusable asset), not a workspace Machine, not platform identity | **Failure Mode 1, but resolved by CAP-F13 alone** — currently mis-modeled as `text`; fix is to author an `Equipment` Machine and change the field to `reference` (Vehicle Type / Vehicle Asset / Service Record / Workshop Entry, connected purely by `reference`, exactly like `Document` below). No CAP-O02 required — CAP-O02 only matters once *another application* also needs the same Equipment records. |
| Employee / Customer / Equipment referenced **across more than one application** | 10 (narrative) | Same identity/reuse pattern as Equipment, but the reuse crosses application boundaries | **Failure Mode 1, genuinely needs CAP-O02** — CAP-F13 alone answers "how do I point at another Machine," not "who owns this Machine when two applications both want to reference it, and what happens on deactivation." This cross-application governance question is what CAP-O02 exists for — it is not required for the single-application case above. |

**Fourth-pass correction:** the first three passes correctly separated *Failure Mode 1* (no reference
target exists) from *Failure Mode 2* (target type resolved, implementation incomplete) — but conflated
two different reasons a Failure-Mode-1 field could be blocked. `Equipment` used only inside one
application is solved completely by CAP-F13 (Prio 1, already top of the queue) plus authoring an
ordinary workspace Machine — no new capability needed at all. Only when the *same* Equipment Machine
must be referenced from a *second, different* application does the harder CAP-O02 question (cross-app
ownership, deactivation semantics) actually arise. Treating every "should be reference" field as
automatically needing master-data governance overstates the blocker for the common, single-application
case — the same shape of overcorrection the third pass caught for Quantity (below).

**Calibration result:** the framework reproduces almost every prior ad hoc decision correctly, and
correctly isolates `Equipment` as a true modeling gap versus merely-unimplemented-yet-correct
choices — while a follow-up question caught that the gap's *size* was initially overstated (see
fourth-pass correction above). One field type (`file`) was initially miscategorized — corrected below.
This mirrors the calibration discipline used for the capability admission test in Study 9
(`capability-lifecycle.md` §6): a good framework should survive repeated adversarial passes, and
this one found something on each pass — which is a feature, not a failure, of doing the passes.

---

# Two Orthogonal Axes Behind "Reference vs. Primitive vs. value_list"

A follow-up question exposed that the tree's second branch ("small, fixed, stable set of options")
was stated as a practical heuristic without naming the deeper reason. Making it explicit prevents a
common misreading: that `value_list` is just "a list of values, which could be text or number,"
differing from `text`/`number` only in input widget (dropdown vs. free text).

That reading conflates three concerns that are actually independent:

| Concern | Example | Level |
|---------|---------|-------|
| **Semantic domain — closed or open?** | `value_list`: only the declared values are valid. `text`/`number`: any value is syntactically valid. | **This is what determines the field type.** |
| **Input widget** | dropdown, radio buttons, chips, autocomplete | Presentation / View concern — orthogonal to type |
| **Storage representation** | stored as a string, or as an integer code (1=Poster, 2=Thumbnail) | Implementation detail — spec `002-field.md` is explicit: "Value Lists describe Business Knowledge. They do not prescribe how values are stored or implemented." |

The real distinction: a `value_list` field has a **closed domain** — the runtime can validate
membership, render category-aware presentation (e.g. `StatusBadge`), and reason about
exhaustiveness ("there are exactly N possible states"). A `text`/`number` field has an **open
domain** — any value is syntactically valid; only a Constraint can narrow it, and even then a
Constraint expresses an arbitrary rule, not an intrinsic, named set of categories.

This means the framework actually has **two orthogonal axes**, not one linear tree:

```text
Axis 1 — Is the domain of valid values CLOSED or OPEN?
    CLOSED → value_list or reference
    OPEN   → primitive (text / number / date / ...)

Axis 2 — If CLOSED, is the enumeration FIXED or does it GROW / need its own governance?
    FIXED  → value_list
    GROWS  → reference (needs identity, lifecycle, possibly master-data governance)
```

The original decision tree already asks these two questions in sequence — this section makes the
*reason* for the second question explicit, rather than leaving it as an unexplained heuristic
("small, fixed, stable").

---

# Field Type Inventory Reconciliation

Classifying every type in `runtime-metadata-schema.md` against the framework:

| Category | Types |
|----------|-------|
| **Pure primitives** (open domain, never reference) | `text`, `rich_text`, `number`, `boolean`, `date`, `time`, `date_time` |
| **Composite primitive** (open domain, but internally pairs a magnitude with a unit) | `duration` — see note below |
| **Reference sugar** (resolves through `reference`, to a special built-in target) | `user` (target: platform identity, pending CAP-O01), `money`'s currency component (target: Currency, a CAP-O02 master-data candidate), `file` (target: a runtime-managed File/Document entity) |
| **General mechanism** | `reference` (target: any workspace Machine, or a master-data-designated Machine per CAP-O02) |
| **Bounded enumeration** | `value_list` (closed domain, fixed — not a reference; see "Two Orthogonal Axes" above) |

## Second-Pass Corrections

Running the same tests a second time (prompted directly, rather than accepted at face value)
surfaced two corrections and one clarification the first pass got wrong or left implicit:

**`money` — reclassified from primitive to reference sugar.** `money` is not a single scalar; it
pairs a numeric amount with a currency. The amount alone is a genuine primitive (no identity), but
the currency component passes all four supporting tests: it has identity (a code, symbol, decimal
precision), a lifecycle (exchange rates change over time — this is exactly what CAP-F17,
"multi-currency: transaction currency + rate + base mirror," already named), it is reused across
every money field in every machine, and two records both saying "USD" must resolve against the same
rate source, not independent free-text labels. `specification/001-object.md` independently confirms
this — it lists **Currency** as an example Object in its own right, alongside Customer and
Department, not as an attribute folded into another type. Currency is therefore a CAP-O02
master-data candidate, joining `Equipment` as a second, independent instance of the same gap.

**`file` — reclassified from primitive to reference sugar.** A stored file has its own identity
(storage key/URL), its own lifecycle (versioning, replacement, deletion, scan status), and can be
reused across records (a shared document attached to multiple approval steps). This is not a
storage-format detail — it is a structural fact about files as a concept. Study 2's platform survey
already contains the evidence for this without it having been named at the time: Frappe's `Attach`
field is a reference to a **File DocType**, Salesforce's file field is `File`/`ContentDocument` (a
distinct object), and Drupal's `file` type is an entity reference — every major platform surveyed
models files as referenceable entities, not inline primitives. CAP-F06's existing gap note ("upload
is not stored") was already symptomatic of exactly this: there is no File entity to point at yet,
the same shape of problem as CAP-F05 waiting on CAP-O01.

**`duration` — checked, confirmed primitive, but the reasoning needed to be explicit.** `duration`
is structurally similar to `money` — it also pairs a magnitude with a unit (100 *minutes* vs. 100
*days* are as different as 100 USD vs. 100 IDR). The difference is the growth axis, not the
identity axis: the set of possible time units is small, universal, and fixed forever (seconds
through weeks) — there is no "exchange rate between units" that changes, no lifecycle, no growing
catalog. It resolves to a `value_list`-shaped inline unit selector, not `reference`, applying the
same Axis 2 test as any other closed-and-fixed enumeration. Composite structure alone does not imply
`reference` — only composite structure *plus* a growing/lifecycle-bearing component does.

**Predicted, not yet evidenced — flagged for Case 5.** `number` in its ordinal/count use (e.g.
`Sequence` in Approval Step) is genuinely primitive — no unit, no identity. But a **Quantity** field
(count + Unit of Measure — kg, pcs, box, liter, with conversion factors between them) would be a
third instance of the same general pattern as `money`, expected when Case 5 (Inventory / Stock
Movement) is written. Not corrected now — Case 5 doesn't exist yet, so there is no case evidence,
only the benchmark-side prediction. Recorded here so the case-portfolio process (declare targets
first) can check this prediction against what Case 5 actually surfaces.

## Third-Pass Refinement — Quantity Is Not "Same as Money", It's Tiered

A follow-up question ("must there be conversion if units differ, and where does it live?") caught
that treating Quantity as a straight copy of the `money`/Currency resolution — i.e. assuming it
always needs a dedicated reference Machine — is itself an overreach in the opposite direction from
the original mistake. The first pass under-modeled `money`/`file` as plain primitives; a naive
application of the correction over-models Quantity as always needing full reference machinery. The
correct answer is conditional on actual cardinality, and splits into three tiers:

**Tier 1 — one fixed conversion pair per item (the common case).** Two extra fields directly on the
`Item` Machine are enough — no new Machine at all:

```yaml
- id: fld_item_base_unit
  name: Base Unit
  type: value_list          # SAK, TON, KG, PCS — a small, fixed, closed set
  values: [SAK, TON, KG, PCS]
- id: fld_item_conversion_factor
  name: Conversion Factor    # how many Base Unit per alternate unit
  type: number
```

Note the correction folded in here: **the unit label itself** (`SAK`, `TON`, `KG`) is usually
`value_list` — small and closed, exactly like a Currency *code* often is for an organization that
only ever deals in 3–4 currencies. It is **only the conversion factor** that behaves like an
exchange rate (changeable, needs a place to live as data) — the earlier framing conflated "the unit
label" with "the thing that needs governance." They are not the same component.

**Tier 2 — one item convertible to more than one other unit.** A single factor no longer fits as two
flat fields (the cardinality is one Item to many Unit/Factor pairs). This becomes a **child table
under Item** — i.e. CAP-F16 (line items), not a new top-level Machine:

```text
Item: Cement
  ├─ Unit: BOX,   Factor: 1
  ├─ Unit: DOZEN, Factor: 12
  └─ Unit: PCS,   Factor: 1
```

**Tier 3 — conversion factors themselves need history** (the factor changes over time and old
movement records must keep reinterpreting under the old factor, e.g. a supplier changes bag size).
Only here does a dedicated, effective-dated reference Machine (what the first pass jumped straight
to) actually earn its cost.

**Escalate only when cardinality or history actually demands it** — the same principle already
established for `reference` vs. `value_list` in general (Two Orthogonal Axes, above) applies
recursively to how the conversion data itself is modeled. Case 5, when written, should declare which
tier its scenario actually needs rather than assuming Tier 3 by default.

### Fourth-pass correction — "declaring the field" and "authoring the conversion schema" are different moments

A follow-up question caught that presenting the three tiers side by side, as if choosing between
them were part of using `Money` or `Quantity`, overstates the field's everyday complexity. This
conflates two separate metadata-authoring moments that should stay separate:

| Moment | Example | Who does it, how often |
|--------|---------|--------------------------|
| **Declaring the field** in a Machine's metadata | `- Amount : Money`, `- Quantity : Number` | Every metadata author (domain expert, developer, or AI writing `.menata`/Runtime Metadata), every time — stays exactly **one line**, identical in effort to declaring any other field type |
| **Authoring the conversion schema** (only if Tier 2/3 is actually needed) | Declaring the extra fields/child-table/Machine that will *hold* conversion factors (Tier 1/2/3, above) | **Once**, by whoever designs that part of the metadata — not repeated per form, per machine, or per use of the field type |

Both rows above are still **metadata** — schema declarations written in `.menata` or Runtime
Metadata, not data entry. The *actual conversion factor values* (e.g. "this specific item's factor
is 40") are business data, entered later through whatever form the resulting application renders
from that schema — a different layer entirely, and out of scope for this document (see next
section). This mirrors how `- Department : Department` already works today: writing that field is
one line of metadata; the fact that Department *records* exist is a separate concern at a different
layer, not a burden on the field declaration itself.

### Fifth-pass clarification — three layers, and where this framework stops

Keeping "declaring" and "authoring the conversion schema" separate only holds up if the layers
underneath are also kept separate. This framework, end to end, speaks from **one** perspective —
the metadata author and the runtime that loads what they wrote — never the end-user of the
resulting application:

| Layer | Who | What happens here | In scope for this document? |
|-------|-----|--------------------|-------------------------------|
| **Metadata** | Domain expert, developer, or AI writing `.menata` / Runtime Metadata | Declares field types, Machine structure, and — for `Money`/`Quantity` — how much schema (Tier 1/2/3's *shape*) the conversion mechanism needs | **Yes — this is what the whole framework decides** |
| **Runtime** | The loader / validator / interpreter | Loads metadata, checks it is complete and well-formed (CAP-X05), builds the Application Model | **Yes — this is where the safeguard below lives** |
| **Data** | End-user of the *resulting application* (e.g. a warehouse clerk, an accountant) | Enters actual records: this item's actual conversion factor, this currency's actual exchange rate on a given date | **No — explicitly out of scope.** This document never discusses end-user forms, data-entry UI, or hints inside the generated app — Menata Runtime interprets metadata into that app, it does not design this document around it. |

Every Tier 1/2/3 distinction above is a statement about **how much schema must exist**, never about
the values that will eventually populate it. Conflating "authoring the conversion schema" (metadata)
with "typing in today's exchange rate" (data) is exactly the perspective slip the next section
closes off explicitly.

### Language-Level Safeguard Against Forgotten Setup

A related risk in keeping "declare" and "author the conversion schema" separate: if declaring
`type: money` stays simple and its conversion schema is authored separately, what stops a metadata
author — human or AI — from *forgetting* to author that schema at all? The answer has to live in
the **metadata language and the runtime** (the two in-scope layers above), not in an end-user
application's UI hints — Menata Runtime does not build that UI, it interprets metadata into it, so
"give the user a hint" is not a lever available at this layer.

**The safeguard: make the companion a required, inline part of the type's own schema — not a second
field the author must remember to add separately.** Menata's schema already does this for other
types: `value_list` requires a `values:` array; `reference` requires a `target_machine:`. The same
discipline extends to `money`:

```yaml
- id: fld_amount
  name: Amount
  type: money
  currency: IDR              # required key of `type: money` itself — omitting it is malformed metadata
```

Metadata declaring `type: money` without a `currency:` (or `currency_field:`, for the multi-currency
tier) is not "the simple case" — it is **incomplete metadata**, exactly as `type: value_list`
without `values:` would be. This turns "the author might forget" into "the metadata cannot describe
a valid `money` field without it" — a schema-completeness property enforced by the language and the
loader, not a matter of authoring discipline or memory.

**Enforced by CAP-X05 (metadata validation before load) — no new capability required.** CAP-X05
already exists to catch dangling references and malformed metadata (Terraform's plan-before-apply is
its yardstick, Study 1). Its scope now explicitly includes: every composite/reference-sugar type
(`money`, and a future `quantity` if registered) must have its required companion present in the
metadata; a missing companion is a load-time rejection — the same discipline CAP-X05 already applies
to a dangling `reference` target under CAP-F13.

**Convention over Configuration still applies — the default must be visible, never silent.** If an
organization only ever uses one currency, requiring `currency: IDR` on every money field is one
short, explicit key, not a burden. The runtime may still apply a workspace-wide default when the key
is omitted — but it must report what it resolved (e.g. a load-time note that a field defaulted to
the workspace base currency) rather than leave the assumption unstated. This closes the "might
forget" risk entirely within the metadata/runtime layer, without needing an interactive, hint-giving
application to compensate for it.

### Where the conversion calculation belongs — not inside the Constraint

A related correction: conversion (looking up a factor and multiplying) must not be embedded inside
a Constraint's logic ("check the unit, if different, convert"). Per spec `004-constraint.md`, a
Constraint expresses a rule to validate — it does not transform data. Mixing "look up and convert"
into a Constraint conflates two Grammar responsibilities that Menata's own language design keeps
separate (each Grammar owns one decision). The correct division:

```text
Raw fields (amount + unit)
        │
        ▼
Computed Field (CAP-F14) — resolves the factor (from Tier 1/2/3, wherever it lives)
        and derives a Normalized Quantity in the item's base unit
        │
        ▼
Constraint (CAP-C05/C07/C10) — validates the already-normalized numbers only
        ("sum(in) − sum(out) ≥ 0"), never touches units or lookups itself
```

This mirrors exactly how `money`'s aggregate constraint (CAP-C10, debit = credit) should work too:
the constraint checks already-comparable numbers; whatever normalizes them into a comparable form
(currency conversion, unit conversion) is a Computed Field concern, not a Constraint concern.

## Boundary Check — Recurring Schedules Belong to Event, Not Field

A recurring operational need — "every day," "every Monday," "repeats every N months" — raised a
different kind of question from money/quantity: not *which* field type to use, but *whether this is
a field concept at all*. Worth answering explicitly here, since it tests the edge of this
framework's own scope (field modeling) against the neighboring Event grammar.

**Recurrence is not a Field concept.** A `date` field holds a single point in time — a value.
Recurrence describes a *pattern of occurrence* — a rule that generates many future points in time.
That is a property of something that *happens* (an Event), not a property of a stored value. Some
tools bundle a "repeat" toggle into their date-picker widget for convenience, but that is the same
widget/type conflation already addressed in "Two Orthogonal Axes" above (§ on `value_list`) — a UI
convenience, not evidence that recurrence belongs to the Date *type*.

**External precedent confirms this placement.** iCalendar (RFC 5545) — the standard behind Google
Calendar, Outlook, and most calendar software — attaches its recurrence rule (`RRULE`) to an
**Event** (`VEVENT`), never to a bare date/timestamp value. This is the most authoritative existing
answer to "where does recurrence live," and it agrees with Menata's own placement.

**Menata already placed it in Event, from the specification onward.** `003-runtime-language.md`
(via the Menata Language's Event grammar) names **Time** as one of four Event sources, with `Every
Day` and `Every Monday` as its own worked examples — the word "Every" is the recurrence signal,
built into Event grammar's vocabulary from the start, not discovered as an afterthought. Case 4
(Maintenance Reminder) already modeled this concretely:

```yaml
evt_mt_due_check:
  trigger: { type: schedule, cron: "0 7 * * *" }   # Event, not Field
```

**Does this fall into a gap if Event only handles single-shot triggers?** No — CAP-E02
("Time-driven event") was registered specifically *because* Time is a recurring source, distinct
from Business Activity events (`When Submit`) which fire once per user action. Two related,
already-registered capabilities cover the full need:

- **CAP-E02** — the recurring trigger itself ("wake up and check every day at 7am")
- **CAP-A11** — date arithmetic reacting to a business event ("advance Next Due Date by Frequency
  when Complete happens")

Both are Event/Action grammar. Neither is a Quantity-style tiered field-modeling problem, because
recurrence doesn't involve an ambiguous *value* needing external reference data to interpret — a
recurrence rule ("every 3 months") is fully meaningful on its own, unlike a bare number needing a
currency or unit to mean anything. It only touches shared/external data at all if the schedule must
respect an organization-wide calendar (skip holidays) — already anticipated as CAP-O06 (Business
Calendar, Study 7), a narrow, optional dependency, not a tiering problem of its own.

Long-term, `user` is not a permanently distinct field type — it is `reference` with a reserved
target, kept as its own named type today only because CAP-O01 (identity & role registry) does not
yet exist to be referenced. The same applies to `money`'s currency component (pending CAP-O02) and
`file` (pending a runtime-managed File/Document entity). Once those exist, each can be redefined as
sugar without changing any existing `.menata` source — a domain expert never sees the distinction;
`- Employee : User`, `- Amount : Money`, and `- Attachment : File` read the same either way.

---

# Final Recap — Every Field Type, Settled Answer

The sections above show the *reasoning journey* across five adversarial passes (that is
deliberate — a framework worth trusting should show its work). This section is the single place to
read the *settled* answer, without re-deriving it — split into three parts that must not be
conflated: actual field **types**, worked **examples** of applying those types to real fields, and
concepts that were tested against this framework and explicitly **excluded** from Field grammar
entirely. Mixing the third group into a "field types" table would misrepresent what was decided.

## Field Types

| Field type | Settled classification | Why | Registry note |
|-----------|------------------------|-----|----------------|
| `text`, `rich_text` | Pure primitive | Open domain, no identity | — |
| `number` (ordinal/count use, e.g. `Sequence`) | Pure primitive | Open domain, no identity, no unit | — |
| `boolean` | Pure primitive | Open domain, no identity | — |
| `date`, `time`, `date_time` | Pure primitive | Open domain; timezone dependency is contextual (org-level, CAP-X09), not a per-field hidden reference | — |
| `duration` | Composite, but stays primitive | Magnitude + unit, but unit set is small/universal/never grows (no exchange-rate-like lifecycle) — unit resolves as an inline `value_list`-shaped selector, not `reference` | — |
| `value_list` | Bounded enumeration (not primitive, not reference) | Closed domain, but fixed — the runtime can validate membership; distinguished from `reference` only by whether the set grows | — |
| `user` | **Reference sugar** | Passes identity/lifecycle test; target is platform identity, not a workspace Machine | CAP-F05 (⚠️ partial — Failure Mode 2), long-term sugar over CAP-F13 + CAP-O01 |
| `money` | **Reference sugar** (amount is primitive; currency is the sugar component) | Currency passes all four supporting tests (identity, lifecycle via exchange rates, reuse, cardinality); independently named as an Object in `specification/001-object.md` | CAP-F17 (❌), currency is a CAP-O02 master-data candidate. `type: money`'s `currency:`/`currency_field:` key should be schema-required, rejected at load if missing (CAP-X05) |
| `file` | **Reference sugar** | Own storage identity + lifecycle (versioning, replacement); matches Frappe/Salesforce/Drupal platform convention (Study 2) | CAP-F06 (⚠️ partial — Failure Mode 2), long-term sugar over CAP-F13 + a runtime-managed File/Document entity |
| `reference` | General mechanism | Target has independent identity and is a workspace-authored (or master-data-designated) Machine | CAP-F13 (❌, Prio 1) |
| `Quantity` (count + Unit of Measure, anticipated for Case 5) — **predicted, not yet a confirmed type** | **Default is simple — declaring the field stays one line, same as any other type.** Only the *conversion schema*, if actually needed, is tiered — a metadata-authoring decision made once, never the same thing as the conversion *values*, which are business data entered later through the resulting application (out of scope here). | Tier 1 (the common case): two flat fields on the referencing Machine's metadata, no reference at all — as easy to declare as any primitive. Tier 2 (multiple unit pairs for one item): a child table (CAP-F16). Tier 3 (factor needs history): a dedicated Machine. Whichever tier, CAP-X05 should reject metadata that declares `money`/`quantity` without its required companion. | Predicted only — no case evidence yet (Case 5 unwritten); default to Tier 1, escalate only when Case 5 actually shows a real need |

## Worked Examples — Applying These Types to Real Fields

These rows are not new field types — they are field *usages* already covered by `reference` above,
recorded because they were the cases that first surfaced a real modeling question.

| Field usage | Settled answer | Why | Registry note |
|-------------|-----------------|-----|----------------|
| `Equipment`-class fields, used within **one** application | Should be `reference` to an ordinary workspace Machine — plain `reference`, nothing more | **Failure Mode 1, but fully resolved by CAP-F13 alone** — author the target Machine (Vehicle Type / Vehicle Asset / Service Record / …, connected purely by `reference`), change the field from `text` to `reference`. No governance capability needed. | CAP-F13 (❌, Prio 1) — same mechanism as any other reference, no special treatment |
| `Equipment`/`Employee`/`Customer`-class fields, referenced from **more than one** application | Still `reference`, but the target Machine's cross-app ownership is undefined | **Failure Mode 1, genuinely needs CAP-O02** — CAP-F13 supplies the pointer mechanics; CAP-O02 answers who owns the Machine and what happens across applications when it's deactivated | CAP-O02 (❌), confirmed by Case 10 (narrative), reinforced by `Currency` (via CAP-F17) |

## Boundary Exclusions — Considered, But Not Field Concepts

These were evaluated *against* this framework specifically to check whether they belonged here —
and the settled answer is that they do not. Listed here so the exclusion itself is not lost, without
polluting the Field Types table above with a "type" that isn't one.

| Concept | Verdict | Why | Where it actually belongs |
|---------|---------|-----|----------------------------|
| **Recurring schedules** (`Every Day`, `Every Monday`, "repeats every N months") | **Not a Field concept.** A `date` value is a point in time; a recurrence rule describes a pattern of occurrence — a property of something that *happens*, not of a stored value. | Spec `003-runtime-language.md`'s Event grammar already names Time as one of four Event sources, with `Every Day` / `Every Monday` as its own examples — recurrence was placed in Event from the start, matching the iCalendar `RRULE` standard (RFC 5545), which attaches recurrence to an Event (`VEVENT`), never to a bare date/timestamp value. | **Event/Action grammar** — CAP-E02 (recurring trigger) + CAP-A11 (date arithmetic reacting to a business event, e.g. advancing a due date), both already registered from Case 4 |

**How to read "settled":** every Field Types row has survived at least the initial pass; `money`,
`file`, and the Quantity tiering additionally survived a correction across the second through fifth
adversarial passes. The Worked Examples and Boundary Exclusion rows exist precisely *because* a
follow-up question forced re-checking something that looked settled. None of this is expected to be
truly final forever — `duration` or `value_list` could in principle be overturned by a future case
the same way `money`/`file` were, if new evidence surfaces. The discipline that matters is
re-running the tests when challenged, not treating any row as beyond question.

---

# Registry Impact

Refines existing entries, no new capability IDs required:

- **CAP-F13** (reference field) — scope note added: must support **three** target flavors from day
  one: (a) a workspace-authored Machine, (b) a reserved built-in identity target (for `user`, once
  CAP-O01 exists), (c) a reserved built-in File/Document target (for `file`, once a runtime-managed
  file entity exists). Designing only for (a) would require a breaking change later.
- **CAP-F05** (`user` field) — relationship clarified: this capability's long-term resolution is
  "sugar over CAP-F13 + CAP-O01", not a permanently separate implementation. Its current ⚠️ partial
  status (free text, no picker) is Failure Mode 2, unaffected by this framework.
- **CAP-F06** (`file` field) — relationship clarified (second-pass correction): this capability's
  long-term resolution is "sugar over CAP-F13 + a runtime-managed File/Document entity" — files have
  their own identity and lifecycle (storage key, versioning, replacement), matching the Frappe
  Attach→File-DocType / Salesforce File-ContentDocument / Drupal file-entity pattern already surveyed
  in `001-platform-capability-survey.md`. Its current ⚠️ partial status ("upload is not stored") is
  the same shape of problem as CAP-F05 waiting on CAP-O01 — not a separate bug, a missing target.
- **CAP-F17** (multi-currency money) — reinforced: this framework independently derives the same
  requirement from first principles (Currency fails the identity/lifecycle/reuse/cardinality tests),
  not only from the Odoo/ERPNext benchmark (Study 6). Currency is also named as an example Object in
  `specification/001-object.md`, independent confirmation from the language spec itself.
- **CAP-O02** (master data designation) — scope corrected (fourth-pass): its evidence is `Currency`
  (via CAP-F17) and the Case 10 cross-application narrative — **not** `Equipment` used only within
  Case 4's own application, which is fully resolved by CAP-F13 alone and needs no governance
  capability. CAP-O02 is specifically about a Machine referenced from **more than one** application;
  it does not gate the basic "this should be reference, not text" fix for a single-application field.
  This narrows the evidence but the requirement still clears the dual-evidence bar (Case 10 + CAP-F17).
- **CAP-F13** (reference field) — scope note reinforced: fixing a mis-modeled field like `Equipment`
  from `text` to `reference`, *when used within one application*, requires nothing beyond CAP-F13
  itself plus authoring an ordinary workspace Machine. CAP-O02 is an additional, separate capability
  for the cross-application case, not a prerequisite for the basic fix.
- **Predicted, not registered:** a future **Quantity** field (Case 5) would be a related instance of
  the same general pattern (count + Unit of Measure) — but *not* a straight copy of the
  `money`/Currency resolution. Third-pass refinement above shows it resolves in **tiers**: a fixed
  conversion pair as two flat fields on Item (no reference needed), escalating to a child table
  (CAP-F16) only with multiple unit pairs, escalating further to a dedicated Machine only if
  conversion history must be preserved. Not added as a capability now — no case evidence yet, per
  the admission test's dual-evidence rule (`capability-lifecycle.md` §2, criterion A1); when Case 5
  is written, it should declare which tier it actually needs rather than assuming the most complex one.
  Fourth-pass note: *declaring* the field stays one line regardless of tier — only the underlying
  conversion data (when Tier 2/3 is genuinely needed) is authored separately, once, by whoever
  manages that master data.
- **CAP-F14** (computed/formula field) — scope clarified: this is the correct home for unit/currency
  conversion calculations (deriving a Normalized Quantity or a base-currency amount), not the
  Constraint grammar. Constraints (CAP-C05/C07/C10) must only validate already-normalized values —
  mixing lookup-and-convert logic into a Constraint's expression conflates two Grammar
  responsibilities that Menata's language design keeps separate.
- **CAP-E02** (time-driven event) and **CAP-A11** (date arithmetic) — reinforced by an explicit
  boundary check: recurring schedules are Event/Action grammar, never a Field concern, confirmed by
  the iCalendar `RRULE` (RFC 5545) precedent of attaching recurrence to an Event, never to a bare
  date value. No scope change — this closes an open question about where recurrence belongs, rather
  than adding new requirements.
- **CAP-X05** (metadata validation before load) — scope extended (fifth-pass): validation must check
  that every composite/reference-sugar field type (`money`, and any future `quantity`) has its
  required companion declared inline (`currency:`/`currency_field:` for `money`) — a missing
  companion is a load-time rejection, the same discipline already applied to a dangling `reference`
  target under CAP-F13. This is the language-level safeguard against a metadata author (human or AI)
  forgetting to set up the conversion mechanism — enforced by the schema and the loader, not by
  hints in the resulting application's UI (out of scope for Menata Runtime, which interprets
  metadata rather than authoring that UI's design).

---

# Guide Impact

`guides/writing-menata.md` §"Tips memilih tipe" (4 informal bullets) is superseded for the
reference-vs-value_list-vs-primitive decision by this framework's decision tree, translated to
plain language for domain experts (no DDD/Codd/MDM terminology — just the four supporting tests).
The guide keeps its own copy, phrased for its audience; this document is the rigorous source it is
derived from.

---

# Maintenance

Re-run the retrofit calibration whenever a new case is written, or when CAP-O01/CAP-O02 land —
at that point, `user` fields and master-data-candidate fields should be re-classified and, where
possible, actually migrated to `reference` in the example metadata.
