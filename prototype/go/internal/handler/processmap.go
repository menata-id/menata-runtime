package handler

import (
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"menata.id/runtime/internal/model"
	"menata.id/runtime/internal/ui"
)

// extractProcessMap (CAP-W05, Process Overlay B2) derives a Machine's
// process shape from what is already loaded -- its Status value_list Field
// and every Event whose own Condition/Actions have the state-guard shape
// (CAP-E06 equality guard + a set_field on that same Field). It never reads
// machine.Process: B1 already proved a compiled Machine and a hand-authored
// one produce byte-identical Event/Permission structs, so deriving the map
// from THAT shared shape, rather than from the raw declaration, makes this
// function work identically on either origin -- and on any pre-existing v1
// Machine that never heard of `process` at all (the decompile direction).
//
// Returns ok=false when the Machine has no Status value_list Field at all
// -- nothing to build a map from, not an error.
func extractProcessMap(machine *model.Machine) (states []string, initial string, edges []ui.ProcessEdge, ok bool) {
	statusField := findFieldByName(machine, "Status")
	if statusField == nil || statusField.Type != model.FieldTypeValueList {
		return nil, "", nil, false
	}
	states = statusField.Options.Values
	if len(states) > 0 {
		initial = states[0]
	}

	// actorsFor an event id: every Permission row that grants it, "Role" or
	// "Role (owner: FieldName)" when it also narrows by ownership (CAP-P02).
	actorsFor := func(eventID string) string {
		var actors []string
		for _, p := range machine.Permissions {
			for _, e := range p.Events {
				if e != eventID {
					continue
				}
				label := p.Role
				if p.OwnerField != "" {
					if f := findFieldByID(machine, p.OwnerField); f != nil {
						label = fmt.Sprintf("%s (owner: %s)", p.Role, f.Name)
					} else {
						label = fmt.Sprintf("%s (owner)", p.Role)
					}
				}
				actors = append(actors, label)
				break
			}
		}
		if len(actors) == 0 {
			return "System" // no human Permission grants it -- an auto/System-chained transition (CAP-E05)
		}
		return strings.Join(actors, ", ")
	}

	for _, ev := range machine.Events {
		if ev.Condition == nil || ev.Condition.Field != statusField.ID || ev.Condition.Operator != "equals" {
			continue // not a state-guarded transition -- an ordinary business event, not an edge
		}
		var to string
		for _, a := range ev.Actions {
			if a.Type == model.ActionSetField {
				if f, _ := a.Params["field"].(string); f == statusField.ID {
					to, _ = a.Params["value"].(string)
					break
				}
			}
		}
		if to == "" {
			continue // guarded by state but never actually moves it -- not a transition edge
		}
		edges = append(edges, ui.ProcessEdge{
			Name:  ev.Name,
			From:  ev.Condition.Value,
			To:    to,
			Actor: actorsFor(ev.ID),
		})
	}
	return states, initial, edges, true
}

// ProcessMap (CAP-W05) renders the read-only process map page. Same shape
// as Report (CAP-V13, handler.go): GetMachine -> workspace scope -> CanRead
// -> the View's own opt-in (nil = 404, same posture as every other
// auxiliary View type) -> extract -> render.
func (h *Handler) ProcessMap(w http.ResponseWriter, r *http.Request) {
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
	if !h.guard.CanRead(machine, role) {
		h.logPermissionDenied(r.Context(), "read", machineID, "", role, h.identity(r))
		http.Error(w, "not permitted", http.StatusForbidden)
		return
	}
	view := h.interp.Get().ProcessMapView(machineID)
	if view == nil {
		http.NotFound(w, r)
		return
	}
	states, initial, edges, hasShape := extractProcessMap(machine)

	a := h.auth(r)
	page := ui.ProcessMap(a.User.Name, a.CSRFToken, h.isWorkspaceAdmin(r), view.Name, states, initial, edges, hasShape, h.unreadCount(r.Context(), a), h.subNavFor(r, machine))
	if err := page.Render(r.Context(), w); err != nil {
		slog.Error("render process map", "error", err)
	}
}
