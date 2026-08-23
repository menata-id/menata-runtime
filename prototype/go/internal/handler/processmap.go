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
	statusField := model.FindFieldByName(machine, "Status")
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

// liftProcess (CAP-W05 backward direction, B6) reconstructs a re-loadable
// *model.Process from a Machine's own compiled shape -- the exact same
// detection extractProcessMap already does (Status field + state-guarded
// Events), but un-flattened back into model.Process's own struct shape
// (Actor as {Role, OwnerField}, not a joined display string) instead of a
// UI display list, so the result can be pasted directly into a Machine's
// `process` column and reloaded (CAP-X04) rather than only rendered.
//
// Deliberately narrow, named explicitly (no case in case-portfolio.md
// forces solving the harder version): reconstructs States, Transitions
// (Name/From/To/Actor/on_transition), and Auto only. Requirements/SLA/
// change_policy are NOT reverse-engineered -- a hand-authored counter+
// Constraint pair is indistinguishable from one that started life as a
// CAP-W01 requirement, genuinely ambiguous with no way to disambiguate
// from the compiled shape alone.
func liftProcess(machine *model.Machine) (*model.Process, error) {
	statusField := model.FindFieldByName(machine, "Status")
	if statusField == nil || statusField.Type != model.FieldTypeValueList {
		return nil, fmt.Errorf("machine %s has no value_list Status field -- nothing to lift", machine.ID)
	}

	isStateGuarded := func(ev *model.Event) (from, to string, ok bool) {
		if ev.Condition == nil || ev.Condition.Field != statusField.ID || ev.Condition.Operator != "equals" {
			return "", "", false
		}
		for _, a := range ev.Actions {
			if a.Type == model.ActionSetField {
				if f, _ := a.Params["field"].(string); f == statusField.ID {
					v, _ := a.Params["value"].(string)
					return ev.Condition.Value, v, v != ""
				}
			}
		}
		return "", "", false
	}
	actorFor := func(eventID string) *model.ProcessActor {
		for _, perm := range machine.Permissions {
			for _, e := range perm.Events {
				if e == eventID {
					return &model.ProcessActor{Role: perm.Role, OwnerField: perm.OwnerField}
				}
			}
		}
		return nil // no Permission grants it -- CAP-E05's own auto/System-chained convention
	}

	// Pass 1: which state-guarded events are auto-shaped (no Permission
	// grants them) -- needed before Pass 2 so a hand-written trigger_event
	// action chaining INTO one of these can be recognized and skipped, not
	// captured into on_transition. Auto is a structural declaration
	// (compileProcess regenerates the identical trigger_event chain from
	// it); keeping a hand-written copy in on_transition too would double
	// the chain once this output is recompiled.
	autoEventIDs := make(map[string]bool)
	for _, ev := range machine.Events {
		if _, _, ok := isStateGuarded(ev); ok && actorFor(ev.ID) == nil {
			autoEventIDs[ev.ID] = true
		}
	}

	p := &model.Process{States: append([]string{}, statusField.Options.Values...)}
	for _, ev := range machine.Events {
		from, to, ok := isStateGuarded(ev)
		if !ok {
			continue // an ordinary business event, not a transition edge
		}
		if autoEventIDs[ev.ID] {
			p.Auto = append(p.Auto, &model.ProcessAuto{From: from, To: to})
			continue
		}
		actor := actorFor(ev.ID)
		if actor == nil {
			continue // unreachable given autoEventIDs above, but never emit a Transition with no actor
		}
		var onTransition []*model.ProcessAction
		skippedStateSetter := false
		for _, a := range ev.Actions {
			if !skippedStateSetter && a.Type == model.ActionSetField {
				if f, _ := a.Params["field"].(string); f == statusField.ID {
					skippedStateSetter = true
					continue
				}
			}
			if a.Type == model.ActionTriggerEvent {
				if target, _ := a.Params["event"].(string); autoEventIDs[target] {
					continue // this exact chain is what the emitted Auto entry above already declares
				}
			}
			onTransition = append(onTransition, &model.ProcessAction{Type: a.Type, Params: a.Params})
		}
		p.Transitions = append(p.Transitions, &model.ProcessTransition{
			Name: ev.Name, From: from, To: to, Actor: *actor, Actions: onTransition,
		})
	}
	if len(p.Transitions) == 0 {
		return nil, fmt.Errorf("machine %s has no state-guarded transitions with a real actor -- nothing to lift", machine.ID)
	}
	return p, nil
}

// LiftProcess (B6) renders a Machine's lifted Process as downloadable JSON
// -- Admin-only, matching CAP-X08's APIExportApplication pattern. Labeled a
// draft in the UI on purpose: this is authoring material to review, never
// applied automatically -- consistent with this project's own "form-based
// authoring, not a visual builder" non-goal.
func (h *Handler) LiftProcess(w http.ResponseWriter, r *http.Request) {
	if !h.isWorkspaceAdmin(r) {
		apiJSON(w, http.StatusForbidden, map[string]string{"error": "not permitted"})
		return
	}
	machineID := chi.URLParam(r, "machineID")
	machine, ok := h.interp.Get().GetMachine(machineID)
	if !ok {
		http.NotFound(w, r)
		return
	}
	workspaceID, _ := h.interp.Get().ScopeFor(machineID)
	if workspaceID != h.workspace(r) {
		http.NotFound(w, r)
		return
	}
	process, err := liftProcess(machine)
	if err != nil {
		apiJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": err.Error()})
		return
	}
	apiJSON(w, http.StatusOK, map[string]any{
		"draft":   true,
		"note":    "Draft lifted from " + machine.ID + " -- review before pasting into a Machine's process column.",
		"process": process,
	})
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
	page := ui.ProcessMap(h.workspaceName(r), a.User.Name, a.CSRFToken, h.isWorkspaceAdmin(r), view.Name, states, initial, edges, hasShape, h.unreadCount(r.Context(), a), h.subNavFor(r, machine))
	if err := page.Render(r.Context(), w); err != nil {
		slog.Error("render process map", "error", err)
	}
}
