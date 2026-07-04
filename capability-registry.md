# Capability Registry

> Artifact 1 of the Capability Roadmap — the single source of record for runtime capabilities.
>
> One row per capability. The registry only grows (ratchet):
> a ✅ capability must never regress — its conformance test guards it.
>
> Status: v0.1 — Study 1 initial seeding | Created: 2026-07-04

Seeded from: the 16-feature platform benchmark (`prototype/README.md`), Case 3 gaps P1–P6
(`prototype/go/docs/examples/README.md`), and Study 1 pattern mapping
(`benchmarks/workflow-patterns-mapping.md`).

**Status** reflects the Go prototype runtime: ✅ supported · ⚠️ partial · ❌ not yet.
**Prio** is the global implementation ordering (blank = supported or not yet prioritized).
**Proof** names the conformance verification once it exists (Study 4 formalizes these).

---

## Field Types

| ID | Capability | Status | Discovered by | Pattern ref | Prio | Proof |
|----|-----------|--------|---------------|-------------|------|-------|
| CAP-F01 | `text` field | ✅ | Case 1 | — | | manual curl (Case 1) |
| CAP-F02 | `rich_text` field (textarea) | ✅ | Case 1 | — | | manual curl (Case 1) |
| CAP-F03 | `value_list` field (select + badge) | ✅ | Case 1 | — | | manual curl (Case 1) |
| CAP-F04 | `date` field | ✅ | Case 1 | — | | manual curl (Case 1) |
| CAP-F05 | `user` field | ⚠️ | Case 1 | — | 8 | renders as free text; no user picker, no identity link |
| CAP-F06 | `file` field | ⚠️ | Case 1 | — | 9 | input renders; upload is not stored |
| CAP-F07 | `number` field | ⚠️ | schema doc | — | 10 | falls back to text input; no numeric validation |
| CAP-F08 | `money` field | ⚠️ | schema doc | — | 10 | same fallback |
| CAP-F09 | `boolean` field | ⚠️ | schema doc | — | 10 | same fallback |
| CAP-F10 | `time` / `date_time` / `duration` fields | ⚠️ | schema doc | — | 10 | same fallback |
| CAP-F13 | `reference` field (link to another machine) | ❌ | Case 3 (P1) | WCP-2/13/14, WRP-3 | **1** | — |

## Event Sources

| ID | Capability | Status | Discovered by | Pattern ref | Prio | Proof |
|----|-----------|--------|---------------|-------------|------|-------|
| CAP-E01 | Business activity event (`When X`) | ✅ | Case 1 | WCP-5/10 | | manual curl (Case 2) |
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
| CAP-A01 | `set_field` with static value | ✅ | Case 1 | — | | manual curl (Case 2) |
| CAP-A02 | `set_field` with dynamic value (`now`, `today`, `current_user`) | ❌ | Case 3 (P2) | WDP-7 Environment Data | 3 | — |
| CAP-A03 | `notify` to role | ⚠️ | Case 1 | — | 5 | slog only — no real delivery channel |
| CAP-A04 | `notify` to dynamic recipient (record's approver/submitter) | ❌ | Case 3 | WRP | 5 | — |
| CAP-A06 | `create_record` in another machine | ❌ | schema doc | WCP-13/14 MI | 13 | — |
| CAP-A07 | `activate_next` (sequential step activation) | ❌ | Case 3 (P3) | WCP-1 Sequence | 4 | — |
| CAP-A08 | `aggregate_status` (parent rollup: all-approved / any-rejected / cancel cascade) | ❌ | Case 3 (P3) | WCP-3/9/19/20 | 4 | — |
| CAP-A09 | Conditional actions (`if` inside events) | ❌ | spec 003 + mapping | WCP-4/6, WDP-39 | 7 | — |

## Constraints

| ID | Capability | Status | Discovered by | Pattern ref | Prio | Proof |
|----|-----------|--------|---------------|-------------|------|-------|
| CAP-C01 | `required` operator | ✅ | Case 1 | — | | manual curl (Case 2) |
| CAP-C02 | `after: today` operator | ✅ | Case 1 | — | | manual curl (Case 2) |
| CAP-C03 | `equals` / `not_equals` (in conditions) | ✅ | Case 1 | — | | manual curl (Case 1) |
| CAP-C04 | Conditional constraint (`condition` block) | ✅ | Case 1 | WDP-38 | | manual curl (Case 1) |
| CAP-C05 | Numeric comparison (`greater_than`, `less_than`) | ❌ | schema doc | — | 10 | — |
| CAP-C07 | Cross-field comparison (End Date after Start Date) | ❌ | Study 1 mapping | — | 10 | — |
| CAP-C08 | Cross-record constraint (one request per employee per day) | ❌ | spec 004 + mapping | — | 14 | — |
| CAP-C09 | **Constraints evaluated on event trigger** (today: Create only) | ❌ | **Study 1 mapping** | WDP-38 | **2** | — |

## Permissions

| ID | Capability | Status | Discovered by | Pattern ref | Prio | Proof |
|----|-----------|--------|---------------|-------------|------|-------|
| CAP-P01 | Role-based event permission | ✅ | Case 1 | WRP-2 | | manual curl (Case 2, 403) |
| CAP-P02 | Record-level ownership (only assigned user may act) | ❌ | Case 3 (P5) | WRP-1 Direct Allocation | 6 | — |
| CAP-P03 | Separation of duties (Requester ≠ Approver) | ❌ | spec 004 + mapping | WRP-5 | 11 | — |
| CAP-P04 | Delegation | ❌ | Study 1 mapping | WRP detour | 15 | — |

## Views

| ID | Capability | Status | Discovered by | Pattern ref | Prio | Proof |
|----|-----------|--------|---------------|-------------|------|-------|
| CAP-V01 | `form` view (fields config drives inputs) | ✅ | Case 1 | — | | manual curl (Case 2) |
| CAP-V02 | `list` view (columns config drives table) | ✅ | Case 1 | — | | manual curl (Case 2) |
| CAP-V03 | `detail` view (all fields) | ✅ | Case 1 | — | | manual curl (Case 2) |
| CAP-V04 | `default_sort` honored in list | ⚠️ | Study 1 code check | — | 9 | loaded into model; store hardcodes `created_at DESC` |
| CAP-V05 | Filtered list (my records / pending my approval) | ❌ | Case 3 | — | 8 | — |
| CAP-V06 | Child records sub-list on parent detail | ❌ | Case 3 (P1) | — | 3 | depends on CAP-F13 |
| CAP-V07 | `dashboard` / `calendar` / `timeline` views | ❌ | schema doc | — | 15 | — |

## Record Lifecycle

| ID | Capability | Status | Discovered by | Pattern ref | Prio | Proof |
|----|-----------|--------|---------------|-------------|------|-------|
| CAP-R01 | Create record (with default status) | ✅ | Case 1 | — | | manual curl (Case 2) |
| CAP-R02 | **Edit / update record via form** | ❌ | **Study 1 code check** | — | **5** | no update form exists — CRUD's U is missing |
| CAP-R03 | Delete / archive record | ❌ | Study 1 code check | — | 12 | — |
| CAP-R04 | Event audit log (record_events, snapshot before mutation) | ⚠️ | Case 1 | — | 9 | logged to DB; no UI to view history |
| CAP-R05 | Pagination on list views | ❌ | Study 1 code check | — | 11 | — |

## Cross-Cutting

| ID | Capability | Status | Discovered by | Pattern ref | Prio | Proof |
|----|-----------|--------|---------------|-------------|------|-------|
| CAP-X01 | Multi-application, multi-machine in one workspace | ✅ | Case 2 | — | | manual curl (Case 2) |
| CAP-X02 | Real authentication (today: prototype role cookie) | ❌ | prototype design | WRP-4 | 13 | — |
| CAP-X03 | Machine-level config block (approval_mode etc.) | ❌ | Case 3 (P4) | — | 4 | — |
| CAP-X04 | Metadata live reload (today: restart required) | ❌ | ADR-002 | K8s reconciliation | 14 | plan in `decisions/002-metadata-loading.md` |
| CAP-X05 | Metadata validation before load (dangling refs, bad operators) | ❌ | Study 1 mapping | Terraform plan-before-apply | 7 | — |
| CAP-X06 | Workspace isolation in routing/authz | ⚠️ | prototype design | — | 15 | loaded but not enforced anywhere |

---

# Implementation Order (consolidated)

| Prio | Capabilities | Theme |
|------|-------------|-------|
| 1 | CAP-F13 | Reference fields — biggest unlock (6 patterns depend on it) |
| 2 | CAP-E06, CAP-C09 | Correctness: state guards + constraints on events |
| 3 | CAP-A02, CAP-V06 | Dynamic values + child sub-list (completes Case 3 basics) |
| 4 | CAP-A07, CAP-A08, CAP-X03 | Workflow actions + machine config (Case 3 complete) |
| 5 | CAP-R02, CAP-A03/A04 | Record editing + real notify |
| 6 | CAP-P02, CAP-E05 | Record-level permission + system events |
| 7 | CAP-E02, CAP-A09, CAP-X05 | Time-driven events + conditional actions + metadata validation |
| 8+ | remainder | See per-table Prio column |

Case 3 numbering map: P1→CAP-F13 · P2→CAP-A02 · P3→CAP-A07/A08 · P4→CAP-X03 · P5→CAP-P02 · P6→CAP-E05.

---

# Rules

1. **Ratchet** — rows are never deleted; status only moves ❌→⚠️→✅.
2. **No silent regression** — every ✅ must gain an automated proof (Study 4); until then, "manual curl" is the recorded evidence.
3. **New gaps register here first** — example YAML annotations (`[NOT YET]`) are pointers; this table is the record.
4. **Every ❌/⚠️ has a priority or a stated reason to wait.**
