package handler

import (
	"context"
	"fmt"
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
	cardMeta := h.buildBoardCardMeta(r.Context(), machine, view.Config.CardMeta, cardRecs)
	lanes, err := buildComposableBoardLanes(machine, composableLanes, cardRecs, laneRecs, laneMachine, colIDs, cardMeta)
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
func buildComposableBoardLanes(machine *model.Machine, composableLanes []composable.BoardLane, cardRecs, laneRecs []*store.Record, laneMachine *model.Machine, colIDs []string, cardMeta map[string]ui.BoardCardMeta) ([]ui.BoardLane, error) {
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
		var meta []ui.BoardCardMeta
		if cardMeta != nil {
			// 17q: CardMeta must line up with Rows by index (board.templ
			// zips them) -- cardMeta is nil only when the View declares no
			// CardMeta config at all, so every existing Board (which never
			// sets it) skips this entirely, zero behavior change.
			meta = make([]ui.BoardCardMeta, len(rows))
			for i, row := range rows {
				meta[i] = cardMeta[row.ID]
			}
		}
		lanes = append(lanes, ui.BoardLane{ID: cl.LaneRecordID, Name: laneName, Rows: rows, CardMeta: meta})
	}
	return lanes, nil
}

// buildBoardCardMeta (17q) computes every card's own opt-in Board metadata
// (label chips, member names, checklist progress, due date) in three
// independent, cheap passes, keyed by card record id -- nil (zero extra
// queries) when the View declares no CardMeta config, so every existing
// Board is untouched. Each piece composes entirely from already-supported
// Grammar (reference/user/boolean/date Fields, CAP-O02 master-data,
// the same reverse-reference discovery CAP-V06's own childLists already
// uses) -- see model.BoardCardMetaConfig's own doc comment.
func (h *Handler) buildBoardCardMeta(ctx context.Context, machine *model.Machine, cfg *model.BoardCardMetaConfig, cardRecs []*store.Record) map[string]ui.BoardCardMeta {
	if cfg == nil {
		return nil
	}
	meta := make(map[string]ui.BoardCardMeta, len(cardRecs))
	if cfg.LabelsMachine != "" {
		for cardID, chips := range h.boardCardLabels(ctx, machine.ID, cfg) {
			m := meta[cardID]
			m.Labels = chips
			meta[cardID] = m
		}
	}
	if cfg.MembersMachine != "" {
		for cardID, names := range h.boardCardMembers(ctx, machine.ID, cfg) {
			m := meta[cardID]
			m.MemberNames = names
			meta[cardID] = m
		}
	}
	if cfg.ProgressMachine != "" {
		for cardID, p := range h.boardCardProgress(ctx, machine.ID, cfg) {
			m := meta[cardID]
			m.Progress = p
			meta[cardID] = m
		}
	}
	if cfg.DueDateField != "" {
		mergeBoardDueDates(meta, cfg.DueDateField, cardRecs)
	}
	return meta
}

// mergeBoardDueDates (17q) is buildBoardCardMeta's own due-date pass,
// split out to keep that function's own branching under Gate 3's
// threshold -- a plain Field read on the Board's own Machine, no
// reverse-reference hop needed.
func mergeBoardDueDates(meta map[string]ui.BoardCardMeta, dueDateField string, cardRecs []*store.Record) {
	for _, rec := range cardRecs {
		v, ok := rec.Data[dueDateField]
		if !ok {
			continue
		}
		s := fmt.Sprintf("%v", v)
		if s == "" {
			continue
		}
		m := meta[rec.ID]
		m.DueDate = s
		meta[rec.ID] = m
	}
}

// boardReverseParentField finds joinMachineID's own `reference` field that
// targets targetMachineID -- the same "found by scanning, never asked for
// explicitly" discovery CAP-V06's own childLists already uses, applied
// here instead of adding a redundant explicit config key.
func (h *Handler) boardReverseParentField(joinMachineID, targetMachineID string) (string, bool) {
	jm, ok := h.interp.Get().GetMachine(joinMachineID)
	if !ok {
		return "", false
	}
	for _, f := range jm.Fields {
		if f.Type == model.FieldTypeReference && f.Options.TargetMachine == targetMachineID {
			return f.ID, true
		}
	}
	return "", false
}

// boardCardLabels (17q) reads every LabelsMachine join row, resolving each
// one's own LabelsRefField to the real Label record's Name/Color -- the
// Label Machine itself is never named in config, it's read off
// LabelsRefField's own declared TargetMachine (the same reference-field
// metadata CAP-F13 already validates at load time).
func (h *Handler) boardCardLabels(ctx context.Context, cardMachineID string, cfg *model.BoardCardMetaConfig) map[string][]ui.BoardLabelChip {
	parentField, ok := h.boardReverseParentField(cfg.LabelsMachine, cardMachineID)
	if !ok {
		return nil
	}
	labelByID := h.boardLabelRecordsByID(ctx, cfg)
	if labelByID == nil {
		return nil
	}
	joinRecs, err := h.records.List(ctx, cfg.LabelsMachine, "", "")
	if err != nil {
		slog.Error("list board label joins", "machine", cfg.LabelsMachine, "error", err)
		return nil
	}
	out := make(map[string][]ui.BoardLabelChip)
	for _, join := range joinRecs {
		cardID, _ := join.Data[parentField].(string)
		labelID, _ := join.Data[cfg.LabelsRefField].(string)
		if chip, ok := labelByID[labelID]; ok && cardID != "" {
			out[cardID] = append(out[cardID], chip)
		}
	}
	return out
}

// boardLabelRecordsByID (17q) resolves LabelsRefField's own declared
// TargetMachine (never named separately in config) and reads every real
// Label record on it into a Name/Color lookup -- split out of
// boardCardLabels itself (Gate 3: keeps its own branching under
// threshold).
func (h *Handler) boardLabelRecordsByID(ctx context.Context, cfg *model.BoardCardMetaConfig) map[string]ui.BoardLabelChip {
	joinMachine, ok := h.interp.Get().GetMachine(cfg.LabelsMachine)
	if !ok {
		return nil
	}
	refField, ok := fieldIndex(joinMachine)[cfg.LabelsRefField]
	if !ok || refField.Type != model.FieldTypeReference {
		return nil
	}
	labelMachine, ok := h.interp.Get().GetMachine(refField.Options.TargetMachine)
	if !ok {
		return nil
	}
	labelRecs, err := h.records.List(ctx, labelMachine.ID, "", "")
	if err != nil {
		slog.Error("list board label records", "machine", labelMachine.ID, "error", err)
		return nil
	}
	labelByID := make(map[string]ui.BoardLabelChip, len(labelRecs))
	for _, rec := range labelRecs {
		labelByID[rec.ID] = ui.BoardLabelChip{
			Name:  fmt.Sprintf("%v", rec.Data[cfg.LabelsNameField]),
			Color: fmt.Sprintf("%v", rec.Data[cfg.LabelsColorField]),
		}
	}
	return labelByID
}

// boardCardMembers (17q) reads every MembersMachine join row's own
// MembersUserField, resolving each user id to a real display name via
// userLabel (CAP-F05) -- initials are computed later, inside internal/ui,
// same division of labor admin.templ's own AvatarStack call site already
// established (memberInitials/initials()).
func (h *Handler) boardCardMembers(ctx context.Context, cardMachineID string, cfg *model.BoardCardMetaConfig) map[string][]string {
	parentField, ok := h.boardReverseParentField(cfg.MembersMachine, cardMachineID)
	if !ok {
		return nil
	}
	joinRecs, err := h.records.List(ctx, cfg.MembersMachine, "", "")
	if err != nil {
		slog.Error("list board member joins", "machine", cfg.MembersMachine, "error", err)
		return nil
	}
	out := make(map[string][]string)
	for _, join := range joinRecs {
		cardID, _ := join.Data[parentField].(string)
		userID, _ := join.Data[cfg.MembersUserField].(string)
		if cardID == "" || userID == "" {
			continue
		}
		name, err := h.userLabel(ctx, userID)
		if err != nil {
			continue
		}
		out[cardID] = append(out[cardID], name)
	}
	return out
}

// boardCardProgress (17q) counts ProgressMachine's own child rows per
// card, rendering "done/total" from ProgressDoneField -- the same real
// child rows CAP-F16's own Checklist mechanism already creates, just
// counted rather than rendered as a full list.
func (h *Handler) boardCardProgress(ctx context.Context, cardMachineID string, cfg *model.BoardCardMetaConfig) map[string]string {
	parentField, ok := h.boardReverseParentField(cfg.ProgressMachine, cardMachineID)
	if !ok {
		return nil
	}
	childRecs, err := h.records.List(ctx, cfg.ProgressMachine, "", "")
	if err != nil {
		slog.Error("list board progress children", "machine", cfg.ProgressMachine, "error", err)
		return nil
	}
	type counts struct{ done, total int }
	byCard := make(map[string]*counts)
	for _, rec := range childRecs {
		cardID, _ := rec.Data[parentField].(string)
		if cardID == "" {
			continue
		}
		c, ok := byCard[cardID]
		if !ok {
			c = &counts{}
			byCard[cardID] = c
		}
		c.total++
		if fmt.Sprintf("%v", rec.Data[cfg.ProgressDoneField]) == "true" {
			c.done++
		}
	}
	out := make(map[string]string, len(byCard))
	for cardID, c := range byCard {
		if c.total > 0 {
			out[cardID] = fmt.Sprintf("%d/%d", c.done, c.total)
		}
	}
	return out
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
