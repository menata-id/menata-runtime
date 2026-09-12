package handler

import (
	"log/slog"
	"net/http"

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
// this handler; nothing about their behavior changes. Scoped deliberately
// narrow: only a List View in "cards" display mode, rendered via
// LowerCardRowComponent/ResolveRecordSummary's own title+subtitle
// RecordSummaryCard contract -- no status badge like the real CAP-V02
// Tier 2 cards feature shows, a named limitation (extending that contract
// is real capability work outside this pilot's scope), not an oversight.
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

	view := h.interp.Get().DefaultListView(machineID)
	if view == nil || view.Config.Display != "cards" {
		http.Error(w, "composable preview only supports a list view with display: cards", http.StatusBadRequest)
		return
	}

	ds, err := composable.BuildDatasetFromView(machine, view)
	if err != nil {
		http.Error(w, "failed to build dataset", http.StatusInternalServerError)
		return
	}
	rowNode, err := composable.LowerCardRowComponent(view, ds)
	if err != nil {
		http.Error(w, "failed to lower card component", http.StatusInternalServerError)
		return
	}

	records, err := h.records.List(r.Context(), machineID, "", "")
	if err != nil {
		http.Error(w, "failed to load records", http.StatusInternalServerError)
		return
	}

	summaries := make([]composable.RecordSummary, 0, len(records))
	for _, rec := range records {
		summary, err := composable.ResolveRecordSummary(machine, rowNode, rec.ID, rec.Data)
		if err != nil {
			http.Error(w, "failed to resolve record summary", http.StatusInternalServerError)
			return
		}
		summaries = append(summaries, summary)
	}

	planExplain := h.explainComposablePlan(r, applicationID, machine, role[0])

	a := h.auth(r)
	page := ui.ComposablePreview(h.workspaceName(r), h.workspaceSlug(r), a.User.Name, a.CSRFToken, h.isWorkspaceAdmin(r), machine, view.Name, h.unreadCount(r.Context(), a), h.subNavFor(r, machine), summaries, planExplain)
	if err := page.Render(r.Context(), w); err != nil {
		slog.Error("render composable preview", "error", err)
	}
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
