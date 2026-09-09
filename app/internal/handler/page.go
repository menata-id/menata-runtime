package handler

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"menata.id/app/internal/model"
	"menata.id/app/internal/store"
	"menata.id/app/internal/ui"
)

// pageListSectionLimit caps an embedded `list` section to its first few
// rows -- a composed page's own section is a summary linking to the full
// list ("View all →"), not a second full interactive list nested inside
// someone else's page (no search/export/New/pagination there either, see
// ui.ListContent's own doc comment). Matches approval-dashboard.html's own
// mockup, which shows exactly 3 rows in its "Pending Documents" section.
const pageListSectionLimit = 8

// Page (CAP-V10 Tier 2) renders a `page` View's entire body, composed
// entirely from its own declared Config.Children -- see model.
// ViewTypePage's own doc comment for the full reasoning, and embed.go's
// renderEmbeddedViews doc comment for why this is the collection-level
// sibling of that record-level mechanism, sharing the same Children field.
func (h *Handler) Page(w http.ResponseWriter, r *http.Request) {
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
	view := h.interp.Get().PageView(machineID)
	if view == nil {
		http.NotFound(w, r)
		return
	}

	var sections []ui.PageSection
	for _, child := range view.Config.Children {
		if sec := h.renderPageChild(r, child); sec != nil {
			sections = append(sections, *sec)
		}
	}

	a := h.auth(r)
	page := ui.ComposedPage(h.workspaceName(r), h.workspaceSlug(r), a.User.Name, a.CSRFToken, h.isWorkspaceAdmin(r), view.Name, buildPageRows(sections), h.unreadCount(r.Context(), a), h.subNavFor(r, machine))
	if err := page.Render(r.Context(), w); err != nil {
		slog.Error("render page", "error", err)
	}
}

// buildPageRows (CAP-V10 Tier 2) pairs two consecutive sections declaring
// Layout "main" then "aside" into one PageRow (a 2/3+1/3 grid row),
// stacking everything else full width in order -- see ui.PageRow's own
// doc comment for why this pairing lives here, not in the template.
func buildPageRows(sections []ui.PageSection) []ui.PageRow {
	var rows []ui.PageRow
	for i := 0; i < len(sections); i++ {
		if i+1 < len(sections) && sections[i].Layout == "main" && sections[i+1].Layout == "aside" {
			aside := sections[i+1]
			rows = append(rows, ui.PageRow{Main: sections[i], Aside: &aside})
			i++
			continue
		}
		rows = append(rows, ui.PageRow{Main: sections[i]})
	}
	return rows
}

// renderPageChild (CAP-V10 Tier 2) is the dispatch this whole View Type
// exists for: a static-content entry (Content set) renders directly; a
// View entry is resolved and dispatched by ITS OWN Type -- never by what
// the host page expected -- the same principle embed.go's renderChildView
// already established for CAP-V20 Tier 2's record-level Children, applied
// here to collection-level composition instead. model.
// PageEmbeddableViewTypes is the load-time half of this contract
// (metadata/validate.go) -- keep both in sync.
func (h *Handler) renderPageChild(r *http.Request, child model.ChildViewRef) *ui.PageSection {
	if child.Content != nil {
		return &ui.PageSection{Title: child.Title, Layout: child.Layout, Content: ui.StaticContent(child.Content)}
	}
	view, ok := h.interp.Get().GetView(child.View)
	if !ok {
		// Same live-drift class embed.go's own renderChildView already
		// names: load-time validation guarantees this against the metadata
		// that was actually loaded, not against a database that has since
		// drifted underneath an already-running process.
		slog.Warn("page: declared children view id does not resolve against loaded metadata", "children_view", child.View)
		return nil
	}
	title := child.Title
	if title == "" {
		title = view.Name
	}
	switch view.Type {
	case model.ViewTypeList:
		return h.renderPageListChild(r, view, title, child.Layout)
	case model.ViewTypeDashboard:
		return h.renderPageDashboardChild(r, view, title, child.Layout)
	default:
		slog.Warn("page: declared children view is not a Type a page can compose", "children_view", child.View, "type", view.Type)
		return nil
	}
}

// renderPageListChild (CAP-V10 Tier 2) builds an embedded `list` section --
// same column/row/filter logic record_crud.go's List uses for the
// standalone route (buildListRows/applyListFilter, shared so the two can
// never render a different result for the same View), capped to
// pageListSectionLimit rows with a "View all →" link to the real thing.
// Permission-checked against the CHILD View's own Machine/role, not the
// host page's -- a page composing a View from a different Machine must
// still respect that Machine's own CAP-P05 read gate.
func (h *Handler) renderPageListChild(r *http.Request, view *model.View, title, layout string) *ui.PageSection {
	machine, ok := h.interp.Get().GetMachine(view.MachineID)
	if !ok {
		return nil
	}
	_, appID := h.interp.Get().ScopeFor(view.MachineID)
	role := h.roleForApp(r, appID)
	if !h.guard.CanRead(machine, role) {
		return nil
	}
	fieldByID := fieldIndex(machine)
	hidden := h.hiddenFields(machine, role)
	colIDs := make([]string, 0, len(view.Config.Columns))
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

	sortField, sortDir := "", ""
	if view.Config.ManualOrder {
		sortField = store.SortOrderField
	} else if view.Config.DefaultSort != nil {
		sortField, sortDir = view.Config.DefaultSort.Field, view.Config.DefaultSort.Direction
	}
	records, err := h.records.List(r.Context(), view.MachineID, sortField, sortDir)
	if err != nil {
		return nil
	}
	records = h.applyListFilter(r, view, records)
	if len(records) > pageListSectionLimit {
		records = records[:pageListSectionLimit]
	}
	rows := h.buildListRows(r, cols, colIDs, fieldByID, view, records)

	return &ui.PageSection{
		Title:       title,
		Layout:      layout,
		ActionLabel: "View all →",
		ActionHref:  "/" + h.workspaceSlug(r) + "/" + view.MachineID,
		Content:     ui.ListContent(h.workspaceSlug(r), view.MachineID, cols, rows, view.Config.Display == "cards"),
	}
}

// renderPageDashboardChild (CAP-V10 Tier 2) builds an embedded `dashboard`
// section -- same tile computation record_crud.go's Dashboard uses for the
// standalone route (buildDashboardTiles, shared for the same "one result"
// reason as the list case above).
func (h *Handler) renderPageDashboardChild(r *http.Request, view *model.View, title, layout string) *ui.PageSection {
	tiles, err := h.buildDashboardTiles(r, view)
	if err != nil {
		return nil
	}
	return &ui.PageSection{
		Title:   title,
		Layout:  layout,
		Content: ui.DashboardContent(h.workspaceSlug(r), tiles),
	}
}
