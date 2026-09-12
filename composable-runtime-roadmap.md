# Composable Runtime Roadmap

> Structured implementation roadmap for turning `007-composable-runtime-architecture.md` into the next generation of Menata Runtime without breaking the proven `app/` baseline.
>
> This roadmap complements `composable-runtime-blueprint.md`. The blueprint explains the architectural transformation and current-state gaps; this document is the implementation sequence, deliverables, dependencies, and proof gates.
>
> **Status:** Active
> **Created:** 2026-09-11

---

# 1. Roadmap Position

The repository has two different roadmap concerns:

- root `roadmap.md` = capability discovery, evidence, and admission governance;
- `app/ROADMAP.md` = graduation/cutover history for the current production runtime;
- this document = next implementation roadmap for composable runtime architecture.

The composable work must not be mixed into the historical graduation phases. `app/` is already the live baseline; composability now evolves it incrementally behind conformance and benchmark gates.

---

# 2. Current Gap Register

The following are the important implementation gaps that must be explicitly tracked rather than left implicit in `007`:

| ID | Gap | Current state | Priority |
|---|---|---|---|
| CR-01 | Canonical Domain/Data/Experience model in code | **Phase 1 slice implemented, 2026-09-11** — `app/internal/composable/` (NEW package, no `prototype/go` counterpart): `NodeIdentity`, `DomainNode` (thin wrapper over `*model.Machine`), `DataSource`/`DatasetRef`/`Projection`/`LogicalQuery`, `PageNode`/`ComponentNode`/`ViewRef`/`LayoutNode`, and `Scope`/`ContextRef`/`Binding` (shapes only, unused until Phase 3). `adapt.go`'s `BuildDatasetRef`/`BuildPageNode`/`BuildApplication` normalize an existing `*model.Application` (Machine + Views) into this model without invoking any View handler. Proof: `adapt_test.go` — deterministic (`reflect.DeepEqual` across repeated `BuildApplication` calls) against both builder fixtures and two real seeds loaded via `metadata.Loader` (`seeds/004_approval.sql` Document Approval, `seeds/032_kanban_lab.sql` standing in for project-management-shaped metadata — Case 19's own Board/List/Card/Checklist trial isn't seeded in `app/` yet, that's Phase 13). Full `go test ./...` and `go vet ./...` pass; zero handler/router/ui changes. Data IR (CR-02–05), Context/Binding validation (CR-06), and everything downstream in §19's dependency chain remain open — this row closes only the "model exists in code" bar, not the rest of Phase 1's own vocabulary breadth. | P0 |
| CR-02 | Data IR | **Phase 2 slice implemented, 2026-09-11** — `app/internal/composable/data.go`'s `DataIR{Datasets []Dataset}` plus `dataset.go`'s `BuildDataIR(app)`, which walks every Machine/View (list/calendar/timeline/board/form via `BuildDatasetFromView`, report via `BuildDatasetFromReport`, dashboard sections via `BuildDatasetFromDashboardSection`) and collects every representable `Dataset`. Deliberately NOT deduplicated across nodes yet — that's CR-11/Phase 7's job; this is the flat resolved-requirement list. | P0 |
| CR-03 | Dataset as reusable semantic contract | **Phase 2 slice implemented, 2026-09-11** — `Dataset` is now content-addressed (`identity.go`'s `datasetIdentity`, keyed on Source+Projection+Filter+Sort+GroupBy+Measures, never on which View asked), replacing Phase 1's per-View `DatasetRef`. Proof: `dataset_test.go`'s `TestSameDatasetSharedAcrossDashboardAndBoardConsumers` builds a Dataset from a Dashboard section and, independently, from a Board view over the same Machine/grouping field via two different adapter functions, and asserts they agree on Source/GroupBy/Relations/Projection — the concrete form of §6's exit criteria ("two different experience consumers can use the same Dataset definition"). | P0 |
| CR-04 | Projection model | **Phase 2 slice implemented, 2026-09-11** — `Projection{Fields []string}` plus `Filter`/`Sort`/`Measure`/`RelationRef` (the last discovered automatically from `reference`-typed fields inside a Dataset's own Projection, `dataset.go`'s `discoverRelations`). Existing `model.FilterCondition`/`model.SortConfig` lower into these directly (`lowerFilters`/`lowerSort`) — proven against real seeds (`seeds/004_approval.sql`'s `vw_as_progress` Sort+Relation, `seeds/008_journal_entry.sql`+`seeds/010_views_lab.sql`'s `vw_jel_report` grouped-sum Measures, `seeds/009_action_lab.sql`+`010`'s `vw_alp_dashboard` grouped/ungrouped count Measures). | P0 |
| CR-05 | Logical Query model independent from SQL/store APIs | partial concepts — `LogicalQuery{Dataset Dataset}` exists as a named type (per §5's own initial vocabulary) but still carries no real query semantics (pagination, cost, execution shape); those are explicitly Phase 9 (Query Planner / Physical Data Integration) work, not manufactured early. **Phase 9 update, 2026-09-11** — `app/internal/composable/physical.go`'s `BuildWhereClause` is a real, proven-correct logical→SQL translation (independent of `internal/store`'s own APIs — it's a pure function producing a parameterized fragment, not a call into the store), but `LogicalQuery` itself still carries no pagination/cost/execution-shape semantics. Still open. | P0 |
| CR-06 | Context / Scope / Binding | **Phase 3 slice implemented, 2026-09-11** — `app/internal/composable/context.go`'s `Scope` now maps each `ContextDomain` to the Machine id it names (`""` for domains with no entity identity), replacing Phase 1's bare domain set. `context_adapt.go` adds the real mechanism: `DeriveChildScope` (only an ambient domain set carries forward automatically — this is what makes "child scopes cannot access undeclared parent values" true by construction, not just documented), `ScopeForDataset` (seeds a Scope from a Phase 2 `Dataset`'s own `Relations`, reusing CR-03/04's real output), and `ValidateBinding` (enforces record/collection context must name a Machine, and a `current_user` binding must use the reserved `$current_user` sentinel — never re-sourced from arbitrary field data). Proof: builder fixtures for all three rules plus a synthetic 4-level Board→List→Card→Checklist chain (Case 19 still unseeded, same stand-in as CR-01/02), and a real-seed test (`TestScopeChainAgainstApprovalCase`) walking Document Approval's own Step→Document relation from Phase 2's `BuildDataIR` output through an accept/reject pair. Rule "context does not become an arbitrary query API" is satisfied by construction (`ContextRef`/`Binding` have no expression field to abuse). `Scope`/`Binding` are still NOT attached to any `PageNode`/`ComponentNode` — that needs Phase 4's real UI IR node tree to mean anything. | P0 |
| CR-07 | UI IR | **Phase 4 slice implemented, 2026-09-11** — `app/internal/composable/experience.go`'s `UINode` (identity/kind/properties/bindings/children/slots/actions/conditions, per §8's own vocabulary) replaces Phase 1/2's flatter `PageNode`/`ComponentNode`. `ui.go`'s `LowerPage`/`LowerApplication` give every `view_ref` node its own `Dataset` (closing the "Dataset per component node, not per page" gap Phases 1–2 left open) and validated `Bindings` (closing Phase 3's own "not yet attached to any node" gap) — proven against real seeds, not just fixtures: `seeds/050_composed_dashboard.sql`'s `vw_ad_page` (a real `page` View) lowers into `Slots["main"]`/`Slots["aside"]` with correct `view_ref`/`static_content` nodes. Renderer is still entirely View-oriented — nothing outside this package calls it, unchanged. | P0 |
| CR-08 | Generic Component contract | **Phase 5 slice implemented, 2026-09-11** — `app/internal/composable/component.go`'s `ComponentContract` (required/optional properties, dataset/children/actions allowance) covers exactly the seven components §9 names (`Heading`, `Text`, `Collection`, `RecordSummaryCard`, `Metric`, `StatusBadge`, `ActionBar`) — a closed set, not a starting point for more. `ValidateComponent` enforces "no giant property bag" directly: any property key outside a contract's own Required/Optional list is rejected. Each of the seven proven against real seeded metadata (`component_seed_test.go`) — Document Approval's Status field/Events/Step list, `seeds/050`'s `Display:"cards"` list, and a real Metric semantic rule (a grouped Dataset is a breakdown, not a single metric, proven against both an accepting and a rejecting real dashboard section). Specialized rendering (`internal/ui`'s own templates) is completely untouched — this contract governs the composable-plane node only, still one layer removed from `internal/handler`/`internal/ui`. | P1 |
| CR-09 | Static Component Registry seam | **Phase 5 slice implemented, 2026-09-11** — `component.go`'s `componentRegistry` is a plain, package-level Go map (never mutated at runtime, no reflection, no plugin loading — the same "static, closed set" pattern `model.go`'s own `SupportedOperators`/`EmbeddableChildViewTypes` already use), and `ResolveComponent` is the *only* path that can produce a valid component `UINode` — contract lookup, then `ValidateComponent`, then the node. `ui.go`'s `lowerStaticContent` migrates `PageContent`'s own "heading"/"text" types through this seam in place (proven against `seeds/050`'s real "Recent Activity" aside content); "button"/"image" still fall back to the pre-Phase-5 generic `static_content` node, per §9's own "migrate only a small number of proven generic components first." | P1 |
| CR-10 | View → generic primitive lowering | **Phase 4 slice implemented, 2026-09-11** — every View type still lowers to a `view_ref` UINode (not yet a real generic primitive — that's Phase 5's Component contract), but CAP-V10/CAP-V20's own Children/embedding mechanism (`model.ChildViewRef`) now lowers generically via `LowerChildren` into `UINode.Slots`, proven against the real cross-machine case (`seeds/042_inline_view_composition.sql`-shaped nesting) and the real named-slot case (`seeds/050_composed_dashboard.sql`'s main/aside row). Still "partial" by design — universal generic-primitive lowering is Phase 5/6's job, not this one's. **Phase 6 slice implemented, same day** — `app/internal/composable/view_lowering.go`'s `LowerViewToComponent`/`LowerDashboardView` prove List (table and "cards" modes), Board, Calendar, Timeline, and Dashboard all lower to real `Collection`/`Metric` components (§10's own "Initial lowering targets" table), including a genuine small gap closed in `dataset.go`: a calendar/timeline's `DateField` now folds into `Dataset.GroupBy` as its Date Dimension, mirroring Board's existing `GroupField` handling. Proven against real seeds (Document Approval, Action Lab's calendar and dashboard, Kanban Lab's board). Per §10's own compatibility rule, `LowerPage`/`LowerChildren` still produce `view_ref` nodes unchanged — these two functions are proven standalone, not wired into the main tree; still "partial" — that wiring, and proving real render-output equivalence, remains open. | P0 |
| CR-11 | Dependency DAG | **Phase 7 slice implemented, 2026-09-11** — `app/internal/composable/dag.go`'s `BuildDependencyDAG` walks a full UI IR tree (`UINode.Children` and `.Slots`, recursively) and collapses every node's own `Dataset` into a deduplicated `DependencyNode`, keyed by `dependencyIdentity` (`identity.go`). Proven against a real naturally-occurring duplicate: `seeds/050_composed_dashboard.sql`'s `vw_ad_pending` appears both as `LowerPage`'s own ordinary `view_ref` child and inside `vw_ad_page`'s own `Slots["main"]` — `BuildDependencyDAG` collapses both into one node with two `Consumers`. §11's own longer identity list (`parameters`, `pagination`, `interpreter/metadata version`) stays unbuilt — no case demands them yet. Coalescing/batching decisions themselves remain Phase 8's Composable Execution Planner. | P0 |
| CR-12 | CEP | proposed; not first-class code | P0 |
| CR-13 | CEP → Query Planner boundary | terminology/architecture needs enforcement. **Phase 9 update, 2026-09-11** — this phase IS that boundary: `physical.go`'s `PhysicalPlan`/`BuildPhysicalPlan` sit strictly after Phase 8's CEP output (`DependencyNode`) and produce planning-level facts only (which pushdowns apply, `BuildWhereClause`'s own SQL fragment) — no SQL is ever executed, no `internal/store` call is made. Proven against real seeded metadata: sort/aggregate pushdown correctly reported true (`RecordStore.List`/`CountGroupedBy` already do this), filter/projection/pagination/index-awareness correctly reported false, citing the exact real symbol responsible for each. The boundary is enforced by construction (this package still makes zero database calls, unchanged since Phase 1), not merely documented. | P0 |
| CR-14 | shared/batched execution | individual mechanisms exist; no common composable planner. **Phase 8 update, 2026-09-11** — the planner scaffolding itself now exists (`app/internal/composable/planner.go`'s `ExecutionGroup`/`GroupByMachine`/`ExecutionPlan`), proven against real seeded metadata (`mch_approval_document`'s three distinct Datasets grouped into one `ExecutionGroup`). Actual sharing/batching (§12's stages 4-8: bounded concurrency, batching, shared execution, cache, async fallback) remain unbuilt — every one needs a real physical executor, which doesn't exist in this package, and roadmap principle #8 forbids claiming an unbenchmarked coalescing strategy. Still "no common composable planner" in the sense of one that actually executes anything — this is its skeleton. | P1 |
| CR-15 | bounded execution width / fan-out | individual limits exist; no composition-level budget | P1 |
| CR-16 | security-aware dependency identity | **Phase 7 slice implemented, 2026-09-11** — `dag.go`'s `SecurityScope`/`ResolveSecurityScope` name the acting Role and that Role's own `HiddenFields` (CAP-P06, `model.Permission`) — the first time this package touches `model.Permission` at all. `SecurityScope.Apply` narrows a `Dataset`'s own Projection to what the Role actually sees, and `dependencyIdentity` stamps the Role directly into the dependency's own canonical identity. Proven against `seeds/012_permissions_lab.sql`'s real HR/Staff split over the same real List View (`vw_ple2_list`): the two Roles produce different dependency identities *and* different effective Projections (Staff's own node excludes `fld_ple2_salary`). Still narrow by design — this only names what's *hidden* for identity purposes; it does not enforce `CanRead`/CRUD-level permission, and planner-level coalescing decisions using this identity remain Phase 8's job. | P0 |
| CR-17 | plan identity / immutable plan reuse | partial metadata/interpreter caching foundations | P1 |
| CR-18 | inference diagnostics | principle exists; inspectability tooling missing. **Phase 8 update, 2026-09-11** — `planner.go`'s `ExecutionPlan.Explain()` is real, deterministic inspectability tooling (group count, per-group Machine/node counts, naive-vs-deduplicated query count), the first concrete tool this principle has, though scoped to execution planning only — broader inference inspectability (e.g. Dataset/Scope-level diagnostics) remains open. | P1 |
| CR-19 | composability benchmark harness | **Phase slice implemented, 2026-09-12 (§17j)** — `scripts/benchmark-composable.sh` + `scripts/loadtest` run 3 of Phase 12's own 9 required scenarios against a real throwaway server with real seeded data, reporting real p50/p95/p99 latency, real CPU/memory (`/proc` sampling), real DB pool stats, and `composable.MeasureComposition`'s own structural facts (previously Go-test-only) now correlated per real request. Cache hit ratio stays honestly N/A (no cache exists); scenarios 2/4/5/6/8/9 stay `SKIP` (no live route). See §17j for the real numbers. | P0 |
| CR-20 | trial applications using shared composable substrate | trial plan exists; runtime migration remains | P0 |
| CR-21 | metadata/schema representation for new composable artifacts | not yet universal | P0 |
| CR-22 | capability registry alignment | composable concepts span existing CAPs but are not yet one tracked implementation program | P1 |
| CR-23 | conformance model for composition validity | existing conformance is capability-oriented; composition proofs need expansion | P1 |
| CR-24 | failure isolation / partial rendering policy | architectural rule; no common composed-request implementation | P2 |
| CR-25 | renderer-neutral View Model / Render Input | **Phase 10 slice implemented, 2026-09-11** — `app/internal/composable/viewmodel.go`'s `FieldValue{Kind, Display}` plus `RecordSummary`/`MetricValue`/`CollectionItem`/`StatusValue`/`ActionSet` (exactly §14's own five named examples) are the renderer-neutral shapes; `viewmodel_resolve.go`'s resolvers are the first code in this package to accept real record data (a plain `map[string]any`, matching `internal/store.Record.Data`'s own shape) without importing `internal/store` itself. Zero Templ/HTML coupling — proven by construction (no such import exists anywhere in the package). Proven against real seeded records (Kanban Lab) and real rows inserted via the actual `store.RecordStore.Create` write path in tests (Typeahead Lab's `reference` field, Expression Lab's real CEL-computed field, Approval Document's real declared Events). Reference/user/group fields resolve to their raw stored id only (no cross-entity label dereferencing — a named, deferred gap); `ActionSet` applies no ownership/permission filtering. | P1 |
| CR-26 | migration compatibility for existing View handlers | **RESOLVED 2026-09-12** — see §2a's Migration Compatibility Contract below, backed by 17a/17b/17c's own real evidence (three shipped increments, not a policy statement). | P0 |
| CR-27 | Grammar-area decision for `Dataset` (new Grammar area `D`, alongside `F/E/A/C/P/V/R/X/I/O`, vs. folding under `View`) | **RESOLVED 2026-09-11, implemented** — owner decision: new Grammar area `D` (Data), consistent with the Domain/Data/Experience plane split `004`/`006`/`007` commit to. `capability-registry.md` gained a `## Data` section; `CAP-V22`/`CAP-V23` are reclassified there (pointer rows) with their IDs **retained unchanged** in `## Views` for stability, per the registry's own ratchet rule (never renumber/delete, only append). `capability-lifecycle.md` A3 and the proposal template now enumerate `D`; `nfr-standards.md` gained `### 2.11 Data (CAP-D*)`. | P0 |
| CR-28 | `007-composable-runtime-architecture.md` §40 (claim-by-claim PROVEN/PROPOSED citation matrix) does not exist | **RESOLVED 2026-09-11** — §40 added to `007`, citing `composable-runtime-blueprint.md` §3 and this file's own `CR-01`–`CR-28` register per claim. | P0 |

---

# 2a. Migration Compatibility Contract (resolves CR-26)

**Not a new mechanism to build — a contract written down from three real, already-shipped
increments (17a/17b/17c) that already are instances of it.** CR-26 sat as "planned" since this
register's own creation; per Principle #8 below (evidence before optimization) and the same
"resolve once real evidence exists" pattern `CR-27`/`CR-28` already used, it is resolved here by
stating, explicitly, the contract practice already established — each rule cited to a specific,
checkable fact, not asserted from memory.

1. **Additive-only routing.** A composable-backed capability lands at a *new* URL path alongside
   an existing route, never by rewriting an existing route's own handler function. `17a` added
   `GET /{machineID}/composable-preview` (`internal/router/router.go`) beside the pre-existing
   `/{machineID}` (List), `/{machineID}/board` (Board), `/{machineID}/{recordID}` (Detail)
   registrations — none of those touched.
2. **Existing handler files stay unedited.** `record_crud.go`, `views.go`, `embed.go`,
   `formfields.go` — the real, currently-shipped View handlers — are never modified to introduce
   a composable-backed alternative. Verified directly against `git show --stat` for all three
   commits (`ab1a557`/`4ea2af9`/`b5fe33c`): every file touched is one of
   `composable_preview.go`/`.templ`/`_templ.go`, `router.go` (17a's one added route line),
   `internal/composable/ui.go`/`ui_seed_test.go` (17c), conformance, or this roadmap document —
   never the four files above.
3. **New composable routes reuse the exact same guard sequence, never a parallel one with
   different semantics.** GetMachine 404 → workspace-scope 404 (CAP-X06) → `roleForApp`/
   `CanRead` 403 (CAP-P02), in that order — `composable_preview.go`'s guard block mirrors every
   other per-machine handler's own; T263/T264/T266 (`conformance/tests/241_composable_pilot.sh`)
   prove identical 403/404 behavior on the new route as on old ones.
4. **The full existing conformance suite is the compatibility gate, zero regressions tolerated,
   checked after every increment** — an actually-run mechanism, not a policy statement: 271 → 273
   passed, 0 failed, across 17a → 17b → 17c, via `./scripts/local-ci.sh` every time.
5. **Gate 5 (import-boundary fitness function) enforces the migration direction structurally,
   not just by convention.** `check-quality-gates.sh`'s Gate 5 forbids `internal/composable`
   production code (non-`_test.go`) from importing `internal/store`, `internal/db`,
   `internal/handler`, or `"net/http"` (verified against the gate's own current script) — so a
   handler may depend on composable, but composable can never become load-bearing for an old
   handler's own behavior. CI-enforced on every `local-ci.sh` run.
6. **Per-page cutover is a separate, later decision — this contract does not authorize or
   schedule one.** Replacing an old route's *own* rendering with the composable path happens
   only once render-output equivalence is actually proven for that specific page. Today,
   `composable-preview` is a parallel preview, never a replacement, for either machine it
   currently covers.

**Deliberately not resolved here, so it isn't conflated with CR-26:** a different, currently
untracked question — how mutation Actions (Submit/Approve/Reject) relate to `internal/composable`
— surfaced while scoping this resolution. It is **not** an open conceptual gap: `006-runtime-
model.md`'s own Domain Model already names Action ("Create, Update, Delete, Submit, Approve,
Reject, Publish, or Cancel") as Domain-plane, distinct from the Data/Experience planes
`internal/composable` governs (`composable-runtime-architecture-map.md`) — so this was answered
by `004`/`006` before this package existed, not left ambiguous. Confirmed against real code, not
just the model documents: `internal/composable/*.go` production code imports neither
`internal/executor` nor any `internal/store` write method today. No `CR-` id is opened for this;
it has no forcing case yet (a real candidate: `Card.Move` in a future Project Management
increment) and manufacturing one now would be exactly what Principle #8 warns against.

---

# 3. Principles for Implementation

1. **Do not rewrite the proven runtime in one pass.** Introduce shared seams incrementally.
2. **View compatibility first.** Existing View metadata and handlers continue to work while lowering is introduced.
3. **Logical before physical.** Build semantic IR before optimizing SQL.
4. **Planner before proliferation.** Establish the dependency/planning boundary before adding many generic components.
5. **Security before coalescing.** Security scope is part of logical dependency identity.
6. **No one-component-one-query rule.** Logical component count must not become physical query count by architecture.
7. **No generic escape hatch.** Components remain bounded; arbitrary code/data access is not introduced as “generic composition.”
8. **Evidence before optimization.** Every coalescing/batching strategy is benchmarked against a naive baseline.
9. **Inference is inspectable.** Important inferred dependencies must be diagnosable.
10. **Trial cases are real consumers.** Document Approval and Project Management use the same substrate.

---

# 4. Phase 0 — Contract Freeze and Documentation Alignment

**Goal:** eliminate conceptual ambiguity before adding code.

### Deliverables

- [x] canonical vocabulary recorded in `composable-runtime-architecture-map.md`;
- [x] source-of-truth hierarchy recorded;
- [x] runtime compilation terminology clarified;
- [x] View / Component / Dataset distinction documented;
- [x] CEP vs Query Planner boundary documented;
- [x] current implementation gaps explicitly recorded;
- [x] inference-inspectability principle added to `001-design-principles.md`;
- [x] `002`, `003`, and `006` aligned with the composable model;
- [x] `004-runtime-metadata.md` aligned with the composable model (Domain/Data/Experience metadata, compilation boundary);
- [x] `005-runtime-lifecycle.md` aligned with compilation/normalization/dependency-analysis/planning terminology;
- [x] cross-links added from `guides/writing-runtime-metadata.md` and `architecture-benchmark.md` to the composable direction docs.

**Detailed backlog:** the deliverables above are the short checklist; the full item-by-item
backlog — `runtime-metadata-schema.md` alignment, capability-governance taxonomy, composition-level
NFRs, README/agent-guidance wording, and benchmark/guide cross-links — was tracked in
`composable-runtime-roadmap-phase0-documentation-alignment.md` (`DOC-01`–`DOC-10`). **All ten items
resolved 2026-09-11**, same session as `CR-27`/`CR-28` above; that addendum is now closed and
condensed to a resolutions table (§2) — the actual corrections live in the target documents each
row names.

### Exit criteria

No Tier 1 document uses “interpreted” to imply “no internal compilation,” and no document presents View as the universal composition primitive. **Met, 2026-09-11** — see `composable-runtime-roadmap-phase0-documentation-alignment.md`'s resolutions table for what closed each item. **Phase 0 is closed.** The next primary implementation step, `CR-01` (Canonical Semantic Model in code), is subject to its own forcing condition — closing Phase 0 does not itself trigger it.

---

# 5. Phase 1 — Canonical Semantic Model

**Goal:** introduce the minimum internal Go model that can represent composition without forcing immediate database schema changes.

### Build

Introduce a package/boundary conceptually equivalent to:

```text
internal/composable/
  domain.go
  data.go
  experience.go
  context.go
  identity.go
```

The exact package name may change during implementation; the contract matters more than the path.

Initial types:

```text
NodeIdentity
DataSource
DatasetRef
Projection
LogicalQuery
Binding
Scope
ContextRef
ComponentRef
PageNode
LayoutNode
ComponentNode
ViewRef
```

Do not implement every proposed 007 object immediately. Start with the smallest model required by the two trial applications.

### Compatibility

Existing `View`, `ViewConfig`, Machine, Event, Permission, and Store structures remain intact. Adapters expose their semantic requirements into the new model.

### Proof

- unit tests for deterministic normalization;
- approval and project-management metadata can produce a canonical semantic representation;
- no change in existing 219+ capability conformance results.

### Exit criteria

The runtime can represent a composed page/data requirement without directly invoking a View handler as the only source of truth.

**Status update (2026-09-11):** exit criteria met for the scope this phase actually
committed to. `app/internal/composable/` normalizes any `*model.Application` into
`PageNode`/`ComponentNode`/`DatasetRef` values purely from `Machine.Views`/
`ViewConfig`, proven deterministic against both synthetic fixtures and two real seeds
(Document Approval, and the existing kanban board proof standing in for project-
management-shaped metadata — see CR-01's own row in §2 for why Case 19 itself isn't
used yet). Existing `View`/`ViewConfig`/handler code is completely untouched — nothing
calls this package yet, by design; that wiring starts at Phase 4/6. Phase 2 (Data IR)
is next per §19's dependency chain.

---

# 6. Phase 2 — Data IR, Dataset, and Projection

**Goal:** decouple semantic data requirements from ViewConfig and store-specific APIs.

### Build

Implement:

```text
DataSource
Dataset
RelationRef
Projection
Filter
Sort
Measure
LogicalQuery
DataIR
```

Start with the minimal vocabulary needed for:

- list/table;
- card collection;
- dashboard metric;
- board grouping;
- approval-step collection.

Existing field/operator filters remain syntax sugar and lower into the common expression/filter representation where practical.

### Important boundary

A Dataset is semantic. It must not contain:

- HTML;
- Templ syntax;
- CSS classes;
- SQL;
- PostgreSQL index names;
- cache implementation details.

### Proof

Create equivalent Data IR from:

1. existing List View;
2. existing Dashboard section;
3. Project Management collection/card;
4. Document Approval approval-step list.

### Exit criteria

At least two different experience consumers can use the same Dataset definition without copying renderer-specific configuration.

**Status update (2026-09-11):** exit criteria met for the vocabulary this phase
committed to (RelationRef/Filter/Sort/Measure/DataIR; full Logical Query semantics
remain Phase 9, see CR-05's own row in §2). `app/internal/composable/dataset.go`'s
`Dataset` is content-addressed, so `BuildDatasetFromView`/`BuildDatasetFromReport`/
`BuildDatasetFromDashboardSection` — three independent adapters for three different
experience shapes — produce equal Datasets whenever the underlying logical
requirement is equal, proven both by builder fixtures
(`TestSameDatasetSharedAcrossDashboardAndBoardConsumers`) and against real seeds
(Document Approval's approval-step list, the Action Lab dashboard's two sections, the
Trial Balance report, and the Kanban Lab board standing in for project-management-
shaped metadata — see CR-01's row for why Case 19 itself isn't used). `Display`
(CAP-V02's own rendering-mode toggle) is proven to never enter a Dataset
(`TestDatasetIdenticalAcrossDisplayModes`), the concrete form of §6's "Important
boundary" rule. Existing `View`/`ViewConfig`/handler code remains completely
untouched. Phase 3 (Context, Scope, Binding) is next per §19's dependency chain.

---

# 7. Phase 3 — Context, Scope, Binding

**Goal:** make nested composition semantically safe and predictable.

### Build

Implement:

```text
Context
Scope
ContextRef
Binding
```

with the initial domains:

```text
page
route
parameters
current_user
workspace
record
parent_record
selection
query_result
variables
```

### Rules

- child scopes cannot access undeclared parent values;
- record/collection context is explicit;
- bindings are validated before rendering;
- context does not become an arbitrary query API;
- security context is never downgraded by a child binding.

### Proof

- Project → List → Card → Checklist context;
- Approval Document → Approval Step → Approver context;
- invalid out-of-scope binding is rejected at load/compile time.

### Exit criteria

Nested trial UI can be expressed without business-specific handler branches for context propagation.

**Status update (2026-09-11):** the validation mechanism itself is proven — `Scope`/
`DeriveChildScope`/`ValidateBinding` (`app/internal/composable/context_adapt.go`) express
arbitrary nesting depth generically (proven to 4 levels synthetically, and against
Document Approval's real Step→Document relation via `ScopeForDataset`) with no
business-specific branch anywhere in the mechanism. Not yet claiming the FULL exit
criteria though: no real "trial UI" exists to express this way yet (Phase 4's UI IR node
tree is what a Scope/Binding would actually attach to), and Case 19 remains unseeded —
see CR-06's own row in §2 for the precise scope of what's proven versus deferred. Phase 4
(UI IR and Generic Composition) is next per §19's dependency chain.

---

# 8. Phase 4 — UI IR and Generic Composition

**Goal:** make Experience composition independent from specialized View handlers.

### Build

Implement a minimal UI IR:

```text
UINode
 ├── identity
 ├── kind
 ├── properties
 ├── bindings
 ├── children
 ├── slots
 ├── actions
 └── conditions
```

Initial node kinds:

```text
page
layout
component
view_ref
static_content
```

### Lowering

```text
Page metadata
   ↓
UI IR
   ↓
render inputs
```

Existing View types are adapted as `view_ref`/preset nodes first. Do not delete existing handlers yet.

### Exit criteria

At least one complete page from each trial case is produced from UI IR while preserving existing rendered behavior.

**Status update (2026-09-11):** met for the vocabulary this phase committed to.
`app/internal/composable/ui.go`'s `LowerPage`/`LowerApplication` produce a complete
`UINode` page for Document Approval's own composed dashboard
(`seeds/050_composed_dashboard.sql`'s `vw_ad_page`, including its real `main`/`aside`
Slots) and for the Kanban Lab board (Project Management stand-in — Case 19 remains
unseeded, see CR-01's row). "Preserving existing rendered behavior" holds trivially:
nothing in the real render path (`internal/handler`/`internal/ui`/`internal/router`) is
touched, same as every prior phase — View lowering into that path is Phase 6, gated on
Phase 5's Component contract landing first per §19's dependency chain. Phase 5
(Component Contract and Static Registry Seam) is next.

---

# 9. Phase 5 — Component Contract and Static Registry Seam

**Goal:** stop specialized renderer dispatch from spreading into handlers.

### Build

Define a closed compile-time component contract and static registry seam:

```text
component type
   ↓
contract
   ↓
validator
   ↓
resolver
   ↓
renderer
```

Migrate only a small number of proven generic components first:

```text
Heading
Text
Collection
RecordSummaryCard
Metric
StatusBadge
ActionBar
```

### Do not do

- dynamic plugin loading;
- arbitrary component code in metadata;
- arbitrary SQL from components;
- client-side metadata interpreter;
- giant `GenericComponent` property bags.

### Exit criteria

New generic components can be added without adding business-specific branches to unrelated handlers.

**Status update (2026-09-11):** met for the vocabulary this phase committed to. Adding
the seven components named by §9 required exactly one `componentRegistry` entry each
(`app/internal/composable/component.go`) plus one small adapter function
(`component_adapt.go`) — zero changes to `internal/handler`, `internal/ui`, or
`internal/router`, and zero business-specific branches anywhere in `ValidateComponent`/
`ResolveComponent` themselves (both operate purely on the closed registry, generic
regardless of which of the seven is being validated). Not yet claiming the full
composition story though: none of the five data-bearing resolvers are wired into
`LowerPage`/`LowerChildren`'s actual tree — see CR-08/CR-09's own rows in §2 for the
precise scope proven versus deferred. Phase 6 (View Lowering and Compatibility Layer) is
next per §19's dependency chain — the phase that actually migrates real View output onto
this contract.

---

# 10. Phase 6 — View Lowering and Compatibility Layer

**Goal:** make existing View types convenience presets over the composable substrate.

### Initial lowering targets

```text
List
 → Collection + Projection + table renderer

Card List
 → Collection + Projection + card renderer

Board
 → Collection + Group Dimension + board renderer

Dashboard
 → Page + Layout + Metric / Collection components

Calendar
 → Collection + Date Dimension + calendar renderer
```

### Compatibility rule

Existing metadata remains valid. Existing View handlers may remain behind the lowering layer until equivalence is proven.

### Exit criteria

At least three existing View types can lower to shared IR without changing conformance behavior.

**Status update (2026-09-11):** met, with room to spare — five types prove out, not just
three: List (in both table and "cards" render modes), Board, Calendar, Timeline, and
Dashboard all lower correctly (`app/internal/composable/view_lowering.go`), against real
seeded metadata rather than only synthetic fixtures. "Without changing conformance
behavior" holds trivially, same as every prior phase: `internal/handler`/`internal/ui`/
`internal/router` are untouched, and `LowerPage`/`LowerChildren` still produce `view_ref`
nodes exactly as before — per this phase's own compatibility rule, existing View handlers
stay "behind the lowering layer" and nothing here has yet attempted to prove real
render-output equivalence with them. Report, Form, Detail, Document, ProcessMap,
DecisionStepper, CoordPlacement, and Page remain unlowered — outside §10's own initial
target table, not a gap in what was attempted. Phase 7 (Dependency DAG) is next per
§19's dependency chain.

---

# 11. Phase 7 — Dependency DAG

**Goal:** make all composed data/expression/security/render dependencies explicit before physical execution.

### Build

Derive:

```text
UI IR
Data IR
Bindings
Permissions
Expressions
Relations
Cache requirements
        ↓
Dependency DAG
```

Each dependency gets canonical identity including, where relevant:

```text
source
security scope
normalized filters
parameters
projection
grouping
measures
pagination
interpreter/metadata version
```

### Proof

The same Dataset consumed by multiple components produces one logical dependency node.

Security-incompatible consumers remain separate nodes.

### Exit criteria

A composed request can be inspected as a DAG before physical execution.

**Status update (2026-09-11):** met for both proof requirements named above.
`composable.BuildDependencyDAG` (`app/internal/composable/dag.go`) returns a plain,
inspectable `DependencyDAG{Nodes []DependencyNode}` — each `DependencyNode` names its own
canonical `Identity`, its security-scoped effective `Dataset`, and every consuming
`UINode`'s own identity — computed entirely before any physical execution (no database
access anywhere in this package, unchanged since Phase 1). Both real-seed proofs pass:
dedup (Document Approval's `vw_ad_pending`) and security separation
(`seeds/012_permissions_lab.sql`'s HR/Staff split). `internal/handler`/`internal/ui`/
`internal/router` remain untouched. Phase 8 (Composable Execution Planner) is next per
§19's dependency chain — the phase that actually uses this DAG to make coalescing/
batching decisions, which this phase deliberately stops short of.

---

# 12. Phase 8 — Composable Execution Planner

**Goal:** make physical economy a first-class runtime boundary.

### Build in this order

1. execution groups;
2. dependency deduplication;
3. security-aware equivalence;
4. bounded concurrency;
5. batching;
6. shared execution;
7. cache opportunities;
8. async fallback;
9. cost estimation;
10. plan diagnostics.

### Planner contract

```text
Dependency DAG
      ↓
Execution Groups
      ↓
Cost / Capability Analysis
      ↓
Costed Execution Plan
      ↓
Physical operations
```

The planner does not replace the database query planner.

### Exit criteria

Planner-enabled execution passes all semantic/security conformance tests and demonstrates lower or equal physical work for the benchmark scenarios defined below.

**Status update (2026-09-11):** the literal exit criteria stays open by design —
"passes all conformance tests" needs real physical execution (Phase 9), and "the
benchmark scenarios defined below" is Phase 12's own harness, neither of which exists
yet; claiming either here would be exactly the unbenchmarked optimization claim roadmap
principle #8 forbids. What Phase 8 actually built, up to but not including physical
execution: `app/internal/composable/planner.go`'s `GroupByMachine` (stage 1, execution
groups — stages 2/3, dependency dedup and security-aware equivalence, turned out to
already be delivered by Phase 7's own `DependencyDAG`), `BuildExecutionPlan`'s
counting-based `NaiveQueryCount`/`DeduplicatedQueryCount` comparison (an honest,
benchmark-free stand-in for stage 9), and `Explain()` (stage 10). Proven against real
seeded metadata: `mch_approval_document`'s three distinct Views grouped into one
`ExecutionGroup`, and a real 4-vs-3 query-count reduction from Phase 7's own proven-safe
deduplication. Stages 4-8 (bounded concurrency, batching, shared execution, cache,
async fallback) remain unbuilt — no physical executor exists to make those decisions
against. Phase 9 (Query Planner / Physical Data Integration) is next per §19's
dependency chain — the phase that finally connects any of this to a real database.

---

# 13. Phase 9 — Query Planner / Physical Data Integration

**Goal:** connect logical execution units to efficient PostgreSQL execution without leaking physical details into metadata.

### Build

Integrate existing scale mechanisms:

- projection pushdown;
- filter pushdown;
- aggregate pushdown;
- index awareness;
- pagination;
- cache;
- workspace concurrency limits;
- separate analytics/report capacity;
- materialization when evidence justifies it.

### Boundary

```text
CEP
 ↓
logical data execution unit
 ↓
Query Planner
 ↓
SQL / cache / materialization
```

### Exit criteria

The runtime can demonstrate why a composed logical request produced a given bounded physical plan.

**Status update (2026-09-11):** met at the explanatory level this phase targeted.
`PhysicalPlan.Explain()` (`app/internal/composable/physical.go`) names, per dependency,
exactly which of §13's nine mechanisms apply and cites the real `internal/store`/
`internal/handler` symbol behind each — proven against real seeded metadata, including a
genuinely mixed case (`seeds/010_views_lab.sql`'s "My Overdue Tasks": one pushdown-safe
filter, one that correctly isn't, in the same real View). What stays open: no physical
plan here is ever actually executed — `internal/store` is unchanged, filter/projection/
pagination pushdown remain real, honestly-documented gaps in the live runtime, not
merely unproven claims. Actually closing those gaps (and building workspace concurrency
limits, separate analytics capacity, cache, or materialization) is real infrastructure
work with no case forcing it yet, left for whoever picks it up next. Phase 10 (Renderer /
View Model Separation) is next per §19's dependency chain.

---

# 14. Phase 10 — Renderer / View Model Separation

**Goal:** ensure renderers consume semantic render inputs rather than raw metadata or raw database records.

### Build

```text
Data IR
 ↓
View Model / Render Input
 ↓
Renderer
```

Initial semantic view-model examples:

```text
RecordSummary
MetricValue
CollectionItem
StatusValue
ActionSet
```

The renderer must not know whether a value originated from a field, expression, or relation.

### Exit criteria

HTML/Templ remains the reference renderer, but the data semantics are not coupled to Templ.

**Status update (2026-09-11):** met by construction — `app/internal/composable/
viewmodel.go`'s five View Model types have no Templ or HTML import anywhere, and
`internal/ui`'s own Templ templates are completely untouched (still coupled to their own
existing types, e.g. `ListRow`/`DetailField`). "The renderer must not know whether a
value originated from a field, expression, or relation" is proven concretely:
`ResolveFieldValue` returns the identical `FieldValue{Kind, Display}` shape for all
three sources, against real data (Kanban Lab's seeded records; real rows this phase's
own tests inserted for Typeahead Lab's reference field, Expression Lab's computed field,
and Approval Document's events). Not yet claiming Templ actually *consumes* these types
— that migration, if it happens, is separate wiring work into `internal/ui`, which this
phase deliberately doesn't attempt, same "prove it standalone" posture used since Phase
5. Phase 11 (Composition Conformance) is next per §19's dependency chain.

---

# 15. Phase 11 — Composition Conformance

**Goal:** extend the existing capability-oriented proof system to composition semantics.

### New proof classes

```text
CMP-01 valid composition tree
CMP-02 cyclic composition rejected
CMP-03 unresolved reference rejected
CMP-04 slot/type mismatch rejected
CMP-05 scope violation rejected
CMP-06 dependency deduplication
CMP-07 security prevents unsafe coalescing
CMP-08 bounded fan-out
CMP-09 planner deterministic
CMP-10 View lowering equivalence
CMP-11 partial/failure isolation
CMP-12 trial-case cross-substrate reuse
```

The existing conformance suite remains authoritative for existing capabilities.

**Status update (2026-09-11):** the twelve `CMP-*` classes now have a real proof system
— `app/internal/composable/conformance_test.go`/`conformance_seed_test.go` — but
deliberately **not** inside `app/conformance/` itself (the real HTTP-driven CAP-xx
suite this package still isn't wired into) or `capability-registry.md` (that ledger's
own A1–A5 admission test is for new user-facing capabilities, not proof classes about
this substrate). Eight classes (`CMP-01`, `05`–`07`, `09`–`11`) consolidate proofs
Phases 2–10 already established; four (`CMP-02`, `03`, `04`, `08`) needed a real code
change — `ui.go`'s `LowerChildren` now genuinely recurses into a resolved `view_ref`'s
own further `Children` (previously "one level only" since Phase 4), made safe by cycle
detection and a depth bound (`maxCompositionDepth = 5`), and now fails loud instead of
silently skipping an unresolved view reference or a malformed `Children` entry. Honestly
scoped: today's real `metadata/validate.go` allow-lists never let a page embed another
page, so `CMP-02`/`CMP-08` are proven with builder-only fixtures (defense in depth, not a
live bug fix) — real seeded metadata backs `CMP-03`/`CMP-04`/`CMP-10` instead.
`CMP-12` is an explicit `t.Skip` (blocked on Case 19/Phase 13), not a silent gap. "The
existing conformance suite remains authoritative for existing capabilities" holds
unchanged — `app/conformance/` itself is untouched. Phase 12 (Composability Performance
Benchmark) is next per §19's dependency chain.

---

# 16. Phase 12 — Composability Performance Benchmark

**Goal:** prove that logical composability does not cause proportional physical work.

### Required scenarios

| Scenario | Expected question |
|---|---|
| 1 component → 1 Dataset | baseline overhead |
| 10 components → 10 independent Datasets | bounded concurrency / fan-out |
| 10 components → 3 shared Datasets | deduplication/coalescing |
| 1 Dataset → multiple projections | projection sharing vs narrower queries |
| mixed OLTP + analytics | pool isolation and class enforcement |
| 100 concurrent workspaces, cold/warm metadata | cache and workspace isolation |
| same logical request, different security scopes | security-aware identity |
| nested Project board | composition depth / context cost |
| Approval detail + stepper | embedded dependency sharing |

### Metrics

```text
p50 / p95 / p99 latency
logical nodes / request
DAG nodes / request
physical queries / request
physical operations / request
rows scanned
rows returned
CPU
memory
DB pool utilization
planner time
render time
cache hit ratio
execution width
```

### Acceptance rule

No planner optimization is accepted merely because query count falls. It must preserve semantics, security, and relevant latency/resource SLOs.

**Status update (2026-09-11):** this phase reports the subset of the metric list above
that is honestly computable without live execution — `app/internal/composable/
benchmark.go`'s `MeasureComposition` (logical nodes, DAG nodes, naive-vs-deduplicated
physical query count, execution width) — run across all nine required scenarios.
`p50/p95/p99` latency, CPU, memory, DB pool utilization, planner time, render time,
cache hit ratio, and rows scanned/returned all stay unbuilt: none exist without real
query execution (Phase 9's own audit — `internal/store` remains untouched), real
rendering (Phase 10 built View Models with zero Templ wiring), or real process/pool
metrics, and fabricating any would violate this very Acceptance rule's own spirit.
Concretely: scenario 1 (baseline) costs exactly one of everything; scenario 3 (10
components → 3 shared Datasets) shows a real 10→3 reduction; scenario 4 (one Machine,
two Projections) *honestly reports* that no sharing happens today — a real, non-ideal
finding, not a failure, since projection-superset merging is Phase 8's own unbuilt
stage 6. Scenarios 5 and 6 (mixed OLTP/analytics pooling; 100 concurrent workspaces)
are explicit `t.Skip`s naming the missing infrastructure, not silently dropped.
**The Acceptance rule itself is only half-checkable this phase** — every reduction
shown is a structural fact whose semantics/security safety Phases 3/7/11 already
proved; the latency/resource-SLO half needs live execution this phase doesn't have, so
nothing here is being *accepted* as a proven optimization, only *reported* as a
structural one. One real, previously-unnoticed gap surfaced while grounding the
"Approval detail + stepper" scenario: `LowerPage` only lowers a page-type View's own
`Children`, never a `detail`-type View's own (`seeds/042_inline_view_composition.sql`'s
`vw_as_detail`) — named here, not fixed; out of scope for a benchmark phase. Phase 13
(Trial Migration) is next per §19's dependency chain.

---

# 17. Phase 13 — Trial Migration

**Goal:** prove the architecture with real user workflows.

### Document Approval

Progressively lower:

```text
Approval Page
Approval Document
Approval Steps
Approver binding
Decision actions
Activity
```

into the shared composable substrate.

### Project Management

Progressively lower:

```text
Project Page
Board
List
Card
Checklist
Members
Activity
Ordering
```

into the same substrate.

### Success criterion

The two applications share the same:

- Page/Layout primitives;
- Dataset/Projection machinery;
- Component contract;
- Context/Binding system;
- permission-aware dependency planning;
- CEP;
- renderer boundary.

Case-specific business behavior may remain in Domain/Action/Event logic. It must not leak into generic composition infrastructure.

**Status update (2026-09-11):** Case 19 (Project Management) is seeded for the first
time (`app/seeds/052_project_management.sql`) — every prior phase's own stand-in
(Kanban Lab) is no longer the only option. **Investigating why it was never seeded
surfaced a real, already-documented capability gap, not just an oversight**:
`capability-registry.md`'s own `CAP-V14 Tier 2` row states the real board view is "a
deliberately narrower cut than Case 19's own literal declaration" — fixed `value_list`
lanes only, never Case 19's own dynamic, user-creatable List records with
drag-between-lists `Card.Move`. Building that faithfully needs new capability work in
`internal/handler`/`internal/model` — a `capability-lifecycle.md`-governed admission
decision. **Confirmed with the user directly**: this phase seeds Case 19 at
composable-substrate-proof scope only (real `CAP-F13` references Board←List←Card,
`CAP-V06` child-lists on Detail pages, `CAP-F16` `ChildLines` for Checklist Items) — no
drag-and-drop board, no scoped manual ordering, named everywhere as a reduced stand-in,
not a faithful Trello UX. **Formally logged, same day**: `roadmap.md`'s Study 41 records
the full finding (dual-sourced against Case 19 itself and `app/web/static/ui-sample/
case-19.html`/`project-board.html`'s own independent design-intent wording), and
`capability-registry.md`'s `CAP-V14 Tier 2` row now carries a named, not-yet-admitted
candidate, `CAP-V14 Tier 3` (dynamic-lane board + composed `Card.Move`). **This gap has
no build phase within this document** — it is a real-runtime capability question
(`internal/handler`/`internal/model`), not a composable-substrate one, so no CR-numbered
row in §2 and no Phase 1–14 slot here is the right home for building it; Phase 13's own
job was only ever to discover and honestly record it, which is now done. Whoever picks
`CAP-V14 Tier 3` up runs `capability-lifecycle.md` §2's own A1–A5 test, tracked entirely
in `capability-registry.md`/`roadmap.md`, independent of this document's own phase
sequence.

**Status update (2026-09-12, found while grounding 17h):** the paragraph above is now stale —
`CAP-V14 Tier 3` (dynamic-lane board + `Card.Move`) has since been built and admitted.
`seeds/052_project_management.sql`'s own `vw_pmc_board` row (added after this section was
written) already declares `group_field: fld_pmc_list`, a `reference` field, with its own comment
citing this as the real proof case; `conformance/tests/240_dynamic_board_lanes.sh`'s T259-T261
prove it live today (one lane per real `mch_pm_list` record, `board-move` writing a real
referential-integrity change). Left in place rather than edited away, per this repo's own
"append, don't rewrite" convention — the paragraph correctly describes the state as of
2026-09-11; it no longer describes the state as of 2026-09-12.

The success criterion itself is proven concretely: `app/internal/composable/
trial_test.go`'s `assertFullSubstrateCoverage` is one shared function, no per-case
branch anywhere in it, exercising all seven listed mechanisms in order — called once
for Document Approval and once for Project Management
(`trial_seed_test.go`), both passing against real seeded metadata and real records.
Nothing changed anywhere else in `internal/composable` — every mechanism this phase
needed already existed from Phases 2–12. `internal/handler`/`internal/ui`/`internal/
router` remain completely unwired from the composable substrate, unchanged since Phase
1 — Case 19's new Machines get ordinary metadata-driven routes automatically, same as
any other seeded Machine, but nothing about *composable* wiring changed. Phase 14
(Production Hardening) is next per §19's dependency chain — though its own opening line
("Only after real trial workloads") should be read against this phase's own honest
scope, not a claim that Case 19 is production-ready.

---

# 17a. Live Wiring Pilot (inserted 2026-09-12)

**Not one of the original 14 phases** — inserted between Phase 13 and Phase 14 after this
session's own review found Phase 14's precondition ("Only after real trial workloads") unmet:
Phase 13's own closing note already recorded that `internal/handler`/`internal/ui`/`internal/
router` remained completely unwired from `internal/composable`, proven equivalent only inside
isolated Go tests (`trial_test.go`), never driving an actual HTTP response. Owner decision (this
conversation, via AskUserQuestion): pilot real live-wiring first, so Phase 14 has something real
to harden. Numbered "17a" deliberately, not "18", to avoid renumbering §18–21 — no other document
cross-references those section numbers by number (`case-portfolio.md` cites only §17/Phase 13).

**Goal:** prove `internal/composable` can drive one real, live HTTP response over real seeded
Postgres data end to end (Metadata → `Dataset`/`UINode` → `Resolve*` view models → HTML), with
zero risk to any existing shipped page.

**What was built:** a new, additive, read-only route, `GET /{machineID}/composable-preview`
(`internal/router/router.go`, alongside the existing `/{machineID}/board` line), handled entirely
by a new file (`internal/handler/composable_preview.go`) that never touches `record_crud.go` or
`views.go`. It reuses the exact same guard sequence (`GetMachine`/workspace-scope 404/`CanRead`
403) every other per-machine route already uses, then calls
`composable.BuildDatasetFromView`/`LowerCardRowComponent` (pure, `internal/composable/
view_lowering.go`) once per View, and `composable.ResolveRecordSummary` (`internal/composable/
viewmodel_resolve.go`) once per real record fetched via the ordinary `h.records.List` call — the
only I/O in the whole path, and it lives in the handler, not in `internal/composable` (Gate 5
stays green, unchanged). A new templ file, `internal/ui/composable_preview.templ`, renders
`[]composable.RecordSummary` directly — `internal/ui` importing `internal/composable`'s View
Model types is the allowed direction (Gate 5 only forbids the reverse).

Pilot case: Document Approval's own `vw_ad_all` View (`seeds/004_approval.sql`,
`mch_approval_document`, `display: cards` since `seeds/048_v02t2_v09t2_realization.sql`) — the
same View T248 (`conformance/tests/210_v02t2_v09t2.sh`) already proves the real, shipped cards
feature against. **Named limitation, not an oversight**: `RecordSummaryCard`'s contract is
title+subtitle only, so this preview does not show `fld_ad_status` the way the real cards feature
does — extending that contract to a status/badge slot is real capability work, out of this
pilot's scope.

**Proof:** `conformance/tests/241_composable_pilot.sh` — T262 (a real record's title/subtitle
posted through the ordinary `Create` flow reach the page via the composable pipeline, not a
stub), T263 (a role with no permission on this Machine's app is denied 403, same guard as every
other route), T264 (a Machine from another Workspace 404s, CAP-X06 pattern). All three pass;
full existing suite stays green (269/270 — the one failure is the pre-existing, unrelated T209
CAP-V21 issue, tracked separately). Manually verified against the real running `menata.app`
server too (restarted via `/root/scripts/server-manager.sh restart menata-runtime`, per this
session's own newly-documented `app/CLAUDE.md` "Server lifecycle" section): logged in as Alice,
`/ws_default/mch_approval_document/composable-preview` rendered the real seeded "Vendor Contract
Q3" / "Contract" record as a card, correct page chrome, correct nav.

**Not done here** (explicitly deferred, not silently skipped): Document Approval's Detail page
and Project Management's board (Case 19, CAP-V14 Tier 3) still run entirely outside the
substrate — one View, one component type, was the whole pilot. Phase 14's own precondition is now
*partially* satisfied by one real route with real traffic potential, not fully — a future
decision point, not assumed here, is whether to keep expanding live-wiring (more Views/component
types) before Phase 14's tuning work would have enough real signal to act on.

---

# 17b. Live Wiring Pilot — Dependency DAG + Execution Planner on the Live Path (2026-09-12)

**Not one of the original 14 phases** — a direct continuation of 17a, closing the specific gap
17a's own closing note named: `internal/composable`'s Dependency DAG/Execution Planner (Phase
7/8) had never run against a real HTTP request, only against Go tests
(`planner_seed_test.go`'s own `TestGroupByMachineAgainstApprovalCase`/
`TestBuildExecutionPlanAgainstApprovalCase`). This is the single largest unmet item in
`composable-apps-trial.md` §15.3 ("execution planning is present on the live runtime path")
standing between the pilot and a real trial.

**Goal:** prove the Dependency DAG/Execution Planner boundary is exercised by a live request,
without claiming physical execution — `LogicalQuery`/CR-05 and Phase 9's own real
query-execution gap remain exactly as open as before this change; only the logical
planning/diagnostics boundary moves onto the live path.

**What was built:** `ComposablePreview` (`app/internal/handler/composable_preview.go`) gained
one new side-channel step, `explainComposablePlan`, added after the existing render-path code
(unchanged — same guards, same `DefaultListView`/`LowerCardRowComponent`/`ResolveRecordSummary`
pipeline 17a already proved). It calls `composable.LowerPage(machine, viewIdx, machineIdx)` on
the **whole machine** (every View, not just the one cards list the page renders) — the same
call the Go test above already proves against `mch_approval_document` — then
`composable.BuildDependencyDAG` + `composable.BuildExecutionPlan(dag).Explain()`, using
`composable.ResolveSecurityScope(machine, role)` for the security scope (a known, documented
simplification: the caller's first held role only, since `roleForApp` returns a set and
`ResolveSecurityScope` takes one — real enforcement stays on `guard.CanRead`'s own multi-role
union logic, completely unaffected). The result is logged (`slog.Info("composable_plan", ...)`,
same `correlation_id` pattern CAP-I04 already uses) and rendered into a hidden
`data-composable-plan` attribute in `composable_preview.templ` — a data attribute rather than an
HTML comment (untested territory for this codebase's `.templ` files) — so it is inspectable from
the response body alone, no server-log access required, per Principle #9 ("inference is
inspectable"). Any lowering error is logged and swallowed; the diagnostic never fails the page.

**Proof:** `conformance/tests/241_composable_pilot.sh`'s new T265 — hits the same
`composable-preview` URL and asserts the response body's `data-composable-plan` attribute
reports `ExecutionPlan: 1 group(s), naive=4 dedup=3` and `mch_approval_document: 3 node(s)` —
the exact real structural fact the Go test already proved (three distinct Datasets, `vw_ad_form`/
`vw_ad_all`/`vw_ad_pending`, with `vw_ad_pending`'s own real duplicate consumption collapsing
naive=4 to dedup=3), now observed from a live HTTP response instead of only a Go test. Full
suite: 271 passed, 0 failed (`./scripts/local-ci.sh`) — no regression, one net new test.

**Not done here** (explicitly deferred, not silently skipped, same posture as 17a): the plan is
still purely a diagnostic side-channel — nothing about how records are actually fetched changed
(`h.records.List` per view, unchanged), so no physical query reduction happens yet; that remains
CR-05/Phase 9's own open gap. Dashboard tiles (`vw_ad_dashboard`) are counted in the plan but
still not rendered as a composable View Model on this route — only the pending-list cards
render, exactly as 17a shipped. Document Approval's Detail page, Project Management, and a
production cutover of this route all remain out of scope, same as 17a's own closing note.

---

# 17c. Detail-View Children Lowering (2026-09-12)

**Not one of the original 14 phases** — closes the specific gap Phase 12's own benchmark work
named while grounding its "Approval detail + stepper" scenario: `internal/composable/ui.go`'s
`LowerPage`/`LowerChildren` recursed into a View's own declared `Config.Children` only when that
View's type was `page` — a `detail`-type View's own Children (CAP-V20's embedded-view mechanism,
decision stepper/coordinate placement) were silently never lowered at all, invisible to the
Dependency DAG/Execution Planner entirely. This is the concrete reason
`composable-apps-trial.md` §15.1's "composed dashboard/**detail**" bullet stayed unmet.

**What was built:** `LowerPage` (`app/internal/composable/ui.go`) now builds each View's own
`view_ref` node first, then recurses into that View's own Children when its type is `page` OR
`detail` — a `page`-type View's Children still populate the whole machine page's own `Slots`
(unchanged), while a `detail`-type View's Children now populate **that view_ref node's own**
`Slots` instead (so two Detail Views on one machine, or a Detail alongside a Page view, can never
collide on one shared slot map — the reason this wasn't simply "add `|| ViewTypeDetail`" to the
existing `page.Slots` line). `LowerChildren` itself needed no change — its own cross-machine
`viewIdx`/`machineIdx` resolution and cycle/depth guards already do the right thing once actually
called with a Detail View's Children.

**Proof:** a real Go test against real seeded metadata, not a synthetic fixture —
`TestLowerPageAttachesDetailChildrenToOwnNode` (`ui_seed_test.go`) against
`seeds/042_inline_view_composition.sql`'s own `vw_as_detail` (`mch_approval_step`'s Detail view),
which embeds `vw_ad_progress` (a `decision_stepper` View on the **different** Machine
`mch_approval_document`) and `vw_as_place` (`coord_placement`, same Machine) — a genuine
cross-machine composition case, proving `machineIdx` resolution works through this new call site
too. One honest limitation stated up front and preserved in the test/route framing:
`BuildDatasetFromView` has no case for `decision_stepper`/`coord_placement` (same as `detail`
itself), so both embedded nodes carry `Dataset == nil` even after this fix — extending that is
real capability work with no case forcing it yet, not attempted here (roadmap Principle #8:
evidence before optimization, not manufacturing a number to move).

**Separate, honestly-scoped live-path extension:** `ComposablePreview`
(`app/internal/handler/composable_preview.go`) no longer 400s for a machine with no
`display: cards` default List View (every machine except `mch_approval_document`, until now) —
it renders an empty card grid instead and still runs 17b's `explainComposablePlan` side-channel,
reaching a **second** real machine's Dependency DAG/Planner boundary on the live path for the
first time. `mch_approval_document`'s own existing behavior (T262-265) is provably unchanged
(same code path, now inside an `if` instead of unconditional). Proof: T266 (`GET
/mch_approval_step/composable-preview` now 200, was 400) and T267 (the same response still
carries a real, non-empty plan diagnostic). T267 deliberately does **not** assert specific
node/group numbers — per the `Dataset == nil` limitation above, `mch_approval_step`'s own
naive/dedup counts are identical whether or not this session's `ui.go` fix is applied (2 nodes:
`vw_as_form`, `vw_as_progress`; the `vw_as_detail` subtree contributes none). T266/T267 prove the
*route* now reaches this machine's planner boundary, not that the Detail-children fix changed any
observable count — two different, both real, claims kept separate on purpose. Full suite: 273
passed, 0 failed (`./scripts/local-ci.sh`) — no regression, two net new tests.

**Not done here** (explicitly deferred, not silently skipped, same posture as 17a/17b): rendering
`mch_approval_step`'s Detail page as an actual composed page (RecordSummary/StatusBadge/ActionBar
for its fields, matching what `record_crud.go`'s real `Handler.Detail` already shows today — full
field list, SLA badges, CAP-V06 reverse-reference child-lists, CAP-P02-filtered
`permittedEvents`, CAP-V20 embed rendering) remains entirely out of scope; this pilot only proves
the DAG/UI-IR tree now has the right *shape*. Document Approval's own Detail page, Project
Management, and a production cutover of this route all remain out of scope too, unchanged from
17a/17b's own closing notes.

---

# 17e. Full Render-Output Equivalence for Document Approval's Cards (2026-09-12)

**Not one of the original 14 phases** — closes the specific precondition §2a's own rule 6
(`17d`) named for any future per-page cutover: "only once render-output equivalence is actually
proven for that specific page." Grounded directly against the real code before starting (not
assumed): equivalence did **not** hold for `mch_approval_document`'s cards (`vw_ad_all`), the
most mature composable case — the composable preview had no avatar, no status badge (despite
`StatusBadge`'s own component contract existing and being proven by tests since Phase 5, zero
non-test call site ever constructed one), and only a 2nd-column subtitle instead of every
remaining column joined `" · "`. Owner decision (this conversation, via AskUserQuestion): close
all three gaps in one increment rather than incrementally.

**What was built:** `internal/composable/view_lowering.go`'s new `CardBadgeField` names which
column is the status badge — the **last** `value_list`-typed non-title column, exactly mirroring
`internal/ui/list.templ`'s own `cardSummary` convention (its own comment: "caught live
2026-09-10... an app's cards Views always put Status last"). `LowerCardRowComponent` gained a
`*model.Machine` parameter to look this up, and now excludes the badge column from the subtitle,
joining every other non-title column's id into `subtitle_field`'s own value as a comma-separated
list — `ResolveRecordSummary` (`viewmodel_resolve.go`) splits and joins their resolved Display
values with `" · "`, byte-identical to before when there is exactly one id (every pre-17e
caller). `ComposablePreview` (`composable_preview.go`) resolves a parallel `[]composable.
StatusValue` via `LowerStatusBadge`/`ResolveStatusValue`, and the templ
(`composable_preview.templ`) now calls the exact same `RecordSummaryCard`/`StatusBadge`/`Avatar`/
`initials` building blocks the real feature uses (same `ui` package, no import needed) instead of
hand-rolled lookalike markup — including matching `listCards`' own wrapper (`space-y-3`) and
empty-state copy ("No records yet. Create the first one."), not just the card's own inner shape.

**Proof:** two new tests, not one, deliberately kept separate — `conformance/tests/
241_composable_pilot.sh`'s T268 establishes the ground truth (the REAL cards page shows the
expected avatar initials, joined subtitle `Report · Sequential`, and badge `Draft` for a freshly
created record — `vw_ad_all`'s own columns are `[title, document_type, approval_mode, status]`,
and the LAST three are all `value_list`-typed, the exact multi-eligible-column scenario
`cardSummary`'s own comment names), and T269 proves `/composable-preview` now reproduces the
identical three facts for the identical record — real equivalence, not merely a passing
resemblance. `Handler.ComposablePreview` was refactored (extracting
`resolveComposableCardSummaries`) to stay under Gate 3's cyclomatic-complexity ratchet rather
than requesting a baseline exception, the same lesson `17c` already established. Full suite: 275
passed, 0 failed (`./scripts/local-ci.sh`) — no regression, two net new tests.

**Not done here** (explicitly deferred, not silently skipped): `SlaBadge` is not reproduced — no
column on `vw_ad_all` is SLA-eligible, so nothing in this page's own real rendering ever reaches
it; building a general composable `SlaBadge` component now would be unforced capability work.
Every other page (`mch_approval_step`'s Detail/board-less cards, Document Approval's own
Detail/Dashboard, Project Management) remains exactly where 17b/17c/17d left it — this increment
proves equivalence for `vw_ad_all` specifically, not universally. **Actual cutover of the real
List route remains untouched and undecided** — §2a's rule 6 treats it as its own separate, later
decision even now that equivalence is proven for this one page; `record_crud.go`/`views.go` are
not modified by this change at all.

---

# 17f. Full Functional Equivalence — Search, Sort, Pagination (2026-09-12)

**Not one of the original 14 phases** — continues closing §2a rule 6's own cutover precondition
one dimension further than 17e: a real cutover needs the whole PAGE to behave equivalently, not
just one card's own visual shape. Grounded directly against the real code before starting (two
Explore passes, not assumed): the real List handler (`record_crud.go`) also does sort, free-text
search (`?q=`), and pagination (`?page=`, 25/page), none of which `/composable-preview` did
before this session.

**Two categories of suspected gap turned out NOT to be gaps at all, once actually checked** —
the same "grounding pays off" lesson 17e's own StatusBadge discovery already taught:

- **Archive/Restore/Move-up/down buttons** live *only* inside `list.templ`'s own `else` branch
  of `if opts.Cards { @listCards(...) } else { <table>...buttons...</table> }` — cards mode
  never renders them, for any role, on any machine. Not deferred; structurally inapplicable to a
  cards-display page by the real feature's own design, worth recording as a general fact (not
  just true for this one page).
- **The "View Archive" toggle** (gated on `opts.CanDelete`) is unreachable on
  `mch_approval_document` regardless: no permission row for any of its three roles (`Submitter`,
  `System`, `Approver`) sets `can_delete` — it stays the migration default `false`. No real,
  observable target exists to prove equivalence against, so — per "prove, don't assume" — this
  stays deferred rather than built unprovably.

**What was built:** three functions extracted verbatim from `record_crud.go`'s own `List` into a
new file, `internal/handler/list_query.go` (not grown further into `record_crud.go` itself,
already over Gate 2's own LOC budget — `sortFieldFor`, `searchListRecords`,
`paginateListRecords`), and two templ components extracted verbatim from `list.templ`'s own
`List` (`searchBox`, `paginationBar`, reusing the already-package-level `pageLink` unchanged).
`List` itself now calls all five instead of its own inline code/markup — a pure, behavior-
preserving refactor (verified by the full existing suite staying green, not a new test
re-proving `List` itself). `ComposablePreview`/`resolveComposableCardSummaries`
(`composable_preview.go`) now call the same five: `sortFieldFor(view)` replaces the `"",""`
literal `RecordStore.List` previously received (correct for `vw_ad_all` only by coincidence
before this), `searchListRecords`/`paginateListRecords` run in the same filter → search →
paginate order `List` already uses, and `composable_preview.templ` calls the same
`searchBox`/`paginationBar` plus the three `Export`/`Import`/`New` chrome links, gated on
`opts.Cards` so the 17c no-cards-view fallback shows neither (nothing real to search/paginate
there).

**Proof:** T270 (a matching `?q=` keeps a freshly created record, a non-matching one excludes
it) and T271 (`?page=999`, after creating 26 fresh records to guarantee more than one real page
regardless of the rest of the suite's own incidental accumulation, clamps to the real last page
with real content, not an empty/error response) — both prove genuine wiring, not merely code
that compiles. Full suite: 277 passed, 0 failed (`./scripts/local-ci.sh`) — no regression
(confirming the `List`/`list.templ` refactor is behavior-preserving), three net new tests.

**Not done here** (explicitly deferred, not silently skipped): sort's own behavioral change is
unobservable on `vw_ad_all` specifically (its `DefaultSort` already coincided with the old
`"",""` behavior) — stated honestly rather than manufacturing a misleading assertion; the fix is
structural correctness for any OTHER cards-display View with a different `DefaultSort`, not
provable on this one page. Archive/Restore/Move/Archived-toggle remain out of scope per Context
above. **Actual cutover of the real List route remains untouched and undecided** — §2a rule 6
still treats it as its own separate, later decision, even now that both visual (17e) and
functional (17f) equivalence are proven for this one page; `record_crud.go`'s own route
registration is not touched, only its internals refactored.

---

# 17g. Cutover — Document Approval's Cards Page (2026-09-12)

**Not one of the original 14 phases — the first time in the whole 17a-17g series that
PRODUCTION-serving code was modified, not just an additive preview route.** With visual (17e)
and functional (17f) equivalence both proven for `mch_approval_document`'s cards page
(`vw_ad_all`), §2a rule 6's own precondition for a real cutover was finally met. Two more
equivalence facts were checked directly against the code before committing to this (same
"ground before scoping" discipline as 17e/17f): CAP-P06 hidden fields (no role on this Machine
declares any — real, deferred gap: `LowerCardRowComponent`/`CardBadgeField` read
`view.Config.Columns` directly with no caller-filtering hook, a latent issue for a future
Machine that DOES declare hidden fields, not this one) and CAP-V05/V09 declarative Filter
(`vw_ad_all` has none configured — cheap to close regardless, done as part of this change since
it needed no composable-package signature change).

**What was built:** `resolveComposableCardSummaries` (`composable_preview.go`) now also applies
`h.applyListFilter` (closing the Filter gap above) and returns raw `searchQuery`/`page`/
`totalPages` instead of a pre-built `ui.ListViewOptions`, so each caller can merge its own
additional fields. `record_crud.go`'s `List` gained a new branch —
`if view.Config.Display == "cards" && !archived` — delegating to a new
`listCardsViaComposable` (extracted to stay within Gate 3's complexity ratchet), which calls the
exact same `resolveComposableCardSummaries`/`explainComposablePlan` `/composable-preview` has
used since 17b/17e/17f. `list.templ`'s `List` gained `summaries`/`badges`/`planExplain`
parameters and a new `composableCardGrid` shared component (extracted from what was
`composable_preview.templ`'s own inline card-loop markup, now called by both templates — one
markup, not two copies that could drift) — `summaries != nil` is the discriminator distinguishing
this new branch from the still-kept-for-generality archived+cards path (`resolveComposable
CardSummaries` always returns a non-nil, if possibly empty, slice; that other path passes a
literal `nil`). `ListContent`'s own embedded rendering (CAP-V10, `vw_ad_page`'s "Pending
Documents" section) is completely untouched — confirmed directly from `list.templ:195-198` that
`listCards`/`cardSummary`/`cardBadge` are shared by `List` AND `ListContent`, so they stay alive
and unchanged; only `List`'s own standalone branch stopped calling them for this one View.

**Proof:** T272 (the real `GET /mch_approval_document` — not `/composable-preview` — now carries
the same `data-composable-plan` diagnostic 17b proved on the preview route, the honest signal
that this specific request was actually served by the composable pipeline, not merely "still
shows a card correctly" which T248 already established and would hold regardless of which code
path renders it), T273 (`?q=` works identically on the real route), T274 (`?page=999` clamps
identically on the real route) — reusing the exact records T270/T271 already created. Full
suite: 280 passed, 0 failed (`./scripts/local-ci.sh`), including T248 and T253-257 (the composed
page's own "Pending Documents" section, `ListContent`-rendered, untouched and still passing
unchanged) — the real regression check proving 17e/17f's equivalence work was genuinely
complete, not merely appearing to be.

**Not done here** (explicitly deferred, not silently skipped): CAP-P06 hidden-fields filtering
inside `LowerCardRowComponent`/`CardBadgeField` remains a named, real gap for a future Machine
that needs it — no forcing case today. No other Machine/page is touched: `mch_approval_step`,
Document Approval's own Detail/Dashboard, Project Management, and `vw_ad_page`'s embedded
section all remain exactly where 17b-17f left them. **The live `menata-runtime` process was not
restarted** — per `app/CLAUDE.md`'s own "Server lifecycle" section, that stays a separate,
deliberate, manual action, not implied by this commit.

---

# 17h. First Live Board Rendering — Project Management (2026-09-12)

**Not one of the original 14 phases — the first time a DIFFERENT real application (Case 19 /
Project Management) got any live composable footprint at all.** 17a-17g all worked on Document
Approval. Grounded directly against the code (two Explore passes) before scoping: `mch_pm_card`'s
board (`vw_pmc_board`, `group_field: fld_pmc_list`, a `reference` field targeting `mch_pm_list`)
is a real, already-shipped, dynamic-lane board — see the dated correction appended to §17 (Phase
13) above. `/composable-preview` never reached Board views before this session (`DefaultListView`
only ever selects `Type == "list"`), and `internal/composable` had zero Board/lane-shaped code —
genuinely new design work, not a reuse of something already proven, unlike every gap 17e/17f/17g
closed. One fact made the new resolver simpler than it could have been: `BuildDatasetFromView`'s
own Board branch already includes `GroupField` when discovering `Relations`
(`dataset.go:105-130`), so a reference-typed `group_field` already produced a `RelationRef` today
— no Dataset-layer change needed, only a resolver that uses this fact. Owner decision (this
conversation, via AskUserQuestion): build the real rendering now, not just a Go-test proof or a
diagnostic-only live check.

**What was built:** two new View Model types (`viewmodel.go`) — `RecordRef` (an already-fetched
record's id+data, since this package does no I/O itself) and `BoardLane` (`LaneRecordID` +
`Cards []CollectionItem`) — and `ResolveBoardLanes` (`viewmodel_resolve.go`), which groups
`cardRecords` into one `BoardLane` per `laneRecords` entry, in order, so every lane record
produces a lane even with zero matching cards (CAP-V14's own "an unused lane still renders
empty" rule, mirroring `internal/handler/views.go`'s real Board handler exactly) — and fails loud
if the Board's `GroupBy` field has no discovered `Relation` (the fixed-value-list lane case,
CAP-V14 Tier 2, e.g. Kanban Lab's own `vw_kbt_board`, is deliberately not handled here).
`ComposablePreview` (`composable_preview.go`) gained a new `else if h.interp.Get().BoardView(...)`
branch (reached only when there's no cards-display List) calling the new
`resolveComposableBoardLanes`, which fetches lane and card records, converts them to
`RecordRef`s, calls `ResolveBoardLanes`, and resolves each lane's real display label by calling
`displayLabel` (`internal/handler/format.go`) exactly as the real Board handler already does —
`internal/composable` can't reach that function itself (Gate 5), so this is the one place the
label logic runs, not a second copy. `composable_preview.templ` gained `composableBoardLanes`, a
new, deliberately narrower-than-`board.templ` rendering (plain cell text, no
`RecordSummaryCard`, no drag-and-drop) with a `data-lane` attribute per lane so a conformance
test can isolate one lane's own content the same way `240_dynamic_board_lanes.sh` already does
for the real board.

**Proof:** a real Go test (`TestResolveBoardLanesAgainstProjectManagement`,
`viewmodel_seed_test.go`) against `seeds/052_project_management.sql`'s own data — the same
lane/card grouping `240_dynamic_board_lanes.sh`'s own T259 already proves over HTTP, plus one
case T259 itself never exercises: a freshly created, zero-card "Backlog" list still produces an
empty `BoardLane`. Live: a new conformance file, `conformance/tests/242_composable_pm_board_
pilot.sh` (T275-T277) — reusing `PM_MEMBER`/`TODO_LIST_ID`/`DOING_LIST_ID`/`DONE_LIST_ID`/
`WIREFRAME_CARD_ID` already resolved into shared shell scope by `240` (file order 240 < 242),
written against the POST-T260 state (`WIREFRAME_CARD_ID` already moved into `DOING_LIST_ID` by
the time 242 runs) — T275 proves `/composable-preview` reaches Board rendering at all for
`mch_pm_card`, T276 proves lane grouping matches the real board's current state with real
display-name headers, T277 proves a fresh empty lane still renders. Full suite: 283 passed, 0
failed (`./scripts/local-ci.sh`) — no regression, three net new tests. The plan's own pre-split
of `resolveComposableBoardLanes` into small helpers (`boardRelationTarget`, `toRecordRefs`) paid
off: Gate 3 passed on the first attempt, no rework cycle like 17c/17e each needed once.

**Not done here** (explicitly deferred, not silently skipped): visual/functional equivalence to
`board.templ` (no `RecordSummaryCard`, no drag-and-drop, no `board-move` wiring) — a later
increment's job, mirroring 17a's own arc before 17e/17f later closed the equivalent gap for
cards. Fixed-value-list lanes (CAP-V14 Tier 2) stay unhandled by `ResolveBoardLanes`, failing
loud rather than silently mishandled. No cutover of the real `/mch_pm_card/board` route is
anywhere near proven enough to consider — this reaches only a 17a-equivalent starting point for
Project Management, not a 17g-equivalent endpoint. The live `menata-runtime` process was not
restarted, same standing rule as every prior increment.

---

# 17i. Visual Equivalence for the Board Preview (2026-09-12)

**Not one of the original 14 phases** — closes the visual gap 17h's own closing note named,
mirroring 17e's arc for Document Approval's cards. Grounded by reading `internal/ui/board.templ`
directly in full (not paraphrased): lane wrapper, header, body, and empty-state classes, and
whether each card is a real link at all, all differed from `composableBoardLanes`' own 17h
markup. `CollectionItem` already carried `RecordID` (set correctly since 17h), so this needed no
`internal/composable` or handler Go change — pure templ.

**What was built:** `composableBoardLanes` (`composable_preview.templ`) adopted `board.templ`'s
own classes verbatim for the lane wrapper (`w-72 shrink-0 rounded-lg border border-slate-200
bg-slate-50`), header (`<h2>` with the count in a parenthesized nested `<span>`, matching
`board.templ:36-39` exactly), body (`min-h-16 space-y-2 p-2`), and empty-state
(`rounded-md border border-dashed border-slate-200 p-3 text-center text-xs text-slate-400`).
Each card became a real `<a href=".../{RecordID}">` with the real card's own visual classes
(`rounded-md border border-slate-200 bg-white p-3 text-sm shadow-sm hover:shadow`) wrapping
`text-slate-700` cells. **Deliberately excluded, not overlooked:** `board-card`/`board-lane`/
`board-lane-body`, `cursor-move`, `draggable`, and the drag Hyperscript — those class/attribute
names exist *for* drag-and-drop; copying the name without the behavior would misrepresent
capability, not prove equivalence. This is the same write-path boundary this conversation already
settled (006-runtime-model.md: Action is Domain-plane, this package governs Data/Experience only).

**Proof:** T278 (the real board renders `WIREFRAME_CARD_ID` as a real link carrying those exact
visual classes — ground truth) and T279 (`/composable-preview` reproduces the identical link +
classes for the same record). **A real regression was found and fixed by this phase's own
verify step** (exactly what "ratchet, never regress" exists to catch): 17h's own T276 checked for
the literal substring `>To Do<` (etc.) — true only by coincidence against 17h's flat
`<span>{name}</span>` markup. Once the header markup changed to match `board.templ`'s own
nested-`<span>`-for-the-count structure, "To Do" is followed by whitespace before the next tag,
not `<` directly — T276 started failing. Fixed by loosening it to `>To Do` (no trailing `<`),
which is exactly what T259's own, already-correct check for this same real markup has done all
along; T276 now checks what T259 already established was the right pattern. Full suite: 285
passed, 0 failed (`./scripts/local-ci.sh`) after the fix — two net new tests plus one corrected
existing one.

**Not done here:** drag-and-drop / `board-move` wiring (a mutation, out of scope per the
write-path boundary), fixed-value-list lanes (CAP-V14 Tier 2), and any cutover of the real
`/mch_pm_card/board` route — this closes 17h's own visual follow-up (the equivalent of 17e for
cards), not a 17g-equivalent cutover step. The live server was not restarted.

---

# 17j. Real-Traffic Composability Benchmark (2026-09-12)

**Not one of the original 14 phases — the largest single increment in the 17-series, and the
first to run sustained concurrent traffic rather than single requests.** Closes `CR-19`. Grounded
before scoping: Phase 12 (§16) already built the *structural* metrics
(`composable.MeasureComposition`, logical/DAG nodes, naive-vs-dedup query count, execution width)
but only ever called them from Go tests; every execution metric (`composable-apps-trial.md`
§12.1: p50/p95/p99, CPU, memory, DB pool, planner/render time, rows scanned/returned, cache hit
ratio) was honestly unbuilt, and no load-testing tool exists anywhere in this repo. Owner
decision (this conversation, via AskUserQuestion): attempt the full Phase 12 battery.

**What was built:** `MeasureComposition` gained its first live call site — `explainComposablePlan`
(`composable_preview.go`) now returns `(string, composable.BenchmarkMetrics)`, and both call
sites (`/composable-preview` and the 17g cutover's `listCardsViaComposable`, moved into this file
from `record_crud.go` since this phase's own instrumentation pushed that already-oversized file
past Gate 2) wrap their own data-fetch/plan/render phases in `time.Since()` and log one new
structured line, `composable_benchmark` (`logComposableBenchmark`), joinable to `slogAccessLog`'s
own line by the same `correlation_id` both already use. `cmd/server/main.go` gained an opt-in-only
`GET /debug/pool-stats` (`pool.Stat()` as JSON), registered only when `ENABLE_DEBUG_ENDPOINTS=1`
— dormant on every real deployment — and added to `isPublicPath` (same trust class as `/health`,
no user data, only connection counts). A new standalone Go program, `scripts/loadtest/main.go`
(stdlib `net/http` only, no new external dependency — confirmed none of wrk/vegeta/k6/hey exist
in this repo before writing it), fires concurrent requests through a real `net/http/cookiejar`
loaded from a Netscape-format jar (the same file `curl -c`/`conformance/lib.sh`'s own
`session_for` already produce), paced by an optional `-rate` flag, and reports p50/p95/p99
latency split by status class (`latency_ms_2xx` isolated from `latency_ms_all`, since a
429/403 short-circuits before any real work and would otherwise misrepresent "how fast a real
request is"). `scripts/benchmark-composable.sh` orchestrates all of it, mirroring `local-ci.sh`'s
own isolated-schema/scratch-port throwaway pattern exactly (never the live `menata-runtime`
process) plus a `/proc/$SERVER_PID` CPU/memory sampler (`VmRSS`, `utime+stime` deltas) needing no
server-code instrumentation at all.

**Two real, previously-unknown things this benchmark's own first run found, not just built:**
1. **The server's own per-IP rate limiter (`ratelimit.go`, 30 req/s burst 120) dominated the
   first attempt** — 66-97% of requests came back `429`, never reaching the composable pipeline
   at all, because the load driver fired as fast as possible from one IP. Fixed two ways: split
   latency reporting by status class (so a mixed sample can't misrepresent itself even if this
   happens again), and paced dispatch at 20 req/s (`-rate`, safely under the real limit) — a
   real, honest constraint on how "real traffic" can be simulated from one test client, not a
   bug to hide.
2. **A freshly migrated+seeded schema has zero real records for `mch_approval_document`** —
   seed files only declare Machine/Field/View/Permission rows, never data; conformance tests are
   what create real Documents, and this script runs none of them. The first clean run reported
   `rows_returned=0.0` for the cutover route — true, but unrepresentative. Fixed by seeding 20 real
   Documents via real `POST` requests (same shape `010_case1_3_core.sh`'s own `AD_SEQ_DATA` uses)
   before measuring.

**The real numbers, from the corrected run** (`BENCH_CONCURRENCY=10`, `BENCH_REQUESTS=100`,
`BENCH_RATE=20`, all defaults):

| Scenario | Status | p50 / p95 / p99 (ms) | logical/DAG nodes | naive→dedup queries | rows returned |
|---|---|---|---|---|---|
| 1: PM board preview (1 component→1 Dataset) | 100×200 | 3.86 / 7.15 / 10.19 | 5 / 3 | 3→3 (no shared Dataset to dedup) | 4 |
| 3: Document Approval cutover (10 components→3 shared) | 100×200 | 3.90 / 7.31 / 8.84 | 11 / 3 | 4→3 (the real, already-known dedup) | 20 |
| 7: same URL, denied role (security-scope substitute) | 100×403 | 2.77 / 6.38 / 7.00 | — (no composable_benchmark line at all — CanRead denies before the pipeline ever runs) | — | — |

`data_time_ms`/`planner_time_ms`/`render_time_ms` all averaged well under 1ms for every
scenario — at this data scale (≤20 rows), the composable pipeline's own overhead is not the
latency's dominant term; most of each request's few milliseconds is everything ELSE a real HTTP
request already does (session/CSRF/workspace-tx middleware, `net/http` itself), unchanged by this
initiative. DB pool: `acquired=1 idle=1 total=2 max=4` — nowhere near saturated at this load.
CPU/memory: ~3% average CPU, ~31MB RSS, over each ~5-second run — light, as expected at 20 req/s.
**Cache hit ratio: N/A, not fabricated — no cache exists anywhere in this runtime.**

**Not done here** (named, matching Phase 12's own precedent for its scenarios 5/6, not silently
skipped): scenarios 2 (10 independent Datasets), 4 (1 Dataset→multiple projections), 5 (mixed
OLTP+analytics), 6 (100 concurrent workspaces), 8 (nested Project board), 9 (Approval detail +
stepper) — none has any live route to measure today (Detail-page lowering incomplete, no live
nested Project Management page, no analytics-pool/multi-workspace distinction exists). No
naive-vs-planner physical-work comparison is claimed — `internal/store` is still never called
differently based on the plan (CR-05/Phase 9's own open gap); this run reports real latency/CPU/
memory *as currently deployed*, not a controlled A/B. The live `menata-runtime` server was never
touched — everything ran against a fresh throwaway instance, exactly like `local-ci.sh`. Full
suite re-verified unchanged after every change in this phase: 285 passed, 0 failed.

---

# 18. Phase 14 — Production Hardening

Only after real trial workloads:

- tune cost model thresholds;
- add persistent/longer-lived plan caching if evidence justifies it;
- tune batching vs parallelism;
- add materialization where warranted;
- add streaming/lazy loading where needed;
- expose inference and execution diagnostics;
- add operational dashboards for planner health;
- document fallback/degradation policy;
- ratchet performance benchmarks in CI where runtime cost is stable enough.

No optimization becomes a new semantic metadata concept merely because it is useful operationally.

---

# 19. Dependencies and Ordering

The critical dependency chain is:

```text
Documentation contract
        ↓
Canonical semantic model
        ↓
Data IR + Projection
        ↓
Context / Binding
        ↓
UI IR
        ↓
Component contract
        ↓
View lowering
        ↓
Dependency DAG
        ↓
CEP
        ↓
Query / physical integration
        ↓
Performance proof
        ↓
Trial migration
```

Do not implement CEP before Data IR/Dataset/Projection and dependency identity exist. A planner without a stable logical dependency model will optimize handler-specific behavior and recreate the architecture problem this roadmap is intended to solve.

---

# 20. Definition of Done for the Program

The composable runtime program is complete enough for production when:

- [ ] Domain/Data/Experience model exists in runtime code;
- [ ] Dataset and Projection are real semantic contracts;
- [ ] Context/Scope/Binding are validated;
- [ ] UI IR is the canonical input to the reference renderer for composed experiences;
- [ ] existing Views lower through the same substrate where applicable;
- [ ] generic Components have bounded contracts and a static dispatch seam;
- [ ] dependency DAG is inspectable;
- [ ] CEP is a first-class runtime boundary;
- [ ] physical execution is bounded and security-aware;
- [ ] Query Planner remains separate from CEP;
- [ ] inference affecting execution is inspectable;
- [ ] composition conformance is executable;
- [ ] performance benchmark demonstrates physical economy;
- [ ] Document Approval and Project Management run on the shared substrate;
- [ ] no existing capability conformance regression is introduced.

---

# 21. Relationship to Existing Plans

This roadmap intentionally reuses rather than replaces existing work:

- `007-composable-runtime-architecture.md` — normative target architecture;
- `composable-runtime-blueprint.md` — transformation blueprint/current-state inventory;
- `composable-runtime-architecture-map.md` — terminology and cross-document contract;
- root `roadmap.md` — capability discovery/admission evidence;
- `app/ROADMAP.md` — completed production graduation history;
- `capability-registry.md` — capability-level status and proof;
- `nfr-standards.md` — security/performance/architecture quality gates;
- `benchmarks/004-scale-architecture-study.md` and related scale studies — existing physical scale evidence;
- `composable-apps-trial.md` — trial applications and shared-substrate validation.

If implementation evidence contradicts a target assumption, update the blueprint/benchmark first and revise the target only through an explicit architecture decision. Do not silently change the meaning of a Tier 1 concept in implementation code.
