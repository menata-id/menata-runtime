# Roadmap

> How Menata Runtime discovers, structures, and closes capability gaps —
> so that the runtime can eventually realize the full range of business process possibilities —
> and, as the effort matured, how the repository's own documentation and structure are kept to the same standard (Phase 4).
>
> Status: Active | Created: 2026-07-04 | Renamed from `capability-roadmap.md` 2026-07-05 (scope grew beyond capability discovery)

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
├── roadmap.md            ← this document (method + work plan)
├── capability-registry.md           ← ARTIFACT 1 — single source of record
├── benchmarks/
│   ├── 000-workflow-patterns-mapping.md ← ARTIFACT 2 — map vs external catalog
│   └── 001-platform-capability-survey.md    (consolidated from 6 prototypes)
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
- [x] `benchmarks/000-workflow-patterns-mapping.md` (Artifact 2) — 20 control-flow + 7 data + 8 resource patterns + 4 event sources, assessed on 3 layers (Language / Metadata / Runtime)
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
- [x] `benchmarks/001-platform-capability-survey.md` — consolidated 23-capability matrix across Salesforce/Frappe/Drupal/Camunda/Directus/Budibase vs Menata Go
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
- [x] `benchmarks/002-portal-ga-cross-domain-survey.md` — three-angle inventory mapped to Menata concepts
- [x] New registry entries — 9 capabilities (registry v0.3): new **Cross-Machine Integration** area (CAP-I01…I05) + CAP-X09 organizational scoping + CAP-V10/V11/V12 (composed dashboard, channel-independent rendering, wizard forms)
- [x] Position statement — **no new Language Grammar needed** (cross-machine events fit the existing Event grammar's "sources"); Runtime Metadata needs a new **Integration** section (subscriptions, contracts, event schemas); dispatcher error-isolation semantics are constitutional runtime requirements

**Headline findings:**
- ADR-0012 Patterns A and B compose from already-registered capabilities; **Pattern C (domain events) is the genuinely new metadata concept** — CAP-I01.
- Portal GA already maintains integration knowledge as YAML documents (event catalog ~65 events, 22 contracts, context map 47 entries) kept true by humans + CI fitness functions — the strongest evidence yet that a metadata-driven runtime can make these documents *the executable system itself*.
- **CAP-X09 organizational scoping** — Menata records have no org-unit/period dimension at all; Portal GA's RULE #11/#12 show this dimension pervades records, queries, permissions, and timezones.
- PICA-style canonical shared machines → composition-governance input for Study 7; constitutional stack (fitness functions, ARB, living registries) → reference implementation input for Study 9.

## Study 6 — Accounting Vertical Benchmark (Odoo / ERPNext) ✅ done (2026-07-04)

Deep vertical benchmark: accounting, tax, financial reporting, data visualization — against Odoo Accounting and ERPNext (Frappe) accounting modules.

**Deliverables:**
- [x] `benchmarks/003-accounting-vertical-survey.md` — Odoo/ERPNext structural inventory (12 concepts) vs Menata registry
- [x] Case 9 (Accounting) target declaration in `case-portfolio.md` — F16, C10, E06+R07, C11, F18, V13, A02; posting derivation & reconciliation deliberately out of scope
- [x] New registry entries — 7 capabilities (registry v0.4): CAP-F16 line items/child table, CAP-F17 multi-currency, CAP-F18 auto-numbering, CAP-C10 aggregate line constraint, CAP-C11 period lock, CAP-R07 immutability-after-state, CAP-V13 aggregate report view

**Headline findings:**
- **CAP-F16 (line items / header-detail documents)** is the biggest structural gap after references — together F13+F16 separate "form apps" from "document apps"; every ERP document needs both.
- Boundary answer: documents, invariants, lifecycle, and reports are metadata-expressible (ERPNext proves tax templates, COA trees, naming series, dashboard charts as pure metadata). **Posting derivation engines are not** — multi-step conditional derivation is where metadata would degenerate into a programming language. Resolution: domain engines as pluggable executor extensions beneath declarative metadata → sharpens Study 9's extension-architecture requirement.
- CAP-F18 (auto-numbering) is universal across platforms — a table-stakes capability Study 2 missed; case+benchmark dual-evidence now satisfied.

## Study 7 — Organization-Wide Composite Integration ✅ done (2026-07-04)

Compose **all prior cases as one organization**: general domains (Cases 1–8) + specific domains (Portal GA patterns from Study 5, accounting from Study 6).

**Deliverables:**
- [x] Case 10 (Organization Composite) — `prototype/go/docs/examples/organization-composite.md`: PT Maju Bersama scenario, 8 applications, one employee crossing 4 applications in one morning
- [x] Emergent-capability findings registered — 6 `[COMPOSITION FINDING]` (registry v0.5): new **Workspace Services** area CAP-O01…O06 (identity & role registry, master data designation, navigation, global search, unified notification center, business calendar)
- [x] Assessment — **the Workspace/Application/Machine hierarchy stands**; no shared-kernel *structure* needed. What composition demands is a new metadata residence: **workspace services** — concerns owned by the workspace itself, belonging to no application. This makes `runtime/004`'s "Workspace owns shared resources" clause concrete for the first time.

**Headline findings:**
- Hypothesis confirmed: 6 capabilities emerged that none of Cases 1–9 could reveal alone.
- Two were *predicted by the spec but never exercised*: Navigation (named in runtime/004 hierarchy) and Holiday (spec 001 example object → business calendar as workspace service).
- Role strings collide across applications (`Manager` in HR ≠ `Manager` in Design) — identity/role registry (CAP-O01) is the highest-priority composition gap.

## Study 8 — Multi-Workspace Scale & Performance Architecture ✅ done (2026-07-04)

**Deliverables:**
- [x] `benchmarks/004-scale-architecture-study.md` — tenancy analysis (A: shared schema + RLS chosen; C database-per-tenant as escape hatch), data structure strategy, programming strategy
- [x] Load-test plan — synthetic generator + workload mix + matrix (X10 on/off) + falsifiable pass thresholds (p95 list < 200ms @ W=100/1M rows, boot < 5s, zero cross-workspace rows under RLS probe)
- [x] ADR-003 — `prototype/go/docs/decisions/003-tenancy-and-indexing.md`

**Headline findings:**
- **What breaks first:** eager `LoadAll` at boot (5,000 machines ≈ 30k queries), then JSONB filter seq-scans, then missing workspace dimension on data.
- **`[SCALE FINDING]` CAP-X10 metadata-driven index management** — the metadata already names every hot field (view filters, sorts, references); index reconciliation à la Kubernetes makes indexing a runtime responsibility, not an ops task.
- **`[SCALE FINDING]` CAP-X11 lazy per-workspace metadata cache** — unifies ADR-002's LISTEN/NOTIFY live reload with the scale cache: one mechanism, two problems solved.
- CAP-X06 (workspace isolation) gets its implementation strategy: PostgreSQL Row-Level Security — enforced by the database, not developer discipline.

## Study 9 — Capability Lifecycle Governance (closing) ✅ done (2026-07-04)

**Deliverables:**
- [x] `capability-lifecycle.md` — lifecycle states (Proposed → Admitted → Incubating → Supported → Deprecated), 5-criteria admission test (dual evidence, universality, single responsibility, non-composability, business language), 9-layer definition-of-done, extension architecture (registries at every engine seam, versioned schema, backward compatibility, incubation flags), proposal template
- [x] Retrofit calibration on 3 capabilities — CAP-F16 PASS, CAP-A11 PASS, **CAP-V11 correctly caught** (single source, possibly composable → HOLD at Proposed, registry annotated `evidence-thin`). The test discriminates: neither vacuous nor impossible.

**Headline notes:**
- Domain engines (Study 6's posting derivation) land at the action-executor seam — pluggable beneath declarative metadata, resolving the metadata-vs-engine boundary architecturally.
- "Unknown = explicit": unrecognized types/operators are load-time reports, honoring the Language spec's conformance clause.

---

# Phase 2 complete

All five Phase 2 studies done (2026-07-04). Registry: **79 capabilities** across 10 areas, 16 supported with conformance proof. The full loop is now operational:

```text
evidence (cases + benchmarks) → admission test → registry → definition-of-done
    → implementation via extension seams → conformance ratchet → repeat
```

**Next concrete work (implementation, per registry priority):**
1. CAP-F13 reference fields (Prio 1)
2. CAP-E06 state guards + CAP-C09 constraints-on-events (Prio 2 — correctness)
3. CAP-F16 line items + CAP-A02 dynamic values (Prio 3)
4. Then Case 5–9 implementations exercise them.

---

# Phase 3 — NFR Standards (study-only)

## Study 10 — World-Class Architecture, Performance & Security per Capability Area ✅ done (2026-07-04)

Kajian-only (no implementation): NFR requirements for **all capabilities**, structured per capability area (10 areas — capabilities in one area share an NFR profile), bound to the lifecycle as Definition-of-Done gates at implementation time.

**Deliverables:**
- [x] `nfr-standards.md` — external yardsticks (OWASP ASVS, STRIDE, Google SRE SLO, ISO 25010, fitness functions); baseline runtime threat model; 5 performance budget classes (P1–P5); NFR profile per all 10 capability areas (security / performance / architecture each)
- [x] `capability-lifecycle.md` §3b amendment — 3 NFR gates (security, performance, architecture) required for Incubating → Supported; waivers must be explicit in the registry row

**Headline findings:**
- **"Metadata is code"** — the novel threat class of a metadata-driven runtime: metadata authors sit between trusted runtime developers and untrusted end users. Four consequences shape every area: metadata injection (stored XSS via field names), logic bombs (declarative mass-actions need runtime-enforced budgets), confused deputy (executors must re-check the *triggering actor's* permission, never their own), and cross-tenant reach (metadata constitutionally unable to name another workspace's objects).
- Constraints are a **security control**, not UX — they must run on every write path (create, update, events, API, import); client-side validation is advisory only.
- Current prototype defaults are inverted vs world-class: allow-by-default reads (must become deny-by-default), value_list values unchecked server-side, no output-encoding verification for metadata-sourced strings.
- `aksi.menata.id` PoC is explicitly exempt (accepted risk, recorded in the threat model).

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

# Phase 4 — Documentation & Structure Quality

A self-audit of the repository itself: `specification/000-006`, `runtime/001-006`, `design-principles.md`, `README.md`, and the folder structure across `guides/`, `specification/`, `runtime/` (including `prototype/*`). Triggered by a full read-through of every foundational document after Phase 3 closed. Two questions: **(a)** what in the existing structure/content needs updating, merging, or removing, and **(b)** what is missing to meet world-class specification/documentation standards (yardsticks: W3C/IETF spec practice, Kubernetes KEP process, semver, Diátaxis documentation framework).

## Study 11 — Repository Structure & Content Audit ✅ done (2026-07-05)

**Findings (factual issues):**
- No `LICENSE` file despite both READMEs claiming Apache 2.0.
- Filename typo: `runtime/004-runtime-metada.md` (missing `ta`), already propagated into a cross-reference.
- `runtime/README.md` does not index any Phase 1–3 artifact (roadmap, registry, lifecycle, nfr-standards, case-portfolio, benchmarks/).
- No sentence anywhere bridges **Object** (specification term) and **Machine** (runtime term) — new readers must infer the mapping themselves.

**Findings (duplication / merge candidates):**
- `runtime/003-runtime-language.md` and `004-runtime-metadata.md` overlap ~70% (machine-first, serialization-independence, scope lists nearly identical). Recommendation: trim, don't merge — 003 keeps language principles, 004 keeps artifact concerns (scope, hierarchy, versioning).
- `runtime-metadata-schema.md` lives under `prototype/go/docs/` but is the normative schema shared by all 7 prototypes — belongs at `runtime/` level.

**Findings (removal candidates):**
- `runtime/prototype/.gitkeep` (folder already populated).
- `prototype/go/web/templates/` (empty, pre-Templ leftover).

**Findings (model/registry gap):**
- `006-runtime-model.md` declares Page/Workflow/Service/API/Theme in the hierarchy; the capability registry has not yet studied most of them (only Navigation → CAP-O03 is covered). Flagged per the "silence is not a decision" principle, not silently dropped.

**World-class gaps identified:**
1. No formal grammar (EBNF) for `.menata` — prose semantics only.
2. No unified RFC/proposal process for *language* grammar (the capability side already has one in `capability-lifecycle.md` §5).
3. Inconsistent document header/changelog format across all `.md` files.
4. No documentation map (Diátaxis-style) in the root README for new readers.
5. No unified glossary bridging specification and runtime terminology.
6. No language conformance test corpus (parallel to the capability side's `conformance/run.sh`).
7. No `CONTRIBUTING.md` despite README inviting contribution.

## Study 12 — Structural Fixes ✅ done (2026-07-05)

Executed Tahap 1 (quick factual fixes) and Tahap 2 (light restructuring) from Study 11's findings.

**Deliverables:**
- [x] Added `LICENSE` (canonical Apache 2.0 text)
- [x] Renamed `004-runtime-metada.md` → `004-runtime-metadata.md`; fixed the 2 cross-references (`prototype/README.md`, `organization-composite.md`)
- [x] Removed `prototype/.gitkeep` and empty `web/templates/`
- [x] Rewrote `runtime/README.md` — added a full Documentation section (Foundational Specification, Practical Guides, Capability Discovery & Governance, Reference Implementation) without disturbing the existing narrative
- [x] Added documentation map to root `README.md` — "I want to..." table routing to the right doc
- [x] Promoted `runtime-metadata-schema.md` from `prototype/go/docs/` to `runtime/`; fixed 3 referencing docs
- [x] Trimmed `003-runtime-language.md` — removed 8 sections duplicating `001-design-principles.md` and `004`; kept only what's unique to the Language-vs-Metadata distinction, added explicit cross-references instead of restating
- [x] Added explicit Object↔Machine bridging section in spec `000` (§Object and Machine) and `runtime/006` (Machine section)
- [x] Added cross-references between `design-principles.md` and spec `000` §Language Goals (both directions)
- [x] Registered the Study 11 model/registry gap as a new "Tracked but Not Yet Studied" section in `capability-registry.md` — Page, Service, Workflow (deliberately emergent, not a gap), API-as-declared-surface, Theme

**Note:** `capability-lifecycle.md` and `roadmap.md` mentions of `runtime-metadata-schema.md` were left as bare filenames (no path) — accurate before and after the move, no fix needed.

## Study 13 — World-Class Completeness ⏳ after Study 12

Address the 7 world-class gaps identified in Study 11.

**Deliverables:**
- [ ] `specification/007-syntax.md` — formal EBNF grammar for `.menata`
- [ ] `PROCESS.md` — unified RFC/proposal process (language grammar + runtime capability, cross-referencing `capability-lifecycle.md` §5)
- [ ] Standardized header + changelog format applied across all specification/runtime documents
- [ ] Unified glossary (specification + runtime terms, with Object↔Machine mapping)
- [ ] `CONTRIBUTING.md`
- [ ] Language conformance test corpus — deferred as documented future work (not built now), noted alongside the existing capability conformance suite

## Study 14 — Internal Package Architecture ✅ done (2026-07-05)

Prompted by a direct question: for metadata-driven apps, what does world-class Go `internal/` structure look like? The current prototype layout is flat (one package per concern) — sufficient to validate Cases 1–2, but not yet shaped to carry the extension seams `capability-lifecycle.md` §4 already sketches (field type / action type / operator / event source / view type / workspace service registries) or the cross-cutting security boundaries `nfr-standards.md` names.

**Deliverables:**
- [x] `prototype/go/docs/decisions/004-internal-package-architecture.md` — target layered structure (`core/`, `engine/`, `metadata/`, `store/`, `security/`, `web/`, `platform/`), reasoning, and a **capability-triggered migration table** (no big-bang refactor)
- [x] `prototype/go/ARCHITECTURE.md` updated — new "Package Structure" section pointing to the ADR, explicit that today's flat layout is correct-for-now, not final

**Headline findings:**
- Three proven patterns combine into the target: **Ports & Adapters** (same family as Portal GA's CBA/Clean, benchmarked in Study 5), **registry-at-init seam** (Go's own `database/sql.Register` idiom — the concrete mechanism behind capability-lifecycle §4's sketched registries), and **consumer-side interfaces** (the same decoupling rule validated in Portal GA's ADR-0012).
- Migration is explicitly **capability-triggered**: `engine/fieldtype/` is created when CAP-F13 implementation begins, not before — moving code into a registry before a second implementation needs one would be premature abstraction, which this project's own `001-design-principles.md` (Infer Before Configure) already warns against.
- `security/` gives the NFR gates (`capability-lifecycle.md` §3b) a concrete home instead of scattering checks across handlers as they're added piecemeal.

---

# Phase 5 — CAP-F13 Pre-Implementation Refinement

Before starting the actual implementation of Prio 1 (CAP-F13 reference fields), one more question surfaced: several fields already modeled across Cases 1–4 (`Requester`, `Employee`, `Approver`, `Equipment`, ...) look like they should be `reference` — is there a rigorous way to decide, instead of case-by-case intuition?

## Study 15 — Field Modeling Decision Framework ✅ done, six adversarial passes (2026-07-05)

**Deliverables:**
- [x] `benchmarks/005-field-modeling-decision-framework.md` — a decision tree (identity/lifecycle test → growth test → target-type test) plus four supporting tests (growth, identity, reuse, cardinality), grounded in Codd's normalization theory, DDD Entity vs. Value Object, MDM, and the platform conventions already surveyed in Studies 2 and 6
- [x] A second, explicit axis made rigorous: closed-vs-open domain (why `value_list` is not "text/number with a dropdown widget," but a field with a validated closed domain) crossed with fixed-vs-growing (why `value_list` and `reference` differ even though both are closed)
- [x] Retrofit calibration against every field in Cases 1–10 — reproduces almost every prior ad hoc choice correctly; a second, adversarial pass corrected two initial miscategorizations, and a third pass corrected an overcorrection (see below)
- [x] A tiered resolution model for composite-with-conversion fields (money's currency, quantity's unit of measure): escalate from flat fields → child table (CAP-F16) → dedicated history-tracking Machine, only as cardinality/history actually demand it — never assume the most complex tier by default
- [x] Explicit Grammar-boundary clarification: unit/currency conversion belongs to Computed Field (CAP-F14), never inside a Constraint — a Constraint validates already-normalized values only
- [x] `guides/writing-menata.md` §"Tips memilih tipe" rewritten — plain-language version of the same tree for domain experts, pointing to the full framework
- [x] `capability-registry.md` refined — CAP-F13 gains an explicit **three**-target-flavor scope note (workspace Machine, built-in identity, built-in File/Document), CAP-F05/F06/F17 gain long-term resolution notes, CAP-F14/C10 gain scope clarifications, CAP-O02 gains two independent confirming cases (`Equipment`, `Currency`)

**Headline findings:**
- **Two distinct failure modes**, easily conflated: **Failure Mode 1 (modeling gap)** — the tree runs out of an answer because no target Machine or master-data designation exists yet (`Equipment`, and by derivation `Currency`); **Failure Mode 2 (execution gap)** — the tree resolves correctly but the runtime hasn't finished implementing that type yet (CAP-F05/F06/F07/F10/F13 — already tracked, already prioritized).
- **Second-pass correction:** a direct follow-up question ("is `money` really a pure primitive?") caught that both `money` and `file` were initially miscategorized as pure primitives. `money` pairs an amount with a Currency that has its own identity and lifecycle (exchange rates change over time) — independently confirmed by `specification/001-object.md`, which names Currency as an example Object in its own right. `file` has its own storage identity and lifecycle (versioning, replacement) — a pattern the Study 2 platform survey already contained evidence for (Frappe Attach→File DocType, Salesforce File/ContentDocument) without it having been named at the time.
- **`duration` checked and confirmed correct** despite the same composite shape (magnitude + unit) as `money` — its unit set is small, universal, and never grows, so it resolves to a `value_list`-shaped inline selector rather than `reference`. Composite structure alone does not imply reference; composite structure *plus* a growing/lifecycle-bearing component does.
- **Third-pass correction — catching an overcorrection:** a follow-up question ("does Quantity need conversion, and where does it live?") caught that naively copying `money`'s resolution onto `Quantity` (assuming it always needs a dedicated reference Machine) swings too far the other way. The corrected model is **tiered**: a fixed conversion pair is just two flat fields on the referencing Machine (no reference needed at all); only multiple unit pairs escalate to a child table (CAP-F16); only changing-factor history escalates to a dedicated Machine. This also caught that "the unit label" (e.g. `SAK`, `TON`) and "the thing needing governance" (the conversion factor) are different components — the label is usually `value_list`, only the factor behaves like an exchange rate.
- **Grammar-boundary correction:** conversion calculation (look up factor, multiply) belongs to Computed Field (CAP-F14), never embedded inside a Constraint's own logic — Constraints (CAP-C05/C07/C10) validate already-normalized values and must stay pure, matching Menata's own single-responsibility-per-Grammar design.
- `type: user` is not a permanently distinct field type — it is `reference` with a reserved built-in target, kept separate only until CAP-O01 exists. This directly shapes CAP-F13's Definition of Done: it must support all three target flavors from day one, or a breaking change is needed later.
- The calibration discipline itself was validated by surviving three successive adversarial passes — each pass caught something the previous one missed, including the second pass's own overcorrection. A framework that only gets checked once is a framework that hasn't really been checked.
- **Fourth-pass correction:** a follow-up question ("doesn't plain reference already solve Equipment — vehicle type, vehicle asset, service record, workshop entry tables?") caught that CAP-O02 had been overstated as a blocker. `Equipment` used only within one application is fully resolved by CAP-F13 alone plus an ordinary workspace Machine — no governance capability required at all. CAP-O02's real evidence is narrower but still valid: the Case 10 cross-application narrative and `Currency` (via CAP-F17). A second follow-up ("isn't Quantity's tiering itself the complexity — the field looks simple by name, but choosing Tier 2/3 adds an unusual extra data-entry step") caught that the tiers had been presented as a per-use decision instead of a one-time, separately-authored master-data concern — *declaring* `Money`/`Quantity` should stay one line, identical in effort to any other field type. A third follow-up ("is recurring time a Date feature?") confirmed CAP-E02/CAP-A11 are the right, sufficient answer, validated against the iCalendar `RRULE` (RFC 5545) precedent — recurrence belongs to Event, not Field, and does not fall into a gap between the two.
- **Fifth-pass — closing a real risk, and a full perspective audit:** a follow-up question ("if declaring stays simple but setup is separate, won't an AI or a human just forget the setup?") identified a genuine gap in the fourth-pass fix — nothing prevented a metadata author from omitting the conversion mechanism entirely. The closing correction stays strictly at the metadata/runtime layer (per explicit request, not the resulting application's end-user experience): `type: money` must carry its `currency:`/`currency_field:` companion as a **required key of the type's own schema** — the same discipline `value_list` already applies to `values:` and `reference` already applies to `target_machine:` — enforced by CAP-X05 at load time, exactly like a dangling reference is rejected today. A full audit of the document's language then confirmed (and where needed, corrected) that every section speaks from the metadata-author/runtime perspective throughout — the framework's own new "Data" layer row makes explicit that end-user data entry in the resulting application is out of scope everywhere in this study.
- **Sixth-pass — CAP-F06 image/compression scope, and a two-source correction of the enforcement model:** a practical question about image vs. non-image files (raised independently of the field-modeling tree — it's a policy question, not an identity/reference question) confirmed `file` needs no new type: whether a file is an image is a **processing policy** in `options` (`accept`, `compress`, `max_dimension`, `format`), not a different reference target — the same reasoning already applied to `rich_text` vs. `text`. A first pass placed server-side compression as merely a fallback for API callers bypassing the widget; a direct correction (drawing on first-hand knowledge of Portal GA's actual `NativeCompressedUpload*` behavior) established the real reason for server-side compression is **browser incapability** (older browsers lacking Web Worker/WebP support), not just bypass — meaning the server-side step is not optional validation but an **authoritative enforcement path**, structurally identical to "client validation is advisory, server enforces" already established for Constraints (CAP-C09). Written into `capability-registry.md` (CAP-F06 scope), `nfr-standards.md` §2.1 (Security: server never trusts client compression; Performance: client-side is the fast path, server fallback is P4/async, never inline), and `runtime-metadata-schema.md` (concrete `options` schema + dual-path contract).

---

# Principles

- **The map before the territory** — benchmark catalogs predict gaps before cases find them.
- **Cases prove, benchmarks guide** — a capability is real only when a case exercises it and a test verifies it.
- **One source of record** — the registry, not scattered annotations.
- **Ratchet, never regress** — supported capabilities are guarded by conformance tests.
- **Silence is not a decision** — out-of-scope patterns need a stated reason.
