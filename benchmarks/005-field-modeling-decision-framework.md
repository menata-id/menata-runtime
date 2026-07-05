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
| `File` / `Attachment` | 1, 3 | Primitive value (stored blob), no identity | `file` — **correct type, Failure Mode 2** (CAP-F06) |
| `Equipment` | 4 | Identity ✓ (reusable asset), not a workspace Machine, not platform identity | **Failure Mode 1** — currently mis-modeled as `text`; should be `reference` once an Equipment Machine + CAP-O02 exist |
| Employee / Customer / Equipment (cross-application) | 10 (narrative) | Same as Equipment, generalized | **Failure Mode 1** — the general case that motivated registering CAP-O02 in Study 7 |

**Calibration result:** the framework reproduces every prior ad hoc decision correctly (no field
that was already right gets flagged wrong), and it correctly isolates `Equipment` as the one true
modeling gap versus the many merely-unimplemented-yet-correct choices. This mirrors the calibration
discipline used for the capability admission test in Study 9 (`capability-lifecycle.md` §6).

---

# Field Type Inventory Reconciliation

Classifying every type in `runtime-metadata-schema.md` against the framework:

| Category | Types |
|----------|-------|
| **Pure primitives** (never reference) | `text`, `rich_text`, `number`, `money`, `boolean`, `date`, `time`, `date_time`, `duration`, `file` |
| **Reference sugar** (resolves through `reference`, to a special built-in target) | `user` (target: platform identity, pending CAP-O01) |
| **General mechanism** | `reference` (target: any workspace Machine, or a master-data-designated Machine per CAP-O02) |
| **Bounded enumeration** | `value_list` (not a reference — a closed, stable set declared inline) |

Long-term, `user` is not a permanently distinct field type — it is `reference` with a reserved
target, kept as its own named type today only because CAP-O01 (identity & role registry) does not
yet exist to be referenced. Once CAP-O01 lands, `type: user` can be redefined as sugar without
changing any existing `.menata` source (a domain expert never sees the distinction — `- Employee : User`
reads the same either way).

---

# Registry Impact

Refines existing entries, no new capability IDs required:

- **CAP-F13** (reference field) — scope note added: must support two target flavors from day one:
  (a) a workspace-authored Machine, (b) a reserved built-in identity target (for `user`, once CAP-O01
  exists). Designing only for (a) would require a breaking change later.
- **CAP-F05** (`user` field) — relationship clarified: this capability's long-term resolution is
  "sugar over CAP-F13 + CAP-O01", not a permanently separate implementation. Its current ⚠️ partial
  status (free text, no picker) is Failure Mode 2, unaffected by this framework.
- **CAP-O02** (master data designation) — reinforced: `Equipment` (Case 4) is now a second concrete,
  named example of the same gap Case 10 (Study 7) first surfaced, strengthening the dual-evidence
  requirement (`capability-lifecycle.md` §2, criterion A1).

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
