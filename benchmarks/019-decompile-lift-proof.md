# Decompile-Lift — CAP-W05 Backward Direction (B6), Proven

> Study 27 of the Capability Roadmap.
>
> `roadmap.md` and `capability-registry.md` have tracked "B6" and "CAP-W05 backward direction" as
> the same open item since B2 shipped the forward direction: the process-map View already
> *renders* a legible diagram for a hand-authored Machine with no `process` block at all (T142,
> "decompile" in this project's own vocabulary — proven on real Case 1/2 metadata, not a
> purpose-built fixture) — but that reconstruction lands in a UI-only display list, not a
> re-loadable `process` declaration. "Lift" is producing that declaration.
>
> **Built deliberately ahead of case evidence, named honestly, not glossed over.** Research done
> before writing any code found `case-portfolio.md` names zero cases needing "migrate a
> hand-authored Machine to the Process Overlay" — by `capability-lifecycle.md`'s own admission
> bar (A1, dual evidence), this is evidence-thin, the same posture as parked rows like
> CAP-W08/CAP-V11. That gap was surfaced directly to the user (`AskUserQuestion`) rather than
> silently built past or silently dropped; the explicit answer was to proceed anyway. Scope was
> kept correspondingly narrow specifically to bound the risk of that departure.

---

# What was reviewed and found

Two questions this study answered before writing any code:

1. **Does `extractProcessMap` already produce anything reusable?** Yes and no. It already does
   the hard part — reconstructing States/Transitions/Actor from a Machine's Status Field and its
   Events' CAP-E06 state guards, working identically whether the Machine is process-compiled,
   hand-authored, or a genuine pre-existing v1 Machine that never heard of `process`. But it
   returns `ui.ProcessEdge` — `Actor` pre-joined into one display string ("Worker (owner:
   Assignee)"), no `Requirements`/`Auto`/`on_transition` captured at all. Not `model.Process`-
   shaped, not re-loadable.
2. **What's the actual scope boundary?** Reconstructing States/Transitions/Actor/`on_transition`/
   `Auto` is unambiguous — it's exactly what `extractProcessMap` already detects, just
   un-flattened. Reconstructing `Requirements`/`SLA`/`change_policy` is not: a hand-authored
   counter Field + gating Constraint pair looks structurally identical whether or not it started
   life as a CAP-W01 requirement declaration. No case forces solving that ambiguity, so this study
   doesn't attempt to — named as a boundary, not silently absent.

---

# What was built

**`liftProcess(machine) (*model.Process, error)`** (`internal/handler/processmap.go`, alongside
`extractProcessMap`, reusing its exact state-guard detection) — a two-pass reconstruction:

1. **Pass 1** identifies which state-guarded Events are auto-shaped (no Permission grants them —
   CAP-E05's own convention, the same signal `extractProcessMap`'s "System" fallback already
   uses).
2. **Pass 2** builds the real `Process`: an auto-shaped Event becomes a `ProcessAuto{From, To}`
   entry; everything else becomes a `ProcessTransition` with its actor un-flattened back into
   `{Role, OwnerField}` from whichever Permission grants it, and every Action beyond the first
   state-setting `set_field` captured as `on_transition` — **except** a `trigger_event` action
   whose target is itself auto-shaped, deliberately excluded.

That exclusion is the one real bug this design caught before it shipped, not after: `compileProcess`
regenerates an `Auto` entry's `trigger_event` chain structurally from the declaration itself. A
hand-authored Machine that ALSO happens to carry that exact chain as an explicit action (as
`mch_ca_manual`'s own `Submit` transition does, chaining to `evt_cam_auto`) would, if that action
were blindly copied into `on_transition` too, fire the chain **twice** once the lifted output is
recompiled — verified by hand against the actual JSON output before writing the round-trip test,
not discovered by a failing assertion.

**`GET /{machineID}/process-lift`** (new route, Admin-only — matching CAP-X08's
`APIExportApplication` Admin-gated JSON-export pattern) returns the result labeled a draft:
```json
{"draft": true, "note": "Draft lifted from mch_ca_manual -- review before pasting into a Machine's process column.", "process": {...}}
```
Never auto-applied — consistent with this project's own "form-based authoring, not a visual
builder" non-goal.

---

# Proof — and a real finding, not a workaround

Lifting `mch_ca_manual` (`seeds/019_overlay_lab.sql`) reproduced its exact declared shape,
confirmed by hand against the raw JSON before any assertion was written — states, all six
transitions with correct actors (including the `Worker (owner: fld_cam_assignee)` narrowing), and
the `Auto: Submitted → Review` entry, with **no spurious duplicate `trigger_event`** in `Submit`'s
`on_transition` — the double-chain exclusion working as designed.

Reapplying that same JSON to a **different** Machine (`mch_ca_lifted`, `seeds/028_lift_lab.sql`)
surfaced something the single-machine lift alone couldn't: `actor.owner_field` names
`fld_cam_assignee` — `mch_ca_manual`'s own field id. Field ids are a single global primary key
across the whole `fields` table, not scoped per Machine, so that exact id cannot exist on a
second Machine at all. This is a genuine, structural property of the metadata model, not a bug in
`liftProcess` — a rule that says "narrow this role to the assignee field" has no portable
identity beyond the literal id it happens to be stored under on its own Machine. The proof
performs the one-time translation a human reviewing the draft would (swapping the field id before
applying it elsewhere) — exactly the "review before pasting" step the API response's own `note`
already names, not a cover-up of a failure.

| Test | Claim | Result |
|---|---|---|
| T165 | `GET /mch_ca_manual/process-lift` returns valid Process JSON for an Admin; a non-Admin gets 403 | ✅ PASS |
| T166 | That JSON (with the one necessary field-id translation), applied to a fresh Machine and reloaded (CAP-X04, zero restart), drives the identical Open→Assigned→In_Progress→Submitted→(auto)Review→Verified→Closed lifecycle T136/T137 already proved for the hand-authored/compiled pair | ✅ PASS |

**166/166 conformance passing, zero regressions on the prior 164** — confirmed on a fully fresh
isolated schema, first clean run after fixing one test-fixture bug caught along the way (a missing
`form` View on `mch_ca_lifted` — without one, `buildFormFields` renders zero fields, unrelated to
`liftProcess` itself).

---

# What this does and doesn't close

**Closes:** B6/CAP-W05's backward direction as tracked since B2 — a Machine's process shape can
now be extracted as re-loadable metadata, not just rendered. Demonstrates the same "compiled
equals hand-authored" equivalence standard this session's other Process Overlay proofs already
hold to, in the reverse direction.

**Does not close, named explicitly:** reverse-engineering `Requirements`/`SLA`/`change_policy`
(genuinely ambiguous from the compiled shape alone, no case forces resolving it). Cross-machine
reapplication of a lift involving `owner_field` always needs a manual field-id translation — not a
limitation this pass could design around, a structural property of globally-unique Field ids.

---

# Registry impact

`capability-registry.md`'s CAP-W05 row moved ⚠️→✅ (both directions now done), with the
evidence-thin admission-discipline departure recorded directly in the row's own note, not
silently dropped. `roadmap.md`'s B2/B6 entries and "Recommended order" list item both marked done.
