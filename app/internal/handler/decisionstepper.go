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
	rec, err := h.records.Get(r.Context(), recordID)
	if err != nil || rec.MachineID != machineID {
		http.NotFound(w, r)
		return
	}
	steps, err := h.computeStepperSteps(r, role, machine, view, rec)
	if err != nil {
		http.Error(w, "failed to load steps", http.StatusInternalServerError)
		return
	}
	if steps == nil {
		http.NotFound(w, r) // Config.steps_machine/steps_parent_field missing -- same as before this helper existed
		return
	}

	a := h.auth(r)
	page := ui.DecisionStepper(h.workspaceName(r), h.workspaceSlug(r), a.User.Name, a.CSRFToken, h.isWorkspaceAdmin(r), machine, rec, view.Name, steps, h.unreadCount(r.Context(), a), h.subNavFor(r, machine))
	if err := page.Render(r.Context(), w); err != nil {
		slog.Error("render decision stepper", "error", err)
	}
}

// computeStepperSteps (CAP-V20) is DecisionStepper's own step-computation
// core, extracted so Detail (record_crud.go) can reuse it verbatim for the
// inline-progress fix below -- both the full-page stepper (DecisionStepper
// above) and the inline card on a Step's own Detail page need the EXACT
// same done/current/pending + per-step triggers logic, and duplicating it
// would let the two silently diverge the next time either changes. Returns
// (nil, nil) -- not an error -- when the parent Machine's own Config is
// missing steps_machine/steps_parent_field, matching DecisionStepper's own
// pre-existing "404, not a 500" posture for a Machine that declares the
// View but not its Config.
func (h *Handler) computeStepperSteps(r *http.Request, role []string, machine *model.Machine, view *model.View, rec *store.Record) ([]ui.StepperStep, error) {
	ds := view.Config.DecisionStepper
	stepsMachineID := machine.Config["steps_machine"]
	stepsParentField := machine.Config["steps_parent_field"]
	if stepsMachineID == "" || stepsParentField == "" {
		return nil, nil
	}
	stepsMachine, ok := h.interp.Get().GetMachine(stepsMachineID)
	if !ok {
		return nil, nil
	}

	// Same filter-in-Go shape childLists already uses (formfields.go) --
	// fetch every record on the steps Machine, keep the ones whose own
	// stepsParentField points back at this record.
	all, err := h.records.List(r.Context(), stepsMachineID, "", "")
	if err != nil {
		return nil, err
	}
	var children []*store.Record
	for _, c := range all {
		if fmt.Sprintf("%v", c.Data[stepsParentField]) == rec.ID {
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
			for _, evt := range h.interp.Get().PermittedEventsForRecord(stepsMachineID, role, identityID, c.Data, h.groupMembersFunc(r.Context())) {
				trig := ui.EventTrigger{Event: evt}
				if len(evt.InputFields) > 0 {
					trig.Inputs = h.buildFormFieldsFor(r.Context(), h.workspaceSlug(r), stepsMachine, evt.InputFields, nil)
				}
				triggers = append(triggers, trig)
			}
		}

		steps = append(steps, ui.StepperStep{
			Label:         stepLabel(h, r.Context(), stepsMachine, ds, c),
			State:         state,
			Triggers:      triggers,
			StepMachineID: stepsMachineID,
			RecordID:      c.ID,
		})
	}
	return steps, nil
}

// renderDecisionStepperChild (CAP-V20 Tier 2) is the "decision_stepper"
// case of embed.go's renderChildView dispatch -- the Type-specific half of
// embedding a decision-stepper View as another View's own Child. hostRec is
// whatever record the HOST page is showing (e.g. one Approval Step); view
// is the ALREADY-RESOLVED, already-type-checked decision_stepper View being
// embedded (e.g. Approval Document's own vw_ad_progress) -- resolving which
// View to embed at all, and confirming it's actually this Type, is
// renderChildView's own job, not this function's.
//
// Two cases, both reusing computeStepperSteps verbatim -- the exact same
// computation the full-page /progress route (DecisionStepper above)
// already runs. (1) Self-host: hostRec's own Machine IS the stepper's
// target Machine (e.g. Approval Document's own detail View embedding its
// own vw_ad_progress) -- hostRec needs no further resolution. (2)
// Child-embeds-parent: hostRec is a DIFFERENT Machine's record (e.g. one
// Approval Step) that names its parent via whatever Field the stepper's
// owning Machine declares as Config["steps_parent_field"] -- that parent
// is looked up and used instead. Returns nil whenever there's genuinely
// nothing to show (a child hostRec doesn't actually reference a parent,
// the role can't read the target Machine, or a referenced parent record
// can't be loaded) -- deliberately no error return, since embedding a
// child is a purely additive extra on a page whose real job (show the
// host record's own fields) must still render even when this fails.
func (h *Handler) renderDecisionStepperChild(r *http.Request, hostRec *store.Record, view *model.View) *ui.EmbeddedSection {
	parent, ok := h.interp.Get().GetMachine(view.MachineID)
	if !ok {
		// view came from the interpreter's own index (renderChildView's
		// GetView call) -- its own MachineID not resolving to a real
		// Machine in that same interpreter would mean the in-memory model
		// is internally inconsistent, not just "nothing to show here."
		slog.Warn("decision_stepper child embed: view's own machine_id does not resolve",
			"view", view.ID, "machine_id", view.MachineID)
		return nil
	}
	// Self-host case (2026-09-10, Case 3 extension note): the page showing
	// its OWN stepper -- e.g. Approval Document's own vw_ad_detail
	// embedding vw_ad_progress -- hostRec already IS the stepper's target
	// record, so there is no steps_parent_field indirection to follow.
	// Distinct from the branch below (a Step embedding its parent
	// Document's stepper), where hostRec is a DIFFERENT Machine's record
	// and steps_parent_field names which field on IT points back to the
	// parent this stepper is actually about.
	if hostRec.MachineID == parent.ID {
		_, appID := h.interp.Get().ScopeFor(parent.ID)
		role := h.roleForApp(r, appID)
		if !h.guard.CanRead(parent, role) {
			return nil
		}
		steps, err := h.computeStepperSteps(r, role, parent, view, hostRec)
		if err != nil {
			slog.Warn("decision_stepper child embed: failed to compute steps", "view", view.ID, "parent_record", hostRec.ID, "error", err)
			return nil
		}
		if steps == nil {
			slog.Warn("decision_stepper child embed: parent machine has no steps_machine/steps_parent_field configured",
				"view", view.ID, "parent_machine", parent.ID)
			return nil
		}
		return &ui.EmbeddedSection{Title: view.Name, Content: ui.StepperList(h.workspaceSlug(r), h.auth(r).CSRFToken, steps)}
	}

	stepsParentField := parent.Config["steps_parent_field"]
	parentID := fmt.Sprintf("%v", hostRec.Data[stepsParentField])
	if parentID == "" || parentID == "<nil>" {
		// Ordinary, expected state -- e.g. a Step whose own parent
		// reference field genuinely isn't set (yet). Not logged: this is
		// ambient data state, not a configuration problem.
		return nil
	}
	_, parentAppID := h.interp.Get().ScopeFor(parent.ID)
	parentRole := h.roleForApp(r, parentAppID)
	if !h.guard.CanRead(parent, parentRole) {
		// Ordinary access-control outcome (this session's own role can't
		// read the parent Machine) -- not logged, same as any other
		// permission-scoped content simply not appearing for this actor.
		return nil
	}
	parentRec, err := h.records.Get(r.Context(), parentID)
	if err != nil {
		// hostRec's own reference field named a parent id that doesn't
		// resolve -- a dangling reference, real data-integrity signal.
		slog.Warn("decision_stepper child embed: parent reference does not resolve to a real record",
			"view", view.ID, "host_record", hostRec.ID, "parent_field", stepsParentField, "parent_id", parentID, "error", err)
		return nil
	}
	steps, err := h.computeStepperSteps(r, parentRole, parent, view, parentRec)
	if err != nil {
		slog.Warn("decision_stepper child embed: failed to compute steps", "view", view.ID, "parent_record", parentRec.ID, "error", err)
		return nil
	}
	if steps == nil {
		// computeStepperSteps' own (nil, nil) means the PARENT Machine
		// (parent, above) is missing Config.steps_machine/steps_parent_field
		// -- a real metadata gap: something declared this decision_stepper
		// View as embeddable, but the Machine it belongs to never finished
		// the CAP-X03 config CAP-V20 depends on.
		slog.Warn("decision_stepper child embed: parent machine has no steps_machine/steps_parent_field configured",
			"view", view.ID, "parent_machine", parent.ID)
		return nil
	}
	return &ui.EmbeddedSection{Title: view.Name, Content: ui.StepperList(h.workspaceSlug(r), h.auth(r).CSRFToken, steps)}
}

// stepLabel names one step "Step <sequence> — <assignee>" when it can,
// falling back to displayLabel's own generic rule otherwise. A step-shaped
// child Machine (Approval Step and anything else CAP-V20 gets pointed at)
// never has a plain-text Field of its own -- displayLabel's fallback to the
// record's own id (a raw UUID) is correct but reads as a technical, not a
// business, label on a screen whose whole point is to look like a real
// approval stepper. Prototype-honest heuristic, same posture as CAP-A07's
// own Sequence/Decision/Approver name-matching: the FIRST `user`-typed Field
// on the step Machine is assumed to be its assignee, since that's what
// every case built on this View shape (Approval Step today) actually means
// by "who this step belongs to" -- not a new metadata concept.
func stepLabel(h *Handler, ctx context.Context, stepsMachine *model.Machine, ds *model.DecisionStepperConfig, c *store.Record) string {
	seq, hasSeq := c.Data[ds.SequenceField]
	if !hasSeq {
		return displayLabel(stepsMachine, c.ID, c.Data)
	}
	for _, f := range stepsMachine.Fields {
		if f.Type != model.FieldTypeUser {
			continue
		}
		uid, _ := c.Data[f.ID].(string)
		if uid == "" {
			continue
		}
		if name, err := h.userLabel(ctx, uid); err == nil && name != "" {
			return fmt.Sprintf("Step %v — %s", seq, name)
		}
		break
	}
	return fmt.Sprintf("Step %v", seq)
}
