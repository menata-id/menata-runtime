# Capability Registry

> Artifact 1 of the Capability Roadmap — the single source of record for runtime capabilities.
>
> One row per capability. The registry only grows (ratchet):
> a ✅ capability must never regress — its conformance test guards it.
>
> Status: v0.25 — + Case 21 (CAP-F21 templated document generation, new; CAP-F20/CAP-C12 fourth instance) — Extended Portfolio (Cases 11–21) complete | Updated: 2026-07-10
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
| CAP-F03 | `value_list` field (select + badge) | ✅ | Case 1 | — | | conformance T06–T08. **Scope note (Case 13):** single-select only — a multi-value variant (e.g. Blog Post Tags) has no representation; worked around today with a comma-separated `text` field, not a real capability |
| CAP-F04 | `date` field | ✅ | Case 1 | — | | conformance T05/T08 |
| CAP-F05 | `user` field | ⚠️ | Case 1 | — | 8 | renders as free text; no user picker, no identity link. Long-term: sugar over CAP-F13 + CAP-O01 (Study 15), not a permanently separate type — kept distinct only until CAP-O01 exists |
| CAP-F06 | `file` field | ⚠️ | Case 1, reinforced by Study 5 (Portal GA `NativeCompressedUpload*`) and a direct follow-up question | Frappe Attach→File DocType, Salesforce File/ContentDocument, Drupal file entity | 9 | input renders; upload is not stored. Long-term (Study 15): sugar over CAP-F13 + a runtime-managed File/Document entity — files have their own identity/lifecycle (storage key, versioning), same shape of gap as CAP-F05 waiting on CAP-O01. **Scope refined (sixth-pass):** not a new capability, but `type: file` gains `options` for image handling — `accept`, `compress`, `max_dimension`, `format` — with a dual-path contract: client-side compression is the fast path (saves upload bandwidth), but the server-side `Store` step never trusts it happened and re-applies the same pipeline whenever an incoming file doesn't already satisfy the declared policy. Matches Portal GA's proven pattern (client WebP compression via Web Worker + server-side fallback for unsupported browsers) and the same "client is advisory, server enforces" principle already applied to Constraints (CAP-C09). |
| CAP-F07 | `number` field | ⚠️ | schema doc | — | 10 | falls back to text input; no numeric validation |
| CAP-F08 | `money` field | ⚠️ | schema doc, first real case evidence from Case 6 (Fixed Imprest Amount) and Case 8 (Invoice/Payment amounts) | — | 10 | same fallback. **Number-vs-money boundary note:** Case 9's Journal Entry Line Debit/Credit Amount fields used plain `number`, not `money` — defensible under Case 9's deliberate single-currency, structural-only scope, but worth a retrofit pass once CAP-F08 is implemented, since real monetary amounts should carry currency/precision semantics even without CAP-F17 multi-currency |
| CAP-F09 | `boolean` field | ⚠️ | schema doc | — | 10 | same fallback |
| CAP-F10 | `time` / `date_time` / `duration` fields | ⚠️ | schema doc | — | 10 | same fallback |
| CAP-F13 | `reference` field (link to another machine) | ✅ | Case 3 (P1) | WCP-2/13/14, WRP-3 | | conformance T13–T16. **Implemented 2026-07-11** (target flavor (a), workspace-authored Machine): `options.target_machine` (renamed from `machine_id` to match `runtime-metadata-schema.md`/`capability-lifecycle.md`'s own worked example); loader rejects dangling `target_machine` at load time (`internal/metadata/loader.go`, "Unknown = explicit"); referential integrity enforced at Create (`internal/store.RecordStore.Exists`, `internal/handler.referenceViolations`) same tier as a required-field violation; form renders a picker, detail/list render a link — display label is a prototype heuristic (target Machine's `text` field named "Name", else its first `text` field, else the record id) pending a real "display field" designation, not yet a capability of its own. Self-reference/tree option proven by Case 18's Employee↔Manager (seeds/003_hr_employee.sql). **Deferred, not done here:** target flavor (b) — a reserved built-in identity target so `user`/`money`/`file` can become real reference sugar (CAP-F05/CAP-F06/CAP-F17/CAP-O01) — still needs its own pass; CAP-V06 (child records sub-list on parent detail) also still depends on this and is not yet built. Fourth-pass (Study 15): this alone (plus an ordinary Machine) fully fixes a single-application mis-modeled field (e.g. `Equipment`) — CAP-O02 is a separate, additional capability only for the cross-application case |
| CAP-F14 | Computed / formula field | ❌ | Study 2 survey | Salesforce formula, Frappe | 13 | design req: derived line generation (tax templates, Study 6). Study 15 (third-pass): this is the correct home for unit/currency conversion (amount × factor → normalized value) — NOT the Constraint grammar. **Three confirmed sub-patterns:** (a) unit/currency conversion (Study 15), (b) cross-record aggregate rollup (Case 5's Stock On Hand), (c) static categorical lookup (Case 9's Normal Balance derived from Account Type — GAAP standard, deterministic table, never independently entered) |
| CAP-F15 | Field default values (beyond status first-value) | ⚠️ | Study 2 survey | universal (6/6 platforms) | 8 | status default works; other fields have none |
| CAP-F16 | Line items / child table inside a record (header-detail document) — a Runtime storage/query capability: efficiently realizing a parent-owned one-to-many relationship (scoped, atomic-with-parent writes; optionally independently queryable for reporting) | ❌ | Study 6 accounting, reconfirmed by Case 5 and Case 9 | Odoo One2many, Frappe Table — universal to document apps | **3** | joins CAP-F13 atop the structural queue. **Language-layer note corrected (2026-07-11):** an earlier version of this note claimed no Menata Language grammar existed for a child-table field, and several example `.menata` files wrote it provisionally as one Fields line (`Table of (...)`). Re-examined: the Language grammar already exists — a parent-owned repeating fact is just an ordinary Object with a Field that references back to its parent (`001-object.md` §Relationships: "no separate relationship definition is required"), exactly as `accounting-journal-entry-line.menata` already modeled correctly. All eight `Table of (...)` usages were split into standalone Objects (`inventory-item-unit-conversion.menata`, `ecommerce-order-line.menata`, `ecommerce-cart-item.menata`, `elearning-lesson.menata`, `hospital-prescription.menata`, `pos-sale-line.menata`, `pm-checklist-item.menata`, plus removing the redundant field from `accounting-journal-entry.menata`). CAP-F16 itself is unaffected by this correction and remains a real ❌ Runtime capability — *how* Machine Interpretation chooses to physically store and query that already-expressible relationship (a real scoped child table vs. an ordinary independent table) is exactly the Machine Interpretation concern this capability tracks. See `roadmap.md`'s dated correction note. **Reporting-independence note (Case 9):** unlike Case 5's Item Unit Conversion (a pure per-parent lookup table, never queried standalone), Journal Entry Line rows must be independently queryable *across* parent documents for CAP-V13 (Trial Balance groups lines by Account across every Journal Entry) — the child-table target Machine needs its own queryable identity even though it is only ever authored atomically with its parent |
| CAP-F17 | Multi-currency money (transaction currency + rate + base mirror) | ❌ | Study 6 accounting + Study 15 (independent derivation) | Odoo/ERPNext; spec 001-object.md names Currency as an Object example | 14 | Study 15 reclassified `money` from primitive to reference sugar — Currency fails identity/lifecycle/reuse/cardinality tests, is a CAP-O02 master-data candidate |
| CAP-F18 | Auto-numbering / document sequences | ❌ | Study 6 accounting | ir.sequence, Naming Series — universal | 7 | Study 2 missed it |
| CAP-F19 | Quantity / unit-of-measure conversion pattern — **not a new field type**; composed from `number` (CAP-F07) + `value_list` (CAP-F03) for the unit label, escalating only when cardinality demands it: Tier 1 flat factor pair · Tier 2 child table (CAP-F16) · Tier 3 history-tracked Machine (CAP-F13) | ❌ | Study 15 (prediction) + Case 5 (case evidence) | SAP/Odoo UoM categories; APICS UoM conversion | 14 | Study 15's Quantity prediction, now cleared to registered status by dual evidence. Unlike CAP-F17 (`money`, genuine reference sugar), Quantity resolves through composition, never a dedicated `type: quantity` — Tier 1/2 both exercised by Case 5 (Rice, Cement), Tier 3 deliberately not exercised — no case has evidenced factor history yet |
| CAP-F20 | Many-to-many relationship (join/junction Machine — both sides independently queryable, neither side "owns" the row) | ❌ | Case 11 [UNTARGETED FINDING] — `Follow`, `Like`; reinforced by Case 12 (`Membership`) and Case 21 (`Enrollment`, fourth instance) | RDBMS junction table; Django/Rails `ManyToManyField` | 5 | Distinct from both CAP-F13 (one-directional reference: a record points to exactly one other) and CAP-F16 (parent-owned child table: a row belongs to exactly one parent, no independent identity). No case before Case 11 needed a relationship with two independent reference sides and no ownership direction |
| CAP-F21 | Templated document generation (render a file — PDF, image — from a template + record data, at runtime) | ❌ | Case 21 [UNTARGETED FINDING] — `Enrollment.Complete` issuing a `Certificate.Generated File` | certificate/invoice-PDF generators (every LMS, every invoicing platform) | 10 | The reverse direction of CAP-F06 (`file`), which only stores a file a *user* uploads. Needs a template (layout + merge fields) and a render step (template + this record's data → file); once rendered, storage reuses CAP-F06's existing identity/lifecycle shape (Study 15) — only the rendering step itself is new |

## Event Sources

| ID | Capability | Status | Discovered by | Pattern ref | Prio | Proof |
|----|-----------|--------|---------------|-------------|------|-------|
| CAP-E01 | Business activity event (`When X`) | ✅ | Case 1 | WCP-5/10 | | conformance T10/T12 |
| CAP-E02 | Time-driven event (`Every Day 08:00`) | ❌ | spec 003 + mapping | escalation (WRP); Study 15 boundary check confirms placement against iCalendar RRULE (RFC 5545) | 7 | recurring schedules are Event/Action grammar, never a Field concern — confirmed, not just assumed |
| CAP-E03 | Date-driven event (`When Due Date - 1 Day`) | ❌ | spec 003 + mapping | — | 11 | — |
| CAP-E04 | External event (webhook, payment) | ❌ | spec 003 + mapping, first real case evidence from Case 8 (payment webhook) | — | 12 | Case 8 also needs CAP-X07-adjacent *inbound* surface (receiving a third-party POST), distinct from CAP-X07's auto-generated outbound CRUD API — both apply, neither subsumes the other |
| CAP-E05 | Internal / system-triggered event | ❌ | Case 3 (P6), second instance in Case 7 (SLA breach auto-escalation) | WRP-11 | 6 | Case 7's flavor is a *same-record* self-trigger (an event firing another event on the record that owns it) — distinct from Case 3's `aggregate_status` (firing an event on a *different*, parent record). Worth splitting into two Proof paths if implementation reveals they need different mechanisms |
| CAP-E06 | **State-conditional event availability** (event allowed only in given status) | ✅ | **Study 1 mapping** | **WCP-18 Milestone, WCP-16** | | conformance T17–T18. **Implemented 2026-07-11:** `events.condition` (JSONB, migrations/003, reuses the same expression shape as `constraints.condition`) — realizes the `if` guard Menata Language's Event grammar already allows (`specification/003-event.md` §Conditions), the runtime just never evaluated it on `When`-triggered events before. `internal/handler.TriggerEvent` checks the condition against the record's pre-event data; a failing guard returns 400, not silently a no-op. Proven on Leave Request (`seeds/002`): Submit only from Draft, Approve/Reject/Cancel only from Submitted — T18 reproduces the exact headline bug this capability was named for (an Approved record could still be Rejected) and confirms it's fixed. |

> CAP-E06 is the headline finding of Study 1: today an Approved record can still be
> Rejected — events are filtered by role only, never by current state. No case had
> surfaced this; the benchmark predicted it.

## Actions

| ID | Capability | Status | Discovered by | Pattern ref | Prio | Proof |
|----|-----------|--------|---------------|-------------|------|-------|
| CAP-A01 | `set_field` with static value | ✅ | Case 1 | — | | conformance T10 |
| CAP-A02 | `set_field` with dynamic value (`now`, `today`, `current_user`) | ❌ | Case 3 (P2) | WDP-7 Environment Data | 3 | Case 7 (Delegate) surfaces a narrower, related gap: reading a field's *own value immediately before this event's actions mutate it* (`previous(field)`) — not covered by `now`/`today`/`current_user`, which only read environment state, never the record's own pre-mutation value. Note here rather than a new CAP until a second case needs it |
| CAP-A03 | `notify` to role | ⚠️ | Case 1 | — | 5 | slog only — no real delivery channel |
| CAP-A04 | `notify` to dynamic recipient (record's approver/submitter) | ❌ | Case 3 | WRP | 5 | — |
| CAP-A06 | `create_record` in another machine | ❌ | schema doc | WCP-13/14 MI | 13 | — |
| CAP-A07 | `activate_next` (sequential step activation) | ❌ | Case 3 (P3) | WCP-1 Sequence | 4 | — |
| CAP-A08 | `aggregate_status` (parent rollup: all-approved / any-rejected / cancel cascade) | ❌ | Case 3 (P3) | WCP-3/9/19/20 | 4 | — |
| CAP-A09 | Conditional actions (`if` inside events) | ❌ | spec 003 + mapping | WCP-4/6, WDP-39 | 7 | — |
| CAP-A10 | Notification delivery channels (email, in-app) | ❌ | Study 2 survey | universal (6/6 platforms) | 5 | prerequisite for CAP-A03 being real |
| CAP-A11 | Date arithmetic in actions (advance by frequency, `+ 1 Month`) | ❌ | Case 4 [UNTARGETED FINDING] | spec 003 date events (`Due Date - 1 Day`); Study 15 confirms this + CAP-E02 fully cover recurring-schedule needs | 7 | Case 7 adds two related but distinct arithmetic needs, kept as scope notes here rather than new IDs pending a second case: (a) *priority-keyed* date offset (`sla_offset(priority)` — High/Medium/Low resolve to different `+N days`, not a flat frequency like Case 4's), (b) plain numeric increment (`reopen_count + 1`) — same "arithmetic in actions" family, but numeric, not date |
| CAP-A12 | Ordinal/enum stepping in actions (move a `value_list` field to its next value in a declared sequence, e.g. `Low → Medium → High`) | ❌ | Case 7 [UNTARGETED FINDING] — `Escalate`'s Priority raise | — | 11 | Distinct from CAP-A11 (arithmetic on numbers/dates): stepping through a closed, ordered `value_list` has no "+1" — needs the sequence itself declared somewhere (on the field, most likely) |
| CAP-A13 | Cross-record field write (`set_field` on a different Machine's record, not just the record whose event fired) | ❌ | Case 8 [UNTARGETED FINDING] — `Payment.Reconcile` writing `Invoice.Amount Paid` / `Invoice.Status` | — | 8 | Distinct from CAP-A06 (`create_record`, makes a new record) and CAP-A08 (`aggregate_status`, a specific status-only rollup) — this is an arbitrary field write on an *existing*, different record. Guarded by CAP-X12 (cross-record write atomicity) once both exist |
| CAP-A14 | Aggregate-conditioned action (an event's action gated on a computed sum/count across many records, not one field on one record) | ❌ | Case 12 [UNTARGETED FINDING] — `Point Ledger Entry.Award`'s badge trigger (`sum(points) >= 100`) | achievement/badge engines (game platforms), threshold alerting | 9 | Every prior conditional action (CAP-A09) tests one field on one record (e.g. Case 4's `if Status = Overdue`). This tests a computed aggregate across many related records — closest relative is CAP-C10 (aggregate constraint), but that *blocks a write*; this *triggers a downstream action*. Composes CAP-F14 (aggregate) + CAP-A09 (condition) + CAP-A06 (create_record), but the combination has no name yet |
| CAP-A15 | Batch/series record generation (one action creates N related records at once from a formula, e.g. an amortization schedule) | ❌ | Case 14 [UNTARGETED FINDING] — `Loan.Disburse` generating Term Months' worth of Repayment Schedule Entries | amortization schedule generation (every lending platform); recurring-invoice generators | 8 | Distinct from CAP-A06 (`create_record`, exactly one record) and CAP-A11 (date arithmetic, one value). Composes CAP-A11 (per-installment date offset) + CAP-F14 (the principal/interest split formula) + a batch-count loop that has no home in the current Action grammar — every existing action operates on exactly one record or field per invocation |

## Constraints

| ID | Capability | Status | Discovered by | Pattern ref | Prio | Proof |
|----|-----------|--------|---------------|-------------|------|-------|
| CAP-C01 | `required` operator | ✅ | Case 1 | — | | conformance T04 |
| CAP-C02 | `after: today` operator | ✅ | Case 1 | — | | conformance T05 |
| CAP-C03 | `equals` / `not_equals` (in conditions) | ✅ | Case 1 | — | | conformance T06 |
| CAP-C04 | Conditional constraint (`condition` block) | ✅ | Case 1 | WDP-38 | | conformance T06/T07 |
| CAP-C05 | Comparison operators (`greater_than`, `less_than`, date `before`) | ❌ | schema doc + Case 4 | — | 10 | only `after: today` exists |
| CAP-C07 | Cross-field comparison (End Date after Start Date) | ❌ | Study 1 mapping | — | 10 | — |
| CAP-C08 | Cross-record constraint (one request per employee per day) | ❌ | spec 004 + mapping, first case evidence from Case 5 (negative-stock guard), second instance in Case 9, third in Case 6 (fund-balance guard, same shape as Case 5) | — | 14 | Case 9's `Fiscal Period.Close` guard is the reverse direction of the pattern — the constraint lives on the "one" side of a one-to-many, checking that every matching "many"-side record (each Journal Entry in the period) satisfies a condition, rather than checking one record against an aggregate on the other side (Case 5/6's shape). Both directions fall under the same capability, not two. **Case 8 deliberately does NOT exercise this**: matching a Payment to an Invoice by correlation heuristics (number/amount/customer) is a *different* sub-pattern (fuzzy matching, not a fixed comparison) — Case 8 keeps matching manual by design, so this remains a scope note, not a registered gap, until a case actually needs automatic correlation |
| CAP-C09 | **Constraints evaluated on event trigger** (today: Create only) | ✅ | **Study 1 mapping** | WDP-38 | | conformance T19. **Implemented 2026-07-11:** `Executor.Simulate` computes an event's resulting data without persisting; `internal/handler.TriggerEvent` runs the machine's existing `constraint.Violations` against that result and only calls `Executor.Persist` if none fire — same tier as Create's validation, no new Constraint grammar. Proven on Leave Request's pre-existing "Start Date must be after today": a record valid at Create can become invalid by the time an Approve fires (time passes between Submit and Approve) — T19 backdates a Submitted record's Start Date and confirms Approve is now blocked, not silently allowed through. |
| CAP-C10 | Aggregate constraint over line items (sum(debit) = sum(credit)) | ❌ | Study 6 accounting | double-entry invariant | 7 | depends on CAP-F16. Study 15 (third-pass): must operate on already-normalized values (post-conversion via CAP-F14) — the constraint itself never performs currency/unit conversion |
| CAP-C11 | Temporal period constraint (no posting into closed period) | ❌ | Study 6 accounting | lock dates, Period Closing Voucher | 10 | — |
| CAP-C12 | Uniqueness constraint (single-field or composite/multi-field) | ❌ | Case 11 [UNTARGETED FINDING] — `Follow`/`Like`'s "only once" rule; reinforced by Case 12 (`Membership`/`Badge Award`) and Case 21 (`Enrollment`, fourth instance) | SQL `UNIQUE` constraint, Django `unique_together` | 6 | No uniqueness mechanism exists today at all, single-field or composite. CAP-F20 (many-to-many) is its first forcing case — a join Machine without this can record the same relationship twice — but single-field uniqueness (e.g. Account Code, Entry Number) has been silently assumed correct in every case since Case 1 |

## Permissions

| ID | Capability | Status | Discovered by | Pattern ref | Prio | Proof |
|----|-----------|--------|---------------|-------------|------|-------|
| CAP-P01 | Role-based event permission | ✅ | Case 1 | WRP-2 | | conformance T11/T12 |
| CAP-P02 | Record-level ownership (only assigned user may act) | ❌ | Case 3 (P5) | WRP-1 Direct Allocation | 6 | — |
| CAP-P03 | Separation of duties (Requester ≠ Approver) | ❌ | spec 004 + mapping, first case evidence from Case 9 (SOX: Prepared By ≠ Posted By) | WRP-5 | 11 | — |
| CAP-P04 | Delegation | ❌ | Study 1 mapping, first case evidence from Case 7 (`Delegate`: current assignee hands off to a peer, keeping an accountability trail via `Delegated By`) | WRP detour | 15 | Distinct from Case 7's own `Escalate` (CAP-A04 dynamic recipient + CAP-A12 ordinal stepping) — delegation is a deliberate peer handoff retaining accountability, escalation is an automatic handoff to a fixed higher-authority role on SLA breach. Previously had no case evidence at all ("not yet in language examples") |
| CAP-P05 | CRUD-level permissions (read/create/edit per role — not just events) | ❌ | Study 2 survey | universal (6/6 platforms) | 6 | today every role sees every machine and record |
| CAP-P06 | Field-level visibility ("Salary visible only to HR") | ❌ | Study 2 survey + spec 004 example, first real case evidence from Case 20 (Medical Record's `Notes`, HIPAA-equivalent weight) | Salesforce field perms, Frappe permlevel | 11 | — |
| CAP-P07 | Public / unauthenticated access (a role requiring no login at all) | ❌ | Case 13 [UNTARGETED FINDING] — Blog `Visitor` reading Published Posts, submitting Comments | Every public-facing CMS/blog platform (WordPress, Ghost, ...) | 7 | Structurally different from "a role with very few permissions" — every permission row in the registry today implicitly assumes CAP-X02 authentication already succeeded. First case where a role is the *absence* of a session, not a restricted one |

## Views

| ID | Capability | Status | Discovered by | Pattern ref | Prio | Proof |
|----|-----------|--------|---------------|-------------|------|-------|
| CAP-V01 | `form` view (fields config drives inputs) | ✅ | Case 1 | — | | conformance T02 |
| CAP-V02 | `list` view (columns config drives table) | ✅ | Case 1 | — | | conformance T03 |
| CAP-V03 | `detail` view (all fields) | ✅ | Case 1 | — | | conformance T09 |
| CAP-V04 | `default_sort` honored in list | ⚠️ | Study 1 code check | — | 9 | loaded into model; store hardcodes `created_at DESC` |
| CAP-V05 | Filtered list (my records / pending my approval) | ❌ | Case 3 | — | 8 | — |
| CAP-V06 | Child records sub-list on parent detail | ❌ | Case 3 (P1) | — | 3 | depends on CAP-F13 |
| CAP-V07 | `dashboard` / `calendar` / `timeline` views | ❌ | schema doc, first real case evidence from Case 20 (Doctor Calendar — "what does Dr. X's Tuesday look like" cannot be served by a flat filtered list) | — | 15 | — |
| CAP-V08 | List search & filter | ❌ | Study 2 survey | universal (6/6 platforms) | 8 | — |
| CAP-V09 | Declarative view-level filter (Due Today, Overdue Tasks) | ❌ | Case 4 [UNTARGETED FINDING] | — | 8 | view `filter` block in metadata |
| CAP-V10 | Composed dashboard view (sections sourcing multiple machines) | ❌ | Study 5 Portal GA | 9 shared DigestSections | 12 | Case 13's one-page site is the same composition shape applied to a **public** landing page rather than an internal dashboard — reinforces the pattern but does not close the registry's separate "not yet studied" Page/Theme gaps (`capability-registry.md` §Tracked but Not Yet Studied); a dedicated study is still needed for those |
| CAP-V11 | Channel-independent view rendering (web + email from one section) | ❌ | Study 5 Portal GA | ADH email digest reuse | 14 | **evidence-thin** (Study 9 retrofit: single source, possibly composable) — HOLD at Proposed until second independent source |
| CAP-V12 | Multi-step form (wizard) view | ❌ | Study 5 Portal GA | HIRADC wizard | 11 | — |
| CAP-V13 | Aggregate report view (group-by, hierarchy rollup, period compare, running balance) | ❌ | Study 6 accounting, field-level design in Case 9 | Trial Balance, P&L, GL | 9 | the report class every vertical needs. Case 9's Trial Balance requires CAP-F16 child-table rows (Journal Entry Line) to be independently queryable across parent documents — see CAP-F16's reporting-independence note |
| CAP-V14 | Manual/free ordering (a record's position is directly user-edited via drag-and-drop, with no formula behind the order — distinct from CAP-V04's *declarative* `default_sort`) | ❌ | Case 19 [UNTARGETED FINDING] — `List.Reorder`, `Card.Move` | Trello/Jira/Notion board reordering — universal to kanban-style tools | 9 | The reorder action must renumber every *sibling* record when one moves, not just the moved record — a batch update shaped like CAP-A15 (series generation) but rewriting existing records instead of creating new ones. `Card.Move` composes this with a cross-record reference write (CAP-F13/A13) when a card changes List |

## Record Lifecycle

| ID | Capability | Status | Discovered by | Pattern ref | Prio | Proof |
|----|-----------|--------|---------------|-------------|------|-------|
| CAP-R01 | Create record (with default status) | ✅ | Case 1 | — | | conformance T08 |
| CAP-R02 | **Edit / update record via form** | ❌ | **Study 1 code check** | — | **5** | no update form exists — CRUD's U is missing |
| CAP-R03 | Delete / archive record | ❌ | Study 1 code check | — | 12 | — |
| CAP-R04 | Event audit log (record_events, snapshot before mutation) | ⚠️ | Case 1, reinforced by Case 9 (SOX: audit trail is a compliance requirement, not optional polish) | — | 9 | logged to DB; no UI to view history |
| CAP-R05 | Pagination on list views | ❌ | Study 1 code check | — | 11 | — |
| CAP-R06 | Data import/export (CSV) | ❌ | Study 2 survey | 5/6 platforms | 12 | — |
| CAP-R07 | Record immutability after state (posted/submitted frozen; amend-via-new-version) | ❌ | Study 6 accounting + Case 6 declaration | docstatus model | 6 | stronger than CAP-E06 — guards edits, not just events |
| CAP-R08 | Editable scratch state (a record with none of its eventual invariants enforced until an explicit commit point converts it into a real, constrained record) | ❌ | Case 15 [UNTARGETED FINDING] — `Cart` freely edited, `Checkout` is the actual commit point into `Order` | shopping cart pattern (every e-commerce platform); form builder "draft" autosave | 10 | Distinct from an ordinary Draft status (Case 9's Journal Entry is validated even in Draft) — this is the opposite end of CAP-R07's spectrum: not "frozen after a state," but "unconstrained before a state." Two ends of the same lifecycle axis, worth keeping in the same area for that reason |

## Cross-Cutting

| ID | Capability | Status | Discovered by | Pattern ref | Prio | Proof |
|----|-----------|--------|---------------|-------------|------|-------|
| CAP-X01 | Multi-application, multi-machine in one workspace | ✅ | Case 2 | — | | conformance T01 |
| CAP-X02 | Real authentication (today: prototype role cookie) | ❌ | prototype design | WRP-4 | 13 | — |
| CAP-X03 | Machine-level config block (approval_mode etc.) | ❌ | Case 3 (P4) | — | 4 | — |
| CAP-X04 | Metadata live reload (today: restart required) | ❌ | ADR-002 | K8s reconciliation | 14 | plan in `decisions/002-metadata-loading.md` |
| CAP-X05 | Metadata validation before load (dangling refs, bad operators) | ❌ | Study 1 mapping | Terraform plan-before-apply | 7 | Scope extended (Study 15, fifth-pass): composite/reference-sugar types (`money`, future `quantity`) must have their required companion (`currency:`/`currency_field:`) declared inline — missing companion = load-time rejection, same discipline as CAP-F13's dangling-reference check. Language-level safeguard against a metadata author (human or AI) forgetting the conversion setup — not an app-UI concern |
| CAP-X06 | Workspace isolation in routing/authz | ⚠️ | prototype design | — | 8 | implementation strategy decided: PostgreSQL RLS (ADR-003) |
| CAP-X07 | Auto-generated REST API per machine | ❌ | Study 2 survey | 5/6 platforms | 10 | Case 8 clarifies this is the *outbound* surface (the runtime exposing CRUD for its own machines) — receiving inbound third-party webhook calls (signature verification, arbitrary payload shape) is CAP-E04's concern, not this one. Both are needed for Case 8, neither subsumes the other |
| CAP-X08 | Metadata package export/import (portable app definition) | ❌ | Study 2 survey | universal (6/6 platforms) | 9 | today: hand-written SQL seeds; blocks "one knowledge, many runtimes" operationally |
| CAP-X09 | Organizational unit scoping (org dimension on records, permissions, selectors, per-unit timezone) | ❌ | Study 5 Portal GA | BranchPeriodSelector + timezone rules | 6 | records today have no org context at all |
| CAP-X10 | Metadata-driven index management (hot fields from view filters/sorts → expression indexes, reconciled) | ❌ | Study 8 [SCALE FINDING] | K8s reconciliation applied to indexes | 10 | ADR-003 |
| CAP-X11 | Lazy per-workspace metadata loading + cache (singleflight, LRU, LISTEN/NOTIFY eviction) | ❌ | Study 8 [SCALE FINDING] | ADR-002 Option C unified with scale cache | 7 | ADR-003; retires boot-time LoadAll |
| CAP-X12 | Cross-record write atomicity (an event's actions across two-or-more Machines — e.g. append a ledger entry and recompute a rollup — commit as one unit or not at all) | ❌ | Case 5 declaration (original, untargeted) + `benchmarks/006-inventory-warehouse-benchmark.md` (receiving reconciliation, picking allocation), reinforced by Case 8's `Payment.Reconcile` (CAP-A13 write + Invoice status check must commit together) | database transaction / unit-of-work pattern | 8 | first cluster identified without a CAP id — registered once Case 5 gave it worked examples. Guards CAP-A06 (`create_record`) chains, CAP-A13 (cross-record field write) chains, and CAP-F14 aggregate rollups from partial-write drift |
| CAP-X13 | Webhook/external-event idempotency (dedupe by provider event ID, atomic check-and-claim, safe to retry) | ❌ | Case 8 [UNTARGETED FINDING] | Stripe/Shopify/GitHub webhook convention: at-least-once delivery is the contract, idempotency keys make duplicates safe, claim via `INSERT ... ON CONFLICT DO NOTHING` (never check-then-act), return success for duplicates | 9 | Distinct from CAP-X12 (atomicity across Machines on a *successful* write) — this guards against *reprocessing the same event twice*, including two near-simultaneous retries racing each other. A prerequisite for CAP-E04 (external events) to be safe in production, not just functional |

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
| CAP-O02 | Master data designation (canonical machines: ownership, cross-app referenceability, deactivation semantics) | ❌ | Case 10 (cross-app narrative), Study 15 (`Currency` via CAP-F17), Case 18 (`Employee` — the textbook cross-app master-data candidate: HR owns it, but Leave Request (Case 2), Helpdesk (Case 17), and any future app all need to reference the same person) | DDD shared kernel, Portal GA PICA + Data Mesh | 8 | flag + rules on machine, not a new hierarchy level. Scope corrected (Study 15 fourth-pass): `Equipment` used only within one application does NOT need this — that's fully resolved by CAP-F13 alone. CAP-O02 evidence is specifically the cross-application case (Case 10) + Currency + Employee, still clearing the dual-evidence bar |
| CAP-O03 | Navigation metadata (app grouping, role-aware menus, workspace home) | ❌ | Case 10 | runtime/004 already names Navigation — spec predicted it | 9 | — |
| CAP-O04 | Workspace-wide search across machines (permission-trimmed) | ❌ | Case 10 | — | 12 | depends on CAP-P05 |
| CAP-O05 | Unified notification center (inbox, per-user channel preferences, digest grouping) | ❌ | Case 10 | Portal GA message dispatcher | 8 | extends CAP-A10 to workspace service |
| CAP-O06 | Business calendar (holidays, working-day rules) consumable by date arithmetic/SLA | ❌ | Case 10 | spec 001 Holiday example | 9 | feeds CAP-A11, CAP-E02 |

---

# Implementation Order (consolidated)

| Prio | Capabilities | Theme |
|------|-------------|-------|
| ~~1~~ | ~~CAP-F13~~ ✅ | ~~Reference fields — biggest unlock (6 patterns depend on it)~~ done 2026-07-11, conformance T13–T16 |
| ~~2~~ | ~~CAP-E06, CAP-C09~~ ✅ | ~~Correctness: state guards + constraints on events~~ done 2026-07-11, conformance T17–T19 |
| 3 | CAP-A02, CAP-V06 | Dynamic values + child sub-list (completes Case 3 basics) — **next up** |
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
