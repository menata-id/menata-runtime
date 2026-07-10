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
| 5 | Inventory / Stock Movement | Calculation & multi-record transaction: quantity math, balance updates | F07, F13, F14, F16, F19, A02, A06, C05, C07, C08, X12 | ⚠️ documented, gaps registered |
| 6 | Petty Cash Ledger | Numeric aggregation & immutability: running balance, append-only, period close | F08, F14, C05, C08, R04 | planned |
| 7 | Customer Complaint | Unstructured case management (CMMN-style): ad-hoc steps, SLA, escalation, reopen | E02, E05, A09, P04, WCP-10 cycles | planned |
| 8 | Payment Confirmation | External events: webhook ingestion, idempotency, reconciliation | E04, X07, C08 | planned |
| 9 | Accounting (journal, monthly close, trial balance) | Vertical depth: header-detail documents, aggregate line invariants, immutability | F13, F14, F16, F18, A02, C10, C11, P03, R04, R07, E06, V13 | ⚠️ documented, gaps registered |
| 10 | Organization Composite | Emergent capabilities at composition: shared identity, master data, cross-app navigation, org-wide reporting | F13+I01–I05+X09+V10+P05 composition test | ⚠️ documented (Study 7) — 6 `[COMPOSITION FINDING]` → CAP-O01…O06 |

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

**Business reality:** A distributor stocks items in more than one unit of measure (e.g. cement moves
in box/dozen/piece; rice moves in kilogram/sack). Goods move in and out of a warehouse. Each
confirmed movement appends an immutable ledger entry and the item's stock-on-hand recomputes from
it. Stock can never go negative — an outgoing movement larger than what is on hand must be rejected.

**External benchmark:** `benchmarks/006-inventory-warehouse-benchmark.md` — the six-stage WMS flow
(receiving → putaway → storage → picking → packing → shipping) and APICS/ASCM inventory-control
concepts (FIFO/FEFO, lot/serial tracking, reservation/allocation, multi-location balance, costed
valuation) benchmarked first, so this case's scope is a deliberate subset — four follow-on case
candidates are queued, not silently dropped.

**Declared targets:**

| Target | Capability | Pattern |
|--------|-----------|---------|
| `Item` → `Stock Movement` / `Stock Ledger` links | CAP-F13 | `reference` field |
| `Stock On Hand` = rollup of Stock Ledger entries | CAP-F14 | Computed / aggregate field |
| `Unit Conversions` child table on Item (Tier 2: Cement — BOX/DOZEN/PCS) | CAP-F16 | Line items — proves Study 15's Quantity Tier 2 |
| `Quantity` + `Unit` with a fixed conversion pair (Tier 1: Rice — KG/SAK) | **CAP-F19 (new)** | Tiered UoM conversion — proves Study 15's Quantity Tier 1 |
| `Movement Date` stamped `Today` at Confirm | CAP-A02 | WDP-7 Environment Data |
| Confirm → `create_record` into Stock Ledger | CAP-A06 | WCP-13/14 Multiple Instance |
| `Quantity greater_than 0` | CAP-C05 | Comparison operator |
| `Movement Date >= Requested Date` (same record) | CAP-C07 | Cross-field comparison |
| `Item.Stock On Hand >= Normalized Quantity` before an Out movement | CAP-C08 | Cross-record constraint |
| Ledger append + balance recompute as one unit | **CAP-X12 (new)** | Cross-record write atomicity — the cluster the original declaration flagged as untargeted |

**Predicted new findings:** CAP-F19 (Quantity's tiered UoM conversion) and CAP-X12 (multi-record
write atomicity) have no registry entry before this case — both should be registered on write-up,
each already carrying dual evidence (benchmark + case) per `capability-lifecycle.md`'s admission test.

Files: `prototype/go/docs/examples/inventory-item.{menata,yaml}`,
`inventory-stock-movement.{menata,yaml}`, `inventory-stock-ledger.{menata,yaml}`
(three Machines, one file pair each — same convention as Case 3's
`approval-document` / `approval-step`)

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

# Case 9 — Accounting (target declaration)

**Business reality:** Small-org bookkeeping — a chart of accounts, manually authored journal
entries (header + debit/credit lines), a monthly close that locks the period, and a trial balance
report. Whoever *prepares* an entry must not be the same person who *posts* it (segregation of
duties), and every state change must be traceable (audit trail) — bookkeeping is a controls
problem as much as a data-entry problem.

**External benchmark:** `benchmarks/003-accounting-vertical-survey.md` — Study 6's original
Odoo/ERPNext platform survey, plus a **World-Class Standards Addendum** (2026-07-10) benchmarking
GAAP chart-of-accounts convention and SOX internal-control requirements directly, not just what two
platforms happen to implement. The addendum caught a real gap in Study 6's own original declaration
(below).

**Declared targets:**

| Target | Capability | Pattern |
|--------|-----------|---------|
| `Journal Entry` → `Journal Entry Line` / `Chart of Account` / `Fiscal Period` links | CAP-F13 | `reference`, including self-reference for COA hierarchy |
| `Normal Balance` derived from `Account Type` (Asset/Expense→Debit, Liability/Equity/Revenue→Credit) | CAP-F14 | Computed field — static categorical lookup (GAAP-standard, deterministic) |
| `Lines` child table on Journal Entry (Account, Debit, Credit, Memo) | CAP-F16 | Line items — the actual line-level facts a Trial Balance reports over |
| `Entry Number` auto-numbered (`JE-2026-00001`) | CAP-F18 | Auto-numbering |
| `Posting Date`, `Prepared By`, `Posted By` stamped at their respective events | CAP-A02 | WDP-7 Environment Data |
| `sum(Lines.Debit) = sum(Lines.Credit)` before Post | CAP-C10 | Aggregate line constraint — the double-entry invariant |
| No posting into a Closed Fiscal Period | CAP-C11 | Temporal period constraint |
| **`Posted By` must not equal `Prepared By`** | **CAP-P03 (first case evidence)** | WRP-5 Separation of Duties — SOX requirement, missed by Study 6's original declared-targets list |
| **Every Draft→Posted transition traceable** | **CAP-R04 (reinforced)** | SOX audit trail — already registered, but this case is the first where it is a compliance requirement, not optional |
| Posted entries frozen | CAP-E06 + CAP-R07 | State-conditional availability + immutability-after-state |
| Trial Balance (group by Account, sum Debit/Credit) | CAP-V13 | Aggregate report view |

**Deliberately out of scope (unchanged from Study 6, restated with reasons in the benchmark
addendum):** invoice posting derivation, reconciliation, multi-currency (CAP-F17). Also
out-of-scope: enforcing the account-number-prefix-matches-type convention — GAAP itself does not
require it, so Menata should not invent a constraint the standard doesn't have.

**Correction to Study 6:** the original seven targets covered *structural* accounting capabilities
but missed the *control* capabilities (CAP-P03, CAP-R04) that make bookkeeping trustworthy under
SOX — found by benchmarking the standard directly, not just the two platforms. Both are added above.

Files: `prototype/go/docs/examples/accounting-chart-of-account.{menata,yaml}`,
`accounting-journal-entry.{menata,yaml}`, `accounting-journal-entry-line.{menata,yaml}`,
`accounting-fiscal-period.{menata,yaml}` (four Machines, one file pair each)

---

# Process per case

```text
1. Declare targets in this document (table above)
2. Write .menata (Business Knowledge — no runtime concerns)
3. Write .yaml with [SUPPORTED]/[NOT YET]/[PARTIAL] annotations
4. Register new findings in capability-registry.md (flag [UNTARGETED FINDING])
5. Seed + exercise the supported subset
6. Update 000-workflow-patterns-mapping.md marks if a pattern is newly exercised
```
