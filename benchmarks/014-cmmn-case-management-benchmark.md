# CMMN vs. the Bounded Model — Is Unstructured Case Management a Real Gap?

> Study 22 of the Capability Roadmap.
>
> Case 7 (Customer Complaint, `case-portfolio.md`) already named CMMN as its external grounding
> and recorded a one-line boundary finding: Menata's flat `When X` + CAP-E06 expresses CMMN's
> *bounded* flexibility (many predefined paths, no fixed Sequence) but not its *unbounded*
> flexibility (a case worker inventing a new task type at runtime) — stated as a design
> boundary, not a gap, and never re-examined since. This study gives that one line the same
> treatment Study 19/20 gave the comparator Workflow BRD: a full construct-by-construct mapping,
> a scan across all 21 portfolio cases (not just the one that named it), and a formal admission
> test — rather than letting a single un-reexamined sentence stand as the final word on whether
> Menata's bounded model is actually sufficient for real unstructured work.
>
> Method note: this is a map-only study (external standard vs. registry + case portfolio), the
> same genre as Study 0/1/19 — no new code, no new conformance tests. Any capability it surfaces
> follows the standing admission discipline (`capability-lifecycle.md`) before anything gets
> built. | Created: 2026-08-22

---

# 1. The question

> Does any case in the 21-case portfolio need something CMMN (Case Management Model and
> Notation, OMG standard) offers that Menata's Event + Constraint + Permission model — including
> the Process Overlay (`brd-menata-runtime-v2.md`, Studies 20–21) — does not already express?

This is deliberately narrower than Study 19/20's question. Those studies compared a *rival
architecture* (first-class Workflow engine) against Menata's emergent model. CMMN is not a rival
runtime architecture here — no comparator ships a CMMN engine, no case demands one. It is a
**vocabulary check**: CMMN is the OMG's own answer to "how do you formally describe work that
doesn't have a fixed sequence," so it is the right yardstick for the one honest question Case 7
raised and never closed.

---

# 2. CMMN primer (the constructs that matter here)

| Construct | What it means |
|---|---|
| **Case Plan Model** | The container: every Stage, Task, and Milestone a case *might* involve, declared up front — CMMN is not schema-less at the model level; what's flexible is *instantiation order*, not *vocabulary* |
| **Stage** | A named phase grouping Plan Items; may nest |
| **Task** (Human / Process / Case / Decision) | A unit of work — same idea as a Menata Event/Action, typed by what performs it |
| **Milestone** | A named achievement with no work of its own — reached when its Sentry fires, regardless of which Task path got there |
| **Sentry** (`ifPart` + `onPart`) | A gate: `onPart` names one or more predecessor Plan Items whose completion arms it; `ifPart` is a boolean condition; both may combine (AND/OR) before the gated item becomes available |
| **Discretionary Item** | A Plan Item marked *available but not required* — a case worker may add it to the running case or not; it is still declared in the model, only its *instantiation* is optional |
| **Case File Item (CFI)** | Loosely-structured case content (documents, data) attached to the running case, not owned by any one Task's own data model |
| **Case Role** | Who may plan/perform discretionary items — CMMN's access-control unit |

The plain-language summary that matters for this study: **CMMN's flexibility is in *when* and
*whether* a pre-declared item runs, never in inventing an item the model never named.** The
model itself (Case Plan Model) is closed and authored ahead of time, same as any Menata Machine.

---

# 3. Construct-by-construct mapping

| CMMN construct | Menata equivalent | Verdict |
|---|---|---|
| Stage | `process.states` grouping, or just multiple `Status` values sharing a prefix | covered — composition, not a new mechanism |
| Task (Human) | Event + Permission (`actor.role`/`owner_field`) | covered (CAP-P01/P02) |
| Task (Process/Case) | `action` chain / `create_record` (CAP-A06) targeting a child Machine | covered |
| Task (Decision) | `computed` field (CAP-F14) or a guard `Constraint` | covered |
| Milestone | a derived/computed condition, or a `Status` value with no outbound Permission (reached, not acted on) | covered — composes CAP-F14, no new mechanism |
| Discretionary Item (repeatable, optional) | a non-state-changing Event, triggerable any number of times while its guard passes (e.g., "Add Investigation Note") | covered — nothing in Menata requires every declared Event to fire, or to fire once |
| Sentry, single predecessor | `condition: {status equals from}` (CAP-E06) | covered |
| Sentry, `onPart` over N sibling records | `requirement: {type: evidence, cardinality}` (CAP-W01) / quorum `N_OF_M` (CAP-W03) | covered — Studies 21's B3/B4 land almost exactly here |
| **Sentry, arbitrary boolean tree over *multiple, heterogeneous* case facts** (e.g. `(TaskA done OR TaskB done) AND Milestone.X reached`) | no single declarative gate — today this needs composing several independent Constraints/Events by hand, which is weaker than one Sentry expression | **gap, see §5.1** |
| Case File Item (schema-less attached content) | none — every fact belongs to a Field on some Machine, declared before load | **non-goal, see §5.2** |
| Case Role | CAP-P01 role | covered |

Nine of eleven rows compose from capabilities already built (several of them *because* of Study
21's Process Overlay work, done independently of this study). Two do not, and the two are
different in kind — one is a real, narrowly-scoped gap; the other is a boundary the architecture
should not cross.

---

# 4. Scan across the 21-case portfolio

Full case-by-case reasoning only where it is not immediately obvious; the rest get one line.

| # | Case | CMMN-shaped? | Reasoning |
|---|---|---|---|
| 1–2 | Design/Leave Request | No | Fixed short sequence, no discretionary work |
| 3 | Document Approval | No | Sequence/parallel is bounded and fully declared (steps_machine) |
| 4 | Maintenance Reminder | No | Time-driven, not case-management |
| 5–6 | Inventory / Petty Cash | No | Calculation-shaped, not process-shaped |
| **7** | **Customer Complaint** | **Yes — the originating case** | See §4.1 |
| 8 | Payment Confirmation | No | External-event idempotency, not case management |
| 9 | Accounting | No | Documented, sequential, compliance-bound — the opposite of discretionary |
| 10 | Organization Composite | No | Composition test, not process |
| 11–13 | Social/Community/Blog | No | No process content at all |
| 14 | Lending | No | Fixed sequential disbursement/repayment |
| 15–16 | E-commerce / POS | No | Cart→Checkout is sequential, not discretionary |
| **17** | **Helpdesk** | **Yes — re-proves Case 7** | Same shape, same domain-portability relationship Case 2 has to Case 1 — not an independent data point (see §6) |
| 18 | HR Operations | No | Master data only |
| 19 | Project Management | Superficially, but no | Free card/list reordering (CAP-V14) is *presentation* ordering, not task-completion gating — a Trello card has no Sentry, it just sits wherever a human drags it |
| 20 | Hospital | No | Scheduling + compliance, sequential per visit |
| 21 | E-learning | No | Deliberately the *opposite* of CMMN — sequential unlock is maximal bounding |

## 4.1 Case 7 re-examined against the full CMMN vocabulary

Re-reading Case 7's declaration (`case-portfolio.md` §Case 7) against every row in §3, not just
the one line it originally wrote:

- **Discretionary, repeatable investigation steps** — Case 7 doesn't actually name a step that
  needs inventing at runtime; it names a *fixed* small set of permitted Events (Triage,
  Investigate-note, Delegate, Escalate, Resolve, Reopen) available in any order once Status
  allows them. That is Stage/Sentry flexibility (order-free among *known* items), not Case File
  Item flexibility (unknown items). The original finding conflated the two.
- **Priority-driven SLA + auto-escalation** — already CAP-A11/E02/E05, now CAP-W04 (Study 21 B4,
  shipped independently of this study, same day).
- **Reopen after resolution** — WCP-10, already ✅, a bounded loop, not case-worker invention.

Nothing in Case 7's actual declared targets needs an item the model never named. The original
"boundary finding" was correct in spirit (Menata is bounded, CMMN's *marketing* is unbounded) but
imprecise in claim — it implied Case 7 itself sits on that boundary. It doesn't. Case 7 is fully
inside CMMN's own *bounded* half (Stage + Sentry + Discretionary-but-declared), which this study
now shows Menata already covers end to end.

---

# 5. What's left — named honestly, not smoothed over

## 5.1 Compound Sentry — a real, narrow gap

CMMN allows one Sentry to gate on an arbitrary boolean combination of multiple, heterogeneous
predecessor facts — e.g., "Stage Review may start once (Photo Evidence submitted OR Manager
Override granted) AND SLA Clock has not breached." Today, expressing this in Menata means
composing several independent Constraints/guards by hand and hoping their interaction is correct
— there is no single declarative expression for "OR across two different Requirement/Milestone
facts, ANDed with a third."

This is real, but **evidence-thin**: no case in the portfolio — including Case 7 — actually needs
a compound boolean Sentry; §4.1 shows Case 7's own real requirements are single-predecessor
throughout. Applying the admission test (`capability-lifecycle.md` §2) directly:

| # | Criterion | Result |
|---|-----------|--------|
| A1 | Dual evidence | **Fails** — one source only (this study, reading the CMMN spec itself); no case names it |
| A2 | Universality/verticality | Plausible (compound approval conditions are common in the wild) but undeclared here |
| A3 | Single Grammar area | Would span Event condition + Constraint — likely two capabilities, not one |
| A4 | Non-composability | Unclear until a real case forces the specific shape needed |
| A5 | Business language | "start Review once we have a photo or a manager override, and the clock hasn't run out" — yes, a domain expert can say this |

**Verdict: stays Proposed, HOLD** — same disposition `capability-lifecycle.md` §6 gave CAP-V11
("evidence-thin... needs a second independent source before implementation"). Recorded here so a
future case that needs it restarts from a named row, not from zero.

## 5.2 Case File Item — a deliberate non-goal, not a future gap

CMMN's Case File Item lets a case attach arbitrary, loosely-typed content whose shape was never
declared in the Case Plan Model. This is not a missing feature Menata should eventually build —
it is the literal negation of **Metadata First** and **Machine First** (`001-design-principles.md`
§1, §3): "Runtime Metadata should avoid ambiguity," and every fact belongs to a Field declared on
a Machine before the record exists. Admitting schema-less case content would mean Menata stops
being able to say what a given piece of data *is* at load time — the same category of thing
`brd-menata-runtime-v2.md` §11 already refuses ("v2 will not become a full BPMN engine / ERP /
RPA"). This sits in that same Non-Goals list, not in the registry as a future ⚠️.

The practical need CFI addresses in real CMMN systems (attach a supporting document to an
in-flight case without pre-modeling its exact type) is already served, inside the bounded model,
by CAP-F16 (line items) or CAP-F06 (file field) on a purpose-declared child Machine — the
difference is that Menata requires naming the attachment's *kind* (a Photo, a Report, a Note) at
metadata-authoring time, even if its *content* is free-form. That naming requirement is a feature,
not friction: it is what makes CAP-W01's evidence cardinality ("2 photos, verified") possible at
all. A truly typeless attachment could never be counted, gated, or validated — CMMN itself pays
for this flexibility by having no equivalent of a compiled cardinality check.

---

# 6. Why Case 7 + 17 count as one data point, not two

`case-portfolio.md` itself already states the relationship: Case 17 (Helpdesk) is "domain-
portability proof only (same relationship Case 2 had to Case 1)... No new capability expected or
found." Admission test A1 requires *independent* sources. Two cases sharing one designer, one
cluster, and one declared purpose ("prove the same shape ports to a new domain") are one source
wearing two hats — exactly the reasoning that kept CAP-V11 at Proposed in the lifecycle
document's own retrofit calibration (§6 there). This is why §5.1's gap stays HOLD rather than
being admitted on the strength of "two cases mention it."

---

# 7. Verdict

**No new capability is admitted.** The bounded model — Event + Constraint + Permission, now with
the Process Overlay's declarative `process` block — already expresses everything CMMN's *real*
(non-marketing) flexibility offers: order-free work among a known set of Tasks, optional/
discretionary repeatable steps, and multi-predecessor fan-in up to the shapes CAP-W01/W03 already
compile. Case 7's original one-line boundary finding is **correct in spirit, tightened by this
study**: the boundary is real, but no case sits on it. One narrow gap (Compound Sentry, §5.1) is
named and parked at Proposed/HOLD, evidence-thin, per the standing discipline. One CMMN feature
(Case File Item, §5.2) is a permanent non-goal, not a future gap — recorded so it is never
silently reproposed.

This closes the one open question `case-portfolio.md` left standing since Case 7 was written,
with the same rigor Study 19/20 gave the comparator BRD, at a fraction of the length — because
the answer turned out to be "already covered," not "needs a new architecture."

---

# 8. Registry impact

- `capability-registry.md` — Workflow section: add a row/note, **CAP-W08 (Proposed, evidence-
  thin)** — "Compound Sentry: boolean (AND/OR) combination of multiple predecessor Requirements/
  Milestones gating one transition" — sourced to this study only, HOLD pending a second
  independent case.
- `capability-registry.md` — Workflow section note: Case File Item explicitly recorded as
  **non-goal** (same list as "full BPMN engine / ERP / RPA"), not a registry row, so it cannot be
  silently reproposed as a future ⚠️.
- `case-portfolio.md` — Case 7's CMMN boundary-finding line gets a cross-reference to this study
  (the full re-examination lives here now, not as a single unexamined sentence).
- `README.md` — Tier 4 table gains this study as Study 22.
