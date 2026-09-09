package handler

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"sort"

	"github.com/go-chi/chi/v5"

	"menata.id/app/internal/model"
	"menata.id/app/internal/store"
	"menata.id/app/internal/ui"
)

// ChildLinesTemplate (CAP-V28) is the HTMX fragment endpoint a form's own
// TriggerField swaps in on change -- see ChildLinesTemplateConfig's own doc
// comment (internal/model/model.go) for the full "category → saved
// config → prefilled instance" shape. Same permission gate as the Create
// form itself (CanCreate): this is a preview of what Create would prefill,
// not a read of arbitrary other records, so it's gated the same way.
func (h *Handler) ChildLinesTemplate(w http.ResponseWriter, r *http.Request) {
	machineID := chi.URLParam(r, "machineID")
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
	if !h.guard.CanCreate(machine, role) {
		h.logPermissionDenied(r.Context(), "create", machineID, "", role, h.identity(r))
		http.Error(w, "not permitted", http.StatusForbidden)
		return
	}
	view := h.interp.Get().FormView(machineID)
	if view == nil || view.Config.ChildLines == nil || view.Config.ChildLinesTemplate == nil {
		http.NotFound(w, r)
		return
	}
	cl := view.Config.ChildLines
	clt := view.Config.ChildLinesTemplate
	childMachine, ok := h.interp.Get().GetMachine(cl.Machine)
	if !ok {
		http.NotFound(w, r)
		return
	}

	var prefill map[int]map[string]string
	if triggerValue := r.URL.Query().Get(clt.TriggerField); triggerValue != "" {
		if tmpl := h.findTemplateRecord(r.Context(), clt, triggerValue); tmpl != nil {
			prefill = h.templateChildValues(r.Context(), clt, tmpl.ID)
		}
	}

	rows := h.childLinesRows(r.Context(), childMachine, cl, prefill)
	frag := ui.ChildLinesSection(childMachine.Name, rows, "", "", "", "")
	if err := frag.Render(r.Context(), w); err != nil {
		slog.Error("render child lines template fragment", "error", err)
	}
}

// findTemplateRecord (CAP-V28) returns the first ChildLinesTemplateConfig.
// TemplateMachine record whose MatchField equals value, or nil if none
// matches. One template per category is assumed, not enforced -- named,
// not silently dropped: a real uniqueness constraint on (MatchField) is
// future work if two templates for the same category ever proves a real
// authoring mistake worth blocking, which CAP-V28's own registry row
// doesn't name as in scope today.
func (h *Handler) findTemplateRecord(ctx context.Context, clt *model.ChildLinesTemplateConfig, value string) *store.Record {
	all, err := h.records.List(ctx, clt.TemplateMachine, "", "")
	if err != nil {
		return nil
	}
	for _, rec := range all {
		if fmt.Sprintf("%v", rec.Data[clt.MatchField]) == value {
			return rec
		}
	}
	return nil
}

// templateChildValues (CAP-V28) fetches every ChildMachine record whose own
// ChildParentField points at templateID, orders them by ChildSequenceField
// (the same "filter every record of a Machine in Go, then sort by its own
// sequence field" shape decisionstepper.go's computeStepperSteps already
// uses for the equivalent reverse lookup one layer up), and maps each into
// a row of prefill values via ChildFieldMap (host child_lines field id →
// template child field id). Row index i is the row SLOT it prefills --
// childLinesRows reads prefill[i][hostFieldID].
func (h *Handler) templateChildValues(ctx context.Context, clt *model.ChildLinesTemplateConfig, templateID string) map[int]map[string]string {
	all, err := h.records.List(ctx, clt.ChildMachine, "", "")
	if err != nil {
		return nil
	}
	var matched []*store.Record
	for _, rec := range all {
		if fmt.Sprintf("%v", rec.Data[clt.ChildParentField]) == templateID {
			matched = append(matched, rec)
		}
	}
	sort.Slice(matched, func(i, j int) bool {
		return toFloat(matched[i].Data[clt.ChildSequenceField]) < toFloat(matched[j].Data[clt.ChildSequenceField])
	})

	out := map[int]map[string]string{}
	for i, rec := range matched {
		row := map[string]string{}
		for hostField, tmplField := range clt.ChildFieldMap {
			if v, ok := rec.Data[tmplField]; ok {
				row[hostField] = fmt.Sprintf("%v", v)
			}
		}
		out[i] = row
	}
	return out
}
