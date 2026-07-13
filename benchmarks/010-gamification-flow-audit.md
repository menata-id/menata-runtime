# Gamification Flow Audit

> Prompted directly by a question about Case 12 (Community/gamification): if the DB table and
> page for a "points" Machine already satisfy CRUD, where does the *interaction* layer live —
> how is an action turned into points, how are points configured, how are they displayed? Answering
> that question required tracing every existing gamification-shaped artifact in this repo, not
> just reading the registry rows. This study records what that trace found.
>
> Distinct from a capability admission study (`008`, `009`) — **no new capability is proposed
> here**. Every mechanism this domain needs is already ✅ (CAP-A06, CAP-A14, CAP-I01, CAP-I03,
> CAP-I05, CAP-V05, CAP-V13). The finding is an **integration debt**: those mechanisms have never
> been proven working *together*, end-to-end, against one running server.
>
> Status: v0.1 | Created: 2026-07-13

---

# The finding, stated precisely

**Three gamification-shaped artifacts exist in this repo. None of them is a complete, running,
conformance-tested proof of the full flow (action → points → threshold → reward → display).
Each proves a different slice, and the three were never reconciled with each other.**

| # | Artifact | Runs on a real server? | What it actually proves | What it skips |
|---|----------|------------------------|--------------------------|----------------|
| 1 | `docs/examples/community-{group,membership,event,points,badge-award}.{menata,yaml}` (Case 12) | **No** — never seeded, no `seeds/*.sql` file references any `mch_group`/`mch_membership`/`mch_point_ledger_entry`/`mch_badge_award` id. Pure design document (`case-portfolio.md`: "target declaration") | The only one with a full business narrative (Group → Membership → Points → Badge) and the only one that names 4 distinct point-earning reasons | Never executed; internal wiring gaps below were never caught because nothing forces this metadata through a loader |
| 2 | `prototype/go/seeds/009_action_lab.sql` (`mch_al_point_entry` / `mch_al_badge`) | **Yes** — conformance T72–T73 | CAP-A14's aggregate gate only: `SUM(points) >= 100` correctly blocks/allows a system-triggered follow-on `create_record` | Points are entered through a plain Create form by role `PM` and the gate event is triggered manually by `PM` — no automatic "action earns points" wiring at all (no CAP-A06/CAP-I01 in this file); no Leaderboard/My Points view, just a plain list |
| 3 | `prototype/go/seeds/014_integration_lab.sql` (`mch_int_points`) | **Yes** — conformance T104–T108 | CAP-I01 (decoupled subscription) + CAP-I03 (contract-gated) + CAP-I05 (per-subscription weight, two unrelated publishers — Order Placed at 10, Referral Completed at 25 — feeding one shared ledger) | No CAP-A14 anywhere in this file — points accumulate but nothing ever reads the threshold and unlocks a reward; plain list view, no Leaderboard |

No file in this repo exercises columns [1]+[2]+[3]+[4]+[5] of the chain
(`source action → weighted link → accrual → threshold gate → display`) together. Proof 2 covers
only [4]; proof 3 covers only [1]+[2]+[3]; proof 1 covers the narrative but was never run, so its
own internal gaps (next section) were never surfaced by a test.

---

# Case 12's own paper design has an un-flagged internal gap

`community-points.menata`/`.yaml` declares `Point Ledger Entry.Reason` as
`Joined Group | Posted Status | Hosted Event | Attended Event` — four ways to earn points. Tracing
each one across all five Case 12 files:

| Reason | Wired to a real triggering event? | Evidence |
|--------|-----------------------------------|----------|
| `Joined Group` | **Yes** | `community-membership.yaml:37-40` — `evt_mem_join` action `create_record: {machine: mch_point_ledger_entry, fields: {points: 10, reason: "Joined Group"}}` |
| `Posted Status` | **No** | No `Post` Machine exists anywhere in Case 12's five files. The enum value names a feature that was never designed, not even stubbed |
| `Hosted Event` | **No** | `community-event.yaml` declares `Event` with **zero events** (`permissions: Admin - Create Event` is the only trigger-shaped thing, and it isn't wired to points) |
| `Attended Event` | **No** | RSVP/Attendance was explicitly deferred ("composable from CAP-F20 without new design effort" — `case-portfolio.md` line 282-283) but never actually composed; there is no Machine to fire this Reason from |

**Only 1 of 4 declared point-earning reasons has real wiring.** This is not marked anywhere —
every other genuine gap in these `.yaml` files carries a `[NOT YET]`/`[PARTIAL]` inline annotation
per this docs folder's own stated convention (`docs/examples/README.md`: "inline annotations...
`[SUPPORTED]`, `[NOT YET]`, `[PARTIAL]`"), but `fld_pts_reason` itself is marked plain
`[SUPPORTED]` — true only for the field *type* (`value_list`), silent on 3 of its 4 values having
no machine behind them. This is exactly the kind of gap `capability-registry.md` Rule 3 ("new gaps
register here first") and `capability-lifecycle.md`'s "silence is not a decision" principle exist
to prevent — it slipped through because this file was never run, only read. **Fixed by this study**
(annotation-only, see Immediate Fixes below).

---

# Two competing architectures for "action → points," never reconciled

Two different mechanisms link a source action to a points record, and this repo uses **both**,
in two different examples, with no stated reason to prefer one:

- **Direct coupling (CAP-A06)** — Case 12's own design: `Membership.Join`'s own action list
  contains a literal `create_record: {machine: mch_point_ledger_entry, ...}`. The Membership
  Machine's metadata must name the Points Ledger Machine by id.
- **Decoupled subscription (CAP-I01 + CAP-I05)** — Integration Lab: `Points Ledger` declares a
  `event_subscriptions` row naming `Order.Place Order` as its publisher. `Order`'s own metadata
  never mentions Points Ledger at all.

`CAP-I05`'s own registry description is explicit about which one gamification should use:
*"Cross-cutting contribution declaration on events (**weights to gamification/KPI machines**)"* —
naming gamification as the motivating case for the *decoupled* pattern, precisely because a real
points ledger is fed by many unrelated publishers (join, post, host, attend, refer, purchase, ...)
that should never each need to know a Points Ledger exists. Case 12's own paper design does the
opposite — it hardcodes the dependency in the *publisher's* (Membership) own metadata, which
doesn't scale past one wired Reason and is architecturally the pattern CAP-I01/Pattern C exists to
avoid. **If Case 12 is ever built for real, it should be redesigned onto CAP-I01/CAP-I05, not
CAP-A06** — matching Integration Lab's proof, not its own current draft.

---

# The display layer is unproven for this domain

`community-points.menata` declares two Views — `My Points : List` and `Leaderboard : Report` —
mapping cleanly to already-✅ mechanisms (CAP-V05's `$current_user` filter, CAP-V13's
`group_by`+`sum` aggregate report). Both mechanisms work and are conformance-tested, but **neither
has ever been tested against a points/gamification dataset** — CAP-V05's proof is "My Overdue
Tasks" (`seeds/010_views_lab.sql`), CAP-V13's proof is a Trial Balance (`seeds/008_journal_entry.sql`).
Composing them for a leaderboard is a reasonable inference from the registry, not a demonstrated
fact. A `group_by: fld_pts_member, sum: fld_pts_points` report view scoped to a `text`/`reference`
member field (not yet the `$identity` target — see below) has an open question worth checking
directly: whether `RecordStore.SumFieldsGroupedBy` handles a large `group_field` cardinality (one
row per distinct member) the same way it handles Trial Balance's small `Account` cardinality.

---

# Why this happened — methodology, not oversight

This isn't a random gap. It's a direct consequence of two of this project's own standing rules:

1. **Admission test A4 (non-composability, `capability-lifecycle.md` §2)** — a case is only
   "worth" a full build-out if it proves something *new*. Once CAP-A14 was admitted via Case 12's
   narrative, the actual conformance proof (Action Lab) was deliberately built as small as
   possible — just enough to isolate the one new mechanism (the aggregate gate), stripped of
   everything CAP-A14 didn't need to prove (no Membership, no Group, no automatic point-earning).
2. **Cases 11–21 are explicitly "target declarations" (`case-portfolio.md` header), not
   implementations.** Their whole purpose is to *discover and register* capabilities
   (CAP-F20, CAP-C12, CAP-A14 all trace to Case 12), not to ship a working feature. The
   business-readable narrative exists to justify admission; nobody was ever tasked with actually
   running it.

The result: the artifact that reads as a coherent business feature (Case 12) never ran, so its
internal gaps were invisible; the artifacts that ran and passed conformance (Action Lab,
Integration Lab) were each narrowed to a single mechanism and lost the business narrative in the
process. No single artifact in this repo is a trustworthy blueprint for "how to actually build
gamification here" — that has to be hand-assembled from the registry, which is exactly what this
audit did.

---

# Recommendation (future work, not admitted here)

Not a capability proposal — every mechanism needed already exists and is ✅ (fails admission test
A4, non-composability). What's missing is a **unified reference implementation**: one seed file,
one running server, one conformance test, exercising the full chain for real:

1. `Group`/`Membership`/`Event` as in Case 12, but with at least 2 real point-earning triggers
   wired (not 1) — e.g. `Membership.Join` and `Event.Host` (drop or genuinely design `Posted
   Status`/`Attended Event` rather than leaving them as unwired enum values).
2. Wired via **CAP-I01/CAP-I05** (decoupled subscription), not CAP-A06 — matching CAP-I05's own
   stated rationale, correcting Case 12's current draft.
3. **CAP-A14** aggregate gate on the shared Points Ledger, same shape as Action Lab, unlocking a
   real `Badge Award` `create_record`.
4. **CAP-V05** `My Points` (`$current_user` filter) and **CAP-V13** `Leaderboard`
   (`group_by`+`sum`) actually seeded and hit once each, closing the open question above.
5. Registry rows CAP-A14/CAP-I01/CAP-I05/CAP-V05/CAP-V13 gain a cross-reference note to this
   combined proof once it exists, the same way CAP-F17/CAP-F19 already cross-reference each other
   as "pure composition" proofs.

Sizing: no new migration, no new Go code expected (every mechanism is implemented) — this is a
seed-file + conformance-test-only effort, closer in shape to Batch 9/10's composition work than to
a new capability build. A reasonable next-session task.

---

# Immediate fixes applied by this study

Annotation-only, no behavior change — closes the un-flagged gap found above, per
`capability-lifecycle.md`'s "silence is not a decision":

- `docs/examples/community-points.yaml` — `fld_pts_reason` annotation now states 3 of 4 values are
  unwired.
- `docs/examples/community-event.yaml` — noted that `Event` has no events at all, so
  `Hosted Event`/`Attended Event` cannot fire.
- `case-portfolio.md` Case 12 entry — cross-referenced to this study.
- `capability-registry.md` — CAP-A14/CAP-I01/CAP-I05 rows gain a note pointing here.
