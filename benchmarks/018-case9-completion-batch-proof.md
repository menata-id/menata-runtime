# Case 9 Completion Batch — CAP-C08/C10/C11, Proven

> Study 26 of the Capability Roadmap.
>
> `roadmap.md`'s Track C named this batch "real, valuable, independent v1 work" back at B3's own
> status update, corrected from an earlier (wrong) assumption that it was a B3 prerequisite — it
> gates nothing in the Process Overlay and nothing gates it. CAP-C10 (Prio 7) and CAP-C11 (Prio
> 10) are both direct `case-portfolio.md` Case 9 targets (`sum(Lines.Debit) = sum(Lines.Credit)`
> before Post; no posting into a Closed Fiscal Period); CAP-C08 (Prio 14) is the general
> capability the registry's own row already said both are instances of: "Case 9's `Fiscal
> Period.Close` guard is the reverse direction of the pattern... Both directions fall under the
> same capability, not two."

---

# What was reviewed and found

Three questions this study answered before writing any code:

1. **Is there an existing precedent for a Constraint that needs storage access?** Yes —
   `"unique"` (CAP-C12). `constraint.Eval` deliberately never touches the database; `Engine.
   Violations` skips any Constraint with `Operator == "unique"`, and `handler.
   uniquenessViolations` checks it separately, at the same tier as `referenceViolations` (Create/
   Update/CSV-import). This is the exact shape a cross-record check needs — reused directly,
   not reinvented.
2. **Does `AggregateCondition` (CAP-A14) already cover CAP-C10?** No — it compares one `SUM` to a
   literal, gating one Event. CAP-C10 needs to compare *two* sums (debit vs. credit) to *each
   other*. Reused as a building block (`RecordStore.SumField`, called twice) but not as the
   gating shape.
3. **Does CAP-R07 (immutability) already cover CAP-C11?** No — it checks a record's *own* field,
   never a *different*, referenced record's field. The registry's own CAP-C08 note already
   confirms this: it names the period guard as a CAP-C08 instance, not a CAP-R07 one.

---

# What was built

**`Constraint.CrossRecord` (CAP-C08)** — a new `constraints.cross_record` JSONB column, one
discriminated-union type (`Kind: "aggregate" | "reference_field"`), the same pattern
`ChangePolicy` (CAP-W07) already established. `Engine.Violations` skips it, same as `"unique"`;
`handler.crossRecordViolations` (new file, `internal/handler/crossrecord.go`) checks it
separately, gated by the Constraint's own ordinary `Condition` — "only check at Post" needed no
new mechanism.

- **`aggregate` (CAP-C10)**: `SUM(FieldA)` vs. `SUM(FieldB)` (or a literal `Value`) across every
  `ChildMachine` record whose `ScopeField` references this one. Compared **numerically**
  (`compareNumeric`, epsilon-tolerant), not string-equal like `constraint.Eval`'s own `"equals"`
  — debit `"100"` and credit `"100.00"` must compare equal, which a plain string comparison would
  get wrong. This is the one real, easy-to-miss trap this study caught by design rather than by a
  failing test.
- **`reference_field` (CAP-C11)**: looks up the record `ReferenceField` points to, compares its
  `TargetField` against `Value`/`Operator` — reuses `constraint.Eval` directly (a plain field-vs-
  literal comparison is exactly what it already does well; only the aggregate half needed its own
  numeric path).

**New call site: `triggerEvent`/CAP-C09.** `uniquenessViolations`/`referenceViolations` are only
called from Create/Update/CSV-import — never from an Event trigger. CAP-C10/C11 both need to fire
specifically at **Post**, an Event trigger on an already-Draft record, not a Create/Update of
header fields — so `crossRecordViolations` is called there too, a genuinely new addition this
capability's own use case required. Named, not silently generalized: `uniquenessViolations`
itself was not also extended to `triggerEvent` — a real, discovered asymmetry, left as a
documented gap rather than fixed speculatively in the same pass.

**Load-time validation** (`internal/metadata/loader.go`'s `validateReferences`, alongside its
existing CAP-W01 target check — same "wait until everything's loaded" reasoning): `aggregate`
requires `ChildMachine` to exist with a `reference` `ScopeField` back to the declaring Machine and
`number`-typed `FieldA`/`FieldB`; `reference_field` requires `ReferenceField` to be a real
`reference` Field and `TargetField` to name a real Field on its target.

---

# Proof

New, self-contained fixture (`seeds/027_case9_completion_lab.sql`) — the existing, already-
conformance-tested `seeds/008_journal_entry.sql` deliberately left untouched. Its own header
comment already disclaimed CAP-C10 for exactly this reason: adding an unconditionally-checked new
Constraint to an already-tested Machine risks breaking whatever existing fixture data doesn't
happen to balance. `mch_c9_fiscal_period` (with its own `Close` event, entirely HTTP-driven — no
mid-test `psql` needed this time), `mch_c9_journal_entry` (references a Fiscal Period; both new
Constraints, gated `Condition: {status equals Posted}`), `mch_c9_journal_entry_line`.

| Test | Claim | Result |
|---|---|---|
| T161 | An unbalanced entry (100 debit, 50 credit) is rejected on Post | ✅ PASS |
| T162 | A balanced entry (100/100) posts successfully — the positive case, proving T161 isn't vacuously always-reject | ✅ PASS |
| T163 | Posting into a Closed Fiscal Period is rejected, even when balanced | ✅ PASS |
| T164 | Posting into an Open Fiscal Period succeeds — same positive-case pairing as T162 | ✅ PASS |

**164/164 conformance passing, zero regressions on the prior 160** — confirmed on a fully fresh
isolated schema (`CREATE SCHEMA`, clean `migrate-up`+`seed`, a throwaway server on its own port),
first run, zero failures of any kind.

---

# What this does and doesn't close

**Closes:** both of Case 9's own declared CAP-C10/C11 targets, fully and directly — not a subset,
not a simplified stand-in. Also closes the two shapes CAP-C08's own row already named as "both
directions" of one capability.

**Does not close, named explicitly:** a third shape CAP-C08's row itself illustrated but which no
case actually declares as a target — a universal/for-all check across every child record (e.g.
"before closing a Fiscal Period, confirm every Journal Entry in it is already Posted"), structurally
different from both an aggregate-sum comparison and a single reference-field lookup. Not built
this pass; CAP-C08 stays ⚠️ for that reason. Downstream, now unblocked but not attempted:
CAP-V15 (live aggregate preview, UI work, follows CAP-C10) and CAP-V19 (live cross-record balance
preview, follows CAP-C08).

---

# Registry impact

`capability-registry.md`: CAP-C10 and CAP-C11 both ❌→✅. CAP-C08 ❌→⚠️ (two of its own named
shapes done, the third named open). `roadmap.md`'s Track C and "Recommended order" list both
updated to match.
