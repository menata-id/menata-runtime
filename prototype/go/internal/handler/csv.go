package handler

import (
	"encoding/csv"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"menata.id/runtime/internal/model"
	"menata.id/runtime/internal/ui"
)

// csvFieldIDs (CAP-R06) returns the field ids export/import agree on --
// the FormView's own Fields, the same set Create/Update read from, kept
// symmetric so an exported CSV re-imports cleanly. Header row uses field
// ids, not display Names -- unambiguous and machine-inspectable, the same
// "ids are the interchange format" choice CAP-F05 already made.
func (h *Handler) csvFieldIDs(machine *model.Machine) []string {
	if fv := h.interp.Get().FormView(machine.ID); fv != nil {
		return fv.Config.Fields
	}
	return nil
}

// ExportCSV (CAP-R06) streams every live (non-archived) record on
// machineID as CSV -- CanRead-gated, same tier as viewing the list itself.
func (h *Handler) ExportCSV(w http.ResponseWriter, r *http.Request) {
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
	records, err := h.records.List(r.Context(), machineID, "", "")
	if err != nil {
		http.Error(w, "failed to load records", http.StatusInternalServerError)
		return
	}
	fieldIDs := h.csvFieldIDs(machine)

	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", `attachment; filename="`+machineID+`.csv"`)
	cw := csv.NewWriter(w)
	if err := cw.Write(fieldIDs); err != nil {
		return
	}
	for _, rec := range records {
		row := make([]string, len(fieldIDs))
		for i, id := range fieldIDs {
			if v, ok := rec.Data[id]; ok {
				row[i] = fmt.Sprintf("%v", v)
			}
		}
		if err := cw.Write(row); err != nil {
			return
		}
	}
	cw.Flush()
}

// ImportCSVForm (CAP-R06) renders the CSV upload page.
func (h *Handler) ImportCSVForm(w http.ResponseWriter, r *http.Request) {
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
	a := h.auth(r)
	page := ui.ImportCSV(a.User.Name, a.CSRFToken, h.isWorkspaceAdmin(r), machine, h.csvFieldIDs(machine), nil, h.unreadCount(r.Context(), a), h.subNavFor(r, machine))
	if err := page.Render(r.Context(), w); err != nil {
		slog.Error("render import form", "error", err)
	}
}

// ImportCSV (CAP-R06) bulk-creates records from an uploaded CSV, one HTTP
// request. Each row is INDEPENDENT -- unlike CAP-F16's atomic parent+child
// create, one bad row is reported and skipped, not a reason to reject every
// other row in the file. Every row goes through the exact same validation
// pipeline Create uses (Violations, referenceViolations,
// userReferenceViolations, uniquenessViolations, including CAP-R08's
// scratch-state exemption) -- CSV import is not a side door around
// ordinary Create rules. Rows are created one at a time, in file order, so
// a uniqueness check on row N correctly sees rows already imported earlier
// in the same file, not just what existed in the database beforehand.
func (h *Handler) ImportCSV(w http.ResponseWriter, r *http.Request) {
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
	if err := r.ParseMultipartForm(5 << 20); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	file, _, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "no file uploaded", http.StatusBadRequest)
		return
	}
	defer file.Close()

	cr := csv.NewReader(file)
	header, err := cr.Read()
	if err != nil {
		http.Error(w, "empty or malformed CSV", http.StatusBadRequest)
		return
	}
	fieldByID := fieldIndex(machine)
	headerFields := map[string]bool{}
	for _, id := range header {
		headerFields[id] = true
	}

	var results []ui.ImportRowResult
	rowNum := 1
	for {
		record, err := cr.Read()
		if err != nil {
			break
		}
		rowNum++

		data := make(map[string]any)
		for _, f := range machine.Fields {
			if !headerFields[f.ID] && f.Type == model.FieldTypeValueList && len(f.Options.Values) > 0 {
				data[f.ID] = f.Options.Values[0]
			}
		}
		for i, id := range header {
			if i < len(record) && record[i] != "" {
				if _, ok := fieldByID[id]; ok {
					data[id] = record[i]
				}
			}
		}

		var violations []string
		if !h.inScratchState(machine, data) {
			violations = h.engine.Violations(machine, withChangePolicyCreatedAt(machine, data, time.Now()))
		}
		if refV, err := h.referenceViolations(r.Context(), machine, data); err == nil {
			violations = append(violations, refV...)
		}
		if userV, err := h.userReferenceViolations(r.Context(), machine, data); err == nil {
			violations = append(violations, userV...)
		}
		if uniqueV, err := h.uniquenessViolations(r.Context(), machine, data, ""); err == nil {
			violations = append(violations, uniqueV...)
		}

		if len(violations) > 0 {
			results = append(results, ui.ImportRowResult{Row: rowNum, Success: false, Message: strings.Join(violations, "; ")})
			continue
		}
		rec, err := h.records.Create(r.Context(), machineID, workspaceID, data)
		if err != nil {
			results = append(results, ui.ImportRowResult{Row: rowNum, Success: false, Message: "failed to create record"})
			continue
		}
		results = append(results, ui.ImportRowResult{Row: rowNum, Success: true, Message: rec.ID})
	}

	a := h.auth(r)
	page := ui.ImportCSV(a.User.Name, a.CSRFToken, h.isWorkspaceAdmin(r), machine, h.csvFieldIDs(machine), results, h.unreadCount(r.Context(), a), h.subNavFor(r, machine))
	if err := page.Render(r.Context(), w); err != nil {
		slog.Error("render import results", "error", err)
	}
}
