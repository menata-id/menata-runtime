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

## Study 1 — Pattern Mapping ✅ done (2026-07-04)

Map Menata Grammar against Workflow Patterns subset.

**Deliverables:**
- [x] `benchmarks/workflow-patterns-mapping.md` (Artifact 2) — 20 control-flow + 7 data + 8 resource patterns + 4 event sources, assessed on 3 layers (Language / Metadata / Runtime)
- [x] `capability-registry.md` initial content (Artifact 1) — 44 capabilities registered, seeded from:
  - the 16 features of the platform benchmark,
  - Case 3 gaps P1–P6,
  - patterns revealed by the mapping itself

**Headline findings:**
- **CAP-E06 (WCP-18 Milestone)** — events are filtered by role only, never by state: an Approved document can still be Rejected. Found by the benchmark, not by any case — validates the dual-track method.
- **CAP-C09** — constraints run only on Create, never on event trigger.
- **CAP-R02** — no record edit form exists (CRUD's U missing).
- The Language layer is nearly complete (almost all ✅) — gaps concentrate in Metadata schema and Runtime, confirming the language design is ahead of the runtime as intended.

## Study 2 — Cross-Platform Capability Survey ✅ done (2026-07-04)

Consolidate what the 6 platform prototypes already documented: which capabilities do **all** platforms provide that Menata does not yet name?

**Deliverables:**
- [x] `benchmarks/platform-capability-survey.md` — consolidated 23-capability matrix across Salesforce/Frappe/Drupal/Camunda/Directus/Budibase vs Menata Go
- [x] New registry entries from survey findings — 9 new capabilities (registry v0.2): CRUD-level permissions, field-level visibility, list search/filter, data import/export, auto REST API, metadata package portability, computed fields, field defaults, notification delivery channels

**Headline findings:**
- 9 "table stakes" capabilities are universal across platforms but were unnamed in Menata.
- State machine enforcement is *the* differentiator — the two lowest-scoring platforms (Directus, Budibase) lost points precisely for lacking it; independently confirms CAP-E06's Prio 2.
- Frappe's DocType is the closest architectural model — the gap list ≈ distance between Menata Machine and DocType.
- DMN is the proven growth path for the constraint engine.

## Study 3 — Case Portfolio Design ✅ done (2026-07-04)

Formalize the 8-case portfolio; define target patterns per case before writing it.

**Deliverables:**
- [x] `case-portfolio.md` (Artifact 3) — 8 cases with declared targets per case (Cases 5–8 targets pre-declared: inventory, ledger, complaint, payment webhook)
- [x] Case 4 (Maintenance Reminder) written as the first portfolio-driven case — `prototype/go/docs/examples/maintenance-reminder.{menata,yaml}`

**Headline findings:**
- Declare-targets-first works: Case 4 confirmed all 4 declared gaps (E02, A09, A02, A04) **and** surfaced 2 untargeted findings — CAP-A11 (date arithmetic in actions) and CAP-V09 (declarative view-level filter).
- Registry now at 55 capabilities (v0.2 + Case 4 additions).

## Study 4 — Conformance Harness ✅ done (2026-07-04)

Formalize executable verification so ✅ capabilities cannot silently regress.

**Deliverables:**
- [x] Conformance test design — HTTP black-box, curl-based: `prototype/go/conformance/README.md`
- [x] Tests retrofitted for Cases 1–2 supported capabilities — `prototype/go/conformance/run.sh`, 13 tests (T00–T12), all passing against the live prototype
- [x] Registry `Proof` column populated — all 16 ✅ capabilities now reference their conformance test IDs

**Notes:**
- Run: `./conformance/run.sh` (local) or `BASE_URL=… ./conformance/run.sh` (any deployment). Exit non-zero = regression.
- When CAP-E06 (state guards) lands, add negative-transition tests (see caveat in conformance README).

---

# Phase 1 complete

All four studies of the initial work plan are done (2026-07-04). Ongoing operation follows the **Process Loop** above. Two work streams run from here:

- **Implementation:** registry Prio 1–2 (CAP-F13 reference fields, CAP-E06 state guards, CAP-C09 constraints-on-events), then the case portfolio (Cases 5–8).
- **Phase 2 studies** below — composite and scale benchmarks.

---

# Phase 2 — Composite & Scale Benchmarks (planned 2026-07-04)

Phase 1 benchmarked **single-machine** capability. Phase 2 asks the questions that only appear when capabilities compose: across domains, across applications, across workspaces — and closes with governance for capability growth itself.

## Study 5 — Portal GA Cross-Domain Benchmark ✅ done (2026-07-04)

Benchmark against a real production system: Portal GA v3 (35 domains, DDD/CQRS, Go+Templ+HTMX — the same stack family as the prototype). Three angles: input patterns, cross-domain integration (ADR-0012 A/B/C, PICA→AAR, consumer contracts), cross-domain information display.

**Deliverables:**
- [x] `benchmarks/portal-ga-cross-domain-survey.md` — three-angle inventory mapped to Menata concepts
- [x] New registry entries — 9 capabilities (registry v0.3): new **Cross-Machine Integration** area (CAP-I01…I05) + CAP-X09 organizational scoping + CAP-V10/V11/V12 (composed dashboard, channel-independent rendering, wizard forms)
- [x] Position statement — **no new Language Grammar needed** (cross-machine events fit the existing Event grammar's "sources"); Runtime Metadata needs a new **Integration** section (subscriptions, contracts, event schemas); dispatcher error-isolation semantics are constitutional runtime requirements

**Headline findings:**
- ADR-0012 Patterns A and B compose from already-registered capabilities; **Pattern C (domain events) is the genuinely new metadata concept** — CAP-I01.
- Portal GA already maintains integration knowledge as YAML documents (event catalog ~65 events, 22 contracts, context map 47 entries) kept true by humans + CI fitness functions — the strongest evidence yet that a metadata-driven runtime can make these documents *the executable system itself*.
- **CAP-X09 organizational scoping** — Menata records have no org-unit/period dimension at all; Portal GA's RULE #11/#12 show this dimension pervades records, queries, permissions, and timezones.
- PICA-style canonical shared machines → composition-governance input for Study 7; constitutional stack (fitness functions, ARB, living registries) → reference implementation input for Study 9.

## Study 6 — Accounting Vertical Benchmark (Odoo / ERPNext) ⏳ next

Deep vertical benchmark: accounting, tax, financial reporting, data visualization — against Odoo Accounting and ERPNext (Frappe) accounting modules.

Scope of survey: chart of accounts, journal entries, **double-entry invariants** (debit = credit — a constraint class Menata has never seen), tax rules and computation, fiscal periods and closing (immutability-after-state), financial reports (trial balance, balance sheet, P&L), drill-down visualization.

**Key question:** where is the boundary between metadata-expressible accounting and what needs a domain engine? Odoo/ERPNext answer this with code; how much can Menata answer with metadata?

**Deliverables:**
- [ ] `benchmarks/accounting-vertical-survey.md` — Odoo/ERPNext capability inventory vs Menata registry
- [ ] Case 9 (Accounting) target declaration in `case-portfolio.md`
- [ ] New registry entries (multi-record invariant constraints, computed aggregation, report/visualization views, period-close immutability)

## Study 7 — Organization-Wide Composite Integration

Compose **all prior cases as one organization**: general domains (Cases 1–8) + specific domains (Portal GA patterns from Study 5, accounting from Study 6) — shared users and roles across applications, cross-application references, org-wide dashboards and notifications.

**Key question:** are new capabilities needed that **no single case reveals**? Hypothesis: yes — capabilities that only emerge at composition: shared identity, shared master data (employee, department, customer used by many applications), cross-application navigation, global search, cross-domain reporting.

**Deliverables:**
- [ ] Case 10 (Organization Composite) — written as a portfolio case with declared targets
- [ ] Emergent-capability findings registered (flagged `[COMPOSITION FINDING]`)
- [ ] Assessment: does Workspace/Application hierarchy suffice, or is a shared-kernel concept needed?

## Study 8 — Multi-Workspace Scale & Performance Architecture

If all capabilities are used and workspaces multiply: what data structure and programming strategy delivers the best performance with optimal resources?

Topics: tenancy models (shared schema + workspace_id vs schema-per-workspace vs database-per-workspace), `records` JSONB indexing strategy (GIN, expression indexes per hot field), per-workspace metadata caching and reload, connection pooling, workspace isolation guarantees (closes CAP-X06), noisy-neighbor mitigation, horizontal scaling of the interpreter.

**Key question:** what breaks first at 100 workspaces × 50 machines × 1M records — and which architecture defers that point longest with the least resources?

**Deliverables:**
- [ ] `benchmarks/scale-architecture-study.md` — tenancy option analysis + recommendation
- [ ] Load-test plan (synthetic workspace/machine/record generator against the prototype)
- [ ] ADR: tenancy and indexing decision for the next runtime iteration

## Study 9 — Capability Lifecycle Governance (closing)

The meta-study: what happens when a **new capability** is proposed?

1. **Admission test — is it worthy?** Evidence from ≥2 independent sources (a case *and* a benchmark); universality check against the platform survey; single-responsibility fit within Grammar; cannot be composed from existing capabilities; business language exists for it (if domain experts can't say it, it doesn't belong in the language).
2. **Completeness design — is it whole?** A capability definition-of-done spanning every layer: Language expression → Metadata schema → Loader → Application Model → Engine (executor/constraint/permission) → UI → Conformance test → Docs → Registry row. Each layer either implemented or explicitly deferred with reason.
3. **Architecture pattern — how does the runtime grow?** Extension points per engine (small-core lesson from VS Code in `architecture-benchmark.md`), versioned metadata schema, backward-compatibility rule (old metadata must keep working), feature flags for incubating capabilities.

**Deliverables:**
- [ ] `capability-lifecycle.md` — governance document: proposal template, admission criteria, definition-of-done checklist, extension architecture pattern
- [ ] Retrofit check: apply the admission test retroactively to 2–3 registered capabilities as calibration

## Sequencing

```text
Study 5 (Portal GA) ──┐
                      ├──► Study 7 (Composite) ──► Study 8 (Scale) ──► Study 9 (Governance)
Study 6 (Accounting) ─┘
```

Studies 5 and 6 are independent and can run in either order. Study 7 composes their findings. Study 8 stresses the composed picture. Study 9 closes the loop by governing everything the previous studies taught us about how capabilities are born.

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
