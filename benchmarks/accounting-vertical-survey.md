# Accounting Vertical Benchmark (Odoo / ERPNext)

> Study 6 deliverable (`../capability-roadmap.md`).
>
> Deep vertical benchmark: accounting, tax, financial reporting, visualization —
> against Odoo Accounting and ERPNext (Frappe) accounting modules.
>
> Status: v0.1 | Created: 2026-07-04

**Why accounting:** it is the hardest widely-standardized business vertical — centuries-old invariants (double-entry), legal immutability requirements, hierarchical reference data, and reporting that every organization needs. If Menata can express meaningful accounting as metadata, most verticals are easier.

---

# How Odoo and ERPNext structure accounting

| Concept | Odoo | ERPNext (Frappe) | Nature |
|---------|------|------------------|--------|
| Chart of Accounts | `account.account`, hierarchical via code/groups | `Account` DocType, **tree** (`is_group`, parent) | Hierarchical reference data |
| Journal Entry | `account.move` (header) + `account.move.line` (One2many) | `Journal Entry` + child table `Journal Entry Account` | **Header + line items in one document** |
| Double-entry rule | `move.line` debits = credits enforced on post | Validate: total_debit = total_credit | Aggregate invariant over lines |
| Posting lifecycle | draft → posted; posted = immutable (reset needs rights) | Draft → Submitted (docstatus 0→1); submitted immutable, cancel = amend | State-gated immutability |
| Fiscal period close | Lock dates (`fiscalyear_lock_date`) — no posting before lock | Period Closing Voucher + `Accounts Settings` lock | Temporal cross-record constraint |
| Tax | `account.tax` (percentage/fixed/group), auto tax lines on invoice | `Sales/Purchase Taxes and Charges Template` (rate rows, metadata!) | Declarative computation → derived lines |
| Document numbering | `ir.sequence` per journal | Naming Series (`ACC-JV-.YYYY.-.#####`) | Auto-numbering sequence |
| Multi-currency | amount_currency + rate + company-currency mirror | `Currency Exchange` + base-currency fields per row | Dual-amount money |
| Trial Balance / P&L / Balance Sheet | Report engine over move lines, grouped by account hierarchy, period compare | Script/Query Reports over GL Entry, tree rollup | Aggregate report views |
| General Ledger | Filtered, running-balance list of move lines | GL Entry report | Running-balance list |
| Reconciliation | `account.partial.reconcile` matching | Payment Reconciliation tool | Cross-record matching (→ Case 8) |
| Visualization | Graph views (bar/line/pie) on any model | Dashboard Charts (metadata-defined) | Chart views |

**Notable:** ERPNext defines tax templates, COA trees, naming series, and dashboard charts **as metadata records** — no code. The code boundary sits at the **posting engine**: deriving GL Entry rows from business documents (invoice → debtor/income/tax lines) is Python in both platforms.

---

# Gap analysis vs Menata registry

## Composable from already-registered capabilities

| Accounting need | Composition |
|-----------------|-------------|
| Account as machine, entries reference accounts | CAP-F13 (reference) |
| Draft → Posted lifecycle | CAP-E01 + CAP-E06 (state guards) |
| Posting date stamping | CAP-A02 (environment data) |
| Report drill-down navigation | CAP-V03 + CAP-F13 |
| Payment matching workflow | Case 8 territory (CAP-E04, C08) |

## New capabilities surfaced

| ID | Capability | Evidence | Why it matters |
|----|-----------|----------|----------------|
| **CAP-F16** | **Line items / child table inside a record** (header-detail as one document) | Odoo One2many, Frappe Table field — *universal to every business document with lines* (journal, invoice, PO, timesheet) | The single biggest structural gap after references. A journal entry is not N loose records — header + lines commit and post as one unit |
| CAP-F17 | Multi-currency money (transaction currency + rate + base mirror) | Both platforms | Any org dealing in >1 currency |
| CAP-F18 | Auto-numbering / document sequences (`ACC-JV-.YYYY.-.#####`) | `ir.sequence`, Naming Series — also universal beyond accounting | Legal document identity; Study 2 missed it |
| CAP-C10 | Aggregate constraint over line items (`sum(debit) = sum(credit)`) | Double-entry validation | First constraint class spanning *lines within one record* |
| CAP-C11 | Temporal period constraint (no posting into locked/closed period) | Lock dates, Period Closing Voucher | Cross-record + reference-data-driven constraint |
| CAP-R07 | Record immutability after state (posted/submitted docs frozen; amend-via-new-version) | docstatus model, posted moves | Stronger than CAP-E06: guards *edits*, not just events. Also needed by Case 6 (ledger) |
| CAP-V13 | Aggregate report view (group-by, hierarchy rollup, period comparison, running balance) | Trial Balance, P&L, Balance Sheet, GL | The report class every vertical eventually needs |
| — | Hierarchical (tree) reference data | Account tree, `is_group` | Recorded as an *option on CAP-F13* (self-reference + rollup), not a separate capability |
| — | Derived line generation (tax lines auto-added from rate templates) | Tax templates (metadata in ERPNext!) | Recorded as design requirement on CAP-F14 + CAP-A06 composition; revisit if composition proves insufficient |

---

# The boundary question (Study 6 key question)

> Where is the boundary between metadata-expressible accounting and a domain engine?

**Metadata-expressible** (both platforms prove it, ERPNext most explicitly):
- structures: COA tree, journal/entry-with-lines, tax rate templates, naming series
- invariants: required fields, debit=credit (declarative aggregate rule), period locks
- lifecycle: draft→posted with immutability
- presentation: report definitions (group-by + rollup + period compare), dashboard charts

**Domain-engine territory** (Python in both platforms):
1. **Posting derivation** — turning a business document (invoice) into balanced GL lines (debtor + income + tax rows). Rule-driven but multi-step and conditional.
2. **Reconciliation algorithms** — partial matching, FIFO application of payments.
3. **Currency revaluation, deferred revenue schedules** — generated record series over time.

**Menata position:** the boundary is *derivation complexity*. Single-step derivations (`set_field`, `create_record` with computed values) stay metadata. Multi-step conditional derivations (posting engine) are where metadata would degenerate into a programming language — contradicting the language's declarative principle. The honest answer: Menata should express accounting **documents, invariants, lifecycle, and reports** as metadata, and treat posting engines as a *pluggable runtime capability* (an executor extension registered per machine), not as metadata. This sharpens the Study 9 question: extension architecture must allow domain engines to plug in beneath declarative metadata.

---

# Case 9 — Accounting (target declaration)

**Business reality:** Small-org bookkeeping — chart of accounts, manual journal entries, monthly close, trial balance.

**Declared targets:** CAP-F16 (entry lines), CAP-C10 (debit=credit), CAP-E06+R07 (post → immutable), CAP-C11 (closed period lock), CAP-F18 (entry numbering), CAP-V13 (trial balance report), CAP-A02 (posting date).

**Deliberately out of scope:** invoice posting derivation, reconciliation, multi-currency (F17) — kept for a later case so Case 9 stays a clean test of the *structural* accounting capabilities.

Registered in `../case-portfolio.md`.

---

# Registry Impact

7 new capabilities (registry v0.4): CAP-F16, CAP-F17, CAP-F18, CAP-C10, CAP-C11, CAP-R07, CAP-V13. CAP-F13 gains a tree/hierarchy option note; CAP-F14 gains a derived-lines design requirement.

**Priority note:** CAP-F16 (line items) joins the reference field (CAP-F13) at the top of the structural queue — together they are what separates "form apps" from "document apps". Every ERP document type needs both.
