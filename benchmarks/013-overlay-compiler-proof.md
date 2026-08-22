# Process Overlay Compiler — B1 Parity Proof

> Study 21 of the Capability Roadmap.
>
> The falsifiable experiment named as the first step of implementing
> `brd-menata-runtime-v2.md`: can a declared `process` block be compiled, deterministically, into
> the substrate primitives Menata Runtime already executes (Events, CAP-E06 guards,
> CAP-P01/P02 Permissions) — with **identical behavior** and **identical runtime cost** to the
> equivalent hand-authored metadata, and with the runtime itself carrying **zero knowledge** that
> a process was ever declared?
>
> Method note, since this study is unusual for the series: it reports a real implementation
> (`internal/metadata/compile.go`, `migrations/019_process_overlay.sql`,
> `seeds/019_overlay_lab.sql`, conformance T136–T139), not a paper survey — the same "cases
> prove, benchmarks guide" discipline this repo already follows, applied to a concept study
> rather than a business case. Context: as of this study, `menata.app` is a **development**
> deployment, not production (corrected 2026-08-22, see `roadmap.md`) — this let the experiment
> touch the loader directly instead of routing through an external preprocessor first.
>
> Status: v1.1 — + Addendum: B2 (process map, CAP-W05 forward direction) implemented and
> conformance-proven, including the decompile claim on genuine pre-existing v1 metadata |
> Created: 2026-08-22

---

# The experiment

Two Machines in a new `app_overlay_lab` Application carry the **same** process — the comparator
BRD's own worked example (Study 19 Appendix §38, Corrective Action), reduced to Process Overlay
B1's scope (states, transitions, actor, one automatic transition — no requirements/SLA/quorum
yet, those are B3/B4):

```text
Open → Assigned → In_Progress → Submitted → (auto) → Review → Verified → Closed
                                                          ↓ Revise
                                                     In_Progress
```

- **`mch_ca_manual`** — hand-authored the v1 way: explicit `events` rows, explicit CAP-E06
  `condition` guards, explicit `event_actions`, explicit `permissions` rows, an explicit
  `Status` `value_list` Field, and an explicit CAP-E05 `trigger_event` chain for the automatic
  `Submitted → Review` step.
- **`mch_ca_overlay`** — declares **only** a `process` JSONB block
  (`migrations/019_process_overlay.sql`, one column on `machines`, same design choice as
  `machines.config`). No `events`, `permissions`, or `Status` field rows exist for it anywhere
  in the seed file.

`internal/metadata/compile.go`'s `compileProcess` runs inside `loadMachineDetails`, after every
hand-authored table has already loaded, and expands the declaration into exactly the same
in-memory structs (`[]*model.Event`, `[]*model.Permission`, a generated `value_list` `Field`) the
manual arm's own loader queries produce — deterministic ids (`evt_<machine>_<slug(name)>`,
`perm_<machine>_<slug(role)>`, `fld_<machine>_status`), so identity stays stable across reloads
(`004-runtime-metadata.md` §Stable Identity).

---

# Results

## Behavioral parity — conformance T136–T139

| Test | Claim | Result |
|---|---|---|
| T136 | Compiled Machine boots, renders, generated Status field starts at the declared initial state | ✅ PASS |
| T137 | Full lifecycle (Open→…→Closed, including the un-clicked automatic Submitted→Review step) succeeds identically on both arms | ✅ PASS |
| T138 | Hand-authored control arm rejects wrong-state (400), wrong-role (403), non-owner (403) | ✅ PASS |
| T139 | Compiled arm rejects the identical three cases with identical codes | ✅ PASS |

**140/140 conformance tests passing** — the four new ones, and **zero regressions** across the
135 that existed before this study (the compiler only activates for a Machine that declares
`process`; every other Machine's load path is untouched).

## Architectural parity — the "zero runtime knowledge" claim, checked directly

```text
$ grep -rln "\.Process\b|model\.Process" internal/ \
    | grep -v "metadata/loader.go\|metadata/compile.go\|model/model.go"
(no output)
```

No file in `internal/handler`, `internal/executor`, `internal/permission`, `internal/constraint`,
or `internal/router` references `model.Process` at all. The compiler runs exactly once, at boot,
inside `metadata.Loader`; everything downstream — `Interpreter`, `Guard.CanTrigger`,
`Executor.Simulate`/`Persist`, `constraint.Engine.Violations`, the entire HTTP handler layer —
operates on `model.Event`/`model.Permission`/`model.Field` structs it cannot distinguish from
hand-authored ones, because they are not distinguished: same struct, same fields, same
`triggerEvent` call path (CAP-E06 guard → CAP-P02 ownership → CAP-C09 constraint re-check →
`Persist` → CAP-A07/A08/E05 workflow actions → CAP-I01 subscriptions).

## Cost parity — the DoD §6.6 claim

`brd-menata-runtime-v2.md` §6.6 requires: *a compiled transition must run at the same statement
count as a hand-authored one.* This follows from the architectural parity result above, not as a
separate measurement: `triggerEvent`'s SQL work (one `SELECT` for the record, one `UPDATE`, one
`INSERT` into `record_events`) is a function of the `model.Event`/`model.Constraint` structs it
receives, not of where those structs came from. Since compilation happens once at boot and
produces byte-identical struct shapes to the manual arm, there is no code path in which a
compiled transition could cost more (or less) than its hand-authored equivalent — the claim is
true by construction, not merely observed. (A live query-log measurement is left as a
mechanical follow-up if this ever needs a number for an external audience; the structural
argument is what the DoD actually asked for.)

---

# What this proves, and what it doesn't yet

**Proves:** the core mechanism of Concept C (`benchmarks/012-process-model-synthesis.md` §6,
"declared process, emergent execution") works exactly as designed for its B1 scope — states,
transitions, role/ownership actors, one automatic transition. The three-way claim (behavior
identical, runtime blind to the declaration, cost identical) all held on first implementation,
with real conformance proof, not just a design argument.

**Does not yet prove:** the harder B3/B4 claims — requirement cardinality with write-time
fan-in (CAP-W01), quorum (CAP-W03), declared SLA (CAP-W04), effective-dated change policy
(CAP-W07) — none of which this experiment exercised. Those are later steps in the v2
implementation plan, each its own falsifiable proof against this same conformance discipline.

**A scope boundary found while building, named not silently dropped:** B1 currently requires a
Machine to declare *either* a `process` block *or* hand-authored `events` — never both
(`compileProcess` errors at load if it sees both). No case has yet needed a Machine that mixes a
declared core process with a few hand-authored side events; the rule exists to keep the merge
semantics unambiguous rather than guess at one, per this project's own "Unknown = explicit"
discipline (`capability-lifecycle.md` §4 rule 3). Revisit if a real case forces it.

---

# Registry impact

- **Workflow row** (`capability-registry.md`, "Tracked but Not Yet Studied") updated: the HOLD
  posture on the Process Overlay is lifted for **B1 specifically** — it is no longer only
  Proposed, it has a working, conformance-proven implementation. B2–B6 (peta proses,
  requirement generik, quorum/SLA, `change_policy`, decompiler) remain Proposed/HOLD, each
  gated on its own proof the same way B1 just was.
- **CAP-W05** (process map) row: B1's compiled result is now real data a future read-only view
  can render directly — no longer purely speculative.
- New capability implied by this study, not yet named in the registry: **compiling a declared
  process into substrate primitives at load time** — recorded here as the B1 mechanism itself
  rather than a new CAP-ID, since `capability-registry.md`'s own Workflow section already
  tracks it under CAP-W01–W07 as compile *products*; this study is what makes those rows
  no longer purely hypothetical for the B1 subset.

---

# Maintenance

- Re-run T136–T139 after any change to `triggerEvent`, `Guard.CanTrigger`, or
  `metadata.Loader` — they are the regression guard for the parity claim itself, not just for
  the overlay lab's own two Machines.
- When B3 (requirements) lands, extend `seeds/019_overlay_lab.sql`'s two arms with a
  requirement-bearing transition and add the matching parity tests, same method as this study.

---

# Addendum — B2: the process map, and the decompile claim (2026-08-22, same day)

Study 20's synthesis (§6.5) named a "two-way door" as a design pillar: a stabilized
hand-authored Machine should be *liftable* into a Process Overlay declaration, and the same
information should be readable back out as a map regardless of which way a Machine was
authored. B2 (CAP-W05, forward direction) tests the readable-map half of that claim, and
deliberately does it in a way that also tests something B1 didn't: **whether the map-derivation
logic itself treats compiled and hand-authored Machines identically, and whether it works on
Machines that never heard of the Process Overlay at all.**

## Design choice that makes this a decompile proof, not just a rendering feature

The map is **not** rendered from `machine.Process` (the raw declaration) — that would only work
for the one Machine that declares it. It is derived from the same shape B1's own parity proof
already established as universal: a Machine's `value_list` "Status" Field plus every `Event`
whose `Condition` is a state-equality guard (CAP-E06) and whose actions include a matching
`set_field`. Since B1 proved compiled and hand-authored Machines produce byte-identical
`Event`/`Permission` structs, this same extraction function (`internal/handler/processmap.go`'s
`extractProcessMap`) works on any of the three:

1. An overlay-compiled Machine (`mch_ca_overlay`).
2. A hand-authored Machine sharing the identical process (`mch_ca_manual`).
3. A genuine **pre-existing v1 Machine that predates the Process Overlay by weeks**
   (`mch_leave_request`, Case 2's original design from the very first capability batches).

## Results

| Test | Claim | Result |
|---|---|---|
| T140 | Compiled Machine's map lists all 7 states (initial marked) and 6 transitions with correct actors, including the auto step (labeled "System" — no human Permission grants it) | ✅ PASS |
| T141 | Hand-authored Machine's map is the byte-identical assertion list — legibility parity, not just execution parity | ✅ PASS |
| T142 | Leave Request's map reconstructs `Draft→Submitted→{Approved,Rejected}` plus `Cancel`, entirely from Events written before `process` existed | ✅ PASS |
| T143 | A Machine with no `process_map` View declared 404s — the same opt-in every other auxiliary View type (`report`/`calendar`/`dashboard`) already uses | ✅ PASS |

**144/144 conformance passing, zero regressions on the prior 140.** A manual render of T142's
page (`GET /mch_leave_request/process-map` as the seeded Employee account) confirms the exact
readable output: `States: Draft (initial) Submitted Approved Rejected Cancelled` /
`Submit Draft→Submitted Employee` / `Approve Submitted→Approved Manager` /
`Reject Submitted→Rejected Manager` / `Cancel Submitted→Cancelled Employee` — a correct,
legible reconstruction of a process that was never once written down as a `process` block.

## What this closes

Study 20 §4.3's named gap — "no single legible process artifact" — is now closed for **any**
Machine with a Status-guarded shape, not only overlay-declared ones. This is a stronger result
than B2 was scoped to prove: the map isn't a feature of the Process Overlay, it's a feature of
the *substrate shape* B1 revealed every state-guarded Machine already has, whether declared or
hand-authored. No metadata migration, no re-authoring of the other 20 case-portfolio Machines
is needed to get a process map for each of them — the same `process_map` View row, dropped into
any of their seed files, would work today.

## Registry impact

`capability-registry.md`'s CAP-W05 row: forward direction (render) implemented and
conformance-proven, including the decompile case on real v1 metadata — the row's own
speculative "derivable... from any hand-authored Machine's own Events+guards" language is no
longer speculative for the render half. The backward-authoring half (drafting a `process` block
*from* a rendered map, i.e. "lift") remains open.
