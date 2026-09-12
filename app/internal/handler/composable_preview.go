package handler

import (
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"menata.id/app/internal/composable"
	"menata.id/app/internal/model"
	"menata.id/app/internal/store"
	"menata.id/app/internal/ui"
)

// ComposablePreview (composable-runtime-roadmap.md §17a, Live Wiring Pilot)
// is a read-only, additive route proving internal/composable can drive a
// real HTTP response over real seeded Postgres data end to end --
// List/Board/Detail (record_crud.go, views.go) are completely untouched by
// this handler; nothing about their behavior changes. The card grid itself
// only renders when the machine has a default List View in "cards"
// display mode; a machine without one still gets a 200 with an empty
// grid, since 17c generalized this route to reach ANY machine's own
// Dependency DAG/Execution Planner diagnostic (explainComposablePlan,
// 17b), not just the one machine that happens to have a cards list.
//
// composable-runtime-roadmap.md 17e: the card grid now reuses the exact
// same RecordSummaryCard/StatusBadge/Avatar rendering the real CAP-V02
// Tier 2 cards feature uses (internal/ui/list.templ) -- avatar initials,
// a joined multi-column subtitle, and a status badge (CardBadgeField's
// own column) all included, proven equivalent to the real feature by
// T268/T269 (conformance/tests/241_composable_pilot.sh). Named remaining
// gap, not an oversight: SlaBadge is not reproduced -- no column on this
// pilot's own real target (vw_ad_all) is SLA-eligible, so nothing forces
// it yet.
//
// composable-runtime-roadmap.md 17f: sort/search/pagination now reuse
// List's own real behavior (sortFieldFor/searchListRecords/
// paginateListRecords, extracted from record_crud.go's List into shared
// functions rather than reimplemented here) -- previously this route
// always passed "","" to RecordStore.List (correct for vw_ad_all's own
// DefaultSort only by coincidence) and never searched or paginated at
// all. Archive/Restore/Move-up/down and the Archived-toggle view stay
// out of scope: cards mode never renders the first three even on the
// real page (list.templ's own `if opts.Cards {...} else {...}` puts them
// only in the table branch), and no role on this Machine has CanDelete
// today, so there is no real target to prove the toggle against.
func (h *Handler) ComposablePreview(w http.ResponseWriter, r *http.Request) {
	machineID := chi.URLParam(r, "machineID")
	machine, ok := h.interp.Get().GetMachine(machineID)
	if !ok {
		http.NotFound(w, r)
		return
	}
	workspaceID, applicationID := h.interp.Get().ScopeFor(machineID)
	if workspaceID != h.workspace(r) {
		// CAP-X06: a Machine from another Workspace 404s exactly like one
		// that doesn't exist at all -- same guard every other per-machine
		// route already applies.
		http.NotFound(w, r)
		return
	}
	role := h.roleForApp(r, applicationID)
	if !h.guard.CanRead(machine, role) {
		h.logPermissionDenied(r.Context(), "read", machineID, "", role, h.identity(r))
		http.Error(w, "not permitted", http.StatusForbidden)
		return
	}

	viewName := machine.Name
	var summaries []composable.RecordSummary
	var badges []composable.StatusValue
	var opts ui.ListViewOptions
	var lanes []composable.BoardLane
	var laneNames map[string]string
	dataStart := time.Now()
	if view := h.interp.Get().DefaultListView(machineID); view != nil && view.Config.Display == "cards" {
		viewName = view.Name
		var searchQuery string
		var pageNum, totalPages int
		var ok bool
		summaries, badges, searchQuery, pageNum, totalPages, ok = h.resolveComposableCardSummaries(w, r, machine, view)
		if !ok {
			return
		}
		opts = ui.ListViewOptions{SearchQuery: searchQuery, Page: pageNum, TotalPages: totalPages, Cards: true}
	} else if boardView := h.interp.Get().BoardView(machineID); boardView != nil {
		// composable-runtime-roadmap.md 17h: the first Project Management
		// live footprint -- a dynamic-lane Board (CAP-V14, reference-typed
		// group_field) rendered read-only via internal/composable, when
		// this machine has no cards-display List (the 17a-17g branch
		// above never fires for a Board-only machine like mch_pm_card).
		viewName = boardView.Name
		var ok bool
		lanes, laneNames, ok = h.resolveComposableBoardLanes(w, r, machine, boardView)
		if !ok {
			return
		}
	}
	dataDuration := time.Since(dataStart)

	planStart := time.Now()
	planExplain, metrics := h.explainComposablePlan(r, applicationID, machine, role[0])
	plannerDuration := time.Since(planStart)

	a := h.auth(r)
	page := ui.ComposablePreview(h.workspaceName(r), h.workspaceSlug(r), a.User.Name, a.CSRFToken, h.isWorkspaceAdmin(r), machine, viewName, h.unreadCount(r.Context(), a), h.subNavFor(r, machine), summaries, badges, opts, lanes, laneNames, planExplain)
	renderStart := time.Now()
	renderErr := page.Render(r.Context(), w)
	renderDuration := time.Since(renderStart)
	if renderErr != nil {
		slog.Error("render composable preview", "error", renderErr)
	}

	logComposableBenchmark(r, machine.ID, rowsReturned(summaries, lanes), dataDuration, plannerDuration, renderDuration, metrics)
}

// rowsReturned (composable-runtime-roadmap.md 17j) counts the actual rows
// in this response -- summaries for a cards preview, every lane's own
// Cards summed for a board preview (the two are mutually exclusive per
// request, same as the templ's own rendering choice).
func rowsReturned(summaries []composable.RecordSummary, lanes []composable.BoardLane) int {
	n := len(summaries)
	for _, lane := range lanes {
		n += len(lane.Cards)
	}
	return n
}

// logComposableBenchmark (composable-runtime-roadmap.md 17j) logs one
// structured line per request combining composable.MeasureComposition's
// own structural facts (Phase 12, previously Go-test-only, never called
// from a live handler before this) with real measured timing --
// joinable to slogAccessLog's own line by the same correlation_id both
// already use. data_time_ms deliberately bundles the DB fetch with
// in-memory filter/search/paginate/resolve rather than isolating a
// precise db_time_ms: internal/store's own queries return every row for
// a Machine and all filtering happens in Go afterward (17f/17g's own
// finding), so there is no SQL-level boundary honest enough to isolate
// further yet.
func logComposableBenchmark(r *http.Request, machineID string, rows int, dataDuration, plannerDuration, renderDuration time.Duration, metrics composable.BenchmarkMetrics) {
	slog.Info("composable_benchmark",
		"correlation_id", middleware.GetReqID(r.Context()),
		"machine_id", machineID,
		"logical_nodes", metrics.LogicalNodes,
		"dag_nodes", metrics.DAGNodes,
		"naive_query_count", metrics.NaiveQueryCount,
		"dedup_query_count", metrics.DeduplicatedQueryCount,
		"execution_width", metrics.ExecutionWidth,
		"rows_returned", rows,
		"data_time_ms", dataDuration.Milliseconds(),
		"planner_time_ms", plannerDuration.Milliseconds(),
		"render_time_ms", renderDuration.Milliseconds(),
	)
}

// resolveComposableCardSummaries resolves view's own cards rendering --
// the RecordSummary/StatusValue slices, and the raw search/pagination
// state each caller merges into its OWN ui.ListViewOptions (List, 17g,
// also needs ManualOrder/CanDelete fields this function has no business
// deciding). Extracted purely to keep its callers within Gate 3's
// cyclomatic-complexity ratchet (composable-runtime-roadmap.md 17e/17f;
// the same lesson 17c's own test refactor already applied). On any
// failure, writes a generic (CWE-209-safe, Gate 1) 500 itself and
// returns ok=false -- the caller's only job on that path is to return
// immediately.
//
// composable-runtime-roadmap.md 17g: now also applies h.applyListFilter
// (CAP-V05/V09's declarative list Filter) -- unobservable on vw_ad_all
// (no filter configured), but List's own real pipeline always applies it
// before search/pagination, and this function is List's own cutover path
// for cards mode now, so leaving it out would be a real, if currently
// invisible, behavior gap for a future filtered cards View.
// listCardsViaComposable is List's own cutover branch (composable-
// runtime-roadmap.md 17g) for a live, cards-display View -- reuses
// resolveComposableCardSummaries and explainComposablePlan (below)
// exactly as /composable-preview does, merging their output into List's
// own real ui.ListViewOptions (which needs ManualOrder/CanDelete fields
// resolveComposableCardSummaries has no business deciding). Lives here,
// not record_crud.go (its original 17g home), since 17j's own timing
// instrumentation pushed that already-oversized file past Gate 2's LOC
// ratchet -- this file is the natural, smaller home for anything in the
// composable-preview family, record_crud.go's own `List` just calls it.
func (h *Handler) listCardsViaComposable(w http.ResponseWriter, r *http.Request, machine *model.Machine, applicationID string, role []string, view *model.View) {
	dataStart := time.Now()
	summaries, badges, searchQuery, pageNum, totalPages, ok := h.resolveComposableCardSummaries(w, r, machine, view)
	if !ok {
		return
	}
	dataDuration := time.Since(dataStart)
	opts := ui.ListViewOptions{
		SearchQuery: searchQuery,
		ManualOrder: view.Config.ManualOrder,
		CanDelete:   h.guard.CanDelete(machine, role),
		Page:        pageNum,
		TotalPages:  totalPages,
		Cards:       true,
	}
	planStart := time.Now()
	planExplain, metrics := h.explainComposablePlan(r, applicationID, machine, role[0])
	plannerDuration := time.Since(planStart)
	a := h.auth(r)
	page := ui.List(h.workspaceName(r), h.workspaceSlug(r), a.User.Name, a.CSRFToken, h.isWorkspaceAdmin(r), machine, nil, nil, h.interp.Get().PermittedEvents(machine.ID, role), h.unreadCount(r.Context(), a), opts, h.subNavFor(r, machine), h.viewNavFor(h.workspaceSlug(r), machine.ID, model.ViewTypeList), summaries, badges, planExplain)
	renderStart := time.Now()
	renderErr := page.Render(r.Context(), w)
	renderDuration := time.Since(renderStart)
	if renderErr != nil {
		slog.Error("render list", "error", renderErr)
	}
	logComposableBenchmark(r, machine.ID, len(summaries), dataDuration, plannerDuration, renderDuration, metrics)
}

func (h *Handler) resolveComposableCardSummaries(w http.ResponseWriter, r *http.Request, machine *model.Machine, view *model.View) (summaries []composable.RecordSummary, badges []composable.StatusValue, searchQuery string, page, totalPages int, ok bool) {
	ds, err := composable.BuildDatasetFromView(machine, view)
	if err != nil {
		http.Error(w, "failed to build dataset", http.StatusInternalServerError)
		return nil, nil, "", 0, 0, false
	}
	rowNode, err := composable.LowerCardRowComponent(machine, view, ds)
	if err != nil {
		http.Error(w, "failed to lower card component", http.StatusInternalServerError)
		return nil, nil, "", 0, 0, false
	}

	var badgeNode *composable.UINode
	if badgeField := composable.CardBadgeField(machine, view.Config.Columns); badgeField != "" {
		node, err := composable.LowerStatusBadge(machine, badgeField)
		if err != nil {
			http.Error(w, "failed to lower status badge", http.StatusInternalServerError)
			return nil, nil, "", 0, 0, false
		}
		badgeNode = &node
	}

	sortField, sortDir := sortFieldFor(view)
	records, err := h.records.List(r.Context(), machine.ID, sortField, sortDir)
	if err != nil {
		http.Error(w, "failed to load records", http.StatusInternalServerError)
		return nil, nil, "", 0, 0, false
	}
	records = h.applyListFilter(r, view, records)

	searchQuery = strings.TrimSpace(r.URL.Query().Get("q"))
	records = searchListRecords(searchQuery, view.Config.Columns, records)

	records, page, totalPages = paginateListRecords(records, r.URL.Query().Get("page"))

	summaries = make([]composable.RecordSummary, 0, len(records))
	badges = make([]composable.StatusValue, 0, len(records))
	for _, rec := range records {
		summary, err := composable.ResolveRecordSummary(machine, rowNode, rec.ID, rec.Data)
		if err != nil {
			http.Error(w, "failed to resolve record summary", http.StatusInternalServerError)
			return nil, nil, "", 0, 0, false
		}
		summaries = append(summaries, summary)

		var badge composable.StatusValue
		if badgeNode != nil {
			badge, err = composable.ResolveStatusValue(machine, *badgeNode, rec.Data)
			if err != nil {
				http.Error(w, "failed to resolve status badge", http.StatusInternalServerError)
				return nil, nil, "", 0, 0, false
			}
		}
		badges = append(badges, badge)
	}
	return summaries, badges, searchQuery, page, totalPages, true
}

// boardRelationTarget (composable-runtime-roadmap.md 17h) returns the
// Machine id a Board Dataset's own GroupBy field points at, via its
// already-discovered Relations (BuildDatasetFromView's own Board branch,
// internal/composable/dataset.go, already includes GroupField when
// discovering Relations -- no Dataset-layer change needed here) -- ""
// when GroupBy names no reference field (the fixed-value-list lane case,
// CAP-V14 Tier 2, not handled by this pilot).
func boardRelationTarget(ds composable.Dataset) string {
	if len(ds.GroupBy) != 1 {
		return ""
	}
	for _, rel := range ds.Relations {
		if rel.ViaField == ds.GroupBy[0] {
			return rel.TargetMachineID
		}
	}
	return ""
}

func toRecordRefs(records []*store.Record) []composable.RecordRef {
	refs := make([]composable.RecordRef, len(records))
	for i, rec := range records {
		refs[i] = composable.RecordRef{ID: rec.ID, Data: rec.Data}
	}
	return refs
}

// resolveComposableBoardLanes (composable-runtime-roadmap.md 17h)
// resolves view's own dynamic-lane Board rendering -- the BoardLane
// slice, and each lane's own real display label. Label resolution
// reuses displayLabel exactly as the real Board handler (views.go)
// already does -- internal/composable can't call it directly (Gate 5),
// so this is the one place that label logic runs for the preview, not a
// second copy. On any failure, writes a generic (CWE-209-safe, Gate 1)
// 500 itself and returns ok=false.
func (h *Handler) resolveComposableBoardLanes(w http.ResponseWriter, r *http.Request, machine *model.Machine, view *model.View) (lanes []composable.BoardLane, laneNames map[string]string, ok bool) {
	ds, err := composable.BuildDatasetFromView(machine, view)
	if err != nil {
		http.Error(w, "failed to build dataset", http.StatusInternalServerError)
		return nil, nil, false
	}
	node, err := composable.LowerViewToComponent(machine, view)
	if err != nil {
		http.Error(w, "failed to lower board component", http.StatusInternalServerError)
		return nil, nil, false
	}
	targetMachineID := boardRelationTarget(ds)
	if targetMachineID == "" {
		http.Error(w, "board view's group field is not a reference -- fixed-value-list lanes are not supported by this preview", http.StatusInternalServerError)
		return nil, nil, false
	}
	laneMachine, found := h.interp.Get().GetMachine(targetMachineID)
	if !found {
		http.Error(w, "board view's target machine not found", http.StatusInternalServerError)
		return nil, nil, false
	}

	laneRecs, err := h.records.List(r.Context(), targetMachineID, "", "")
	if err != nil {
		http.Error(w, "failed to load lane records", http.StatusInternalServerError)
		return nil, nil, false
	}
	cardRecs, err := h.records.List(r.Context(), machine.ID, "", "")
	if err != nil {
		http.Error(w, "failed to load card records", http.StatusInternalServerError)
		return nil, nil, false
	}

	lanes, err = composable.ResolveBoardLanes(machine, node, toRecordRefs(cardRecs), toRecordRefs(laneRecs))
	if err != nil {
		http.Error(w, "failed to resolve board lanes", http.StatusInternalServerError)
		return nil, nil, false
	}

	laneNames = make(map[string]string, len(laneRecs))
	for _, rec := range laneRecs {
		laneNames[rec.ID] = displayLabel(laneMachine, rec.ID, rec.Data)
	}
	return lanes, laneNames, true
}

// explainComposablePlan (composable-runtime-roadmap.md 17b) lowers
// machine's own FULL page (LowerPage -- every one of its Views, not just
// the single cards list view this route renders) and runs the result
// through BuildDependencyDAG + BuildExecutionPlan, purely as a
// side-channel diagnostic: the first live HTTP request where the
// Dependency DAG/Execution Planner (Phase 7/8) actually runs against
// real request-scoped metadata, proving the planner boundary exists on
// the live runtime path (composable-apps-trial.md §15.3) -- it does not
// yet drive physical execution (CR-05/Phase 9's own open gap, unaffected
// by this), only made inspectable per Principle #9 ("inference is
// inspectable"). Never fails the request: any error here is logged and
// swallowed, since a diagnostic must not break the page it's attached to.
//
// role is the caller's FIRST held role only (roleForApp's own []string,
// guaranteed non-empty here since CanRead already passed) -- a known
// simplification, not a security decision: guard.CanRead's own real
// multi-role union semantics remain the actual enforcement, completely
// unaffected by this diagnostic-only scope choice.
//
// composable-runtime-roadmap.md 17j: also computes composable.
// MeasureComposition (Phase 12) over the same lowered tree -- the first
// live call site for that function, previously Go-test-only
// (benchmark_seed_test.go). Returned alongside explain so the caller can
// fold it into logComposableBenchmark; on any failure both return values
// are zero-valued, same "never fails the request" posture as before.
func (h *Handler) explainComposablePlan(r *http.Request, applicationID string, machine *model.Machine, role string) (string, composable.BenchmarkMetrics) {
	app, ok := h.interp.Get().GetApplication(applicationID)
	if !ok {
		return "", composable.BenchmarkMetrics{}
	}
	viewIdx := composable.IndexViews(app)
	machineIdx := composable.IndexMachines(app)
	datasetIdx := composable.IndexDatasets(app)
	pageNode, err := composable.LowerPage(machine, viewIdx, machineIdx, datasetIdx)
	if err != nil {
		slog.Warn("composable plan: lower page", "correlation_id", middleware.GetReqID(r.Context()), "machine_id", machine.ID, "error", err)
		return "", composable.BenchmarkMetrics{}
	}
	scope := composable.ResolveSecurityScope(machine, role)
	dag := composable.BuildDependencyDAG([]composable.UINode{pageNode}, scope)
	explain := composable.BuildExecutionPlan(dag).Explain()
	metrics := composable.MeasureComposition([]composable.UINode{pageNode}, scope)
	slog.Info("composable_plan", "correlation_id", middleware.GetReqID(r.Context()), "machine_id", machine.ID, "explain", explain)
	return explain, metrics
}
