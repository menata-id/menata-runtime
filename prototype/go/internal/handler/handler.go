package handler

import (
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
			cells[j] = ui.ListCell{
				Value:         val,
				IsStatusBadge: cols[j].Type == model.FieldTypeValueList,
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
	if err := ui.Form(h.role(r), machine, buildFormFields(machine, h.interp, nil), nil).Render(r.Context(), w); err != nil {
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

	if violations := h.engine.Violations(machine, data); len(violations) > 0 {
		if err := ui.Form(h.role(r), machine, buildFormFields(machine, h.interp, data), violations).Render(r.Context(), w); err != nil {
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
		fields = append(fields, ui.DetailField{Name: f.Name, Value: val})
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

func buildFormFields(machine *model.Machine, interp *interpreter.Interpreter, vals map[string]any) []ui.FormField {
	view := interp.FormView(machine.ID)
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
		fields = append(fields, ui.FormField{Field: f, Value: val})
	}
	return fields
}
