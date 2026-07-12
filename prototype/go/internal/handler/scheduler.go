package handler

import (
	"context"
	"log/slog"
	"time"

	"menata.id/runtime/internal/model"
)

// RunScheduledEvents (CAP-E02/E03) sweeps every Event on machines scoped to
// workspaceID whose own Schedule is due right now, firing it (actorRole
// "System", same convention CAP-A08/CAP-E05's internal cascades already
// use) on every record that qualifies. Called once per tick per workspace
// by cmd/server/main.go's scheduler loop, inside that same call's own
// workspace-scoped transaction (SET LOCAL app.workspace_id) -- this
// function itself has no transaction/workspace concerns of its own, same
// division of responsibility as every other Handler method.
//
// CAP-E02 (time-driven, "Every Day 08:00"): fires on every record on the
// Machine once per day, at or after the declared HH:MM. CAP-E03
// (date-driven, "When Due Date - 1 Day"): fires per-record, once per day,
// when today equals that record's own date Field plus the declared
// offset. Both de-duplicate via RecordStore.FiredToday (CAP-R04's
// existing audit trail), not a new tracking table -- a tick that runs
// more than once a day must not fire the same record twice.
func (h *Handler) RunScheduledEvents(ctx context.Context, workspaceID string) {
	now := time.Now().UTC()
	for _, m := range h.interp.AllMachines() {
		ws, _ := h.interp.ScopeFor(m.ID)
		if ws != workspaceID {
			continue
		}
		for _, e := range m.Events {
			if e.Schedule == nil {
				continue
			}
			if !h.scheduleWindowOpen(e.Schedule, now) {
				continue
			}
			h.fireScheduledEvent(ctx, m, e, now)
		}
	}
}

// scheduleWindowOpen (CAP-E02) is the fixed-time half: true once today's
// UTC clock has reached the declared HH:MM. CAP-E03 (date_field set
// instead) has no single fixed time -- always "open," gated per-record
// inside fireScheduledEvent instead, by that record's own date Field.
func (h *Handler) scheduleWindowOpen(s *model.Schedule, now time.Time) bool {
	if s.DateField != "" {
		return true
	}
	scheduled, err := time.Parse("15:04", s.Time)
	if err != nil {
		return false
	}
	nowMinutes := now.Hour()*60 + now.Minute()
	scheduledMinutes := scheduled.Hour()*60 + scheduled.Minute()
	return nowMinutes >= scheduledMinutes
}

func (h *Handler) fireScheduledEvent(ctx context.Context, machine *model.Machine, event *model.Event, now time.Time) {
	records, err := h.records.List(ctx, machine.ID, "", "")
	if err != nil {
		slog.Error("scheduled event: list records", "event", event.ID, "error", err)
		return
	}
	today := now.Format("2006-01-02")
	for _, rec := range records {
		if event.Schedule.DateField != "" && !dateFieldDue(event.Schedule, rec.Data, today) {
			continue
		}
		fired, err := h.records.FiredToday(ctx, rec.ID, event.ID)
		if err != nil {
			slog.Error("scheduled event: check fired today", "event", event.ID, "record", rec.ID, "error", err)
			continue
		}
		if fired {
			continue
		}
		if err := h.triggerEvent(ctx, machine, event, rec, "System", "Scheduler", "", nil); err != nil {
			// A rule violation (e.g. CAP-E06's state guard) just means this
			// record doesn't currently qualify -- same "skip, don't fail
			// the sweep" posture as any other record in the batch failing
			// its own Constraints would get.
			continue
		}
	}
}

// dateFieldDue (CAP-E03) reports whether today is exactly
// data[field] + offsetDays -- "When Due Date - 1 Day" fires the one day
// this equation holds, not every day from then on.
func dateFieldDue(s *model.Schedule, data map[string]any, today string) bool {
	raw, ok := data[s.DateField]
	if !ok {
		return false
	}
	str, ok := raw.(string)
	if !ok || str == "" {
		return false
	}
	base, err := time.Parse("2006-01-02", str)
	if err != nil {
		return false
	}
	return base.AddDate(0, 0, s.OffsetDays).Format("2006-01-02") == today
}
