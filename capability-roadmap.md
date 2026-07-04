# Capability Roadmap

> How Menata Runtime discovers, structures, and closes capability gaps —
> so that the runtime can eventually realize the full range of business process possibilities.
>
> Status: Active | Created: 2026-07-04

---

# Problem

The prototype phase has validated the metadata-driven foundation (Cases 1–2) and surfaced its first real gaps (Case 3 — Document Approval, gaps P1–P6).

Gaps will keep appearing. That is expected — we are prototyping.

The question this document answers:

> What is the best pattern to discover and structure capability findings,
> so the runtime provably converges toward completeness —
> instead of chasing whatever the last case happened to reveal?

---

# Method: Dual-Track Discovery

Two discovery methods are already in use in this repository, informally:

| Track | Where it already happened | Formal name |
|-------|--------------------------|-------------|
| Scoring 16 features across 7 platform prototypes | `prototype/README.md` | Conformance benchmarking |
| Case 3 boundary test with `[NOT YET]` annotations → P1–P6 | `prototype/go/docs/examples/` | Case-driven gap discovery |

Each track alone is insufficient:

- **Case-driven alone** is biased — it only finds gaps in cases someone thought to write.
- **Benchmark alone** is theoretical — it doesn't prove the runtime actually works.

World-class practice combines both: an **external pattern catalog** as the map, **cases** as the terrain-truth, and a **capability registry** as the single source of record connecting them.

---

# External Benchmarks (the map)

## Workflow Patterns Initiative — primary yardstick

The canonical academic benchmark for workflow capability completeness
(van der Aalst & ter Hofstede, [workflowpatterns.com](http://www.workflowpatterns.com)).
Used to evaluate BPEL, BPMN, YAWL, jBPM, Staffware, and others.

- **43 Control-Flow Patterns** — sequence, parallel split, synchronization, exclusive choice, multi-merge, multi-instance, cancellation, …
- **40 Data Patterns** — data visibility, data interaction, data-based routing, environment data
- **43 Resource Patterns** — role-based allocation, direct allocation, separation of duties, escalation, delegation

Evidence this maps directly to our gaps — Case 3 findings translated:

| Case 3 gap | Workflow Pattern |
|-----------|------------------|
| Sequential step activation (P3) | WCP-1 Sequence |
| Parallel approval mode | WCP-2 Parallel Split |
| All-approved → parent approved (P3) | WCP-3 Synchronization |
| Any-rejected → reject document | WCP-9 Discriminator + cancellation |
| Only assigned approver may act (P5) | Resource: Direct Allocation |
| `now` / `current_user` values (P2) | Data: Environment Data |

Had the pattern mapping existed first, P1–P6 would have been predicted before Case 3 was written.

## OMG standards — category map

| Standard | Covers | Relevance |
|----------|--------|-----------|
| **BPMN 2.0** | Structured processes | Conformance classes (Descriptive → Analytic → Executable) model how to version runtime capability levels |
| **CMMN** | Unstructured case management | Entire category untouched by Cases 1–3 (investigations, complaints, ad-hoc work) |
| **DMN** | Decision tables | Where the constraint engine is heading — Camunda scored highest on constraints in our benchmark precisely because of DMN |

## Industrial platform catalogs — empirical completeness

What business applications actually need, distilled from 20+ years of platform evolution:

- Salesforce Metadata API types
- Frappe DocType feature set
- Odoo module domains
- This repo's own `prototype/*/docs/` mappings (Drupal, Frappe, Directus, Budibase, Salesforce, Camunda) — raw material already collected

## TCK model — proof discipline

From the Java Technology Compatibility Kit: **a capability exists only if an executable test proves it**. The curl-based verification used for Case 2 (constraint rejection, 403 permission, status transition) is the embryo of a conformance suite.

---

# Artifacts (the structure)

```text
runtime/
├── capability-roadmap.md            ← this document (method + work plan)
├── capability-registry.md           ← ARTIFACT 1 — single source of record
├── benchmarks/
│   ├── workflow-patterns-mapping.md ← ARTIFACT 2 — map vs external catalog
│   └── platform-capability-survey.md    (consolidated from 6 prototypes)
└── case-portfolio.md                ← ARTIFACT 3 — deliberate case selection
```

## Artifact 1 — Capability Registry

One row per capability. The single source of record.

| Column | Meaning |
|--------|---------|
| ID | Stable identifier (`CAP-…`), never reused |
| Area | Field / Event / Action / Constraint / Permission / View / Cross-cutting |
| Status | ✅ supported / ⚠️ partial / ❌ not yet |
| Discovered by | Which case or benchmark surfaced it |
| Pattern ref | Workflow Pattern / BPMN / CMMN reference where applicable |
| Priority | P1… ordering for implementation |
| Proof | Conformance test that verifies it (once implemented) |

Rules:

- All `[NOT YET]` annotations in example YAML files migrate here.
- The registry only grows (ratchet). A ✅ capability must never regress — its conformance test guards it.

## Artifact 2 — Pattern Benchmark Mapping

Menata Grammar mapped against the external catalogs:

- ~20 basic Control-Flow Patterns
- core Data Patterns
- core Resource Patterns

Each pattern marked: **covered / planned / out-of-scope-by-design**.
Out-of-scope requires a stated reason — silence is not a decision.

## Artifact 3 — Case Portfolio

Cases are chosen deliberately to hit untested pattern clusters — not at random.

| # | Case | Pattern cluster exercised | Status |
|---|------|--------------------------|--------|
| 1 | Design Request | CRUD + simple state machine | ✅ done |
| 2 | Leave Request | Same, different domain (portability proof) | ✅ done |
| 3 | Document Approval | Sequence, synchronization, resource allocation | ⚠️ documented, gaps P1–P6 |
| 4 | Recurring Reminder / Scheduling | **Time-driven events** — `Every Day` exists in the spec but has never been exercised | planned |
| 5 | Inventory / Stock | Calculation, quantity, multi-record transaction | planned |
| 6 | Petty Cash / Ledger | Balance, numeric aggregation, immutability | planned |
| 7 | Customer Complaint | CMMN-style unstructured case, escalation, SLA | planned |
| 8 | Payment Integration | External events, webhook | planned |

---

# Process Loop

```text
Pick next case from portfolio (targeting untested patterns)
        │
        ▼
Write .menata + .yaml with [SUPPORTED]/[NOT YET]/[PARTIAL] annotations
        │
        ▼
Extract gaps → register in Capability Registry (ID, pattern ref, priority)
        │
        ▼
Prioritize → implement runtime extension
        │
        ▼
Conformance test (seed + executable verification)
        │
        ▼
Update Registry + Pattern Mapping → repeat
```

---

# Work Plan

## Study 1 — Pattern Mapping ⏳ next

Map Menata Grammar against Workflow Patterns subset.

**Deliverables:**
- [ ] `benchmarks/workflow-patterns-mapping.md` (Artifact 2)
- [ ] `capability-registry.md` initial content (Artifact 1) — seeded from:
  - the 16 features of the platform benchmark,
  - Case 3 gaps P1–P6,
  - patterns revealed by the mapping itself

## Study 2 — Cross-Platform Capability Survey

Consolidate what the 6 platform prototypes already documented: which capabilities do **all** platforms provide that Menata does not yet name?

**Deliverables:**
- [ ] `benchmarks/platform-capability-survey.md`
- [ ] New registry entries from survey findings

## Study 3 — Case Portfolio Design

Formalize the 8-case portfolio; define target patterns per case before writing it.

**Deliverables:**
- [ ] `case-portfolio.md` (Artifact 3)
- [ ] Case 4 (Scheduling) written as the first portfolio-driven case

## Study 4 — Conformance Harness

Formalize executable verification so ✅ capabilities cannot silently regress.

**Deliverables:**
- [ ] Conformance test design (curl-based or Go test)
- [ ] Tests retrofitted for Cases 1–2 supported capabilities
- [ ] Registry `Proof` column populated

---

# Current Gap Snapshot

Known gaps at time of writing (detail in `prototype/go/docs/examples/README.md`, Case 3):

| Priority | Gap | Blocks |
|----------|-----|--------|
| P1 | Reference field type | All cross-machine features |
| P2 | Dynamic values (`now`, `today`, `current_user`) | Timestamp/user stamping |
| P3 | `activate_next` + `aggregate_status` actions | Sequential + rollup workflows |
| P4 | Machine-level config | Approval mode switching |
| P5 | Record-level permissions | Assigned-approver enforcement |
| P6 | Internal event triggering | System-fired events |

These migrate into the Capability Registry as its first entries (Study 1).

---

# Principles

- **The map before the territory** — benchmark catalogs predict gaps before cases find them.
- **Cases prove, benchmarks guide** — a capability is real only when a case exercises it and a test verifies it.
- **One source of record** — the registry, not scattered annotations.
- **Ratchet, never regress** — supported capabilities are guarded by conformance tests.
- **Silence is not a decision** — out-of-scope patterns need a stated reason.
