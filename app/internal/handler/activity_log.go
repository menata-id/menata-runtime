package handler

import (
	"log/slog"
	"net/http"

	"menata.id/app/internal/model"
	"menata.id/app/internal/store"
	"menata.id/app/internal/ui"
)

// activityMetaFormat is the pre-formatted timestamp string every
// ActivityItem.Meta uses -- one place to change it, matching this
// codebase's own "format once, reuse" precedent for display strings.
const activityMetaFormat = "Jan 2, 2006 15:04"

// activityFeedLimit caps the cross-record mode's own query -- an
// unbounded whole-Machine feed would grow without limit; a record-scoped
// feed (bounded by that one record's own real event count) needs none.
const activityFeedLimit = 10

// eventNameByID resolves machine's own declared Event ids to their Name
// -- a plain in-memory lookup, no store call, shared by both render
// modes below (Gate 3: keeps each caller's own complexity small).
func eventNameByID(machine *model.Machine, eventID string) string {
	for _, e := range machine.Events {
		if e.ID == eventID {
			return e.Name
		}
	}
	return eventID // unknown event id (live drift) -- show the raw id rather than nothing
}

// renderActivityLogChild (CAP-R04 "R28", composable-runtime-roadmap.md
// 17p) is the record-scoped mode: hostRec's own event history, most
// recent first. Embedded via CAP-V20's existing renderChildView dispatch
// (embed.go) -- what makes this mode "record-scoped" is simply that a
// real host record already exists at this call site, not any Config
// flag on view itself.
func (h *Handler) renderActivityLogChild(r *http.Request, hostRec *store.Record, view *model.View) *ui.EmbeddedSection {
	machine, ok := h.interp.Get().GetMachine(hostRec.MachineID)
	if !ok {
		slog.Warn("activity_log child embed: host record's own machine_id does not resolve", "record", hostRec.ID, "machine_id", hostRec.MachineID)
		return nil
	}
	events, err := h.records.ListEventsForRecord(r.Context(), hostRec.ID)
	if err != nil {
		slog.Warn("activity_log child embed: list events", "record", hostRec.ID, "error", err)
		return nil
	}
	items := make([]ui.ActivityItem, len(events))
	for i, e := range events {
		items[i] = ui.ActivityItem{Actor: e.PerformedBy, Line: eventNameByID(machine, e.EventID), Meta: e.PerformedAt.Format(activityMetaFormat)}
	}
	return &ui.EmbeddedSection{Title: view.Name, Content: ui.ActivityLogSection(items)}
}

// renderPageActivityLogChild (17p) is the cross-record mode: the most
// recent events across EVERY record of view's own MachineID, most
// recent first, capped at activityFeedLimit -- the real thing closing
// vw_ad_page's own "Recent Activity requires..." placeholder (17k).
// Embedded via CAP-V10 Tier 2's existing renderPageChild dispatch
// (page.go) -- what makes this mode "cross-record" is simply that no
// host record exists at this call site.
func (h *Handler) renderPageActivityLogChild(r *http.Request, view *model.View, title, layout string) *ui.PageSection {
	machine, ok := h.interp.Get().GetMachine(view.MachineID)
	if !ok {
		slog.Warn("activity_log page child: view's own machine_id does not resolve", "view", view.ID, "machine_id", view.MachineID)
		return nil
	}
	events, err := h.records.ListEventsForMachine(r.Context(), view.MachineID, activityFeedLimit)
	if err != nil {
		slog.Warn("activity_log page child: list events", "machine_id", view.MachineID, "error", err)
		return nil
	}
	items := make([]ui.ActivityItem, 0, len(events))
	for _, e := range events {
		items = append(items, h.activityItemForCrossRecordEvent(r, machine, e))
	}
	return &ui.PageSection{Title: title, Layout: layout, Content: ui.ActivityLogSection(items)}
}

// activityItemForCrossRecordEvent resolves one cross-record event's own
// owning record's display label, folding it into Line (the mockup's own
// "actor verb object" shape: "{actor} {Event Name} {record label}").
// Split out of renderPageActivityLogChild (Gate 3).
func (h *Handler) activityItemForCrossRecordEvent(r *http.Request, machine *model.Machine, e store.RecordEvent) ui.ActivityItem {
	line := eventNameByID(machine, e.EventID)
	if rec, err := h.records.Get(r.Context(), e.RecordID); err == nil {
		line = line + " " + displayLabel(machine, rec.ID, rec.Data)
	}
	return ui.ActivityItem{Actor: e.PerformedBy, Line: line, Meta: e.PerformedAt.Format(activityMetaFormat)}
}
