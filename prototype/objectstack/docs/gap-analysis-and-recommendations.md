# Gap Analysis and Recommendations — What Menata Runtime Should Take from ObjectStack

> Part of Study 37 (`../README.md`). Builds on `architecture-and-backend.md` and
> `capability-comparison.md`; cites the same ObjectStack files (clone @ `ac76425f`) and
> `capability-registry.md` v0.54 rows. **Nothing here is admitted** — each candidate is pre-screened
> against `capability-lifecycle.md` §2 (A1–A5) and left for the owner to admit or reject.
> Status: v1.0 | Created: 2026-09-07 | Updated: 2026-09-07

---

## 1. Framing — the owner's two words

The owner's brief defines the target: **flexible = can build anything; powerful = performance
stays good on efficient server resources.** Those two pull in opposite directions in ObjectStack's
own design — it buys flexibility with a 45-package plugin kernel, a 17k-line SQL driver, a DAG
engine, a React SPA and code escape hatches, and then needs cluster primitives, an authz-cache
invalidation contract (ADR-0127) and a schema-drift subsystem to stay correct and fast. Menata
Runtime's constraints are the opposite starting point and are not negotiable in this study:

| Constraint | Source |
|---|---|
| Single Go binary, no containers, shared VPS | `app/ARCHITECTURE.md` §"Deployment target: no containers, by constraint" |
| Metadata only — no code hatch | `001-design-principles.md` Metadata First; `capability-lifecycle.md` A5 "business language exists" |
| Runtime owns behavior; no client framework | `app/ARCHITECTURE.md` §"Client-side JavaScript policy" |
| Emergent workflow + Process Overlay, not a DAG engine | Studies 19–21 (`benchmarks/011`–`013`), server-economy measurement in Study 20 |
| One capability = one registry row = one conformance proof | `capability-registry.md` ratchet; `capability-lifecycle.md` §3 |

So the question for every ObjectStack feature is not "do they have it" but: *does a case need it,
can it be said in business language, can it be built as metadata the existing Go runtime
interprets, and does it stay cheap at the data volumes Study 8 measured?* The candidates below
are the ones that survive that filter; §5 lists what deliberately does not.

## 2. Consolidated gap table

One row per gap found in `capability-comparison.md`, with the registry's current state and the
candidate that would close it. "Existing row" means a registry row already names the gap — this
study adds evidence, not a new capability.

| # | Area | Gap (ObjectStack has / Menata lacks) | ObjectStack evidence | Menata state | Candidate |
|---|---|---|---|---|---|
| G1 | Expression | General expression language for formulas, conditions, visibility, filters, old-vs-new predicates | `packages/formula` (CEL + stdlib, SQL pushdown); used on 9 surfaces | `CAP-F14` ⚠️ one multiply; fixed operator set; `CAP-A02` fixed tokens | **R1** |
| G2 | Analytics | Semantic dataset layer (one metric definition reused by every report/dashboard) | ADR-0021; `ui/dataset.zod.ts`; `service-analytics/dataset-compiler.ts` | `CAP-V13`/`V10`/`A14` each inline | **R2** |
| G3 | Analytics | Chart rendering on dashboards/reports; period-over-period `compareTo` | `ui/chart.zod.ts` (~20 types); `examples/app-crm/src/dashboards/pipeline.dashboard.ts` | `dashboard.templ:7` "deliberately count-based, not charts" | **R3** |
| G4 | Storage/perf | Indexes on hot fields | table-per-object + `indexes[]` | `GIN(data)` only; `CAP-X10` ❌ deferred | existing row **`CAP-X10`** (R4) |
| G5 | Permissions | Scope depth `own / own_and_reports / unit / unit_and_below / org` | `security/permission.zod.ts:17`; ADR-0057 | `CAP-P02` own vs all only; `CAP-X09` dissolved | **R5** |
| G6 | Identity | Organizational unit tree + manager designation on users | `sys_business_unit`, `sys_user.manager_id` | Case 18 has Employee↔Manager as a Machine, not on `CAP-O01` users | **R6** |
| G7 | Approvals | Manager-chain / unit-subtree approver resolution | `ApproverType` `manager`, `department` | role / user-field / group only | **R7** (depends on R6) |
| G8 | Approvals | Per-group sign-off ("each group must approve") | `behavior: per_group` | `CAP-W03` ANY / N_OF_M | **R8** (`CAP-W03` Tier 2) |
| G9 | Approvals | SLA breach *actions* (auto-approve / auto-reject / reassign) | `ApprovalEscalationSchema.action` | `CAP-W04` `on_breach` = notify + escalate_to | **R9** (`CAP-W04` Tier 2) |
| G10 | Approvals | Send-back-for-revision with revision cap and distinct `returned` state | ADR-0044, `approval_revise` node, `maxRevisions` | Reject→Draft composition, uncounted | **R10** |
| G11 | Automation | Outbound HTTP / webhook action | `http` node (outbox-backed), `plugin-webhooks` | no action type makes an outbound request | **R11** |
| G12 | Automation | Automation run trace (which event → which actions → which records, per request) | `sys_automation_run`, run summaries, Console viewer | `CAP-I04` ⚠️ correlation id half | existing row **`CAP-I04`** |
| G13 | Automation | Predicate over the *previous* value (`old.amount`) | `record_change` flow conditions | `record_events.snapshot` exists, conditions can't reference it | folded into **R1** |
| G14 | Fields | Multi-select value list | `multiselect`, `tags`, `checkboxes` | `CAP-F03` row's own scope note (Case 13 tags) | **R12** (`CAP-F03` Tier 2) |
| G15 | Fields | Format validators (email / url / phone) | `email`, `url`, `phone` types; `format` validation rule | none | **R13** |
| G16 | Fields | Percent | `percent` (+ `percent-scale.ts`) | `number` | **R14** |
| G17 | Fields | Display-field designation for `reference` pickers/links | `nameField` (ADR-0079) | `CAP-F13` row: "heuristic … pending a real display-field designation" | **R15** (`CAP-F13` Tier 2) |
| G18 | Fields | Encrypted `secret` field | ADR-0100 | none | defer — needed only with R11 credentials; fold into R11's design |
| G19 | Views | Gantt / gallery / map / tree lenses | `VisualizationTypeSchema` | 12 View types, none of these | **R16** (one row each if admitted) |
| G20 | Views | Per-user saved views (personal filter/sort/columns) | `type: 'personal' | 'collaborative'`, `client.views.*` | `CAP-V05` metadata-declared only | **R17** |
| G21 | Views | Multi-Machine screen flow (create A, then B linked to A, one wizard) | `screen` nodes with `objectName` + `idVariable` | `CAP-V12` single-Machine wizard | **R18** (`CAP-V12` Tier 2) |
| G22 | Views | Form sections / multi-column layout; column footers; row colour rules | `sections[].columns`, `ColumnSummarySchema`, row colour | one column of fields | **R19** (presentation; low) |
| G23 | Views | XLSX export | `xlsx-module.ts` | CSV only (`CAP-R06`) | **R20** (`CAP-R06` Tier 2) |
| G24 | Views | Drawer / modal navigation modes | `NavigationModeSchema` | page only | presentation; defer to a design pass (ADR-008 lineage) |
| G25 | Platform | Governed MCP surface (every Machine/Event an AI tool under the same permissions) | `packages/mcp`, ADR-0011/0109 | none | **R21** |
| G26 | Platform | Execution-pinned metadata versions (paused work keeps its definition) | ADR-0009 | `CAP-W02` ❌ superseded by `CAP-W07` | existing row **`CAP-W02`** — re-scope narrowly (see R22 note) |
| G27 | Platform | Outbound email transport | `plugin-email`, ADR-0012 | `CAP-A10` in-app only; `CAP-O10` ❌ | existing row **`CAP-O10`** |
| G28 | Platform | Scheduled report delivery | `plugin-reports` digests | `CAP-V11` ❌ HOLD | existing row **`CAP-V11`** (blocked on G27) |
| G29 | Platform | i18n labels | translation bundles, `service-i18n` | none | defer — no case names a second UI language |
| G30 | Permissions | Criteria sharing rules; explain engine; record-view auditing; SSO/MFA; field visible-but-read-only per role (`CAP-P06` is read-surface only) | ADR-0057/0066; `explain-engine.ts`; ADR-0069; `FieldPermissionSchema.editable` | none / `CAP-P06` ✅ read side | defer — no case pressure; explain becomes cheap after R5; read-only field = `CAP-P06` Tier 2 when a case names it |
| G31 | Discipline | Silent metadata no-ops become load-time errors | ADR-0049/0078, tombstones | `runtime-metadata-schema.md` documents the no-op list | §6 |

## 3. Recommendations — prioritized

Tiers are by leverage (how many gaps one capability closes) and by dependency, not by effort.
Each entry: what, why (best-practice basis), evidence for A1, admission pre-screen, performance
note, suggested registry ID *if admitted*.

### Tier 1 — foundations that close many gaps at once

**R1 — Expression layer for formulas and predicates** (closes G1, G13; subsumes date macros,
old-value conditions, conditional visibility; unblocks R2/R5's predicates)

- *What.* Admit one expression grammar usable wherever the metadata today takes a
  `{field, operator, value}` triple or a `computed` field: constraint conditions, event
  conditions, view filters, `computed` fields, action values. Keep the existing operators as sugar
  that compiles to the same evaluator so no seed changes.
- *Best-practice basis.* CEL (Common Expression Language) is the industry answer to "safe,
  non-Turing-complete, linear-time predicates in configuration": Kubernetes admission policies,
  Envoy, Google Cloud IAM, Firebase rules. Its reference implementation is Go
  (`github.com/google/cel-go`) — Menata would adopt the *language*, zero ObjectStack code, with a
  static type-checker and cost estimator that bound evaluation cost per expression (the property
  that keeps it "powerful"). Salesforce formula fields, Frappe, Airtable, ServiceNow all converged
  on a bounded expression language for exactly this slot (Study 2 survey already lists Salesforce
  formula as `CAP-F14`'s pattern ref).
- *A1 evidence.* Map: ObjectStack (9 surfaces) + Salesforce/Frappe (Study 2). Terrain: `CAP-F14`
  ⚠️'s own deferred sub-patterns (b)/(c), Case 8 interest/late-fee formulas, Case 14 lending
  schedules, Case 15 discounts — all modelled with workarounds today.
- *Pre-screen.* A1 ✅ · A2 ✅ table stakes · A3 ⚠️ it touches Field (F14) *and* Constraint/Event
  conditions — per A3 register it as **two rows**: `CAP-F14` completion (formula field) and a new
  Constraint-area row for an `expression` operator · A4 ✅ not composable · A5 ✅ "Total = Amount ×
  (1 − Discount)" is business language; `.menata` guides already carry derived-value sentences.
- *Powerful.* Evaluate in-process against the already-loaded record map (no DB round trip);
  cel-go compiles once at metadata load (`CAP-X04` swap) and checks cost; push simple comparisons
  down to JSONB predicates for list filters (ObjectStack's `cel-to-filter.ts` shows the shape and
  its `cel-pushdown-limits.ts` shows where to stop). No new process, no client JS.
- *Guardrail to copy.* ObjectStack's hook-body model — declare what an expression may touch, fail
  closed otherwise (`data/hook-body.zod.ts`). For Menata: expressions read the current record,
  `old` snapshot, `current_user`, `today/now`, and nothing else; no I/O, ever. That keeps R1 from
  becoming a code hatch.
- *Suggested IDs.* `CAP-F14` → ✅ (formula), **`CAP-C13`** expression operator.

**R2 — Semantic dataset layer** (closes G2; prerequisite for R3 to be honest)

- *What.* A `dataset` metadata object: base Machine + reference joins (via `CAP-F13`/`F16`
  back-references) + named dimensions + named measures (count/sum/avg/min/max/count_distinct,
  derived ratio/difference). `report` and `dashboard` Views bind a dataset by name and pick
  dimensions/measures instead of declaring aggregates inline.
- *Best-practice basis.* ADR-0021 documents the failure Menata is walking toward — three inline
  aggregate grammars → "revenue" defined three times → finance numbers diverge — and the industry
  answer (Looker LookML, Power BI datasets, dbt metrics, Salesforce CRM Analytics datasets): a
  governed semantic layer below, thin presentations above. Study 6 (accounting) already demands
  "one balance, many views".
- *A1 evidence.* Map: ObjectStack ADR-0021 + the four BI platforms it cites. Terrain: Case 9
  (trial balance *and* a dashboard tile both compute balance), Case 15/16 (sales totals by
  product/day on list, report, dashboard).
- *Pre-screen.* A1 ✅ · A2 ✅ · A3 ✅ View area (a data-shape declaration Views consume) · A4 ⚠️ a
  single report is composable today; the *sharing* of one definition across Views is not — that is
  the capability · A5 ✅ "Sales by Region is Sum of Amount grouped by Region".
- *Powerful.* Compile a dataset to one SQL `GROUP BY` over `records.data->>` (with R4's expression
  indexes on the dimension fields); cache results per (dataset, filter, permission scope) with a
  short TTL — ObjectStack's `cube-registry.ts` + `read-scope-sql.ts` is the shape (permission
  filter *inside* the aggregate, never post-filtered). Dashboards then cost one query per
  dataset, not one per tile.
- *Suggested ID.* **`CAP-V22`** dataset (semantic aggregate definition).

**R3 — Chart widget on dashboards and reports** (closes G3)

- *What.* A `chart` section type for `CAP-V10` dashboards and `CAP-V13` reports: bar / line /
  pie / kpi-with-delta, bound to an R2 dataset, rendered as **server-side SVG** (Templ) — no
  charting library, no client JS, honouring `app/ARCHITECTURE.md`'s policy. `compareTo:
  previous_period | previous_year` as a KPI delta.
- *Basis.* Every platform in Study 2 renders charts from metadata; ObjectStack's `chart.zod.ts`
  taxonomy (comparison / trend / distribution / single-value) is a sound minimal set.
- *Pre-screen.* A1 ✅ (Study 2 six platforms + Case 9/15/16 dashboards in `case-portfolio.md`) ·
  A2 ✅ · A3 ✅ View · A4 ✅ · A5 ✅.
- *Powerful.* SVG generated from R2's cached result; a bar chart is a few hundred bytes of markup.
- *Suggested ID.* **`CAP-V23`** chart section (depends on `CAP-V22`).

**R4 — Build `CAP-X10` (metadata-driven expression indexes) instead of table-per-object** (G4)

- *What.* The registered-but-deferred row: derive B-tree expression indexes
  (`(data->>'fld_x')`, typed casts for dates/numbers) from the fields that Views filter/sort on
  and that R2 datasets group by; reconcile on metadata load.
- *Basis.* ObjectStack's table-per-object buys indexes at the price of a DDL/drift subsystem
  (`schema-drift.ts`, `os migrate`) and a 17k-line driver. Postgres expression indexes over JSONB
  give the same plans for the fields that matter, with zero schema migration per Machine — the
  standard advice for "schemaless with hot keys" (PostgreSQL docs §"Indexes on Expressions";
  Study 8's own recommendation). Keep `GIN(data)` for containment.
- *Pre-screen.* Already registered (Study 8, ADR-003); this study adds a comparator that chose
  the other path and paid for it. The row's own "not until measurably slow" caveat stands —
  **R2/R3 dashboards are what will make it measurably slow**, so sequence R4 right after R2.
- *Powerful.* This is the single biggest "stay fast on a small VPS" item in the list.

### Tier 2 — permissions and org model (ERP-grade access on one shared VPS)

**R5 — Scope-depth on CRUD/Event permissions** (G5) and **R6 — Organizational unit tree +
manager designation** (G6). Two rows (A3), one build.

- *What.* `CAP-O01` users gain an optional `unit` (reference to a workspace-level Unit tree —
  a built-in Machine like Groups, `CAP-O07`) and an optional `manager` (user reference).
  Every `CAP-P05`/`CAP-P01` permission gains `scope: own | own_and_reports | unit |
  unit_and_below | all` (default `all`, so nothing changes until declared).
- *Basis.* ObjectStack ADR-0057 surveys Salesforce / Dataverse / ServiceNow / SAP / NetSuite and
  finds scope depth is the axis they all share and "the single highest-leverage ERP feature";
  Dataverse's own model (user / business unit / parent-child BU / organization) is the reference
  shape. `CAP-X09` was dissolved here (v0.41) into ownership/groups/selectors — this study is the
  second independent source saying the *permissions* half of that dissolution is still an open
  gap, and gives it a concrete, cheap form.
- *A1 evidence.* Map: ADR-0057 + Dataverse/Salesforce. Terrain: Case 18 (HR: a manager sees their
  reports' leave requests), Case 20 (a department head sees the department's appointments), Case
  17 (helpdesk team queues).
- *Pre-screen.* R5: A1 ✅ · A2 ✅ · A3 ✅ Permission · A4 ✅ (`CAP-O07` groups are flat and
  role-bound; they can't express "and below") · A5 ✅ "A Manager may read Leave Requests of their
  reports". R6: A3 ✅ Workspace Services · A5 ✅ "Employees belong to a Unit; Units have a parent".
- *Powerful.* Resolve `unit_and_below`/`own_and_reports` **once per session** into an id set
  (BFS over a tree of hundreds, not millions) and apply it as `created_by = ANY($1)` /
  `data->>'owner' = ANY($1)` — ADR-0055/0057's "pre-resolved IN-sets, never a subquery" — same
  cost class as today's ownership check.
- *Suggested IDs.* **`CAP-P08`** scope depth; **`CAP-O12`** unit tree + manager designation.

### Tier 3 — approvals (Menata is already strong here; these are enrichments)

| Rec | What | Basis / evidence | Pre-screen | ID |
|---|---|---|---|---|
| **R7** | Approver resolver `manager_of: <user field>` and `unit_of: <user field>` (+ `and_below`) on `CAP-F23`-style pickers and on Overlay `requirements[].actor` | ObjectStack `ApproverType.manager/department`; Case 18/20; depends on R6 | A1 ✅ A3 ✅ Permission (actor resolution) A5 ✅ "approved by the requester's manager" | **`CAP-P09`** |
| **R8** | `CAP-W03` Tier 2: quorum `EACH_GROUP` (one approval from every named group) | `behavior: per_group`; Case 7 (finance *and* legal) | tier on existing row | `CAP-W03` T2 |
| **R9** | `CAP-W04` Tier 2: `on_breach.action: notify | escalate | auto_approve | auto_reject | reassign` | `ApprovalEscalationSchema.action`; Case 7 SLA, Case 17 helpdesk SLA | tier on existing row; keep the business calendar (`CAP-O06`) — ObjectStack lacks it | `CAP-W04` T2 |
| **R10** | Revision loop: a `return_for_revision` requirement outcome with `max_revisions` and a distinct `Returned` state the Overlay compiles (Returned → editable → Resubmit re-enters the same requirement, round N+1) | ADR-0044 (`returned` ≠ `rejected` ≠ `recalled`; cap auto-rejects); Case 3 (document sent back), Case 7 | A1 ✅ A3 ✅ Workflow (W) A4 ⚠️ expressible by hand today *without* the counter/cap — the cap is the capability A5 ✅ | **`CAP-W09`** |

### Tier 4 — automation reach and observability

| Rec | What | Basis / evidence | Pre-screen | ID |
|---|---|---|---|---|
| **R11** | `call_webhook` action: outbound HTTP POST of an event payload (`CAP-I02` schema) to a declared URL, routed through the `CAP-W06` outbox (retry/backoff), with a signed `X-Signature` header; credential stored encrypted (this is where a `secret`-style option lands, G18) | ObjectStack `http` node + `plugin-webhooks`; Study 5 Portal GA integration patterns (`CAP-I01–I04`); Case 8 payment, Case 15 e-commerce fulfilment — today only *inbound* exists (`CAP-E04`) | A1 ✅ A2 ✅ universal A3 ✅ Action A4 ✅ A5 ✅ "notify the courier system" | **`CAP-A16`** |
| **R12** | `CAP-I04` second half: an admin "trace" View listing every `record_events` row sharing one correlation id, with the outbox rows it spawned | `sys_automation_run` + run summaries; the correlation id already exists | tier on existing row | `CAP-I04` → ✅ |

### Tier 5 — field and view palette (cheap, case-driven)

| Rec | What | Evidence | Pre-screen | ID |
|---|---|---|---|---|
| **R13** (G14) | `value_list` `multiple: true` (stored as JSON array, rendered as checkboxes/tags, filterable with "contains") | `multiselect`; `CAP-F03`'s own Case 13 note; Case 11 hashtags, Case 17 ticket tags | tier on existing row | `CAP-F03` T2 |
| **R14** (G15) | Constraint operators `format: email | url | phone` | `format` rule; Case 11/12/15 (user email/URL) | A3 ✅ Constraint A5 ✅ "Email must be a valid address" | **`CAP-C14`** |
| **R15** (G16) | `percent` field (display + 0–100 bound) | `percent`; Case 8/14 rates, Case 15 discount | A4 ⚠️ composable from `number` + constraint; admit only as display sugar | `CAP-F07` T2 |
| **R16** (G17) | `reference.options.display_field` | `nameField` ADR-0079; `CAP-F13` row names it | tier on existing row | `CAP-F13` T2 |
| **R17** (G19) | New lenses, one row each, admitted only on case pressure: **gantt** (Case 19 Trello-like already declares boards; a project timeline is its natural next target), **gallery** (Case 11 Instagram-like, Case 15 catalog), **tree** (R6's Unit tree, Case 18 org chart), **map** (no case yet — hold) | `VisualizationTypeSchema`; `case-portfolio.md` Cases 11/15/18/19 | A1 per lens | `CAP-V24`–`V26` |
| **R18** (G20) | Personal saved views: per-user filter/sort/column persistence on a list View | `type: 'personal'`; every platform in Study 2 | A3 ✅ View A5 ✅ "my view of Tickets" | **`CAP-V27`** |
| **R19** (G21) | `CAP-V12` Tier 2: wizard steps that span Machines (step 2 creates a record referencing step 1's) | `screen` flow `objectName`/`idVariable`; Case 3 (request + attachment), Case 19 (board + first list), Case 20 (patient + appointment) | tier on existing row | `CAP-V12` T2 |
| **R20** (G22, G23) | Form sections/columns; column footers; XLSX export | presentation; `xlsx-module.ts` | low; bundle into a design pass | `CAP-V01` T2, `CAP-R06` T2 |

### Tier 6 — platform surfaces

**R21 — Governed MCP surface** (G25)

- *What.* An MCP server endpoint over the existing `CAP-X07` JSON API: one tool per Machine
  (`list`, `get`, `create`) and one per Event declared `ai_exposed: true`; every call runs through
  the same `permission.Guard`, constraints and executor as the UI, under a session or an API key
  bound to a user.
- *Basis.* ObjectStack's headline (ADR-0011, ADR-0109, design principle IV) — and the reasoning is
  sound independent of ObjectStack: an AI operating a business app must act through the same
  typed, permission-checked surface as a human, never through raw SQL or scraped UI. MCP is the
  de-facto open standard for that surface (adopted across the major model vendors in 2025).
  `002-architecture.md`'s "runtime owns behavior" principle makes this a *thin* adapter, not a
  new engine.
- *Pre-screen.* A1 ⚠️ map only (ObjectStack + the MCP ecosystem); no portfolio case yet names an
  AI operator — record as Proposed pending a case (e.g. Case 17 helpdesk triage assistant) · A3 ✅
  Cross-Cutting · A5 ⚠️ the *exposure flag* is business language ("this event may be used by an
  assistant"); the protocol itself is runtime internals.
- *Powerful.* Zero extra process; JSON over the existing handlers.
- *Suggested ID.* **`CAP-X16`**.

**R22 — Re-scope `CAP-W02` to ObjectStack ADR-0009's narrow form** (G26)

- Study 20 §6.4 rejected blanket pinning because it multiplies the metadata cache per live
  version. ADR-0009 pins *only* execution-bearing definitions (the Overlay's compiled Events/
  requirements for a Machine) by content hash, keeps old hashes resolvable, never GCs them — and
  `CAP-W07`'s `records_in_states` policy already decides *which* records stay on the old
  definition. The two compose: W07 is the policy, a narrow W02 is the mechanism that makes
  "keep evaluating against the old version" actually true for records that W07 leaves behind.
  Recommend updating the `CAP-W02` row's note with this design; still ❌, still no case pressure
  beyond Case 7's long-running approvals.

**R23 — Email transport (`CAP-O10`) is the prerequisite for four other rows** (G27, G28, G10's
actionable links, R9's escalation). Both projects deferred it; ObjectStack's ledger rated it P0.
No new capability — but this study is the second source saying the transport decision named in
Study 35 §5.6 is now on the critical path.

## 4. Where Menata Runtime is ahead — keep, don't dilute

| Menata capability | ObjectStack state | Why it matters |
|---|---|---|
| Metadata-only, no code hatch | hooks (TS), hook bodies (sandbox), `functions`, jobs, React pages | the "AI holds the whole app" promise is only true when there is no code outside the metadata diff |
| Conformance-gated registry (✅ = proven) | spec surface ahead of enforcement; enforce-or-remove ADRs cleaning up years later (§H7) | every ObjectStack "declared but inert" incident is a Menata registry row that would have stayed ❌ |
| Process Overlay + decompile (`CAP-W01/W05`) | DAG engine with 21 node types; state machine as a separate rule | Study 20's server-economy measurement; one grammar, enforced since Case 1 |
| Business calendar behind SLAs (`CAP-O06` + `CAP-W04`) | escalation is wall-clock only (`approval.zod.ts:633`) | Indonesian public-holiday-heavy SLAs |
| Effective-dated change policy (`CAP-W07`) | draft/publish + pinning | compliance changes *should* reach open cases; W07 says which |
| Separation of duties (`CAP-P03`), delegation (`CAP-P04`) | expressible, not first-class / partial | audit-grade approvals |
| PDF signature compositing + placement + stepper (`CAP-F22/V21/V20`) | `signature` field only | document-centric approval (Case 3) |
| Idempotent inbound webhooks (`CAP-X13`), cross-record atomicity (`CAP-X12`) | ADR-0034 fixed a deadlock; webhook idempotency only outbound | correctness under retries |
| Group-restricted approver pickers (`CAP-F23`) | position/team resolvers | ✓ |
| Server-rendered, zero client framework, single 21k-line binary | React SPA + 790k-line runtime + cluster primitives | the owner's "powerful" criterion, literally |

## 5. What not to copy, and why

| ObjectStack choice | Don't copy because |
|---|---|
| Microkernel + runtime plugin DI | Menata's compile-time registry seams (`capability-lifecycle.md` §4) give the same extension points with no plugin-order/signature/health machinery; Go's type system is the DI |
| TypeScript authoring | The Authoring Layer's audience is domain experts via `.menata` (`003-runtime-language.md`); Zod-in-TS optimises for coding agents' priors, the opposite audience. Take the *validation* ideas (H5), not the medium |
| React SPA console | ADR-001 (`prototype/go/docs/decisions/001-techstack.md`) already decided; HTMX + Hyperscript covers every view in the portfolio |
| DAG flow engine, `subflow`, `parallel_gateway`, `boundary_event` | Studies 19–22 measured and chose emergent + Overlay; CMMN composition proven (Study 22); borrow *specific actions* (R11) not the engine |
| Table-per-object storage + drift subsystem | R4 gets the query plans without the DDL surface |
| 50 field types wholesale | Study 15's decision framework: composition over new types; admit only R13–R16 on case pressure |
| Code hatches (hooks/functions/script node/React pages) | violates Metadata First; R1's fail-closed expression scope is the maximum |
| Three tenancy postures with enterprise-gated walls | Workspace-per-schema + RLS is simpler, always-on and open (`CAP-X06`) |
| Cluster primitives / Redis | no multi-instance target (`app/ARCHITECTURE.md`); `CAP-X11` stays ❌ |

## 6. A discipline to adopt regardless of capabilities

`runtime-metadata-schema.md` §"Load-Time Contract — What's Enforced, What Silently No-ops" is the
best document of its kind in either repo — and it is a *list of things the loader accepts and
ignores*. ObjectStack's ADR-0049 / ADR-0078 went one step further: every declared-but-unread key
is a load-time rejection carrying the prescription ("`softDelete` was removed in 16.0 — use
`managedBy`"). Recommend a small ratchet in `internal/metadata/validate.go`: unknown keys and
known-no-op options are errors, with the message naming the capability row that would make them
real. It costs nothing at runtime and closes the exact gap (H7) that cost ObjectStack a
year of enforce-or-remove ADRs. No registry row — this is `CAP-X05`'s own definition of done.

## 7. Suggested order for upcoming sessions

**Superseded (2026-09-07, same day):** `second-opinion-reconciliation.md` §5 carries the current
order, after six additions (R24–R29) and one re-prioritisation (R21). The list below is kept as
written, per the append-don't-rewrite convention.

1. **R23 → decide the email transport** (`CAP-O10`) — it gates R9/R10 links, `CAP-V11`, and real
   `CAP-A10`. Both projects paid for deferring it.
2. **R1 expression layer** (`CAP-F14` ✅ + `CAP-C13`) — highest leverage; cel-go; fail-closed scope.
3. **R2 + R3 + R4 together** (`CAP-V22` dataset → `CAP-V23` chart → `CAP-X10` indexes) — one
   analytics pass with a measured before/after on Case 9 and Case 15 data volumes.
4. **R6 + R5** (`CAP-O12` unit tree → `CAP-P08` scope depth), then **R7** (`CAP-P09`) — one
   permissions pass, Case 18/20 as the proof cases.
5. **R8/R9/R10** approval enrichments as Tier-2 notes on `CAP-W03`/`W04` + `CAP-W09`.
6. **R11** `CAP-A16` outbound webhook (needs a `secret`-style credential option — design once).
7. **R13–R19** as their cases arrive; **R21** MCP when a case names an assistant.
8. §6 discipline ratchet — can land in any session, alongside whatever capability touches
   `validate.go` next.

Admission of any of these is the owner's call; this document is the evidence, not the decision.
