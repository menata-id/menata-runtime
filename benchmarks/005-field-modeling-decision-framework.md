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
> Status: v0.1 | Created: 2026-07-05

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
| `File` / `Attachment` | 1, 3 | **Identity ✓** (own storage key, versioning, lifecycle) | **Reclassified — see below.** Was scored as Failure Mode 2 in the first pass; a second pass (prompted by a direct question) shows this was wrong — see "Second-Pass Corrections" |
| `Equipment` | 4 | Identity ✓ (reusable asset), not a workspace Machine, not platform identity | **Failure Mode 1** — currently mis-modeled as `text`; should be `reference` once an Equipment Machine + CAP-O02 exist |
| Employee / Customer / Equipment (cross-application) | 10 (narrative) | Same as Equipment, generalized | **Failure Mode 1** — the general case that motivated registering CAP-O02 in Study 7 |

**Calibration result:** the framework reproduces almost every prior ad hoc decision correctly, and
correctly isolates `Equipment` as a true modeling gap versus merely-unimplemented-yet-correct
choices. One field type (`file`) was initially miscategorized — corrected below. This mirrors the
calibration discipline used for the capability admission test in Study 9
(`capability-lifecycle.md` §6): a good framework should survive a second, adversarial pass, and
this one found something on the second pass — which is a feature, not a failure, of doing the pass.

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
third instance of the exact `money`/currency pattern, expected when Case 5 (Inventory / Stock
Movement) is written. Not corrected now — Case 5 doesn't exist yet, so there is no case evidence,
only the benchmark-side prediction. Recorded here so the case-portfolio process (declare targets
first) can check this prediction against what Case 5 actually surfaces.

Long-term, `user` is not a permanently distinct field type — it is `reference` with a reserved
target, kept as its own named type today only because CAP-O01 (identity & role registry) does not
yet exist to be referenced. The same applies to `money`'s currency component (pending CAP-O02) and
`file` (pending a runtime-managed File/Document entity). Once those exist, each can be redefined as
sugar without changing any existing `.menata` source — a domain expert never sees the distinction;
`- Employee : User`, `- Amount : Money`, and `- Attachment : File` read the same either way.

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
- **CAP-O02** (master data designation) — reinforced twice: `Equipment` (Case 4) and now `Currency`
  (via CAP-F17) are two concrete, named, independent examples of the same gap Case 10 (Study 7) first
  surfaced, strengthening the dual-evidence requirement (`capability-lifecycle.md` §2, criterion A1)
  well past the minimum bar.
- **Predicted, not registered:** a future **Quantity** field (Case 5) would be a third instance of
  the same pattern (count + Unit of Measure). Not added as a capability now — no case evidence yet,
  per the admission test's dual-evidence rule (`capability-lifecycle.md` §2, criterion A1).

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
