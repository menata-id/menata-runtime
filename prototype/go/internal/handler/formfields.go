package handler

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"slices"
	"strings"

	"github.com/go-chi/chi/v5"

	"menata.id/runtime/internal/constraint"
	"menata.id/runtime/internal/model"
	"menata.id/runtime/internal/ui"
)

// hiddenFields (CAP-P06/CAP-O07): a field is hidden only if EVERY held
// role's own matching Permission row hides it -- visible if ANY held role
// reveals it. Same union-favors-access posture as
// Guard.CanRead/CanCreate/CanEdit (holding more roles only ever grants
// more, never less) applied to this narrower, opt-in restriction: a role
// with no Permission row on this machine at all doesn't count towards
// hiding (CAP-P05's deny-by-default already governs whether it can see the
// Machine at all).
func (h *Handler) hiddenFields(machine *model.Machine, roles []string) map[string]bool {
	var hiddenSets []map[string]bool
	for _, perm := range machine.Permissions {
		if !slices.Contains(roles, perm.Role) {
			continue
		}
		set := make(map[string]bool, len(perm.HiddenFields))
		for _, id := range perm.HiddenFields {
			set[id] = true
		}
		hiddenSets = append(hiddenSets, set)
	}
	if len(hiddenSets) == 0 {
		return map[string]bool{}
	}
	out := map[string]bool{}
	for id := range hiddenSets[0] {
		hiddenByAll := true
		for _, set := range hiddenSets[1:] {
			if !set[id] {
				hiddenByAll = false
				break
			}
		}
		if hiddenByAll {
			out[id] = true
		}
	}
	return out
}

func (h *Handler) buildFormFields(ctx context.Context, machine *model.Machine, vals map[string]any) []ui.FormField {
	var fieldIDs []string
	if view := h.interp.Get().FormView(machine.ID); view != nil {
		fieldIDs = view.Config.Fields
	}
	return h.buildFormFieldsFor(ctx, machine, fieldIDs, vals)
}

// buildFormFieldsFor is buildFormFields narrowed to an explicit fieldIDs
// subset -- CAP-V12's own use, one step's worth of fields at a time, rather
// than a FormView's whole declared set.
// typeaheadThreshold (CAP-V16) is the candidate-pool size past which a
// reference/user picker switches from an eager <select> to a search-as-
// you-type fragment swap -- a judgment call, not derived from a case,
// matching the existing pageSize=25 convention (record_crud.go's List)
// rather than inventing a second number.
const typeaheadThreshold = 25

func (h *Handler) buildFormFieldsFor(ctx context.Context, machine *model.Machine, fieldIDs []string, vals map[string]any) []ui.FormField {
	fieldByID := fieldIndex(machine)

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
		switch f.Type {
		case model.FieldTypeReference:
			opts = h.referenceOptions(ctx, f.Options.TargetMachine)
		case model.FieldTypeUser:
			opts = h.userFieldOptions(ctx, machine.ApplicationID)
		}
		ff := ui.FormField{Field: f, Name: f.ID, Value: val, Options: opts}
		if (f.Type == model.FieldTypeReference || f.Type == model.FieldTypeUser) && len(opts) > typeaheadThreshold {
			ff.Typeahead = true
			ff.Options = nil // nothing to render into a <select> with -- TypeaheadPicker queries TypeaheadURL instead
			ff.TypeaheadURL = "/" + machine.ID + "/field-options?field=" + f.ID
			if val != "" {
				if f.Type == model.FieldTypeReference {
					ff.TypeaheadLabel, _ = h.referenceLabel(ctx, f.Options.TargetMachine, val)
				} else {
					ff.TypeaheadLabel, _ = h.userLabel(ctx, val)
				}
			}
		}
		// CAP-V19: this reference Field is what a CrossRecord{Kind:
		// "reference_field"} Constraint (CAP-C11) on this SAME Machine
		// gates on -- surface a live preview of the referenced record's
		// own TargetField, fetched from CAP-X07's existing API, no new
		// route.
		if f.Type == model.FieldTypeReference {
			for _, c := range machine.Constraints {
				if c.CrossRecord == nil || c.CrossRecord.Kind != "reference_field" || c.CrossRecord.ReferenceField != f.ID {
					continue
				}
				ff.LivePreviewURL = "/api/" + f.Options.TargetMachine + "/"
				ff.LivePreviewField = c.CrossRecord.TargetField
				if targetMachine, ok := h.interp.Get().GetMachine(f.Options.TargetMachine); ok {
					if tf, ok := fieldIndex(targetMachine)[c.CrossRecord.TargetField]; ok {
						ff.LivePreviewLabel = tf.Name
					}
				}
				break
			}
		}
		fields = append(fields, ff)
	}
	return fields
}

// childRowName (CAP-F16) builds a repeated child-line row's indexed HTML
// input name -- "child_0_fld_x", "child_1_fld_x", ... -- so a fixed-slot
// row-editor's fields don't collide with each other or the parent form's
// own field names. parseChildRows below is this function's exact inverse.
func childRowName(row int, fieldID string) string {
	return fmt.Sprintf("child_%d_%s", row, fieldID)
}

// buildChildLinesData (CAP-F16) builds a form's embedded child-Machine row
// editor, if the FormView declares one -- MaxRows blank row slots (or
// MaxRows pre-filled from existingRows on... no: CREATE-time only, see
// ChildLinesConfig's own doc comment for why edit doesn't use this).
// existingRows is nil at Create; kept as a param so a future edit-time
// extension has an obvious seam, not because anything passes it non-nil today.
func (h *Handler) buildChildLinesData(ctx context.Context, machine *model.Machine) *ui.ChildLinesData {
	view := h.interp.Get().FormView(machine.ID)
	if view == nil || view.Config.ChildLines == nil {
		return nil
	}
	cl := view.Config.ChildLines
	childMachine, ok := h.interp.Get().GetMachine(cl.Machine)
	if !ok {
		return nil
	}
	fieldByID := fieldIndex(childMachine)
	maxRows := cl.MaxRows
	if maxRows <= 0 {
		maxRows = 10
	}

	rows := make([][]ui.FormField, maxRows)
	for i := 0; i < maxRows; i++ {
		row := make([]ui.FormField, 0, len(cl.Fields))
		for _, fid := range cl.Fields {
			f, ok := fieldByID[fid]
			if !ok || fid == cl.ParentField {
				continue
			}
			var opts []ui.ReferenceOption
			switch f.Type {
			case model.FieldTypeReference:
				opts = h.referenceOptions(ctx, f.Options.TargetMachine)
			case model.FieldTypeUser:
				opts = h.userFieldOptions(ctx, childMachine.ApplicationID)
			}
			row = append(row, ui.FormField{Field: f, Name: childRowName(i, fid), Options: opts})
		}
		rows[i] = row
	}
	data := &ui.ChildLinesData{Title: childMachine.Name, Rows: rows}

	// CAP-V15: if this parent declares an aggregate cross-record check
	// (CAP-C10, internal/handler/crossrecord.go) against this SAME child
	// Machine, surface FieldA/FieldB for a live client-side running total
	// -- purely presentational, the real gate stays the already-proven
	// server-side check unchanged. Matched by convention (ChildMachine ==
	// ChildLinesConfig.Machine on the same parent), the same class of
	// heuristic this codebase already documents elsewhere (CLAUDE.md).
	for _, c := range machine.Constraints {
		if c.CrossRecord == nil || c.CrossRecord.Kind != "aggregate" || c.CrossRecord.ChildMachine != cl.Machine {
			continue
		}
		data.SumFieldA = c.CrossRecord.FieldA
		if fa, ok := fieldByID[c.CrossRecord.FieldA]; ok {
			data.SumFieldALabel = fa.Name
		}
		if c.CrossRecord.FieldB != "" {
			data.SumFieldB = c.CrossRecord.FieldB
			if fb, ok := fieldByID[c.CrossRecord.FieldB]; ok {
				data.SumFieldBLabel = fb.Name
			}
		}
		break
	}
	return data
}

// validateChildRows (CAP-F16) reads every non-empty row the fixed-slot
// child row editor submitted and validates each against the child
// Machine's own Constraints (CAP-C01..C12, the same rules a direct Create
// on that Machine would run) -- except any constraint on ParentField
// itself, which deliberately isn't set yet here (the parent record this
// row will reference doesn't exist until AFTER the parent insert that
// follows a clean validation pass -- see Create's own two-phase call to
// this then insertChildRows). A row is "empty" (silently skipped, not an
// error) when every one of its fields was left blank -- matches the UI's
// own "leave a row blank to skip it" hint. Returned data never has
// ParentField set; insertChildRows adds it once the parent's real id exists.
func (h *Handler) validateChildRows(ctx context.Context, r *http.Request, cl *model.ChildLinesConfig) ([]map[string]any, []string, error) {
	childMachine, ok := h.interp.Get().GetMachine(cl.Machine)
	if !ok {
		return nil, nil, fmt.Errorf("child_lines.machine %q not found", cl.Machine)
	}
	maxRows := cl.MaxRows
	if maxRows <= 0 {
		maxRows = 10
	}

	var rowsData []map[string]any
	var violations []string
	for i := 0; i < maxRows; i++ {
		data := make(map[string]any)
		anySet := false
		for _, fid := range cl.Fields {
			if fid == cl.ParentField {
				continue
			}
			v := r.FormValue(childRowName(i, fid))
			if v != "" {
				data[fid] = v
				anySet = true
			}
		}
		if !anySet {
			continue
		}

		var rowViolations []string
		for _, c := range childMachine.Constraints {
			if c.Expression.Field == cl.ParentField || c.Expression.Operator == "unique" {
				// ParentField's own constraints (e.g. "required") don't
				// apply yet -- it isn't set until insertChildRows. Uniqueness
				// needs the database (handler.uniquenessViolations below),
				// not constraint.Eval, same as everywhere else this
				// distinction is made.
				continue
			}
			if c.Condition != nil && !constraint.Eval(*c.Condition, data) {
				continue
			}
			if !constraint.Eval(c.Expression, data) {
				rowViolations = append(rowViolations, c.Rule)
			}
		}
		refViolations, err := h.referenceViolations(ctx, childMachine, data)
		if err != nil {
			return nil, nil, err
		}
		rowViolations = append(rowViolations, refViolations...)
		userViolations, err := h.userReferenceViolations(ctx, childMachine, data)
		if err != nil {
			return nil, nil, err
		}
		rowViolations = append(rowViolations, userViolations...)
		for _, v := range rowViolations {
			violations = append(violations, fmt.Sprintf("%s, row %d: %s", childMachine.Name, i+1, v))
		}
		rowsData = append(rowsData, data)
	}
	return rowsData, violations, nil
}

// insertChildRows (CAP-F16) writes every validated child row from
// validateChildRows, stamping ParentField with the just-created parent's
// real id on each -- called only after that parent insert has succeeded,
// so every row's back-reference is valid by construction, never a
// dangling one validateChildRows would have had to reject.
func (h *Handler) insertChildRows(ctx context.Context, cl *model.ChildLinesConfig, workspaceID string, rowsData []map[string]any, parentID string) error {
	for _, data := range rowsData {
		data[cl.ParentField] = parentID
		if _, err := h.records.Create(ctx, cl.Machine, workspaceID, data); err != nil {
			return err
		}
	}
	return nil
}

// childLists finds every Machine with a `reference` field pointing at
// machine, and lists the records where that field equals recordID (CAP-V06).
// Generic by construction — it doesn't special-case Employee/Manager, so any
// future reference relationship gets a sub-list automatically.
func (h *Handler) childLists(ctx context.Context, machine *model.Machine, recordID string) []ui.ChildList {
	var out []ui.ChildList
	for _, m := range h.interp.Get().AllMachines() {
		for _, f := range m.Fields {
			if f.Type != model.FieldTypeReference || f.Options.TargetMachine != machine.ID {
				continue
			}
			records, err := h.records.List(ctx, m.ID, "", "")
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
	records, err := h.records.List(ctx, targetMachineID, "", "")
	if err != nil {
		slog.Error("list reference options", "target_machine", targetMachineID, "error", err)
		return nil
	}
	targetMachine, _ := h.interp.Get().GetMachine(targetMachineID)
	opts := make([]ui.ReferenceOption, 0, len(records))
	for _, rec := range records {
		opts = append(opts, ui.ReferenceOption{ID: rec.ID, Label: displayLabel(targetMachine, rec.Data)})
	}
	return opts
}

// FieldOptions (CAP-V16) is TypeaheadPicker's own search endpoint -- an
// HTML fragment (FieldOptionsFragment), not JSON, matching this project's
// HTMX-driven "no SPA framework" posture rather than extending CAP-X07's
// JSON API with a new query surface. Read-gated the same as List/Detail on
// the SAME Machine the field lives on (?field= must name a real Field on
// it) -- a picker never leaks more than the form itself already would.
// Substring match on Label reuses CAP-V08's own case-insensitive
// contains-match logic (record_crud.go's List), applied here to the
// already-built ReferenceOption list rather than to raw record fields.
func (h *Handler) FieldOptions(w http.ResponseWriter, r *http.Request) {
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
	f, ok := fieldIndex(machine)[r.URL.Query().Get("field")]
	if !ok {
		http.NotFound(w, r)
		return
	}
	var opts []ui.ReferenceOption
	switch f.Type {
	case model.FieldTypeReference:
		opts = h.referenceOptions(r.Context(), f.Options.TargetMachine)
	case model.FieldTypeUser:
		opts = h.userFieldOptions(r.Context(), applicationID)
	default:
		http.NotFound(w, r)
		return
	}
	if q := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("q"))); q != "" {
		kept := opts[:0]
		for _, o := range opts {
			if strings.Contains(strings.ToLower(o.Label), q) {
				kept = append(kept, o)
			}
		}
		opts = kept
	}
	page := ui.FieldOptionsFragment(f.ID, opts)
	if err := page.Render(r.Context(), w); err != nil {
		slog.Error("render field options", "error", err)
	}
}

// referenceLabel resolves one record's display label, for rendering an
// already-set reference value (detail/list views).
func (h *Handler) referenceLabel(ctx context.Context, targetMachineID, recordID string) (string, error) {
	rec, err := h.records.Get(ctx, recordID)
	if err != nil {
		return "", err
	}
	targetMachine, _ := h.interp.Get().GetMachine(targetMachineID)
	return displayLabel(targetMachine, rec.Data), nil
}

// userFieldOptions (CAP-F05) is a `user` field's picker candidate pool --
// scoped to people who hold any role in the field's own Machine's
// Application (UserStore.ListForApplicationRole), the CAP-O01-derived
// query-time filter this capability's own design settled on rather than a
// new metadata concept (see benchmarks/007-user-role-management-survey.md).
func (h *Handler) userFieldOptions(ctx context.Context, applicationID string) []ui.ReferenceOption {
	users, err := h.users.ListForApplicationRole(ctx, applicationID)
	if err != nil {
		slog.Error("list user field options", "application", applicationID, "error", err)
		return nil
	}
	opts := make([]ui.ReferenceOption, 0, len(users))
	for _, u := range users {
		opts = append(opts, ui.ReferenceOption{ID: u.ID, Label: u.Name})
	}
	return opts
}

// userLabel (CAP-F05) resolves one user id's display name, for rendering an
// already-set `user` field value (detail/list views) -- the person-field
// counterpart to referenceLabel. Unlike a reference field, there's no
// profile page to link to in this prototype, so callers render plain text,
// never a link.
func (h *Handler) userLabel(ctx context.Context, userID string) (string, error) {
	u, err := h.users.GetByID(ctx, userID)
	if err != nil {
		return "", err
	}
	return u.Name, nil
}
