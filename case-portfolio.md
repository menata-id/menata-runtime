# Case Portfolio

> Artifact 3 of the Capability Roadmap — deliberate case selection.
>
> Cases are chosen to hit untested pattern clusters, not at random.
> Target patterns are declared **before** the case is written, so each case
> is a designed experiment — and surprises (patterns the case reveals that
> were not targeted) are themselves findings.
>
> Status: v0.1 — Study 3 deliverable | Created: 2026-07-04

---

# Rules

1. **Declare targets first.** Before writing a case, list the capabilities/patterns it is designed to exercise.
2. **One dominant cluster per case.** A case may touch many capabilities but should be *the* proving ground for one cluster.
3. **Business realism over synthetic coverage.** Every case must be a process a real organization runs — Business Knowledge first, benchmark second.
4. **Document the surprises.** Capabilities surfaced that were not in the target list get flagged `[UNTARGETED FINDING]` and registered.

---

# Portfolio

| # | Case | Dominant cluster | Primary targets (CAP) | Status |
|---|------|------------------|----------------------|--------|
| 1 | Design Request | CRUD + simple state machine | F01–F04, E01, A01, C01–C04, P01, V01–V03 | ✅ done |
| 2 | Leave Request | Domain portability (same cluster, different domain) | same as Case 1 | ✅ done |
| 3 | Document Approval | Multi-instance workflow: sequence, synchronization, resource allocation | F13, A07, A08, X03, P02, E05 | ⚠️ documented, gaps registered |
| 4 | Maintenance Reminder | **Time-driven behavior**: schedules, escalation, environment data | E02, E03, A02, A09, A11 | ⚠️ documented (this study) |
| 5 | Inventory / Stock Movement | Calculation & multi-record transaction: quantity math, balance updates | F07, F14, A06, C05, C07 | planned |
| 6 | Petty Cash Ledger | Numeric aggregation & immutability: running balance, append-only, period close | F08, F14, C05, C08, R04 | planned |
| 7 | Customer Complaint | Unstructured case management (CMMN-style): ad-hoc steps, SLA, escalation, reopen | E02, E05, A09, P04, WCP-10 cycles | planned |
| 8 | Payment Confirmation | External events: webhook ingestion, idempotency, reconciliation | E04, X07, C08 | planned |
| 9 | Accounting (journal, tax, reports) | Vertical depth: multi-record invariants (debit=credit), period-close immutability, report views | targets declared at Study 6 | planned (Phase 2) |
| 10 | Organization Composite | Emergent capabilities at composition: shared identity, master data, cross-app navigation, org-wide reporting | targets declared at Study 7 | planned (Phase 2) |

Sequencing follows the registry's implementation order: Case 4 (time) precedes Case 5–6 (calculation) because escalation and scheduling appear in Cases 6–7 too; external events (Case 8) come last because they depend on API surface (X07).

---

# Case 4 — Maintenance Reminder (target declaration)

**Business reality:** Equipment needs recurring maintenance. Tasks are due on a schedule; overdue tasks escalate to a supervisor. Whoever completes a task records it, and the next due date advances by the frequency.

**Declared targets:**

| Target | Capability | Pattern |
|--------|-----------|---------|
| `Every Day 07:00` machine-level schedule | CAP-E02 | Event source: Time |
| `if Next Due Date = Today` on a time event | CAP-A09 | WCP-4, WDP-38 |
| `Last Completed: Today` stamping | CAP-A02 | WDP-7 Environment Data |
| Overdue escalation to a second role | CAP-E02 + A09 | WRP escalation |
| `Next Due Date advance by Frequency` | **new — date arithmetic in actions** | — |

**Predicted new findings:** date arithmetic (`+ 1 Month`, `advance by frequency`) has no capability entry yet — this case should force its registration.

Files: `prototype/go/docs/examples/maintenance-reminder.menata` / `.yaml`

---

# Case 5 — Inventory / Stock Movement (target declaration)

**Business reality:** Goods in/out of a warehouse. Each movement changes stock on hand. Stock cannot go negative.

**Declared targets:** number fields with real numeric handling (F07), computed field for stock-on-hand (F14), `create_record` for movement log (A06), `greater_than` operators (C05), cross-field comparison (C07), and a new cluster: **multi-record atomicity** (movement + balance must update together).

---

# Case 6 — Petty Cash Ledger (target declaration)

**Business reality:** Small cash box. Every expense recorded, running balance maintained, month is closed and cannot be edited after closing.

**Declared targets:** money fields (F08), running balance (F14 aggregate variant), cross-record constraint — expenses cannot exceed balance (C08), **immutability after state** (period close — a stronger form of CAP-E06 state guards applied to editing, not just events), audit trail visibility (R04).

---

# Case 7 — Customer Complaint (target declaration)

**Business reality:** Complaints arrive, get triaged, may need ad-hoc investigation steps, have response SLAs, can be reopened by the customer.

**Declared targets:** SLA timers (E02/E03), escalation & delegation (P04), reopen cycles (WCP-10 — already ✅, proving it in a richer flow), system events (E05), and the CMMN-style question: **can Menata express work that has no predefined step sequence?** This case deliberately probes the language boundary, not just the runtime.

---

# Case 8 — Payment Confirmation (target declaration)

**Business reality:** Customer pays via bank/payment gateway; a webhook confirms payment; the matching invoice must update exactly once (idempotent), unmatched payments queue for manual reconciliation.

**Declared targets:** external events (E04), REST/webhook surface (X07), idempotency — duplicate webhook must not double-apply (new capability, likely), cross-record matching (C08 variant).

---

# Process per case

```text
1. Declare targets in this document (table above)
2. Write .menata (Business Knowledge — no runtime concerns)
3. Write .yaml with [SUPPORTED]/[NOT YET]/[PARTIAL] annotations
4. Register new findings in capability-registry.md (flag [UNTARGETED FINDING])
5. Seed + exercise the supported subset
6. Update workflow-patterns-mapping.md marks if a pattern is newly exercised
```
