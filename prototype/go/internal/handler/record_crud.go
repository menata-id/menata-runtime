package handler

import (
	"fmt"
	htmltemplate "html/template"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"menata.id/runtime/internal/constraint"
	"menata.id/runtime/internal/model"
	"menata.id/runtime/internal/store"
	"menata.id/runtime/internal/ui"
)

// List — list view of records for a machine.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	machineID := chi.URLParam(r, "machineID")
	machine, ok := h.interp.Get().GetMachine(machineID)
	if !ok {
		http.NotFound(w, r)
		return
	}
	workspaceID, applicationID := h.interp.Get().ScopeFor(machineID)
	if workspaceID != h.workspace(r) {
		// CAP-X06: a Machine from another Workspace 404s exactly like one
		// that doesn't exist at all -- app-layer guard alongside RLS
		// (migrations/009), not instead of it.
		http.NotFound(w, r)
		return
	}
	role := h.roleForApp(r, applicationID)
	// CAP-P05 — deny-by-default: no permission row for this role on this
	// machine means no read access, not implicitly allowed.
	if !h.guard.CanRead(machine, role) {
		h.logPermissionDenied(r.Context(), "read", machineID, "", role, h.identity(r))
		http.Error(w, "not permitted", http.StatusForbidden)
		return
	}

	view := h.interp.Get().DefaultListView(machineID)
	fieldByID := fieldIndex(machine)

	// CAP-P06: field-level visibility -- a column this role's Permission
	// hides never reaches the List page at all, not just hidden client-side.
	hidden := h.hiddenFields(machine, role)
	colIDs := []string{}
	if view != nil {
		for _, id := range view.Config.Columns {
			if !hidden[id] {
				colIDs = append(colIDs, id)
			}
		}
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

	// CAP-R03: ?archived=1 shows the archive itself (ListArchived) instead
	// of the live list -- the one place a soft-deleted record can still be
	// found and restored. Sort/filter/search/pagination below all apply the
	// same way to either set.
	archived := r.URL.Query().Get("archived") == "1"
	var records []*store.Record
	var err error
	if archived {
		records, err = h.records.ListArchived(r.Context(), machineID)
	} else {
		sortField, sortDir := "", ""
		if view != nil && view.Config.ManualOrder {
			// CAP-V14 wins over DefaultSort when both are declared on the same
			// View -- manual order only means anything if it's what's actually
			// shown.
			sortField = store.SortOrderField
		} else if view != nil && view.Config.DefaultSort != nil {
			sortField, sortDir = view.Config.DefaultSort.Field, view.Config.DefaultSort.Direction
		}
		records, err = h.records.List(r.Context(), machineID, sortField, sortDir)
	}
	if err != nil {
		http.Error(w, "failed to load records", http.StatusInternalServerError)
		return
	}

	// CAP-V05/V09: a list View's declarative filter, AND-combined, reusing
	// constraint.Eval's own expression grammar. $current_user (CAP-V05) is
	// resolved to the acting identity's id here, request-time, before Eval
	// ever sees it -- Eval itself has no notion of "who's asking."
	if view != nil && len(view.Config.Filter) > 0 {
		identityID := h.identityID(r)
		kept := records[:0]
		for _, rec := range records {
			match := true
			for _, fc := range view.Config.Filter {
				val := fc.Value
				if val == "$current_user" {
					val = identityID
				}
				if !constraint.Eval(model.ConstraintExpression{Field: fc.Field, Operator: fc.Operator, Value: val}, rec.Data) {
					match = false
					break
				}
			}
			if match {
				kept = append(kept, rec)
			}
		}
		records = kept
	}

	// CAP-V08: free-text search across this View's visible columns, ?q=.
	// Substring, case-insensitive, HTTP black-box (a plain GET query param,
	// no JS) -- matches this prototype's no-SPA-framework posture.
	searchQuery := strings.TrimSpace(r.URL.Query().Get("q"))
	if searchQuery != "" {
		q := strings.ToLower(searchQuery)
		kept := records[:0]
		for _, rec := range records {
			match := false
			for _, id := range colIDs {
				if v, ok := rec.Data[id]; ok && strings.Contains(strings.ToLower(fmt.Sprintf("%v", v)), q) {
					match = true
					break
				}
			}
			if match {
				kept = append(kept, rec)
			}
		}
		records = kept
	}

	// CAP-R05: pagination applies AFTER filter/search, on the final matching
	// set, not as a SQL LIMIT/OFFSET before them -- otherwise a filter could
	// discard most of one SQL page and never see matching rows sitting on
	// the next one. In-memory slicing costs nothing extra at this
	// prototype's scale, the same tradeoff CAP-V08/V09's own in-memory
	// filtering already made.
	const pageSize = 25
	pageNum, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if pageNum < 1 {
		pageNum = 1
	}
	totalRecords := len(records)
	totalPages := (totalRecords + pageSize - 1) / pageSize
	if totalPages < 1 {
		totalPages = 1
	}
	if pageNum > totalPages {
		pageNum = totalPages
	}
	start := (pageNum - 1) * pageSize
	end := start + pageSize
	if start > totalRecords {
		start = totalRecords
	}
	if end > totalRecords {
		end = totalRecords
	}
	records = records[start:end]

	rows := make([]ui.ListRow, 0, len(records))
	for _, rec := range records {
		cells := make([]ui.ListCell, len(colIDs))
		for j, id := range colIDs {
			val := ""
			if v, ok := rec.Data[id]; ok {
				val = fmt.Sprintf("%v", v)
			}
			link := ""
			switch {
			case cols[j].Type == model.FieldTypeReference && val != "":
				refID := val
				target := fieldByID[id].Options.TargetMachine
				if label, err := h.referenceLabel(r.Context(), target, refID); err == nil && label != "" {
					val = label
					link = "/" + target + "/" + refID
				}
			case cols[j].Type == model.FieldTypeUser && val != "":
				if label, err := h.userLabel(r.Context(), val); err == nil && label != "" {
					val = label
				}
			case cols[j].Type == model.FieldTypeBoolean:
				val = boolLabel(val) // CAP-F09
			case cols[j].Type == model.FieldTypeMoney && val != "":
				val = formatMoney(val, fieldByID[id], rec.Data) // CAP-F08
			case cols[j].Type == model.FieldTypeFile && val != "":
				link = "/files/" + val // CAP-F06
			case cols[j].Type == model.FieldTypeComputed:
				val = computedValue(fieldByID[id], rec.Data) // CAP-F14
			}
			urgency := ""
			if view != nil && id == view.Config.SlaField {
				if label, u, ok := slaUrgency(val, view.Config.SlaWarningDays); ok { // CAP-V17
					val, urgency = label, u
				}
			}
			cells[j] = ui.ListCell{
				Value:         val,
				IsStatusBadge: cols[j].Type == model.FieldTypeValueList,
				Link:          link,
				SlaUrgency:    urgency,
			}
		}
		rows = append(rows, ui.ListRow{ID: rec.ID, Cells: cells})
	}

	opts := ui.ListViewOptions{
		SearchQuery: searchQuery,
		ManualOrder: view != nil && view.Config.ManualOrder,
		Archived:    archived,
		CanDelete:   h.guard.CanDelete(machine, role),
		Page:        pageNum,
		TotalPages:  totalPages,
	}
	a := h.auth(r)
	page := ui.List(h.workspaceName(r), a.User.Name, a.CSRFToken, h.isWorkspaceAdmin(r), machine, cols, rows, h.interp.Get().PermittedEvents(machineID, role), h.unreadCount(r.Context(), a), opts, h.subNavFor(r, machine), h.viewNavFor(machineID, model.ViewTypeList))
	if err := page.Render(r.Context(), w); err != nil {
		slog.Error("render list", "error", err)
	}
}

// Archive/Restore (CAP-R03) soft-delete/undelete a record. CanDelete-gated
// -- a distinct, opt-in permission tier from CanEdit (migrations/012's own
// note on why can_delete defaults false).
func (h *Handler) Archive(w http.ResponseWriter, r *http.Request) {
	h.setDeleted(w, r, true)
}

func (h *Handler) Restore(w http.ResponseWriter, r *http.Request) {
	h.setDeleted(w, r, false)
}

func (h *Handler) setDeleted(w http.ResponseWriter, r *http.Request, deleted bool) {
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
	if !h.guard.CanDelete(machine, role) {
		h.logPermissionDenied(r.Context(), "delete", machineID, "", role, h.identity(r))
		http.Error(w, "not permitted", http.StatusForbidden)
		return
	}
	// CAP-R07: an immutable record can't be archived either (not restored
	// -- undeleting isn't a business-data mutation), same gate Update uses
	// -- "frozen" means frozen against every mutation path, not just field
	// edits.
	if deleted {
		if rec, err := h.records.Get(r.Context(), recordID); err == nil && rec.MachineID == machineID {
			if reason := h.immutabilityViolation(machine, rec.Data); reason != "" {
				http.Error(w, reason, http.StatusForbidden)
				return
			}
			// CAP-O02: a master-data record (Machine.Config["master_data"])
			// can't be archived while any OTHER record, on ANY Machine
			// (cross-app by design), still references it -- archiving it
			// would silently break every one of those references. Reuses
			// CAP-V06's own childLists scan (every Machine's `reference`
			// fields targeting this one) rather than a second
			// implementation of "who points at me."
			if machine.Config["master_data"] == "true" {
				if refs := h.childLists(r.Context(), machine, recordID); len(refs) > 0 {
					http.Error(w, fmt.Sprintf("cannot archive: still referenced by %s", refs[0].Title), http.StatusConflict)
					return
				}
			}
		}
	}
	var err error
	if deleted {
		err = h.records.Archive(r.Context(), recordID)
	} else {
		err = h.records.Restore(r.Context(), recordID)
	}
	if err != nil {
		http.Error(w, "failed to update record", http.StatusInternalServerError)
		return
	}
	if deleted {
		http.Redirect(w, r, "/"+machineID, http.StatusSeeOther)
	} else {
		http.Redirect(w, r, "/"+machineID+"/"+recordID, http.StatusSeeOther)
	}
}

// MoveRecord (CAP-V14) reorders recordID up or down among its siblings on
// machineID's manual-order list View. CanEdit-gated -- reordering changes
// something about the record's presentation, the same permission tier as
// changing one of its fields, not a separate concept.
func (h *Handler) MoveRecord(w http.ResponseWriter, r *http.Request) {
	machineID := chi.URLParam(r, "machineID")
	recordID := chi.URLParam(r, "recordID")
	direction := chi.URLParam(r, "direction")
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
	if !h.guard.CanEdit(machine, role) {
		h.logPermissionDenied(r.Context(), "edit", machineID, "", role, h.identity(r))
		http.Error(w, "not permitted", http.StatusForbidden)
		return
	}
	if err := h.records.Move(r.Context(), machineID, recordID, direction); err != nil {
		http.Error(w, "failed to move record", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/"+machineID, http.StatusSeeOther)
}

// BoardMove (CAP-V14 Tier 2) handles a kanban card drop -- CanEdit-gated
// same as MoveRecord above, no constraint re-validation, the same "trusted
// same-record field write triggered by a UI action, not a business Event"
// posture MoveRecord itself already takes for sort_order. lane must be one
// of the GroupField's own declared value_list options -- rejected otherwise
// (this project's "Unknown = explicit" discipline), not written as an
// arbitrary string into the record's data.
func (h *Handler) BoardMove(w http.ResponseWriter, r *http.Request) {
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
	if !h.guard.CanEdit(machine, role) {
		h.logPermissionDenied(r.Context(), "edit", machineID, recordID, role, h.identity(r))
		http.Error(w, "not permitted", http.StatusForbidden)
		return
	}
	view := h.interp.Get().BoardView(machineID)
	if view == nil || view.Config.GroupField == "" {
		http.NotFound(w, r)
		return
	}
	groupField, ok := fieldIndex(machine)[view.Config.GroupField]
	if !ok {
		http.NotFound(w, r)
		return
	}
	lane := r.FormValue("lane")
	validLane := false
	for _, v := range groupField.Options.Values {
		if v == lane {
			validLane = true
			break
		}
	}
	if !validLane {
		http.Error(w, "invalid lane", http.StatusBadRequest)
		return
	}
	if err := h.records.MoveToLane(r.Context(), machineID, recordID, view.Config.GroupField, lane); err != nil {
		http.Error(w, "failed to move record", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/"+machineID+"/board", http.StatusSeeOther)
}

// Report renders a CAP-V13 aggregate report View -- grouped SUMs computed
// at render time from ANOTHER Machine's own records (view.Config.Report),
// not this Machine's. Read access is checked against the SOURCE machine
// (the data being aggregated), same reasoning as CAP-V06's reverse-lookup
// child lists reading the child Machine's own records.
// Document handles GET /{machineID}/{recordID}/document (CAP-F21) -- renders
// the Machine's own `document`-type View (Config.Template, an html/template
// source with {{.fld_x}} placeholders) against one record's Data. html/
// template auto-escapes every interpolated value, so a record whose data
// happens to contain HTML/script-looking text can't inject anything into
// the rendered page. Output is HTML, not a binary PDF/image -- see
// model.ViewTypeDocument's own doc comment for why that's a deliberate,
// named scope cut, not an oversight.
func (h *Handler) Document(w http.ResponseWriter, r *http.Request) {
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
	view := h.interp.Get().DocumentView(machineID)
	if view == nil || view.Config.Template == "" {
		http.NotFound(w, r)
		return
	}
	rec, err := h.records.Get(r.Context(), recordID)
	if err != nil || rec.MachineID != machineID {
		http.NotFound(w, r)
		return
	}
	tmpl, err := htmltemplate.New(view.ID).Parse(view.Config.Template)
	if err != nil {
		slog.Error("parse document template", "view", view.ID, "error", err)
		http.Error(w, "failed to render document", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.Execute(w, rec.Data); err != nil {
		slog.Error("render document", "view", view.ID, "error", err)
	}
}

// NewForm — form for creating a new record.
func (h *Handler) NewForm(w http.ResponseWriter, r *http.Request) {
	machineID := chi.URLParam(r, "machineID")
	machine, ok := h.interp.Get().GetMachine(machineID)
	if !ok {
		http.NotFound(w, r)
		return
	}
	workspaceID, applicationID := h.interp.Get().ScopeFor(machineID)
	if workspaceID != h.workspace(r) {
		// CAP-X06: a Machine from another Workspace 404s exactly like one
		// that doesn't exist at all -- app-layer guard alongside RLS
		// (migrations/009), not instead of it.
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

	// CAP-V12: a FormView declaring Steps renders as a multi-step wizard
	// instead of the single Form -- step 0, no carried-forward values yet.
	if fv := h.interp.Get().FormView(machine.ID); fv != nil && len(fv.Config.Steps) > 0 {
		page := ui.WizardForm(h.workspaceName(r), a.User.Name, a.CSRFToken, h.isWorkspaceAdmin(r), machine, 0, len(fv.Config.Steps), h.buildFormFieldsFor(r.Context(), machine, fv.Config.Steps[0], nil), nil, nil, h.unreadCount(r.Context(), a), h.subNavFor(r, machine))
		if err := page.Render(r.Context(), w); err != nil {
			slog.Error("render wizard form", "error", err)
		}
		return
	}

	page := ui.Form(h.workspaceName(r), a.User.Name, a.CSRFToken, h.isWorkspaceAdmin(r), machine, "", h.buildFormFields(r.Context(), machine, nil), nil, h.unreadCount(r.Context(), a), h.buildChildLinesData(r.Context(), machine), h.subNavFor(r, machine))
	if err := page.Render(r.Context(), w); err != nil {
		slog.Error("render form", "error", err)
	}
}

// Create — handle new record form submission.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	machineID := chi.URLParam(r, "machineID")
	machine, ok := h.interp.Get().GetMachine(machineID)
	if !ok {
		http.NotFound(w, r)
		return
	}
	workspaceID, applicationID := h.interp.Get().ScopeFor(machineID)
	if workspaceID != h.workspace(r) {
		// CAP-X06: a Machine from another Workspace 404s exactly like one
		// that doesn't exist at all -- app-layer guard alongside RLS
		// (migrations/009), not instead of it.
		http.NotFound(w, r)
		return
	}
	if role := h.roleForApp(r, applicationID); !h.guard.CanCreate(machine, role) {
		h.logPermissionDenied(r.Context(), "create", machineID, "", role, h.identity(r))
		http.Error(w, "not permitted", http.StatusForbidden)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	fv := h.interp.Get().FormView(machine.ID)

	// CAP-V12: an intermediate wizard-step submission renders the NEXT step
	// instead of creating anything -- only the final step's POST falls
	// through to the ordinary Create logic below. No session state: every
	// prior step's value travels forward as a hidden input on each step's
	// page, so by the final POST every field is present in r.Form exactly
	// like a single-step form's would be.
	if fv != nil && len(fv.Config.Steps) > 0 {
		step, convErr := strconv.Atoi(r.FormValue("wizard_step"))
		if convErr != nil || step < 0 || step >= len(fv.Config.Steps) {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		if step+1 < len(fv.Config.Steps) {
			var carried []ui.HiddenField
			for i := 0; i <= step; i++ {
				for _, id := range fv.Config.Steps[i] {
					if v := r.FormValue(id); v != "" {
						carried = append(carried, ui.HiddenField{Name: id, Value: v})
					}
				}
			}
			a := h.auth(r)
			page := ui.WizardForm(h.workspaceName(r), a.User.Name, a.CSRFToken, h.isWorkspaceAdmin(r), machine, step+1, len(fv.Config.Steps),
				h.buildFormFieldsFor(r.Context(), machine, fv.Config.Steps[step+1], nil), carried, nil, h.unreadCount(r.Context(), a), h.subNavFor(r, machine))
			if err := page.Render(r.Context(), w); err != nil {
				slog.Error("render wizard form", "error", err)
			}
			return
		}
		// Final step -- fall through; every field (this step's and every
		// earlier one's, carried as hidden inputs) is in r.Form already.
	}

	// Any value_list field the Create form doesn't expose (Status, Decision, ...)
	// starts at its first declared value — the same "first value = initial
	// state" convention guides/writing-menata.md teaches .menata authors,
	// generalized from what was previously a "status"-named-field-only rule.
	formFieldIDs := map[string]bool{}
	if fv != nil {
		for _, id := range fv.Config.Fields {
			formFieldIDs[id] = true
		}
		for _, step := range fv.Config.Steps {
			for _, id := range step {
				formFieldIDs[id] = true
			}
		}
	}
	data := make(map[string]any)
	for _, f := range machine.Fields {
		if formFieldIDs[f.ID] || f.Type == model.FieldTypeComputed {
			continue
		}
		if f.Type == model.FieldTypeValueList && len(f.Options.Values) > 0 {
			data[f.ID] = f.Options.Values[0]
		} else if f.Options.Default != "" {
			// CAP-F15: any field's own declared default, generalized from
			// the value_list-only convention above -- applies whenever the
			// Create form doesn't expose the field at all.
			data[f.ID] = f.Options.Default
		}
	}
	for _, f := range machine.Fields {
		if f.Type == model.FieldTypeComputed {
			continue // CAP-F14: never a stored value, never read from a form
		}
		if v := r.FormValue(f.ID); v != "" {
			data[f.ID] = v
		} else if f.Type == model.FieldTypeBoolean && formFieldIDs[f.ID] {
			data[f.ID] = "false" // CAP-F09: an unchecked checkbox submits nothing at all
		} else if f.Options.Default != "" && formFieldIDs[f.ID] {
			data[f.ID] = f.Options.Default // CAP-F15: exposed but left blank
		}
	}
	// CAP-F06: `file` fields' actual uploaded bytes -- csrfProtect
	// (cmd/server/main.go) already called ParseMultipartForm for a
	// multipart request before this handler ever runs, so r.MultipartForm
	// is already populated here.
	uploaded, err := h.processFileUploads(r, machine)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	for fieldID, key := range uploaded {
		data[fieldID] = key
	}

	// CAP-F18: an auto-number field left blank gets the next sequence value
	// -- atomic per (machine, field), never a submitter-supplied string.
	for _, f := range machine.Fields {
		if f.Options.AutoNumberPrefix == "" {
			continue
		}
		if v, ok := data[f.ID]; ok && fmt.Sprintf("%v", v) != "" {
			continue
		}
		n, err := h.records.NextSequence(r.Context(), machineID, f.ID)
		if err != nil {
			http.Error(w, "failed to generate document number", http.StatusInternalServerError)
			return
		}
		data[f.ID] = formatAutoNumber(f.Options, n)
	}

	// CAP-R08: see the matching comment in Update -- a record created
	// directly into its declared "scratch" state skips business-rule
	// Constraints too (referential integrity still applies, below).
	var violations []string
	if !h.inScratchState(machine, data) {
		violations = h.engine.Violations(machine, withChangePolicyCreatedAt(machine, data, time.Now()))
	}
	refViolations, err := h.referenceViolations(r.Context(), machine, data)
	if err != nil {
		http.Error(w, "failed to validate references", http.StatusInternalServerError)
		return
	}
	violations = append(violations, refViolations...)
	userViolations, err := h.userReferenceViolations(r.Context(), machine, data)
	if err != nil {
		http.Error(w, "failed to validate references", http.StatusInternalServerError)
		return
	}
	violations = append(violations, userViolations...)
	uniqueViolations, err := h.uniquenessViolations(r.Context(), machine, data, "")
	if err != nil {
		http.Error(w, "failed to validate uniqueness", http.StatusInternalServerError)
		return
	}
	violations = append(violations, uniqueViolations...)
	crossRecViolations, err := h.crossRecordViolations(r.Context(), machine, data, "")
	if err != nil {
		http.Error(w, "failed to validate cross-record constraints", http.StatusInternalServerError)
		return
	}
	violations = append(violations, crossRecViolations...)

	// CAP-F16: a form with embedded child rows validates them together with
	// the parent -- one combined violations list, so a bad child row blocks
	// the whole submission exactly like a bad parent field would, before
	// anything (parent or child) is written.
	var childLines *model.ChildLinesConfig
	var childRowsData []map[string]any
	if fv != nil {
		childLines = fv.Config.ChildLines
	}
	if childLines != nil {
		rows, rowViolations, err := h.validateChildRows(r.Context(), r, childLines)
		if err != nil {
			http.Error(w, "failed to validate child rows", http.StatusInternalServerError)
			return
		}
		childRowsData = rows
		violations = append(violations, rowViolations...)
	}

	if len(violations) > 0 {
		role := h.roleForApp(r, applicationID)
		h.logRuleViolation(r.Context(), "create", machineID, "", role, h.identity(r), strings.Join(violations, "; "))
		a := h.auth(r)
		// CAP-V12: a violation on the wizard's final step re-renders that
		// same last step (with everything typed so far preserved), not the
		// plain single-step Form -- a wizard View's own Fields is empty
		// (Steps replaces it), so ui.Form would otherwise render nothing.
		if fv != nil && len(fv.Config.Steps) > 0 {
			last := len(fv.Config.Steps) - 1
			var carried []ui.HiddenField
			for i := 0; i < last; i++ {
				for _, id := range fv.Config.Steps[i] {
					if v, ok := data[id]; ok {
						carried = append(carried, ui.HiddenField{Name: id, Value: fmt.Sprintf("%v", v)})
					}
				}
			}
			page := ui.WizardForm(h.workspaceName(r), a.User.Name, a.CSRFToken, h.isWorkspaceAdmin(r), machine, last, len(fv.Config.Steps),
				h.buildFormFieldsFor(r.Context(), machine, fv.Config.Steps[last], data), carried, violations, h.unreadCount(r.Context(), a), h.subNavFor(r, machine))
			if err := page.Render(r.Context(), w); err != nil {
				slog.Error("render wizard form (violations)", "error", err)
			}
			return
		}
		page := ui.Form(h.workspaceName(r), a.User.Name, a.CSRFToken, h.isWorkspaceAdmin(r), machine, "", h.buildFormFields(r.Context(), machine, data), violations, h.unreadCount(r.Context(), a), h.buildChildLinesData(r.Context(), machine), h.subNavFor(r, machine))
		if err := page.Render(r.Context(), w); err != nil {
			slog.Error("render form (violations)", "error", err)
		}
		return
	}

	rec, err := h.records.Create(r.Context(), machineID, workspaceID, data)
	if err != nil {
		http.Error(w, "failed to create record", http.StatusInternalServerError)
		return
	}
	if childLines != nil {
		if err := h.insertChildRows(r.Context(), childLines, workspaceID, childRowsData, rec.ID); err != nil {
			http.Error(w, "failed to create child rows", http.StatusInternalServerError)
			return
		}
	}
	// CAP-W01: write-time fan-in -- if this record references a Machine
	// whose Process declares a Requirement naming machineID as its target,
	// stamp that parent's counter now. Error propagates to a real 5xx (the
	// CAP-X12 lesson: a swallowed error here would leave the child
	// committed but its parent's counter silently stale) so workspaceTx
	// rolls the whole request back, not just this write.
	if err := h.stampRequirementCounters(r.Context(), machine, rec); err != nil {
		http.Error(w, "failed to update requirement counter", http.StatusInternalServerError)
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
	machine, ok := h.interp.Get().GetMachine(machineID)
	if !ok {
		http.NotFound(w, r)
		return
	}
	workspaceID, applicationID := h.interp.Get().ScopeFor(machineID)
	if workspaceID != h.workspace(r) {
		// CAP-X06: a Machine from another Workspace 404s exactly like one
		// that doesn't exist at all -- app-layer guard alongside RLS
		// (migrations/009), not instead of it.
		http.NotFound(w, r)
		return
	}
	role := h.roleForApp(r, applicationID)
	if !h.guard.CanEdit(machine, role) {
		h.logPermissionDenied(r.Context(), "edit", machineID, "", role, h.identity(r))
		http.Error(w, "not permitted", http.StatusForbidden)
		return
	}
	rec, err := h.records.Get(r.Context(), recordID)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	// CAP-R07: an immutable record's edit form isn't even offered -- Update
	// re-checks the same gate as defense in depth, same "guard the mutation
	// path, not just the button" reasoning CAP-P05 already uses elsewhere.
	if reason := h.immutabilityViolation(machine, rec.Data); reason != "" {
		http.Error(w, reason, http.StatusForbidden)
		return
	}
	a := h.auth(r)
	page := ui.Form(h.workspaceName(r), a.User.Name, a.CSRFToken, h.isWorkspaceAdmin(r), machine, recordID, h.buildFormFields(r.Context(), machine, rec.Data), nil, h.unreadCount(r.Context(), a), nil, h.subNavFor(r, machine))
	if err := page.Render(r.Context(), w); err != nil {
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
	machine, ok := h.interp.Get().GetMachine(machineID)
	if !ok {
		http.NotFound(w, r)
		return
	}
	workspaceID, applicationID := h.interp.Get().ScopeFor(machineID)
	if workspaceID != h.workspace(r) {
		// CAP-X06: a Machine from another Workspace 404s exactly like one
		// that doesn't exist at all -- app-layer guard alongside RLS
		// (migrations/009), not instead of it.
		http.NotFound(w, r)
		return
	}
	if role := h.roleForApp(r, applicationID); !h.guard.CanEdit(machine, role) {
		h.logPermissionDenied(r.Context(), "edit", machineID, "", role, h.identity(r))
		http.Error(w, "not permitted", http.StatusForbidden)
		return
	}
	rec, err := h.records.Get(r.Context(), recordID)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	// CAP-R07: defense in depth -- EditForm already refuses to render for
	// an immutable record, but Update is reachable directly (a replayed
	// form, a hand-crafted POST) without going through that form first.
	if reason := h.immutabilityViolation(machine, rec.Data); reason != "" {
		http.Error(w, reason, http.StatusForbidden)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	formFieldIDs := map[string]bool{}
	if fv := h.interp.Get().FormView(machine.ID); fv != nil {
		for _, id := range fv.Config.Fields {
			formFieldIDs[id] = true
		}
	}
	data := make(map[string]any, len(rec.Data))
	for k, v := range rec.Data {
		data[k] = v
	}
	for _, f := range machine.Fields {
		if !formFieldIDs[f.ID] || f.Type == model.FieldTypeComputed {
			continue // CAP-F14: never a stored value, never read from a form
		}
		if f.Type == model.FieldTypeBoolean {
			// CAP-F09: an unchecked checkbox submits nothing at all --
			// FormValue("") would otherwise silently write an empty string
			// instead of "false".
			if r.FormValue(f.ID) == "true" {
				data[f.ID] = "true"
			} else {
				data[f.ID] = "false"
			}
			continue
		}
		if f.Type == model.FieldTypeFile {
			// CAP-F06: a file input's FormValue is always "" (the browser
			// sends the bytes as a multipart file part, not a form value)
			// -- leave the record's existing stored key alone here; a real
			// new upload (if any) overlays it below, AFTER this loop.
			continue
		}
		data[f.ID] = r.FormValue(f.ID)
	}
	uploaded, err := h.processFileUploads(r, machine)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	for fieldID, key := range uploaded {
		data[fieldID] = key
	}

	// CAP-R08: a record still in its declared "scratch" state (e.g. a Cart
	// before Checkout) has none of its eventual business-rule Constraints
	// enforced yet -- CAP-C09's own trigger-time re-validation is the real
	// commit-point gate, once an event moves it out of that state.
	// Referential integrity (below) still applies even in scratch state --
	// a scratch record can be incomplete, not corrupt.
	var violations []string
	if !h.inScratchState(machine, data) {
		violations = h.engine.Violations(machine, withChangePolicyCreatedAt(machine, data, rec.CreatedAt))
	}
	refViolations, err := h.referenceViolations(r.Context(), machine, data)
	if err != nil {
		http.Error(w, "failed to validate references", http.StatusInternalServerError)
		return
	}
	violations = append(violations, refViolations...)
	userViolations, err := h.userReferenceViolations(r.Context(), machine, data)
	if err != nil {
		http.Error(w, "failed to validate references", http.StatusInternalServerError)
		return
	}
	violations = append(violations, userViolations...)
	uniqueViolations, err := h.uniquenessViolations(r.Context(), machine, data, recordID)
	if err != nil {
		http.Error(w, "failed to validate uniqueness", http.StatusInternalServerError)
		return
	}
	violations = append(violations, uniqueViolations...)
	crossRecViolations, err := h.crossRecordViolations(r.Context(), machine, data, recordID)
	if err != nil {
		http.Error(w, "failed to validate cross-record constraints", http.StatusInternalServerError)
		return
	}
	violations = append(violations, crossRecViolations...)

	if len(violations) > 0 {
		role := h.roleForApp(r, applicationID)
		h.logRuleViolation(r.Context(), "update", machineID, "", role, h.identity(r), strings.Join(violations, "; "))
		a := h.auth(r)
		page := ui.Form(h.workspaceName(r), a.User.Name, a.CSRFToken, h.isWorkspaceAdmin(r), machine, recordID, h.buildFormFields(r.Context(), machine, data), violations, h.unreadCount(r.Context(), a), nil, h.subNavFor(r, machine))
		if err := page.Render(r.Context(), w); err != nil {
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

	machine, ok := h.interp.Get().GetMachine(machineID)
	if !ok {
		http.NotFound(w, r)
		return
	}
	workspaceID, applicationID := h.interp.Get().ScopeFor(machineID)
	if workspaceID != h.workspace(r) {
		// CAP-X06: a Machine from another Workspace 404s exactly like one
		// that doesn't exist at all -- app-layer guard alongside RLS
		// (migrations/009), not instead of it.
		http.NotFound(w, r)
		return
	}
	role := h.roleForApp(r, applicationID)
	if !h.guard.CanRead(machine, role) {
		h.logPermissionDenied(r.Context(), "read", machineID, "", role, h.identity(r))
		http.Error(w, "not permitted", http.StatusForbidden)
		return
	}
	rec, err := h.records.Get(r.Context(), recordID)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	// CAP-P06: a field this role's Permission hides never reaches the
	// Detail page.
	hidden := h.hiddenFields(machine, role)
	detailView := h.interp.Get().DetailView(machineID) // CAP-V17: nil if none declared, same as every other optional View lookup
	fields := make([]ui.DetailField, 0, len(machine.Fields))
	for _, f := range machine.Fields {
		if hidden[f.ID] {
			continue
		}
		val := ""
		if v, ok := rec.Data[f.ID]; ok {
			val = fmt.Sprintf("%v", v)
		}
		link := ""
		switch {
		case f.Type == model.FieldTypeReference && val != "":
			refID := val
			target := f.Options.TargetMachine
			if label, err := h.referenceLabel(r.Context(), target, refID); err == nil && label != "" {
				val = label
				link = "/" + target + "/" + refID
			}
		case f.Type == model.FieldTypeUser && val != "":
			if label, err := h.userLabel(r.Context(), val); err == nil && label != "" {
				val = label
			}
		case f.Type == model.FieldTypeBoolean:
			val = boolLabel(val) // CAP-F09
		case f.Type == model.FieldTypeMoney && val != "":
			val = formatMoney(val, f, rec.Data) // CAP-F08
		case f.Type == model.FieldTypeFile && val != "":
			link = "/files/" + val // CAP-F06
		case f.Type == model.FieldTypeComputed:
			val = computedValue(f, rec.Data) // CAP-F14
		}
		urgency := ""
		if detailView != nil && f.ID == detailView.Config.SlaField {
			if label, u, ok := slaUrgency(val, detailView.Config.SlaWarningDays); ok { // CAP-V17
				val, urgency = label, u
			}
		}
		fields = append(fields, ui.DetailField{Name: f.Name, Value: val, Link: link, SlaUrgency: urgency})
	}

	childLists := h.childLists(r.Context(), machine, recordID)
	events := h.interp.Get().PermittedEventsForRecord(machineID, role, h.identityID(r), rec.Data)
	// CAP-P04: an event declaring InputFields (e.g. "delegate to") renders
	// an inline picker alongside its trigger button, same field/options
	// shape a Form uses -- built here, not in ui, since resolving a `user`
	// field's own picker options needs the UserStore (buildFormFieldsFor).
	permittedEvents := make([]ui.EventTrigger, len(events))
	for i, evt := range events {
		permittedEvents[i] = ui.EventTrigger{Event: evt}
		if len(evt.InputFields) > 0 {
			permittedEvents[i].Inputs = h.buildFormFieldsFor(r.Context(), machine, evt.InputFields, nil)
		}
	}
	coordPlaceURL := "" // CAP-V21: only linked when this Machine declares one
	if h.interp.Get().CoordPlacementView(machineID) != nil {
		coordPlaceURL = "/" + machineID + "/" + recordID + "/place"
	}
	a := h.auth(r)
	page := ui.Detail(h.workspaceName(r), a.User.Name, a.CSRFToken, h.isWorkspaceAdmin(r), machine, rec, fields, permittedEvents, childLists, h.unreadCount(r.Context(), a), h.subNavFor(r, machine), coordPlaceURL)
	if err := page.Render(r.Context(), w); err != nil {
		slog.Error("render detail", "error", err)
	}
}
