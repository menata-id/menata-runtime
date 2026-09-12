package handler

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"menata.id/app/internal/composable"
	"menata.id/app/internal/model"
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
	}

	planExplain := h.explainComposablePlan(r, applicationID, machine, role[0])

	a := h.auth(r)
	page := ui.ComposablePreview(h.workspaceName(r), h.workspaceSlug(r), a.User.Name, a.CSRFToken, h.isWorkspaceAdmin(r), machine, viewName, h.unreadCount(r.Context(), a), h.subNavFor(r, machine), summaries, badges, opts, planExplain)
	if err := page.Render(r.Context(), w); err != nil {
		slog.Error("render composable preview", "error", err)
	}
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
func (h *Handler) explainComposablePlan(r *http.Request, applicationID string, machine *model.Machine, role string) string {
	app, ok := h.interp.Get().GetApplication(applicationID)
	if !ok {
		return ""
	}
	viewIdx := composable.IndexViews(app)
	machineIdx := composable.IndexMachines(app)
	pageNode, err := composable.LowerPage(machine, viewIdx, machineIdx)
	if err != nil {
		slog.Warn("composable plan: lower page", "correlation_id", middleware.GetReqID(r.Context()), "machine_id", machine.ID, "error", err)
		return ""
	}
	scope := composable.ResolveSecurityScope(machine, role)
	dag := composable.BuildDependencyDAG([]composable.UINode{pageNode}, scope)
	explain := composable.BuildExecutionPlan(dag).Explain()
	slog.Info("composable_plan", "correlation_id", middleware.GetReqID(r.Context()), "machine_id", machine.ID, "explain", explain)
	return explain
}
