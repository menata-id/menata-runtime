# Quorum's Declarative Form — CAP-W03, Proven

> Study 25 of the Capability Roadmap.
>
> CAP-W03's quorum-core landed same-day as B4 (2026-08-22) with a deliberate, named gap:
> `doAggregateStatus`'s new `min_approvals` parameter proved N-of-M parallel approval works, but
> only hand-authored — `roadmap.md`'s own B4 status update called expressing it through
> `process.requirements[].type: approval` "a genuinely new cross-machine compile capability the
> Process Overlay hasn't needed before," named explicit future work rather than attempted
> speculatively. This study builds that capability.

---

# What was reviewed and found

Three questions this study answered before writing any code:

1. **What's the actual blocker?** `compileProcess`/`compileRequirements`
   (`internal/metadata/compile.go`) only ever touch the one `*model.Machine` they're given, and
   `internal/metadata/loader.go` loads+compiles Machines one at a time, `ORDER BY name`
   (alphabetical, not dependency order) — so a parent's declarative requirement can't reach into a
   child Machine's own compiled Events while the loader is still mid-load. Confirmed by reading
   the actual loop structure, not assumed.
2. **Does a fix already exist for a similar timing problem?** Yes — `validateReferences`
   (`internal/metadata/loader.go`) already validates a CAP-W01 `evidence` requirement's `target`
   only *after* every Workspace/Application/Machine has fully loaded, specifically because "a
   Requirement may target a Machine in an Application that hasn't loaded yet at the point its own
   Machine compiles" (its own doc comment). This study reuses that exact checkpoint rather than
   inventing a second one.
3. **What does the hand-authored shape actually need, precisely?** Read `handler.doAggregateStatus`
   directly rather than working from `runtime-metadata-schema.md`'s older sketch grammar
   (`{type: approval, target: mch_x_step, quorum: "2_of_3"}`). Two findings that reshaped the
   grammar, both diverging from that sketch, named explicitly:
   - `doAggregateStatus` re-tallies fresh from every sibling record on every call and reads the
     CURRENT value of a `value_list` field literally named `"Decision"` — it never needs to be
     told which value means "approved." So neither does the compiler: it injects onto *every*
     target Event that sets the Decision field, found generically, no `approve_state`/
     `reject_state` params needed.
   - "M" (total voters) is never a declared constant in the existing hand-authored shape either —
     `doAggregateStatus` computes it as "however many sibling records currently reference this
     parent," a runtime fact. The sketch's `"2_of_3"` implied a fixed M; the actual mechanism has
     none. Only `min_approvals` (the "N") needed a place in the grammar.

---

# What was built

**Grammar** — `model.ProcessRequirement` (`internal/model/model.go`) gains `MinApprovals int`,
`OnQuorumApproved string`, `OnQuorumRejected string` (the last two: transition *names* on the
declaring Machine's own process, not raw event ids — an author writes what a human would type,
the compiler resolves the deterministic id).

**Per-machine validation** — `compileRequirements` now accepts `type: "approval"` alongside
`"evidence"`. For approval, it validates locally (no cross-machine wait needed for this part):
`min_approvals > 0`; both `on_quorum_*` names resolve to real transitions on this Machine's own
`process.transitions`; and — **enforced, not just conventional** — each of those transitions'
`actor.role` must be exactly `"System"`. A quorum-controlled outcome reachable by a human role
would make the quorum guarantee directly bypassable, so this is a load-time error, not a lint
suggestion. Nothing is generated on the declaring Machine for this type — no counter Field, no
gating Constraint (unlike `evidence`) — because `approval` compiles to an *injected Action on a
different Machine*, which can only happen once every Machine has loaded.

**The cross-machine pass** — new `compileApprovalRequirements` (`internal/metadata/loader.go`),
called from `LoadAll` right after `validateReferences` succeeds (deliberately not folded into
that function — keeps "generic target+backref check" separate from "actually compile and
inject"). For each declared `approval` requirement: resolves the target's `"Decision"` field
(must be `value_list`, must declare both `"Approved"` and `"Rejected"`) and its back-reference
field to the parent (`model.FindReferenceFieldTo`); scans every Event on the target whose
`Actions` include a `set_field` on the Decision field and appends a new `aggregate_status`
`EventAction`, `Params` built to match the hand-authored shape (`seeds/022_quorum_lab.sql`)
exactly — including one real gotcha caught by reading `doAggregateStatus`'s own type assertion
before writing this code: `params["min_approvals"].(float64)` expects a `float64` (every other
action's Params round-trips through `json.Unmarshal`, where JSON numbers always decode that way);
this action is built purely in-memory, so it must set `float64(r.MinApprovals)` explicitly, not a
plain Go `int`, or the assertion silently fails and quorum never fires — no error, just nothing
happening, the worst kind of bug to debug blind.

**`model.FindFieldByName`/`model.FindReferenceFieldTo`** — moved (mechanical, unchanged bodies)
from `internal/handler/handler.go` to `internal/model/model.go` and exported, so the new
loader-side pass could reuse the exact heuristics `handler.doAggregateStatus`/other CAP-A07/A08
code already relies on, without a package cycle (`internal/metadata` can't import
`internal/handler`).

---

# Proof

New, self-contained fixture pair (`seeds/026_quorum_declarative_lab.sql`, boot-time — no
`DATABASE_URL`-gated mid-run seed needed, unlike CAP-X04/CAP-W07's own recent proofs) mirroring
`022_quorum_lab.sql`'s exact shape: `mch_ql2_request` (parent, now **process-overlay-declared** —
`Submit` carries the `approval` requirement, `Approve`/`Reject` are `System`-only outcomes) and
`mch_ql2_vote` (child, **still hand-authored**, proving the target doesn't need to be
process-declared too) — with **zero `aggregate_status` action written by hand** anywhere in the
file; the loader injects it.

| Test | Claim | Result |
|---|---|---|
| T159 | 2-of-3 votes Approved (3rd left Pending) reaches Approved without waiting on the 3rd — identical to T149's own hand-authored assertion | ✅ PASS |
| T160 | 2-of-3 votes Rejected reaches Rejected once quorum is mathematically impossible — identical to T150's own assertion | ✅ PASS |

**Negative proof** (this codebase's "dual evidence, positive and negative" NFR gate) — two
load-time-failure fixtures, applied against a throwaway isolated schema, confirming the server
refuses to boot with a specific, correct error rather than silently accepting broken metadata:
1. `on_quorum_approved` renamed to a transition with a human actor role →
   `"transition \"Approve\" is named as an approval-quorum outcome but its actor role is
   \"Requester\", not \"System\""`.
2. Target machine with no Event that sets its own `"Decision"` field →
   `"target has no event that sets its own \"Decision\" field, nothing to attach quorum
   tallying to"`.

**160/160 conformance passing, zero regressions** — confirmed on a fully fresh isolated schema
(`CREATE SCHEMA`, clean `migrate-up`+`seed`, a throwaway server on its own port; single run, zero
failures of any kind) and again on the shared dev database (only the same five pre-existing,
already-diagnosed non-idempotency artifacts from this session's own repeated manual runs —
T65/T70/T118/T151/T154 — appeared; none of them touch anything this study built).

---

# What this does and doesn't close

**Closes:** the actual gap B4's own status update named — quorum can now be authored the same way
`evidence` already is, inside `process.requirements[]`, with the compiler doing the cross-machine
wiring a human previously had to type by hand into the *target* Machine's own events. Directly
proves the general "cross-machine compile capability the Process Overlay hasn't needed before"
claim, in a form narrow enough to stay provable: one requirement type, one injected action shape,
reusing an existing load-time checkpoint rather than restructuring the loader.

**Does not close, named explicitly:** more than one `approval` requirement per Machine (no
dedup/merge logic across transitions the way `evidence` has — no case has needed a second yet, and
building it speculatively would violate this project's own "escalate only when cardinality
demands it" discipline). A UI affordance for authoring `process.requirements[].type: approval`
(today: direct metadata/seed authoring only, same tier as every other Process Overlay
declaration).

---

# Registry impact

`capability-registry.md`'s CAP-W03 row moved ⚠️→✅ (both core and declarative form now proven).
`roadmap.md`'s Sequencing Guide: Track B's Quorum line and the "Recommended order for upcoming
sessions" list's #3 both marked done.
