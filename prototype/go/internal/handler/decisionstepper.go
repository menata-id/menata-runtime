package handler

import (
	"fmt"
	"log/slog"
	"net/http"
	"sort"

	"github.com/go-chi/chi/v5"

	"menata.id/runtime/internal/store"
	"menata.id/runtime/internal/ui"
)

// DecisionStepper (CAP-V20) renders a "decision_stepper" View: this
// record's own child records (found via Machine.Config's existing
// steps_machine/steps_parent_field, CAP-X03 -- the same config keys
// approval-document.yaml has carried since Case 3's original build,
// previously read by no code) as an ordered done/current/pending
// progress indicator, with the CURRENT step's real Approve/Reject buttons
// -- PermittedEventsForRecord (already CAP-P02 ownership-filtered) and the
// existing /{machineID}/{recordID}/events/{eventID} route, no new write
// path. Purely presentational: nothing here is stored, recomputed fresh
// every request from the children's own already-declared fields.
func (h *Handler) DecisionStepper(w http.ResponseWriter, r *http.Request) {
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
	view := h.interp.Get().DecisionStepperView(machineID)
	if view == nil || view.Config.DecisionStepper == nil {
		http.NotFound(w, r)
		return
	}
	ds := view.Config.DecisionStepper
	stepsMachineID := machine.Config["steps_machine"]
	stepsParentField := machine.Config["steps_parent_field"]
	if stepsMachineID == "" || stepsParentField == "" {
		http.NotFound(w, r)
		return
	}
	stepsMachine, ok := h.interp.Get().GetMachine(stepsMachineID)
	if !ok {
		http.NotFound(w, r)
		return
	}
	rec, err := h.records.Get(r.Context(), recordID)
	if err != nil || rec.MachineID != machineID {
		http.NotFound(w, r)
		return
	}

	// Same filter-in-Go shape childLists already uses (formfields.go) --
	// fetch every record on the steps Machine, keep the ones whose own
	// stepsParentField points back at this record.
	all, err := h.records.List(r.Context(), stepsMachineID, "", "")
	if err != nil {
		http.Error(w, "failed to load steps", http.StatusInternalServerError)
		return
	}
	var children []*store.Record
	for _, c := range all {
		if fmt.Sprintf("%v", c.Data[stepsParentField]) == recordID {
			children = append(children, c)
		}
	}
	sort.Slice(children, func(i, j int) bool {
		return toFloat(children[i].Data[ds.SequenceField]) < toFloat(children[j].Data[ds.SequenceField])
	})

	// approval_mode_field is the same Machine.Config key CAP-A07's own
	// sequential guard already reads (events.go) -- Parallel means every
	// still-Pending step is simultaneously "current", matching that guard's
	// own no-gating behavior for that mode.
	parallelMode := false
	if modeField := machine.Config["approval_mode_field"]; modeField != "" {
		parallelMode = fmt.Sprintf("%v", rec.Data[modeField]) == "Parallel"
	}

	identityID := h.identityID(r)
	firstPendingAssigned := false
	steps := make([]ui.StepperStep, 0, len(children))
	for _, c := range children {
		decided := fmt.Sprintf("%v", c.Data[ds.DecisionField]) != "Pending"
		state := "pending"
		switch {
		case decided:
			state = "done"
		case parallelMode, !firstPendingAssigned:
			state = "current"
			firstPendingAssigned = true
		}

		var triggers []ui.EventTrigger
		if state == "current" {
			for _, evt := range h.interp.Get().PermittedEventsForRecord(stepsMachineID, role, identityID, c.Data) {
				trig := ui.EventTrigger{Event: evt}
				if len(evt.InputFields) > 0 {
					trig.Inputs = h.buildFormFieldsFor(r.Context(), stepsMachine, evt.InputFields, nil)
				}
				triggers = append(triggers, trig)
			}
		}

		steps = append(steps, ui.StepperStep{
			Label:         displayLabel(stepsMachine, c.Data),
			State:         state,
			Triggers:      triggers,
			StepMachineID: stepsMachineID,
			RecordID:      c.ID,
		})
	}

	a := h.auth(r)
	page := ui.DecisionStepper(h.workspaceName(r), a.User.Name, a.CSRFToken, h.isWorkspaceAdmin(r), machine, rec, view.Name, steps, h.unreadCount(r.Context(), a), h.subNavFor(r, machine))
	if err := page.Render(r.Context(), w); err != nil {
		slog.Error("render decision stepper", "error", err)
	}
}
