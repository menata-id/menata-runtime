package executor

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"menata.id/runtime/internal/model"
	"menata.id/runtime/internal/store"
)

type Executor struct {
	records       *store.RecordStore
	notifications *store.NotificationStore
}

func New(records *store.RecordStore, notifications *store.NotificationStore) *Executor {
	return &Executor{records: records, notifications: notifications}
}

// Simulate computes a record's data after applying event's set_field actions,
// without persisting anything. CAP-C09 validates constraints against this
// result before Persist commits it — a record must satisfy every Constraint
// after an event, not just at Create.
//
// actorIdentity resolves CAP-A02's "current_user" dynamic value. CAP-P02
// added a real identity concept distinct from role (the menata_identity
// cookie); "current_user" now resolves to that identity, falling back to
// actorRole when identity is empty (internal "System"-triggered events pass
// both as "System"). "today"/"now" are unaffected by either.
func (e *Executor) Simulate(event *model.Event, record *store.Record, actorRole, actorIdentity string) map[string]any {
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
			newData[field] = resolveValue(value, actorRole, actorIdentity)
		}
	}
	return newData
}

// resolveValue resolves CAP-A02 dynamic value tokens; any other string is a
// static literal, returned unchanged.
func resolveValue(value, actorRole, actorIdentity string) string {
	switch value {
	case "today":
		return time.Now().Format("2006-01-02")
	case "now":
		return time.Now().Format(time.RFC3339)
	case "current_user":
		if actorIdentity != "" {
			return actorIdentity
		}
		return actorRole
	default:
		return value
	}
}

// Persist saves newData (already validated by the caller — CAP-C09) as the
// record's new state, runs this event's non-data actions (notify, ...), and
// logs the event with the pre-event data as its snapshot. machineName is
// only used for a notification's message text — Executor still doesn't touch
// the Interpreter, this is just a display string the caller already has.
func (e *Executor) Persist(ctx context.Context, event *model.Event, record *store.Record, newData map[string]any, machineName string) error {
	snapshot := record.Data

	for _, action := range event.Actions {
		switch action.Type {
		case model.ActionNotify:
			e.doNotify(ctx, action, event, record, newData, machineName)

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

// doNotify implements CAP-A03 (static `role` recipient) and CAP-A04 (dynamic
// `recipient_field` recipient — the record's own field value, e.g. its
// Submitted By, rather than a whole role). recipient_field wins when its
// resolved value is non-empty; role is the fallback (and the only option
// most existing metadata declares). CAP-A10 in-app delivery: the resolved
// recipient is written as a real Notification row a matching role-cookie
// session can list and mark read, not just logged.
func (e *Executor) doNotify(ctx context.Context, action *model.EventAction, event *model.Event, record *store.Record, newData map[string]any, machineName string) {
	role, _ := action.Params["role"].(string)
	recipientFieldID, _ := action.Params["recipient_field"].(string)

	recipient := role
	if recipientFieldID != "" {
		if v, ok := newData[recipientFieldID]; ok {
			if s := fmt.Sprintf("%v", v); s != "" {
				recipient = s
			}
		}
	}
	if recipient == "" {
		return
	}

	slog.Info("notify", "event", event.ID, "recipient", recipient, "record", record.ID)

	if e.notifications == nil {
		return
	}
	message := fmt.Sprintf("%s: %s", machineName, event.Name)
	if err := e.notifications.Create(ctx, recipient, message, record.MachineID, record.ID); err != nil {
		slog.Error("create notification", "event", event.ID, "recipient", recipient, "error", err)
	}
}
