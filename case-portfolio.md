# Case Portfolio

> Artifact 3 of the Capability Roadmap — deliberate case selection.
>
> Cases are chosen to hit untested pattern clusters, not at random.
> Target patterns are declared **before** the case is written, so each case
> is a designed experiment — and surprises (patterns the case reveals that
> were not targeted) are themselves findings.
>
> Status: v0.34 — Case 3 gains a 2026-09-08 note (curated navigation visibility — owner-observed
> nav clutter, `CAP-O03 Tier 4` admitted ❌ Proposed same day). Previously v0.33 — Case 3's
> 2026-09-06 note gets its admission-test outcome (2026-09-07): the
> approver-type toggle and saved-template gaps pass all five criteria (`CAP-F24`, `CAP-V28`, both
> ❌ Proposed); the step-reorder/authoring gap fails A4 (composes from already-✅ `CAP-F16`, no new
> row). Previously v0.32 — Case 3 gains a 2026-09-06 note: a new owner request (per-Step user-or-group
> approver choice, submission-time step reordering, and a saved default flow per Document Type),
> prototyped only as a static UI-sample mockup (`app/web/static/ui-sample/document-submit.html`),
> not yet run through `capability-registry.md`'s admission test — three candidate gaps named,
> none registered as a CAP row yet. Previously v0.31 — Case 3 gains a 2026-08-23 extension note
> (PDF signature placement, Study 32, `benchmarks/024-pdf-signature-approval-study.md`), new
> CAP-F22 registered. Previously v0.3 — full 21-case portfolio documented (Cases 1–10 original +
> Cases 11–21 Extended Portfolio); Cases 1–2 ✅ done, the remaining 19 ⚠️ documented with
> targets/gaps registered against `capability-registry.md` | Created: 2026-07-04 |
> Updated: 2026-09-06

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

**Correction (2026-08-29, six days later)**: the line above overclaimed. Verified this session,
directly with the owner: CAP-O07 builds *who holds a role via group membership* (a permission
concern — `GroupStore.RolesForUser`) — it does not, on its own, narrow a `user` Field's own
candidate *picker* to a Group's members (a rendering/query-scoping concern). Different mechanisms;
having CAP-O07 didn't actually give this note's own claim for free. The real gap was admitted and
closed as **CAP-F23** (see below and `capability-registry.md`'s own row) the same day this
correction was written.

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
`capability-registry.md`'s CAP-F22 row.

**CAP-V21 implemented 2026-08-29, same day** — conformance T209–T211, full suite 211/211, zero
regressions. The Signature Placement screen this note's own §3 designed is now real: Approval
Step's Detail page gains a "Set Position" link to a new `/{machineID}/{recordID}/place` page
previewing the Document's PDF with a draggable pin, defaulting to center until moved. Full build
notes: `capability-registry.md`'s CAP-V21 row.

**CAP-V20 implemented 2026-08-29, same session — Study 32's own capability list is now closed**
(CAP-F22, CAP-V21, CAP-V20 all done). The Decision screen's own stepper is real: Approval
Document's Detail page gains a "View Progress" link to `/{machineID}/{recordID}/progress`, an
ordered done/current/pending list over its Steps with the current step's real Approve/Reject
buttons inline, scoped to whoever actually owns that step. Full build notes:
`capability-registry.md`'s CAP-V20 row.

**CAP-F23 implemented 2026-08-29, same day — Case 3 is now actually closed against Study 32's
original business requirement**, not just its own narrowed capability list. The "approvers are
drawn from a work group" line at the top of this note was real and had never been built (see the
correction above, this same file). Approval Step's `fld_as_approver` field now declares
`{"restrict_to_group":"Document Approvers"}`; its picker offers only that Group's own members,
by intersection with the Approver role it already required — not a new authorization mechanism,
the existing role/ownership check at Approve time is completely unchanged. Full build notes and
admission reasoning: `capability-registry.md`'s CAP-F23 row.

**New request (2026-09-06), UI-sample exploration only, not built:** restated close to source
(Bahasa Indonesia, kept close to the original so nothing is lost in translation) — "approval bisa
dipilih, oleh user yang ditunjuk atau grup yang ditunjuk, dan bisa diatur urutannya, yang akan
dipakai jika sequential. Per dokumen, harus ada pengaturan ini, bisa juga flow approval tersebut
disimpan, menurut kategori dokumen." In English: for each Approval Step, the submitter must be
able to choose an approver that is *either* a specific user *or* a specific Group (not one fixed
at Machine-design time), reorder the steps at submission time (the order is what Sequential mode
actually follows), have this configurable **per document being submitted** rather than fixed once
for the whole Approval Document Machine, and optionally **save the whole step chain as a reusable
default keyed by Document Type/category**, pre-filling future submissions of that category while
staying editable per document.

Prototyped as a static mockup only — `app/web/static/ui-sample/document-submit.html`'s "Approval
steps" section (per-step User/Group toggle, approver picker, ▲▼ reorder, add/remove) and its
"Save this as the default approval flow for `<Document Type>` documents" checkbox; index card at
`app/web/static/ui-sample/index.html`. No backend, storage, or metadata schema behind it — this is
a design exploration surfacing a gap, the same role Study 32's own mockups played before CAP-F22/
CAP-F23 existed.

**Why this is not already covered by CAP-A07/CAP-A08/CAP-F23:** CAP-A07/CAP-A08 give the
Approval Document → Approval Step Sequential/Parallel machinery its per-mode gating, but say
nothing about *who* creates the Step records or *in what order* — today that is whatever the
Machine's own metadata/records already declare, decided once at design time. CAP-F23 lets a
Step's `user` Field narrow its candidate picker to one *statically declared* Group
(`{"restrict_to_group":"Document Approvers"}`) — it does not let the picker toggle between a
specific user and a specific group per submission, nor let the submitter choose or reorder Steps
at all. Saving a chosen step chain as a reusable default keyed by Document Type is a distinct
concern again — closest existing precedent in this registry is none; it is not template/default-
value machinery that exists anywhere today. Three candidate gaps, not yet run through the
admission test in `capability-registry.md`: (1) a per-record, submission-time approver-type
toggle (user vs. group) rather than a design-time-fixed restriction: (2) submission-time
Step ordering/authoring, rather than Steps being pre-existing records; (3) a saved-template
mechanism scoped by Document Type. Left as documented, undecided gaps here — no CAP row opened
yet, consistent with this file's own role (name the business requirement and check it against
what's ✅) versus `capability-registry.md`'s (run the actual admission test and register a row).

**Admission test run (2026-09-07):** all three checked against `capability-lifecycle.md` §2. (1)
the approver-type toggle passes all five criteria — registered **`CAP-F24`** ❌ Proposed, dual
evidence being this note plus WRP-3 Deferred Allocation (already cited on `CAP-F13`'s own row for
this same case). (2) submission-time Step ordering/authoring fails A4 (non-composability) — it
composes entirely from already-✅ `CAP-F16` (`child_lines`) once `CAP-F24`'s field shape exists,
so it needs no capability row of its own; a build task, not an admission blocker. (3) the
saved-template mechanism passes all five — registered **`CAP-V28`** ❌ Proposed, dual evidence
being this note plus the DocuSign/Adobe Sign Templates and Salesforce Approval Process pattern,
depending on `CAP-F24` existing first. Full reasoning on each row in `capability-registry.md`.
Neither is built yet — admission only, per this repo's own "a capability is real only once a case
exercises it AND a test verifies it."

# Case 3 — extension note (2026-09-07): approval assistant (MCP) — target declaration

**Business reality:** an approver (or the submitter) asks an AI assistant, in plain language,
*"which documents are waiting for my approval, and why is each one blocked?"* or *"approve every
document in my queue that satisfies the Finance policy"* — and the assistant does it **inside the
same application**, seeing only what that person may see and performing only the Events that
person may perform, with every action landing in the same audit trail as a click in the UI.
Raised by Study 37 (`prototype/objectstack/`, the ObjectStack comparator study) and its
second-opinion reconciliation (`docs/second-opinion-reconciliation.md` §3): ObjectStack's headline
is that every object/action is a governed MCP tool under the same RBAC/RLS/FLS as a human; the
worked prompts above are the second review's own examples and are a natural continuation of this
case's lineage (Study 32 signatures → per-step approvers → saved flows). Declared here, per this
file's rules, **before** anything is built, so `CAP-X16` has the terrain half of admission A1.

**Declared targets:**

| Target | Capability | Pattern |
|--------|-----------|---------|
| "Perform Approve on Step X" from an assistant, with the same permission/ownership/SoD checks as the UI | **CAP-X07 Tier 2 (new)** — `POST /api/v1/{machine}/{record}/events/{event}` + full Data API + OpenAPI | The Event *is* the tool; no second authorization path |
| Expose Machines and `ai_exposed` Events as MCP tools under a session or user-bound API key | **CAP-X16 (new)** | ObjectStack `packages/mcp`, ADR-0011/0109 |
| "Which documents await *my* approval" | CAP-V05 (filtered list "pending my approval") + CAP-P05/P06 trimming | Reused — the assistant lists what the person's own inbox would |
| "Why is this one blocked" | CAP-W01 requirements + CAP-W05 process map + CAP-V20 stepper state | Reused — the Overlay already knows which requirement is unmet; the assistant reads it |
| "Approve every one that satisfies policy X" | CAP-C13 expression (proposed) as the policy predicate, CAP-P03 SoD unchanged | The assistant filters; the runtime still refuses what the person may not do |
| Every assistant action auditable to the person, not to "the AI" | CAP-R04 `record_events` (`performed_by` = the user the key is bound to) + CAP-I04 correlation | Reused — ObjectStack design principle IV, same posture |

**Deliberately out of scope:** the assistant's own model, prompting, or conversation memory — the
runtime exposes tools, it does not host an agent (ObjectStack keeps those in its commercial ObjectOS
tier too). Any autonomous approval without a human-bound credential is a non-goal, not a future ⚠️.

**Status:** target declaration only; `CAP-X16` and `CAP-X07` Tier 2 registered ❌ Proposed in
`capability-registry.md` v0.56 the same day. Not built.

---

# Case 3 — extension note (2026-09-08): curated navigation visibility

**Business reality:** owner-observed, directly against the live `app_approval` application while
verifying this case's own metadata-only mockup fidelity (`app/docs/ui-component-library.md`'s
v1.6/v1.7 status notes): *"aplikasi butuh menu yang spesifik... tidak semua halaman harus ada di
menu"* — a good application curates its navigation to the user's real flow, not one link per
Machine regardless of whether that Machine is ever meant to be a direct destination. Concrete,
observed symptom: `app_approval`'s sub-nav strip (`CAP-O03` Tier 2) lists `Approval Document`,
`Approval Step`, and `Signature` as three equal-weight links. Only `Approval Document` is a real
entry point a Submitter or Approver would ever choose from a menu — `Approval Step` is reached
exclusively through a Document's own flow (embedded authoring via `child_lines`, the Step's own
Detail page linked from the Decision Progress stepper) and `Signature` only through the signature
registration flow; neither is a "browse all Signatures" or "browse all Steps" destination for a
real user. `subNavFor`/`AppMachines` (`app/internal/handler/handler.go`) render every Machine in
an Application, trimmed only by `Guard.CanRead` (`CAP-P05`) — there is no notion of "this Machine
exists and is reachable, but isn't a menu destination."

**Prior art within this repo, checked before treating this as new:** `capability-registry.md`'s
own `CAP-O03` row already named an adjacent gap and deliberately left it unbuilt — "explicit
menu-ordering/icon/label-override metadata... no case has asked for curated ordering yet; add a
real Navigation metadata table only if one does, matching this project's own 'Infer Before
Configure' principle" (`001-design-principles.md` §6). That note is about *ordering/label*, not
*visibility* — a different question (this app's own Approval Document/Approval Step/Signature are
already alphabetically fine; the problem is that all three appear at all) — but it is the exact
trigger condition CAP-O03's own row named: a real case asking for it. This note is that case.

**Declared target:**

| Target | Capability | Pattern |
|--------|-----------|---------|
| Approval Step / Signature reachable via the Document's own flow, but absent from the app's sub-nav strip and app-launcher card list | **`CAP-O03` Tier 4 (new)** — an exception flag on the Machine, read by the already-existing `subNavFor`/`AppMachines` | Salesforce App Manager (Tabs — an Object not assigned a Tab has no nav entry but is fully functional via relationships), Odoo Technical Settings (most line/detail models have no menu item, reached only through their parent), Frappe Desk (many DocTypes ship with no default List sidebar entry), Notion/Airtable (hide-from-sidebar per page/table) |

**Why not composed from an existing mechanism (checked, not assumed):** `Permission.can_read:
false` removes read access entirely — it would also break the legitimate direct link from a
Document's own Decision Progress stepper into its Step. Inferring visibility from "is this Machine
some other View's `child_lines.machine` or `steps_machine` target" was considered and rejected —
`CAP-F16`'s own row already proves this heuristic is unsound: Journal Entry Line (a `child_lines`
target) and Item Unit Conversion (also a `child_lines` target) have opposite correct nav answers
per that row's own "Reporting-independence note" — Journal Entry Line *must* stay independently
browsable (CAP-V13's Trial Balance groups it across parent documents), Item Unit Conversion never
is. The same structural shape, two different correct outcomes — a real counter-example, not
speculation, ruling out silent inference for this specific decision.

**Deliberately out of scope for this note:** menu ordering, icons, and label overrides — CAP-O03's
own row already named those as deliberately unbuilt pending a real case, and this note's own
evidence doesn't ask for them; only visibility (show/hide) is in scope.

**Status:** `CAP-O03` Tier 4 registered ❌ Proposed in `capability-registry.md` v0.64 the same day,
full A1–A5 admission test in that row. Not built.

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
