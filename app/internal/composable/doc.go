// Package composable is a NEW package — it did not exist in prototype/go, and it has
// no counterpart to graduate from.
//
// Scope: Phase 1 (composable-runtime-roadmap.md §5, CR-01), Phase 2 (§6, CR-02 through
// CR-05), Phase 3 (§7, CR-06), Phase 4 (§8, CR-07/CR-10), Phase 5 (§9, CR-08/CR-09),
// Phase 6 (§10, CR-10 again), Phase 7 (§11, CR-11/CR-16), Phase 8 (§12, CR-14/CR-18),
// Phase 9 (§13, CR-05/CR-13), Phase 10 (§14, CR-25), Phase 11 (§15), and Phase 12 (§16)
// — the minimum Go model that can represent a composed page/data requirement as data,
// adapted FROM the existing Application Model (internal/model) rather than replacing it.
// Machine/Field/Event/Constraint/Permission remain the real Domain plane; View/ViewConfig
// remain the real source of truth for rendering. This package only adds a read-only
// normalization layer on top: given a *model.Machine and its Views, produce a canonical,
// deterministic representation (NodeIdentity, Dataset, UINode) that doesn't require
// invoking a View handler to know "what page and what data does this Machine imply."
//
// Phase 2's own Dataset is content-addressed (identity.go's datasetIdentity) -- two
// Datasets built from different experience shapes (a list, a board, a dashboard section)
// with the same logical requirement compare equal, which is what makes Phase 2's exit
// criteria ("two different experience consumers can use the same Dataset definition")
// checkable rather than asserted. Phase 3's context_adapt.go builds on that:
// ScopeForDataset seeds a nested Scope directly from a Dataset's own Relations (Phase 2
// output, not re-derived), DeriveChildScope is the actual mechanism behind "child scopes
// cannot access undeclared parent values", and ValidateBinding is what makes "bindings
// are validated before rendering" and "security context is never downgraded by a child
// binding" checkable rather than aspirational.
//
// Phase 4's ui.go is where those two strands actually meet a real node tree for the
// first time: UINode (experience.go) replaces Phase 1/2's flatter PageNode/ComponentNode
// (nothing outside this package ever used them, same latitude already used to replace
// DatasetRef with Dataset in Phase 2 and redesign Scope.Domains in Phase 3) --
// LowerPage/LowerApplication give every view_ref node its OWN Dataset (closing the
// "Phase 2 associates a Dataset per component node, not per page" TODO Phases 1-2 left
// behind) and validated Bindings (closing Phase 3's own "Scope/Binding are not yet
// attached to any node" gap), and LowerChildren lowers CAP-V10/CAP-V20's existing
// Children/embedding mechanism (model.ChildViewRef) into UINode's own Slots. LayoutNode,
// a Phase 1 placeholder that nothing populated, has now graduated into the UINodeLayout
// kind on this one unified node type rather than staying a second, parallel shape.
//
// Nothing in internal/handler, internal/ui, or internal/router calls this package yet —
// that wiring (View lowering) is Phase 6 territory, gated on this phase's own exit
// criteria and proven equivalence first, per the roadmap's dependency chain (§19). Phase
// 4 also does not produce actual "render inputs" (Phase 10's own Renderer/View Model
// separation) -- UI IR is as far as this phase goes.
//
// Phase 5's component.go/component_adapt.go is what finally gives UINodeComponent a real
// meaning: a closed ComponentType set (exactly the seven §9 names -- Heading, Text,
// Collection, RecordSummaryCard, Metric, StatusBadge, ActionBar, not a starting point for
// more), a static componentRegistry mapping type -> ComponentContract (a plain Go map,
// never a runtime-registered plugin), ValidateComponent (the closed-set/required-property/
// no-property-bag/Dataset-Children-Actions enforcement), and ResolveComponent (the only
// way to produce a valid component node). ui.go's lowerStaticContent migrates
// PageContent's own "heading"/"text" types through this contract in place -- "button"/
// "image" still fall back to a bare static_content node, exactly "migrate only a small
// number of proven generic components first" (§9's own Build section). Every one of the
// seven is proven against real seeded metadata (Document Approval's own Status field/
// Events/Approval-Step list, seeds/050's Display:"cards" list, Action Lab's and
// Approval's own dashboard sections -- the latter pair also proving Metric's one real
// semantic rule: a grouped Dataset is a breakdown, not a single metric).
//
// Deliberately minimal per the roadmap's own "do not implement every proposed 007 object
// immediately" instruction: none of the five non-static-content resolvers
// (LowerCollection/LowerRecordSummaryCard/LowerMetric/LowerStatusBadge/LowerActionBar)
// are wired into LowerPage/LowerChildren's actual output yet -- every view_ref node Phase
// 4 produces stays a view_ref; replacing them with real components is Phase 6's "View
// Lowering" migration proper, not this phase's. There is no renderer here at all --
// resolver is the last pipeline stage this phase builds (component type -> contract ->
// validator -> resolver -> renderer, §9's own diagram); turning a validated component
// into HTML/Templ is Phase 6/10. UINode's own Actions/Conditions fields are populated
// only for ActionBar's own Actions -- Conditions remains declared-but-unpopulated, same
// "declared because a later phase needs the shape" posture. DataIR's own flat
// non-deduplicated list, and RelationRef's naming-only (never traversing) a relationship,
// remain Phase 7 (Dependency DAG) and Phase 9 (Query Planner) concerns respectively,
// unchanged from Phase 2.
//
// Phase 6's view_lowering.go is what finally exercises those five unwired resolvers:
// LowerViewToComponent proves List (both table and "cards" render modes),
// Board, Calendar, and Timeline all lower to a Collection (Board's own GroupField and
// a calendar/timeline's own DateField -- the latter a genuine small addition to
// dataset.go's BuildDatasetFromView, its "Date Dimension" per §10's table -- both land
// in Dataset.GroupBy, the same vocabulary Phase 2 already established), with the
// render-mode distinction (table/cards/board/calendar/timeline) carried as a
// Component-level "display" property -- never folded into the Dataset, preserving Phase
// 2's own boundary rule (TestDatasetIdenticalAcrossDisplayModes). LowerDashboardView
// gives UINodeLayout its first real emitter, dispatching each DashboardSection to Metric
// or Collection by the same GroupBy rule LowerMetric's own rejection logic already
// encoded. Per §10's own compatibility rule ("existing View handlers may remain behind
// the lowering layer until equivalence is proven"), NEITHER function is wired into
// LowerPage/LowerChildren's actual output -- both are proven standalone against real
// seeded metadata (Document Approval, Action Lab, Kanban Lab), the same "prove it works,
// don't make it the default yet" posture Phase 5 already used for its own five
// resolvers.
//
// Phase 7's dag.go is the first place this package touches model.Permission at all --
// SecurityScope/ResolveSecurityScope name which Role is asking and that Role's own
// HiddenFields (CAP-P06), and SecurityScope.Apply narrows a Dataset's own Projection to
// what that Role actually sees. dependencyIdentity (identity.go) extends datasetIdentity
// with that security scope as its own explicit dimension -- the concrete answer to
// roadmap principle #5, "security before coalescing" -- and BuildDependencyDAG walks a
// full UI IR tree (Children AND Slots, recursively) collapsing every node's own Dataset
// into a deduplicated DependencyNode: the same Dataset consumed by multiple components
// under the SAME scope becomes one node with multiple Consumers (proven against a real
// naturally-occurring duplicate, seeds/050's own vw_ad_pending appearing both as an
// ordinary child and inside vw_ad_page's own Slots); the same nominal View requested
// under two Roles with different HiddenFields becomes two separate nodes with different
// effective Datasets (proven against seeds/012_permissions_lab.sql's real HR/Staff
// split). §11's own longer identity list (parameters, pagination, interpreter/metadata
// version) stays unbuilt -- no case demands them yet, same discipline every prior phase
// already applied.
//
// Phase 8's planner.go picks up §12's own "Build in this order" list against what
// already existed: stages 2 (dependency deduplication) and 3 (security-aware
// equivalence) turn out to already be delivered, verbatim, by Phase 7's own
// DependencyDAG -- Phase 8 doesn't rebuild them, it consumes them. GroupByMachine is the
// one genuinely new piece (stage 1, execution groups): partitioning a DependencyDAG's
// own Nodes by Source.MachineID, grouping without merging (Phase 7 already proved which
// nodes are truly equal and which must stay separate). ExecutionPlan's own
// NaiveQueryCount/DeduplicatedQueryCount is a deliberately honest stand-in for stage 9
// (cost estimation): a plain count over already-proven facts (Consumers/Nodes), not a
// fabricated weighted cost model -- inventing real numbers here, with no benchmark
// behind them, would be exactly what roadmap principle #8 ("evidence before
// optimization") forbids. Explain() is stage 10 (plan diagnostics). Stages 4-8 (bounded
// concurrency, batching, shared execution, cache opportunities, async fallback) are
// deliberately NOT built -- every one needs a real physical executor, which doesn't
// exist anywhere in this package (unchanged since Phase 1: no database access, no HTTP,
// nothing outside internal/model as input). §12's own literal exit criteria
// ("demonstrates lower or equal physical work for the benchmark scenarios defined
// below") stays open pending Phase 9's physical integration and Phase 12's own benchmark
// harness — this phase only proves the STRUCTURAL claim (DeduplicatedQueryCount <=
// NaiveQueryCount, by construction) against real seeded metadata, not a live benchmark
// result.
//
// Phase 9's physical.go is the first place this package's own claims get checked
// against the REAL physical execution layer (internal/store/record_store.go,
// internal/handler/record_crud.go) -- still without touching either: PhysicalPlan/
// BuildPhysicalPlan is an accurate audit (sort and aggregate pushdown are real today,
// via RecordStore.List's own ORDER BY and CountGroupedBy/SumFieldsGroupedBy's own SQL
// SUM/COUNT/GROUP BY; projection, filter, and pagination pushdown are NOT -- List/Get
// always select the whole `data` column, and record_crud.go's own applyListFilter and
// CAP-R05 pagination both run in Go after every row is already fetched), and
// RuntimeCapabilities audits the four whole-runtime mechanisms (workspace concurrency
// limits, separate analytics capacity, cache, materialization) as uniformly absent.
// BuildWhereClause is the one genuinely new "Query Planner" translation the roadmap's
// own boundary diagram names (CEP -> logical data execution unit -> Query Planner ->
// SQL) -- a parameterized SQL WHERE fragment from Dataset.Filter, but deliberately
// narrow: only equals/not_equals are pushdown-safe (the same semantics internal/
// constraint's own Eval already gives them), because after/before/on_or_after/
// on_or_before need "today"-sentinel resolution and date parsing, and greater_than/
// less_than/etc. need a numeric cast -- getting either wrong silently produces WRONG
// filter results, which is worse than the honest "not pushed down" status quo. Proven
// against a real, naturally-occurring mixed filter: seeds/010_views_lab.sql's own "My
// Overdue Tasks" view combines one safe (equals) and one unsafe (before) filter in the
// SAME real View, and BuildWhereClause correctly fails loud naming the unsafe operator
// rather than silently dropping it or mistranslating it. Nothing here is wired into
// internal/store or internal/handler -- that integration is separate, higher-risk work
// this phase deliberately doesn't attempt, the same "prove it standalone" posture
// Phases 5 and 6 already used for their own unwired resolvers.
//
// Phase 10's viewmodel.go/viewmodel_resolve.go are the first files in this package that
// accept real RECORD data (a plain map[string]any, the exact shape internal/store.
// Record.Data already uses) rather than only metadata -- production code still imports
// neither internal/store nor internal/db; only this phase's own tests reach into the
// real store to fetch or insert rows for an end-to-end proof, the same "prove it
// against real data, stay database-free in production code" split every phase's tests
// already used for metadata. FieldValue{Kind, Display} is §14's own rule made literal
// ("the renderer must not know whether a value originated from a field, expression, or
// relation") -- Display is always a plain string; Kind exists only for diagnostics.
// ResolveFieldValue's three Kind cases are each a deliberate, named scope cut:
// reference/user/group fields resolve to their RAW STORED id, not a dereferenced
// display label (that needs a second store lookup, internal/handler's own
// displayLabel, consistent with Phase 2's RelationRef only ever naming a relationship);
// computed fields resolve only via a declared Expression (CAP-C13's general form),
// through internal/expr.Eval -- reused directly, an explicitly I/O-free evaluator, not
// reimplemented; the older SourceField/Factor sugar returns an error, not a silently
// wrong number. ResolveActionSet resolves an ActionBar's declared Events to real
// {EventID, Label} pairs but applies no CAP-P02 ownership/permission filtering, the
// same posture Phase 3's ValidateBinding already took toward CanRead. ResolveMetricValue
// only formats an already-computed number -- no aggregation happens here. Proven against
// real seeded records (Kanban Lab's own Task rows) and real rows this phase's own tests
// insert via the real store.RecordStore.Create write path where a seed file has the
// right metadata but no data rows yet (seeds/031's Typeahead Lab reference field,
// seeds/041's Expression Lab computed field, seeds/004's Approval Document events).
//
// Phase 11's conformance_test.go/conformance_seed_test.go are this package's own proof
// system for §15's twelve CMP-* composition-conformance classes -- deliberately NOT
// app/conformance/ (the real HTTP-driven CAP-xx suite this package still isn't wired
// into) and NOT capability-registry.md (the real capability-admission ledger, governed
// by capability-lifecycle.md's own A1-A5 test for genuinely new capabilities, not proof
// classes about this substrate). Eight of the twelve were already proven by name in
// Phases 2-10 and are only consolidated/labeled here; four needed a real, if narrow,
// code change: ui.go's LowerChildren now actually recurses into a resolved view_ref's
// own further Children when that resolved View is itself page-type (previously
// deliberately "one level only, no recursion" since Phase 4), with cycle detection
// (CMP-02) and a depth bound (CMP-08, maxCompositionDepth = 5) making that recursion
// safe, and fails loud instead of silently skipping an unresolved view reference
// (CMP-03) or a Children entry naming neither view nor content (CMP-04) -- this
// package's CAP-X05 "Unknown = explicit" posture applied to Children for the first
// time. Honestly stated: today's real metadata/validate.go allow-lists
// (PageEmbeddableViewTypes/EmbeddableChildViewTypes) never let a page embed another
// page, so no real seeded metadata can actually cycle or recurse deep enough to hit the
// bound yet -- CMP-02/08 are proven with builder-only fixtures that bypass real
// metadata validation, defense in depth for the substrate itself, not a fix to a
// reachable bug. CMP-12 (trial-case cross-substrate reuse) is an explicit t.Skip, not a
// silent omission -- it needs Case 19 (Project Management) seeded, Phase 13's own job,
// the same gap flagged since Phase 1.
//
// Phase 12's benchmark.go is deliberately NOT a real performance benchmark -- §16 asks
// for p50/p95/p99 latency, CPU, memory, DB pool utilization, render time, cache hit
// ratio, and rows scanned/returned across nine named scenarios, but every one of those
// needs live query execution (Phase 9's own audit: internal/store is still untouched),
// live rendering (Phase 10 built View Models with zero Templ wiring), or live process/
// pool metrics that don't exist in this package by design. Fabricating any of them
// would be exactly what roadmap principle #8 forbids. MeasureComposition instead
// reports the four §16 metrics that ARE honestly computable from this package's own
// data (logical nodes, DAG nodes, naive-vs-deduplicated physical query count, execution
// width), run across all nine required scenarios: three synthetic shapes matching the
// exact cardinalities named (1->1, 10->10, 10->3), one honestly-reported non-ideal
// finding (a Dataset with two different Projections produces two DAG nodes today, not
// one -- projection-superset sharing is Phase 8's own unbuilt stage 6), two real-seed
// cases reused from Phases 3/7/11 (security-scope separation; the vw_ad_page/
// vw_ad_pending embedded-dependency pair), one Case-19 stand-in (a synthetic nested
// chain, same honesty as every prior phase), and two explicit t.Skips naming exactly
// which missing infrastructure blocks them (mixed OLTP/analytics pooling; 100
// concurrent workspaces' cache/isolation). One real, previously-unnoticed gap surfaced
// while grounding the "Approval detail + stepper" scenario: LowerPage only ever calls
// LowerChildren for a page-type View -- a detail-type View's own Config.Children
// (seeds/042_inline_view_composition.sql's vw_as_detail, CAP-V20 Tier 2's
// EmbeddableChildViewTypes allow-list) is never lowered at all. Named here, not fixed --
// out of scope for a benchmark phase. §16's own Acceptance rule ("no planner
// optimization accepted merely because query count falls... must preserve semantics,
// security, and relevant latency/resource SLOs") is only half-checkable: the semantics/
// security half is already proven (Phases 3/7/11); the latency/resource-SLO half needs
// live execution this phase doesn't have, so every reduction reported here is a
// structural fact, not an accepted optimization.
//
// Phase 13's trial_test.go/trial_seed_test.go finally close the Case-19 gap named in
// every prior phase's own comments: app/seeds/052_project_management.sql seeds Board/
// List/Card/Checklist Item at composable-substrate-proof scope (real CAP-F13
// references, CAP-V06 child-lists, CAP-F16 ChildLines) -- NOT a faithful Trello-style
// board. Investigating why Case 19 was never seeded surfaced a real, already-documented
// capability gap, not just an oversight: capability-registry.md's own CAP-V14 Tier 2
// row states the real board view is "a deliberately narrower cut than Case 19's own
// literal declaration" (fixed value_list lanes only, never a dynamic reference) --
// building a faithful Case 19 needs new capability work in internal/handler/internal/
// model, a capability-lifecycle.md-governed admission decision explicitly NOT made
// here (confirmed with the user directly). assertFullSubstrateCoverage
// (trial_test.go) is the actual proof of §17's own success criterion ("the two
// applications share the same X"): ONE shared Go function, no per-case branch anywhere
// in it, called once for Document Approval and once for Project Management, exercising
// all seven named mechanisms (Page/Layout, Dataset/Projection, Component contract,
// Context/Binding, permission-aware dependency planning, CEP, renderer boundary) in
// order. No changes anywhere else in this package -- every mechanism Phase 13 needed
// already existed from Phases 2-12; this phase only proves two real cases share it.
//
// See 007-composable-runtime-architecture.md for the normative target and
// composable-runtime-roadmap.md §5/§6/§7/§8/§9/§10/§11/§12/§13/§14/§15/§16/§17 for
// these phases' build/proof/exit-criteria contracts.
package composable
