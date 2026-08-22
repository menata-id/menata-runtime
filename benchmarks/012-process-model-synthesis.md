# Process Model Synthesis

> Study 20 of the Capability Roadmap.
>
> A deeper re-examination of Study 19's comparison (`011-metadata-workflow-orchestration-brd-benchmark.md`),
> run at higher reasoning effort per direct request, with three additions Study 19 did not have:
> a **quantified flexibility test** (both concepts graded against all 21 portfolio cases, plus two
> reverse tests to control for selection bias), a **server-economy analysis** (structural
> statement-count, storage-growth, and lock-behavior comparison, grounded in Study 8 and the live
> deployment), and a **synthesis** — Concept C, a new model designed to beat both parents on the
> two stated criteria: flexibility (pass the whole portfolio) and power (stay light and economical
> on a modest server).
>
> Comparator BRD text: preserved verbatim in Study 19's Appendix — not repeated here.
>
> Status: v1.1 — + Addendum (same day): concept profiles at a glance, what Concept C is and is
> not, and the "Concept A plus a presentation model" counterfactual | Created: 2026-08-22

---

# Relationship to Study 19

Study 19's 30-concept mapping and its three structural gaps (no generic Requirement contract, no
version-pinned metadata, no single legible process artifact) are accepted as input and not
re-derived. This study **revises** Study 19 in four places:

1. **A category finding Study 19 missed**: the comparator BRD has *no read-surface model at all* —
   see §1 below. This changes what the two documents can fairly be compared as.
2. **Quantification**: Study 19 said the emergent model "covers the same ground for the majority of
   cases"; §2 below replaces that qualitative claim with a graded 21-case table and a tally.
3. **A strength of the comparator Study 19 under-credited**: its §46 synchronous/asynchronous
   action split names a real weakness of this runtime (every action runs inline inside the
   request's own transaction) — registered below as CAP-W06.
4. **A revision to CAP-W02's shape**: blanket instance-pinned versioning is the wrong import;
   effective-dated change policy is cheaper and more expressive — registered below as CAP-W07,
   with CAP-W02's row amended (ratchet: amended, not deleted).

Terminology, this document only: **Concept A** = the comparator BRD's model (first-class
Workflow / State / Requirement / Transition, Workflow Instance as a runtime entity). **Concept B**
= Menata Runtime's model as specified and built (Machine = Fields + Events + Constraints +
Permissions + Views; workflow emergent; records + one transaction per request).

---

# 1. What is actually being compared — a category finding

Reading the comparator BRD end to end against `runtime-metadata-schema.md` surfaces a fact
Study 19's concept-by-concept table obscured: **the two documents do not describe the same
category of system.**

The comparator's §34 metadata inventory is: Entity, Form, Field, Permission, Workflow, Workflow
Version, State, Transition, Actor, Requirement, Condition, Action, SLA, Event. Searching its 58
sections for any read surface — a list, a detail page, a report, a dashboard, a calendar, a search,
a notification inbox — finds **none**. Its Form (§8, §15) is a data-*collection* mechanism only.
Its positioning diagram (§4) places it midway between Camunda and Frappe/Drupal, but its actual
specified content is almost entirely the Camunda half: an orchestration engine plus an
entity/form/evidence store, with the entire application surface (everything a user *looks at*
between transitions) left to an unspecified second system.

Concept B's specification covers the whole application: 19 view/page capabilities (CAP-V01–V19),
navigation (CAP-O03), search (CAP-O04), a notification center (CAP-O05), REST API (CAP-X07),
import/export (CAP-R06), public access (CAP-P07) — alongside the process primitives.

Consequence for this study's method: comparing "A vs B as application platforms" would be unfair
to A (it forfeits every view-shaped target by omission). So §2 grades each case's **dominant
cluster** (per `case-portfolio.md`'s own Rule 2) — the process/data logic the case exists to
prove — and counts the missing read surface **once**, as this structural finding, rather than
re-penalizing A in every row. Even with that concession, the tally below is not close.

A second fairness rule, stated up front: **Concept A is graded as-specified-and-fully-built** (it
is a draft BRD with no implementation; it gets the benefit of assuming everything it specifies
works). **Concept B is graded on its registry** — ● requires the dominant cluster's capabilities
to be ✅/⚠️ implemented; ◐ means partly implemented with the remainder registered and designed
within the same model; ○ means the model as specified has no story for the cluster.

---

# 2. Flexibility test — the 21-case gauntlet

Grades: ● native fit · ◐ passes with strain, composition effort, or scheduled-but-unbuilt parts ·
○ the concept as specified has no story for the case's dominant cluster.

| # | Case | Dominant cluster | A | B | Deciding facts |
|---|------|------------------|---|---|----------------|
| 1 | Design Request | CRUD + simple state machine | ◐ | ● | Both enforce the state machine. A requires a full Workflow Definition/Version/Publish apparatus for a 4-state lifecycle; B is one Machine, states inferred from a `value_list` |
| 2 | Leave Request | Domain portability | ◐ | ● | Same as Case 1 |
| 3 | Document Approval | Multi-instance approval: sequence, sync, allocation | **●** | ◐ | **A's home turf and its only outright win.** Parallel/sequential approval, per-step actors, decisions are its native vocabulary, incl. `N_OF_M` (§25). B proves the same flow but via six capabilities (F13, A07, A08, X03, P02, E05), three `Machine.Config` keys, and documented name-matching heuristics (`Sequence`/`Decision` — `prototype/go/CLAUDE.md` names these as prototype-honest heuristics) |
| 4 | Maintenance Reminder | Time-driven recurrence, date arithmetic, escalation | ○ | ● | A's timers are **state-scoped SLAs** (§28); a perpetual recurring schedule is not a state SLA, and A's action vocabulary (§26) has no field arithmetic — `Next Due Date advance by Frequency` is inexpressible. B: CAP-E02/E03/A11 all ✅ |
| 5 | Inventory / Stock | Quantity math, ledger append, negative-stock guard, UoM | ○ | ◐ | A has no computed fields, no UoM model, and its condition grammar (§23) is field-vs-literal — `Item.Stock On Hand >= Normalized Quantity` (cross-record aggregate) is inexpressible. B: F14/F16/F19/X12 ✅, C08 registered |
| 6 | Petty Cash | Aggregation, immutability, period close, SoD | ◐ | ◐ | A's period-close workflow fits; but **A has no Separation-of-Duties concept anywhere in its 58 sections** (checked — §21 actors, §44 security: absent), and the reconciliation formula (`Cash Counted + sum(Vouchers) = Imprest`) is not one of its requirement types. B: P03/R07/E06 ✅, C08/C10 registered |
| 7 | Customer Complaint | CMMN-style ad-hoc case, SLA, reopen | ◐ | ● | Split verdict, and instructive both ways: A's declared `State.SLA` (§28) is **cleaner than B's** (B composes SLA from 4 primitives: A11+E02+A09+E05); but A's closed transition graph ("user must perform a valid transition", §12) strains against no-fixed-sequence discretionary work — the exact CMMN boundary Case 7 exists to test. B proved the whole case |
| 8 | Payment Confirmation | Webhook ingestion, idempotency, reconciliation | ◐ | ● | A specifies `Payment Received` (§27) and idempotency (§45) — but its instance-centric model has no answer for the **entityless event**: a webhook arriving before any matching instance exists has no Workflow Instance to resolve to. B collapses the edge: the webhook event is just a record (E04/X13/A13/X12 all ✅) |
| 9 | Accounting | Header-detail, debit=credit invariant, period lock, trial balance | ○ | ◐ | The posting workflow alone fits A; the case's actual substance — the double-entry aggregate invariant, the temporal period lock, the Trial Balance report — is all outside A's model. B: F16/F18/R07/P03/V13 ✅, C10/C11 registered |
| 10 | Organization Composite | Shared identity, master data, nav, org-wide reporting | ○ | ● | Entirely outside A's scope: no workspace model, no identity registry, no master-data designation, no navigation, no search. B: CAP-O01–O06 all ✅ |
| 11 | Social App | Many-to-many, uniqueness, feed | ○ | ◐ | **No workflow exists in this case at all** — Follow/Like are relationship writes with composite uniqueness. A's differentiating machinery is dead weight and its entity model has no many-to-many concept. B: C12 ✅, F20 registered |
| 12 | Community + gamification | Aggregate-triggered automation | ○ | ◐ | `sum(points) >= 100 → award badge`: A's conditions don't aggregate, its events don't include accrual thresholds. B: A14 ✅ (unified end-to-end flow still unproven — `benchmarks/010`) |
| 13 | Blog / One-Page Site | Public/unauthenticated access, page composition | ○ | ● | A's security model (§44) is `USER → ROLE → ACTOR → …` — a Visitor who is the *absence* of a user does not fit; and A has no page/view model at all (§1 above). B: P07/V10 ✅ |
| 14 | Lending | Batch schedule generation, SoD, overdue | ◐ | ● | A's application→approval→disbursement workflow fits well; but `CREATE_ENTITY` creates one record — Term-Months'-worth of amortization entries from one formula (the case's declared novelty) is absent. B: A15/A13/P03/E02 ✅ |
| 15 | E-commerce | Cart as scratch state, checkout commit | ◐ | ● | Genuine credit to A: state-scoped requirements model the cart naturally (a Cart state with no requirements = free editing; Checkout's requirements = the commit gate). B: R08/F16 ✅. Both pass; B's is built |
| 16 | Point of Sale | Composition of 5+8+15 | ○ | ● | Inherits A's Case 5 failure (stock math). B: composition verified loadable (all 50 example YAMLs load and render — roadmap 2026-07-12 bulk check) |
| 17 | Helpdesk | Case 7 re-proof | ◐ | ● | Same split as Case 7 |
| 18 | HR / Employee master | Master data, self-reference tree | ○ | ● | No workflow in the case at all; A has no master-data or referential-deactivation concept. B: F13-tree, O02 ✅ |
| 19 | Project Management | Manual ordering, kanban | ○ | ◐ | **A excludes this case by its own §48 Non-Goals** ("Project management application"). B: V14 ✅ (Tier 2 drag-and-drop registered) |
| 20 | Hospital | Resource calendar, field-level visibility | ○ | ◐ | A has no calendar/view model and no field-level permission concept (its permissions are transition-scoped). B: V07/P06/F16 ✅, V18 registered |
| 21 | E-learning | Sequential unlock, certificate generation | ◐ | ◐ | Sequential unlock = state guards, native in both. Certificate *generation*: A's `DOCUMENT` requirement asks *for* a document, never produces one. B: F21 ⚠️ (HTML output), E06 ✅, F20 registered |

## Tally

| | ● native | ◐ passes with strain / scheduled parts | ○ no story |
|---|---|---|---|
| **Concept A** (as specified, fully built) | 1 | 9 | **11** |
| **Concept B** (as specified + registry status) | 12 | 9 | **0** |

**Concept B passes 21/21; Concept A fails 11/21** — and the failures are not scattered: every ○
lands on a case whose dominant cluster is **data-shaped, not process-shaped** (calculation,
aggregation, relationships, master data, content, scheduling views). Concept A's model covers one
band of the application space — the closed, designed, human-task process — with real elegance, and
has nothing to say about the rest. Concept B covers the whole space, and pays for it with
composition effort exactly in A's band (Case 3's six-capability assembly is the measured cost).

## Controlling for selection bias — two reverse tests

The portfolio was selected by and for Concept B, so a B-favorable tally is partly baked in. Two
tests run in the opposite direction:

**Reverse test 1 — Concept A's own three worked examples vs Concept B** (Study 19 §5.3, accepted):
Corrective Action, Purchase Request, and the three-entity Audit flow are all buildable on B today
by composition — 3/3 pass, with two named strains (no evidence-cardinality check; no single
process artifact for the Audit span).

**Reverse test 2 — Concept A's own §53 success criteria (20 items) scored against Concept B:**

| §53 item | B status |
|---|---|
| 1–4 workflow/state/transition/actor without source-code change | ✅ (as metadata; no builder UI — but §41 itself accepts form-based authoring, and A has no builder either, it's a draft) |
| 5–9 form/entity/evidence/task/approval requirements | ◐ each attachable by composition; no generic declaration |
| 10 block transition on unmet requirement | ◐ data-shape guards only (E06/C09); no child-completion check |
| 11 conditional branching | ✅ A09/C04 |
| 12 revision loop | ✅ WCP-10 proven (Case 7 Reopen) |
| 13 parallel approval | ◐ ALL/first-reject only, no N_OF_M |
| 14 automation | ✅ A01–A15 |
| 15 SLA | ◐ composed, proven in Case 7; not declared |
| 16 audit trail on every transition | ✅ R04 (append-only, DB-enforced) |
| 17–18 versioning + version-pinned instances | ❌ |
| 19 many domains, one engine | ✅ 21 cases, 48 machines, one binary |
| 20 new workflow without engine change | ✅ metadata-only, proven repeatedly |

**Score: ~10 ✅ + 5 ◐ + 2 ❌ (items 17–18) of A's own 20 criteria.** Concept B, built, already
satisfies half of Concept A's definition of success outright and partially satisfies most of the
rest; the clean misses are versioning (both items) and the generic-requirement family. Concept A,
graded against B's portfolio, fails half the cases outright. The asymmetry survives the bias
control: **B is the broader concept; A's genuine advantages are concentrated and enumerable**
(requirement contract, declared SLA, quorum, versioning, process legibility) — which is exactly
what makes them importable (§6).

---

# 3. Power test — server economy

The stated criterion: kinerja tetap ringan dan ekonomis di server. Neither concept has a load-test
result to cite (Study 8's harness is designed, not yet run), so this section does **structural
analysis** — statement counts, row growth, and lock behavior derivable from each concept's own
data model — plus the one piece of empirical evidence that exists: Concept B's reference
implementation runs live at `menata.app` on a single modest shared VPS (co-tenanting several other
apps), 136 conformance tests green, with the whole 21-case portfolio's metadata (48 machines)
loaded in one process.

## 3.1 The canonical write: one transition with 3 requirements

Concept B (as built — `triggerEvent`, one Postgres transaction per request, metadata fully
in-memory since boot):

```text
SELECT record                      -- 1
guard + constraint eval            -- 0 queries (in-memory model, CAP-E06/C09)
UPDATE records                     -- 1
INSERT record_events (audit)       -- 1
                                   ≈ 3 statements
with approval rollup (Case 3):     + sibling SELECT + parent UPDATE + parent audit ≈ 6
Metadata queries per request: 0
```

Concept A (minimum faithful implementation of §32/§33/§54, *granting it the same metadata
caching*):

```text
SELECT workflow_instance                                -- 1
SELECT current instance state                           -- 1 (or joined)
requirement checks — one query each, uncacheable        -- 3
  (form_submission EXISTS, evidence COUNT, approvals)      (runtime data, not metadata)
actor resolution (dynamic resolver entity.branch.manager)-- 0–2
UPDATE workflow_instance                                -- 1
INSERT workflow_instance_state (new state visit)        -- 1
UPSERT workflow_instance_requirement (new state's reqs) -- ≥1
INSERT workflow_transition_log                          -- 1
INSERT workflow_audit                                   -- 1
UPDATE entity (denormalized status, §37)                -- 1
                                                        ≈ 10–13 statements,
                                                          growing linearly with requirement count
```

The requirement engine is the structural cost: **its checks interrogate runtime data, so they can
never be cached the way metadata can** — every transition pays O(requirements) queries at the
moment of transition. §3.4 below shows this cost is avoidable, which becomes a design pillar of
Concept C.

## 3.2 Storage growth

Per completed process (say 6 state visits): Concept A writes ≥ 4 bookkeeping rows per transition
(instance_state, requirement upserts, transition_log, audit) ≈ **24+ rows/process** across four
runtime tables, *in addition to* the business entity. Concept B writes 1 audit row per transition
≈ **6 rows/process** in one append-only table, and the business record mutates in place. At
Study 8's reference scale (1M records), that is the difference between ~6M audit rows in one
partitionable table and ~24M+ rows spread over four tables that all need their own indexes,
retention, and vacuum behavior.

## 3.3 Concurrency and locks

Concept A's Workflow Instance is a **write hotspot by construction**: parallel approval (§25)
means N approvers' completions all converge as writes on one instance row — row-lock
serialization exactly at the moment the process is most concurrent. Concept B's shape avoids
this structurally: each Approval Step is its own record, written independently; the parent is
touched once, by whichever decision completes the quorum (CAP-A08's rollup). Version pinning
(§30) adds a memory dimension: the metadata cache multiplies by the number of live versions.

## 3.4 Where Concept A is structurally *better* — two honest wins

**(a) The worklist query.** "Everything pending my decision" in A is one indexed query over
`workflow_instance_requirement` (`WHERE actor = X AND status = pending`). In B it is a filtered
scan: fetch candidate records, evaluate guards per record at render time — Study 8's breakage #2
(JSONB scans) in its most user-visible form. A materializes at write time what B recomputes at
read time. *The fix does not require A's model* — a maintained flag/counter on the parent record,
stamped by the same post-commit mechanism CAP-A08/CAP-I01 already use, makes B's worklist a
single indexed query too. This "write-time fan-in, read-time O(1)" rule is a pillar of Concept C.

**(b) The synchronous/asynchronous split (§46).** In B today, *every* action — notifications,
cross-machine subscriptions, batch generation — runs inline inside the triggering request's own
transaction. A slow action chain extends request latency and holds the transaction (and its
locks) open. Concept A specifies the split and a queue for long-running work; B has neither.
This is a real, previously unregistered weakness of B, found by this analysis → **CAP-W06**.

## 3.5 Verdict on power

On the stated criterion — light and economical on a modest server — **Concept B wins clearly**:
~3–6 statements per transition vs 10–13, one bookkeeping table vs four, no per-transition
requirement-query fan-out, no version-multiplied metadata cache, no instance-row hotspot, zero
metadata queries per request, server-rendered HTML with no SPA payload. Both concepts can serve
statelessly and scale horizontally (A asks for it in §46; B's immutable in-memory interpreter
delivers it, per Study 8). Concept A's two structural advantages (indexed worklists, async split)
are both importable into B's substrate without importing A's cost model.

---

# 4. Additional criteria

Beyond the two stated criteria, four more axes matter for a metadata platform and separate the
concepts cleanly:

| Criterion | A | B | Notes |
|---|---|---|---|
| **Authoring cost & reuse** | ● | ◐ | A declares a requirement type or SLA once and reuses it everywhere. B re-derives: Case 3 took 6 capabilities + 3 config keys + name heuristics; B's own guides (`writing-runtime-metadata.md`) exist because the composition knowledge doesn't live in the metadata itself |
| **Process legibility & governance** | ● | ○ | A: one readable Workflow Definition, a DRAFT→PUBLISHED lifecycle, a §42 graph validator (unreachable states, dangling transitions). B: the process is enforced but scattered across Events/Constraints/Config — legible to the runtime, reconstructible by a human only by reading every Event. B's CAP-X05 validation is strong but validates *elements*, not the *graph* |
| **Evolvability under change** | ◐ | ◐ | A: version pinning protects open instances but is a blunt instrument — a compliance change *should* hit open cases, and pinning silently prevents it, while multiplying cached metadata. B: every change hits everything immediately (and needs a restart — CAP-X04 ❌). **Neither concept can express "this change applies to open cases; that one doesn't"** — a gap in both, answered by effective-dating (§6, CAP-W07) |
| **Determinism & machine/AI authorability** | ◐ | ● | B's grammar is small and orthogonal: 5 sections per Machine, load-time validated, one conformance harness. A's §34 inventory is 14 metadata object types with many cross-object invariants — its own §42 validator list is the measure of how many ways a definition can be internally broken. For generated/AI-assisted authoring (a stated Menata direction), the smaller closed grammar wins; for a human process designer, A's vocabulary matches how they think — which is an argument for A's vocabulary *as an authoring surface*, not as a runtime model |
| **Edge-case semantics** | ○ | ● | Everything in B is a record, so edges collapse: an entityless webhook, a record with no process, a process spanning records, an ad-hoc step — all ordinary. A's instance-centric model needs special handling for each (Case 8's unmatched payment; Case 7's discretionary step; Case 11's no-process-at-all) |

The last row generalizes the tally in §2: **A's model has a privileged object (the Instance) and
every case that doesn't revolve around one becomes an edge case. B's model has no privileged
object, so nothing is an edge case — and nothing is privileged, so the process is invisible.**
That sentence is the whole comparison, and it fixes what the synthesis must do: make the process
*visible and declarable* without making it *privileged at runtime*.

---

# 5. Verdict

| Criterion | Winner |
|---|---|
| Flexibility (21-case portfolio) | **B**, 21/21 vs 10/21 |
| Power (server economy) | **B**, ~3–6 vs ~10–13 statements/transition, 1 vs 4 bookkeeping tables, no hotspot, no version-multiplied cache |
| Authoring reuse | **A** |
| Process legibility & governance | **A** |
| Evolvability | neither (shared gap) |
| Machine/AI authorability | **B** |
| Edge-case semantics | **B** |
| Worklist query economics | **A** (importable) |
| Async execution | **A** (importable) |

**Overall: Concept B is the superior foundation** — it wins both stated criteria and most added
ones, and its losses are all *declaration-level* (reuse, legibility, quorum, SLA-as-one-line,
versioning policy), while A's losses are *architecture-level* (runtime cost, privileged instance,
missing application surface). Declaration-level gaps can be closed by adding a layer;
architecture-level gaps require removing one. That asymmetry dictates the synthesis.

---

# 6. Concept C — Declared Process, Emergent Execution (the Process Overlay)

## 6.1 The principle

> **The process is declared in metadata; the runtime never sees it.**
>
> A Process Overlay is an authoring-level declaration that the metadata loader **compiles,
> deterministically, into the substrate primitives that already exist** — Events, Conditions,
> Constraints, Permissions, Config, scheduler entries. No Workflow Instance table. No second
> engine. No new runtime privileged object. The runtime executes exactly what it executes today,
> at the cost profile measured in §3.1; the overlay exists for authors, validators, and readers.

This is the same move this registry has already made twice and proven: CAP-F17 (multi-currency)
and CAP-F19 (UoM) are "composition, not new mechanisms" — a declared shape that expands into
existing primitives. Concept C promotes that pattern from field-level to process-level.

## 6.2 The compile mapping

| Overlay declaration | Compiles to (existing substrate) | New engine work needed |
|---|---|---|
| `state: X` | a value of the Status `value_list` + generated per-state guards | none |
| `transition {from, to, actor, name}` | an Event + `condition(status=from)` (CAP-E06) + `set_field(status=to)` + a Permission row (CAP-P01/P02) | none |
| `requirement: form-shape fields` | Constraints re-checked at trigger time (CAP-C09) | none |
| `requirement: evidence, cardinality N..*` | a trigger-time count check against a **maintained counter on the record itself** (see 6.3) | one new `count_at_least` check — the single genuinely new engine piece in the whole overlay (CAP-W01's core) |
| `requirement: approval, quorum ALL / ANY / N_OF_M` | steps Machine + the existing CAP-X03 config keys + CAP-A08's rollup with a generalized quorum parameter | extend `aggregate_status` params (CAP-W03) |
| `requirement: task / entity` | `create_record` on transition (CAP-A06) + a satisfied-flag rollup stamped on the parent via the existing post-commit subscription site (CAP-I01's shape) | none |
| `sla: {duration, warn_at, escalate_to}` | a date-arithmetic action at state entry (CAP-A11, business-day-aware via CAP-O06) + scheduled events (CAP-E02/E03) + an escalate Event + notify (CAP-A03/A04) | none (CAP-W04 = this row) |
| `on_entry / on_exit actions` | ordinary actions on the compiled transition Events | none |
| `change policy` (see 6.4) | state-scoped guards materialized at compile time | loader logic only (CAP-W07) |
| the process map | **rendered from the declaration** — and *derivable backward* from any hand-authored Machine's own Events+guards (decompilation) | a read-only view (CAP-W05) |

Almost the entire comparator BRD's §9 Workflow object compiles away. The one genuinely new
runtime check is evidence/child cardinality — everything else is loader work over mechanisms that
are already ✅ and conformance-guarded.

## 6.3 The performance keystone: write-time fan-in, read-time O(1)

The comparator's Requirement Engine costs O(requirements) uncacheable queries per transition
(§3.1). The overlay avoids inheriting that cost by a single rule:

> **Any requirement whose truth lives on other records is maintained on the parent record at
> write time, not computed at transition time.**

When an evidence file is attached, a task completes, or an approval step is decided, the same
post-commit dispatch site that CAP-A08 and CAP-I01 already use stamps a counter/flag onto the
parent record. The transition check then reads only the record's own data — the exact O(1),
~3-statement profile B already has. The same materialized flags make **worklist views indexable**
("pending my decision" = an indexed filter on the parent's own fields), closing A's one genuine
query-economics win (§3.4a) without A's tables. This composes with Study 8's CAP-X10
(metadata-driven expression indexes): the compiler knows exactly which flags it materializes, so
it can declare their indexes too.

## 6.4 Versioning done right: effective-dated change policy (CAP-W07, revising CAP-W02)

Blanket instance-pinning (comparator §30, Study 19's CAP-W02) is the wrong import: it multiplies
the metadata cache per live version, and it is *semantically* wrong half the time — a compliance
rule change **should** reach open cases, and pinning silently prevents it. Neither concept can
express the actual business question: *which in-flight work does this change apply to?*

The overlay answers it per change, at compile time:

```yaml
change_policy: new_records            # default today's behavior: applies to everything
# or
change_policy: { applies_to: records_in_states: [Draft, Submitted] }
# or
change_policy: all_records            # the compliance case — explicit, not accidental
```

The compiler materializes the policy as ordinary state-scoped guards — no version-pinned caches,
no parallel metadata trees, one live model. Old behavior for old work is expressed *in the
current metadata*, which also keeps it auditable: the metadata itself says what applies to whom.

## 6.5 The two-way door

Because the overlay **is** the primitives after compilation:

- **Eject**: a process that grows ad-hoc (Case 7's shape) drops out of the overlay into raw
  Events at any time — nothing is lost, because nothing else ever existed at runtime. Concept A
  has no equivalent escape hatch; its closed graph is load-bearing.
- **Lift**: a stabilized hand-built Machine can be lifted *into* an overlay declaration; the
  decompiler (CAP-W05's backward direction) drafts it from the existing Events+guards.
- **Simple stays simple**: Cases 1/2/11/18-shaped Machines never need an overlay at all —
  "Infer Before Configure" (`001-design-principles.md`) is preserved, where Concept A forces
  every case through Workflow Definition ceremony.

## 6.6 What Concept C rejects, from each parent

From **A**: the Workflow Instance as a runtime entity; the four instance/bookkeeping tables; the
11-component workflow runtime; blanket version pinning; process-before-data authoring; the
privileged object that turns half the portfolio into edge cases.

From **B**: the *silence* — the fact that building Case 3 requires knowing six CAP mechanisms,
three config keys, and the heuristics documented in `prototype/go/CLAUDE.md`. The overlay is
precisely that knowledge, moved into metadata where it belongs, validated by the loader
(§42-style graph checks — unreachable states, dangling transitions — become CAP-X05-family
load-time errors on the *declaration*), and rendered back as the process map neither an auditor
nor an administrator can currently see.

And one straight adoption from A, outside the overlay: the **async action outbox** (CAP-W06) —
slow actions (notification fan-out, subscription chains, batch generation) enqueue in the same
transaction (an outbox row, atomic with the business write) and execute after commit, off the
request path. This fixes B's real inline-execution weakness (§3.4b) with the standard
transactional-outbox shape, consistent with CAP-X13's existing idempotency discipline.

## 6.7 Worked example — the comparator's own Corrective Action (§38), as an overlay

```yaml
machine:
  id: mch_corrective_action
  # fields: ... (ordinary; Status value_list is generated from process.states)
  process:                                   # the overlay — compiled, never executed
    states: [Open, Assigned, In_Progress, Submitted, Review, Verified, Closed]
    transitions:
      - { name: Assign,   from: Open,        to: Assigned,    actor: { role: Supervisor } }
      - { name: Start,    from: Assigned,    to: In_Progress, actor: { owner_field: fld_assignee } }
      - { name: Submit,   from: In_Progress, to: Submitted,   actor: { owner_field: fld_assignee },
          requirements:
            - { type: form,     fields: [fld_completion_notes] }
            - { type: evidence, target: fld_completion_photo, cardinality: "1..*" }
            - { type: evidence, target: fld_completion_report, cardinality: "1..1" } }
      - { name: Approve,  from: Review,      to: Verified,    actor: { role: Reviewer } }
      - { name: Revise,   from: Review,      to: In_Progress, actor: { role: Reviewer },
          on_transition: [ { set_field: { field: fld_revision_count, value: increment } } ] }
      - { name: Close,    from: Verified,    to: Closed,      actor: { role: Supervisor } }
    auto: [ { from: Submitted, to: Review } ]
    sla:
      - { state: Review, duration: "2 Business Days",
          on_breach: { notify: { role: Manager }, escalate: true } }
```

Every line compiles to a mechanism that is already ✅ except the two `cardinality` checks
(CAP-W01's counter) — and the compiled result runs at ~3 statements per transition, renders a
process map, validates as a graph before load, and can be ejected to raw Events the day this
process stops being this tidy. The comparator BRD's entire differentiator (§50: State →
Requirements → Transition) is preserved **as an authoring truth**, at none of its runtime cost.

---

# 7. Registry impact

- **CAP-W01–W05 reframed** (rows amended, ratchet-preserved): no longer five independent
  additive layers (Study 19's framing) but **five compile products of one mechanism** — the
  Process Overlay. Implementation order inverts accordingly: the loader-side compiler is the
  trunk; W01's counter check and W03's quorum parameter are its only engine-side leaves.
- **CAP-W02 amended**: blanket instance-pinned versioning superseded as a target by CAP-W07's
  effective-dated change policy (cheaper: one live model, no version-multiplied cache; sharper:
  the compliance case becomes expressible instead of silently prevented). Row kept per ratchet.
- **CAP-W06 registered (new)**: async action outbox — transactional-outbox execution for slow
  actions, off the request path. Discovered by §3.4b's analysis of B's inline action execution;
  comparator §46 is the pattern source. This one is *not* HOLD-gated on a workflow case: any
  existing case with notification fan-out or subscription chains (Cases 3, 10, 12) evidences the
  latency shape.
- **CAP-W07 registered (new)**: effective-dated metadata change policy, per §6.4.
- **Still HOLD**: the overlay itself (W01/W03/W04/W05 and the compiler) keeps Study 19's
  admission posture — benchmark evidence exists, portfolio-case evidence does not yet. The first
  real case that needs a declared multi-actor process (a Case 3 successor authored by someone
  *other than* this runtime's own developers) is the admission trigger; the overlay's entire
  value proposition is that author's experience.

---

# 8. Maintenance

- Re-run §2's tally whenever a ◐ capability lands (B's column can only improve; A's is frozen —
  it is a document, not a project).
- §3's statement counts are structural claims — verify them against reality when Study 8's load
  harness first runs, and replace the estimates with measurements.
- If a Process Overlay prototype is built, its Definition of Done includes proving §6.3's claim:
  a compiled transition must run at the same statement count as a hand-authored one.

---

# Addendum (2026-08-22, same day) — follow-up questions

Three questions raised on first review of this study, answered here so the answers live with the
study rather than in a conversation: (1) a side-by-side profile of all three concepts, (2) what
Concept C actually is and is not, and (3) whether Concept A would become the superior concept if
its missing presentation model were added.

## A.1 Concept profiles at a glance

**Concept A — first-class Workflow orchestration (the comparator BRD):**

| Strengths | Weaknesses |
|---|---|
| Reusable declarations: requirement types, SLA, quorum defined once, reused everywhere | No presentation model at all — no list/report/dashboard/calendar/search anywhere in its 58 sections (§1) |
| Process legibility: one readable Workflow document, DRAFT→PUBLISHED lifecycle, §42 graph validator | Thin entity model: no computed fields, aggregates, many-to-many, master data, UoM, batch generation |
| Governance/audit-friendly: an auditor reads one artifact instead of reconstructing from Events | A privileged object (the Instance) turns every non-process case into an edge case — 11/21 portfolio failures (§2) |
| Cheap worklists: "pending my decision" is one indexed query (§3.4a) | Runtime cost: ~10–13 statements/transition, O(requirements) uncacheable queries, Instance row is a lock hotspot under parallel approval (§3) |
| `N_OF_M` quorum and one-line SLA are native | Version pinning multiplies the metadata cache per live version |
| Sync/async action split specified (§46) | Closed transition graph strains ad-hoc/CMMN work (Case 7) |

**Concept B — data-first application runtime with emergent workflow (Menata Runtime as built):**

| Strengths | Weaknesses |
|---|---|
| Full application surface: views, navigation, search, API, import/export, public access | The process is invisible — scattered across Events/Constraints/Config, no single readable artifact |
| Passes 21/21 portfolio cases, 12 natively (§2); small orthogonal grammar → machine/AI-authorable, load-time validated | Per-case composition cost: Case 3 took six capabilities, three config keys, and name heuristics |
| Light: ~3–6 statements/transition, zero metadata queries per request, one audit table, proven live on one modest shared VPS | No `N_OF_M` quorum; SLA assembled from four primitives per case |
| No privileged object → no edge cases: entityless webhooks, no-process records, ad-hoc steps are all ordinary (§4) | No versioning/change policy; worklists are filtered scans; every action runs inline in the request transaction (§3.4) |

**Concept C — the Process Overlay (§6):** everything in B's column unchanged (the runtime is
*identical*, so the performance profile and the 21/21 coverage carry over by construction), plus
A's entire declaration-level column (requirement contract with cardinality, one-line SLA, quorum,
process map, graph validation, a lifecycle on the declaration), plus two things **neither parent
has**: the effective-dated change policy (§6.4 — both parents fail to express "does this change
apply to open cases?") and the two-way door (§6.5 — eject to raw Events, lift a stabilized
Machine into a declaration). Honest caveats: C is a design, not an implementation; its complexity
concentrates in the loader-side compiler, whose Definition of Done includes proving a compiled
transition costs the same as a hand-authored one (§8); and the overlay keeps its HOLD admission
posture (§7).

## A.2 What Concept C is — and is not

C exists because the directive that commissioned this study asked for it explicitly: not only a
comparison, but "a new concept for a metadata-based application that is more flexible and more
powerful." §1–§5 are the measuring instrument; §6 is the deliverable.

C **is** a combination of A and B — but not a 50:50 merger, and the reason is §5's asymmetry:
every loss of B is *declaration-level* (closable by adding a layer), every loss of A is
*architecture-level* (closable only by removing something). So the recipe is specific and
one-directional: **B's runtime architecture + A's authoring vocabulary, joined by a compiler.**
A's Workflow becomes a *language feature*, not an *engine* — the same way a high-level language
compiles to machine code: the expressiveness is gained without a second CPU. What C takes from A
is its vocabulary and two specific mechanisms (indexed worklists via materialized flags, the
async outbox); what it refuses from A is its entire runtime (§6.6).

## A.3 Counterfactual — would Concept A win if its presentation gap were fixed?

The sharpest challenge to this study's verdict: A's biggest single absence is the read surface
(§1), so grant it a full presentation model — does A then come out ahead? **No, and the reason is
quantifiable: presentation is only one of A's three gap classes.** Re-reading §2's eleven ○
failures by *cause*:

| Gap class | Cases it explains | Fixed by adding presentation? |
|---|---|---|
| Missing read surface (views/reports/calendars/pages/search) | parts of 9 (Trial Balance), 10 (nav/search), 13 (page composition), 19 (kanban view), 20 (calendar) | **Yes** |
| Thin entity/constraint model (computed fields, UoM, cross-record aggregates, many-to-many, uniqueness, master data, SoD, field-level & public permissions) | 5, 6, 11, 16, 18, and the substance of 9 (debit=credit, period lock), 13 (Visitor), 20 (Notes visibility) | No |
| Thin automation model (date arithmetic, recurring schedules, aggregate-triggered actions, batch generation) | 4, 12, 14 | No |

Adding a presentation model lifts A from 10/21 to roughly **12–14/21, mostly to ◐** — the
remaining failures are entity-, constraint-, and automation-model gaps that views do not touch.
And the structural point behind the arithmetic: to close *all* of A's gaps you must add a
presentation model **and** a rich entity model **and** a rich constraint model **and** a rich
automation model **and** a complete security model — at which point **you have rebuilt Concept B,
with A's workflow engine still sitting on top of it**. The question then inverts: given that
foundation, does the workflow engine still need to be a separate runtime with instance tables?
§3 already answered it — the runtime cost (10–13 statements, the Instance hotspot, the
requirement-query fan-out) is inherent to the instance-centric architecture, not to the missing
views, so it does not disappear when views are added. "A + presentation + rich entity model"
converges exactly onto "B + an expensive first-class engine," and §6 shows the engine's benefits
are available as a compile layer instead. The counterfactual therefore *strengthens* the verdict
rather than overturning it.

One honest boundary on that conclusion, restated from §2's tally: A is genuinely superior
*within its band*. An organization whose problem space is overwhelmingly human-task approval
processes is well served by an A-shaped engine — and that band is already served in the real
world by mature dedicated engines (Camunda, Flowable). The 21-case gauntlet measures **breadth**
across the application space; A loses on breadth because its architecture privileges one problem
shape, not because it lacks features within that shape — Case 3, its home turf, is the one case
it wins outright.
