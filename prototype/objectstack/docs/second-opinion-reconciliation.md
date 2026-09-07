# Second-Opinion Reconciliation — an Independent ObjectStack Review vs. This Study

> Part of Study 37 (`../README.md`). On 2026-09-07 the owner brought a second, independently
> produced review of the same two repositories (36 sections, a 25-row star-rating matrix, a
> top-10 gap ranking) and asked: *compare it with your analysis — what can serve as additional
> reference or recommendation, and is your list already sufficient?* This document is that
> comparison. The second review is quoted by its own section numbers (§1–§36); this study's
> recommendations by `R#`/`G#` from `gap-analysis-and-recommendations.md`. Every verdict below was
> checked against `capability-registry.md` v0.54 rows and `app/` source, not taken from either
> review's prose.
> Status: v1.0 | Created: 2026-09-07 | Updated: 2026-09-07

---

## 1. Where the two reviews agree — and the one place they agree for the same reason

The second review's own conclusion (§36) is this study's conclusion in different words: *don't
chase feature parity; build the generic primitives ObjectStack proved matter, then use them as
substrate under Menata's own Business-Knowledge layer; keep the Process Overlay compiling to
primitives and never let it become a second engine (§35).* Its sharpest sentence — "the biggest
gap is not too few features but too few generic runtime primitives" — is exactly this study's
Tier-1 framing (R1–R4), and its "Document Approval as the vertical slice to find which primitive
is really missing" is the proof-case strategy this repo has been running since Case 3 / Study 32
(`cfd14fa` per-step user/group approver).

Agreements, with the matching item here:

| Second review | This study | Note |
|---|---|---|
| §11 Expression engine — "technical feature #1 to take from ObjectStack" | **R1** (Tier 1, rank 1) | Both rank it first. It suggests "an expression AST that is reusable"; R1 names cel-go and a fail-closed scope. Same thing, one step more concrete |
| §21/§32 MCP / AI-native layer | **R21** (`CAP-X16`) | It ranks this High; this study parked it at Tier 6 pending a case. **Revised — see §3 below** |
| §22 Metadata draft/validate/diff/publish/rollback | **H2 / R22** (`CAP-W02` narrow re-scope) | It asks for more than R22 — **accepted as an addition, §3** |
| §23 Observability with `trace_id` across request/event/constraint/permission | **R12** (`CAP-I04` trace UI) | Its list is broader (permission decisions, constraint outcomes in the trace) — folded into R12 |
| §5 Security: permission set → position → object → sharing → RLS → FLS | **R5/R6/R7** scope depth + unit tree; G30 sharing/explain deferred | Same layers identified; this study is narrower on purpose (only the layers with case pressure) |
| §13 Actions as first-class (visible_when / confirm / params / locations) | partially **R1** (visibility predicate) + `CAP-P04` `input_fields` | **Partially accepted, §3** |
| §17 Search as a declared primitive | `CAP-O04` ✅ already; G — | **Partially accepted, §3** (searchable-field declaration only) |
| §24–25 Keep Business Knowledge as first-class; don't become an ObjectStack clone with `Object → Machine` renamed | §4 "Where Menata is ahead", §5 "What not to copy" | Full agreement |
| §27 ObjectStack minus: complexity, "metadata platform becomes programming language", many capabilities not finished, power in commercial ObjectOS | **H7** declared ≠ enforced; §5 | Full agreement — and this study has the file-level evidence for it |

## 2. Where they disagree — checked against the code

The second review read `README.md`, `ARCHITECTURE.md`, the spec overview and
`implementation-status.mdx` on the ObjectStack side and the registry on the Menata side; this
study read the Zod schemas, the ADRs, the driver, the plugin sources and the 2026-05 template-gap
ledger. That difference explains every disagreement below: the second review inherits
ObjectStack's *declared* surface as if it were enforced (the H7 problem), and under-reads several
Menata rows that are already ✅.

| Second review claim | Verdict | Evidence |
|---|---|---|
| §4/§15 **Plugin/extension kernel — 🔴 Critical**, `Plugin{Init,Start,Stop}` interface in Go | **Rejected as a runtime mechanism; the goal is already met at compile time.** `capability-lifecycle.md` §4's registry seams (field-type / action-type / view-type tables in `internal/model` + `internal/executor`) are the extension points; Study 33 added fitness functions for them. A runtime plugin bus on a single Go binary with no third-party plugin ecosystem buys plugin-order, signature, health and DI machinery (`packages/core/src/{plugin-order,dependency-resolver,plugin-artifact-signature,health-monitor}.ts`) for nothing a case needs | `architecture-and-backend.md` §3; §5 of the gap doc |
| §16 **Storage abstraction / multi-DB drivers** | **Rejected.** Postgres is a decision (`prototype/go/docs/decisions/001-techstack.md`, RLS + JSONB + schema-per-workspace all depend on it), and `internal/store` already isolates SQL from business semantics — "business semantics must not depend on PostgreSQL" is *already* true; "the store must run on MySQL" is not a requirement any case has. ObjectStack's price for it is a 17,486-line driver plus a drift subsystem | `architecture-and-backend.md` §6 |
| §10 **Relationship model — 🔴 Critical**: lookup, master-detail, 1:1, 1:N, M:N, through, self, polymorphic, inverse, cascade, ownership | **Partially valid — three real pieces, the rest already exist or are non-goals.** Already ✅: `reference` (`CAP-F13`), self-reference (Case 18), header-detail (`CAP-F16`), M:N via junction Machine (`CAP-F20`, four case instances), reverse sub-list on the parent generated for *every* reference automatically (`CAP-V06`), referential integrity at write. Study 15's decision framework deliberately prefers composition over relation *types*. **Really missing:** (a) a delete policy on `reference` — today `CAP-R03` soft-deletes without `restrict / cascade / set_null` semantics; (b) permission-follows-parent for child rows (ObjectStack `controlled_by_parent`, ADR-0055); (c) a business name for the inverse relation (`CAP-V06`'s own row: section titles are "a prototype-honest generic label … since Menata Language has no way yet for a business author to name the relationship"). **Non-goals:** polymorphic references (schema-less target — Study 22's Case-File-Item ruling applies) and `through` relations (that *is* `CAP-F20`) | **Accepted as a new candidate, §3 (R24)** |
| §12 **UI metadata "far behind"** — a Page/Section/Component/Slot ontology | **Rejected in that form; the concrete gaps are already R3/R17–R20.** A Component/Slot vocabulary is what a client-rendered SPA needs to compose widgets; Menata's runtime renders server-side and `app/ARCHITECTURE.md`'s JS policy is a core principle, not a gap. What the review actually wants — pages composing several Views — exists for dashboards (`CAP-V10` sections) and the honest addition is a Tier 2 on `CAP-V10` ("a `page` composing any Views, not only summary tiles"), not a new ontology | `capability-comparison.md` §2.3 |
| §9 **Object model too simple** — Machine should carry identity, lifecycle, relations, formulas, aggregates, policies, pages, actions, workflows, integrations, search, files, analytics | **Rejected as structure, accepted as a list of specific gaps.** Menata already has these as separate Grammar areas rather than sub-keys of Machine (identity = `CAP-F18`/`CAP-O02`, lifecycle = Status + `CAP-E06`/`CAP-R07`/`CAP-R08`, aggregates = `CAP-A14`/`CAP-C10`, integrations = `CAP-I01–I04`, config = `CAP-X03`, process = `CAP-W*`). Folding everything under one Machine object is the path to ObjectStack's 3,195-line `object.zod.ts`. The genuinely missing sub-items are formulas (R1), analytics (R2/R3), search fields (R25) — already covered |  |
| §14 **API abstraction — Data/Metadata/Action/Query/Event API** | **Partially valid.** `CAP-X07` is `GET /api/v1/{machine}`, `GET …/{record}`, `POST …` only — no update/delete, no "perform Event" endpoint, no query filters, no OpenAPI (named in `app/ARCHITECTURE.md`'s gap table). Metadata API = `CAP-X08` export ✅ | **Accepted as a Tier 2 on `CAP-X07`, §3 (R26)** — and it is the prerequisite for R21 (MCP) |
| §17 **Search ★★ vs ★★★★** | **Overstated on both sides.** ObjectStack's `$search` was a silent no-op on every surface until ADR-0061 (2026-06); Menata's `CAP-O04` is ✅ permission-trimmed workspace search since 2026-07-12 plus `CAP-V08`/`CAP-V16`. The one real item: `CAP-O04` matches on the default list View's columns "rather than a new *searchable fields* declaration" (its own row) | **Accepted narrowly, §3 (R25)** |
| §18 **Audit/history**: record, field, action, permission-decision, workflow, login, metadata history | **Partially valid.** `record_events` (`CAP-R04` ⚠️) already stores actor + correlation id + pre-mutation snapshot, so *field* history is derivable (diff consecutive snapshots) but has no View; permission-decision logging = ObjectStack `explain` (G30, deferred); metadata history = R22/R27; login history = an NFR item, no case | **Accepted narrowly, §3 (R28: history timeline View, `CAP-R04` → ✅)** |
| §19 **Realtime abstraction "from the start"** | **Deferred stands** (H9). No case needs it; when one does (Case 11 feed, Case 17 queue) an SSE endpoint feeding HTMX's `sse` extension is a small, single-process addition — no bus abstraction needed up front. "From the start" is the kind of speculative substrate `capability-lifecycle.md` A2 exists to refuse |  |
| §20 **Connector family** (REST/OpenAPI/SQL/queue/external app) | **R11 + R23 are the case-backed subset**; the rest has no case |  |
| §26 matrix: *Approval — ObjectStack slightly ahead* | **Tie at worst.** ObjectStack ahead on send-back/per-group/manager-chain (R7–R10); Menata ahead on SoD (`CAP-P03`), delegation (`CAP-P04` — ObjectStack's is "partial in community edition" per its own ledger), business-calendar SLA (`CAP-O06`+`CAP-W04`; ObjectStack is wall-clock only), PDF signature (`CAP-F22`), and approval-steps-as-ordinary-records | `capability-comparison.md` §4 |
| §26 matrix: *Multi-tenancy — ObjectStack* | **Menata ahead.** Schema-per-workspace + RLS always on and open (`CAP-X06`), `/{slug}/` (`CAP-X14`), multi-workspace identity (`CAP-O11`); ObjectStack's walled postures need the *enterprise* runtime to activate (ADR-0105 D12) |  |
| §26 matrix: *Workflow — ObjectStack slightly ahead* | **Different bet, not a ranking.** Study 20 measured the two models; ObjectStack's own ADR-0020 is the cautionary tale for its side. Borrow nodes as actions (R11), not the engine |  |
| §26 matrix: *File/storage ★★★★★ vs ★★★* | **Reversed on evidence.** Menata's `CAP-F06` (upload, MIME enforcement, server-side resize/WebP) is conformance-proven; ObjectStack's own 2026-05 ledger lists "file field exists but no upload UI verified" as P0 #3 |  |
| §7/§28 Menata's "heuristics" (display field, cross-record) must be resolved before scaling | **Agreed** — that is R16 (`CAP-F13` Tier 2 `display_field`) and R24(c) inverse-relation naming |  |

**On the ratings themselves.** A star matrix without a proof column is the H7 failure mode in
miniature — it rates schema breadth. Where the second review's stars contradict a ✅ registry
row (search, files, tenancy, approvals), the row wins, because the row has a test ID behind it.

## 3. What this study adds because of the second review

Six additions. Each is a genuine gap the first pass under-weighted, now with the same pre-screen
as R1–R23. Numbering continues from the gap doc.

| Rec | What | Why the second review is right | Pre-screen | Suggested ID |
|---|---|---|---|---|
| **R24 — Relation policy on `reference`** | `options.on_delete: restrict \| cascade \| set_null` (default `restrict` — today a soft-deleted parent leaves dangling children); `options.permission_follows_parent: true` (child rows readable/editable exactly when the parent is — ObjectStack `controlled_by_parent`, ADR-0055, resolved as an `IN`-set like R5); `options.inverse_name` ("Direct Reports") so `CAP-V06` sub-lists and Menata Language can name the relationship | §10's list is mostly already ✅, but these three are real, cheap, and Case 3/9/19/20 all have parent-owned children (`CAP-F16`) that currently inherit nothing | A1 ✅ (ObjectStack ADR-0035/0055 + Cases 9/19/20) · A3 ✅ Field (options on an existing type) · A4 ✅ · A5 ✅ "deleting an Order deletes its Lines"; "a Prescription is visible to whoever may see its Medical Record" | `CAP-F13` Tier 2 (three options, one build) |
| **R25 — Declared searchable fields** | `machine.search: { fields: [...] }` consumed by `CAP-O04`/`CAP-V08`/`CAP-V16` instead of the default list View's columns; secrets/hidden fields never searchable (ObjectStack ADR-0061 D2/D5) | `CAP-O04`'s row names the missing declaration itself | A3 ✅ Workspace Services (search) · A4 ⚠️ composable today by shaping the list View — the *decoupling* is the capability · A5 ✅ "Customers are searchable by name, code and phone" | `CAP-O04` Tier 2 |
| **R26 — `CAP-X07` Tier 2: complete Data + Event API, OpenAPI** | `PUT/PATCH/DELETE`, `GET …?filter=…&sort=…` with the same operators as `CAP-V09`, `POST /api/v1/{machine}/{record}/events/{event}` (perform an Event with `input_fields`), generated OpenAPI document | §14 is right that only Create/Read exist; `app/ARCHITECTURE.md` already names OpenAPI as a gap; R21 (MCP) is a thin adapter *only if* this exists first | tier on existing row; A5 ✅ (an Event is already business language — the API just exposes it) | `CAP-X07` Tier 2 |
| **R27 — Metadata snapshots, diff, rollback** | Store each `CAP-X08` export as a versioned snapshot on every `CAP-X04` reload; structural diff between two snapshots (Machines/Fields/Events/… added, removed, changed); rollback = import snapshot N + reload. Draft/publish is *not* included — `CAP-W07` change policy already decides what a change means for in-flight records, which is the hard part draft/publish usually papers over | §22 is right that "live evolution" is a philosophy with export/import and change policy but no *history*; R22 (`CAP-W02`) is about pinning executions, not about being able to go back | A1 ✅ (ObjectStack ADR-0008/0067 + every platform in Study 2 has metadata versioning) · A3 ✅ Cross-Cutting · A4 ⚠️ composable from `CAP-X08`+`CAP-X04` by hand today — the snapshot log and diff are the capability · A5 ✅ "restore the Application as it was on Monday" | **`CAP-X17`** |
| **R28 — Record history timeline View (`CAP-R04` → ✅)** | A View type rendering `record_events` as a timeline: who, when, which Event, and the *field diff* between consecutive snapshots; already-collected data, no new writes | §18: "important for document approval" — Case 3/9 SOX-weight; `CAP-R04`'s own ⚠️ names the missing read surface | tier on existing row · A5 ✅ "show the document's history" | `CAP-R04` → ✅ |
| **R29 — Event presentation options** (`CAP-P04` Tier 2) | `event.presentation: { locations: [list_row, detail_header, detail_more], confirm: "…", icon, style: primary\|danger }` — the *rendering* half of ObjectStack's Action; the behaviour half (permission, availability, params) is already the Event itself | §13 is right that placement/confirm are missing; wrong that a new "Action" concept is needed — the Event *is* the action, which is a Menata strength (one thing to secure, one thing to log) | presentation; low · A5 ✅ "Approve is a primary button on the document" | `CAP-P04` Tier 2 |

**Revision to an existing recommendation.** **R21 (MCP, `CAP-X16`)** moves from "Tier 6, pending a
case" to **"propose the case now"**: the second review's worked prompts — *"which documents are
waiting for my approval and why is each blocked?"*, *"approve every invoice that satisfies the
policy"* — are a legitimate Case 3 extension (Document Approval assistant) and would give R21 the
terrain half of A1. Recommend adding it to `case-portfolio.md` as a declared target the same way
Case 3's PDF-signature extension was (Study 32), with R26 as its prerequisite.

## 4. What this study declines to add, and why (so silence isn't a decision)

| Second review item | Declined because |
|---|---|
| §15 runtime plugin kernel | see §2 — compile-time seams already are the extension architecture; no third-party plugin ecosystem exists to serve |
| §16 multi-database drivers | Postgres is a decision with RLS/JSONB dependants; no case; ObjectStack's cost is measurable |
| §12 Page/Section/Component/Slot ontology | SPA vocabulary; conflicts with the server-rendered principle; concrete needs covered by R3/R17–R20 + a `CAP-V10` Tier 2 |
| §19 realtime bus "from the start" | speculative substrate; SSE is a small later addition |
| §20 full connector family | R11 (outbound webhook) + R23 (email) are the case-backed subset |
| §31 Business-semantic vocabulary extensions (Policy, Obligation, Outcome, Responsibility, Decision) | **Wrong repo.** These are Menata *Language* concerns (`menata-id/menata`); this runtime realizes what the language declares (`004-runtime-metadata.md`, and the layer test in this repo's own memory notes). Evidence/SLA/Escalation/Requirement already exist on the runtime side (`CAP-W01/W04`). Raise them upstream, then register runtime capabilities when a case needs them |
| §26 star ratings as inputs | no proof column; contradicted by ✅ rows in four areas (§2) |

## 5. Is the original list sufficient?

**Mostly, but not entirely — the second review earned six additions (R24–R29) and one
re-prioritisation (R21), all of them narrow.** The pattern is consistent: where it argued from
ObjectStack's *architecture* (plugins, drivers, UI ontology, connector family, realtime bus) it
was proposing substrate Menata has no case for and would pay ObjectStack's price to build; where
it argued from *what a business application needs to be operated and trusted* (relation policy,
history, API completeness, metadata rollback, searchable fields, an AI operator for approvals) it
found real gaps this study had folded away too quickly. That second category is the reference
value of the document, and it has been absorbed above.

The updated priority order (superseding gap doc §7):

1. R23 email transport decision (`CAP-O10`) — unchanged, still first.
2. R1 expression layer (`CAP-F14` + `CAP-C13`).
3. **R24 relation policy + R28 history timeline** — new, cheap, both strengthen Case 3 directly; do them before the analytics trio because they close correctness gaps (dangling children, no history surface), not only breadth.
4. R2 + R3 + R4 analytics trio (`CAP-V22`, `CAP-V23`, `CAP-X10`).
5. R6 + R5 + R7 permissions pass (`CAP-O12`, `CAP-P08`, `CAP-P09`).
6. **R26 API completion (`CAP-X07` T2) → R21 MCP (`CAP-X16`)** — as one integration pass, with the Case 3 assistant extension as its proof case.
7. R8–R10 approval enrichments; R11 outbound webhook (`CAP-A16`); **R27 metadata snapshots (`CAP-X17`)**.
8. R13–R20, R25, R29 as cases arrive; §6 discipline ratchet whenever `validate.go` is next touched.

Still **no capability admitted** — the owner's decision, row by row.
