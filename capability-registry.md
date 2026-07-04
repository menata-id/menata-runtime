# Capability Registry

> Artifact 1 of the Capability Roadmap — the single source of record for runtime capabilities.
>
> One row per capability. The registry only grows (ratchet):
> a ✅ capability must never regress — its conformance test guards it.
>
> Status: v0.3 — + Study 5 Portal GA cross-domain additions | Updated: 2026-07-04

Seeded from: the 16-feature platform benchmark (`prototype/README.md`), Case 3 gaps P1–P6
(`prototype/go/docs/examples/README.md`), Study 1 pattern mapping
(`benchmarks/workflow-patterns-mapping.md`), Study 2 platform survey
(`benchmarks/platform-capability-survey.md`), and Study 5 Portal GA survey
(`benchmarks/portal-ga-cross-domain-survey.md`).

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
| CAP-F05 | `user` field | ⚠️ | Case 1 | — | 8 | renders as free text; no user picker, no identity link |
| CAP-F06 | `file` field | ⚠️ | Case 1 | — | 9 | input renders; upload is not stored |
| CAP-F07 | `number` field | ⚠️ | schema doc | — | 10 | falls back to text input; no numeric validation |
| CAP-F08 | `money` field | ⚠️ | schema doc | — | 10 | same fallback |
| CAP-F09 | `boolean` field | ⚠️ | schema doc | — | 10 | same fallback |
| CAP-F10 | `time` / `date_time` / `duration` fields | ⚠️ | schema doc | — | 10 | same fallback |
| CAP-F13 | `reference` field (link to another machine) | ❌ | Case 3 (P1) | WCP-2/13/14, WRP-3 | **1** | — |
| CAP-F14 | Computed / formula field | ❌ | Study 2 survey | Salesforce formula, Frappe | 13 | — |
| CAP-F15 | Field default values (beyond status first-value) | ⚠️ | Study 2 survey | universal (6/6 platforms) | 8 | status default works; other fields have none |

## Event Sources

| ID | Capability | Status | Discovered by | Pattern ref | Prio | Proof |
|----|-----------|--------|---------------|-------------|------|-------|
| CAP-E01 | Business activity event (`When X`) | ✅ | Case 1 | WCP-5/10 | | conformance T10/T12 |
| CAP-E02 | Time-driven event (`Every Day 08:00`) | ❌ | spec 003 + mapping | escalation (WRP) | 7 | — |
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
| CAP-A11 | Date arithmetic in actions (advance by frequency, `+ 1 Month`) | ❌ | Case 4 [UNTARGETED FINDING] | spec 003 date events (`Due Date - 1 Day`) | 7 | — |

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
| CAP-V11 | Channel-independent view rendering (web + email from one section) | ❌ | Study 5 Portal GA | ADH email digest reuse | 14 | — |
| CAP-V12 | Multi-step form (wizard) view | ❌ | Study 5 Portal GA | HIRADC wizard | 11 | — |

## Record Lifecycle

| ID | Capability | Status | Discovered by | Pattern ref | Prio | Proof |
|----|-----------|--------|---------------|-------------|------|-------|
| CAP-R01 | Create record (with default status) | ✅ | Case 1 | — | | conformance T08 |
| CAP-R02 | **Edit / update record via form** | ❌ | **Study 1 code check** | — | **5** | no update form exists — CRUD's U is missing |
| CAP-R03 | Delete / archive record | ❌ | Study 1 code check | — | 12 | — |
| CAP-R04 | Event audit log (record_events, snapshot before mutation) | ⚠️ | Case 1 | — | 9 | logged to DB; no UI to view history |
| CAP-R05 | Pagination on list views | ❌ | Study 1 code check | — | 11 | — |
| CAP-R06 | Data import/export (CSV) | ❌ | Study 2 survey | 5/6 platforms | 12 | — |

## Cross-Cutting

| ID | Capability | Status | Discovered by | Pattern ref | Prio | Proof |
|----|-----------|--------|---------------|-------------|------|-------|
| CAP-X01 | Multi-application, multi-machine in one workspace | ✅ | Case 2 | — | | conformance T01 |
| CAP-X02 | Real authentication (today: prototype role cookie) | ❌ | prototype design | WRP-4 | 13 | — |
| CAP-X03 | Machine-level config block (approval_mode etc.) | ❌ | Case 3 (P4) | — | 4 | — |
| CAP-X04 | Metadata live reload (today: restart required) | ❌ | ADR-002 | K8s reconciliation | 14 | plan in `decisions/002-metadata-loading.md` |
| CAP-X05 | Metadata validation before load (dangling refs, bad operators) | ❌ | Study 1 mapping | Terraform plan-before-apply | 7 | — |
| CAP-X06 | Workspace isolation in routing/authz | ⚠️ | prototype design | — | 15 | loaded but not enforced anywhere |
| CAP-X07 | Auto-generated REST API per machine | ❌ | Study 2 survey | 5/6 platforms | 10 | — |
| CAP-X08 | Metadata package export/import (portable app definition) | ❌ | Study 2 survey | universal (6/6 platforms) | 9 | today: hand-written SQL seeds; blocks "one knowledge, many runtimes" operationally |
| CAP-X09 | Organizational unit scoping (org dimension on records, permissions, selectors, per-unit timezone) | ❌ | Study 5 Portal GA | BranchPeriodSelector + timezone rules | 6 | records today have no org context at all |

## Cross-Machine Integration

| ID | Capability | Status | Discovered by | Pattern ref | Prio | Proof |
|----|-----------|--------|---------------|-------------|------|-------|
| CAP-I01 | Cross-machine event subscription (Pattern C as metadata) | ❌ | Study 5 Portal GA | ADR-0012 Pattern C, 52 subscriptions in Context Map | 5 | dispatcher must be born with the 4 error-isolation rules |
| CAP-I02 | Event schema declaration (versioned payload, category, deprecation) | ❌ | Study 5 Portal GA | Canonical Event Schema (~65 events) | 7 | — |
| CAP-I03 | Integration contract (consumer expectations + on-violation behavior) | ❌ | Study 5 Portal GA | Consumer Contract Registry (22 contracts) | 9 | boundary constraint — same family as Constraint grammar |
| CAP-I04 | Correlation trace + integration observability (correlation_id, SLO) | ❌ | Study 5 Portal GA | BRD R17.1, SLO registry | 10 | — |
| CAP-I05 | Cross-cutting contribution declaration on events (weights to gamification/KPI machines) | ❌ | Study 5 Portal GA | BRD §10 Contribution Law | 13 | — |

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

# Rules

1. **Ratchet** — rows are never deleted; status only moves ❌→⚠️→✅.
2. **No silent regression** — every ✅ must gain an automated proof (Study 4); until then, "manual curl" is the recorded evidence.
3. **New gaps register here first** — example YAML annotations (`[NOT YET]`) are pointers; this table is the record.
4. **Every ❌/⚠️ has a priority or a stated reason to wait.**
