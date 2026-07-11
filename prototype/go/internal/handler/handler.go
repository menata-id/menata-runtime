package handler

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
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
	interp  *interpreter.Interpreter
	records *store.RecordStore
	engine  *constraint.Engine
	guard   *permission.Guard
	exec    *executor.Executor
}

func New(interp *interpreter.Interpreter, records *store.RecordStore) *Handler {
	return &Handler{
		interp:  interp,
		records: records,
		engine:  &constraint.Engine{},
		guard:   &permission.Guard{},
		exec:    executor.New(records),
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
	if err := ui.Home(h.role(r), cards).Render(r.Context(), w); err != nil {
		slog.Error("render home", "error", err)
	}
}

// LoginForm — role selection page.
func (h *Handler) LoginForm(w http.ResponseWriter, r *http.Request) {
	if err := ui.LoginPage(h.role(r)).Render(r.Context(), w); err != nil {
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
	if err := ui.List(role, machine, cols, rows, h.interp.PermittedEvents(machineID, role)).Render(r.Context(), w); err != nil {
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
	if err := ui.Form(h.role(r), machine, h.buildFormFields(r.Context(), machine, nil), nil).Render(r.Context(), w); err != nil {
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

	data := make(map[string]any)
	for _, f := range machine.Fields {
		if strings.ToLower(f.Name) == "status" && f.Type == model.FieldTypeValueList && len(f.Options.Values) > 0 {
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
		if err := ui.Form(h.role(r), machine, h.buildFormFields(r.Context(), machine, data), violations).Render(r.Context(), w); err != nil {
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
	if err := ui.Detail(role, machine, rec, fields, h.interp.PermittedEvents(machineID, role)).Render(r.Context(), w); err != nil {
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
	if err := h.exec.Apply(r.Context(), event, rec); err != nil {
		http.Error(w, "event failed", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/"+machineID+"/"+recordID, http.StatusSeeOther)
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

// referenceOptions lists a target Machine's records as picker choices.
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
