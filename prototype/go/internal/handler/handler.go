package handler

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"menata.id/runtime/internal/constraint"
	"menata.id/runtime/internal/executor"
	"menata.id/runtime/internal/interpreter"
	"menata.id/runtime/internal/model"
	"menata.id/runtime/internal/permission"
	"menata.id/runtime/internal/store"
	"menata.id/runtime/internal/ui"
)

type Handler struct {
	interp        *interpreter.Interpreter
	records       *store.RecordStore
	notifications *store.NotificationStore
	engine        *constraint.Engine
	guard         *permission.Guard
	exec          *executor.Executor
}

func New(interp *interpreter.Interpreter, records *store.RecordStore, notifications *store.NotificationStore) *Handler {
	return &Handler{
		interp:        interp,
		records:       records,
		notifications: notifications,
		engine:        &constraint.Engine{},
		guard:         &permission.Guard{},
		exec:          executor.New(records, notifications),
	}
}

func (h *Handler) role(r *http.Request) string {
	c, err := r.Cookie("menata_role")
	if err != nil || c.Value == "" {
		return "Requester"
	}
	return c.Value
}

// Home — list of all machines.
func (h *Handler) Home(w http.ResponseWriter, r *http.Request) {
	machines := h.interp.AllMachines()
	cards := make([]ui.MachineCard, len(machines))
	for i, m := range machines {
		cards[i] = ui.MachineCard{
			ID:          m.ID,
			Name:        m.Name,
			Description: fmt.Sprintf("%d fields · %d events", len(m.Fields), len(m.Events)),
		}
	}
	if err := ui.Home(h.role(r), cards, h.unreadCount(r.Context(), h.role(r))).Render(r.Context(), w); err != nil {
		slog.Error("render home", "error", err)
	}
}

// LoginForm — role selection page.
func (h *Handler) LoginForm(w http.ResponseWriter, r *http.Request) {
	if err := ui.LoginPage(h.role(r), h.unreadCount(r.Context(), h.role(r))).Render(r.Context(), w); err != nil {
		slog.Error("render login", "error", err)
	}
}

// Login — set role cookie.
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	role := r.FormValue("role")
	if role == "" {
		role = "Requester"
	}
	http.SetCookie(w, &http.Cookie{Name: "menata_role", Value: role, Path: "/"})
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// List — list view of records for a machine.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	machineID := chi.URLParam(r, "machineID")
	machine, ok := h.interp.GetMachine(machineID)
	if !ok {
		http.NotFound(w, r)
		return
	}

	view := h.interp.DefaultListView(machineID)
	fieldByID := fieldIndex(machine)

	colIDs := []string{}
	if view != nil {
		colIDs = view.Config.Columns
	}
	cols := make([]ui.ColumnDef, 0, len(colIDs))
	for _, id := range colIDs {
		def := ui.ColumnDef{ID: id, Name: id}
		if f, ok := fieldByID[id]; ok {
			def.Name = f.Name
			def.Type = f.Type
		}
		cols = append(cols, def)
	}

	records, err := h.records.List(r.Context(), machineID)
	if err != nil {
		http.Error(w, "failed to load records", http.StatusInternalServerError)
		return
	}

	rows := make([]ui.ListRow, 0, len(records))
	for _, rec := range records {
		cells := make([]ui.ListCell, len(colIDs))
		for j, id := range colIDs {
			val := ""
			if v, ok := rec.Data[id]; ok {
				val = fmt.Sprintf("%v", v)
			}
			link := ""
			if cols[j].Type == model.FieldTypeReference && val != "" {
				refID := val
				target := fieldByID[id].Options.TargetMachine
				if label, err := h.referenceLabel(r.Context(), target, refID); err == nil && label != "" {
					val = label
					link = "/" + target + "/" + refID
				}
			}
			cells[j] = ui.ListCell{
				Value:         val,
				IsStatusBadge: cols[j].Type == model.FieldTypeValueList,
				Link:          link,
			}
		}
		rows = append(rows, ui.ListRow{ID: rec.ID, Cells: cells})
	}

	role := h.role(r)
	if err := ui.List(role, machine, cols, rows, h.interp.PermittedEvents(machineID, role), h.unreadCount(r.Context(), role)).Render(r.Context(), w); err != nil {
		slog.Error("render list", "error", err)
	}
}

// NewForm — form for creating a new record.
func (h *Handler) NewForm(w http.ResponseWriter, r *http.Request) {
	machineID := chi.URLParam(r, "machineID")
	machine, ok := h.interp.GetMachine(machineID)
	if !ok {
		http.NotFound(w, r)
		return
	}
	role := h.role(r)
	if err := ui.Form(role, machine, "", h.buildFormFields(r.Context(), machine, nil), nil, h.unreadCount(r.Context(), role)).Render(r.Context(), w); err != nil {
		slog.Error("render form", "error", err)
	}
}

// Create — handle new record form submission.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	machineID := chi.URLParam(r, "machineID")
	machine, ok := h.interp.GetMachine(machineID)
	if !ok {
		http.NotFound(w, r)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	// Any value_list field the Create form doesn't expose (Status, Decision, ...)
	// starts at its first declared value — the same "first value = initial
	// state" convention guides/writing-menata.md teaches .menata authors,
	// generalized from what was previously a "status"-named-field-only rule.
	formFieldIDs := map[string]bool{}
	if fv := h.interp.FormView(machine.ID); fv != nil {
		for _, id := range fv.Config.Fields {
			formFieldIDs[id] = true
		}
	}
	data := make(map[string]any)
	for _, f := range machine.Fields {
		if !formFieldIDs[f.ID] && f.Type == model.FieldTypeValueList && len(f.Options.Values) > 0 {
			data[f.ID] = f.Options.Values[0]
		}
	}
	for _, f := range machine.Fields {
		if v := r.FormValue(f.ID); v != "" {
			data[f.ID] = v
		}
	}

	violations := h.engine.Violations(machine, data)
	refViolations, err := h.referenceViolations(r.Context(), machine, data)
	if err != nil {
		http.Error(w, "failed to validate references", http.StatusInternalServerError)
		return
	}
	violations = append(violations, refViolations...)

	if len(violations) > 0 {
		role := h.role(r)
		if err := ui.Form(role, machine, "", h.buildFormFields(r.Context(), machine, data), violations, h.unreadCount(r.Context(), role)).Render(r.Context(), w); err != nil {
			slog.Error("render form (violations)", "error", err)
		}
		return
	}

	rec, err := h.records.Create(r.Context(), machineID, data)
	if err != nil {
		http.Error(w, "failed to create record", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/"+machineID+"/"+rec.ID, http.StatusSeeOther)
}

// EditForm — form for editing an existing record (CAP-R02). Reuses the same
// FormView/field set Create uses — Menata Language has no separate "edit
// form" view declared in metadata — pre-filled with the record's current
// data.
func (h *Handler) EditForm(w http.ResponseWriter, r *http.Request) {
	machineID := chi.URLParam(r, "machineID")
	recordID := chi.URLParam(r, "recordID")
	machine, ok := h.interp.GetMachine(machineID)
	if !ok {
		http.NotFound(w, r)
		return
	}
	rec, err := h.records.Get(r.Context(), recordID)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	role := h.role(r)
	if err := ui.Form(role, machine, recordID, h.buildFormFields(r.Context(), machine, rec.Data), nil, h.unreadCount(r.Context(), role)).Render(r.Context(), w); err != nil {
		slog.Error("render edit form", "error", err)
	}
}

// Update — handle edit form submission (CAP-R02). Only the fields the
// FormView exposes are overwritten; everything else on the record (Status,
// and any other field driven by events rather than the form) is carried
// over unchanged, the same "only touch what the form declares" rule Create
// applies to its value_list defaults.
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	machineID := chi.URLParam(r, "machineID")
	recordID := chi.URLParam(r, "recordID")
	machine, ok := h.interp.GetMachine(machineID)
	if !ok {
		http.NotFound(w, r)
		return
	}
	rec, err := h.records.Get(r.Context(), recordID)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	formFieldIDs := map[string]bool{}
	if fv := h.interp.FormView(machine.ID); fv != nil {
		for _, id := range fv.Config.Fields {
			formFieldIDs[id] = true
		}
	}
	data := make(map[string]any, len(rec.Data))
	for k, v := range rec.Data {
		data[k] = v
	}
	for _, f := range machine.Fields {
		if formFieldIDs[f.ID] {
			data[f.ID] = r.FormValue(f.ID)
		}
	}

	violations := h.engine.Violations(machine, data)
	refViolations, err := h.referenceViolations(r.Context(), machine, data)
	if err != nil {
		http.Error(w, "failed to validate references", http.StatusInternalServerError)
		return
	}
	violations = append(violations, refViolations...)

	if len(violations) > 0 {
		role := h.role(r)
		if err := ui.Form(role, machine, recordID, h.buildFormFields(r.Context(), machine, data), violations, h.unreadCount(r.Context(), role)).Render(r.Context(), w); err != nil {
			slog.Error("render form (violations)", "error", err)
		}
		return
	}

	if err := h.records.Update(r.Context(), recordID, data); err != nil {
		http.Error(w, "failed to update record", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/"+machineID+"/"+recordID, http.StatusSeeOther)
}

// Detail — detail view of a single record.
func (h *Handler) Detail(w http.ResponseWriter, r *http.Request) {
	machineID := chi.URLParam(r, "machineID")
	recordID := chi.URLParam(r, "recordID")

	machine, ok := h.interp.GetMachine(machineID)
	if !ok {
		http.NotFound(w, r)
		return
	}
	rec, err := h.records.Get(r.Context(), recordID)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	fields := make([]ui.DetailField, 0, len(machine.Fields))
	for _, f := range machine.Fields {
		val := ""
		if v, ok := rec.Data[f.ID]; ok {
			val = fmt.Sprintf("%v", v)
		}
		link := ""
		if f.Type == model.FieldTypeReference && val != "" {
			refID := val
			target := f.Options.TargetMachine
			if label, err := h.referenceLabel(r.Context(), target, refID); err == nil && label != "" {
				val = label
				link = "/" + target + "/" + refID
			}
		}
		fields = append(fields, ui.DetailField{Name: f.Name, Value: val, Link: link})
	}

	role := h.role(r)
	childLists := h.childLists(r.Context(), machine, recordID)
	if err := ui.Detail(role, machine, rec, fields, h.interp.PermittedEvents(machineID, role), childLists, h.unreadCount(r.Context(), role)).Render(r.Context(), w); err != nil {
		slog.Error("render detail", "error", err)
	}
}

// TriggerEvent — handle event button.
func (h *Handler) TriggerEvent(w http.ResponseWriter, r *http.Request) {
	machineID := chi.URLParam(r, "machineID")
	recordID := chi.URLParam(r, "recordID")
	eventID := chi.URLParam(r, "eventID")

	machine, ok := h.interp.GetMachine(machineID)
	if !ok {
		http.NotFound(w, r)
		return
	}
	role := h.role(r)
	if !h.guard.CanTrigger(machine, role, eventID) {
		http.Error(w, "not permitted", http.StatusForbidden)
		return
	}
	event, ok := h.interp.GetEvent(machineID, eventID)
	if !ok {
		http.NotFound(w, r)
		return
	}
	rec, err := h.records.Get(r.Context(), recordID)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	if err := h.triggerEvent(r.Context(), machine, event, rec, role); err != nil {
		var rv *ruleViolation
		if errors.As(err, &rv) {
			http.Error(w, rv.Error(), http.StatusBadRequest)
		} else {
			http.Error(w, "event failed", http.StatusInternalServerError)
		}
		return
	}
	http.Redirect(w, r, "/"+machineID+"/"+recordID, http.StatusSeeOther)
}

// ruleViolation marks a rejection as a business-rule failure (400), as
// opposed to an infrastructure error (500) — the same distinction Create
// already draws by rendering violations back into the form instead of
// erroring out.
type ruleViolation struct{ msg string }

func (e *ruleViolation) Error() string { return e.msg }

// triggerEvent is the single path every event trigger runs through, whether
// from an HTTP button (TriggerEvent) or fired internally by another event's
// aggregate_status action (CAP-A08, doAggregateStatus) — same guards, same
// validation, same persistence, so a system-triggered transition can never
// skip a check a user-triggered one would have to pass.
func (h *Handler) triggerEvent(ctx context.Context, machine *model.Machine, event *model.Event, rec *store.Record, actorRole string) error {
	// CAP-E06 — state guard: the event may only fire when the record's
	// CURRENT data satisfies its condition (e.g. Reject only from Submitted).
	if event.Condition != nil && !constraint.Eval(*event.Condition, rec.Data) {
		return &ruleViolation{fmt.Sprintf("%s is not allowed in the record's current state", event.Name)}
	}

	// CAP-A07 — sequential step guard: in a Sequential-mode parent, a step may
	// only be decided once every sibling with a lower Sequence has already
	// left Pending. Cross-record, so it can't be expressed as an event
	// Condition (CAP-E06 only reads the record's own data).
	if msg := h.sequentialGuardViolation(ctx, machine, event, rec); msg != "" {
		return &ruleViolation{msg}
	}

	// CAP-C09 — constraints evaluated on event trigger, not just Create:
	// simulate the event's effect first, validate the result, only persist
	// if it still satisfies every declared Constraint. CAP-A02 — actorRole
	// resolves this event's "current_user" dynamic values, if any.
	newData := h.exec.Simulate(event, rec, actorRole)
	if violations := h.engine.Violations(machine, newData); len(violations) > 0 {
		return &ruleViolation{strings.Join(violations, " ")}
	}

	if err := h.exec.Persist(ctx, event, rec, newData, machine.Name); err != nil {
		return err
	}

	// CAP-A07/CAP-A08 — workflow actions run after a successful commit, using
	// the now-current data: activate_next notifies the next pending sibling,
	// aggregate_status may internally trigger a rollup event on the parent.
	h.runWorkflowActions(ctx, machine, event, rec.ID, newData)
	return nil
}

// --- helpers -----------------------------------------------------------------

func fieldIndex(m *model.Machine) map[string]*model.Field {
	out := make(map[string]*model.Field, len(m.Fields))
	for _, f := range m.Fields {
		out[f.ID] = f
	}
	return out
}

func (h *Handler) buildFormFields(ctx context.Context, machine *model.Machine, vals map[string]any) []ui.FormField {
	view := h.interp.FormView(machine.ID)
	fieldByID := fieldIndex(machine)

	var fieldIDs []string
	if view != nil {
		fieldIDs = view.Config.Fields
	}

	fields := make([]ui.FormField, 0, len(fieldIDs))
	for _, id := range fieldIDs {
		f, ok := fieldByID[id]
		if !ok {
			continue
		}
		val := ""
		if vals != nil {
			if v, ok := vals[id]; ok {
				val = fmt.Sprintf("%v", v)
			}
		}
		var opts []ui.ReferenceOption
		if f.Type == model.FieldTypeReference {
			opts = h.referenceOptions(ctx, f.Options.TargetMachine)
		}
		fields = append(fields, ui.FormField{Field: f, Value: val, Options: opts})
	}
	return fields
}

// childLists finds every Machine with a `reference` field pointing at
// machine, and lists the records where that field equals recordID (CAP-V06).
// Generic by construction — it doesn't special-case Employee/Manager, so any
// future reference relationship gets a sub-list automatically.
func (h *Handler) childLists(ctx context.Context, machine *model.Machine, recordID string) []ui.ChildList {
	var out []ui.ChildList
	for _, m := range h.interp.AllMachines() {
		for _, f := range m.Fields {
			if f.Type != model.FieldTypeReference || f.Options.TargetMachine != machine.ID {
				continue
			}
			records, err := h.records.List(ctx, m.ID)
			if err != nil {
				slog.Error("list child records", "machine", m.ID, "error", err)
				continue
			}
			var items []ui.ChildListItem
			for _, rec := range records {
				v, ok := rec.Data[f.ID]
				if !ok {
					continue
				}
				if refID, _ := v.(string); refID == recordID {
					items = append(items, ui.ChildListItem{
						Label: displayLabel(m, rec.Data),
						Link:  "/" + m.ID + "/" + rec.ID,
					})
				}
			}
			if len(items) > 0 {
				out = append(out, ui.ChildList{Title: fmt.Sprintf("%s (via %s)", m.Name, f.Name), Items: items})
			}
		}
	}
	return out
}

func (h *Handler) referenceOptions(ctx context.Context, targetMachineID string) []ui.ReferenceOption {
	records, err := h.records.List(ctx, targetMachineID)
	if err != nil {
		slog.Error("list reference options", "target_machine", targetMachineID, "error", err)
		return nil
	}
	targetMachine, _ := h.interp.GetMachine(targetMachineID)
	opts := make([]ui.ReferenceOption, 0, len(records))
	for _, rec := range records {
		opts = append(opts, ui.ReferenceOption{ID: rec.ID, Label: displayLabel(targetMachine, rec.Data)})
	}
	return opts
}

// referenceLabel resolves one record's display label, for rendering an
// already-set reference value (detail/list views).
func (h *Handler) referenceLabel(ctx context.Context, targetMachineID, recordID string) (string, error) {
	rec, err := h.records.Get(ctx, recordID)
	if err != nil {
		return "", err
	}
	targetMachine, _ := h.interp.GetMachine(targetMachineID)
	return displayLabel(targetMachine, rec.Data), nil
}

// displayLabel picks a human-readable stand-in for a record: the target
// Machine's `text` field named "Name" if one exists, else its first `text`
// field, falling back to the record id. Menata Language doesn't (yet) let a
// business author declare "this is the field people should see when
// referencing a record" — this heuristic is a prototype stand-in for that
// missing capability, not a final design.
func displayLabel(machine *model.Machine, data map[string]any) string {
	if machine != nil {
		var firstText *model.Field
		for _, f := range machine.Fields {
			if f.Type != model.FieldTypeText {
				continue
			}
			if firstText == nil {
				firstText = f
			}
			if strings.EqualFold(f.Name, "name") {
				firstText = f
				break
			}
		}
		if firstText != nil {
			if v, ok := data[firstText.ID]; ok {
				if s := fmt.Sprintf("%v", v); s != "" {
					return s
				}
			}
		}
	}
	if id, ok := data["id"]; ok {
		return fmt.Sprintf("%v", id)
	}
	return ""
}

// referenceViolations enforces CAP-F13 referential integrity: a `reference`
// field's value must resolve to a real record on the target Machine. This is
// intrinsic to the field type, the same way `required` is intrinsic to a
// Field's Required flag — not a declared Constraint row.
func (h *Handler) referenceViolations(ctx context.Context, machine *model.Machine, data map[string]any) ([]string, error) {
	var out []string
	for _, f := range machine.Fields {
		if f.Type != model.FieldTypeReference {
			continue
		}
		v, ok := data[f.ID]
		if !ok {
			continue
		}
		recordID, _ := v.(string)
		if recordID == "" {
			continue
		}
		exists, err := h.records.Exists(ctx, f.Options.TargetMachine, recordID)
		if err != nil {
			return nil, err
		}
		if !exists {
			target, _ := h.interp.GetMachine(f.Options.TargetMachine)
			targetName := f.Options.TargetMachine
			if target != nil {
				targetName = target.Name
			}
			out = append(out, fmt.Sprintf("%s does not reference an existing %s record.", f.Name, targetName))
		}
	}
	return out, nil
}

// --- CAP-A07 / CAP-A08 workflow orchestration --------------------------------
//
// Both capabilities need the same handful of lookups on the child Machine
// (Approval Step, in Case 3's proof): which Field references the parent
// record, which Field holds the step's own Sequence, which Field holds its
// Decision. None of these are named in the action's own params — Menata
// Language doesn't have a way for a business author to name "the field that
// scopes this" beyond writing an ordinary `reference` Field — so, like
// displayLabel, this resolves them by name/type heuristic (a `reference`
// Field pointing at the parent Machine; a Field literally named "Sequence" or
// "Decision", case-insensitive). Prototype-honest, not a final design.

func findFieldByName(machine *model.Machine, name string) *model.Field {
	for _, f := range machine.Fields {
		if strings.EqualFold(f.Name, name) {
			return f
		}
	}
	return nil
}

func findFieldByID(machine *model.Machine, id string) *model.Field {
	for _, f := range machine.Fields {
		if f.ID == id {
			return f
		}
	}
	return nil
}

func findReferenceFieldTo(machine *model.Machine, targetMachineID string) *model.Field {
	for _, f := range machine.Fields {
		if f.Type == model.FieldTypeReference && f.Options.TargetMachine == targetMachineID {
			return f
		}
	}
	return nil
}

func toFloat(v any) float64 {
	s := fmt.Sprintf("%v", v)
	f, _ := strconv.ParseFloat(s, 64)
	return f
}

// sequentialGuardViolation implements CAP-A07's hard block: in Sequential
// mode, a step may only be Approved or Rejected once every sibling with a
// lower Sequence has already left Pending. Applies to any event carrying an
// aggregate_status action — both evt_as_approve and evt_as_reject declare
// one, so both are gated uniformly, even though only Approve also declares
// activate_next. Returns "" when the event isn't part of a sequential
// workflow at all (no aggregate_status action, no config, or Parallel mode).
func (h *Handler) sequentialGuardViolation(ctx context.Context, machine *model.Machine, event *model.Event, rec *store.Record) string {
	var parentFieldID string
	for _, a := range event.Actions {
		if a.Type == model.ActionAggregateStatus {
			parentFieldID, _ = a.Params["parent_field"].(string)
			break
		}
	}
	if parentFieldID == "" {
		return ""
	}
	parentField := findFieldByID(machine, parentFieldID)
	if parentField == nil || parentField.Type != model.FieldTypeReference {
		return ""
	}
	parentMachine, ok := h.interp.GetMachine(parentField.Options.TargetMachine)
	if !ok || parentMachine.Config == nil {
		return ""
	}
	modeFieldID := parentMachine.Config["approval_mode_field"]
	if modeFieldID == "" {
		return ""
	}
	parentID, _ := rec.Data[parentFieldID].(string)
	if parentID == "" {
		return ""
	}
	parentRec, err := h.records.Get(ctx, parentID)
	if err != nil {
		return ""
	}
	if fmt.Sprintf("%v", parentRec.Data[modeFieldID]) != "Sequential" {
		return ""
	}

	seqField := findFieldByName(machine, "Sequence")
	decisionField := findFieldByName(machine, "Decision")
	if seqField == nil || decisionField == nil {
		return ""
	}
	mySeq := toFloat(rec.Data[seqField.ID])

	siblings, err := h.records.List(ctx, machine.ID)
	if err != nil {
		return ""
	}
	for _, sib := range siblings {
		if sib.ID == rec.ID {
			continue
		}
		if sp, _ := sib.Data[parentFieldID].(string); sp != parentID {
			continue
		}
		if fmt.Sprintf("%v", sib.Data[decisionField.ID]) != "Pending" {
			continue
		}
		if toFloat(sib.Data[seqField.ID]) < mySeq {
			return fmt.Sprintf("%s is not allowed yet — an earlier step is still Pending", event.Name)
		}
	}
	return ""
}

// runWorkflowActions handles the two action types Executor.Persist doesn't
// touch (CAP-A07/A08 need cross-record lookups Executor has no access to —
// this stays at the Handler layer, same reasoning as childLists/
// referenceViolations).
func (h *Handler) runWorkflowActions(ctx context.Context, machine *model.Machine, event *model.Event, recordID string, data map[string]any) {
	for _, a := range event.Actions {
		switch a.Type {
		case model.ActionActivateNext:
			h.doActivateNext(ctx, machine, a.Params, recordID, data)
		case model.ActionAggregateStatus:
			h.doAggregateStatus(ctx, machine, a.Params, data)
		}
	}
}

// doActivateNext (CAP-A07): in Sequential mode, notifies the next still-
// Pending sibling's Approver that it's their turn. Doesn't write anything —
// the sequential guard above is what actually enforces order; this is purely
// the hand-off notification (logged only, same as every other `notify` in
// this prototype).
func (h *Handler) doActivateNext(ctx context.Context, machine *model.Machine, params map[string]any, recordID string, data map[string]any) {
	modeFieldID, _ := params["mode_field"].(string)
	if modeFieldID == "" {
		return
	}
	parentMachine := findMachineContainingField(h.interp, modeFieldID)
	if parentMachine == nil || parentMachine.Config == nil {
		return
	}
	parentRefField := findReferenceFieldTo(machine, parentMachine.ID)
	if parentRefField == nil {
		return
	}
	parentID, _ := data[parentRefField.ID].(string)
	if parentID == "" {
		return
	}
	parentRec, err := h.records.Get(ctx, parentID)
	if err != nil || fmt.Sprintf("%v", parentRec.Data[modeFieldID]) != "Sequential" {
		return
	}

	seqField := findFieldByName(machine, "Sequence")
	decisionField := findFieldByName(machine, "Decision")
	approverField := findFieldByName(machine, "Approver")
	if seqField == nil || decisionField == nil {
		return
	}
	mySeq := toFloat(data[seqField.ID])

	siblings, err := h.records.List(ctx, machine.ID)
	if err != nil {
		return
	}
	var next *store.Record
	var nextSeq float64
	for _, sib := range siblings {
		if sib.ID == recordID {
			continue
		}
		if sp, _ := sib.Data[parentRefField.ID].(string); sp != parentID {
			continue
		}
		if fmt.Sprintf("%v", sib.Data[decisionField.ID]) != "Pending" {
			continue
		}
		s := toFloat(sib.Data[seqField.ID])
		if s > mySeq && (next == nil || s < nextSeq) {
			next, nextSeq = sib, s
		}
	}
	if next == nil {
		return
	}
	approver := ""
	if approverField != nil {
		approver = fmt.Sprintf("%v", next.Data[approverField.ID])
	}
	slog.Info("activate_next: next step ready (prototype: notify logged only)",
		"machine", machine.ID, "next_record", next.ID, "sequence", nextSeq, "approver", approver)
}

// doAggregateStatus (CAP-A08): rolls sibling steps' Decision up to the parent
// — any Rejected cascades the parent to its "any rejected" event immediately
// (doesn't wait for the rest to decide); only once every sibling is Approved
// does the parent reach its "all approved" event. Internally triggers the
// resolved parent event through the same triggerEvent path an HTTP request
// would use (CAP-E06/CAP-C09 guards still apply), as "System" — the acting
// role recorded for this rollup's dynamic values, matching approval-
// document.yaml's own declared System permission role.
func (h *Handler) doAggregateStatus(ctx context.Context, machine *model.Machine, params map[string]any, data map[string]any) {
	parentFieldID, _ := params["parent_field"].(string)
	allApprovedEvt, _ := params["parent_event_if_all_approved"].(string)
	anyRejectedEvt, _ := params["parent_event_if_any_rejected"].(string)
	if parentFieldID == "" {
		return
	}
	parentField := findFieldByID(machine, parentFieldID)
	if parentField == nil || parentField.Type != model.FieldTypeReference {
		return
	}
	parentMachine, ok := h.interp.GetMachine(parentField.Options.TargetMachine)
	if !ok {
		return
	}
	parentID, _ := data[parentFieldID].(string)
	if parentID == "" {
		return
	}

	decisionField := findFieldByName(machine, "Decision")
	if decisionField == nil {
		return
	}
	siblings, err := h.records.List(ctx, machine.ID)
	if err != nil {
		return
	}

	found, allApproved, anyRejected := false, true, false
	for _, sib := range siblings {
		if sp, _ := sib.Data[parentFieldID].(string); sp != parentID {
			continue
		}
		found = true
		switch fmt.Sprintf("%v", sib.Data[decisionField.ID]) {
		case "Rejected":
			anyRejected = true
		case "Approved":
			// stays eligible for allApproved
		default:
			allApproved = false
		}
	}
	if !found {
		return
	}

	var targetEventID string
	switch {
	case anyRejected && anyRejectedEvt != "":
		targetEventID = anyRejectedEvt
	case allApproved && allApprovedEvt != "":
		targetEventID = allApprovedEvt
	default:
		return
	}

	targetEvent, ok := h.interp.GetEvent(parentMachine.ID, targetEventID)
	if !ok {
		return
	}
	parentRec, err := h.records.Get(ctx, parentID)
	if err != nil {
		return
	}
	if err := h.triggerEvent(ctx, parentMachine, targetEvent, parentRec, "System"); err != nil {
		slog.Error("aggregate_status: failed to trigger parent event",
			"parent_machine", parentMachine.ID, "parent_record", parentID, "event", targetEventID, "error", err)
	}
}

func findMachineContainingField(interp *interpreter.Interpreter, fieldID string) *model.Machine {
	for _, m := range interp.AllMachines() {
		if findFieldByID(m, fieldID) != nil {
			return m
		}
	}
	return nil
}

// --- CAP-A10 in-app notification inbox ---------------------------------------

func (h *Handler) unreadCount(ctx context.Context, role string) int {
	n, err := h.notifications.UnreadCount(ctx, role)
	if err != nil {
		slog.Error("count unread notifications", "error", err)
		return 0
	}
	return n
}

// Notifications — the current session's in-app notification inbox: every
// `notify` action (CAP-A03/A04) whose resolved recipient matches this
// session's role cookie, the same identity-is-role caveat CAP-A02 already
// carries.
func (h *Handler) Notifications(w http.ResponseWriter, r *http.Request) {
	role := h.role(r)
	notifs, err := h.notifications.ListForRecipient(r.Context(), role)
	if err != nil {
		http.Error(w, "failed to load notifications", http.StatusInternalServerError)
		return
	}
	items := make([]ui.NotificationItem, len(notifs))
	for i, n := range notifs {
		link := ""
		if n.MachineID != "" && n.RecordID != "" {
			link = "/" + n.MachineID + "/" + n.RecordID
		}
		items[i] = ui.NotificationItem{
			ID:      n.ID,
			Message: n.Message,
			Link:    link,
			Unread:  n.ReadAt == nil,
			When:    n.CreatedAt.Format("2006-01-02 15:04"),
		}
	}
	if err := ui.Notifications(role, items, h.unreadCount(r.Context(), role)).Render(r.Context(), w); err != nil {
		slog.Error("render notifications", "error", err)
	}
}

// MarkNotificationRead — POST target for a single notification's "Mark read"
// button. Scoped to the current session's role in the store layer (a role
// can only mark its own notifications read), the same access-control shape
// as every other role-gated action in this prototype.
func (h *Handler) MarkNotificationRead(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	role := h.role(r)
	if err := h.notifications.MarkRead(r.Context(), id, role); err != nil {
		http.Error(w, "failed to mark notification read", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/notifications", http.StatusSeeOther)
}
