package handler

import (
	"fmt"
	"log/slog"
	"net/http"
	"sort"

	"github.com/go-chi/chi/v5"

	"menata.id/runtime/internal/model"
	"menata.id/runtime/internal/store"
	"menata.id/runtime/internal/ui"
)

// viewNavFor (CAP-O03 Tier 3) resolves the within-Machine view-type nav
// pill for a collection-level page -- List/Report/Board/Calendar/Timeline
// -- linking to this same Machine's own other declared View types,
// active-highlighting whichever one is being rendered. Distinct from
// subNavFor's cross-Machine axis: every candidate here lives on the same
// Machine the caller already passed a CanRead check for, so (unlike
// subNavFor) no separate permission trim is needed. Detail/Form/WizardForm
// never call this -- ADR-008 (docs/decisions/008-mobile-ui-navigation-
// standard.md): a single record isn't a "view" to switch between, and a
// Form is a focused task.
func (h *Handler) viewNavFor(machineID string, active model.ViewType) []ui.ViewNavLink {
	interp := h.interp.Get()
	var links []ui.ViewNavLink
	if interp.DefaultListView(machineID) != nil {
		links = append(links, ui.ViewNavLink{ID: "list", Name: "List", Path: "/" + machineID, Active: active == model.ViewTypeList})
	}
	if interp.ReportView(machineID) != nil {
		links = append(links, ui.ViewNavLink{ID: "report", Name: "Report", Path: "/" + machineID + "/report", Active: active == model.ViewTypeReport})
	}
	if interp.BoardView(machineID) != nil {
		links = append(links, ui.ViewNavLink{ID: "board", Name: "Board", Path: "/" + machineID + "/board", Active: active == model.ViewTypeBoard})
	}
	if interp.CalendarView(machineID) != nil {
		links = append(links, ui.ViewNavLink{ID: "calendar", Name: "Calendar", Path: "/" + machineID + "/calendar", Active: active == model.ViewTypeCalendar})
	}
	if interp.TimelineView(machineID) != nil {
		links = append(links, ui.ViewNavLink{ID: "timeline", Name: "Timeline", Path: "/" + machineID + "/timeline", Active: active == model.ViewTypeTimeline})
	}
	if len(links) < 2 {
		return nil // nothing to switch to
	}
	return links
}

func (h *Handler) Report(w http.ResponseWriter, r *http.Request) {
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
	view := h.interp.Get().ReportView(machineID)
	if view == nil || view.Config.Report == nil {
		http.NotFound(w, r)
		return
	}
	rc := view.Config.Report

	srcFieldByID := map[string]*model.Field{}
	if src, ok := h.interp.Get().GetMachine(rc.Machine); ok {
		srcFieldByID = fieldIndex(src)
	}
	sumLabels := make([]string, len(rc.SumFields))
	for i, f := range rc.SumFields {
		sumLabels[i] = f
		if sf, ok := srcFieldByID[f]; ok {
			sumLabels[i] = sf.Name
		}
	}

	groups, err := h.records.SumFieldsGroupedBy(r.Context(), rc.Machine, rc.GroupField, rc.SumFields)
	if err != nil {
		http.Error(w, "failed to load report", http.StatusInternalServerError)
		return
	}
	rows := make([]ui.ReportRow, len(groups))
	for i, g := range groups {
		sums := make([]string, len(rc.SumFields))
		for j, f := range rc.SumFields {
			sums[j] = fmt.Sprintf("%.2f", g.Sums[f])
		}
		rows[i] = ui.ReportRow{Group: g.Group, Sums: sums}
	}

	a := h.auth(r)
	page := ui.Report(h.workspaceName(r), a.User.Name, a.CSRFToken, h.isWorkspaceAdmin(r), machine, view.Name, sumLabels, rows, h.unreadCount(r.Context(), a), h.subNavFor(r, machine), h.viewNavFor(machineID, model.ViewTypeReport))
	if err := page.Render(r.Context(), w); err != nil {
		slog.Error("render report", "error", err)
	}
}

// calendarTimeline backs both Calendar and Timeline (CAP-V07) -- same
// grouped-by-date_field rendering, only the View lookup differs.
func (h *Handler) calendarTimeline(w http.ResponseWriter, r *http.Request, view *model.View) {
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
	if view == nil {
		http.NotFound(w, r)
		return
	}
	fieldByID := fieldIndex(machine)
	colIDs := view.Config.Columns
	cols := make([]ui.ColumnDef, 0, len(colIDs))
	for _, id := range colIDs {
		def := ui.ColumnDef{ID: id, Name: id}
		if f, ok := fieldByID[id]; ok {
			def.Name = f.Name
			def.Type = f.Type
		}
		cols = append(cols, def)
	}

	records, err := h.records.List(r.Context(), machineID, view.Config.DateField, "asc")
	if err != nil {
		http.Error(w, "failed to load records", http.StatusInternalServerError)
		return
	}

	a := h.auth(r)
	if view.Config.ResourceField == "" {
		groups := groupByDate(records, view.Config.DateField, colIDs, cols)
		page := ui.CalendarTimeline(h.workspaceName(r), a.User.Name, a.CSRFToken, h.isWorkspaceAdmin(r), machine, view.Name, cols, groups, h.unreadCount(r.Context(), a), h.subNavFor(r, machine), h.viewNavFor(machineID, view.Type))
		if err := page.Render(r.Context(), w); err != nil {
			slog.Error("render calendar/timeline", "error", err)
		}
		return
	}

	// CAP-V18: a second grouping dimension (resource) on top of the same
	// date_field grouping above -- every resource gets its own section,
	// including one with zero records for the date range shown (fetched
	// from the resource Machine directly, not derived from `records`,
	// which would silently drop an idle resource).
	resourceField := fieldByID[view.Config.ResourceField]
	byResource := map[string][]*store.Record{}
	var order []string
	if resourceField != nil {
		if resourceMachine, ok := h.interp.Get().GetMachine(resourceField.Options.TargetMachine); ok {
			resources, err := h.records.List(r.Context(), resourceMachine.ID, "", "")
			if err == nil {
				for _, rres := range resources {
					label := displayLabel(resourceMachine, rres.ID, rres.Data)
					order = append(order, label)
					byResource[label] = nil
				}
			}
		}
	}
	for _, rec := range records {
		label := "Unassigned"
		if resourceField != nil {
			if refID, _ := rec.Data[view.Config.ResourceField].(string); refID != "" {
				if l, err := h.referenceLabel(r.Context(), resourceField.Options.TargetMachine, refID); err == nil && l != "" {
					label = l
				}
			}
		}
		if _, seen := byResource[label]; !seen {
			order = append(order, label)
		}
		byResource[label] = append(byResource[label], rec)
	}
	sort.Strings(order)
	resGroups := make([]ui.ResourceCalendarGroup, 0, len(order))
	for _, label := range order {
		resGroups = append(resGroups, ui.ResourceCalendarGroup{
			Resource: label,
			Dates:    groupByDate(byResource[label], view.Config.DateField, colIDs, cols),
		})
	}

	page := ui.ResourceCalendarTimeline(h.workspaceName(r), a.User.Name, a.CSRFToken, h.isWorkspaceAdmin(r), machine, view.Name, cols, resGroups, h.unreadCount(r.Context(), a), h.subNavFor(r, machine), h.viewNavFor(machineID, view.Type))
	if err := page.Render(r.Context(), w); err != nil {
		slog.Error("render resource calendar/timeline", "error", err)
	}
}

// groupByDate (CAP-V07, extracted for CAP-V18's reuse) buckets records
// already sorted by dateField into consecutive same-date runs -- a manual
// "flush on change" pass over an already-sorted slice, not a SQL GROUP BY.
func groupByDate(records []*store.Record, dateField string, colIDs []string, cols []ui.ColumnDef) []ui.CalendarGroup {
	var groups []ui.CalendarGroup
	var cur string
	var curRows []ui.ListRow
	flush := func() {
		if curRows != nil {
			groups = append(groups, ui.CalendarGroup{Date: cur, Rows: curRows})
		}
	}
	first := true
	for _, rec := range records {
		date := fmt.Sprintf("%v", rec.Data[dateField])
		if date == "<nil>" {
			date = ""
		}
		if first || date != cur {
			flush()
			cur = date
			curRows = nil
			first = false
		}
		cells := make([]ui.ListCell, len(colIDs))
		for j, id := range colIDs {
			val := ""
			if v, ok := rec.Data[id]; ok {
				val = fmt.Sprintf("%v", v)
			}
			cells[j] = ui.ListCell{Value: val, IsStatusBadge: cols[j].Type == model.FieldTypeValueList}
		}
		curRows = append(curRows, ui.ListRow{ID: rec.ID, Cells: cells})
	}
	flush()
	return groups
}

// Calendar renders a CAP-V07 calendar View: records grouped by date_field.
func (h *Handler) Calendar(w http.ResponseWriter, r *http.Request) {
	h.calendarTimeline(w, r, h.interp.Get().CalendarView(chi.URLParam(r, "machineID")))
}

// Timeline renders a CAP-V07 timeline View: the same grouping, read
// chronologically.
func (h *Handler) Timeline(w http.ResponseWriter, r *http.Request) {
	h.calendarTimeline(w, r, h.interp.Get().TimelineView(chi.URLParam(r, "machineID")))
}

// Dashboard renders a CAP-V10 composed dashboard View -- one tile per
// declared Section, each possibly a DIFFERENT Machine than the one the
// dashboard View itself is declared on (the actual point of "composed").
// Each section's own read permission is checked independently -- a role
// that can't read a given section's Machine just doesn't get that tile,
// rather than the whole dashboard 403ing.
func (h *Handler) Dashboard(w http.ResponseWriter, r *http.Request) {
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
	view := h.interp.Get().DashboardView(machineID)
	if view == nil {
		http.NotFound(w, r)
		return
	}

	tiles := make([]ui.DashboardTile, 0, len(view.Config.Sections))
	for _, sec := range view.Config.Sections {
		secMachine, ok := h.interp.Get().GetMachine(sec.Machine)
		if !ok {
			continue
		}
		_, secAppID := h.interp.Get().ScopeFor(sec.Machine)
		secRole := h.roleForApp(r, secAppID)
		if !h.guard.CanRead(secMachine, secRole) {
			continue
		}
		counts, err := h.records.CountGroupedBy(r.Context(), sec.Machine, sec.GroupField)
		if err != nil {
			http.Error(w, "failed to load dashboard", http.StatusInternalServerError)
			return
		}
		tile := ui.DashboardTile{Title: sec.Title, MachineID: sec.Machine}
		for _, c := range counts {
			tile.Total += c.Count
			if sec.GroupField != "" {
				tile.Breakdown = append(tile.Breakdown, ui.DashboardBreakdown{Label: c.Group, Count: c.Count})
			}
		}
		tiles = append(tiles, tile)
	}

	a := h.auth(r)
	page := ui.Dashboard(h.workspaceName(r), a.User.Name, a.CSRFToken, h.isWorkspaceAdmin(r), view.Name, tiles, h.unreadCount(r.Context(), a), h.subNavFor(r, machine))
	if err := page.Render(r.Context(), w); err != nil {
		slog.Error("render dashboard", "error", err)
	}
}

// Board (CAP-V14 Tier 2) renders a kanban board View -- one lane per
// GroupField value_list option, sorted by sort_order within each lane (the
// same manual-order column CAP-V14's Up/Down buttons already use), so a
// card dropped into a lane by BoardMove renders where it landed. Every
// declared option gets a lane, even an empty one -- from GroupField's own
// Options.Values, not a distinct-values scan of the records (an option
// nobody's used yet still has to be a valid drop target).
func (h *Handler) Board(w http.ResponseWriter, r *http.Request) {
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
	view := h.interp.Get().BoardView(machineID)
	if view == nil || view.Config.GroupField == "" {
		http.NotFound(w, r)
		return
	}
	fieldByID := fieldIndex(machine)
	groupField, ok := fieldByID[view.Config.GroupField]
	if !ok {
		http.NotFound(w, r)
		return
	}

	hidden := h.hiddenFields(machine, role)
	colIDs := []string{}
	for _, id := range view.Config.Columns {
		if !hidden[id] {
			colIDs = append(colIDs, id)
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

	records, err := h.records.List(r.Context(), machineID, store.SortOrderField, "")
	if err != nil {
		http.Error(w, "failed to load records", http.StatusInternalServerError)
		return
	}

	byLane := make(map[string][]ui.ListRow, len(groupField.Options.Values))
	for _, rec := range records {
		laneVal := fmt.Sprintf("%v", rec.Data[view.Config.GroupField])
		cells := make([]ui.ListCell, len(colIDs))
		for j, id := range colIDs {
			val := ""
			if v, ok := rec.Data[id]; ok {
				val = fmt.Sprintf("%v", v)
			}
			cells[j] = ui.ListCell{Value: val}
		}
		byLane[laneVal] = append(byLane[laneVal], ui.ListRow{ID: rec.ID, Cells: cells})
	}

	lanes := make([]ui.BoardLane, 0, len(groupField.Options.Values))
	for _, v := range groupField.Options.Values {
		lanes = append(lanes, ui.BoardLane{Name: v, Rows: byLane[v]})
	}

	a := h.auth(r)
	page := ui.Board(h.workspaceName(r), a.User.Name, a.CSRFToken, h.isWorkspaceAdmin(r), machine, view.Name, view.Config.GroupField, cols, lanes, h.unreadCount(r.Context(), a), h.subNavFor(r, machine), h.viewNavFor(machineID, model.ViewTypeBoard))
	if err := page.Render(r.Context(), w); err != nil {
		slog.Error("render board", "error", err)
	}
}
