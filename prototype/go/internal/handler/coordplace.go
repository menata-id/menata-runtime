package handler

import (
	"fmt"
	"log/slog"
	"net/http"
	"path/filepath"
	"slices"
	"strconv"

	"github.com/go-chi/chi/v5"
	pdfapi "github.com/pdfcpu/pdfcpu/pkg/api"

	"menata.id/runtime/internal/model"
	"menata.id/runtime/internal/store"
	"menata.id/runtime/internal/ui"
)

// CoordPlace (CAP-V21) renders a "coord_placement" View: a preview of
// another record's own file (found via Config.CoordPlacement.ReferenceField
// -> PreviewField) with a pin at this record's own currently-declared
// (page, x%, y%). Read-only (no drag affordance, no JS wiring) whenever the
// acting role can't edit this record -- same component, no second route,
// covering CAP-V21's own "read-only single-pin mode... preview-only use".
func (h *Handler) CoordPlace(w http.ResponseWriter, r *http.Request) {
	machineID := chi.URLParam(r, "machineID")
	recordID := chi.URLParam(r, "recordID")
	machine, ok := h.interp.Get().GetMachine(machineID)
	if !ok {
		http.NotFound(w, r)
		return
	}
	workspaceID, applicationID := h.interp.Get().ScopeFor(machineID)
	if workspaceID != h.workspace(r) {
		http.NotFound(w, r)
		return
	}
	role := h.roleForApp(r, applicationID)
	if !h.guard.CanRead(machine, role) {
		h.logPermissionDenied(r.Context(), "read", machineID, "", role, h.identity(r))
		http.Error(w, "not permitted", http.StatusForbidden)
		return
	}
	view := h.interp.Get().CoordPlacementView(machineID)
	if view == nil || view.Config.CoordPlacement == nil {
		http.NotFound(w, r)
		return
	}
	cp := view.Config.CoordPlacement
	rec, err := h.records.Get(r.Context(), recordID)
	if err != nil || rec.MachineID != machineID {
		http.NotFound(w, r)
		return
	}
	previewKey, pageCount, isPDF, ok := h.coordPlacePreview(r, cp, rec)
	if !ok {
		http.NotFound(w, r)
		return
	}

	page := int(toFloat(rec.Data[cp.PageField]))
	if q := r.URL.Query().Get("page"); q != "" {
		if p, err2 := parsePagePositive(q); err2 == nil {
			page = p
		}
	}
	if page < 1 {
		page = 1
	}
	if page > pageCount {
		page = pageCount
	}
	x, y := toFloat(rec.Data[cp.XField]), toFloat(rec.Data[cp.YField])
	if x == 0 && y == 0 {
		x, y = 50, 50 // an unset pin starts centered, not pinned to the corner
	}

	editable := h.guard.CanEdit(machine, role) && coordPlaceOwnerOK(machine, role, h.identityID(r), rec.Data)
	a := h.auth(r)
	pageComp := ui.CoordPlace(h.workspaceName(r), a.User.Name, a.CSRFToken, h.isWorkspaceAdmin(r), machine, rec, view.Name, previewKey, isPDF, page, pageCount, x, y, editable, h.unreadCount(r.Context(), a), h.subNavFor(r, machine))
	if err := pageComp.Render(r.Context(), w); err != nil {
		slog.Error("render coord place", "error", err)
	}
}

// SetCoordPlace (CAP-V21) writes a dropped pin's (page, x%, y%) back to this
// record. Same "trusted same-record field write triggered by a UI action,
// not a business Event" posture BoardMove already takes -- no Executor.
// Persist, no constraint re-validation. Unlike BoardMove, this additionally
// requires per-record ownership when the acting role's own Permission row
// declares one (coordPlaceOwnerOK) -- board's shared lanes have no owner
// concept, but a signature-placement-shaped use very much does: without
// this, any Approver could reposition a DIFFERENT approver's own pin.
func (h *Handler) SetCoordPlace(w http.ResponseWriter, r *http.Request) {
	machineID := chi.URLParam(r, "machineID")
	recordID := chi.URLParam(r, "recordID")
	machine, ok := h.interp.Get().GetMachine(machineID)
	if !ok {
		http.NotFound(w, r)
		return
	}
	workspaceID, applicationID := h.interp.Get().ScopeFor(machineID)
	if workspaceID != h.workspace(r) {
		http.NotFound(w, r)
		return
	}
	role := h.roleForApp(r, applicationID)
	rec, err := h.records.Get(r.Context(), recordID)
	if err != nil || rec.MachineID != machineID {
		http.NotFound(w, r)
		return
	}
	if !h.guard.CanEdit(machine, role) || !coordPlaceOwnerOK(machine, role, h.identityID(r), rec.Data) {
		h.logPermissionDenied(r.Context(), "edit", machineID, recordID, role, h.identity(r))
		http.Error(w, "not permitted", http.StatusForbidden)
		return
	}
	view := h.interp.Get().CoordPlacementView(machineID)
	if view == nil || view.Config.CoordPlacement == nil {
		http.NotFound(w, r)
		return
	}
	cp := view.Config.CoordPlacement
	_, pageCount, _, ok := h.coordPlacePreview(r, cp, rec)
	if !ok {
		http.NotFound(w, r)
		return
	}

	page, err := parsePagePositive(r.FormValue("page"))
	if err != nil {
		http.Error(w, "invalid page", http.StatusBadRequest)
		return
	}
	if page > pageCount {
		page = pageCount
	}
	x, y := clamp01to100(toFloat(r.FormValue("x"))), clamp01to100(toFloat(r.FormValue("y")))

	newData := make(map[string]any, len(rec.Data))
	for k, v := range rec.Data {
		newData[k] = v
	}
	newData[cp.PageField] = page
	newData[cp.XField] = x
	newData[cp.YField] = y
	if err := h.records.Update(r.Context(), recordID, newData); err != nil {
		http.Error(w, "failed to save position", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/"+machineID+"/"+recordID+"/place", http.StatusSeeOther)
}

// coordPlacePreview resolves the referenced record's own preview file,
// returning its storage key, page count (1 for a non-PDF image -- CAP-V21
// is not signature/PDF-specific, "equally usable to mark a defect location
// on an equipment photo" per its own registry row), whether it's a PDF, and
// whether a usable preview was found at all.
func (h *Handler) coordPlacePreview(r *http.Request, cp *model.CoordPlacementConfig, rec *store.Record) (key string, pageCount int, isPDF bool, ok bool) {
	refID := fmt.Sprintf("%v", rec.Data[cp.ReferenceField])
	if refID == "" || refID == "<nil>" {
		return "", 0, false, false
	}
	previewRec, err := h.records.Get(r.Context(), refID)
	if err != nil {
		return "", 0, false, false
	}
	key, _ = previewRec.Data[cp.PreviewField].(string)
	if key == "" {
		return "", 0, false, false
	}
	isPDF = filepath.Ext(key) == ".pdf"
	pageCount = 1
	if isPDF {
		if pc, err := pdfapi.PageCountFile(filepath.Join(uploadsDir, key)); err == nil && pc > 0 {
			pageCount = pc
		}
	}
	return key, pageCount, isPDF, true
}

// coordPlaceOwnerOK mirrors permission.Guard.CanTrigger's own OwnerField
// comparison (internal/permission/guard.go) but against a Permission row
// generally, not tied to one eventID -- a coord_placement write is a plain
// field write (BoardMove's own category), not an event trigger, so there's
// no eventID to match against. A role with no OwnerField declared is
// permissive, the same default CanTrigger uses; identityID, never a display
// name (CAP-F05 -- OwnerField always names a `user` Field, which stores an
// id, matching CanTrigger's own established comparison).
func coordPlaceOwnerOK(machine *model.Machine, roles []string, identityID string, recordData map[string]any) bool {
	for _, perm := range machine.Permissions {
		if !slices.Contains(roles, perm.Role) {
			continue
		}
		if perm.OwnerField == "" {
			return true
		}
		return fmt.Sprintf("%v", recordData[perm.OwnerField]) == identityID
	}
	return true // guard.CanEdit already gated on a matching row existing at all
}

// parsePagePositive parses s as a page number -- must be a real integer
// >= 1, never silently coerced from a float or a missing/garbage value the
// way toFloat's zero-default would (a page number of 0 is never valid, so
// this can't reuse toFloat's tolerant parsing).
func parsePagePositive(s string) (int, error) {
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0, err
	}
	if n < 1 {
		return 0, fmt.Errorf("page must be >= 1, got %d", n)
	}
	return n, nil
}

func clamp01to100(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 100 {
		return 100
	}
	return v
}
