# Case Portfolio

> Artifact 3 of the Capability Roadmap — deliberate case selection.
>
> Cases are chosen to hit untested pattern clusters, not at random.
> Target patterns are declared **before** the case is written, so each case
> is a designed experiment — and surprises (patterns the case reveals that
> were not targeted) are themselves findings.
>
> Status: v0.31 — Case 3 gains a 2026-08-23 extension note (PDF signature placement, Study 32,
> `benchmarks/024-pdf-signature-approval-study.md`), new CAP-F22 registered. Previously v0.3 — full
> 21-case portfolio documented (Cases 1–10 original + Cases 11–21 Extended Portfolio); Cases 1–2 ✅
> done, the remaining 19 ⚠️ documented with targets/gaps registered against `capability-
> registry.md` | Created: 2026-07-04 | Updated: 2026-08-23

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
| 3 | Document Approval | Multi-instance workflow: sequence, synchronization, resource allocation | F13, A07, A08, X03, P02, E05 | ⚠️ documented, gaps registered — extended 2026-08-23, see note below |
| 4 | Maintenance Reminder | **Time-driven behavior**: schedules, escalation, environment data | E02, E03, A02, A09, A11 | ⚠️ documented (this study) |
| 5 | Inventory / Stock Movement | Calculation & multi-record transaction: quantity math, balance updates | F07, F13, F14, F16, F19, A02, A06, C05, C07, C08, X12 | ⚠️ documented, gaps registered |
| 6 | Petty Cash Ledger | Numeric aggregation & immutability: running balance, append-only, period close | F08, F13, F14, A02, C08, C10, P03, E06+R07, R04 | ⚠️ documented, gaps registered |
| 7 | Customer Complaint | Unstructured case management (CMMN-style): ad-hoc steps, SLA, escalation, reopen | E02, E05, A09, A11, A12(new), A13(new), P04, C09, V09, WCP-10 cycles | ⚠️ documented, gaps registered |
| 8 | Payment Confirmation | External events: webhook ingestion, idempotency, reconciliation | E04, X07, X13(new), A06, A13(new), X12, C08 | ⚠️ documented, gaps registered |
| 9 | Accounting (journal, monthly close, trial balance) | Vertical depth: header-detail documents, aggregate line invariants, immutability | F13, F14, F16, F18, A02, C10, C11, P03, R04, R07, E06, V13 | ⚠️ documented, gaps registered |
| 10 | Organization Composite | Emergent capabilities at composition: shared identity, master data, cross-app navigation, org-wide reporting | F13+I01–I05+X09+V10+P05 composition test | ⚠️ documented (Study 7) — 6 `[COMPOSITION FINDING]` → CAP-O01…O06 |

Sequencing follows the registry's implementation order: Case 4 (time) precedes Case 5–6 (calculation) because escalation and scheduling appear in Cases 6–7 too; external events (Case 8) come last because they depend on API surface (X07).

---

# Case 3 — extension note (2026-08-23): PDF signature placement

Case 3's original P1–P6 gaps are all ✅ (see `capability-registry.md`'s CAP-F13/A07/A08/X03/P02/E05
rows) — the workflow itself (sequential/parallel approval, per-step ownership) has been done since
2026-07-11. A new, real requirement extends it, requested directly by the owner: the approved
artifact must be an actual PDF, each approver's own signature **image** must be composited onto a
position on that PDF decided *before* submission (not a generic stamp), and approvers are drawn
from a work group rather than assigned one at a time.

Full write-up, page-by-page screen design, and gap analysis: `benchmarks/024-pdf-signature-
approval-study.md` (Study 32). Headline findings, in brief: a genuinely new capability is needed
(**CAP-F22**, binary PDF signature compositing — neither CAP-F06's plain file storage nor
CAP-F21's HTML-only template render covers opening an existing uploaded PDF and editing it),
registered ❌ Proposed; coordinate storage and the per-user signature image need no new
capability, pure composition from CAP-F07/CAP-F05/CAP-F06/CAP-F13, same precedent as CAP-F19/F20;
and the group-based approver picker leans on **CAP-O07** (Groups/Teams), which remains ❌ and
deliberately deferred — not "in development" as initially assumed, though this case is real new
pressure toward building it eventually. Four new mockup screens were added to the existing
Menata Apps Builder design canvas (Study 29/30/31's artifact), matching its established visual
system with no new chrome introduced.

**Correction (2026-08-23, later the same day)**: CAP-O07 moved ❌→✅ the same day, implemented and
conformance-proven (T194–T205) in a separate concurrent session — see `capability-registry.md`'s
CAP-O07 row. `SignaturePlacement.dc.html`'s group-sourced approver list can now be built against
the real `groups`/`group_members` mechanism directly, not just a future one.

**Same-day views-configurability check**: two of the four screens are already fully declarable
via today's `views` metadata (plain `form`, and `detail` over an ordinary `file` field once
CAP-F22 exists); the other two are not, registering two more previously-untracked View-type
gaps — **CAP-V20** (sequential decision stepper — a Study 29 design sketch that had never been
given a registry row until now) and **CAP-V21** (coordinate-placement editor, the signature-pin
screen itself). Full reasoning in Study 32 §5.

**CAP-F22 implemented 2026-08-29** — conformance T206–T208, full suite 208/208, zero regressions.
The "Final Signed Document" screen named above (✅ once CAP-F22 exists) is now real: a Sequential
Step's Approve composites the acting approver's own registered Signature image onto the Document's
current file (the original upload on the first approval, the previous approver's own output on
every one after) at the coordinates declared on that Step. Full build notes:
`capability-registry.md`'s CAP-F22 row. CAP-V20/CAP-V21 (this note's own two screens above) remain
unbuilt — placement is set via plain `number` fields for now, not a drag-to-place UI.

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
| `Item Unit Conversion` — own Object, back-reference to Item (Tier 2: Cement — BOX/DOZEN/PCS) | CAP-F16 | Line items — proves Study 15's Quantity Tier 2 |
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
`inventory-item-unit-conversion.{menata,yaml}`,
`inventory-stock-movement.{menata,yaml}`, `inventory-stock-ledger.{menata,yaml}`
(four Machines, one file pair each — same convention as Case 3's
`approval-document` / `approval-step`)

---

# Case 6 — Petty Cash Ledger (target declaration)

**Business reality:** A small cash box run as an imprest fund — a fixed float, one accountable
Custodian, every expense recorded against the running balance, and a periodic reconciliation
performed by someone *other than* the Custodian before the period closes and freezes.

**External grounding:** the imprest-fund control pattern (fixed float, single custodian,
independent reconciliation) — real accounting practice, not a platform convention.

**Declared targets:**

| Target | Capability | Pattern |
|--------|-----------|---------|
| `Fixed Imprest Amount`, `Amount`, `Cash Counted` as `money` | CAP-F08 | First real case evidence for `money` (previously schema-doc only) |
| `Fund` reference on Voucher / Period | CAP-F13 | `reference` |
| `Current Balance` = Imprest Amount − open vouchers | CAP-F14 | Aggregate-rollup sub-pattern (same shape as Case 5's Stock On Hand) |
| `Recorded By`, stamped at Record | CAP-A02 | WDP-7 Environment Data |
| `Voucher Amount <= Fund Current Balance` | CAP-C08 | Cross-record constraint — third case instance |
| `Cash Counted + sum(Vouchers) = Fixed Imprest Amount` | CAP-C10 | Aggregate line constraint — reconciliation-formula variant of Case 9's debit=credit |
| `Reconciled By != Fund Custodian` | CAP-P03 | Separation of duties — third case instance (independent-audit control, not approval) |
| Closed period frozen | CAP-E06 + CAP-R07 | State-conditional availability + immutability-after-state |
| Voucher/reconciliation trail | CAP-R04 | Audit trail |

Files: `prototype/go/docs/examples/pettycash-fund.{menata,yaml}`,
`pettycash-voucher.{menata,yaml}`, `pettycash-period.{menata,yaml}`

---

# Case 7 — Customer Complaint (target declaration)

**Business reality:** Complaints arrive, get triaged, may need any number of ad-hoc investigation
steps in no fixed order, have a priority-driven SLA, auto-escalate to a supervisor on breach, and
can be reopened by the customer after resolution.

**External grounding:** CMMN (Case Management Model and Notation, OMG standard) — Case File Item,
discretionary Task, Stage, Milestone, Sentry. The question this case exists to answer: can Menata
express work with no predefined step sequence?

**Declared targets:**

| Target | Capability | Pattern |
|--------|-----------|---------|
| No `activate_next` anywhere — any permitted event fires in any order, gated by Status | — | **The CMMN boundary finding**, not a capability: Menata's flat `When X` + CAP-E06 expresses CMMN's *bounded* flexibility (many predefined paths, no fixed Sequence) but not its *unbounded* flexibility (a case worker inventing a new task type at runtime) — stated explicitly, not a gap. **Re-examined in full against every CMMN construct and all 21 portfolio cases in `benchmarks/014-cmmn-case-management-benchmark.md` (Study 22):** this case's own actual targets turn out to be entirely within CMMN's *bounded* half (fixed permitted Events, order-free) — the line above was correct in spirit but overstated as applied to this case specifically |
| `SLA Due Date` set by Priority at Triage | CAP-A11 | Date arithmetic — priority-keyed offset, a new sub-pattern |
| `Every Day 08:00` + compound condition → auto-`Escalate` | CAP-E02 + CAP-A09 + CAP-E05 | Time-driven event, compound condition, system-triggered (same-record self-trigger, a new CAP-E05 sub-pattern) |
| `Priority` raised one level on Escalate | **CAP-A12 (new)** | Ordinal/enum stepping in actions |
| `Delegate`: `Delegated By` = previous Assigned To | **CAP-P04 (first case evidence)** | WRP Delegation — previously "not yet in language examples" |
| `Reopen Count + 1`, only reachable from Resolved | CAP-A11 (numeric sibling) + CAP-E06 | WCP-10 (already ✅) proven in a richer flow than Case 1's rework loop |
| `Resolution Notes` required only at Resolve | CAP-C09 | Constraints evaluated on event trigger |
| Overdue Complaints (compound filter) | CAP-V09 | Declarative view-level filter |

Files: `prototype/go/docs/examples/complaint.{menata,yaml}` (one Machine — see the CMMN finding
above for why this case doesn't need a child Machine per step, unlike Case 3 or Case 9)

---

# Case 8 — Payment Confirmation (target declaration)

**Business reality:** A customer pays via bank transfer or payment gateway; a webhook confirms it;
the same webhook delivered twice (every provider delivers at-least-once) must not double-apply;
the matching Invoice updates exactly once; unmatched payments queue for manual reconciliation.

**External grounding:** webhook idempotency convention (Stripe/Shopify/GitHub) — dedupe by the
provider's own event ID, atomic check-and-claim (never check-then-act), return success for
duplicates, "receive fast, process safe" (raw event ingestion separated from domain processing).

**Declared targets:**

| Target | Capability | Pattern |
|--------|-----------|---------|
| `Payment Webhook Event.Receive` fired by an inbound call | CAP-E04 | First real case evidence for external events |
| The webhook receiving surface itself | CAP-X07 (clarified) | Inbound third-party surface — distinct from X07's outbound auto-generated CRUD API; both needed |
| Duplicate `Provider Event ID` → halt, return success | **CAP-X13 (new)** | Idempotent external-event ingestion |
| `Receive` → `create_record` into Payment | CAP-A06 | WCP-13/14 MI |
| `Reconcile` writes `Invoice.Amount Paid` / `Invoice.Status` | **CAP-A13 (new)** | Cross-record field write — distinct from `create_record` and `aggregate_status` |
| Reconcile's write chain commits as one unit | CAP-X12 | Reinforced — cross-record write atomicity |
| Payment↔Invoice matching | CAP-C08 | **Deliberately manual, not automatic** — correlation matching is a different sub-pattern than Case 5/6/9's fixed comparisons; kept out of scope so Case 8 stays a clean test of idempotency, not fuzzy matching |

Files: `prototype/go/docs/examples/payment-invoice.{menata,yaml}`,
`payment-webhook-event.{menata,yaml}`, `payment.{menata,yaml}`

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
| `Journal Entry Line` — own Object, back-reference to Journal Entry (Account, Debit, Credit, Memo) | CAP-F16 | Line items — the actual line-level facts a Trial Balance reports over |
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

# Extended Portfolio (Cases 11–21)

The original 10-case portfolio (Study 3) targeted untested *pattern clusters*. This extension
targets untested *business verticals*, screened first against the registry to avoid writing a case
that only re-proves what an earlier case already proved (Rule 3: business realism, not synthetic
coverage). Each row states its novelty honestly — several are deliberately light-touch because
their capability cluster already has case evidence.

| # | Case | Novelty vs. Cases 1–10 | Primary targets (CAP) | Status |
|---|------|------------------------|----------------------|--------|
| 11 | Social App (Instagram-like) | **High** — many-to-many relationships, feed | F20(new), C12(new), F14, V05 | ⚠️ documented |
| 12 | Community Site | Medium — builds on 11 + gamification | F20, C12, A14(new) | ⚠️ documented |
| 13 | Blog / One-Page Site | **High** — public/unauthenticated access | P07(new), V10, F03 scope note | ⚠️ documented |
| 14 | Lending Services | **High** — schedule generation | A15(new), F13, A02, A06, P03, E02 | ⚠️ documented |
| 15 | E-commerce | Medium — cart as mutable pre-commit doc | R08(new), composes Case 5/8/9 | ⚠️ documented |
| 16 | Point of Sale | Low — composition of Case 5+8+15 | composes only, no new CAP | ⚠️ documented |
| 17 | Helpdesk | Low — re-proves Case 7, domain-portability only | composes Case 7, no new CAP | ⚠️ documented |
| 18 | HR Operations | Low — Employee master only; payroll flagged domain-engine | F13 tree (2nd), O02 (3rd) | ⚠️ documented |
| 19 | Project Management (Trello-like) | Medium — manual ordering | V14(new), F13, F16 | ⚠️ documented |
| 20 | Hospital System | Medium — scheduling + compliance weight | V07 (1st evidence), P06 (1st evidence), F16 | ⚠️ documented |
| 21 | E-learning | Medium — sequential unlock + certificates | F21(new), F20, C12, E06 (reused) | ⚠️ documented |

## Case 11 — Social App / Instagram-like (target declaration)

**Business reality:** Members post photos with captions; other members like and comment; members
follow each other and see a feed of posts from people they follow.

**Declared targets:**

| Target | Capability | Pattern |
|--------|-----------|---------|
| `Follow` / `Like` join Machines | **CAP-F20 (new)** | Many-to-many relationship — neither CAP-F13 (one-directional) nor CAP-F16 (parent-owned) names this |
| `(Follower, Followee)` / `(User, Post)` must be unique | **CAP-C12 (new)** | Composite uniqueness constraint — also retroactively explains an assumption every prior case made silently for single-field uniqueness (Account Code, Entry Number, ...) |
| `Post.Like Count` / `Comment Count` | CAP-F14 | A maintained-counter sub-pattern, distinct from Case 5/9's read-time aggregate rollup — this case surfaces the open design question rather than resolving it |
| Feed = posts by anyone I follow | CAP-V05 (extended) | A two-hop relationship-filtered list, not a direct-field "my records" match |

Files: `prototype/go/docs/examples/social-post.{menata,yaml}`, `social-follow.{menata,yaml}`,
`social-like.{menata,yaml}`, `social-comment.{menata,yaml}` (four Machines, one file pair each)

## Case 12 — Community Site (target declaration)

**Business reality:** Members join Groups, Groups host Events, members post within a Group, and
earn points for participation that automatically unlock badges once a threshold is crossed.

**Declared targets:**

| Target | Capability | Pattern |
|--------|-----------|---------|
| `Membership` join Machine | CAP-F20 | Third case instance (after Case 11's Follow, Like) |
| `(Group, Member)` / `(Member, Badge)` uniqueness | CAP-C12 | Second and third instance |
| Badge auto-awarded when `sum(points) >= 100` | **CAP-A14 (new)** | Aggregate-conditioned action — distinct from CAP-A09 (single-field condition) and CAP-C10 (aggregate constraint blocks a write; this triggers one) |

**Deliberately not written as its own Machine:** RSVP (Event attendance) — a fourth instance of
the same CAP-F20 shape already proven three times; composable without new design effort.

Files: `prototype/go/docs/examples/community-group.{menata,yaml}`,
`community-membership.{menata,yaml}`, `community-event.{menata,yaml}`,
`community-points.{menata,yaml}`, `community-badge-award.{menata,yaml}` (five Machines)

**Audited 2026-07-13** (`benchmarks/010-gamification-flow-audit.md`): this case's own metadata
was never seeded/run (target-declaration only, per this section's own header), and re-reading it
in full surfaced an un-flagged gap — `Point Ledger Entry.Reason` names 4 point-earning reasons but
only 1 (`Joined Group`) has a real triggering event wired; `Posted Status` has no `Post` Machine at
all, `Hosted Event`/`Attended Event` have no wiring on `Event` (RSVP was deferred and never
composed). The study also found this case's `create_record`-based wiring (CAP-A06, publisher
knows its subscriber) contradicts CAP-I05's own stated rationale for gamification (decoupled
`event_subscriptions`, proven instead by `seeds/014_integration_lab.sql`). No unified,
conformance-tested proof of the full action→points→threshold→badge→display chain exists anywhere
in this repo — see the study for the full inventory and recommended next-session task.

## Case 13 — Blog / One-Page Site (target declaration)

**Business reality:** A public blog. Anyone, logged in or not, reads Published posts and leaves
comments; only an authenticated Author writes and moderates.

**Declared targets:**

| Target | Capability | Pattern |
|--------|-----------|---------|
| `Visitor` role reads Published Posts / submits Comments with no login | **CAP-P07 (new)** | Public/unauthenticated access — every prior case assumed CAP-X02 authentication already succeeded |
| One-page landing composing multiple content sections | CAP-V10 | Reinforced — same composition shape as Portal GA's dashboards, applied publicly. Does **not** close the registry's separate Page/Theme "not yet studied" gaps |
| `Tags` wants multi-select | CAP-F03 (scope note) | `value_list` is single-select only; worked around with comma-separated text |

Files: `prototype/go/docs/examples/blog-post.{menata,yaml}`, `blog-comment.{menata,yaml}`

## Case 14 — Lending Services (target declaration)

**Business reality:** A borrower applies for a loan; a Loan Officer other than the borrower
approves it; on disbursement, a full monthly repayment schedule is generated at once; repayments
are recorded against installments; overdue installments are flagged automatically.

**Declared targets:**

| Target | Capability | Pattern |
|--------|-----------|---------|
| `Disburse` generates Term Months' worth of schedule entries from one formula | **CAP-A15 (new)** | Batch/series record generation — distinct from `create_record` (one record) and date arithmetic (one value) |
| `Approved By != Borrower` | CAP-P03 | Fourth case instance |
| `Repayment.Record` writes `Schedule Entry.Status` | CAP-A13 | Reused from Case 8 |
| Overdue installment check | CAP-E02 + CAP-A09 | Same shape as Case 4's Overdue Tasks |

Files: `prototype/go/docs/examples/lending-loan-application.{menata,yaml}`,
`lending-loan.{menata,yaml}`, `lending-repayment-schedule-entry.{menata,yaml}`,
`lending-repayment.{menata,yaml}` (four Machines)

## Case 15 — E-commerce (target declaration)

**Business reality:** Customers browse Products, add them to a Cart, and Checkout converts the
Cart into a real Order. Payment reuses Case 8's Payment machine — not rebuilt.

**Declared targets:**

| Target | Capability | Pattern |
|--------|-----------|---------|
| `Cart` freely edited, no invariants until Checkout | **CAP-R08 (new)** | Editable scratch state — the opposite end of CAP-R07's spectrum (unconstrained *before* a state, not frozen *after* one) |
| `Order` header + `Order Line` items | CAP-F16 | Same shape as Case 9's Journal Entry, including its reporting-independence note |
| Product ~ Item, Payment ~ Case 8's Payment | — | Deliberate composition, not re-derived |

Files: `prototype/go/docs/examples/ecommerce-product.{menata,yaml}`,
`ecommerce-cart.{menata,yaml}`, `ecommerce-cart-item.{menata,yaml}`,
`ecommerce-order.{menata,yaml}`, `ecommerce-order-line.{menata,yaml}` (five Machines)

## Case 16 — Point of Sale (target declaration)

**Business reality:** A cashier rings up line items and takes payment in one motion — no cart
lingering, no webhook delay.

**Declared targets:** pure composition of Case 5 (stock deduction), Case 8 (payment, collapsed
into one event since there's no async webhook), Case 15 (line-item shape). No new capability
expected or found — written to confirm the composition actually holds, not to discover anything.

Files: `prototype/go/docs/examples/pos-sale.{menata,yaml}`, `pos-sale-line.{menata,yaml}` (two Machines)

## Case 17 — Helpdesk (target declaration)

**Business reality:** Internal IT support tickets — same discretionary-task, SLA-timed,
reopenable shape as Case 7's Complaint, aimed at employees instead of customers.

**Declared targets:** domain-portability proof only (same relationship Case 2 had to Case 1) —
CAP-E02/A11 (SLA), CAP-E06 (Reopen guard), CAP-C09 (Resolution Notes at event time), WCP-10. No
new capability expected or found.

Files: `prototype/go/docs/examples/helpdesk-ticket.{menata,yaml}` (one Machine)

## Case 18 — HR Operations (target declaration)

**Business reality:** Case 2 (Leave Request) already is an HR process. What's missing is the
Employee master itself, which Leave Request, Helpdesk, and every future app need to reference.

**Declared targets:** `Manager` self-reference (CAP-F13 tree option, second instance after Case
9's Chart of Account), `Employee` as a cross-app master-data candidate (CAP-O02, third instance).
**Deliberately out of scope:** payroll calculation — same domain-engine boundary Study 6 drew for
posting derivation, not a metadata concept.

Files: `prototype/go/docs/examples/hr-employee.{menata,yaml}` (one Machine)

## Case 19 — Project Management / Trello-like (target declaration)

**Business reality:** Boards contain Lists (columns); Lists contain Cards; both Lists and Cards
are freely reordered by drag-and-drop, and Cards move between Lists.

**Declared targets:**

| Target | Capability | Pattern |
|--------|-----------|---------|
| `List.Reorder`, `Card.Move` — user-set position, no formula | **CAP-V14 (new)** | Manual/free ordering — distinct from CAP-V04's declarative `default_sort` |
| Renumbering every sibling on reorder | CAP-V14 | Batch update shaped like CAP-A15, rewriting existing records instead of creating new ones |
| `Card.Move` changing List | CAP-F13 + CAP-A13 | Reused |
| `Checklist Item` — own Object, back-reference to Card | CAP-F16 | Same shape as every prior case |

Files: `prototype/go/docs/examples/pm-board.{menata,yaml}`, `pm-list.{menata,yaml}`,
`pm-card.{menata,yaml}`, `pm-checklist-item.{menata,yaml}` (four Machines)

## Case 20 — Hospital System (target declaration)

**Business reality:** Patients are scheduled for Appointments; each visit produces a Medical
Record with sensitive clinical notes visible only to the treating clinician. Clinical decision
support (drug interactions, dosage limits) is deliberately out of scope.

**Declared targets:**

| Target | Capability | Pattern |
|--------|-----------|---------|
| Doctor Calendar | **CAP-V07 (first real case evidence)** | A flat filtered list cannot serve "what does Dr. X's Tuesday look like" |
| `Notes` visible only to the treating Clinician | **CAP-P06 (first real case evidence)** | HIPAA-equivalent weight, same class as Case 9's SOX findings |
| `Prescription` — own Object, back-reference to Medical Record | CAP-F16 | Same shape as Case 9's Journal Entry Lines |

**Deliberately out of scope:** clinical decision rules (drug interaction checks, dosage limits) —
same domain-engine boundary Study 6 drew for posting derivation, Case 18 drew for payroll.

Files: `prototype/go/docs/examples/hospital-patient.{menata,yaml}`,
`hospital-appointment.{menata,yaml}`, `hospital-medical-record.{menata,yaml}`,
`hospital-prescription.{menata,yaml}` (four Machines)

## Case 21 — E-learning (target declaration, closes the Extended Portfolio)

**Business reality:** Students enroll in Courses, progress through sequentially-unlocked Lessons,
and receive a rendered Certificate once complete.

**Declared targets:**

| Target | Capability | Pattern |
|--------|-----------|---------|
| `Certificate.Generated File` rendered from a template | **CAP-F21 (new)** | The reverse direction of CAP-F06 — rendering a file at runtime, not storing an upload |
| `Enrollment` join Machine | CAP-F20 + CAP-C12 | Fourth instance of both |
| Sequential Lesson unlock | CAP-E06 | Reused, no new capability |

Files: `prototype/go/docs/examples/elearning-course.{menata,yaml}`,
`elearning-lesson.{menata,yaml}`,
`elearning-enrollment.{menata,yaml}`, `elearning-certificate.{menata,yaml}` (four Machines)

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
