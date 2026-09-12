package handler

import (
	"log/slog"
	"net/http"

	"menata.id/app/internal/composable"
	"menata.id/app/internal/model"
	"menata.id/app/internal/store"
	"menata.id/app/internal/ui"
)

// boardViaComposable (composable-runtime-roadmap.md 17n) renders the real
// production Board route for a reference-typed group_field (CAP-V14
// Tier 3, dynamic lanes) through internal/composable -- the same
// composable.ResolveBoardLanes (17h) already proven equivalent on the
// additive /composable-preview route, now driving the real page. Split
// into its own file rather than views.go (already at its own Gate 2
// baseline) or composable_preview.go (398 lines).
//
// Three real constraints found while grounding, all preserved here:
//   - CAP-V14 Tier 2 (value_list group_field, e.g. Kanban Lab's own
//     vw_kbt_board) is NOT handled here -- views.go's own dispatch only
//     calls this for a reference-typed group_field; ResolveBoardLanes has
//     no Tier 2 support, by design (see its own doc comment).
//   - The real board.templ has drag-and-drop; composable_preview.templ's
//     own simplified card rendering deliberately does not. This function
//     renders through the SAME ui.Board template the existing handler
//     already uses -- only where the lane/card data comes from changes.
//   - CAP-P06 hidden-field filtering: each cell is resolved individually
//     via composable.ResolveFieldValue against this role's own
//     hidden-filtered colIDs, never by indexing into a Dataset's raw,
//     unfiltered Projection.Fields (which also carries the group field
//     itself when it isn't already a declared column).
func (h *Handler) boardViaComposable(w http.ResponseWriter, r *http.Request, machine *model.Machine, applicationID string, role []string, view *model.View, groupField *model.Field) {
	hidden := h.hiddenFields(machine, role)
	colIDs, cols := visibleBoardColumns(machine, view, hidden)

	node, err := composable.LowerViewToComponent(machine, view)
	if err != nil {
		http.Error(w, "failed to lower board component", http.StatusInternalServerError)
		return
	}
	laneMachine, ok := h.interp.Get().GetMachine(groupField.Options.TargetMachine)
	if !ok {
		http.Error(w, "board view's target machine not found", http.StatusInternalServerError)
		return
	}
	laneRecs, err := h.records.List(r.Context(), groupField.Options.TargetMachine, store.SortOrderField, "")
	if err != nil {
		http.Error(w, "failed to load board lanes", http.StatusInternalServerError)
		return
	}
	cardRecs, err := h.records.List(r.Context(), machine.ID, store.SortOrderField, "")
	if err != nil {
		http.Error(w, "failed to load records", http.StatusInternalServerError)
		return
	}

	composableLanes, err := composable.ResolveBoardLanes(machine, node, toRecordRefs(cardRecs), toRecordRefs(laneRecs))
	if err != nil {
		http.Error(w, "failed to resolve board lanes", http.StatusInternalServerError)
		return
	}
	lanes, err := buildComposableBoardLanes(machine, composableLanes, cardRecs, laneRecs, laneMachine, colIDs)
	if err != nil {
		http.Error(w, "failed to build board lanes", http.StatusInternalServerError)
		return
	}

	planExplain, _ := h.explainComposablePlan(r, applicationID, machine, role[0])

	a := h.auth(r)
	page := ui.Board(h.workspaceName(r), h.workspaceSlug(r), a.User.Name, a.CSRFToken, h.isWorkspaceAdmin(r), machine, view.Name, view.Config.GroupField, cols, lanes, h.unreadCount(r.Context(), a), h.subNavFor(r, machine), h.viewNavFor(h.workspaceSlug(r), machine.ID, model.ViewTypeBoard), planExplain)
	if err := page.Render(r.Context(), w); err != nil {
		slog.Error("render board", "error", err)
	}
}

// visibleBoardColumns computes colIDs (view.Config.Columns minus this
// role's own hidden fields, CAP-P06) and their matching ui.ColumnDef --
// the exact same shape views.go's own value_list Board branch already
// builds. Split out of boardViaComposable itself (Gate 3).
func visibleBoardColumns(machine *model.Machine, view *model.View, hidden map[string]bool) ([]string, []ui.ColumnDef) {
	colIDs := make([]string, 0, len(view.Config.Columns))
	for _, id := range view.Config.Columns {
		if !hidden[id] {
			colIDs = append(colIDs, id)
		}
	}
	fieldByID := fieldIndex(machine)
	cols := make([]ui.ColumnDef, 0, len(colIDs))
	for _, id := range colIDs {
		def := ui.ColumnDef{ID: id, Name: id}
		if f, ok := fieldByID[id]; ok {
			def.Name = f.Name
			def.Type = f.Type
		}
		cols = append(cols, def)
	}
	return colIDs, cols
}

// buildComposableBoardLanes converts composable's own lane grouping
// (RecordID-only) into the real ui.BoardLane/ui.ListRow shape the
// existing board.templ already renders -- resolving each real, already-
// hidden-filtered colID via composable.ResolveFieldValue against the
// ORIGINAL record data, never via a Dataset's own unfiltered Projection.
// Split out of boardViaComposable itself (Gate 3: keeps its own
// complexity small).
func buildComposableBoardLanes(machine *model.Machine, composableLanes []composable.BoardLane, cardRecs, laneRecs []*store.Record, laneMachine *model.Machine, colIDs []string) ([]ui.BoardLane, error) {
	cardByID := make(map[string]*store.Record, len(cardRecs))
	for _, rec := range cardRecs {
		cardByID[rec.ID] = rec
	}
	laneByID := make(map[string]*store.Record, len(laneRecs))
	for _, rec := range laneRecs {
		laneByID[rec.ID] = rec
	}

	lanes := make([]ui.BoardLane, 0, len(composableLanes))
	for _, cl := range composableLanes {
		rows, err := buildComposableBoardRows(machine, cl, cardByID, colIDs)
		if err != nil {
			return nil, err
		}
		laneName := cl.LaneRecordID
		if laneRec, ok := laneByID[cl.LaneRecordID]; ok {
			laneName = displayLabel(laneMachine, laneRec.ID, laneRec.Data)
		}
		lanes = append(lanes, ui.BoardLane{ID: cl.LaneRecordID, Name: laneName, Rows: rows})
	}
	return lanes, nil
}

// buildComposableBoardRows resolves one lane's own real cards into
// ui.ListRow, each Cell in colIDs order. Split out of
// buildComposableBoardLanes (Gate 3).
func buildComposableBoardRows(machine *model.Machine, lane composable.BoardLane, cardByID map[string]*store.Record, colIDs []string) ([]ui.ListRow, error) {
	rows := make([]ui.ListRow, 0, len(lane.Cards))
	for _, card := range lane.Cards {
		rec, ok := cardByID[card.RecordID]
		if !ok {
			continue
		}
		cells := make([]ui.ListCell, len(colIDs))
		for i, colID := range colIDs {
			fv, err := composable.ResolveFieldValue(machine, colID, rec.Data)
			if err != nil {
				return nil, err
			}
			cells[i] = ui.ListCell{Value: fv.Display}
		}
		rows = append(rows, ui.ListRow{ID: rec.ID, Cells: cells})
	}
	return rows, nil
}
