package handler

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"menata.id/app/internal/composable"
	"menata.id/app/internal/model"
	"menata.id/app/internal/ui"
)

// RecordComposablePreview (composable-runtime-roadmap.md 17m, Detail-Page
// Composition Pilot) is a read-only, additive route proving
// internal/composable can drive a real single-record field list end to
// end -- the same "Live Wiring Pilot" methodology 17a already used for
// List, applied here to Detail for the first time. record_crud.go's real
// Detail handler is completely untouched; nothing about its behavior
// changes.
//
// Design decision (see internal/composable/dataset.go's own new
// ViewTypeDetail case, and component_adapt.go's LowerCollection): a
// Detail view's field list needs no new ComponentType -- it's exactly one
// Collection component's own CollectionItem, resolved for one record
// instead of many.
//
// Named, proven boundary, not overclaimed: ResolveFieldValue's own
// documented scope (raw stored id for reference/user/group, no label
// dereferencing; no money/computed-via-sugar formatting, no SLA urgency)
// means this preview will NOT byte-match the real Detail page for every
// field type -- conformance/tests/242_composable_detail_pilot.sh only
// asserts equivalence on the fields where it actually holds.
//
// Narrow by construction, not silently guessed: only reachable when the
// Machine declares a real `detail`-type View (DetailView) -- unlike the
// real Detail handler, which renders every Machine's own Fields even with
// none declared. No case in this trial forces the "no Detail View at all"
// path yet (every real seeded Machine used here has one); a 404 there is
// honest, not a workaround.
func (h *Handler) RecordComposablePreview(w http.ResponseWriter, r *http.Request) {
	machineID := chi.URLParam(r, "machineID")
	recordID := chi.URLParam(r, "recordID")
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
	rec, err := h.records.Get(r.Context(), recordID)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	view := h.interp.Get().DetailView(machineID)
	if view == nil {
		http.NotFound(w, r)
		return
	}

	ds, err := composable.BuildDatasetFromView(machine, view)
	if err != nil {
		slog.Warn("record composable preview: build dataset", "machine_id", machineID, "error", err)
		http.Error(w, "unable to build preview", http.StatusInternalServerError)
		return
	}
	node, err := composable.LowerCollection(ds)
	if err != nil {
		slog.Warn("record composable preview: lower collection", "machine_id", machineID, "error", err)
		http.Error(w, "unable to build preview", http.StatusInternalServerError)
		return
	}
	item, err := composable.ResolveCollectionItem(machine, node, rec.ID, rec.Data)
	if err != nil {
		slog.Warn("record composable preview: resolve collection item", "machine_id", machineID, "error", err)
		http.Error(w, "unable to build preview", http.StatusInternalServerError)
		return
	}

	fields := buildPreviewDetailFields(fieldIndex(machine), ds.Projection.Fields, item.Cells)

	a := h.auth(r)
	page := ui.RecordComposablePreview(h.workspaceName(r), h.workspaceSlug(r), a.User.Name, a.CSRFToken, h.isWorkspaceAdmin(r), machine, view.Name, fields, h.unreadCount(r.Context(), a), h.subNavFor(r, machine))
	if err := page.Render(r.Context(), w); err != nil {
		slog.Error("render record composable preview", "error", err)
	}
}

// buildPreviewDetailFields pairs projectionFields (composable.Dataset's
// own Projection.Fields, in order) with fieldByID's own display Names and
// cells' resolved values -- the ColumnDef-style pairing
// renderPageListChild already does for table columns. Split out of
// RecordComposablePreview itself (Gate 3: keeps its own complexity
// small).
func buildPreviewDetailFields(fieldByID map[string]*model.Field, projectionFields []string, cells []composable.FieldValue) []ui.PreviewDetailField {
	fields := make([]ui.PreviewDetailField, len(projectionFields))
	for i, fieldID := range projectionFields {
		name := fieldID
		if f, ok := fieldByID[fieldID]; ok {
			name = f.Name
		}
		fields[i] = ui.PreviewDetailField{Name: name, Value: cells[i]}
	}
	return fields
}
