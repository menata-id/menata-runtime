package executor

import (
	"context"
	"log/slog"
	"time"

	"menata.id/runtime/internal/model"
	"menata.id/runtime/internal/store"
)

type Executor struct {
	records *store.RecordStore
}

func New(records *store.RecordStore) *Executor {
	return &Executor{records: records}
}

// Simulate computes a record's data after applying event's set_field actions,
// without persisting anything. CAP-C09 validates constraints against this
// result before Persist commits it — a record must satisfy every Constraint
// after an event, not just at Create.
//
// actorRole resolves CAP-A02's dynamic values: "today", "now", and
// "current_user". This prototype has no real per-user session — role
// selection (the login cookie) is the only identity concept that exists —
// so current_user resolves to the acting role, not a person. Real identity
// is CAP-O01/CAP-F05 territory, not part of this capability.
func (e *Executor) Simulate(event *model.Event, record *store.Record, actorRole string) map[string]any {
	newData := make(map[string]any, len(record.Data))
	for k, v := range record.Data {
		newData[k] = v
	}
	for _, action := range event.Actions {
		if action.Type != model.ActionSetField {
			continue
		}
		field, _ := action.Params["field"].(string)
		value, _ := action.Params["value"].(string)
		if field != "" {
			newData[field] = resolveValue(value, actorRole)
		}
	}
	return newData
}

// resolveValue resolves CAP-A02 dynamic value tokens; any other string is a
// static literal, returned unchanged.
func resolveValue(value, actorRole string) string {
	switch value {
	case "today":
		return time.Now().Format("2006-01-02")
	case "now":
		return time.Now().Format(time.RFC3339)
	case "current_user":
		return actorRole
	default:
		return value
	}
}

// Persist saves newData (already validated by the caller — CAP-C09) as the
// record's new state, runs this event's non-data actions (notify, ...), and
// logs the event with the pre-event data as its snapshot.
func (e *Executor) Persist(ctx context.Context, event *model.Event, record *store.Record, newData map[string]any) error {
	snapshot := record.Data

	for _, action := range event.Actions {
		switch action.Type {
		case model.ActionNotify:
			role, _ := action.Params["role"].(string)
			slog.Info("notify (prototype: logged only)",
				"event", event.ID, "role", role, "record", record.ID)

		case model.ActionCreateRecord:
			slog.Info("create_record action (prototype: not yet implemented)",
				"event", event.ID)
		}
	}

	if err := e.records.Update(ctx, record.ID, newData); err != nil {
		return err
	}
	return e.records.LogEvent(ctx, record.ID, event.ID, snapshot)
}
