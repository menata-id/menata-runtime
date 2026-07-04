package executor

import (
	"context"
	"log/slog"

	"menata.id/runtime/internal/model"
	"menata.id/runtime/internal/store"
)

type Executor struct {
	records *store.RecordStore
}

func New(records *store.RecordStore) *Executor {
	return &Executor{records: records}
}

// Apply runs all actions of an event on a record, saves the update, and logs the event.
func (e *Executor) Apply(ctx context.Context, event *model.Event, record *store.Record) error {
	// capture snapshot before mutation
	snapshot := make(map[string]any, len(record.Data))
	for k, v := range record.Data {
		snapshot[k] = v
	}

	for _, action := range event.Actions {
		switch action.Type {
		case model.ActionSetField:
			field, _ := action.Params["field"].(string)
			value, _ := action.Params["value"].(string)
			if field != "" {
				record.Data[field] = value
			}

		case model.ActionNotify:
			role, _ := action.Params["role"].(string)
			slog.Info("notify (prototype: logged only)",
				"event", event.ID, "role", role, "record", record.ID)

		case model.ActionCreateRecord:
			slog.Info("create_record action (prototype: not yet implemented)",
				"event", event.ID)
		}
	}

	if err := e.records.Update(ctx, record.ID, record.Data); err != nil {
		return err
	}
	return e.records.LogEvent(ctx, record.ID, event.ID, snapshot)
}
