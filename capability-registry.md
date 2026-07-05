# Capability Registry

> Artifact 1 of the Capability Roadmap — the single source of record for runtime capabilities.
>
> One row per capability. The registry only grows (ratchet):
> a ✅ capability must never regress — its conformance test guards it.
>
> Status: v0.12 — + Study 15 fifth-pass (CAP-X05 language-level safeguard against forgotten setup) | Updated: 2026-07-05
> Lifecycle governance (admission test, definition-of-done, extension architecture): `capability-lifecycle.md`
> Field type selection procedure: `benchmarks/005-field-modeling-decision-framework.md`

Seeded from: the 16-feature platform benchmark (`prototype/README.md`), Case 3 gaps P1–P6
(`prototype/go/docs/examples/README.md`), Study 1 pattern mapping
(`benchmarks/000-workflow-patterns-mapping.md`), Study 2 platform survey
(`benchmarks/001-platform-capability-survey.md`), Study 5 Portal GA survey
(`benchmarks/002-portal-ga-cross-domain-survey.md`), Study 6 accounting vertical survey
(`benchmarks/003-accounting-vertical-survey.md`), Study 7 Case 10 composition findings
(`prototype/go/docs/examples/organization-composite.md`), and Study 15 field modeling framework
(`benchmarks/005-field-modeling-decision-framework.md`).

**Status** reflects the Go prototype runtime: ✅ supported · ⚠️ partial · ❌ not yet.
**Prio** is the global implementation ordering (blank = supported or not yet prioritized).
**Proof** names the conformance verification once it exists (Study 4 formalizes these).

---

## Field Types

| ID | Capability | Status | Discovered by | Pattern ref | Prio | Proof |
|----|-----------|--------|---------------|-------------|------|-------|
| CAP-F01 | `text` field | ✅ | Case 1 | — | | conformance T06–T08 |
| CAP-F02 | `rich_text` field (textarea) | ✅ | Case 1 | — | | conformance T02/T08 |
| CAP-F03 | `value_list` field (select + badge) | ✅ | Case 1 | — | | conformance T06–T08 |
| CAP-F04 | `date` field | ✅ | Case 1 | — | | conformance T05/T08 |
| CAP-F05 | `user` field | ⚠️ | Case 1 | — | 8 | renders as free text; no user picker, no identity link. Long-term: sugar over CAP-F13 + CAP-O01 (Study 15), not a permanently separate type — kept distinct only until CAP-O01 exists |
| CAP-F06 | `file` field | ⚠️ | Case 1 | Frappe Attach→File DocType, Salesforce File/ContentDocument, Drupal file entity | 9 | input renders; upload is not stored. Long-term (Study 15): sugar over CAP-F13 + a runtime-managed File/Document entity — files have their own identity/lifecycle (storage key, versioning), same shape of gap as CAP-F05 waiting on CAP-O01 |
| CAP-F07 | `number` field | ⚠️ | schema doc | — | 10 | falls back to text input; no numeric validation |
| CAP-F08 | `money` field | ⚠️ | schema doc | — | 10 | same fallback |
| CAP-F09 | `boolean` field | ⚠️ | schema doc | — | 10 | same fallback |
| CAP-F10 | `time` / `date_time` / `duration` fields | ⚠️ | schema doc | — | 10 | same fallback |
| CAP-F13 | `reference` field (link to another machine) | ❌ | Case 3 (P1) | WCP-2/13/14, WRP-3 | **1** | includes tree/hierarchy option (self-reference + rollup — COA, Study 6). Scope note (Study 15): must support two target flavors from day one — (a) workspace-authored Machine, (b) reserved built-in identity target for CAP-F05/CAP-O01 — designing only for (a) forces a breaking change later. Fourth-pass: this alone (plus an ordinary Machine) fully fixes a single-application mis-modeled field (e.g. `Equipment`) — CAP-O02 is a separate, additional capability only for the cross-application case |
| CAP-F14 | Computed / formula field | ❌ | Study 2 survey | Salesforce formula, Frappe | 13 | design req: derived line generation (tax templates, Study 6). Study 15 (third-pass): this is the correct home for unit/currency conversion (amount × factor → normalized value) — NOT the Constraint grammar |
| CAP-F15 | Field default values (beyond status first-value) | ⚠️ | Study 2 survey | universal (6/6 platforms) | 8 | status default works; other fields have none |
| CAP-F16 | Line items / child table inside a record (header-detail document) | ❌ | Study 6 accounting | Odoo One2many, Frappe Table — universal to document apps | **3** | joins CAP-F13 atop the structural queue |
| CAP-F17 | Multi-currency money (transaction currency + rate + base mirror) | ❌ | Study 6 accounting + Study 15 (independent derivation) | Odoo/ERPNext; spec 001-object.md names Currency as an Object example | 14 | Study 15 reclassified `money` from primitive to reference sugar — Currency fails identity/lifecycle/reuse/cardinality tests, is a CAP-O02 master-data candidate |
| CAP-F18 | Auto-numbering / document sequences | ❌ | Study 6 accounting | ir.sequence, Naming Series — universal | 7 | Study 2 missed it |

## Event Sources

| ID | Capability | Status | Discovered by | Pattern ref | Prio | Proof |
|----|-----------|--------|---------------|-------------|------|-------|
| CAP-E01 | Business activity event (`When X`) | ✅ | Case 1 | WCP-5/10 | | conformance T10/T12 |
| CAP-E02 | Time-driven event (`Every Day 08:00`) | ❌ | spec 003 + mapping | escalation (WRP); Study 15 boundary check confirms placement against iCalendar RRULE (RFC 5545) | 7 | recurring schedules are Event/Action grammar, never a Field concern — confirmed, not just assumed |
| CAP-E03 | Date-driven event (`When Due Date - 1 Day`) | ❌ | spec 003 + mapping | — | 11 | — |
| CAP-E04 | External event (webhook, payment) | ❌ | spec 003 + mapping | — | 12 | — |
| CAP-E05 | Internal / system-triggered event | ❌ | Case 3 (P6) | WRP-11 | 6 | — |
| CAP-E06 | **State-conditional event availability** (event allowed only in given status) | ❌ | **Study 1 mapping** | **WCP-18 Milestone, WCP-16** | **2** | — |

> CAP-E06 is the headline finding of Study 1: today an Approved record can still be
> Rejected — events are filtered by role only, never by current state. No case had
> surfaced this; the benchmark predicted it.

## Actions

| ID | Capability | Status | Discovered by | Pattern ref | Prio | Proof |
|----|-----------|--------|---------------|-------------|------|-------|
| CAP-A01 | `set_field` with static value | ✅ | Case 1 | — | | conformance T10 |
| CAP-A02 | `set_field` with dynamic value (`now`, `today`, `current_user`) | ❌ | Case 3 (P2) | WDP-7 Environment Data | 3 | — |
| CAP-A03 | `notify` to role | ⚠️ | Case 1 | — | 5 | slog only — no real delivery channel |
| CAP-A04 | `notify` to dynamic recipient (record's approver/submitter) | ❌ | Case 3 | WRP | 5 | — |
| CAP-A06 | `create_record` in another machine | ❌ | schema doc | WCP-13/14 MI | 13 | — |
| CAP-A07 | `activate_next` (sequential step activation) | ❌ | Case 3 (P3) | WCP-1 Sequence | 4 | — |
| CAP-A08 | `aggregate_status` (parent rollup: all-approved / any-rejected / cancel cascade) | ❌ | Case 3 (P3) | WCP-3/9/19/20 | 4 | — |
| CAP-A09 | Conditional actions (`if` inside events) | ❌ | spec 003 + mapping | WCP-4/6, WDP-39 | 7 | — |
| CAP-A10 | Notification delivery channels (email, in-app) | ❌ | Study 2 survey | universal (6/6 platforms) | 5 | prerequisite for CAP-A03 being real |
| CAP-A11 | Date arithmetic in actions (advance by frequency, `+ 1 Month`) | ❌ | Case 4 [UNTARGETED FINDING] | spec 003 date events (`Due Date - 1 Day`); Study 15 confirms this + CAP-E02 fully cover recurring-schedule needs | 7 | — |

## Constraints

| ID | Capability | Status | Discovered by | Pattern ref | Prio | Proof |
|----|-----------|--------|---------------|-------------|------|-------|
| CAP-C01 | `required` operator | ✅ | Case 1 | — | | conformance T04 |
| CAP-C02 | `after: today` operator | ✅ | Case 1 | — | | conformance T05 |
| CAP-C03 | `equals` / `not_equals` (in conditions) | ✅ | Case 1 | — | | conformance T06 |
| CAP-C04 | Conditional constraint (`condition` block) | ✅ | Case 1 | WDP-38 | | conformance T06/T07 |
| CAP-C05 | Comparison operators (`greater_than`, `less_than`, date `before`) | ❌ | schema doc + Case 4 | — | 10 | only `after: today` exists |
| CAP-C07 | Cross-field comparison (End Date after Start Date) | ❌ | Study 1 mapping | — | 10 | — |
| CAP-C08 | Cross-record constraint (one request per employee per day) | ❌ | spec 004 + mapping | — | 14 | — |
| CAP-C09 | **Constraints evaluated on event trigger** (today: Create only) | ❌ | **Study 1 mapping** | WDP-38 | **2** | — |
| CAP-C10 | Aggregate constraint over line items (sum(debit) = sum(credit)) | ❌ | Study 6 accounting | double-entry invariant | 7 | depends on CAP-F16. Study 15 (third-pass): must operate on already-normalized values (post-conversion via CAP-F14) — the constraint itself never performs currency/unit conversion |
| CAP-C11 | Temporal period constraint (no posting into closed period) | ❌ | Study 6 accounting | lock dates, Period Closing Voucher | 10 | — |

## Permissions

| ID | Capability | Status | Discovered by | Pattern ref | Prio | Proof |
|----|-----------|--------|---------------|-------------|------|-------|
| CAP-P01 | Role-based event permission | ✅ | Case 1 | WRP-2 | | conformance T11/T12 |
| CAP-P02 | Record-level ownership (only assigned user may act) | ❌ | Case 3 (P5) | WRP-1 Direct Allocation | 6 | — |
| CAP-P03 | Separation of duties (Requester ≠ Approver) | ❌ | spec 004 + mapping | WRP-5 | 11 | — |
| CAP-P04 | Delegation | ❌ | Study 1 mapping | WRP detour | 15 | — |
| CAP-P05 | CRUD-level permissions (read/create/edit per role — not just events) | ❌ | Study 2 survey | universal (6/6 platforms) | 6 | today every role sees every machine and record |
| CAP-P06 | Field-level visibility ("Salary visible only to HR") | ❌ | Study 2 survey + spec 004 example | Salesforce field perms, Frappe permlevel | 11 | — |

## Views

| ID | Capability | Status | Discovered by | Pattern ref | Prio | Proof |
|----|-----------|--------|---------------|-------------|------|-------|
| CAP-V01 | `form` view (fields config drives inputs) | ✅ | Case 1 | — | | conformance T02 |
| CAP-V02 | `list` view (columns config drives table) | ✅ | Case 1 | — | | conformance T03 |
| CAP-V03 | `detail` view (all fields) | ✅ | Case 1 | — | | conformance T09 |
| CAP-V04 | `default_sort` honored in list | ⚠️ | Study 1 code check | — | 9 | loaded into model; store hardcodes `created_at DESC` |
| CAP-V05 | Filtered list (my records / pending my approval) | ❌ | Case 3 | — | 8 | — |
| CAP-V06 | Child records sub-list on parent detail | ❌ | Case 3 (P1) | — | 3 | depends on CAP-F13 |
| CAP-V07 | `dashboard` / `calendar` / `timeline` views | ❌ | schema doc | — | 15 | — |
| CAP-V08 | List search & filter | ❌ | Study 2 survey | universal (6/6 platforms) | 8 | — |
| CAP-V09 | Declarative view-level filter (Due Today, Overdue Tasks) | ❌ | Case 4 [UNTARGETED FINDING] | — | 8 | view `filter` block in metadata |
| CAP-V10 | Composed dashboard view (sections sourcing multiple machines) | ❌ | Study 5 Portal GA | 9 shared DigestSections | 12 | — |
| CAP-V11 | Channel-independent view rendering (web + email from one section) | ❌ | Study 5 Portal GA | ADH email digest reuse | 14 | **evidence-thin** (Study 9 retrofit: single source, possibly composable) — HOLD at Proposed until second independent source |
| CAP-V12 | Multi-step form (wizard) view | ❌ | Study 5 Portal GA | HIRADC wizard | 11 | — |
| CAP-V13 | Aggregate report view (group-by, hierarchy rollup, period compare, running balance) | ❌ | Study 6 accounting | Trial Balance, P&L, GL | 9 | the report class every vertical needs |

## Record Lifecycle

| ID | Capability | Status | Discovered by | Pattern ref | Prio | Proof |
|----|-----------|--------|---------------|-------------|------|-------|
| CAP-R01 | Create record (with default status) | ✅ | Case 1 | — | | conformance T08 |
| CAP-R02 | **Edit / update record via form** | ❌ | **Study 1 code check** | — | **5** | no update form exists — CRUD's U is missing |
| CAP-R03 | Delete / archive record | ❌ | Study 1 code check | — | 12 | — |
| CAP-R04 | Event audit log (record_events, snapshot before mutation) | ⚠️ | Case 1 | — | 9 | logged to DB; no UI to view history |
| CAP-R05 | Pagination on list views | ❌ | Study 1 code check | — | 11 | — |
| CAP-R06 | Data import/export (CSV) | ❌ | Study 2 survey | 5/6 platforms | 12 | — |
| CAP-R07 | Record immutability after state (posted/submitted frozen; amend-via-new-version) | ❌ | Study 6 accounting + Case 6 declaration | docstatus model | 6 | stronger than CAP-E06 — guards edits, not just events |

## Cross-Cutting

| ID | Capability | Status | Discovered by | Pattern ref | Prio | Proof |
|----|-----------|--------|---------------|-------------|------|-------|
| CAP-X01 | Multi-application, multi-machine in one workspace | ✅ | Case 2 | — | | conformance T01 |
| CAP-X02 | Real authentication (today: prototype role cookie) | ❌ | prototype design | WRP-4 | 13 | — |
| CAP-X03 | Machine-level config block (approval_mode etc.) | ❌ | Case 3 (P4) | — | 4 | — |
| CAP-X04 | Metadata live reload (today: restart required) | ❌ | ADR-002 | K8s reconciliation | 14 | plan in `decisions/002-metadata-loading.md` |
| CAP-X05 | Metadata validation before load (dangling refs, bad operators) | ❌ | Study 1 mapping | Terraform plan-before-apply | 7 | Scope extended (Study 15, fifth-pass): composite/reference-sugar types (`money`, future `quantity`) must have their required companion (`currency:`/`currency_field:`) declared inline — missing companion = load-time rejection, same discipline as CAP-F13's dangling-reference check. Language-level safeguard against a metadata author (human or AI) forgetting the conversion setup — not an app-UI concern |
| CAP-X06 | Workspace isolation in routing/authz | ⚠️ | prototype design | — | 8 | implementation strategy decided: PostgreSQL RLS (ADR-003) |
| CAP-X07 | Auto-generated REST API per machine | ❌ | Study 2 survey | 5/6 platforms | 10 | — |
| CAP-X08 | Metadata package export/import (portable app definition) | ❌ | Study 2 survey | universal (6/6 platforms) | 9 | today: hand-written SQL seeds; blocks "one knowledge, many runtimes" operationally |
| CAP-X09 | Organizational unit scoping (org dimension on records, permissions, selectors, per-unit timezone) | ❌ | Study 5 Portal GA | BranchPeriodSelector + timezone rules | 6 | records today have no org context at all |
| CAP-X10 | Metadata-driven index management (hot fields from view filters/sorts → expression indexes, reconciled) | ❌ | Study 8 [SCALE FINDING] | K8s reconciliation applied to indexes | 10 | ADR-003 |
| CAP-X11 | Lazy per-workspace metadata loading + cache (singleflight, LRU, LISTEN/NOTIFY eviction) | ❌ | Study 8 [SCALE FINDING] | ADR-002 Option C unified with scale cache | 7 | ADR-003; retires boot-time LoadAll |

## Cross-Machine Integration

| ID | Capability | Status | Discovered by | Pattern ref | Prio | Proof |
|----|-----------|--------|---------------|-------------|------|-------|
| CAP-I01 | Cross-machine event subscription (Pattern C as metadata) | ❌ | Study 5 Portal GA | ADR-0012 Pattern C, 52 subscriptions in Context Map | 5 | dispatcher must be born with the 4 error-isolation rules |
| CAP-I02 | Event schema declaration (versioned payload, category, deprecation) | ❌ | Study 5 Portal GA | Canonical Event Schema (~65 events) | 7 | — |
| CAP-I03 | Integration contract (consumer expectations + on-violation behavior) | ❌ | Study 5 Portal GA | Consumer Contract Registry (22 contracts) | 9 | boundary constraint — same family as Constraint grammar |
| CAP-I04 | Correlation trace + integration observability (correlation_id, SLO) | ❌ | Study 5 Portal GA | BRD R17.1, SLO registry | 10 | — |
| CAP-I05 | Cross-cutting contribution declaration on events (weights to gamification/KPI machines) | ❌ | Study 5 Portal GA | BRD §10 Contribution Law | 13 | — |

## Workspace Services

All discovered by Case 10 `[COMPOSITION FINDING]` — capabilities that belong to the workspace itself, not to any application (Study 7).

| ID | Capability | Status | Discovered by | Pattern ref | Prio | Proof |
|----|-----------|--------|---------------|-------------|------|-------|
| CAP-O01 | Workspace identity & role registry (users, namespaced roles, user→role assignment as metadata) | ❌ | Case 10 | spec 005 (roles ≠ users) | 6 | prototype role cookie is workspace-blind |
| CAP-O02 | Master data designation (canonical machines: ownership, cross-app referenceability, deactivation semantics) | ❌ | Case 10 (cross-app narrative), Study 15 (`Currency` via CAP-F17) | DDD shared kernel, Portal GA PICA + Data Mesh | 8 | flag + rules on machine, not a new hierarchy level. Scope corrected (Study 15 fourth-pass): `Equipment` used only within one application does NOT need this — that's fully resolved by CAP-F13 alone. CAP-O02 evidence is specifically the cross-application case (Case 10) + Currency, still clearing the dual-evidence bar |
| CAP-O03 | Navigation metadata (app grouping, role-aware menus, workspace home) | ❌ | Case 10 | runtime/004 already names Navigation — spec predicted it | 9 | — |
| CAP-O04 | Workspace-wide search across machines (permission-trimmed) | ❌ | Case 10 | — | 12 | depends on CAP-P05 |
| CAP-O05 | Unified notification center (inbox, per-user channel preferences, digest grouping) | ❌ | Case 10 | Portal GA message dispatcher | 8 | extends CAP-A10 to workspace service |
| CAP-O06 | Business calendar (holidays, working-day rules) consumable by date arithmetic/SLA | ❌ | Case 10 | spec 001 Holiday example | 9 | feeds CAP-A11, CAP-E02 |

---

# Implementation Order (consolidated)

| Prio | Capabilities | Theme |
|------|-------------|-------|
| 1 | CAP-F13 | Reference fields — biggest unlock (6 patterns depend on it) |
| 2 | CAP-E06, CAP-C09 | Correctness: state guards + constraints on events |
| 3 | CAP-A02, CAP-V06 | Dynamic values + child sub-list (completes Case 3 basics) |
| 4 | CAP-A07, CAP-A08, CAP-X03 | Workflow actions + machine config (Case 3 complete) |
| 5 | CAP-R02, CAP-A03/A04, CAP-A10 | Record editing + real notify (with delivery channels) |
| 6 | CAP-P02, CAP-P05, CAP-E05 | Record/CRUD-level permission + system events |
| 7 | CAP-E02, CAP-A09, CAP-X05 | Time-driven events + conditional actions + metadata validation |
| 8+ | remainder | See per-table Prio column |

Case 3 numbering map: P1→CAP-F13 · P2→CAP-A02 · P3→CAP-A07/A08 · P4→CAP-X03 · P5→CAP-P02 · P6→CAP-E05.

---

# Tracked but Not Yet Studied

`runtime/006-runtime-model.md` declares a hierarchy under Machine — Page, View, Service, Workflow, Navigation, API, Configuration. Per the "silence is not a decision" rule, each is recorded here rather than silently left out:

| Model concept | Registry coverage | Note |
|---------------|-------------------|------|
| View | CAP-V01…V13 | Studied (Study 1–8) |
| Navigation | CAP-O03 | Studied (Study 7, as a Workspace Service) |
| Configuration | CAP-X03 (machine-level) | Studied (Case 3) |
| **Page** | none | Not yet studied — how a Page composes multiple Views is undefined in the registry |
| **Service** | none | Not yet studied — background jobs, scheduled execution as a declared concept (overlaps CAP-E02 but not identical) |
| **Workflow** | none | Not yet studied as its own concept — current registry treats workflow as emergent from Event+Constraint+Permission+Action, per `runtime/004`'s own stated design ("Workflow behavior should emerge from events, constraints, permissions, actions"); revisit if a case shows this is insufficient |
| **API** | CAP-X07 (auto-generated REST) | Partially studied — API *as declared surface* (vs auto-generated) not yet examined |
| **Theme** | none | Not yet studied — presentation/branding metadata |

Action: candidates for a future study once Phase 4 restructuring completes.

---

# Rules

1. **Ratchet** — rows are never deleted; status only moves ❌→⚠️→✅.
2. **No silent regression** — every ✅ must gain an automated proof (Study 4); until then, "manual curl" is the recorded evidence.
3. **New gaps register here first** — example YAML annotations (`[NOT YET]`) are pointers; this table is the record.
4. **Every ❌/⚠️ has a priority or a stated reason to wait.**
