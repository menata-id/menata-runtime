package handler

import (
	"bytes"
	"fmt"
	"log/slog"
	"net/http"
	"path/filepath"
	"slices"
	"strconv"

	"github.com/go-chi/chi/v5"
	pdfapi "github.com/pdfcpu/pdfcpu/pkg/api"
	pdfmodel "github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"

	"menata.id/app/internal/model"
	"menata.id/app/internal/store"
	"menata.id/app/internal/ui"
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
	previewKey, pageCount, ok := h.coordPlacePreview(r, cp, rec)
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
	pageComp := ui.CoordPlace(h.workspaceName(r), h.workspaceSlug(r), a.User.Name, a.CSRFToken, h.isWorkspaceAdmin(r), machine, rec, view.Name, previewKey, page, pageCount, x, y, editable, h.unreadCount(r.Context(), a), h.subNavFor(r, machine))
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
	_, pageCount, ok := h.coordPlacePreview(r, cp, rec)
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
	http.Redirect(w, r, "/"+h.workspaceSlug(r)+"/"+machineID+"/"+recordID+"/place", http.StatusSeeOther)
}

// renderCoordPlacementChild (CAP-V20 Tier 2) is the "coord_placement" case
// of embed.go's renderChildView dispatch -- see that switch's own doc
// comment. Unlike renderDecisionStepperChild, this one is same-record: a
// coord_placement View's own Config (ReferenceField/PageField/XField/
// YField) names Fields on the SAME record being embedded, not a different
// parent's -- Approval Step's own Detail page embeds ITS OWN signature
// position, not some other record's, closing document-approval.html's own
// "Your Signature Position" panel exactly as drawn (same screen as the
// Approve/Reject bar). Reuses coordPlacePreview and coordPlaceOwnerOK
// verbatim -- the exact same resolution/ownership logic CoordPlace (the
// standalone /place route above) already runs, just against hostRec
// directly instead of a URL param. Returns nil whenever there's genuinely
// nothing to show -- deliberately no error return, same "purely additive
// extra" posture as renderDecisionStepperChild. Two DIFFERENT reasons can
// produce that nil, logged at two different levels on purpose: no preview
// yet (coordPlacePreview's own `ok=false` -- e.g. the reference isn't set,
// or the file hasn't been uploaded) is an ordinary in-progress record, not
// worth a log line; hostRec.MachineID not resolving to a real Machine
// would mean this record's own data is pointing at something metadata no
// longer declares -- structurally shouldn't happen given load-time
// validation, exactly the class of drift this codebase has hit for real
// before (see capability-registry.md's CAP-V20 row, the mch_ca_lifted
// incident), so it's worth a Warn with enough context to actually debug.
func (h *Handler) renderCoordPlacementChild(r *http.Request, hostRec *store.Record, view *model.View) *ui.EmbeddedSection {
	cp := view.Config.CoordPlacement
	previewKey, pageCount, ok := h.coordPlacePreview(r, cp, hostRec)
	if !ok {
		return nil
	}
	hostMachine, ok := h.interp.Get().GetMachine(hostRec.MachineID)
	if !ok {
		slog.Warn("coord_placement child embed: host record's own machine_id does not resolve",
			"view", view.ID, "record", hostRec.ID, "machine_id", hostRec.MachineID)
		return nil
	}
	page := int(toFloat(hostRec.Data[cp.PageField]))
	if page < 1 {
		page = 1
	}
	if page > pageCount {
		page = pageCount
	}
	x, y := toFloat(hostRec.Data[cp.XField]), toFloat(hostRec.Data[cp.YField])
	if x == 0 && y == 0 {
		x, y = 50, 50 // an unset pin starts centered, matching CoordPlace's own default
	}
	_, appID := h.interp.Get().ScopeFor(hostMachine.ID)
	role := h.roleForApp(r, appID)
	editable := h.guard.CanEdit(hostMachine, role) && coordPlaceOwnerOK(hostMachine, role, h.identityID(r), hostRec.Data)

	sec := &ui.EmbeddedSection{
		Title:   view.Name,
		Content: ui.CoordPlacePreview(h.workspaceSlug(r), h.auth(r).CSRFToken, hostMachine.ID, hostRec.ID, previewKey, page, pageCount, x, y, editable),
	}
	if pageCount > 1 {
		// Page-switching only works on the standalone /place route (its own
		// handler reads ?page= -- see CoordPlacePreview's own doc comment
		// for why the embedded copy can't support that itself).
		sec.ActionLabel = "Open full preview →"
		sec.ActionHref = "/" + h.workspaceSlug(r) + "/" + hostMachine.ID + "/" + hostRec.ID + "/place"
	}
	return sec
}

// coordPlacePreview resolves the referenced record's own preview file,
// returning its storage key, page count (1 for a non-PDF image -- CAP-V21
// is not signature/PDF-specific, "equally usable to mark a defect location
// on an equipment photo" per its own registry row), and whether a usable
// preview was found at all.
//
// Deliberately does NOT return "is this a PDF" (2026-09-07 cleanup) --
// isPDF used to be a fourth return value, threaded all the way down through
// both callers into ui.CoordPlace/ui.CoordPlacePreview as its own separate
// bool parameter, even though it's a pure function of `key` (its own file
// extension) with no other information in it. A render function deciding
// `<object>` vs `<img>` should derive that itself from the previewKey it
// already has (CoordPlacePreview's own isPDFPreview, coordplace.templ) --
// not be handed a second, independently-computed fact that could in
// principle disagree with the first. This function still computes isPDF
// LOCALLY, because IT has a real, different reason to need it (whether to
// even attempt counting PDF pages below) -- that's a backend concern
// unrelated to the render layer's own tag choice, so it stays internal.
func (h *Handler) coordPlacePreview(r *http.Request, cp *model.CoordPlacementConfig, rec *store.Record) (key string, pageCount int, ok bool) {
	refID := fmt.Sprintf("%v", rec.Data[cp.ReferenceField])
	if refID == "" || refID == "<nil>" {
		return "", 0, false
	}
	previewRec, err := h.records.Get(r.Context(), refID)
	if err != nil {
		slog.Warn("coord_placement reference_field points at a record that no longer resolves",
			"reference_field", cp.ReferenceField, "referenced_id", refID, "error", err)
		return "", 0, false
	}
	key, _ = previewRec.Data[cp.PreviewField].(string)
	if key == "" {
		return "", 0, false
	}
	pageCount = 1
	if filepath.Ext(key) == ".pdf" {
		if data, _, err := h.storage.Get(key); err == nil {
			if pc, err := pdfapi.PageCount(bytes.NewReader(data), pdfmodel.NewDefaultConfiguration()); err == nil && pc > 0 {
				pageCount = pc
			}
		} else {
			slog.Warn("coord_placement preview file missing from storage", "key", key, "error", err)
		}
	}
	return key, pageCount, true
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
