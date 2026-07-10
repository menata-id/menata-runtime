# Metadata Examples

Four cases documented here. Cases 1 and 2 run fully on the current prototype. Cases 3 and 4 are boundary tests — showing what Business Knowledge looks like for complex domains, and where the current runtime needs to be extended. Case selection follows `runtime/case-portfolio.md`.

## How to read these files

Each case has two files:

| File | What it is |
|------|-----------|
| `*.menata` | Business Knowledge in Menata Language — written by a domain expert, technology-independent |
| `*.yaml` | Runtime Metadata — the machine-readable realization, maps directly to the DB schema |

The `.yaml` files for Case 3 include inline annotations: `[SUPPORTED]`, `[NOT YET]`, `[PARTIAL]`.

---

## Case 1 — Design Request

**Domain:** Creative services workflow  
**Application:** Design  
**Roles:** Requester, Designer  
**Seed:** `seeds/001_design_request.sql`  
**Status:** ✅ Fully supported

```
design-request.menata   Menata Language source
design-request.yaml     Runtime Metadata (DB realization)
```

**Workflow:** Requester submits → Designer accepts/rejects → starts work → completes  
**Notable:** Conditional constraint — Attachment required only when Design Type = Banner 2:1

| Grammar | Count |
|---------|-------|
| Fields | 7 (user, value_list ×2, date, text, rich_text, file) |
| Events | 5 (Submit, Accept, Reject, Start, Complete) |
| Constraints | 4 (2 required, 1 date future, 1 conditional) |
| Permissions | 2 roles |
| Views | 4 (form, list ×2, detail) |

---

## Case 2 — Leave Request

**Domain:** HR — employee leave approval  
**Application:** HR  
**Roles:** Employee, Manager  
**Seed:** `seeds/002_leave_request.sql`  
**Status:** ✅ Fully supported

```
leave-request.menata    Menata Language source
leave-request.yaml      Runtime Metadata (DB realization)
```

**Workflow:** Employee submits → Manager approves or rejects; Employee may cancel before approval  
**Notable:** Different application, different roles from Case 1 — no code change required

| Grammar | Count |
|---------|-------|
| Fields | 6 (user, value_list ×2, date ×2, rich_text) |
| Events | 4 (Submit, Approve, Reject, Cancel) |
| Constraints | 2 (reason required, start date future) |
| Permissions | 2 roles |
| Views | 4 (form, list ×2, detail) |

---

## Case 3 — Document Approval System

**Domain:** Multi-approver document approval with sequential or parallel mode  
**Application:** Approval  
**Roles:** Submitter, Approver, System  
**Seed:** — (not yet, pending runtime extensions)  
**Status:** ⚠️ Partially supported — see gap analysis below

```
approval-document.menata    Menata Language source — Approval Document
approval-document.yaml      Runtime Metadata + inline gap annotations
approval-step.menata        Menata Language source — Approval Step
approval-step.yaml          Runtime Metadata + inline gap annotations
```

**Workflow:**
```
Submitter creates Document → sets approvers + mode (Sequential | Parallel)
    ↓
Submit → Status: In Review → notify Approvers
    ↓
Each Approver acts on their Approval Step
    │
    ├── Sequential: Step 2 activates only after Step 1 Approved
    └── Parallel:   All Steps active simultaneously
    ↓
All Steps Approved → Document: Approved → notify Submitter
Any Step Rejected  → Document: Rejected → notify Submitter
```

**Two machines, linked by reference:**

| Machine | Grammar | Notes |
|---------|---------|-------|
| Approval Document | 6 fields, 4 events, 3 constraints, 2 roles, 5 views | Parent |
| Approval Step | 6 fields, 2 events, 1 conditional constraint, 1 role, 2 views | Child — references Document |

---

### Gap Analysis

#### What works now

| Feature | Status | Notes |
|---------|--------|-------|
| Approval Document as standalone machine | ✅ | Fields, form, list, detail all render |
| Approval Step as standalone machine | ✅ | Same |
| Conditional constraint (Notes required if Rejected) | ✅ | operator: equals in condition |
| set_field actions (Decision, Status) | ✅ | Works on both machines independently |
| notify action | ✅ | Static role string |
| evt_ad_withdraw | ✅ | Simple set_field |

#### What does not work yet

| Feature | Gap | What's needed |
|---------|-----|---------------|
| Step links to Document | `type: reference` field not implemented | New field type in model, loader, store, handler, UI |
| Steps shown on Document detail page | Cross-machine query (list steps by parent) | `store.ListByParent(machineID, parentFieldID, parentRecordID)` |
| "Pending My Approval" filtered list | Cross-machine filter by current user | Record-level query + user context in store |
| Sequential activation | `activate_next` action type doesn't exist | New executor action: find sibling step with sequence+1, set it active |
| All approved → Document approved | `aggregate_status` action type doesn't exist | New executor action: check all siblings, trigger parent event if resolved |
| System-triggered events | No internal event trigger mechanism | Internal event bus or post-action hook in executor |
| `value: now` in set_field | Dynamic value expressions not supported | Value resolver: `now`, `today`, `current_user` |
| Record-level permission (only assigned Approver) | Permission checks role string only | Field-level ownership check: `fld_as_approver = current_user` |
| Approval mode drives behavior | Machine-level config not in schema | New `config` block on Machine in Runtime Metadata schema |

---

### What Case 3 reveals about the runtime roadmap

The gaps above map to concrete runtime extensions, in priority order:

**P1 — Reference fields** (blocks everything else in Case 3)
- Schema: add `type: reference` + `target_machine` to field definition
- Loader: load reference config from `options` JSONB
- Store: `ListByParent` query
- Handler: Detail page renders child records as sub-list
- UI: reference field renders as link, not free text input

**P2 — Dynamic value expressions in set_field**
- Executor: value resolver for `now`, `today`, `current_user`
- Enables: `Decided At` stamping on approve/reject

**P3 — New action types**
- `activate_next` — sequential approval step activation
- `aggregate_status` — parent status rollup when all/any children resolve

**P4 — Machine-level config**
- Schema: `config` block on Machine (approval_mode_field, steps_machine, steps_parent_field)
- Loader: load and expose machine config to executor

**P5 — Record-level permissions**
- Permission guard: check `field = current_user` in addition to role string
- Enables: only the assigned Approver can act on their Step

**P6 — Internal event triggering**
- Executor: fire an event on a record without HTTP request (for system-triggered events)
- Enables: aggregate_status triggering parent Approve/Reject automatically

---

## Case 4 — Maintenance Reminder

**Domain:** Facility — recurring equipment maintenance with overdue escalation
**Application:** Facility
**Roles:** Technician, Supervisor
**Seed:** — (pending time-driven event support)
**Status:** ⚠️ Documented boundary test — first portfolio-driven case (targets declared before writing, see `runtime/case-portfolio.md`)

```
maintenance-reminder.menata    Menata Language source
maintenance-reminder.yaml      Runtime Metadata + inline gap annotations
```

**Workflow:**
```
Task scheduled with Frequency (Daily | Weekly | Monthly) and Next Due Date
    ↓
Every Day 07:00: Next Due Date = Today → Status: Due → notify Assignee
    ↓
Every Day 07:00: still Due past date → Status: Overdue → notify Supervisor (escalation)
    ↓
Complete → stamp Last Completed = today → advance Next Due Date by Frequency → back to Scheduled
```

**Declared targets vs findings:**

| Target | Result |
|--------|--------|
| Time-driven events (CAP-E02) | confirmed gap — no `trigger` block in schema |
| Event conditions (CAP-A09) | confirmed gap — including compound AND conditions |
| Environment data `today` (CAP-A02) | confirmed gap |
| Dynamic notify recipient (CAP-A04) | confirmed gap — escalation needs record's Supervisor |
| Date arithmetic | **[UNTARGETED FINDING]** → registered as CAP-A11 |
| Declarative view filter (Due Today list) | **[UNTARGETED FINDING]** → registered as CAP-V09 |

---

## Case 5 — Inventory / Stock Movement

**Domain:** Warehouse — items stocked in multiple units of measure, movements ledgered, balance never negative
**Application:** Warehouse
**Roles:** Warehouse Staff
**Seed:** — (pending reference field + computed field support)
**Status:** ⚠️ Documented boundary test — proves Study 15's Quantity tiering framework (`runtime/benchmarks/005-field-modeling-decision-framework.md`); external benchmark done first (`runtime/benchmarks/006-inventory-warehouse-benchmark.md`), see `runtime/case-portfolio.md`

```
inventory-item.menata               Menata Language source — Item (master, UoM tiering)
inventory-item.yaml                 Runtime Metadata + inline gap annotations
inventory-stock-movement.menata     Menata Language source — Stock Movement (request)
inventory-stock-movement.yaml       Runtime Metadata + inline gap annotations
inventory-stock-ledger.menata       Menata Language source — Stock Ledger Entry (append-only)
inventory-stock-ledger.yaml         Runtime Metadata + inline gap annotations
```

**Workflow:**
```
Item carries Base Unit + (Tier 1: a fixed conversion pair, e.g. Rice KG<->SAK,
                          Tier 2: a Unit Conversions child table, e.g. Cement BOX/DOZEN/PCS)
    ↓
Stock Movement requested: Item, Movement Type (In/Out/Adjustment), Quantity, Unit
    ↓
Confirm: Movement Date stamped Today
       → if Out, check Item.Stock On Hand >= Normalized Quantity (else rejected)
       → append Stock Ledger Entry (signed, normalized quantity)
       → Status: Confirmed
    ↓
Item.Stock On Hand recomputes as the rollup of its Stock Ledger Entries
```

**Declared targets vs findings:**

| Target | Result |
|--------|--------|
| Reference fields (CAP-F13) | confirmed gap — Item↔Movement↔Ledger links |
| Computed / aggregate field (CAP-F14) | confirmed gap — Normalized Quantity + Stock On Hand rollup |
| Child table (CAP-F16) | confirmed gap — Unit Conversions (Tier 2 proof) |
| Quantity/UoM tiered conversion | **[UNTARGETED FINDING, promoted]** → registered as CAP-F19 (Study 15 prediction, now case-evidenced) |
| Environment data `today` (CAP-A02) | confirmed gap — same as Case 4 |
| `create_record` (CAP-A06) | confirmed gap — Confirm → Stock Ledger Entry |
| Comparison operator `greater_than` (CAP-C05) | confirmed gap |
| Cross-field comparison (CAP-C07) | confirmed gap — Movement Date vs Requested Date |
| Cross-record constraint (CAP-C08) | confirmed gap — the negative-stock invariant |
| Multi-record write atomicity | **[UNTARGETED FINDING, promoted]** → registered as CAP-X12 (named but unregistered since the original Case 5 declaration; now case-evidenced) |

---

## Case 6 — Petty Cash Ledger

**Domain:** Small cash box run as an imprest fund — fixed float, one accountable Custodian, independent reconciliation
**Application:** Petty Cash
**Roles:** Custodian, Auditor
**Seed:** — (pending reference field + computed field support)
**Status:** ⚠️ Documented boundary test — grounded in imprest-fund control practice, see `runtime/case-portfolio.md`

```
pettycash-fund.menata / .yaml       Petty Cash Fund (master, running balance)
pettycash-voucher.menata / .yaml    Petty Cash Voucher (each expense)
pettycash-period.menata / .yaml     Cash Period (reconciliation cycle, closes and freezes)
```

**Declared targets vs findings:**

| Target | Result |
|--------|--------|
| `money` field (CAP-F08) | **first real case evidence** — previously schema-doc only |
| Reference, computed rollup (CAP-F13, CAP-F14) | confirmed gap — same shapes as Case 5 |
| Fund-balance guard (CAP-C08) | confirmed gap — third case instance |
| Reconciliation formula (CAP-C10) | confirmed gap — a scalar-plus-aggregate variant of Case 9's debit=credit |
| Independent reconciler (CAP-P03) | confirmed gap — third case instance (audit control, not approval) |
| Period close + freeze (CAP-E06 + CAP-R07) | confirmed gap |

---

## Case 7 — Customer Complaint

**Domain:** Customer service — ad-hoc investigation, SLA-timed, reopenable, escalated on breach
**Application:** Customer Service
**Roles:** Agent, Supervisor, Customer
**Seed:** — (pending event-condition + state-guard support)
**Status:** ⚠️ Documented boundary test — grounded in CMMN (OMG standard), see `runtime/case-portfolio.md`

```
complaint.menata / .yaml    Complaint (one Machine — see the CMMN finding below)
```

**Declared targets vs findings:**

| Target | Result |
|--------|--------|
| No fixed step sequence — any permitted event, any order, gated by Status | **The CMMN boundary finding**: Menata expresses *bounded* flexibility, not *unbounded* (case-worker-invented) flexibility. Stated explicitly, not a gap |
| Priority-keyed SLA offset (CAP-A11) | confirmed gap — new sub-pattern (keyed offset, not flat frequency) |
| Auto-escalation (CAP-E02 + CAP-A09 + CAP-E05) | confirmed gap — CAP-E05's second instance, a same-record self-trigger flavor |
| Ordinal priority stepping | **[UNTARGETED FINDING]** → registered as CAP-A12 |
| Delegation (CAP-P04) | **first real case evidence** — previously "not yet in language examples" |
| Reopen cycle (WCP-10) | reinforced — a richer instance than Case 1's rework loop |

---

## Case 8 — Payment Confirmation

**Domain:** Payments — webhook-confirmed payment, idempotent ingestion, manual reconciliation to Invoice
**Application:** Payments
**Roles:** Finance, System
**Seed:** — (pending reference field + external event support)
**Status:** ⚠️ Documented boundary test — grounded in Stripe/Shopify/GitHub webhook idempotency convention, see `runtime/case-portfolio.md`

```
payment-invoice.menata / .yaml         Invoice (the receivable)
payment-webhook-event.menata / .yaml   Payment Webhook Event (raw ingestion, dedup)
payment.menata / .yaml                 Payment (matched/unmatched, reconciled manually)
```

**Declared targets vs findings:**

| Target | Result |
|--------|--------|
| External event (CAP-E04) | **first real case evidence** |
| Webhook idempotency | **[UNTARGETED FINDING]** → registered as CAP-X13 |
| `create_record` on Receive (CAP-A06) | confirmed gap |
| Cross-record field write | **[UNTARGETED FINDING]** → registered as CAP-A13 |
| Write-chain atomicity (CAP-X12) | reinforced |
| Payment↔Invoice matching (CAP-C08) | **deliberately kept manual** — correlation matching is a different sub-pattern, out of scope by design |

---

## Case 9 — Accounting

**Domain:** Small-org bookkeeping — chart of accounts, journal entries, monthly close, trial balance
**Application:** Accounting
**Roles:** Accountant, Supervisor
**Seed:** — (pending reference field + child table + computed field support)
**Status:** ⚠️ Documented boundary test — field-level design checked against GAAP chart-of-accounts convention and SOX internal controls, not just the Odoo/ERPNext platform survey (`runtime/benchmarks/003-accounting-vertical-survey.md` — Study 6 + World-Class Standards Addendum), see `runtime/case-portfolio.md`

```
accounting-chart-of-account.menata      Menata Language source — Chart of Account (master, hierarchy)
accounting-chart-of-account.yaml        Runtime Metadata + inline gap annotations
accounting-journal-entry.menata         Menata Language source — Journal Entry (header)
accounting-journal-entry.yaml           Runtime Metadata + inline gap annotations
accounting-journal-entry-line.menata    Menata Language source — Journal Entry Line (report source)
accounting-journal-entry-line.yaml      Runtime Metadata + inline gap annotations
accounting-fiscal-period.menata         Menata Language source — Fiscal Period (monthly close)
accounting-fiscal-period.yaml           Runtime Metadata + inline gap annotations
```

**Workflow:**
```
Chart of Account: 5 GAAP types (Asset/Liability/Equity/Revenue/Expense), Normal Balance
                   derived from Type, hierarchical via Parent Account
    ↓
Journal Entry drafted: Entry Date, Fiscal Period, Prepared By (stamped at Create), Lines
    ↓
Post: Posting Date + Posted By stamped Today/current_user
    → sum(Lines.Debit) must equal sum(Lines.Credit)          (double-entry invariant)
    → Fiscal Period must not be Closed                       (period lock)
    → Posted By must not equal Prepared By                   (SOX segregation of duties)
    → Status: Posted (frozen thereafter)
    ↓
Fiscal Period.Close: every Journal Entry in the period must already be Posted
    ↓
Trial Balance: Journal Entry Line grouped by Account, Debit/Credit summed across every entry
```

**Declared targets vs findings:**

| Target | Result |
|--------|--------|
| Reference fields (CAP-F13) | confirmed gap — Journal Entry↔Line↔Account↔Period links, plus Account's self-reference hierarchy (first case evidence for the tree option) |
| Computed field, static lookup sub-pattern (CAP-F14) | confirmed gap — Normal Balance derived from Account Type; **third confirmed CAP-F14 sub-pattern**, alongside Study 15's conversion and Case 5's aggregate rollup |
| Child table (CAP-F16) | confirmed gap — Journal Entry Lines; also surfaces a **reporting-independence note**: unlike Case 5's Item Unit Conversion, these rows must stay queryable across every parent document for the Trial Balance |
| Auto-numbering (CAP-F18) | confirmed gap — Entry Number |
| Environment data `today`/`current_user` (CAP-A02) | confirmed gap — same as Case 4 and Case 5 |
| Aggregate line constraint (CAP-C10) | confirmed gap — the double-entry invariant |
| Temporal period constraint (CAP-C11) | confirmed gap — closed-period lock |
| Separation of duties (CAP-P03) | **first real case evidence** — registered since Study 1 (spec 004 example) but no case had exercised it until now; the World-Class Standards Addendum caught that Study 6's original 7 targets missed this SOX-driven capability entirely |
| Event audit log (CAP-R04) | reinforced — accounting is the first case where an audit trail is a compliance requirement, not optional |
| State guard + immutability (CAP-E06 + CAP-R07) | confirmed gap — Posted entries frozen |
| Aggregate report view (CAP-V13) | confirmed gap — Trial Balance |
| Cross-record constraint (CAP-C08) | confirmed gap — **second case instance**, and in the *reverse direction* from Case 5's (one Fiscal Period checks all its many Journal Entries, vs. Case 5's one Movement checking an aggregate on one Item) |

---

# Extended Portfolio (Cases 11–21)

Screened against the registry first — cases that would only re-prove an earlier case's capability
cluster are marked light-touch rather than written as if novel. See `runtime/case-portfolio.md` for
the full novelty screening table.

## Case 11 — Social App (Instagram-like)

**Domain:** Social — posts, likes, comments, follows, feed
**Application:** Social
**Roles:** Member
**Seed:** — (pending reference field + join Machine + counter support)
**Status:** ⚠️ Documented boundary test — first case to need a many-to-many relationship, see `runtime/case-portfolio.md`

```
social-post.menata / .yaml       Post (the content)
social-follow.menata / .yaml     Follow (many-to-many join — new pattern)
social-like.menata / .yaml       Like (second many-to-many join instance)
social-comment.menata / .yaml    Comment (ordinary one-to-many, for contrast)
```

**Declared targets vs findings:**

| Target | Result |
|--------|--------|
| Many-to-many relationship | **[UNTARGETED FINDING]** → registered as CAP-F20 — neither CAP-F13 (one-directional) nor CAP-F16 (parent-owned) names a join Machine where both sides are independently queryable |
| Composite uniqueness | **[UNTARGETED FINDING]** → registered as CAP-C12 — also retroactively explains an assumption every case since Case 1 made silently for single-field uniqueness |
| Counter fields (Like Count, Comment Count) | confirmed gap — surfaces an open design question (maintained counter vs. read-time aggregate) rather than resolving it |
| Feed as relationship-filtered list | confirmed gap — a two-hop extension of CAP-V05, not a new capability |

---

## Case 12 — Community Site

**Domain:** Community — groups, events, status posts, participation points, auto-awarded badges
**Application:** Community
**Roles:** Member, Admin, System
**Seed:** — (pending reference field + join Machine + aggregate-conditioned action support)
**Status:** ⚠️ Documented boundary test — builds on Case 11's many-to-many finding, adds gamification, see `runtime/case-portfolio.md`

```
community-group.menata / .yaml          Group
community-membership.menata / .yaml     Membership (third CAP-F20 join instance)
community-event.menata / .yaml          Event
community-points.menata / .yaml         Point Ledger Entry (ledger-shaped, like Case 6)
community-badge-award.menata / .yaml    Badge Award (auto-triggered on threshold)
```

**Declared targets vs findings:**

| Target | Result |
|--------|--------|
| Many-to-many (CAP-F20) | reinforced — third case instance |
| Uniqueness (CAP-C12) | reinforced — second and third instance |
| Aggregate-conditioned action | **[UNTARGETED FINDING]** → registered as CAP-A14 — a badge trigger gated on `sum(points)`, not a single field |

---

## Case 13 — Blog / One-Page Site

**Domain:** Public blog — Posts and Comments readable by anyone, written/moderated by an authenticated Author
**Application:** Blog
**Roles:** Author, Visitor
**Seed:** — (pending reference field + public-access support)
**Status:** ⚠️ Documented boundary test — first case where a role requires NO login, see `runtime/case-portfolio.md`

```
blog-post.menata / .yaml       Post
blog-comment.menata / .yaml    Comment (from an unauthenticated Visitor)
```

**Declared targets vs findings:**

| Target | Result |
|--------|--------|
| Public/unauthenticated access | **[UNTARGETED FINDING]** → registered as CAP-P07 — structurally different from "a role with few permissions"; every case since Case 1 assumed a logged-in role |
| One-page composed landing | reinforces CAP-V10, doesn't close the separate Page/Theme registry gaps |
| Multi-select tags | confirmed scope gap on CAP-F03 (single-select only) |

---

## Case 14 — Lending Services

**Domain:** Lending — loan approval, disbursement with a generated repayment schedule, repayment tracking, overdue detection
**Application:** Lending
**Roles:** Loan Officer
**Seed:** — (pending reference field + batch-generation support)
**Status:** ⚠️ Documented boundary test — see `runtime/case-portfolio.md`

```
lending-loan-application.menata / .yaml         Loan Application
lending-loan.menata / .yaml                     Loan (generates its own schedule on Disburse)
lending-repayment-schedule-entry.menata / .yaml Repayment Schedule Entry (the generated series)
lending-repayment.menata / .yaml                Repayment
```

**Declared targets vs findings:**

| Target | Result |
|--------|--------|
| Batch schedule generation | **[UNTARGETED FINDING]** → registered as CAP-A15 — one action creating N related records from a formula, no existing action shape covers it |
| Separation of duties (CAP-P03) | reinforced — fourth case instance |
| Cross-record write (CAP-A13) | reinforced |
| Time-driven overdue check (CAP-E02) | reinforced — same shape as Case 4 |

---

## Case 15 — E-commerce

**Domain:** E-commerce — product catalog, cart, checkout into a real order
**Application:** E-commerce
**Roles:** Customer, Fulfillment
**Seed:** — (pending reference field + child table support)
**Status:** ⚠️ Documented boundary test — heavily composes Cases 5/8/9, see `runtime/case-portfolio.md`

```
ecommerce-product.menata / .yaml    Product
ecommerce-cart.menata / .yaml       Cart (mutable pre-commit document — new pattern)
ecommerce-order.menata / .yaml      Order (Case 9's header+line shape, reused)
```

**Declared targets vs findings:**

| Target | Result |
|--------|--------|
| Cart as editable scratch state | **[UNTARGETED FINDING]** → registered as CAP-R08 — no invariants enforced until Checkout, unlike Case 9's Draft Journal Entry which is validated immediately |
| Order header+lines | reuses Case 9's CAP-F16 shape, including its reporting-independence note |

---

## Case 16 — Point of Sale

**Domain:** Retail — a single in-person transaction, line items + immediate payment + stock deduction
**Application:** Point of Sale
**Roles:** Cashier
**Status:** ⚠️ Documented — pure composition of Cases 5/8/15, no new capability, see `runtime/case-portfolio.md`

```
pos-sale.menata / .yaml    Sale (one Machine, deliberately)
```

No new findings — written to confirm the composition holds, not to discover anything.

---

## Case 17 — Helpdesk

**Domain:** Internal IT support — same shape as Case 7's Complaint, internal audience
**Application:** Helpdesk
**Roles:** Agent, Requester
**Status:** ⚠️ Documented — domain-portability proof only (Case 2's relationship to Case 1), no new capability, see `runtime/case-portfolio.md`

```
helpdesk-ticket.menata / .yaml    Ticket (one Machine, deliberately)
```

---

## Case 18 — HR Operations

**Domain:** HR — the Employee master data other apps reference
**Application:** HR
**Roles:** HR
**Status:** ⚠️ Documented — Employee master only; payroll flagged as domain-engine territory, see `runtime/case-portfolio.md`

```
hr-employee.menata / .yaml    Employee (one Machine, deliberately)
```

**Declared targets vs findings:**

| Target | Result |
|--------|--------|
| Manager self-reference | reinforces CAP-F13's tree option — second instance after Case 9's Chart of Account |
| Employee as master data | reinforces CAP-O02 — third instance after Case 10 and Currency |
| Payroll calculation | explicitly out of scope — domain-engine boundary, same as Case 9's posting derivation |

---

## Case 19 — Project Management (Trello-like)

**Domain:** Project management — Boards of Lists of Cards, manually reordered
**Application:** Project Management
**Roles:** Member
**Status:** ⚠️ Documented boundary test — first case needing user-editable manual ordering, see `runtime/case-portfolio.md`

```
pm-board.menata / .yaml    Board
pm-list.menata / .yaml     List (first CAP-V14 instance)
pm-card.menata / .yaml     Card (second CAP-V14 instance, plus cross-List move)
```

**Declared targets vs findings:**

| Target | Result |
|--------|--------|
| Manual/free ordering | **[UNTARGETED FINDING]** → registered as CAP-V14 — distinct from CAP-V04's declarative sort; the reorder action renumbers every sibling record |
| Card moving between Lists | reuses CAP-F13 + CAP-A13 |
| Checklist child table | reuses CAP-F16 |

---

## What the cases prove together

| Capability | Case 1 | Case 2 | Case 3 |
|------------|--------|--------|--------|
| Metadata-driven single machine | ✅ | ✅ | ✅ |
| Multiple applications in one workspace | ✅ | ✅ | ✅ |
| Conditional constraints | ✅ | — | ✅ (partial) |
| Role-based event permissions | ✅ | ✅ | ✅ (partial) |
| Cross-machine references | — | — | ⚠️ needs P1 |
| Sequential workflow logic | — | — | ⚠️ needs P3 |
| Parent-child status aggregation | — | — | ⚠️ needs P3 |
| Record-level ownership | — | — | ⚠️ needs P5 |
| System-triggered events | — | — | ⚠️ needs P6 |

Cases 1 and 2 validate the metadata-driven foundation.  
Case 3 defines the next layer of runtime capability needed to handle real-world workflow complexity.

---

## Adding a new case

1. Write the `.menata` source.
2. Write the `.yaml` Runtime Metadata (annotate `[NOT YET]` where applicable).
3. Translate to `seeds/00N_<name>.sql` for the parts that are supported.
4. `psql $DATABASE_URL -f seeds/00N_<name>.sql`
5. Restart the server.
