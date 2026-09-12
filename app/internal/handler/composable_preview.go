package handler

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"menata.id/app/internal/composable"
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

	a := h.auth(r)
	page := ui.ComposablePreview(h.workspaceName(r), h.workspaceSlug(r), a.User.Name, a.CSRFToken, h.isWorkspaceAdmin(r), machine, view.Name, h.unreadCount(r.Context(), a), h.subNavFor(r, machine), summaries)
	if err := page.Render(r.Context(), w); err != nil {
		slog.Error("render composable preview", "error", err)
	}
}
