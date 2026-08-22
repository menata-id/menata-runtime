# Effective-Dated Change Policy — CAP-W07, Proven

> Study 24 of the Capability Roadmap.
>
> `capability-registry.md`'s CAP-W07 row was registered ❌ at Study 20 §6.4 (`benchmarks/
> 012-process-model-synthesis.md`): the synthesized answer to a gap both the comparator BRD
> (blanket version-pinning) and this runtime's own pre-CAP-W07 behavior shared — neither could
> express "this metadata change applies to open cases; that one doesn't." It stayed unbuilt
> because it had no provable precondition: `new_records`/`records_in_states`/`all_records` only
> mean something once metadata can change while records stay open, and this runtime's only deploy
> mechanism was a full restart (CAP-X04's own row: "today: restart required") — a restart reloads
> everything for everyone atomically, so there is no live boundary between "before" and "after"
> to discriminate across. `roadmap.md`'s Sequencing Guide named this explicitly and built CAP-X04
> first, same day (`benchmarks/015-metadata-live-reload-proof.md`); this study is what that
> unblocked.
>
> Method note: the design docs' own sketch grammar (§6.4's sample YAML) was internally
> inconsistent — flagged during this pass's research, not copied verbatim. This study made one
> real scope decision after reading the current engine rather than guessing from the sketch:
> `change_policy` attaches to a **Constraint**, not to the `process` JSONB block. Constraints are
> already this runtime's general "rule" primitive — a `required` Field and CAP-W01's requirement
> counters are both Constraint rows under the hood — so this is the more general, more minimal
> hook, and it works identically whether the Constraint is hand-authored or process-overlay-
> compiled.

---

# What was reviewed and found

Three questions this study answered before writing any code:

1. **Where does a "which records does this apply to" check actually plug in?** CAP-C09's own
   hook (`internal/handler/handler.go`'s `triggerEvent`, and the equivalent Create/Update paths)
   already re-validates every declared Constraint against the record's data at every write —
   simulated post-action data for events, the raw submitted data for Create/Update. A
   `change_policy`-derived Constraint needs nothing new here: it's checked by the exact same
   mechanism every other Constraint already goes through.
2. **Can `records_in_states` reuse the existing state-guard pattern?** Yes, directly —
   `compileRequirements` (CAP-W01) already gates a generated Constraint on `{field: <Status
   field>, operator: equals, value: <to-state>}`. `records_in_states` needs the same shape with a
   *list* of states instead of one, which the engine didn't support (`equals` only compares one
   value) — a small, generically useful `"in"` operator closes that gap.
3. **Can `new_records` reuse anything, or does it need new plumbing?** `records` has a real
   `created_at` column, but `Executor.Simulate` and the raw `data map[string]any` Constraints
   evaluate against **never** expose it — a record's creation time lives on `store.Record`, not in
   the JSONB blob Constraints read. This is the one genuinely new piece of plumbing this
   capability needed: a synthetic, never-persisted key surfaced into that map only where a
   `new_records` policy is actually declared.

---

# What was built

**`change_policy` (new `constraints.change_policy` JSONB column, `migrations/020_change_policy.sql`)**
— attaches to any Constraint. Compiled entirely at load time
(`internal/metadata/compile.go`'s new `compileChangePolicies`, called from `loadMachineDetails`
right after `compileProcess`) into that Constraint's own `Condition` — no engine change:

- `records_in_states: [...]` → `Condition = {field: <Status field>, operator: "in", values: [...]}`
  — validated against the Machine's own Status field and its declared values at load time (the
  same "Unknown = explicit" discipline `validateReferences`/CAP-F13 already established for
  dangling references).
- `new_records` (with `effective_from: "YYYY-MM-DD"`) → `Condition = {field:
  "__created_at__", operator: "on_or_after", value: effective_from}`.
- `all_records` → no-op, explicit-but-inert — the same behavior as declaring no `change_policy`
  at all, nameable on purpose so the choice is auditable in the metadata itself (matching Study
  20 §6.4's own framing: "the compliance case — explicit, not accidental").

**Two small, reusable operators** (`internal/constraint/engine.go`'s `Eval`, registered in
`model.SupportedOperators` for CAP-X05's load-time check): `"in"` (membership against a new
`ConstraintExpression.Values []string`) and `"on_or_after"`/`"on_or_before"` (inclusive siblings
of the existing `after`/`before`, same date-parsing path, `"today"` keyword included).

**`model.ChangePolicyCreatedAtField` (`"__created_at__"`)** — the synthetic key. A new handler
helper, `withChangePolicyCreatedAt(machine, data, t)`, injects it into a **copy** of the data map
(never mutates the original — that same map is what gets persisted as the record's JSONB `data`
column right after the Violations check runs) at all 5 call sites that run `engine.Violations`:
Create (`handler.go`, CSV import, `api.go`'s JSON Create — all use `time.Now()`, since a record
with no persisted id yet is by definition new) and Update / `triggerEvent`/CAP-C09 (both use the
real `rec.CreatedAt`). Gated by a new `Machine.NeedsCreatedAtGuard` bool, set only when a Machine
actually declares a `new_records` policy, so the map-copy cost is paid by zero of the ~30
pre-existing Machines that don't use this.

**Named, deferred limitation, not silently dropped:** `change_policy` cannot be combined with a
Constraint that already has its own `Condition` (e.g. a CAP-W01 requirement counter, which is
already gated on a transition's `to` state) — the loader rejects that combination at load time
with a clear error rather than guessing at how the two conditions should combine. `Constraint`
supports exactly one `Condition`, no AND-list; extending it to `Conditions []ConstraintExpression`
would be a real (if small) refactor, not justified by any case yet.

---

# Proof

New, self-contained fixture: `mch_policy_case` (`seeds/024_change_policy_lab.sql`, boot-time —
three baseline records already open, no `change_policy` anywhere yet) plus
`seeds/025_change_policy_activate.sql` — the actual metadata *change*, two new Constraints,
**deliberately excluded from `make seed`'s boot-time list** (the same pattern CAP-X04's own
`023_reload_lab.sql`/T151 established), applied mid-conformance-run via a direct `psql` call, then
picked up by `POST /admin/reload` — directly reusing CAP-X04 to prove the "records stay open,
metadata changes, some get gated and some don't" claim this capability exists to make.

The two Constraints deliberately gate two *different* fields ("Compliance Note" for
`records_in_states`, "Approval Reference" for `new_records`) rather than sharing one — a record
that happens to be simultaneously "not Draft" *and* "newly created" would otherwise get blocked
by whichever constraint a given test wasn't trying to exercise. Each assertion below fills in the
field the *other* constraint cares about, isolating the one dimension (state vs. date) it proves.

| Test | Claim | Result |
|---|---|---|
| T154 | Before the policy exists, updating the Draft record with a blank required field succeeds | ✅ PASS |
| T155 | `records_in_states: [Draft]` rejects a blank Compliance Note on a Draft record | ✅ PASS |
| T156 | A record already past Draft when the rule arrived is grandfathered — `records_in_states` never reaches it | ✅ PASS |
| T157 | A record created before `new_records`' `effective_from` (2026-01-01) is untouched by the new policy | ✅ PASS |
| T158 | `new_records` rejects a blank Approval Reference on a record created after the effective date | ✅ PASS |

T155/T156 together are the direct proof of the BRD v2 worked example (§12 criterion 7):
`records_in_states: [Draft]` gates open Draft cases without touching ones already past Draft.
T157/T158 are the equivalent proof for `new_records`, discriminating purely on the record's actual
creation date rather than its current state.

**158/158 conformance passing, zero regressions on the prior 153** (150 + CAP-X04's T151-T153 +
these 5) — confirmed twice: once against the shared dev database, and a second time from a fully
fresh isolated schema (`CREATE SCHEMA`, clean `migrate-up`+`seed`, a throwaway server on a
different port, per this codebase's own `CLAUDE.md`-documented isolation pattern) specifically to
rule out any state accumulated by this session's own repeated manual test runs against the shared
database — T151/T65/T70/T118 all showed transient, unrelated failures on the shared DB from
running the full suite several times in a row without a reset in between (pre-existing
non-idempotency in a handful of count/mid-run-seed-based tests, not something this study
introduced), and all passed cleanly on the fresh schema.

---

# What this does and doesn't close

**Closes:** the actual gap Study 20 §6.4 named — a metadata author can now declare, per rule,
whether a change reaches in-flight work or only new work, materialized as an ordinary guard with
no new engine mechanism and no version-pinned metadata caches. `brd-menata-runtime-v2.md`'s
§12 criterion 7 is directly demonstrated, not just designed.

**Does not close, named explicitly:** combining `change_policy` with a Constraint that already
has its own `Condition` (documented above). Declaring `change_policy` inside the `process` JSONB
block itself (e.g. scoping one specific `requirements[]` entry) — this pass's Constraint-level
attachment covers process-overlay-compiled Constraints too (they're ordinary Constraint rows by
the time they'd need this), so no case has yet demanded the nested form. A UI affordance for
authoring `change_policy` (today: direct metadata/seed authoring only, same tier as every other
Constraint) — consistent with this codebase's existing "form-based authoring, not a visual
builder" non-goal.

---

# Registry impact

`capability-registry.md`'s CAP-W07 row moved ❌→⚠️ (the Condition-combination limitation is the
named reason it isn't ✅). `roadmap.md`'s Sequencing Guide: B5 marked done; the "Recommended order
for upcoming sessions" list's #2 struck through.
