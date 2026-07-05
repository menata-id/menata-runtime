# Portal GA Cross-Domain Benchmark

> Study 5 deliverable (`../capability-roadmap.md`).
>
> Benchmark against Portal GA v3 — a production Go application (35 domains,
> DDD/CQRS, Go + Templ + HTMX: the same stack family as the Menata prototype)
> with a ratified constitutional framework for cross-domain integration.
>
> Status: v0.1 | Created: 2026-07-04

**Sources studied (portal-ga3 repo):** `CLAUDE.md`, `ADR-0012-cross-domain-integration-pattern.md`, `domain-integration/01-CONSTITUTIONAL-BRD.md` (§9 PICA→AAR, §10 Cross-Cutting Contribution), `02-IMPLEMENTATION-GUIDE.md` (12-step), appendices: `CONTEXT-MAP-MATRIX.md` (47 integration entries), `CANONICAL-EVENT-SCHEMA.md` (~65 events), `CONSUMER-CONTRACT-REGISTRY.md` (22 contracts), plus `internal/domains/dashboard/snapshot/` (shared digest sections).

---

# Why this benchmark matters for Menata

Portal GA is what Menata aspires to *generate*: dozens of business domains, one organization, integrated. Crucially, Portal GA already expresses its integration knowledge **as documents that look exactly like metadata**:

- an event catalog in YAML (name, version, payload, consumers, deprecation),
- consumer contracts in YAML (required fields, semantic constraints, on-violation behavior),
- a context map matrix (who integrates with whom, via which pattern).

Today humans maintain these as *documentation* and enforce them with CI scripts (fitness functions). **Menata's opportunity: make this executable Runtime Metadata instead of documentation about code.**

---

# Angle 1 — Input Patterns

| Portal GA pattern | Description | Menata mapping |
|-------------------|-------------|----------------|
| CSRF-protected forms + layout selection | Every form flows token through handler → template | Runtime concern — Menata runtime should provide this invisibly (not metadata) |
| **Multi-step wizard** (HIRADC `create_step*.templ`) | One business object filled across sequential steps, per-step validation, dynamic per-location blocks | **Gap — no wizard/multi-step form view type.** New CAP-V12 |
| **Compressed photo upload** (`NativeCompressedUpload*`, `CompactPhotoUpload`) | Client-side WebP compression, camera capture on Android, non-blocking worker; two variants (structured form vs inline composer) | Enriches CAP-F06 (file field) — upload UX is a runtime realization concern, but "photo evidence" as a field flavor is worth metadata (`accept`, compression hints) |
| **BranchPeriodSelector** (CLAUDE.md RULE #11) | Branch + period (year/month) context selected once, **must flow to every query on the page**, persisted via cookies, role-dependent defaults | **Major gap — organizational scoping dimension.** Menata records have no org-unit or period context at all. New CAP-X09 |
| Pagination mandate + query-param limits | Every list page paginated; search params length-limited | CAP-R05 (already registered) + input hardening note |
| Selection components (checkbox-reveal, typeahead chips) | Rich value-list input variants | Runtime rendering realization of CAP-F03 — no new metadata needed |
| HTMX form reset via server reload | Reload form fragment from server after submit rather than client-side reset | Runtime realization pattern — validates the prototype's server-driven approach |

**Angle 1 verdict:** two real capability gaps (wizard views, organizational scoping); the rest are runtime realization quality, not metadata.

---

# Angle 2 — Cross-Domain Integration Patterns

## ADR-0012's three patterns, in Menata terms

| Pattern | Portal GA mechanics | Menata mapping |
|---------|--------------------|----------------|
| **A — Cron batch** | Consumer pulls source data nightly, aggregates (KPI auto-feed, benchmarking) | CAP-E02 (time events) + cross-machine read + CAP-F14 (computed aggregation). Composable from planned capabilities |
| **B — Direct call via interface** | Source triggers consumer synchronously through consumer-defined interface, fire-and-forget (observation closed → improvement created) | CAP-A06 (`create_record` in another machine) + CAP-E05 (system events). Composable |
| **C — Domain events via dispatcher** | Source publishes generic event; N consumers subscribe independently; zero source changes to add a consumer | **Not composable from existing capabilities** — needs machine-level event subscription as metadata. New CAP-I01 |

## What Portal GA declares as metadata-shaped documents

### Canonical Event Schema (~65 events)

```yaml
event_name: action.aar_submitted
version: v1.0
category: lifecycle
payload: {id, type, occurred_at, actor_id, branch_id, correlation_id, action_id, ...}
cross_cutting_contribution:
  gamification_points: 30
  competency_tags: ["refleksi", "tanggung_jawab"]
  kpi_metric_refs: ["aar_submission_rate"]
consumers: [messaging, gamification]
deprecation: {deprecated: false}
```

Everything here is expressible as Runtime Metadata: event identity, versioned payload schema, **declared consumers**, deprecation lifecycle, and — remarkably — *cross-cutting contribution weights carried on the event itself*. → CAP-I02, CAP-I05

### Consumer Contracts (22 registered)

```yaml
consumer_expectations:
  required_fields: [{name: ObservedBy, type: uuid, description: poin attribution}]
  semantic_constraints: ["BehaviorType='AT_RISK' → at-risk points; else safe points"]
consumer_behavior:
  on_contract_violation: log_and_continue
  circuit_breaker: enabled
versioning: {contract_version: v1.1, compatible_schema_range: "v1.0 - v1.x"}
```

A contract is **a constraint at the integration boundary** — same conceptual family as Menata's Constraint grammar, applied between machines instead of within one. → CAP-I03

### Error isolation rules (constitutional, 4 rules)

Consumer failure must never break the source; batch continues on item failure; integration dependencies are nil-safe; event handlers fail independently. These are **runtime execution semantics**, not metadata — Menata's dispatcher must be born with these rules. → recorded as design requirements on CAP-I01

### PICA → AAR → Improvement (universal pattern, BRD §9)

Every corrective-action-producing domain MUST flow through one shared machine (Monitor PICA) with a canonical state machine (OPEN → IN_PROGRESS → COMPLETED → AAR_PENDING → AAR_SUBMITTED → VERIFIED → auto-publish), ISO 45001-aligned. In Menata terms this is **composition governance**: a workspace-level rule that machines of a certain kind must reference a canonical machine. The mechanics are covered by CAP-F13 + CAP-A06 + CAP-E06 state guards; the *mandate* is a governance concern → input for Study 9, and a workspace-policy concept flagged for Study 7.

### Correlation & observability

`correlation_id` propagates across every cross-domain event (BRD R17.1); every integration logs `integration_type`, `source_domain`, `target_domain`; SLOs registered per integration. → CAP-I04

**Angle 2 verdict:** Patterns A and B compose from already-registered capabilities. Pattern C and its governing artifacts (event schema, subscriptions, contracts, trace) are genuinely new metadata concepts — registered as the new **Cross-Machine Integration** capability area (CAP-I01…I05).

---

# Angle 3 — Cross-Domain Information Display

| Portal GA pattern | Description | Menata mapping |
|-------------------|-------------|----------------|
| **Shared digest sections** (`dashboard/snapshot/section/`: pica_status, safety_climate, ftw_today, walks_5r, bbs_cadence, compliance_priorities, activity_weekly, metrics_detail) | Each section sources one domain; dashboard composes many sections | **Composed dashboard view** — a view whose sections each bind to a different machine's data. New CAP-V10 |
| **Same sections reused in ADH email digest** | 9 DigestSections render to both web dashboard and Monday-morning email, per-branch, per-timezone cron | **Channel-independent view rendering** — one section definition, multiple render targets (web, email). New CAP-V11 |
| Branch report aggregation (`periodBounds(year, month)`) | Every aggregate query parameterized by org unit + period | Reinforces CAP-X09 (organizational scoping) — display side of the same dimension |
| Cross-domain dashboard cards (FTW national card on home) | Home page surfaces summary cards from many domains | CAP-V10 variant |
| Per-branch timezone rendering (RULE #12) | "Today" differs per branch (WIB/WITA/WIT); display formatted per user timezone | Enriches CAP-A02 (environment data): `today` must be *contextual* — per org-unit timezone, not server UTC. Recorded as a design requirement on CAP-A02/X09 |

**Angle 3 verdict:** composition of views across machines is a real capability family (V10, V11), inseparable from the organizational scoping dimension (X09).

---

# Position Statement — the Study 5 key question

> Can Menata express integration between machines/applications as metadata,
> or does this require a new Grammar?

**Language layer: no new Grammar needed.** The Menata Language already reads naturally for integration — `When Payment Received` (external events, spec 003) generalizes to *"When [another Object's] [Event]"*; `Notify`, `Record`, `Create` already express reactions. A domain expert can say everything Portal GA's integrations do in existing business language. Cross-machine events fit the Event grammar's existing "sources" concept.

**Metadata layer: a new section is needed.** Runtime Metadata must grow an **Integration** concern that the current schema cannot represent: event schema declarations with versioning, consumer subscriptions, contracts with on-violation behavior, and contribution weights. This mirrors how View config already exceeds what the language states — the language says *what*, metadata says *how it binds*.

**Runtime layer: dispatcher semantics are constitutional.** Portal GA's four error-isolation rules must be built into the Menata dispatcher from day one — they are what make N-domain composition survivable in production.

**Strongest evidence:** Portal GA *already maintains its integration knowledge as YAML documents* (event catalog, contracts, context map) and pays humans + CI scripts to keep them true. A metadata-driven runtime makes those documents the executable system itself — eliminating the drift the fitness functions exist to catch.

---

# Registry Impact

New capability area **Cross-Machine Integration** + additions (registry v0.3):

| ID | Capability | Evidence |
|----|-----------|----------|
| CAP-I01 | Cross-machine event subscription (Pattern C as metadata) | 52 subscriptions in Context Map |
| CAP-I02 | Event schema declaration: versioned payload, category, deprecation lifecycle | ~65 events in Canonical Event Schema |
| CAP-I03 | Integration contract (consumer expectations + on-violation behavior) | 22 contracts in registry |
| CAP-I04 | Correlation trace + integration observability (correlation_id, structured integration logs, SLO) | BRD R17.1, SLO registry (26 entries) |
| CAP-I05 | Cross-cutting contribution declaration on events (weights feeding gamification/KPI-like machines) | BRD §10 Contribution Law |
| CAP-X09 | Organizational unit scoping (org dimension on records, permissions, selectors, timezone) | BranchPeriodSelector RULE #11 + timezone RULE #12 |
| CAP-V10 | Composed dashboard view (sections sourcing multiple machines) | 9 shared DigestSections |
| CAP-V11 | Channel-independent view rendering (web + email from one section definition) | ADH email digest reuse |
| CAP-V12 | Multi-step form (wizard) view | HIRADC wizard |

**Input for Study 7 (composite):** PICA-style *canonical shared machines* + workspace-level composition mandates — evidence that composition needs governance concepts beyond per-machine metadata.

**Input for Study 9 (governance):** Portal GA's constitutional stack (fitness functions in CI, ARB decision log, living registries, amendment process) is a working reference implementation of capability governance.

---

# Maintenance

Revisit when implementing CAP-I01…I05 — the Canonical Event Schema and Consumer Contract Registry are the richest available test corpus for Menata's integration metadata design.
