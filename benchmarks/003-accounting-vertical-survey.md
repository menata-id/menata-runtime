# Accounting Vertical Benchmark (Odoo / ERPNext)

> Study 6 deliverable (`../roadmap.md`).
>
> Deep vertical benchmark: accounting, tax, financial reporting, visualization —
> against Odoo Accounting and ERPNext (Frappe) accounting modules.
>
> Status: v0.2 — + World-Class Standards Addendum (GAAP chart-of-accounts convention, SOX internal controls) | Created: 2026-07-04 | Updated: 2026-07-10

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

# Case 9 — Accounting (target declaration, v1 — Study 6)

**Business reality:** Small-org bookkeeping — chart of accounts, manual journal entries, monthly close, trial balance.

**Declared targets:** CAP-F16 (entry lines), CAP-C10 (debit=credit), CAP-E06+R07 (post → immutable), CAP-C11 (closed period lock), CAP-F18 (entry numbering), CAP-V13 (trial balance report), CAP-A02 (posting date).

**Deliberately out of scope:** invoice posting derivation, reconciliation, multi-currency (F17) — kept for a later case so Case 9 stays a clean test of the *structural* accounting capabilities.

Full declaration and field-level design: `../case-portfolio.md`. Grounded in the World-Class Standards Addendum below.

---

# World-Class Standards Addendum (2026-07-10)

Study 6 benchmarked two *platforms* (Odoo, ERPNext). Before Case 9's fields are actually
designed, this addendum benchmarks the *standards* those platforms themselves are built to
comply with — GAAP chart-of-accounts convention and SOX internal-control requirements — so the
case's field set is checked against accounting practice, not just software precedent.

## Standard Chart of Accounts

| Concept | Standard | Menata-relevant pattern |
|---------|----------|--------------------------|
| Five account types | Asset, Liability, Equity, Revenue, Expense — the fixed top-level classification every COA uses | `value_list` — closed, small, universal (same shape as Study 15's closed-domain axis) |
| Numbering convention | First digit = category (1=Asset, 2=Liability, 3=Equity, 4=Revenue, 5=Expense); **not GAAP-mandated**, a widely-adopted convention only | Free-text `Account Code` — deliberately *not* constrained to match the type. Enforcing the prefix would be inventing a rule GAAP itself doesn't require |
| Normal balance | Deterministic from account type: Asset/Expense → Debit; Liability/Equity/Revenue → Credit | A **static category lookup**, never entered independently — new CAP-F14 sub-pattern (see below), not a second `value_list` field a metadata author could accidentally desync from Account Type |
| Hierarchy | Header/group accounts roll up leaf accounts (e.g. "Current Assets" groups "Cash", "Accounts Receivable") | CAP-F13 self-reference + rollup — already an option note on CAP-F13 from Study 6; Case 9 gives it its first case evidence |
| Leaf vs. group posting | Only leaf ("postable") accounts may receive journal lines; group accounts exist for rollup only | A record-level flag (`Is Group`), enforced as a constraint on Journal Entry Line's Account reference — composable from CAP-F13 + CAP-C05, not a new capability |

## SOX Internal Controls (journal entry lifecycle)

| Concept | Standard | Menata-relevant pattern |
|---------|----------|--------------------------|
| Segregation of duties | The person who **prepares** a journal entry must not be the same person who **posts** it — "no unchecked ability to prepare, approve, and post high-impact entries" | **CAP-P03** (Separation of Duties) — already registered since Study 1 mapping (spec 004 example), but had **no case evidence** until Case 9. This is the single biggest miss in Study 6's original declared-targets list |
| Audit trail | Every state change (prepare → post) must be recorded in tamper-proof form, with who/when, sufficient to reconstruct events | **CAP-R04** (event audit log) — registered since Case 1, ⚠️ partial (logged to DB, no UI). Not in Study 6's original declared list either; Case 9 is the case that actually needs it to matter (an accounting audit trail is not optional, unlike a facility maintenance log) |
| Approval workflow before posting | Preventive control: Draft must pass through an authorization step before becoming an immutable Posted record | Already covered by CAP-E06 + CAP-R07 (Study 6's own targets) — SOX just confirms this isn't optional polish, it's a compliance requirement |
| Access restriction | Only authorized roles may post; unposting/reversal requires elevated rights | CAP-P05 (CRUD-level permissions) — already registered (Study 2), reinforced here |

**Correction to Study 6's original declaration:** the seven originally declared targets
(F16, C10, E06+R07, C11, F18, V13, A02) covered the *structural* accounting capabilities well but
missed the **control** capabilities that make a bookkeeping system trustworthy, not just
functional. **CAP-P03 and CAP-R04 are added as declared targets in the v2 case design below** —
found by checking against the standard, exactly the failure mode the roadmap's dual-track method
exists to catch (a platform survey alone would not have surfaced this; SOX is a compliance
standard, not a platform feature).

## Deliberately still out of scope (unchanged from Study 6, reason restated)

| Concept | Reason |
|---------|--------|
| Multi-currency (CAP-F17) | Same architectural slot as Case 5's Rice/Cement UoM tiering — real, but doubles the case's dominant cluster. ISO 4217 currency codes are the standard to follow when this is picked up in a later case |
| Invoice posting derivation | Domain-engine territory (Study 6's own boundary finding) — a pluggable executor beneath declarative metadata, not a metadata concept |
| Reconciliation | Case 8 territory (external events + cross-record matching) |
| XBRL / statutory filing formats | A reporting *export* standard, not a structural or control concept — revisit only once CAP-V13 (aggregate report view) is implemented and a filing use case actually arises |

---

# Registry Impact

7 new capabilities from Study 6 (registry v0.4): CAP-F16, CAP-F17, CAP-F18, CAP-C10, CAP-C11, CAP-R07, CAP-V13. CAP-F13 gains a tree/hierarchy option note; CAP-F14 gains a derived-lines design requirement.

From this addendum: no new capability IDs — **CAP-P03 and CAP-R04 gain their first case evidence**
(previously spec-example/Case-1-incidental only), and **CAP-F14 gains a third sub-pattern** (static
categorical lookup — Account Type → Normal Balance — alongside Study 15's unit/currency-conversion
sub-pattern and Case 5's aggregate-rollup sub-pattern). See `../capability-registry.md`.

**Priority note:** CAP-F16 (line items) joins the reference field (CAP-F13) at the top of the structural queue — together they are what separates "form apps" from "document apps". Every ERP document type needs both.
