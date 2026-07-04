package handler

import (
	"fmt"
	"html/template"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"menata.id/runtime/internal/constraint"
	"menata.id/runtime/internal/executor"
	"menata.id/runtime/internal/interpreter"
	"menata.id/runtime/internal/model"
	"menata.id/runtime/internal/permission"
	"menata.id/runtime/internal/store"
)

type Handler struct {
	interp  *interpreter.Interpreter
	records *store.RecordStore
	engine  *constraint.Engine
	guard   *permission.Guard
	exec    *executor.Executor
	tmpl    *template.Template
}

func New(interp *interpreter.Interpreter, records *store.RecordStore) *Handler {
	h := &Handler{
		interp:  interp,
		records: records,
		engine:  &constraint.Engine{},
		guard:   &permission.Guard{},
		exec:    executor.New(records),
	}
	h.tmpl = template.Must(template.New("").Funcs(template.FuncMap{
		"fieldValue": func(data map[string]any, fieldID string) string {
			v := data[fieldID]
			if v == nil {
				return ""
			}
			return fmt.Sprintf("%v", v)
		},
	}).Parse(htmlTemplates))
	return h
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
	h.render(w, "home", map[string]any{
		"Role":     h.role(r),
		"Machines": h.interp.AllMachines(),
	})
}

// LoginForm — role selection page.
func (h *Handler) LoginForm(w http.ResponseWriter, r *http.Request) {
	h.render(w, "login", map[string]any{"Role": h.role(r)})
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

// --- list view ---------------------------------------------------------------

type listData struct {
	Role            string
	Machine         *model.Machine
	Columns         []columnDef
	Rows            []listRow
	PermittedEvents []*model.Event
}

type columnDef struct{ ID, Name string }

type listRow struct {
	ID     string
	Values []string
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	machineID := chi.URLParam(r, "machineID")
	machine, ok := h.interp.GetMachine(machineID)
	if !ok {
		http.NotFound(w, r)
		return
	}
	view := h.interp.DefaultListView(machineID)

	// Resolve column definitions from view config
	fieldByID := make(map[string]*model.Field)
	for _, f := range machine.Fields {
		fieldByID[f.ID] = f
	}
	colIDs := []string{}
	if view != nil {
		colIDs = view.Config.Columns
	}
	cols := make([]columnDef, 0, len(colIDs))
	for _, id := range colIDs {
		name := id
		if f, ok := fieldByID[id]; ok {
			name = f.Name
		}
		cols = append(cols, columnDef{ID: id, Name: name})
	}

	records, err := h.records.List(r.Context(), machineID)
	if err != nil {
		http.Error(w, "failed to load records", http.StatusInternalServerError)
		return
	}

	rows := make([]listRow, 0, len(records))
	for _, rec := range records {
		vals := make([]string, len(colIDs))
		for j, id := range colIDs {
			if v, ok := rec.Data[id]; ok {
				vals[j] = fmt.Sprintf("%v", v)
			}
		}
		rows = append(rows, listRow{ID: rec.ID, Values: vals})
	}

	role := h.role(r)
	h.render(w, "list", listData{
		Role:            role,
		Machine:         machine,
		Columns:         cols,
		Rows:            rows,
		PermittedEvents: h.interp.PermittedEvents(machineID, role),
	})
}

// --- form view ---------------------------------------------------------------

type formField struct {
	Field *model.Field
	Value string
}

type formData struct {
	Role    string
	Machine *model.Machine
	Fields  []formField
	Errors  []string
}

func (h *Handler) NewForm(w http.ResponseWriter, r *http.Request) {
	machineID := chi.URLParam(r, "machineID")
	machine, ok := h.interp.GetMachine(machineID)
	if !ok {
		http.NotFound(w, r)
		return
	}
	h.render(w, "form", h.buildFormData(machine, nil, h.role(r), nil))
}

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
	// seed default status from first value in the status field
	for _, f := range machine.Fields {
		if strings.ToLower(f.Name) == "status" && f.Type == model.FieldTypeValueList && len(f.Options.Values) > 0 {
			data[f.ID] = f.Options.Values[0]
		}
	}
	// overlay form values
	for _, f := range machine.Fields {
		if v := r.FormValue(f.ID); v != "" {
			data[f.ID] = v
		}
	}

	if violations := h.engine.Violations(machine, data); len(violations) > 0 {
		h.render(w, "form", h.buildFormData(machine, data, h.role(r), violations))
		return
	}

	rec, err := h.records.Create(r.Context(), machineID, data)
	if err != nil {
		http.Error(w, "failed to create record", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/"+machineID+"/"+rec.ID, http.StatusSeeOther)
}

func (h *Handler) buildFormData(machine *model.Machine, vals map[string]any, role string, errors []string) formData {
	view := h.interp.FormView(machine.ID)
	fieldIDs := []string{}
	if view != nil {
		fieldIDs = view.Config.Fields
	}
	fieldByID := make(map[string]*model.Field)
	for _, f := range machine.Fields {
		fieldByID[f.ID] = f
	}

	fields := make([]formField, 0, len(fieldIDs))
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
		fields = append(fields, formField{Field: f, Value: val})
	}
	return formData{Role: role, Machine: machine, Fields: fields, Errors: errors}
}

// --- detail view -------------------------------------------------------------

type detailField struct{ Name, Value string }

type detailData struct {
	Role            string
	Machine         *model.Machine
	Record          *store.Record
	Fields          []detailField
	PermittedEvents []*model.Event
}

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

	fields := make([]detailField, 0, len(machine.Fields))
	for _, f := range machine.Fields {
		val := ""
		if v, ok := rec.Data[f.ID]; ok {
			val = fmt.Sprintf("%v", v)
		}
		fields = append(fields, detailField{Name: f.Name, Value: val})
	}

	role := h.role(r)
	h.render(w, "detail", detailData{
		Role:            role,
		Machine:         machine,
		Record:          rec,
		Fields:          fields,
		PermittedEvents: h.interp.PermittedEvents(machineID, role),
	})
}

// --- event trigger -----------------------------------------------------------

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

func (h *Handler) render(w http.ResponseWriter, name string, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.tmpl.ExecuteTemplate(w, name, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
