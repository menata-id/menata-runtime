# Metadata Workflow Orchestration BRD Benchmark

> Study 19 of the Capability Roadmap.
>
> Maps Menata Runtime against an external comparator BRD — a "Metadata-Based Workflow
> Orchestration Application" (v1.0, Draft, 2026-08-22) supplied directly by the repo owner —
> a State/Requirement/Transition-centric process engine (Workflow → State → Requirement →
> Transition → Next State), positioned between Camunda-style process orchestration and
> Frappe/Drupal-style application platforms.
>
> The full comparator BRD is preserved verbatim in the Appendix so it can be re-read without
> depending on the conversation that produced this study.
>
> Status: v1.1 — partially revised by Study 20 (`012-process-model-synthesis.md`), same day: the
> 30-concept mapping and the verbatim Appendix stand; §6's framing of CAP-W01–W05 as five
> independent additive layers is superseded (they are now compile products of one Process Overlay
> mechanism), and CAP-W02's blanket version pinning is superseded as a target by CAP-W07's
> effective-dated change policy | Created: 2026-08-22

---

# How to read this document

Same three-layer discipline as `000-workflow-patterns-mapping.md`, applied to a whole external
BRD instead of a pattern catalog: for every named comparator concept, this study asks what
Menata Runtime's closest realization is today (by `CAP-…` ID, `capability-registry.md`), how
complete that realization is, and whether the gap is real or a difference in architectural
philosophy rather than missing capability.

This is a **registration-only** study, matching `008-ui-workflow-interaction-benchmark.md`'s own
closing rule: findings are logged as candidate capabilities below and in `capability-registry.md`
("Tracked but Not Yet Studied" → Workflow row, updated), not implemented in this pass.

---

# 1. What the comparator BRD actually is

Stripped to its core model (comparator §1, §9–14):

```text
WORKFLOW
   ↓
STATE
   ↓
REQUIREMENT
   ↓
TRANSITION
   ↓
NEXT STATE
```

Workflow is a **first-class, versioned metadata object** (comparator §9–10, §30–31) with its own
lifecycle (`DRAFT → PUBLISHED → ACTIVE → DEPRECATED → ARCHIVED`), composed of States,
Transitions, Actors, Requirements, Conditions, Actions, and an SLA — and a running case
(**Workflow Instance**) is permanently pinned to the Workflow Version it was created under.

The comparator's own differentiator claim (§50, §58) is not "having a workflow" — it is a
**generic, extensible Requirement contract** (§35): every state declares what must be fulfilled
(`Type, Target, Required, Cardinality, Actor, Condition, Validation`) before a transition is
permitted, and the seven starting Requirement types (`FORM, ENTITY, TASK, APPROVAL, EVIDENCE,
DOCUMENT, DECISION`) are meant to grow (`SIGNATURE, PAYMENT, MEASUREMENT, INTEGRATION, …`)
**without changing the core workflow model** (§47).

---

# 2. Menata Runtime's own starting position

`capability-registry.md`'s own "Tracked but Not Yet Studied" table already named the gap this
study formalizes, before this study existed:

> **Workflow** | none | Not yet studied as its own concept — current registry treats workflow as
> emergent from Event+Constraint+Permission+Action, per `runtime/004`'s own stated design
> ("Workflow behavior should emerge from events, constraints, permissions, actions"); revisit if
> a case shows this is insufficient.

`004-runtime-metadata.md` §Workflow states the design position directly:

> Workflow behavior should emerge from: events, constraints, permissions, actions. Business
> processes should remain declarative.

So Menata Runtime today has **no first-class Workflow, State, Transition, or Requirement
object**. "Process" is an emergent property of one Machine's own metadata:

- **State** = the current value of a `value_list` Field conventionally named `Status`.
- **Transition** = an `Event`, gated by `events.condition` (`CAP-E06`, state-conditional
  availability — the WCP-18 Milestone fix) and by a re-checked `Constraint` at trigger time
  (`CAP-C09`).
- **Requirement** = not a declared object at all — it is whatever combination of Fields,
  Constraints, child Machines, and approval-step Machines a case author hand-builds to make a
  transition practically impossible without the data/activity being present.
- **Actor** = `Permissions` (role, `CAP-P01`) + `owner_field` record-level ownership (`CAP-P02`)
  + `input_fields`-based delegation (`CAP-P04`).
- **Action** = the `Executor`'s action vocabulary (`set_field`, `create_record`,
  `cross_set_field`, `notify`, `trigger_event`, `batch_generate`, …).

This is the single fact that shapes every finding below: the comparator BRD asks "does Menata
Runtime have a Workflow Runtime component for X" — and the honest answer is almost always "no
distinct component, but the same job is done by Y, one layer lower and less generic."

---

# 3. Concept-by-concept mapping

Marks: ✅ fully covered · ⚠️ partial (primitive exists, generic contract doesn't) · ❌ no
equivalent · `~` philosophical difference, not a capability gap.

| # | Comparator concept (§) | Menata Runtime equivalent | Mark | Note |
|---|------------------------|---------------------------|------|------|
| 1 | Workflow as first-class object: Version/States/Transitions/Actors/Requirements/Conditions/Actions/SLA (§9) | None — emergent from one Machine's Events+Constraints+Permissions+Actions | ❌ | The structural gap everything else below flows from |
| 2 | Workflow Definition vs Instance, instance pinned to definition version (§10, §30) | Metadata is loaded whole-process at boot (`CAP-X04` live-reload ❌, restart required); no per-record version pin | ❌ | Real gap for long-running cases — see §5 below |
| 3 | State w/ Actor/Requirement/Condition/Transition/SLA/Entry/Exit rule (§11) | `value_list` Field value + `CAP-E06` state-guarded Events | ⚠️ | State guard exists; no declared per-state Requirement list, no Entry/Exit rule hook |
| 4 | Transition blocks until Requirement satisfied (§12, §14) | `CAP-E06` (state guard) + `CAP-C09` (constraint re-check at trigger) block on the record's **own field data** only | ⚠️ | Blocks on data-shape, not on "has a required child Task/Evidence/Approval actually been completed" |
| 5 | Requirement Engine, generic & extensible (§13, §35, §47) | None | ❌ | **Largest single gap** — see §4 |
| 6 | Form Requirement (§15) | Ordinary Machine + `form` View (`CAP-V01`) | ✅ | Fully equivalent, just not declared as a state-attached "requirement" |
| 7 | Entity Requirement — request or create another entity (§16) | `CAP-A06` `create_record`, `CAP-F13` reference field | ✅ | Composable already; Case 3/8/14 all prove this shape |
| 8 | Task Requirement w/ Assignee/Due Date/Priority/Checklist/Result/Evidence (§17) | No `Task` primitive — a case models its own Task-shaped Machine by hand (Case 4 Maintenance, Action Lab's Task) | ⚠️ | Works via ordinary Machine composition, not a reusable Task requirement type |
| 9 | Evidence Requirement w/ cardinality (min N photos) (§18) | `CAP-F06` file field (real upload+compress) exists; no generic "at least N of type X attached" cardinality check | ⚠️ | The comparator's own worked example ("Minimum 2 photos") has no direct Menata Runtime equivalent — closest is `CAP-C08`'s cross-record aggregate shape, never wired for evidence counting |
| 10 | Approval Requirement w/ Approve/Reject/Revision decision (§19) | `CAP-A07` (sequential guard) + `CAP-A08` (aggregate rollup: all-approved / first-rejected) | ✅ | Proven on Case 3, real and working, just hand-composed per case rather than declared once |
| 11 | Decision from Form/Approval/Rule/Condition routing next transition (§20) | `CAP-A09` (conditional actions, `if` inside events) + `CAP-C04`/`CAP-C07` (condition grammar) | ✅ | Genuine strength — reused uniformly across Events, Constraints, and Views (`CAP-V09`) |
| 12 | Actor: user/role/group/manager/entity-owner/requester/previous-actor/dynamic-resolver (§21) | Role (`CAP-P01`), single-hop ownership (`CAP-P02`), delegation (`CAP-P04`); Group/Team (`CAP-O07`) explicitly deferred | ⚠️ | Multi-hop resolver (`entity.branch.manager`) only reachable via manual `CAP-F13` reference chaining per case, no declared path-expression mechanism |
| 13 | Actor Resolution engine (metadata → actual user) (§22) | `Guard.CanTrigger` (role) + `owner_field` comparison (user id, per `CAP-F05`) | ⚠️ | Single-field resolution only; no chained/computed resolver |
| 14 | Condition Engine over entity/form/context/actor/requirement-result (§23) | `constraint.Eval` shared grammar, reused by Events/Constraints/Views | ✅ | Cannot reference "requirement completion result" since Requirements aren't objects — see #5 |
| 15 | Revision Loop w/ reviewer/timestamp/reason/comment/prior submission/count (§24) | `CAP-E01` (WCP-10 arbitrary cycles) proven on Case 7's Reopen (counter + state guard) | ⚠️ | The cycle mechanic works; structured revision history (reason/comment/prior-submission) has no generic object — hand-modeled per case |
| 16 | Parallel Workflow w/ ALL / ANY / N_OF_M (§25) | `CAP-A08`: ALL-approved and first-reject-cancels only | ⚠️ | No configurable `N_OF_M` quorum |
| 17 | Action Engine (`CREATE_TASK/CREATE_ENTITY/UPDATE_ENTITY/NOTIFY/ASSIGN_ACTOR/START_TIMER/STOP_TIMER/WRITE_AUDIT/TRIGGER_WORKFLOW`) (§26) | `CAP-A01–A15` — richer in several respects (date arithmetic `CAP-A11`, ordinal stepping `CAP-A12`, aggregate-gated actions `CAP-A14`, batch generation `CAP-A15`) | ⚠️ | No explicit timer object (`START_TIMER`/`STOP_TIMER`); no `TRIGGER_WORKFLOW` since no Workflow object exists to trigger; no "On State Entry/Exit" hook — only "On Event"/"On Trigger" |
| 18 | Event sources: Business Activity/Time/Date/External (§27) | `CAP-E01`/`E02`/`E03`/`E04` — all ✅ | ✅ | Fully covered, arguably ahead (real background scheduler, idempotent webhook `CAP-X13`) |
| 19 | SLA & Timer w/ Warning/Breach/Escalation as a declared `State.SLA` (§28) | `CAP-A11` (business-day-aware date arithmetic, `CAP-O06`) + `CAP-E02`/`E03` (time/date events) + Case 7's proven SLA-breach → auto-escalate chain | ⚠️ | The primitives are real and proven end-to-end in a real case, but assembled from 3–4 primitives per case, not one declared `State.SLA{duration, warning_threshold}` property. `CAP-V17` (countdown badge) still ❌ |
| 20 | Notification channels: IN_APP/EMAIL/PUSH/WHATSAPP (§29) | `CAP-A10` — in-app only, real (inbox + unread badge + digest preference) | ⚠️ | 3 of 4 channels genuinely absent — "no mail infrastructure in this prototype," named not faked |
| 21 | Workflow Versioning, instance pinned to version (§30) | Same as #2 | ❌ | |
| 22 | Workflow Lifecycle DRAFT→PUBLISHED→ACTIVE→DEPRECATED→ARCHIVED (§31) | No such lifecycle for any metadata artifact — metadata loads wholesale at boot | ❌ | Applies to ALL Menata Runtime metadata, not workflow-specific |
| 23 | Workflow Runtime components (§32): Resolver/StateManager/TransitionEngine/RequirementEngine/ActorResolver/ConditionEngine/ActionEngine/EventEngine/SLA-TimerEngine/NotificationEngine/AuditEngine | Interpreter+Executor+Guard+`constraint.Engine` map onto 7 of 11 named components (Condition≈`constraint.Eval`, Action≈`Executor`, Event≈scheduler+`triggerEvent`, Actor Resolver≈`Guard`, Audit≈`CAP-R04`, Notification≈`CAP-A10`, Transition≈`triggerEvent` itself) | ⚠️ | **State Manager and Requirement Engine are the two structurally missing components** — exactly the two the registry already flagged as unstudied |
| 24 | Workflow Builder UI, form-based acceptable (§41) | No metadata-authoring UI exists anywhere in this prototype (`CAP-X08`'s own note: "Machines/Fields/Events are only ever created via seed SQL") | ❌ | Not workflow-specific — true of all Menata Runtime metadata authoring; the Authoring Layer is explicitly a separate, external concern per `002-architecture.md` |
| 25 | Workflow Validator (start/end state reachable, dangling transitions, actor/requirement/form/entity/condition valid) (§42) | `CAP-X05` (load-time validation): dangling references (`CAP-F13`), unrecognized operators, `owner_field` type-check — same *discipline*, applied to what exists today | ⚠️ | The fail-loud-at-load-time principle is already real and arguably stronger (server refuses to boot on a bad metadata file); the specific checks named don't apply 1:1 since State/Transition/Requirement objects don't exist to validate |
| 26 | Audit Trail: Actor/Transition/PrevState/NewState/Timestamp/Comment/RelatedData (§43) | `CAP-R04` — append-only at the DB level (`REVOKE UPDATE/DELETE`), actor + `correlation_id` (`CAP-I04`) shared across cascades | ⚠️ | Missing a structured "Comment" capture on a transition; no UI to browse history yet (named gap already in `CAP-R04`'s own row) |
| 27 | Security: `EXECUTE transition()` never `UPDATE state`; Actor→Permission→Requirement→Condition→Transition check order (§44) | Exactly Menata Runtime's own architecture already: users trigger named Events, never write `Status` directly; `Guard` checks role→ownership→SoD→state-guard→constraint before `Persist` | ✅ | **Strongest alignment point** — the comparator's core security principle is already Menata Runtime's default, not a gap |
| 28 | Reliability: atomic transition, retry/idempotency (§45) | `CAP-X12` (one Postgres tx per request) + `CAP-X13` (webhook idempotency via `INSERT…ON CONFLICT`) | ✅ | Fully covered, already production-hardened |
| 29 | Performance: metadata caching, sync/async split, queue for long processes, stateless engine (§46) | Metadata cached in memory at boot (fast, but see `CAP-X04`); engine is stateless per-request (real ✅) | ⚠️ | No sync/async action split — every action runs inline in the triggering request's own transaction; no queue for long-running actions |
| 30 | Extensible Requirement types (§47) | N/A — no Requirement type registry to extend | ❌ | Depends entirely on #5 |

**Scorecard:** of 30 mapped concepts, 6 ✅ fully covered, 16 ⚠️ partial (a real primitive exists,
one layer below the comparator's generic contract), 8 ❌ no equivalent. The 8 hard gaps cluster
into exactly three structural absences (§4).

---

# 4. The three structural gaps (not thirty small ones)

Reading the 30-row table by cause, not by symptom, collapses to three root gaps:

## 4.1 No generic Requirement contract (rows 5, 8, 9, 12–13, 16, 19, 30)

The comparator's §35 `Requirement{Type, Target, Required, Cardinality, Actor, Condition,
Validation}` is a **type-level** abstraction: declare `EVIDENCE, cardinality=2..*` once and the
engine enforces "at least 2 files of this kind" on *any* state, in *any* workflow, without new
code. Menata Runtime has no equivalent generic layer — every one of the seven requirement types
is independently, manually composed per case:

- Form requirement = an ordinary Machine (real, ✅, just not declared as a requirement)
- Approval requirement = Case 3's Approval Document/Step pattern (real, ✅, hand-built per case)
- Evidence cardinality = **no equivalent at all** — `CAP-F06` stores files, but nothing counts
  them against a minimum
- Task/Decision requirement = hand-modeled Machines, not a reusable type

This is why the comparator BRD's own worked example (§14, "SUBMIT_RESULT requires Completion
Form + Completion Photo + Completion Report") is buildable in Menata Runtime **today**, but only
as three separately-designed mechanisms bolted together per case — not as three lines declaring
`Requirement(FORM), Requirement(EVIDENCE, cardinality=1), Requirement(DOCUMENT)`.

## 4.2 No metadata versioning with instance pinning (rows 2, 21, 22)

`004-runtime-metadata.md` names versioning as an aspiration ("Runtime Metadata should support
versioning… enables… rollback… auditing") but nothing in `capability-registry.md` implements it.
`CAP-X08` (metadata package export/import) is ⚠️ — export only, no import, no per-record version
pin. Today, changing an Event's `condition` in production metadata affects **every open record
immediately** (and even that requires a server restart — `CAP-X04` is ❌). For a long-running
business process — the comparator's own Corrective Action example can sit in `REVIEW` for weeks —
this means a metadata change made for next month's cases can silently reshape what's required to
finish *this* month's already-in-flight ones. The comparator's Workflow Definition/Instance split
with version pinning exists specifically to prevent that; Menata Runtime has no such guarantee.

## 4.3 No first-class Workflow object / process map (rows 1, 3–4, 17, 23–25)

Because there is no Workflow object, there is also no single artifact a person can read to see
"what does this process look like end to end" — the shape is implicit across one Machine's
Fields (for state), Events (for transitions, each carrying its own scattered `condition`), and
`Machine.Config` (for the handful of workflow-shaped settings `CAP-X03` already carries, e.g.
Approval's `approval_mode_field`). This also explains why `State Entry/Exit` hooks and
`TRIGGER_WORKFLOW` have no equivalent — there is no state-lifecycle event to hook, and no
Workflow to trigger.

---

# 5. Flexibility assessment — which model is actually more flexible

The honest answer is two-sided, and the split runs along exactly one axis: **how structured is
the process, and is it bound to one Entity or many.**

## 5.1 Where the comparator BRD's model is genuinely more flexible

- **Reusable requirement types** (§4.1 above) — once, an org declares what "Evidence" or
  "Approval" means as a requirement type; every new workflow reuses it. Menata Runtime re-derives
  the equivalent composition by hand each time a new case needs it — more configuration effort
  per new process even though zero code is written, which works against the very "faster
  application development" business value (comparator §51) both platforms claim.
- **Version-pinned instances** (§4.2) — for long-running, SLA-bound processes (exactly what
  "aplikasi kerja" means in practice: corrective actions, purchase approvals, audits spanning
  weeks), instance-safety under metadata change is a real production requirement Menata Runtime
  does not yet guarantee at any granularity.
- **One legible process document** (§4.3) — an administrator (or an auditor) can read one
  Workflow Definition and see the whole shape: states, requirements, actors, SLA, in one place.
  In Menata Runtime the same information is correct and enforced, but scattered — legible to the
  runtime, not to a human without reading every Event on the Machine.
- **Configurable `N_OF_M`** (row 16) vs Menata Runtime's fixed ALL/first-reject shape — a real,
  narrower gap for "3 of 5 committee members" business rules.

## 5.2 Where Menata Runtime's model is genuinely more flexible

- **No forced two-layer authoring step.** The comparator requires designing a Workflow *before*
  attaching it to an Entity (§37's own diagram: Entity → Workflow Instance → Form/Task/Approval/
  Evidence/Document/Decision, a hub-and-spoke shape with Workflow Instance as the hub). Most real
  "aplikasi kerja" needs (Case 1 Design Request, Case 2 Leave Request) are simple CRUD + a
  handful of state-guarded transitions — forcing every one of those through a full
  Workflow-Definition/Version/Publish apparatus is overhead the comparator's own model cannot
  avoid, while Menata Runtime's "Infer Before Configure" principle (`001-design-principles.md`)
  lets the simple case stay simple: Fields + Events + Constraints, nothing else to design first.
- **Unstructured / CMMN-style case management is a first-class fit, not a workaround.** The
  comparator's model is explicitly State/Transition-centric ("user must not change state
  directly, user must perform a valid transition," §12) — built for closed, designed process
  graphs (BPMN/WCP-style), and its own Non-Goals (§48) disclaim being a full BPMN engine without
  addressing CMMN-style ad-hoc case management at all. Menata Runtime's Case 7 (Customer
  Complaint) is documented in `case-portfolio.md` as exactly this shape — "Unstructured case
  management (CMMN-style): ad-hoc steps, SLA, escalation, reopen" — and it works *because*
  nothing forces the process into a rigid graph: Reopen, Delegate, and ad-hoc Escalate are just
  ordinary Events with guards, addable at any time without redesigning a Workflow object.
- **Cross-Machine orchestration doesn't need a Workflow to mediate it.** `CAP-I01`
  (cross-machine event subscription, decoupled pub/sub — a SUBSCRIBER declares interest, the
  PUBLISHER never names its subscribers), `CAP-A13` (cross-record field write), `CAP-A06`
  (create_record), and `CAP-A14` (aggregate-conditioned actions across records) already let one
  Machine's event ripple into other Machines' own independent lifecycles — proven on Case 10
  (Organization Composite) and Case 12 (gamification: two unrelated publishers feeding one shared
  Points Ledger). The comparator's own §40 Audit example (Audit → Finding → Corrective Action,
  three Entities under one Workflow) needs the SAME primitives Menata Runtime already has
  (create-on-transition, cross-record write) — but the comparator wants them mediated by one
  Workflow object owning the whole span, where Menata Runtime lets three independently-designed
  Machines collaborate without any one of them being "the" orchestrator. Neither is strictly
  more powerful here — the comparator's version is more *legible as one process*; Menata
  Runtime's version is more *evolvable*, since a fourth Entity can join the collaboration later
  by simply subscribing, without editing a central Workflow definition at all.
- **Security, atomicity, and idempotency are already implemented, not just specified.** The
  comparator states §44–46 as requirements; Menata Runtime has working, tested implementations
  for the equivalent surface today (`CAP-X02` real auth+CSRF, `CAP-X06` Postgres RLS multi-
  tenancy, `CAP-X12` transactional atomicity, `CAP-X13` idempotent webhooks) — this is not a
  flexibility axis so much as a maturity one, but it is real evidence about which model is
  closer to something a real "aplikasi kerja" could run on today.

## 5.3 Cross-check against the comparator's own three worked examples

| Comparator example (§) | Nearest proven Menata Runtime case(s) | Buildable today? | What's harder / missing |
|---|---|---|---|
| Corrective Action (§38): single-entity linear+revision-loop, Form+Photo+Report requirements at two states | Case 7 (Reopen cycle, SLA, escalation) + Case 3 (approval steps) + `CAP-F06` (photo evidence) + `CAP-E01` (WCP-10 cycles) | Yes, by composition | "Minimum 2 photos" cardinality has no generic check (§4.1); process shape lives implicitly across one Machine's Events, not as one document |
| Purchase Request (§39): single-entity, amount-conditioned branch, multi-department linear handoff | Case 3 (conditional branch via `CAP-A09`/`CAP-C04`) + Case 6 (fund-balance cross-record constraint, `CAP-C08`) + Case 9 (multi-department handoff via role-gated Events) | Yes, no real gap | None specific — this is the comparator example Menata Runtime already fits most cleanly |
| Audit → Finding → Corrective Action (§40): ONE workflow spanning THREE Entities, with a fail branch looping back two Entities up | No single case proves this exact shape; nearest primitives are `CAP-A06` (create-on-transition) + `CAP-I01` (cross-machine subscription) + `CAP-A13` (cross-record write) | Yes, by composition, but as three separately-designed Machines calling each other | No single artifact describes "the Audit workflow" as one thing — an administrator must read three Machines' own Events to reconstruct the flow. This is the sharpest instance of gap §4.3 |

**Conclusion:** for the majority of real "aplikasi kerja" cases — single-entity, state-guarded,
role-gated business processes (which both worked examples #1 and #2 above, and the bulk of the
existing 21-case portfolio, actually are) — Menata Runtime's emergent model already covers the
same ground the comparator BRD asks for, generally with equal or greater primitive richness, at
the cost of more per-case authoring effort and less legibility as a single reviewable document.
The comparator's model pulls ahead specifically where **many similar processes need to reuse the
same requirement shape**, where **long-running instances need version safety**, or where **one
process spans several Entities and needs to be seen as one thing** — none of which is about raw
capability so much as about generic reusability and legibility at scale.

---

# 6. Candidate capabilities registered (not implemented)

Following `008-ui-workflow-interaction-benchmark.md`'s own precedent: named here and cross-
referenced into `capability-registry.md`, not built in this pass.

| Candidate | Description | Comparator ref | Nearest existing primitive |
|-----------|--------------|-----------------|------------------------------|
| **CAP-W01** | Generic `Requirement` contract (`{type, target, required, cardinality, actor, condition}`) attachable to a state-guarded Event, reusable across Machines without re-deriving each requirement type by hand | §13, §35, §47 | Composes `CAP-E06` (state guard) with a new declared cardinality/count check; Evidence-cardinality is the sharpest concrete case (§4.1) |
| **CAP-W02** | Metadata versioning with per-record version pin — an in-flight record keeps evaluating against the Event/Constraint definitions active when it was created, not the latest deployed metadata | §10, §30 | Extends `CAP-X08` (export, ⚠️) and depends on `CAP-X04` (live reload, ❌) — a real architectural undertaking, not a batch-sized addition |
| **CAP-W03** | Configurable parallel-approval quorum rule (`ANY` / `N_OF_M`), extending `CAP-A08`'s existing `ALL`/first-reject shapes | §25 | `CAP-A08` |
| **CAP-W04** | Declared `State.SLA{duration, warning_threshold}` property, composing the already-proven `CAP-A11`+`CAP-E02`/`E03`+escalation chain (Case 7) into one declaration instead of 3–4 hand-assembled primitives | §28 | `CAP-A11`, `CAP-E02`, `CAP-E03`, `CAP-V17` (❌, countdown badge) |
| **CAP-W05** | Process map — a read-only rendered view of one Machine's own Status values + guarded Events as a state/transition diagram, purely presentational, no new metadata concept (addresses legibility, §4.3, without requiring a first-class Workflow object) | §9, §41 (partial) | `CAP-E06` (state guard) + `CAP-V02`-family server-rendered views; same "no new mechanism, compose existing data" posture as `CAP-V15`/`CAP-V17` |

Deliberately **not** proposed: a first-class `Workflow`/`Transition` object replacing the
emergent model wholesale. Section 5.2's findings are a real, structural reason to keep — Case 7's
CMMN-style flexibility and Case 10/12's cross-Machine collaboration both depend on nothing being
locked into a closed Workflow envelope. `CAP-W01`–`CAP-W05` are proposed as **additive layers on
top of** existing primitives, matching the standing "escalate the existing mechanism" pattern
(`CAP-F19`'s own Tier 1/2/3 framing, and `CAP-V14 Tier 2`'s own precedent) rather than a
replacement architecture.

**Admission test:** per `capability-lifecycle.md`'s own rule, a benchmark alone is not enough —
admission needs a real case declaring the need. None of `CAP-W01`–`CAP-W05` has portfolio-case
evidence yet (the closest, Evidence cardinality under `CAP-W01`, is implied but not explicitly
declared by Case 7's `[NOT YET]` annotations). Recorded as **Proposed, HOLD for case evidence** —
matching `CAP-V11`'s own standing HOLD posture — not admitted outright.

---

# 7. Appendix — comparator BRD, verbatim

> Preserved exactly as supplied (2026-08-22), for future study independent of this session.
> Original Indonesian headers kept as-is.

## BUSINESS REQUIREMENTS DOCUMENT (BRD)

# Metadata-Based Workflow Orchestration Application

**Versi:** 1.0
**Status:** Draft
**Tanggal:** 22 Agustus 2026

---

## 1. EXECUTIVE SUMMARY

Aplikasi ini merupakan **Metadata-Based Workflow Orchestration Platform**, yaitu platform aplikasi yang memungkinkan organisasi mendefinisikan dan menjalankan proses kerja secara fleksibel berdasarkan metadata, tanpa harus membuat business logic khusus untuk setiap jenis proses.

Platform tidak diposisikan sebagai task management biasa.

Task management konvensional umumnya menggunakan pola:

```text
TO DO → DOING → DONE
```

Sedangkan platform ini menggunakan model:

```text
WORKFLOW
   ↓
STATE
   ↓
REQUIREMENT
   ↓
TRANSITION
   ↓
NEXT STATE
```

Setiap state dapat mempunyai berbagai requirement yang harus dipenuhi, misalnya:

* Form
* Entity
* Task
* Approval
* Evidence
* Document
* Decision
* Signature
* Measurement

Workflow kemudian menentukan bagaimana proses berpindah dari satu state ke state berikutnya berdasarkan actor, condition, requirement, dan transition.

Prinsip utama platform:

> **Configure the process, don't code the process.**

Kode menyediakan **engine**, sedangkan metadata mendefinisikan **proses dan perilaku aplikasi**.

---

# 2. LATAR BELAKANG

Dalam aplikasi bisnis konvensional, workflow sering kali dibuat langsung dalam source code.

Contoh:

```text
if status == "review" {
    showApproveButton()
}
```

Pendekatan tersebut menyebabkan perubahan proses bisnis membutuhkan perubahan source code.

Padahal proses organisasi sering berubah.

Contoh proses:

```text
Pengajuan
→ Approval
→ Penugasan
→ Pengerjaan
→ Submit Hasil
→ Review
→ Revisi
→ Review
→ Verifikasi
→ Selesai
```

Proses lain dapat memiliki pola berbeda:

```text
Proposal
→ Approval Manager
→ Approval Finance
→ Execution
→ Evidence
→ Verification
→ Reporting
→ Closed
```

Atau:

```text
Request
→ Assessment
→
   ├── Simple → Execute
   │
   └── Complex → Approval → Execute
                       ↓
                    Review
```

Aplikasi harus mampu menangani variasi tersebut tanpa membuat workflow baru secara hard-code.

---

# 3. VISI PRODUK

Membangun platform metadata-driven yang memungkinkan organisasi membuat berbagai aplikasi dan proses kerja dengan mengkonfigurasi:

```text
ENTITY
FORM
PERMISSION
WORKFLOW
REQUIREMENT
RULE
TASK
EVIDENCE
AUTOMATION
```

tanpa perlu mengubah core application engine.

Platform diharapkan dapat menjadi **application platform**, bukan sekadar task management application.

---

# 4. POSITIONING

Platform berada di antara beberapa kategori teknologi:

```text
                    PROCESS ORCHESTRATION
                           ↑
                           │
                       Camunda
                           │
                           │
             Workflow Orchestration
                           │
                           │
                    PLATFORM INI
                           │
                           │
                  Frappe / ERPNext
                           │
                           │
                      Drupal
                           │
                           ↓
                    APPLICATION PLATFORM
```

Secara konseptual:

* Drupal kuat pada metadata/configuration-driven application.
* Frappe kuat pada metadata-driven data model dan application generation.
* Camunda kuat pada process orchestration.
* Platform ini menggabungkan pendekatan tersebut dengan **workflow sebagai native part dari metadata application**.

---

# 5. TUJUAN

## 5.1 Tujuan Utama

Menyediakan workflow orchestration engine yang dapat mengatur lifecycle pekerjaan, data, task, approval, evidence, review, dan keputusan berdasarkan metadata.

## 5.2 Tujuan Khusus

Platform harus mampu:

1. Membuat workflow tanpa perubahan source code.
2. Membuat state secara konfiguratif.
3. Membuat transition secara konfiguratif.
4. Menentukan actor pada setiap aktivitas.
5. Menentukan requirement pada setiap state.
6. Menghubungkan workflow dengan form.
7. Menghubungkan workflow dengan entity.
8. Membuat task berdasarkan workflow.
9. Meminta approval.
10. Meminta evidence.
11. Meminta dokumen.
12. Meminta decision.
13. Melakukan conditional branching.
14. Melakukan parallel approval.
15. Melakukan revision loop.
16. Melakukan escalation.
17. Mengelola SLA.
18. Menjalankan automation.
19. Mencatat audit trail.
20. Mendukung workflow versioning.
21. Menjalankan banyak workflow dengan engine yang sama.

---

# 6. PRINSIP ARSITEKTUR

## 6.1 Metadata First

Metadata merupakan sumber definisi aplikasi.

```text
Metadata
   ↓
Runtime
   ↓
Application Behavior
```

## 6.2 Configure, Don't Code

Business process tidak boleh bergantung pada hard-coded workflow logic.

Source code menyediakan:

```text
ENGINE
```

Metadata menyediakan:

```text
PROCESS
```

## 6.3 Entity-Centric

Entity merupakan objek bisnis.

Contoh:

```text
Corrective Action
Purchase Request
Maintenance Request
Audit Finding
Project
Inspection
Program
```

Workflow mengatur lifecycle dan hubungan entity tersebut.

## 6.4 Workflow as Orchestration Layer

Workflow bukan sekadar field `status`.

Workflow mengorkestrasi:

```text
Entity
Form
Task
Approval
Evidence
Document
Decision
Actor
Notification
Automation
```

## 6.5 Requirement-Centric

State tidak hanya mempunyai action.

State mempunyai **requirement**.

Pertanyaan utama workflow:

> Apa yang harus tersedia atau selesai sebelum proses dapat berpindah?

---

# 7. KONSEP UTAMA

## 7.1 Entity

Entity adalah objek bisnis yang dikelola aplikasi.

Contoh:

```text
Corrective Action
Maintenance Request
Purchase Request
Inspection
Audit Finding
```

Entity mempunyai metadata:

```text
Entity
├── Fields
├── Form
├── Permission
├── Validation
├── Relationship
└── Workflow
```

---

# 8. FORM

Form merupakan mekanisme untuk mengumpulkan data dari user.

Contoh:

```text
Inspection Form
├── inspection_date
├── inspector
├── result
├── finding
└── notes
```

Form dapat digunakan sebagai requirement dalam workflow.

---

# 9. WORKFLOW

Workflow merupakan definisi proses.

Contoh:

```text
Corrective Action Workflow
```

Workflow terdiri atas:

```text
Workflow
├── Version
├── States
├── Transitions
├── Actors
├── Requirements
├── Conditions
├── Actions
└── SLA
```

---

# 10. WORKFLOW DEFINITION VS WORKFLOW INSTANCE

Harus dibedakan antara definisi workflow dan pelaksanaan workflow.

## Workflow Definition

Template proses:

```text
Corrective Action v2
```

## Workflow Instance

Pelaksanaan nyata:

```text
Corrective Action #CA-2026-00125
```

Contoh:

```text
Workflow Definition
        ↓
Corrective Action v2
        ↓
Workflow Instance
        ↓
CA-2026-00125
```

Satu workflow definition dapat mempunyai ribuan workflow instance.

---

# 11. WORKFLOW STATE

State menggambarkan kondisi proses pada suatu waktu.

Contoh:

```text
OPEN
ASSIGNED
IN_PROGRESS
SUBMITTED
REVIEW
REVISION
APPROVED
VERIFIED
CLOSED
```

State dapat mempunyai:

* Actor
* Requirement
* Condition
* Transition
* SLA
* Entry rule
* Exit rule

State bukan sekadar nilai status pada entity.

---

# 12. WORKFLOW TRANSITION

Transition adalah perpindahan antar-state.

Contoh:

```text
IN_PROGRESS
    ↓ SUBMIT
REVIEW
```

Transition mempunyai:

```text
Transition
├── Name
├── Source State
├── Target State
├── Actor
├── Condition
├── Requirement
└── Action
```

User tidak boleh mengubah state secara langsung.

User harus melakukan transition yang valid.

---

# 13. REQUIREMENT ENGINE

Requirement merupakan konsep inti platform.

Requirement mendefinisikan apa yang harus dipenuhi pada suatu state atau transition.

Jenis requirement minimal:

```text
FORM
ENTITY
TASK
APPROVAL
EVIDENCE
DOCUMENT
DECISION
```

Dapat dikembangkan menjadi:

```text
SIGNATURE
PAYMENT
QUESTIONNAIRE
MEASUREMENT
INTEGRATION
```

---

# 14. STATE REQUIREMENT

Contoh:

```text
STATE: SUBMIT_RESULT

Required Requirements:

1. Work Result Form
2. Completion Photo
3. Completion Report
```

Engine harus memvalidasi requirement sebelum transition dapat dilakukan.

```text
SUBMIT_RESULT
      ↓
Requirement Validation
      ↓
 ┌────┴────┐
 NO        YES
 ↓          ↓
BLOCK    TRANSITION
```

---

# 15. FORM REQUIREMENT

State dapat meminta user mengisi form.

Contoh:

```text
STATE: REVIEW

Requirement:
Review Form

Fields:
- decision
- score
- comment
- recommendation
```

Submission form menjadi bagian dari workflow instance.

---

# 16. ENTITY REQUIREMENT

Workflow dapat meminta atau menghasilkan entity lain.

Contoh:

```text
STATE: IMPLEMENTATION

Requirement:
Corrective Action Result
```

Atau workflow dapat membuat entity baru:

```text
APPROVAL
    ↓
CREATE ENTITY
    ↓
Purchase Order
```

---

# 17. TASK REQUIREMENT

Workflow dapat menghasilkan task.

Contoh:

```text
STATE: EXECUTION

Requirement:
Maintenance Task
```

Task dapat mempunyai:

```text
Task
├── Assignee
├── Due Date
├── Priority
├── Description
├── Checklist
├── Result
└── Evidence
```

---

# 18. EVIDENCE REQUIREMENT

Workflow dapat meminta evidence.

Jenis:

```text
PHOTO
DOCUMENT
VIDEO
ATTACHMENT
URL
MEASUREMENT
SIGNATURE
```

Contoh:

```text
STATE: VERIFICATION

Required:
- Minimum 2 photos
- Inspection report
- Verification form
```

---

# 19. APPROVAL REQUIREMENT

Approval merupakan requirement khusus untuk mendapatkan keputusan dari actor tertentu.

Contoh:

```text
STATE: APPROVAL

Requirements:
- Manager Approval
- Finance Approval
```

Decision:

```text
APPROVE
REJECT
REVISION
```

---

# 20. DECISION

Decision dapat menjadi hasil dari:

* Form
* Approval
* Rule
* Condition
* User decision

Decision dapat menentukan transition berikutnya.

Contoh:

```text
Assessment Result

IF risk = HIGH
    → HIGH_RISK_APPROVAL

IF risk = LOW
    → STANDARD_EXECUTION
```

---

# 21. ACTOR

Actor adalah pihak yang melakukan aktivitas.

Contoh:

```text
REQUESTER
ASSIGNEE
REVIEWER
APPROVER
VERIFIER
MANAGER
OWNER
```

Actor tidak harus selalu berupa role.

Actor dapat berupa:

```text
Specific User
Role
Group
Manager
Entity Owner
Requester
Previous Actor
Dynamic Resolver
```

Contoh:

```text
Reviewer = Branch Manager dari entity tersebut
```

---

# 22. ACTOR RESOLUTION

Engine harus mampu menentukan user yang sebenarnya berdasarkan metadata.

Contoh:

```text
Actor:
REVIEWER

Resolver:
entity.branch.manager
```

Engine:

```text
Workflow
 ↓
Actor Definition
 ↓
Actor Resolver
 ↓
Actual User
```

---

# 23. CONDITION ENGINE

Condition menentukan apakah transition tersedia atau cabang mana yang digunakan.

Contoh:

```text
amount > 50.000.000
```

maka:

```text
→ GM APPROVAL
```

Jika:

```text
amount <= 50.000.000
```

maka:

```text
→ MANAGER APPROVAL
```

Condition harus dapat menggunakan metadata entity, form submission, workflow context, actor, dan hasil requirement.

---

# 24. REVISION LOOP

Workflow harus mendukung revisi.

Contoh:

```text
WORKING
   ↓
SUBMIT
   ↓
REVIEW
   │
   ├── APPROVE → VERIFIED
   │
   └── REVISION
          ↓
       WORKING
          ↓
       SUBMIT
          ↓
       REVIEW
```

Setiap revision harus menyimpan:

* reviewer
* timestamp
* reason
* comment
* previous submission
* revision count

---

# 25. PARALLEL WORKFLOW

Workflow harus dapat menjalankan beberapa requirement secara paralel.

Contoh:

```text
APPROVAL
   │
   ├── Finance
   ├── Manager
   └── GA
         │
         ↓
   ALL APPROVED
         ↓
      EXECUTE
```

Rule:

```text
ALL
ANY
N_OF_M
```

---

# 26. ACTION ENGINE

Action bukan inti workflow.

Action merupakan automation yang dapat dijalankan oleh workflow.

Contoh:

```text
CREATE_TASK
CREATE_ENTITY
UPDATE_ENTITY
SEND_NOTIFICATION
ASSIGN_ACTOR
START_TIMER
STOP_TIMER
WRITE_AUDIT
TRIGGER_WORKFLOW
```

Action dapat dipicu:

```text
On State Entry
On State Exit
On Transition
On Requirement Completed
On Event
On SLA Breach
```

---

# 27. EVENT

Workflow dapat merespons event.

Contoh:

```text
Entity Created
Form Submitted
Payment Received
Task Completed
Approval Completed
SLA Breached
Document Uploaded
```

Event dapat memicu:

```text
Condition
Transition
Action
Notification
```

---

# 28. SLA DAN TIMER

State dapat memiliki SLA.

Contoh:

```text
REVIEW
SLA = 2 Working Days
```

Engine dapat menghasilkan:

```text
SLA Warning
SLA Breach
Escalation
```

Contoh:

```text
REVIEW
 ↓
2 days
 ↓
SLA BREACH
 ↓
Notify Manager
 ↓
Escalate Reviewer
```

---

# 29. NOTIFICATION

Notification dapat dipicu oleh:

* assignment
* state change
* transition
* approval request
* revision
* SLA warning
* SLA breach
* completion

Channel:

```text
IN_APP
EMAIL
PUSH
WHATSAPP
```

---

# 30. WORKFLOW VERSIONING

Workflow harus mempunyai versioning.

Contoh:

```text
Corrective Action v1
Corrective Action v2
Corrective Action v3
```

Workflow instance yang sudah berjalan tidak boleh rusak ketika workflow definition berubah.

Instance harus tetap mengacu pada workflow version yang digunakan saat instance dibuat.

---

# 31. WORKFLOW LIFECYCLE

Workflow definition:

```text
DRAFT
   ↓
PUBLISHED
   ↓
ACTIVE
   ↓
DEPRECATED
   ↓
ARCHIVED
```

Workflow yang sudah digunakan tidak boleh diedit secara destruktif.

Perubahan signifikan menghasilkan version baru.

---

# 32. WORKFLOW RUNTIME

Workflow Runtime bertanggung jawab menjalankan workflow.

Komponen:

```text
Workflow Resolver
State Manager
Transition Engine
Requirement Engine
Actor Resolver
Condition Engine
Action Engine
Event Engine
SLA / Timer Engine
Notification Engine
Audit Engine
```

---

# 33. WORKFLOW RUNTIME FLOW

```text
User / Event
     ↓
Workflow Instance
     ↓
Current State
     ↓
Requirement Engine
     ↓
Actor Resolver
     ↓
Permission Check
     ↓
Condition Engine
     ↓
Transition
     ↓
New State
     ↓
Actions / Events
     ↓
Audit
```

---

# 34. METADATA ARCHITECTURE

Core metadata:

```text
ENTITY
FORM
FIELD
PERMISSION
WORKFLOW
WORKFLOW VERSION
STATE
TRANSITION
ACTOR
REQUIREMENT
CONDITION
ACTION
SLA
EVENT
```

Runtime data:

```text
WORKFLOW INSTANCE
INSTANCE STATE
INSTANCE REQUIREMENT
TRANSITION LOG
TASK
EVENT
AUDIT
```

Business data:

```text
ENTITY
FORM SUBMISSION
DOCUMENT
EVIDENCE
APPROVAL
```

---

# 35. KONSEP GENERIC REQUIREMENT

Requirement harus bersifat extensible.

Struktur konseptual:

```text
Requirement
├── Type
├── Target
├── Required
├── Cardinality
├── Actor
├── Condition
└── Validation
```

Contoh:

```text
Type       = FORM
Target     = inspection_form
Required   = TRUE
Actor      = inspector
```

Contoh:

```text
Type        = EVIDENCE
Target      = PHOTO
Required    = TRUE
Cardinality = 2..*
```

Contoh:

```text
Type        = ENTITY
Target      = corrective_action
Required    = TRUE
Cardinality = 1..*
```

Konsep ini memungkinkan requirement type berkembang tanpa mengubah core workflow model.

---

# 36. APPLICATION METADATA STACK

Platform dapat dibangun dengan beberapa lapisan metadata:

```text
APPLICATION
     ↓
ENTITY METADATA
     ↓
FORM METADATA
     ↓
PERMISSION METADATA
     ↓
WORKFLOW METADATA
     ↓
REQUIREMENT METADATA
     ↓
RULE METADATA
     ↓
AUTOMATION METADATA
     ↓
RUNTIME
```

Hasil akhirnya:

```text
METADATA
   ↓
RUNTIME
   ↓
APPLICATION BEHAVIOR
```

---

# 37. RELATIONSHIP ENTITY DAN WORKFLOW

Workflow tidak menjadi pemilik business data.

Workflow menjadi orchestration layer.

```text
ENTITY
   │
   ▼
WORKFLOW INSTANCE
   │
   ├── FORM
   ├── TASK
   ├── APPROVAL
   ├── EVIDENCE
   ├── DOCUMENT
   └── DECISION
```

Dengan demikian workflow dapat mengorkestrasi banyak entity.

---

# 38. CONTOH END-TO-END

## Corrective Action

```text
OPEN
  ↓
ASSIGNED
  ↓
IN_PROGRESS
  ↓
SUBMIT_RESULT
```

Requirement:

```text
- Completion Form
- Completion Photo
- Completion Report
```

Kemudian:

```text
REVIEW
```

Requirement:

```text
- Review Form
- Reviewer Decision
```

Transition:

```text
APPROVE
    ↓
VERIFICATION

REVISION
    ↓
IN_PROGRESS
```

Verification:

```text
Requirement:
- Verification Form
- Verification Evidence
```

Jika berhasil:

```text
VERIFIED
   ↓
CLOSED
```

Tidak ada workflow logic khusus untuk `Corrective Action` di source code.

Workflow tersebut merupakan metadata.

---

# 39. CONTOH WORKFLOW LAIN

## Purchase Request

```text
DRAFT
 ↓
SUBMITTED
 ↓
ASSESSMENT
 ↓
CONDITION
 ├── <= 50M → MANAGER APPROVAL
 │
 └── > 50M → GM APPROVAL
                ↓
             FINANCE
                ↓
             PROCUREMENT
                ↓
             DELIVERY
                ↓
           VERIFICATION
                ↓
             CLOSED
```

Workflow engine yang sama dapat digunakan.

---

# 40. CONTOH AUDIT WORKFLOW

```text
AUDIT
 ↓
FINDING
 ↓
CORRECTIVE ACTION
 ↓
IMPLEMENTATION
 ↓
EVIDENCE
 ↓
VERIFICATION
 ├── PASS → CLOSED
 └── FAIL → CORRECTIVE ACTION
```

Satu workflow dapat mengorkestrasi:

```text
Audit Entity
Finding Entity
Corrective Action Entity
Task
Evidence
Verification Form
```

---

# 41. WORKFLOW BUILDER

Administrator harus dapat membuat workflow melalui UI.

Minimal:

```text
Workflow
    ↓
States
    ↓
Transitions
    ↓
Actors
    ↓
Requirements
    ↓
Conditions
    ↓
Actions
    ↓
Publish
```

Tahap awal tidak harus menggunakan visual drag-and-drop.

Form-based builder sudah mencukupi.

Visual workflow builder dapat dikembangkan kemudian.

---

# 42. WORKFLOW VALIDATOR

Sebelum workflow dipublish, engine harus melakukan validasi.

Validasi minimal:

* Start state tersedia.
* End state tersedia.
* Semua transition memiliki source.
* Semua transition memiliki target.
* Tidak ada state tanpa jalan keluar kecuali end state.
* Actor valid.
* Requirement valid.
* Form target valid.
* Entity target valid.
* Condition valid.
* Tidak ada transition yang tidak mungkin dicapai.
* Tidak ada dependency metadata yang hilang.

---

# 43. AUDIT TRAIL

Semua perubahan runtime harus dicatat.

Contoh:

```text
08:00
User: Budi
Transition: SUBMIT
IN_PROGRESS → REVIEW

10:30
User: Andi
Transition: REQUEST_REVISION
REVIEW → REVISION

14:00
User: Budi
Transition: RESUBMIT
REVISION → REVIEW

16:00
User: Andi
Transition: APPROVE
REVIEW → VERIFIED
```

Audit trail minimal:

```text
Actor
Transition
Previous State
New State
Timestamp
Comment
Related Data
```

---

# 44. SECURITY

Security harus berbasis:

```text
USER
 ↓
ROLE / PERMISSION
 ↓
ACTOR
 ↓
WORKFLOW
 ↓
TRANSITION
```

User tidak boleh:

```text
UPDATE state = "approved"
```

secara langsung.

User hanya boleh:

```text
EXECUTE transition("approve")
```

Engine kemudian memvalidasi:

1. Actor.
2. Permission.
3. Requirement.
4. Condition.
5. Transition.

---

# 45. RELIABILITY

Transition harus atomic.

Contoh:

```text
Submit Review
```

harus memastikan:

```text
State berubah
+
Requirement tercatat
+
Audit tercatat
```

sebagai satu transaksi yang konsisten.

Jika action gagal, engine harus memiliki mekanisme retry/idempotency untuk mencegah duplicate operation.

---

# 46. PERFORMANCE

Platform harus:

* melakukan caching metadata yang sering digunakan;
* memisahkan synchronous dan asynchronous action;
* meminimalkan query metadata berulang;
* mendukung database transaction untuk transition;
* mendukung queue untuk proses panjang;
* menjaga workflow engine tetap stateless sejauh memungkinkan.

---

# 47. EXTENSIBILITY

Requirement type harus dapat diperluas.

Awal:

```text
FORM
ENTITY
TASK
APPROVAL
EVIDENCE
DOCUMENT
DECISION
```

Kemudian:

```text
SIGNATURE
PAYMENT
INTEGRATION
QUESTIONNAIRE
MEASUREMENT
API_CALL
AI_REVIEW
```

Engine tidak perlu diubah secara fundamental.

---

# 48. NON-GOALS

Tahap awal tidak bertujuan menjadi:

* BPMN engine penuh.
* ERP.
* Project management application.
* RPA platform.
* AI agent orchestration platform.
* Enterprise ESB.
* Distributed microservice orchestration engine.

Fokus utama adalah:

> **Metadata-driven business workflow orchestration.**

---

# 49. PERBANDINGAN KONSEPTUAL DENGAN PLATFORM LAIN

## Drupal

Kuat pada:

```text
Entity
Field
Bundle
Form
View
Configuration
```

Pendekatan:

> Metadata-driven application/content platform.

## Frappe / ERPNext

Kuat pada:

```text
DocType
Field
Form
Permission
Workflow
Report
```

Pendekatan:

> Metadata-driven business application platform.

## Camunda

Kuat pada:

```text
Process
Task
Event
Gateway
Human Task
Service Task
```

Pendekatan:

> Process orchestration platform.

## Platform ini

Menggabungkan:

```text
Metadata Application Platform
+
Workflow Orchestration
+
Requirement Engine
```

dengan prinsip:

> **Workflow menjadi native citizen dari metadata application.**

---

# 50. DIFERENSIASI KONSEPTUAL

Pembeda utama platform bukan sekadar memiliki workflow.

Pembeda adalah:

```text
STATE
  ↓
REQUIREMENTS
  ↓
TRANSITION
```

Requirement menjadi generic contract antara proses dan data/aktivitas.

Contoh:

```text
STATE: VERIFICATION

Required:
✓ Verification Form
✓ 2 Photos
✓ Inspection Entity
✓ Reviewer Decision
```

Workflow dapat mengorkestrasi semua hal tersebut tanpa mengetahui domain spesifiknya.

---

# 51. BUSINESS VALUE

Platform memberikan manfaat:

### Fleksibilitas

Perubahan proses tidak selalu membutuhkan perubahan kode.

### Reusability

Satu workflow engine dapat digunakan untuk banyak proses.

### Consistency

Semua proses menggunakan mekanisme approval, audit, permission, dan transition yang sama.

### Traceability

Setiap perpindahan proses dapat ditelusuri.

### Governance

Organisasi dapat mendefinisikan siapa yang boleh melakukan apa.

### Scalability

Workflow baru dapat ditambahkan sebagai metadata.

### Faster Application Development

Pengembangan aplikasi dapat bergeser dari:

```text
CODE → DEPLOY → TEST → CHANGE
```

menjadi:

```text
DEFINE METADATA → VALIDATE → PUBLISH
```

---

# 52. IMPLEMENTATION ROADMAP

## Phase 1 — Core Workflow Engine

Implementasi:

```text
Workflow
Workflow Version
State
Transition
Actor
Workflow Instance
Audit
```

## Phase 2 — Requirement Engine

Implementasi:

```text
Form
Entity
Task
Evidence
Approval
Document
Decision
```

## Phase 3 — Rule & Orchestration

Implementasi:

```text
Condition
Branching
Parallel Approval
Action
Event
Notification
```

## Phase 4 — Operational Workflow

Implementasi:

```text
SLA
Timer
Escalation
Delegation
Reminder
```

## Phase 5 — Workflow Designer

Implementasi:

```text
Visual Builder
Workflow Validator
Version Comparison
Simulation
Test Mode
```

---

# 53. SUCCESS CRITERIA

Platform dianggap berhasil apabila:

1. Administrator dapat membuat workflow tanpa mengubah source code.
2. Administrator dapat membuat state.
3. Administrator dapat membuat transition.
4. Administrator dapat menentukan actor.
5. Administrator dapat menambahkan form ke state.
6. Administrator dapat menambahkan entity ke workflow.
7. Administrator dapat menentukan evidence requirement.
8. Administrator dapat membuat task requirement.
9. Administrator dapat membuat approval requirement.
10. Engine dapat memblok transition jika requirement belum terpenuhi.
11. Engine dapat melakukan conditional branching.
12. Engine dapat melakukan revision loop.
13. Engine dapat melakukan parallel approval.
14. Engine dapat menjalankan automation.
15. Engine dapat mengelola SLA.
16. Seluruh transition tercatat dalam audit trail.
17. Workflow dapat dibuat dalam beberapa versi.
18. Instance lama tetap berjalan menggunakan workflow version sebelumnya.
19. Workflow engine dapat digunakan oleh berbagai domain aplikasi.
20. Penambahan workflow baru tidak membutuhkan perubahan source code core engine.

---

# 54. CORE DATA MODEL — KONSEPTUAL

## Metadata

```text
workflow
workflow_version
workflow_state
workflow_transition
workflow_actor
workflow_requirement
workflow_condition
workflow_action
workflow_event
workflow_sla
```

## Runtime

```text
workflow_instance
workflow_instance_state
workflow_instance_requirement
workflow_transition_log
workflow_task
workflow_event_log
workflow_audit
```

## Application

```text
entity
entity_field
form
form_submission
document
evidence
approval
```

---

# 55. CORE RUNTIME MODEL

Konsep runtime:

```text
EVENT
  ↓
WORKFLOW INSTANCE
  ↓
CURRENT STATE
  ↓
REQUIREMENT ENGINE
  ↓
ACTOR RESOLVER
  ↓
PERMISSION CHECK
  ↓
CONDITION ENGINE
  ↓
TRANSITION
  ↓
NEW STATE
  ↓
ACTIONS
  ↓
EVENTS
  ↓
AUDIT
```

---

# 56. ARSITEKTUR KONSEPTUAL

```text
┌───────────────────────────────────────────────┐
│                  APPLICATION                 │
├───────────────────────────────────────────────┤
│ Entity Metadata                               │
│ Form Metadata                                 │
│ Permission Metadata                           │
│ Workflow Metadata                             │
│ Requirement Metadata                          │
│ Rule Metadata                                 │
└───────────────────────┬───────────────────────┘
                        │
                        ▼
┌───────────────────────────────────────────────┐
│          WORKFLOW ORCHESTRATION RUNTIME       │
├───────────────────────────────────────────────┤
│ Workflow Resolver                             │
│ State Manager                                 │
│ Transition Engine                             │
│ Requirement Engine                            │
│ Actor Resolver                                │
│ Condition Engine                              │
│ Action Engine                                 │
│ Event Engine                                  │
│ SLA / Timer Engine                            │
│ Notification Engine                           │
│ Audit Engine                                  │
└───────────────────────┬───────────────────────┘
                        │
                        ▼
┌───────────────────────────────────────────────┐
│                     DATA                     │
├───────────────────────────────────────────────┤
│ Business Entities                             │
│ Forms                                         │
│ Tasks                                         │
│ Approvals                                     │
│ Evidence                                      │
│ Documents                                     │
│ Workflow Instances                            │
│ Audit Logs                                    │
└───────────────────────────────────────────────┘
```

---

# 57. PRINCIPLE UTAMA PRODUK

Platform harus selalu menjaga prinsip berikut:

> **The application is defined by metadata; the process is orchestrated by workflow; the runtime interprets both.**

Atau secara sederhana:

```text
METADATA
   +
RUNTIME
   =
APPLICATION
```

Dengan workflow:

```text
ENTITY
   +
FORM
   +
REQUIREMENT
   +
WORKFLOW
   +
RULE
   +
TASK
   +
EVIDENCE
   =
BUSINESS PROCESS
```

---

# 58. KESIMPULAN

Platform ini bukan sekadar:

> **Task Management Application**

dan bukan pula sekadar:

> **Workflow Engine**

Platform ini dirancang sebagai:

> **Metadata-Based Application Platform dengan Native Workflow Orchestration.**

Fondasi arsitekturnya adalah:

```text
ENTITY
FORM
PERMISSION
WORKFLOW
STATE
REQUIREMENT
TRANSITION
ACTOR
CONDITION
ACTION
EVENT
TASK
EVIDENCE
AUDIT
```

Dengan model tersebut, organisasi dapat mendefinisikan berbagai proses tanpa mengubah core application.

Konsep paling fundamental yang membedakan platform ini adalah:

```text
STATE
  ↓
WHAT MUST BE FULFILLED?
  ↓
REQUIREMENTS
  ↓
CAN THE PROCESS MOVE?
  ↓
TRANSITION
  ↓
NEXT STATE
```

Sehingga workflow tidak menjadi sekadar kumpulan tombol atau status, tetapi menjadi **mesin yang mengorkestrasi data, pekerjaan, manusia, keputusan, dan bukti dalam sebuah proses bisnis yang dapat dikonfigurasi**.
