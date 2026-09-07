# ObjectStack vs Menata Runtime — Capability Comparison

> Part of Study 37 (`../README.md`). ObjectStack facts cite files in the clone at commit `ac76425f`
> (2026-09-07); Menata facts cite `capability-registry.md` v0.54 rows (`CAP-*`), `app/` source, or
> `runtime-metadata-schema.md`. ✅/⚠️/❌ in Menata columns are the registry's own statuses; for
> ObjectStack, "declared" means the Zod schema exists and "enforced" means a runtime reader was
> found or an ADR/test asserts it — the distinction matters (§H7).
> Status: v1.0 | Created: 2026-09-07 | Updated: 2026-09-07

---

## 0. Vocabulary map

| Menata Runtime (`006-runtime-model.md`) | ObjectStack | Notes |
|---|---|---|
| Workspace | Organization (data tenancy, `organization_id`) / Environment (deployment identity) | ObjectStack splits the two (ADR-0006); Menata's Workspace is both |
| Application | App (`App.create` — navigation shell) + Package (`defineStack` manifest, `com.example.crm`) | Menata `CAP-O03` navigation ≈ `app.navigation[]`; `CAP-X08` export ≈ package artifact |
| Machine | Object (`ObjectSchema.create`) | 1:1 |
| Field | Field | 14 vs 50 type keywords (§1) |
| Status `value_list` + state-conditional Events (`CAP-E06`) | `select` field + `state_machine` validation rule (`transitions: {from: [to]}`) | ObjectStack enforces the transition table on write (ADR-0020); Menata enforces via Event availability + `activate_next`/Overlay guards |
| Event (`When X`) + Actions | `record_change` Flow (start node + nodes), or `hook`, or `action` (UI button → flow) | Menata has no separate "button" concept — an Event *is* the button (`CAP-P04`'s `input_fields` ≈ action `params`) |
| Event source `schedule` / date-driven / webhook (`CAP-E02–E04`) | Flow types `schedule`, time-relative trigger, `api` flow / webhook plugin; `defineJob` | ObjectStack's time-relative trigger = Menata `CAP-E03` |
| Constraint | Validation rule (`script`, `cross_field`, `unique`, `format`, `json_schema`, `state_machine`, `conditional`) + field `required/min/max` | Menata operators are a closed set; ObjectStack conditions are CEL |
| Permission (role-based event/CRUD/field, `CAP-P*`) | Permission set (object CRUD + FLS) bound to Positions; OWD `sharingModel`; sharing rules; RLS policies; scope depth | §5 |
| Role (`CAP-O01`) / Group (`CAP-O07`) | Position (flat, ADR-0090 D3) / Team (`sys_team`) + Business Unit tree (`sys_business_unit`) | ObjectStack has the org tree Menata's `CAP-X09` dissolved |
| View (`form/list/detail/dashboard/calendar/timeline/report/document/process_map/board/coord_placement/decision_stepper`) | View (list types × 9, form types), Page, Dashboard, Report | §2, §6 |
| Process Overlay (`process` block, `CAP-W01/03/04/05`) | Flow with `approval` nodes + `state_machine` rule | opposite architectural bet (Study 20) |
| `record_events` (`CAP-R04` ⚠️) | `sys_audit_log` (plugin-audit) + `sys_automation_run` + approval audit objects | ADR-0052: "audit is not the activity feed" — two separate things |
| Notification (`CAP-A03/A04/A10/O05`) | Notification platform: ingress → outbox → preferences/quiet hours/digest → channels (ADR-0012/0030) | both have in-app; both lacked real email for a long time (§H3) |

## 1. Field types

ObjectStack's `FieldType` enum (`packages/spec/src/data/field.zod.ts:40`) has 50 members. Grouped,
against Menata's 14 (`runtime-metadata-schema.md` §"Field Types", `CAP-F*`):

| Group | ObjectStack types | Menata equivalent | Gap? |
|---|---|---|---|
| Text | `text`, `textarea`, `email`, `url`, `phone`, `password`, `secret` | `text`, `rich_text` | **email/url/phone** are text + a format validator — no `format` constraint operator exists (`CAP-C*` has none); `secret` (encrypted-at-rest, masked-on-read, ADR-0100) has no equivalent — needed the day a webhook/connector credential must live in metadata |
| Rich content | `markdown`, `html`, `richtext` | `rich_text` (textarea) | presentation only; no gap for cases so far |
| Numbers | `number`, `currency`, `percent` | `number`, `money` (`CAP-F08`, with `currency`/`currency_field` — richer than ObjectStack's `currency` scale/format) | **percent** missing (Case 8/14 interest rates, Case 15 discounts use `number` today) |
| Date/time | `date`, `datetime`, `time` | `date`, `date_time`, `time`, `duration` (`CAP-F10`) | Menata has `duration` (integer minutes) — ObjectStack has none; ADR-0053 fixes date-vs-datetime semantics the way `CAP-F10` did |
| Logic | `boolean`, `toggle` | `boolean` (`CAP-F09`) | none (`toggle` is a widget) |
| Selection | `select`, `multiselect`, `radio`, `checkboxes`, `tags` | `value_list` (`CAP-F03`, **single-select only** — its own row names the Case 13 multi-value gap, worked around with comma-separated text) | **multi-select** is the one real gap; radio/checkboxes are widgets |
| Relational | `lookup`, `master_detail` (`sharingModel: 'controlled_by_parent'`, cascade, `inlineEdit: 'grid'`, ADR-0035/0055), `tree` | `reference` (`CAP-F13`, self-reference proven), line items (`CAP-F16`, own Machine with back-reference), M:N (`CAP-F20`) | Menata expresses master-detail as composition rather than a type — equivalent; ObjectStack's **`inlineEdit: 'grid'` (child rows edited inside the parent form, saved atomically)** ≈ `CAP-V15` live line rows + `CAP-X12` — already covered. **Display-field designation** for a lookup: ObjectStack `nameField`/`displayNameField` (`object.zod.ts:2121`, ADR-0079); Menata uses a heuristic ("a `text` field named Name, else first text field") its own `CAP-F13` row flags as not-yet-a-capability |
| User | `user` (lookup to `sys_user`, single or multiple) | `user` (`CAP-F05`), group-restricted picker (`CAP-F23`) | Menata is ahead on scoping the picker; ObjectStack allows *multiple* users (watchers/collaborators) — Menata would compose via `CAP-F20` |
| Media | `image`, `file`, `avatar`, `video`, `audio` | `file` (`CAP-F06`, real upload + server-side image resize/WebP) | none real — ObjectStack's own template gap list (`docs/PLATFORM_GAPS_FROM_TEMPLATES.md` #3) says the upload UI was *unverified* as of 2026-05; Menata's is conformance-proven (T130/T131) |
| Calculated | `formula` (CEL), `summary` (server-side rollup of child rows, recomputed on child insert/update/delete), `autonumber` | `computed` (`CAP-F14` ⚠️, one multiply), aggregate via `CAP-A14`/`CAP-C10`, `CAP-F18` autonumber | **formula** is the headline gap (§R1 in gap doc); `summary` ≈ `CAP-A14` `aggregate_condition` — Menata computes at event time, ObjectStack maintains a stored rollup |
| Embedded | `composite`, `repeater`, `record` (JSON on the parent row, no separate table) | none — Menata always uses a child Machine (`CAP-F16`) | deliberate: Machine First. A `repeater` would be schema-less rows inside a record — Study 22 already ruled schema-less case content a permanent non-goal |
| Enhanced | `location`, `address`, `code`, `json`, `color`, `rating`, `slider`, `signature`, `qrcode`, `progress` | `CAP-F22` PDF signature compositing (different thing: signs an uploaded PDF), `CAP-V21` coordinate placement | `location`/`address` have no case pressure yet in `case-portfolio.md` (Case 5 WMS uses bin codes); `json` is a schema-less escape hatch — non-goal |
| AI | `vector` | none | out of scope until an AI surface exists |

**Field options worth noting**: `searchable` (feeds `object.searchableFields`, ADR-0061),
`unique: 'organization' | 'global'` (ADR-0120), `readonly`, `defaultValue` incl. context tokens,
`indexed`, `min/max/maxLength`, select `options[].color` (Menata renders `value_list` as a badge
but colours are not metadata).

## 2. Views

### 2.1 List / record views

| Aspect | ObjectStack (`packages/spec/src/ui/view.zod.ts`, 5,368 lines) | Menata Runtime | Gap? |
|---|---|---|---|
| List lenses | `VisualizationTypeSchema`: `grid`, `kanban`, `gallery`, `calendar`, `timeline`, `gantt`, `map`, `chart`, `tree` (9) | `list`, `board` (kanban, `CAP-V14` Tier 2 drag-drop), `calendar` (+ resource-grouped `CAP-V18`), `timeline`, `report`, `dashboard`, `process_map`, `document`, `coord_placement`, `decision_stepper` (`internal/model/model.go:610–660`) | **gantt, gallery, map, tree** missing; Menata has four lenses ObjectStack lacks (`process_map`, `document`, `coord_placement`, `decision_stepper`) |
| Columns | per-column `width/align/pinned/format/summary` (`ColumnSummarySchema` — column footers), row height, grouping with aggregate headers, row colour rules, conditional formatting | `columns[]` field ids; `default_sort` (`CAP-V04`); SLA countdown badge (`CAP-V17`) | column summaries/footers and row-colour rules are cheap wins with no registry row |
| Filters | `filterableFields`, filter UI with and/or + operator set, 36 relative-date macros (`{current_quarter_start}`, `{1_years_ago}`), `$search` server-resolved (ADR-0061) | declarative view filter (`CAP-V09`: Due Today, Overdue), list search & filter (`CAP-V08`), typeahead (`CAP-V16`), workspace search (`CAP-O04`) | relative-date *macros* exist in Menata only as fixed conditions (`today`, `+1 Month` arithmetic) — an expression layer (§R1) subsumes this |
| Saved views | `type: 'personal' | 'collaborative'` (`view.zod.ts:980`), `client.views.*` create/share; per-user "My records" resolves via permission model | `CAP-V05` filtered lists are *metadata-declared*; no per-user saved view | **personal saved views** missing |
| Navigation mode | `page`, `drawer`, `modal`, `split`, `popover`, `new_window`, `none` (`view.zod.ts:1515`) | page only (HTMX partials) | UX-level; drawer/modal are feasible server-rendered |
| Toolbar toggles | `UserActionsConfigSchema` (sort/filter/group/hideFields/rowColor/export/import/bulk) | fixed | none real |
| Export | `exportOptions.formats: ['csv','xlsx','json']` (`packages/rest/src/xlsx-module.ts`) | CSV (`CAP-R06`) | **XLSX** missing — small |
| Bulk actions | `bulk-action.zod.ts` (declared); template gap list #17 says bulk UI "exists but not end-to-end verified" | `CAP-A15` batch generation is server-side; no bulk *UI* selection | evidence-thin on both sides |

### 2.2 Forms

| Aspect | ObjectStack | Menata Runtime | Gap? |
|---|---|---|---|
| Layout | `formViews.<name>.type: simple | tabbed | wizard`; `sections[]{label, columns: 1..n, fields[]{field, required, readonly, visible}}`; separate create vs edit layouts (`content/docs/ui/create-vs-edit-form.mdx`); ADR-0050 layout vs presentation | `form` View `fields[]` (`CAP-V01`); wizard as `Steps [][]string` (`CAP-V12`, stateless — the browser is the state) | **sections / multi-column** layout missing (a form is one column of fields); conditional show/required is missing on *both* (ObjectStack template gap #8) |
| Public forms | anonymous entry routes scoped to an object slice; guest permission set INSERT-only (`examples/app-crm/src/security/sales-positions.ts` `GuestPortalProfile`) | `CAP-P07` public/unauthenticated role | equivalent |
| Screen flows | multi-object wizard as a `screen` Flow with `object-form` steps, each persisting a record and binding its id (`convert-lead.flow.ts`) | `CAP-V12` is single-Machine | **multi-Machine wizard** (create A then B linked to A) is a real gap — Case 3/19/20 patterns hit it |
| i18n | translation bundles per locale (`translations/crm.translation.ts`), `service-i18n`, label resolver | none — labels are the metadata `name` | hold until a case names a second language |

### 2.3 Pages, apps, dashboards, reports — see §6 for analytics; here the shell:

| Aspect | ObjectStack | Menata Runtime |
|---|---|---|
| App shell | `App.create({ navigation: [{type:'group', children:[{type:'object'|'dashboard'|'page'|'url'}]}], branding: {primaryColor} })` | `CAP-O03` navigation metadata, Tiers 2–3 (sub-nav, within-Machine auxiliary Views), ADR-008 mobile-first shell, Studies 29–31 |
| Pages | free-form widget layouts; React pages (parsed JSX, ADR-0080) | `CAP-V10` composed dashboard View is the only free-form surface |
| Setup/admin | the Setup app is itself rendered from the protocol | `/admin/*` handlers (Templ) — hand-written, not metadata |
| Doc pages | package docs as metadata rendered in the console (ADR-0046) | none |

## 3. Workflow and automation

### 3.1 The two models

**ObjectStack** (`content/docs/automation/flows.mdx`, `packages/spec/src/automation/flow.zod.ts`,
`packages/services/service-automation`): a Flow is a DAG of typed nodes joined by edges;
`type: autolaunched | record_change | schedule | screen | api`. 21 built-in node types
(`FlowNodeAction`, `flow.zod.ts:27`): `start, end, decision, assignment, loop, create_record,
update_record, delete_record, get_record, http, notify, script, screen, wait, subflow, map,
connector_action, parallel_gateway, join_gateway, boundary_event` (+ plugin-contributed
`approval`, `approval_revise`). Edges carry a CEL `condition`, `isDefault`, `type: default |
fault | conditional | back`. Runs are durable: an `approval`/`screen`/`wait` node suspends the run
into `sys_automation_run` + a suspended-run store and resumes on decision/input/timer, with
resume-authority gates ("who may resume — the gate is the suspended node"). Structured control
flow (ADR-0031): loop containers, parallel blocks, try/catch/retry; `errorHandling.strategy:
fail | retry | continue` with backoff; run summaries and a Console flow viewer/test runner.
Separately: `state_machine` validation rule enforced on the write path (ADR-0020), TypeScript
`hooks`, `defineJob` cron functions, `plugin-webhooks` outbound fan-out of `data.record.*` events,
time-relative triggers ("30 days before contract expiry", ADR-0041), BPMN interop schema.

**Menata Runtime**: no engine, no DAG. A Machine's Events (`When X`) carry Actions (`set_field`,
`notify`, `create_record`, `activate_next`, `aggregate_status`, `trigger_event`,
`cross_set_field`, `batch_generate`, `composite_pdf_signature` — `model.go:434–447`) with
conditional actions (`CAP-A09`), Constraints, and Permissions; state-conditional Event
availability (`CAP-E06`) *is* the state machine; sequential and parallel multi-record processes
emerge from `activate_next` + `aggregate_status` across records (`CAP-A07/A08`); schedule /
date-driven / webhook / internal sources (`CAP-E02–E05`); cross-Machine subscriptions with
schemas and contracts (`CAP-I01–I03`); async outbox for slow actions (`CAP-W06`); and the
**Process Overlay** — a `process` block (states, requirements with cardinality, SLA, quorum) that
`compile.go` expands into ordinary Events/guards/Permissions (`CAP-W01/W03/W04/W05`), with the
reverse "decompile-lift" proven too (Study 27). Study 20 measured this at ~3–6 declared
statements per transition vs ~10–13 for a DAG-style comparator and chose it deliberately.

### 3.2 Feature-by-feature

| Capability | ObjectStack | Menata Runtime | Gap? |
|---|---|---|---|
| Record-change trigger with old/new comparison ("amount crossed 500k fires once") | `record_change` flow, `triggerType: record-after-update`, condition over `record`/`old` | `CAP-E01` + Event `condition` (`CAP-A09`); before-snapshot exists in `record_events` but conditions can't reference the *previous* value | **old-value predicate** missing — subsumed by §R1 expression layer (`old.amount`) |
| Schedule | `schedule` flow → `IJobService` (leader-elected in a cluster); `defineJob` for code | `CAP-E02` | equivalent |
| Date-relative ("30 days before expiry") | time-relative trigger (declarative sweep, ADR-0041) | `CAP-E03` (`When Due Date - 1 Day`) | equivalent — Menata got there first (Case 7) |
| Manual/button | `action` (`type: flow | script | url | modal | api | form`, `locations`, `visible` CEL, `params`) | an Event with `permissions` + `input_fields` (`CAP-P04`) | equivalent; ObjectStack's `visible` predicate ≈ `CAP-E06` state-conditional availability, but general |
| External call-in | `api` flow, webhook receiver | `CAP-E04` + `CAP-X13` idempotency | Menata ahead on idempotency (ObjectStack's is in the outbound plugin) |
| **Outbound HTTP / webhook** | `http` node (outbox-backed), `plugin-webhooks` fan-out, connector actions | **none** — no action type performs an outbound request (`grep http.NewRequest internal/executor` → nothing) | **real gap** (§R9) — the only way today to reach another system is `CAP-I01` subscriptions *inside* the workspace |
| Branching | `decision` node + edge conditions; default edge | `CAP-A09` `if` inside events; state-conditional events | equivalent for single-record logic |
| Parallel / join | `parallel_gateway` / `join_gateway` | `CAP-A08` `aggregate_status` across child records (parallel approval), `CAP-W03` quorum | equivalent by composition (Study 22 §CMMN) |
| Loop / batch | `loop`, `map` (per-item subflow, each may pause) | `CAP-A15` batch generation (create N records from a formula) | ObjectStack's loop is general; Menata's is "generate N" — no case has needed "for each existing record do X" yet |
| Wait / timer inside a process | `wait` node (duration or until date), boundary timer events | `CAP-E03` date-driven Event on a field, `CAP-W04` SLA due-date field | equivalent (Menata's form is "declare the date, react to it", not "block here") |
| Human step (screen) | `screen` node with `waitForInput`, object-form screens | Event `input_fields` (`CAP-P04`), wizard `CAP-V12` | multi-Machine screen flow missing (§2.2) |
| Sub-process reuse | `subflow` node | none — the Overlay compiles per Machine; cross-Machine chains via `create_record` + `CAP-I01` | no case pressure for reusable sub-processes; Study 22 covers CMMN composition |
| Error handling / retry | per-flow `errorHandling` (`fail|retry|continue`, backoff, jitter), fault edges, try/catch | `CAP-W06` outbox retries slow actions; synchronous actions fail the request atomically (`CAP-X12`) | equivalent at the level cases need |
| Run observability | `sys_automation_run`, run summaries, Console viewer, per-node outcome | `record_events` + request-id correlation (`CAP-I04` ⚠️ half) | **automation trace UI** — the missing half of `CAP-I04` |
| State machine | `state_machine` validation rule — write-path enforced (ADR-0020: previously three shapes, zero enforcement) | `CAP-E06` + Overlay states, `CAP-R07` immutability after state, `CAP-R08` scratch state, `CAP-W05` process map | equivalent; Menata's has been enforced since Case 1 |
| Approvals | see §4 | see §4 | — |
| Notifications | `notify` node → messaging service (channels, templates `email-template.zod.ts`, quiet hours, digest — ADR-0012/0030) | `notify` action (`CAP-A03/A04`), inbox + preferences + digest (`CAP-O05`), channels (`CAP-A10`, in-app real, email deliberately not faked) | equivalent in-app; both lack proven email (§H3) |
| Code | `script` node → registered function; hooks | none, by principle | not a gap |
| Metadata pin for in-flight runs | `executionPinned` types (ADR-0009) | `CAP-W07` change policy (effective-dated), `CAP-W02` ❌ | §H2 |

## 4. Approvals

ObjectStack: `packages/spec/src/automation/approval.zod.ts` (838 lines) + `plugin-approvals`
(ADR-0019 "approval as a flow node", ADR-0042 SLA escalation, ADR-0043 actionable links,
ADR-0044 send-back). Menata: approvals are records + Events (Case 3 Document Approval, Case 7
Multi-step Approval, Study 25 quorum, Study 32 PDF signature).

| Feature | ObjectStack | Menata Runtime | Gap? |
|---|---|---|---|
| Approver resolution | `ApproverType` (`approval.zod.ts:31`): `manager` (submitter's `sys_user.manager_id`), `position`, `department` (+ descendants), `team`, `field` (user id on the record), `expression` (CEL over `current.*/trigger.*/vars.*`), `org_membership_level`, `user`; `queue` deprecated *because it was never enforced* (#3508) | role (`CAP-P01`), record `user` field (`CAP-P02` owner / `CAP-A04` dynamic recipient), Group-restricted picker (`CAP-F23` + `CAP-O07`), per-step user/group approver (Case 3 extension, `cfd14fa`) | **manager-chain** missing (needs a "manager" designation on users — Case 18 has Employee↔Manager on a Machine, not on `CAP-O01` users); **department subtree** missing (no org tree, §5) |
| Decision behavior | `behavior: first_response | unanimous | quorum (minApprovals, clamped) | per_group`; one rejection vetoes in every mode | `CAP-A08` all-approved/any-rejected (= unanimous + veto), `CAP-W03` `ANY` / `N_OF_M` (= first_response / quorum) | **`per_group`** ("finance *and* legal must each approve") missing — a Tier 2 on `CAP-W03` |
| Multi-level ladder | successive `approval` nodes; routing by amount via edge conditions | `CAP-A07` `activate_next` sequential steps; conditions via `CAP-A09` | equivalent |
| Record lock while pending | `recordLock` (configurable) | `CAP-R07` immutability after state (`posted`/`submitted` frozen) | equivalent |
| Status mirror on the record | `approvalStatusField` (readonly select the runtime writes) | the Status field *is* the process state | equivalent (Menata's is simpler) |
| SLA / escalation | `escalation: { timeoutHours (wall-clock — "the platform ships no business-hours calendar"), action: reassign | auto_approve | auto_reject | notify, notifySubmitter }` (`approval.zod.ts:597–646`) | `CAP-W04` `process.sla[]{state, duration, on_breach{notify, escalate_to}}` computed against `CAP-O06` **business calendar** (holidays, working days) | Menata ahead on the calendar; behind on **breach actions** (`auto_approve`/`auto_reject`/`reassign` not expressible — `model.go:97` `ProcessOnBreach` has notify + escalate only) |
| Send back for revision | `approval_revise` node, `revise` out-edge, back-edge re-entry, `maxRevisions` (default 3) auto-rejects beyond budget, terminal status `returned` distinct from `rejected`/`recalled` (ADR-0044) | expressible as Reject → Draft → Submit again (Case 3), but no revision counter, no distinct `returned` state, no cap | **revision loop with cap** missing — small but real (Case 3/7 pressure) |
| Recall by submitter | `recall` verb, `recalled` status | an Event on the submitter's role (Case 3 `Cancel`) | equivalent |
| Delegation / out-of-office | `sys_approval_delegation` object; template gap #20 says community edition has partial delegation | `CAP-P04` delegation ✅ | equivalent or Menata ahead |
| Actionable links | one-click approve/reject via signed token in email (`sys_approval_token`, `action-link-pages.ts`, ADR-0043) | none — depends on email transport (`CAP-O10` ❌) | blocked on email, not on approvals |
| Inbox | approvals inbox for batch processing; `sys_approval_request/action/approver` audit trail; payload redaction for approvers who may not read every field | `CAP-V05` "pending my approval" filtered list, `CAP-O05` notification center, `record_events` | equivalent; **payload redaction ≈ `CAP-P06`** field visibility already applies |
| Separation of duties | not a first-class concept (expressible via CEL on approver expression) | `CAP-P03` ✅ | Menata ahead |
| Signature | `signature` field (draw a signature) | `CAP-F22` PDF signature compositing + `CAP-V21` placement + `CAP-V20` stepper | Menata ahead for document-centric approval |
| Approval as *data* | six `sys_approval_*` platform objects | approval steps are ordinary records of an ordinary Machine (Case 7) — queryable, reportable, exportable like anything else | Menata's is more uniform |

## 5. Permissions and identity

ObjectStack (`packages/spec/src/security/*`, `identity/*`, `plugin-security`, ADR-0057/0066/0090/
0095/0105, `content/docs/capabilities/permissions.mdx`) — four enforced layers:

| Layer | ObjectStack | Menata Runtime | Gap? |
|---|---|---|---|
| 1. Capability (object CRUD) | `definePermissionSet({ objects: { obj: { allowRead/Create/Edit/Delete, viewAllRecords, modifyAllRecords } } })`; effective = union of held sets; distributed via Positions (`sys_position_permission_set`); `isDefault` binds to the built-in `everyone` position; `guest` position for anonymous | `CAP-P05` CRUD per role, `CAP-P01` per-Event permission, `CAP-P07` public role | equivalent — Menata's per-*Event* permission is finer than CRUD |
| 2. Data depth (scope) | `ObjectAccessScopeSchema`: `own | own_and_reports | unit | unit_and_below | org` on each grant (`permission.zod.ts:17`, ADR-0057); resolves to pre-computed `owner IN (…)` membership sets over the `sys_business_unit` tree + `manager_id` chain — no subquery | `CAP-P02` ownership (own) and everything else (all); `CAP-X09` org-unit scoping **dissolved** by design review (v0.41) into records/permissions/selectors — the permissions half "still unbuilt" | **the biggest permission gap**: no `unit`/`unit_and_below`/`own_and_reports`. ADR-0057 calls scope depth "the single highest-leverage ERP feature" and it's the axis every ERP (Dataverse, Salesforce, ServiceNow, SAP) converges on |
| 3. Record sharing | OWD `sharingModel: private | public_read | public_read_write | controlled_by_parent` (required on every object — "forgot to configure" cannot happen, `os lint security-owd-unset`); `externalSharingModel` for portal users; criteria sharing rules (`condition` CEL → materialised `sys_record_share`; `owner`-type rules removed as unenforceable); manual shares to user/team/position/unit; share links | `CAP-P02` owner_field; `CAP-F23`/`CAP-O07` groups on pickers; no rule-based widening | **criteria sharing rules** missing; no case has asked yet (Case 3 "share this document with X" is the nearest) |
| 4. Field-level security | `fields: { 'obj.field': { readable, editable } }`; hidden fields stripped server-side from API/reports too; metadata-plane masking (ADR-0106) | `CAP-P06` field visibility ("Salary visible only to HR") — its row says "scoped to read surfaces" (`permissions.hidden_fields` strips from List/Detail), write-side deliberately out of scope | **`editable` (visible but read-only for a role)** missing — a `CAP-P06` Tier 2; folded into gap G30 in the gap doc, no case names it yet |
| RLS predicates | `RowLevelSecurityPolicySchema` `{ name, positions, operation: select|insert|update|delete|all, using: CEL, check: CEL }` compiled to SQL (`rls-compiler.ts`); "no active organization" fails closed | Postgres RLS for workspace isolation (`CAP-X06`) — infrastructure, not authorable | authorable RLS is subsumed by scope depth + sharing for every case in the portfolio; hold |
| Org model | `sys_business_unit` tree (kind: department/…), `sys_position` (flat, no hierarchy — ADR-0090 D3), `sys_team`, `sys_user.manager_id`, `sys_member.role` = org admin tier only | `CAP-O01` users/roles, `CAP-O07` groups, `CAP-O11` multi-workspace membership | **org unit tree + manager designation** missing (the same two things §4 needs) |
| Tenancy | three postures; Layer-0 wall; multi-org needs the enterprise runtime | Workspace schema + RLS, always on, all open | Menata ahead (and simpler) |
| Admin | platform admin, delegated admin gate (`delegated-admin-gate.ts`), `adminScope` on sets, tab permissions | Admin role per workspace; `/admin/*` | equivalent |
| Explain | `explain-engine.ts` — "why can this person see this record?" layer by layer (`content/docs/permissions/explain.mdx`) | none | nice-to-have; cheap once scope depth exists |
| Audit | `plugin-audit` (who/what/when/old/new), opt-in record-*view* auditing, ADR-0052 audit ≠ activity feed; grant recertification (ADR-0091) | `CAP-R04` ⚠️ `record_events` snapshots | equivalent for mutations; view-auditing has no case (Case 20 HIPAA-weight notes is the candidate) |
| Auth | better-auth: sessions, API keys, MCP OAuth, SSO/MFA/lockout in enterprise (ADR-0069); SCIM schema | `CAP-X02` password + session, CSRF, rate limit, `CAP-O09/O11` signup/picker | SSO/MFA: no case pressure; API keys become relevant with §H1 |
| Safety discipline | ADR-0049 "no unenforced security properties" — any grant scope the engine can't enforce is a compile error, never fail-open | negative conformance tests per permission capability (`capability-lifecycle.md` §3b) | same principle |

## 6. Analytics, dashboards, reports

ObjectStack (ADR-0021, `packages/spec/src/{data/analytics.zod.ts, ui/dataset.zod.ts, ui/dashboard.zod.ts,
ui/chart.zod.ts, ui/report.zod.ts}`, `packages/services/service-analytics`, `plugin-reports`):

```
Presentation   report (tabular | summary | matrix | joined) · dashboard (12-col widget canvas) · list-view chart lens
                    │ bind by name; pick dimensions/measures
Semantic       dataset — base object + included relationships + declared dimensions (string/number/date/boolean/lookup)
                         + measures (count/sum/avg/min/max/count_distinct + derived ratio/sum/difference/product)
                    │ compiles to
Runtime        Cube (IAnalyticsService): metrics · dimensions (string/number/boolean/time/geo) · joins · time granularity
               strategies: objectql (in-engine) | native-sql; RLS read-scope applied inside the aggregate (read-scope-sql.ts)
```

| Feature | ObjectStack | Menata Runtime | Gap? |
|---|---|---|---|
| One definition per metric | `dataset` is the only analytics form since spec 9.0 — the three inline shapes were *removed* because "revenue" was defined three times in three grammars and drifted (ADR-0021 §TL;DR) | `CAP-V13` report View declares its own group-by/rollup per View; `CAP-V10` dashboard sections declare their own counts; `CAP-A14` aggregates declare theirs — three inline grammars, the exact pre-ADR-0021 shape | **semantic dataset layer** missing — the metric-drift risk is structural, not hypothetical, once Case 9's trial balance and a dashboard tile both say "balance" |
| Charts | `ChartTypeSchema`: bar, horizontal-bar, column, line, area, pie, donut, funnel, scatter, treemap, sankey, combo, metric/kpi, gauge, … (~20); axes, series, reference lines, drill-through target (`drawer|dialog|navigate`) | **none** — `internal/ui/dashboard.templ:7`: "Deliberately count-based, not charts … without a charting dependency" | **chart widget** missing; server-side SVG keeps the no-client-framework rule |
| Dashboard | 12-column grid `layout {x,y,w,h}`, widget `filter`, `compareTo: { kind: previousPeriod | previousYear }` (parallel aggregate for the prior window, "+12% vs last quarter"), runtime filters across the top, full-bleed TV display pages | `CAP-V10` composed sections (count tiles, lists) | period-over-period compare exists in `CAP-V13` (period compare) but not as a dashboard primitive |
| Reports | four shapes incl. **matrix** (rows × columns) and **joined** blocks; runtime filters; export; scheduled email digests (`plugin-reports`) | `CAP-V13`: group-by, hierarchy rollup (Case 9 chart of accounts), period compare, running balance — richer than ObjectStack's `summary` on the accounting axis; no matrix; no scheduled delivery (`CAP-V11` ❌ HOLD, email ❌) | matrix report missing; digest blocked on email |
| Permissions inside analytics | RLS/read-scope applied to every aggregate; FLS strips fields from reports | `CAP-V13`/`V10` render through the same permission-trimmed store reads | equivalent |
| Date macros | 36 relative tokens + custom offsets, one vocabulary for filters/dashboards (template gap #11 says they were *inconsistent* across APIs) | fixed `today` + date arithmetic in actions (`CAP-A11`) | subsumed by §R1 |
| Formula fields in analytics filters | template gap #10: not supported as of 2026-05 | n/a | — |
| Computation cost | native-SQL strategy pushes GROUP BY to the DB; cube registry caches definitions | `CAP-V13` computes at render time from JSONB (`data->>'fld'`), no cache | with `CAP-X10` expression indexes and a per-(dataset, filter, scope) TTL cache this stays cheap — see gap doc §R2/§R4 |

## 7. Other highlights (H1–H10)

**H1 — AI operability (MCP).** The runtime serves an MCP server at `/api/v1/mcp` by default
(`packages/mcp`); every object's CRUD becomes tools, actions opt in with `ai: { exposed: true,
description }` (ADR-0011, ADR-0109); each deployment is its own OAuth server so `claude mcp add
--transport http` signs in via browser; API keys for headless. Agents act under the same RBAC/RLS/
FLS as humans (design principle IV "Agent-Ready Boundaries"). Also spec'd: agents, tools,
knowledge/RAG (`packages/spec/src/ai/*`, `service-knowledge`, `knowledge-{memory,ragflow}`),
natural-language queries. Menata Runtime has `CAP-X07` REST per Machine and nothing AI-facing; an
MCP adapter over `CAP-X07` + `CAP-P*` is thin (gap doc §R16).

**H2 — Metadata versioning and rollback.** `sys_metadata_history` per item (ADR-0008), ADR-0067
commit history + rollback for AI-authored changes, `executionPinned` types whose historical
versions are never GC'd and are resolvable by hash (ADR-0009), draft → published with a visibility
gate (ADR-0027/0045), protocol-major compatibility check at load (`engines: { protocol: '^17' }`,
ADR-0087). Menata: `CAP-W07` change policy ✅, `CAP-X04` live reload ✅, `CAP-X08` export/import ⚠️,
`CAP-W02` pin ❌ (superseded as a target by W07). ADR-0009's narrow form — pin *only*
execution-bearing definitions, never GC them — is the design that would make a future `CAP-W02`
cheap instead of "multiplying the metadata cache per live version" (Study 20 §6.4's objection).

**H3 — Notifications: both runtimes hit the same wall.** ObjectStack's own template-gap ledger
(`docs/PLATFORM_GAPS_FROM_TEMPLATES.md`, 2026-05) lists **"no outbound channel (email/IM/webhook)"
as P0 #1** — flows fired, nobody was notified; the notification platform (ingress → outbox →
preferences → quiet hours → digest collapse → channels) was built afterwards (ADR-0012/0030,
`plugin-email`, `service-messaging`, `service-sms`). Menata's `CAP-A10` row says the same thing
in its own words ("email is deliberately not implemented — no mail infrastructure exists … don't
fake the part you can't"), and `CAP-O10` names the transport decision as still open. The
convergence is the finding: an outbound transport is infrastructure every real deployment needs
on day one, and both projects deferred it past the point cases needed it.

**H4 — Search.** ADR-0061: `$search` was parsed by every surface and executed by none until
2026-06; fixed as one server-resolved, metadata-driven resolver (`object.searchableFields`,
select label→value mapping, RLS/FLS applied, `pg_trgm`/`tsvector` as Tier 2), plus a pinyin
companion column for Chinese names (ADR-0098). Menata: `CAP-O04` workspace-wide,
permission-trimmed search ✅ and `CAP-V16` typeahead ✅ — already at ADR-0061's Tier 1.

**H5 — Validation gates and "no silently inert metadata".** Four gates (typed → `os validate` →
reviewed diff → governed runtime); `.strict()` objects with typo hints; retired keys become
tombstones that *reject with a prescription*; ADR-0049 enforce-or-remove; ADR-0078 no-silently-
inert-metadata; ADR-0054 every authorable surface needs runtime proof. Menata: `CAP-X05` ✅ plus
`runtime-metadata-schema.md`'s "Load-Time Contract — What's Enforced, What Silently No-ops"
section — which *lists* silent no-ops rather than turning them into load errors. The gap doc
recommends closing that list (§"discipline").

**H6 — Packaging and ecosystem.** Packages as first-class (ADR-0003, ADR-0070 package-first
authoring), package registry/marketplace/app-store schemas (`packages/spec/src/cloud/*`,
`service-package`), per-environment customization overlays (ADR-0005/0126), cross-package
collision rules (ADR-0048), namespace isolation (ADR-0028), `skills/` bundle so coding agents
start with the protocol's rules. Menata: `CAP-X08` ⚠️ Application export/import; `CAP-O02`
master data across Applications ✅. Marketplace-scale packaging has no case pressure.

**H7 — Declared ≠ enforced: read ObjectStack's feature lists with care.** The clearest lesson of
the source tree is how often a schema shipped years ahead of a runtime reader, and how much
machinery was later needed to detect that: state machine (three shapes, zero enforcement,
ADR-0020); `$search` (no-op on every surface, ADR-0061); `IWorkflowService` never implemented;
sharing `full` level "byte-equivalent to `edit`"; `queue` approver "resolves to nobody"; `softDelete`,
`indexes[].partial`, `contextVariables`, `crypto.hash` — all "authorable but had zero runtime
consumers" and removed under ADR-0049. `content/docs/capabilities/approvals.mdx` still advertises
"eight ways to pick approvers" including the shared queue the schema marks non-authorable. The
2026-05 template-gap ledger listed P0/P1 gaps (no outbound notification, upload UI unverified, no
CSV import wizard, single-step approvals, no conditional form fields, no PDF export) that the
capability pages didn't mention. **This is exactly the failure mode `capability-registry.md`'s
"✅ only with a conformance test" ratchet exists to prevent**, and it is the strongest argument in
this study for keeping that discipline while borrowing ObjectStack's ideas.

**H8 — Import/export.** REST import runner with dry-run, mapping, coercion, idempotency, cancel,
historical read-only insert (`packages/rest/src/import-*.ts`); CSV/XLSX/JSON export. Menata:
`CAP-R06` CSV both ways through the exact Create validation pipeline ✅ — equivalent minus XLSX and
a mapping UI.

**H9 — Realtime.** `service-realtime` WebSocket subscriptions, realtime protocol schema
(`api/realtime.zod.ts`). Menata: none; HTMX polling would be the server-rendered analogue. No case
pressure (Case 11 social feed is the candidate).

**H10 — Documentation as a product.** 438 MDX pages; `references/` generated from Zod
`.describe()` so the schema doc can't drift; `capabilities/*.mdx` written for business readers;
`request-template.mdx` for asking for a capability. Menata's `runtime-metadata-schema.md` +
`guides/*` are the hand-maintained equivalent; the generated-reference idea is worth noting for the
Authoring Layer someday.
