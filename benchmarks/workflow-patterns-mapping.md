# Workflow Patterns Mapping

> Artifact 2 of the Capability Roadmap.
>
> Maps Menata against the Workflow Patterns Initiative catalog
> (van der Aalst & ter Hofstede, workflowpatterns.com) —
> the canonical benchmark used to evaluate BPEL, BPMN, YAWL, jBPM, and others.
>
> Status: v0.1 — Study 1 deliverable | Created: 2026-07-04

---

# How to read this document

Each pattern is assessed on **three layers**, because a pattern may be expressible in the language long before the runtime can execute it:

| Layer | Question |
|-------|----------|
| **L** — Menata Language | Can a domain expert express this in `.menata`? |
| **M** — Runtime Metadata | Does the metadata schema define a representation for it? |
| **R** — Go Prototype Runtime | Does the current runtime actually execute it? |

Marks: ✅ yes · ⚠️ partial · ❌ no · `—` out-of-scope-by-design (reason stated)

Capability IDs (`CAP-…`) reference `../capability-registry.md`.

---

# Control-Flow Patterns (basic 20)

| # | Pattern | L | M | R | Notes | CAP |
|---|---------|---|---|---|-------|-----|
| WCP-1 | Sequence | ✅ | ❌ | ❌ | Language: "Activate Next Step". Metadata/runtime: `activate_next` action does not exist. | CAP-A07 |
| WCP-2 | Parallel Split | ✅ | ⚠️ | ❌ | Modeled as multiple child records (Approval Steps). Needs reference fields to link children to parent. | CAP-F13 |
| WCP-3 | Synchronization | ✅ | ❌ | ❌ | "All steps approved → document approved". Needs `aggregate_status` action. | CAP-A08 |
| WCP-4 | Exclusive Choice | ✅ | ❌ | ❌ | Spec 003-event.md defines event conditions (`if Design Type = Poster`). Metadata has no condition on actions. | CAP-A09 |
| WCP-5 | Simple Merge | ✅ | ✅ | ✅ | Multiple events converging on the same status value works naturally in the state model. | CAP-E01 |
| WCP-6 | Multi-Choice | ✅ | ❌ | ❌ | Several conditional responses may fire. Same gap as WCP-4. | CAP-A09 |
| WCP-7 | Structured Synchronizing Merge | — | — | — | Advanced merge semantics. Deferred until WCP-3/6 exist. |  |
| WCP-8 | Multi-Merge | — | — | — | Same. |  |
| WCP-9 | Structured Discriminator | ✅ | ❌ | ❌ | "Any step rejected → document rejected" — first negative decision wins. | CAP-A08 |
| WCP-10 | Arbitrary Cycles | ✅ | ✅ | ✅ | Rework loops work today: an event may set Status back to Draft (e.g. Revise). | CAP-E01 |
| WCP-11 | Implicit Termination | ✅ | ✅ | ✅ | A record simply stops receiving events; no explicit terminator required. | — |
| WCP-12 | MI without Synchronization | — | — | — | Multiple-instance activities without sync — no identified business case yet. |  |
| WCP-13 | MI design-time knowledge | ✅ | ❌ | ❌ | Fixed set of approval steps per document type. Needs reference + child creation. | CAP-F13, CAP-A06 |
| WCP-14 | MI run-time knowledge | ✅ | ❌ | ❌ | Approver list chosen per document at submit time (Case 3 core scenario). | CAP-F13, CAP-A06 |
| WCP-15 | MI without a priori knowledge | — | — | — | Steps added while in flight — defer until WCP-14 works. |  |
| WCP-16 | Deferred Choice | ✅ | ⚠️ | ⚠️ | Approve and Reject offered simultaneously, first wins — but nothing prevents the second from also firing (see WCP-18). | CAP-E06 |
| WCP-17 | Interleaved Parallel Routing | — | — | — | Ordering constraints without parallelism — no business case yet. |  |
| WCP-18 | Milestone | ✅ | ❌ | ❌ | **Event availability by state.** Runtime filters events by role only — an Approved document can still be Rejected. Most fundamental correctness gap found by this mapping. | CAP-E06 |
| WCP-19 | Cancel Activity | ✅ | ✅ | ⚠️ | Withdraw/Cancel events set a terminal status, but do not deactivate sibling steps. | CAP-A08 |
| WCP-20 | Cancel Case | ✅ | ❌ | ❌ | Halting an entire document + all its steps needs cross-record cascade. | CAP-A08 |

**Mapping yield:** WCP-18 (Milestone / state guards) is a gap that no case had surfaced explicitly — the benchmark predicted it. This validates the dual-track method.

---

# Data Patterns (core subset)

| # | Pattern | L | M | R | Notes | CAP |
|---|---------|---|---|---|-------|-----|
| WDP-4 | Case Data | ✅ | ✅ | ✅ | `records.data` JSONB — all fields visible across the record lifecycle. | — |
| WDP-7 | Environment Data | ✅ | ❌ | ❌ | `now`, `today`, `current_user` in set_field values. | CAP-A02 |
| WDP-9/10 | Data Interaction between cases | ✅ | ❌ | ❌ | Step updates Document — cross-record data flow. | CAP-F13, CAP-A08 |
| WDP-38 | Data-based Routing (pre) | ✅ | ⚠️ | ⚠️ | Constraint `condition` works — but constraints run only on Create, never on event trigger. | CAP-C09 |
| WDP-39 | Event conditions (post) | ✅ | ❌ | ❌ | `if` inside events (spec 003) has no metadata representation. | CAP-A09 |
| — | Cross-field comparison | ✅ | ❌ | ❌ | "End Date must be after Start Date" — operators compare field↔literal only. | CAP-C07 |
| — | Cross-record constraint | ✅ | ❌ | ❌ | "One leave request per employee per day" (spec 004 example). | CAP-C08 |

---

# Resource Patterns (core subset)

| # | Pattern | L | M | R | Notes | CAP |
|---|---------|---|---|---|-------|-----|
| WRP-2 | Role-Based Allocation | ✅ | ✅ | ✅ | Permission Guard: role string → permitted events. Proven by Case 2 (403 test). | CAP-P01 |
| WRP-1 | Direct Allocation | ✅ | ❌ | ❌ | "Only the assigned Approver may act on their Step" — record-level ownership. | CAP-P02 |
| WRP-3 | Deferred Allocation | ✅ | ❌ | ❌ | Approver chosen at submit time (Case 3) — depends on reference fields. | CAP-F13 |
| WRP-4 | Authorization | ✅ | ✅ | ⚠️ | Role permissions exist; real authentication is a prototype cookie. | CAP-X02 |
| WRP-5 | Separation of Duties | ✅ | ❌ | ❌ | "Requester must not be the Approver" (spec 004 example). | CAP-P03 |
| WRP-11 | Automatic Execution | ✅ | ❌ | ❌ | System-triggered events without a user request. | CAP-E05 |
| — | Delegation | ⚠️ | ❌ | ❌ | Not yet in language examples. Defer. | CAP-P04 |
| — | Escalation | ✅ | ❌ | ❌ | "Every Day / if Due Date < Today / Notify" (spec 003 example) — needs time-driven events. | CAP-E02 |

---

# Event Source Coverage (from spec 003-event.md)

The Event specification names four sources. Only one is realized:

| Source | Example in spec | L | M | R | CAP |
|--------|----------------|---|---|---|-----|
| Business Activity | `When Submit` | ✅ | ✅ | ✅ | CAP-E01 |
| Time | `Every Day 08:00`, `Every Monday` | ✅ | ❌ | ❌ | CAP-E02 |
| Date | `When Due Date - 1 Day` | ✅ | ❌ | ❌ | CAP-E03 |
| External | `When Payment Received`, `When Webhook Received` | ✅ | ❌ | ❌ | CAP-E04 |

---

# Summary Scorecard

| Category | Assessed | ✅ R | ⚠️ R | ❌ R | Out of scope |
|----------|----------|------|------|------|--------------|
| Control-Flow (basic 20) | 20 | 3 | 2 | 10 | 5 |
| Data (core) | 7 | 1 | 1 | 5 | 0 |
| Resource (core) | 8 | 1 | 1 | 6 | 0 |
| Event Sources | 4 | 1 | 0 | 3 | 0 |

**Headline findings of Study 1:**

1. **CAP-E06 State-conditional event availability (WCP-18 Milestone)** — the most fundamental correctness gap; found by the benchmark, not by any case. An Approved document can still be Rejected today.
2. **CAP-C09 Constraints only run on Create** — never on event trigger. Data-based preconditions are silently unenforced for events.
3. **Reference fields (CAP-F13)** remain the single biggest unlock — they appear in 6 of the mapped patterns.
4. The **Menata Language already expresses nearly everything** (L column almost all ✅) — the gaps are concentrated in Metadata schema and Runtime. This confirms the language design is ahead of the runtime, as intended.

---

# Maintenance

- Re-assess after each runtime extension lands; move marks ❌→⚠️→✅.
- When a case exercises a pattern, add the case number to Notes.
- Out-of-scope (`—`) entries must keep a stated reason; revisit each quarter.
