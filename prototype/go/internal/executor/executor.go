package executor

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5/middleware"

	"menata.id/runtime/internal/constraint"
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

// ResolveFields exports resolveActionFields (CAP-A06's own field-mapping
// resolution -- "field:<id>" copies from sourceData, a literal is a
// literal, dynamic tokens resolve the same way) for CAP-I01's Subscription
// processing (handler.processSubscriptions), which lives in Handler (not
// Executor -- it needs Interpreter access to find subscribers, the same
// "cross-record/cross-machine logic belongs in Handler" boundary
// Executor's own doc comment already establishes) but wants the exact same
// resolution a create_record action already gets, not a second
// implementation of it.
func (e *Executor) ResolveFields(raw any, sourceData map[string]any, actorRole, actorIdentity string, holidays map[string]bool) map[string]any {
	return resolveActionFields(raw, sourceData, actorRole, actorIdentity, holidays)
}

// Simulate computes a record's data after applying event's set_field actions,
// without persisting anything. CAP-C09 validates constraints against this
// result before Persist commits it — a record must satisfy every Constraint
// after an event, not just at Create. machine is needed for CAP-A12's "next"
// value_list stepping (the target field's own declared Options.Values order)
// — Executor otherwise has no Interpreter access, this is the one exception,
// passed in by the caller (already has it) rather than reached for.
//
// actorIdentity resolves CAP-A02's "current_user" dynamic value. CAP-P02
// added a real identity concept distinct from role (the menata_identity
// cookie); "current_user" now resolves to that identity, falling back to
// actorRole when identity is empty (internal "System"-triggered events pass
// both as "System"). "today"/"now" are unaffected by either.
//
// eventInput (CAP-P04) carries fresh values submitted alongside THIS
// trigger request, for a Field the Event declares in its own InputFields —
// "who to delegate to" can only be known at the moment of delegating, not
// baked into the record beforehand. A `set_field.value` of "input:<field>"
// reads from here, distinct from "field:<id>" (reads the record's own
// EXISTING data, used by create_record/batch_generate's field-copying).
// nil/empty for every trigger that isn't CAP-P04-shaped.
//
// holidays (CAP-O06) is workspaceID's own declared non-working dates
// (Interpreter.Holidays), consumed by resolveDateArithmetic's "N Business
// Days" unit -- nil is safe (business-day math still skips weekends, just
// without workspace-specific holiday exclusions).
func (e *Executor) Simulate(machine *model.Machine, event *model.Event, record *store.Record, actorRole, actorIdentity string, eventInput map[string]string, holidays map[string]bool) map[string]any {
	newData := make(map[string]any, len(record.Data))
	for k, v := range record.Data {
		newData[k] = v
	}
	fieldByID := make(map[string]*model.Field, len(machine.Fields))
	for _, f := range machine.Fields {
		fieldByID[f.ID] = f
	}
	for _, action := range event.Actions {
		if action.Type != model.ActionSetField {
			continue
		}
		if !actionApplies(action, newData) {
			continue
		}
		field, _ := action.Params["field"].(string)
		value, _ := action.Params["value"].(string)
		if field != "" {
			newData[field] = resolveValue(value, newData, fieldByID[field], actorRole, actorIdentity, eventInput, holidays)
		}
	}
	return newData
}

// actionApplies (CAP-A09): an action carrying an "if" param only runs when
// that condition (the same field/operator/value shape a Constraint's own
// `condition` uses) is satisfied against the record's current data — an
// action with no "if" always applies, unchanged from before this existed.
func actionApplies(action *model.EventAction, data map[string]any) bool {
	raw, ok := action.Params["if"]
	if !ok {
		return true
	}
	expr, ok := paramsToExpr(raw)
	if !ok {
		return true
	}
	return constraint.Eval(expr, data)
}

func paramsToExpr(v any) (model.ConstraintExpression, bool) {
	m, ok := v.(map[string]any)
	if !ok {
		return model.ConstraintExpression{}, false
	}
	expr := model.ConstraintExpression{}
	expr.Field, _ = m["field"].(string)
	expr.Operator, _ = m["operator"].(string)
	expr.Value, _ = m["value"].(string)
	expr.ValueField, _ = m["value_field"].(string)
	return expr, expr.Field != "" && expr.Operator != ""
}

var dateArithRe = regexp.MustCompile(`^(.+?)\s*\+\s*(-?\d+)\s*(Day|Days|Week|Weeks|Month|Months|Year|Years|Business Day|Business Days)$`)

// resolveDateArithmetic (CAP-A11): "today + 1 Month" or "<field_id> + N Unit"
// — the flat, unkeyed date-arithmetic family (advance a date literal or
// another field's own value by a fixed offset). Deliberately NOT the
// priority-keyed variant complaint.yaml's sla_offset(priority) wanted (a
// different, still-unimplemented lookup-by-another-field's-value flavor,
// noted but not built here). ok=false when value doesn't match this shape
// at all, or the base doesn't resolve to a real date.
//
// "N Business Day(s)" (CAP-O06) is the one unit that isn't a fixed
// calendar step -- it counts only weekdays, additionally skipping any date
// in holidays (workspaceID's own declared non-working dates,
// Interpreter.Holidays; nil skips weekends only, still correct, just
// without workspace-specific exclusions).
func resolveDateArithmetic(value string, data map[string]any, holidays map[string]bool) (string, bool) {
	m := dateArithRe.FindStringSubmatch(value)
	if m == nil {
		return "", false
	}
	base := strings.TrimSpace(m[1])
	n, err := strconv.Atoi(m[2])
	if err != nil {
		return "", false
	}
	unit := m[3]

	var baseDate time.Time
	if base == "today" {
		baseDate = time.Now().Truncate(24 * time.Hour)
	} else {
		raw, ok := data[base]
		if !ok {
			return "", false
		}
		t, err := time.Parse("2006-01-02", fmt.Sprintf("%v", raw))
		if err != nil {
			return "", false
		}
		baseDate = t
	}

	var result time.Time
	switch unit {
	case "Day", "Days":
		result = baseDate.AddDate(0, 0, n)
	case "Week", "Weeks":
		result = baseDate.AddDate(0, 0, n*7)
	case "Month", "Months":
		result = baseDate.AddDate(0, n, 0)
	case "Year", "Years":
		result = baseDate.AddDate(n, 0, 0)
	case "Business Day", "Business Days":
		result = addBusinessDays(baseDate, n, holidays)
	default:
		return "", false
	}
	return result.Format("2006-01-02"), true
}

// addBusinessDays (CAP-O06) advances base by n weekdays, additionally
// skipping any date present in holidays (nil-safe -- weekends alone still
// get skipped). n may be negative ("N Business Days ago").
func addBusinessDays(base time.Time, n int, holidays map[string]bool) time.Time {
	step := 1
	if n < 0 {
		step = -1
		n = -n
	}
	d := base
	for n > 0 {
		d = d.AddDate(0, 0, step)
		if wd := d.Weekday(); wd == time.Saturday || wd == time.Sunday {
			continue
		}
		if holidays[d.Format("2006-01-02")] {
			continue
		}
		n--
	}
	return d
}

// resolveNextValue (CAP-A12): "next" advances a value_list field to the
// option immediately after its current value in that field's own declared
// Options.Values order -- already at (or past) the last value stays put,
// deliberately not wrapping around, since "next" past the end usually means
// a metadata/workflow bug, not an intentional cycle back to the start.
func resolveNextValue(field *model.Field, data map[string]any) (string, bool) {
	if field == nil || field.Type != model.FieldTypeValueList || len(field.Options.Values) == 0 {
		return "", false
	}
	current := fmt.Sprintf("%v", data[field.ID])
	for i, v := range field.Options.Values {
		if v == current && i+1 < len(field.Options.Values) {
			return field.Options.Values[i+1], true
		}
	}
	return current, true
}

// resolveValue resolves CAP-A02 dynamic value tokens, CAP-A11 date
// arithmetic, CAP-A12 value_list stepping, and CAP-P04's "input:<field>"
// (eventInput, see Simulate's own doc comment); any other string is a
// static literal, returned unchanged. field is the field this value is
// being written to (nil where that isn't known/relevant, e.g.
// create_record's target fields) -- only CAP-A12's "next" needs it.
func resolveValue(value string, data map[string]any, field *model.Field, actorRole, actorIdentity string, eventInput map[string]string, holidays map[string]bool) string {
	switch value {
	case "today":
		return time.Now().Format("2006-01-02")
	case "now":
		return time.Now().Format(time.RFC3339)
	case "current_user":
		return actorLabel(actorRole, actorIdentity)
	case "next":
		if v, ok := resolveNextValue(field, data); ok {
			return v
		}
		return value
	}
	if inputField, ok := strings.CutPrefix(value, "input:"); ok {
		return eventInput[inputField]
	}
	if v, ok := resolveDateArithmetic(value, data, holidays); ok {
		return v
	}
	return value
}

// actorLabel is the one place "who did this" resolves to a single string —
// identity when set (CAP-P02's real, distinct-from-role identity), falling
// back to role otherwise (internal "System"-triggered events pass both as
// "System"). Used both for CAP-A02's current_user and for CAP-R04's
// record_events.performed_by attribution, so the two stay consistent.
func actorLabel(actorRole, actorIdentity string) string {
	if actorIdentity != "" {
		return actorIdentity
	}
	return actorRole
}

// Persist saves newData (already validated by the caller — CAP-C09) as the
// record's new state, runs this event's non-data actions (notify,
// create_record, cross_set_field, batch_generate, ...), and logs the event
// with the pre-event data as its snapshot. machine is used for CAP-A15/A06's
// field-type-aware value resolution and machineName for a notification's
// message text — Executor still doesn't touch the Interpreter, machine is
// just the same struct the caller (Handler) already has.
//
// actorRole/actorIdentity attribute the audit row (CAP-R04's performed_by,
// via actorLabel — same resolution CAP-A02's current_user uses). The
// correlation id (CAP-I04) comes from ctx, not a param: chi's
// middleware.RequestID sets one per HTTP request, and because every internal
// cascade (CAP-A08 aggregate_status, CAP-E05 trigger_event) reuses the same
// ctx as the request that triggered it, every record_events row one request
// produces — even across different records — shares one id for free, no
// extra plumbing needed.
// Persist writes an event's resulting data and runs its actions, inside the
// caller's own request-scoped transaction (store.WithTx, see
// cmd/server/main.go's workspaceTx) -- CAP-X12: a cross-record action
// (create_record/cross_set_field/batch_generate) failing partway through
// must abort the WHOLE event, not just skip that one action, or the
// transaction still commits the record's own set_field changes plus
// whichever earlier actions in the loop already succeeded, silently
// dropping only the failed one. Every do* helper below therefore returns an
// error instead of just logging one, and this loop stops at the first
// failure -- the caller's non-2xx response is what makes workspaceTx's
// deferred Rollback discard everything, restoring true all-or-nothing
// semantics across every Machine an event's actions touch.
func (e *Executor) Persist(ctx context.Context, machine *model.Machine, event *model.Event, record *store.Record, newData map[string]any, machineName, actorRole, actorIdentity, workspaceID string, holidays map[string]bool) error {
	snapshot := record.Data

	for _, action := range event.Actions {
		if !actionApplies(action, newData) {
			continue
		}
		switch action.Type {
		case model.ActionNotify:
			e.doNotify(ctx, action, event, record, newData, machineName, workspaceID)

		case model.ActionCreateRecord:
			if err := e.doCreateRecord(ctx, action, newData, actorRole, actorIdentity, workspaceID, holidays); err != nil {
				return fmt.Errorf("create_record action: %w", err)
			}

		case model.ActionCrossSetField:
			if err := e.doCrossSetField(ctx, action, newData, holidays); err != nil {
				return fmt.Errorf("cross_set_field action: %w", err)
			}

		case model.ActionBatchGenerate:
			if err := e.doBatchGenerate(ctx, action, newData, actorRole, actorIdentity, workspaceID, holidays); err != nil {
				return fmt.Errorf("batch_generate action: %w", err)
			}
		}
	}

	if err := e.records.Update(ctx, record.ID, newData); err != nil {
		return err
	}
	return e.records.LogEvent(ctx, record.ID, event.ID, actorLabel(actorRole, actorIdentity), middleware.GetReqID(ctx), workspaceID, snapshot)
}

// doNotify implements CAP-A03 (static `role` recipient) and CAP-A04 (dynamic
// `recipient_field` recipient — the record's own field value, e.g. its
// Submitted By, rather than a whole role). recipient_field wins when its
// resolved value is non-empty; role is the fallback (and the only option
// most existing metadata declares). CAP-A10 in-app delivery: the resolved
// recipient is written as a real Notification row a matching role-cookie
// session can list and mark read, not just logged.
func (e *Executor) doNotify(ctx context.Context, action *model.EventAction, event *model.Event, record *store.Record, newData map[string]any, machineName, workspaceID string) {
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
	if err := e.notifications.Create(ctx, recipient, message, record.MachineID, record.ID, workspaceID); err != nil {
		slog.Error("create notification", "event", event.ID, "recipient", recipient, "error", err)
	}
}

// resolveActionFields resolves a create_record/batch_generate action's
// "fields" param map -- each value is either a dynamic token/date-arithmetic
// expression (via resolveValue, no target Field known here so CAP-A12
// "next" isn't available on these paths), or `field:<source_field_id>` to
// copy a value straight from the record that triggered this action
// (CAP-A06's only way to carry data from the triggering record into the new
// one -- deliberately not full template interpolation like `{{ this.x }}`,
// which nothing in this runtime evaluates).
func resolveActionFields(raw any, sourceData map[string]any, actorRole, actorIdentity string, holidays map[string]bool) map[string]any {
	m, _ := raw.(map[string]any)
	out := make(map[string]any, len(m))
	for field, v := range m {
		s := fmt.Sprintf("%v", v)
		if copied, ok := strings.CutPrefix(s, "field:"); ok {
			out[field] = sourceData[copied]
			continue
		}
		out[field] = resolveValue(s, sourceData, nil, actorRole, actorIdentity, nil, holidays)
	}
	return out
}

// doCreateRecord implements CAP-A06: params {"machine": "...", "fields":
// {...}} — creates one new record on another Machine. Deliberately not
// itself validated against the target Machine's own Constraints (unlike a
// real HTTP Create) -- a metadata-declared action is trusted input the same
// way a System-triggered event already is elsewhere in this codebase, not a
// place to re-derive full CAP-C01..C12 enforcement.
func (e *Executor) doCreateRecord(ctx context.Context, action *model.EventAction, sourceData map[string]any, actorRole, actorIdentity, workspaceID string, holidays map[string]bool) error {
	targetMachine, _ := action.Params["machine"].(string)
	if targetMachine == "" {
		return nil
	}
	data := resolveActionFields(action.Params["fields"], sourceData, actorRole, actorIdentity, holidays)
	if _, err := e.records.Create(ctx, targetMachine, workspaceID, data); err != nil {
		return fmt.Errorf("target machine %s: %w", targetMachine, err)
	}
	return nil
}

// doCrossSetField implements CAP-A13: params {"machine": "...",
// "record_field": "fld_ref", "field": "fld_target", "value": "..."} --
// record_field names a `reference` field ON THE TRIGGERING RECORD whose
// value is the id of ANOTHER record (on `machine`) to update. Unlike this
// event's own set_field actions (validated via CAP-C09 before Persist even
// runs), a cross-record write isn't re-validated against the target
// Machine's Constraints here either -- same trusted-metadata-action
// reasoning as doCreateRecord, and consistent with this codebase's existing
// System-actor precedent (CAP-A08's aggregate_status already writes to a
// different record via the full triggerEvent path when it needs guards to
// apply; this is the lighter-weight "just set one field" sibling of that,
// for when a full event trigger on the target would be overkill).
func (e *Executor) doCrossSetField(ctx context.Context, action *model.EventAction, sourceData map[string]any, holidays map[string]bool) error {
	recordFieldID, _ := action.Params["record_field"].(string)
	targetField, _ := action.Params["field"].(string)
	value, _ := action.Params["value"].(string)
	if recordFieldID == "" || targetField == "" {
		return nil
	}
	targetID := fmt.Sprintf("%v", sourceData[recordFieldID])
	if targetID == "" || targetID == "<nil>" {
		return nil
	}
	targetRecord, err := e.records.Get(ctx, targetID)
	if err != nil {
		return fmt.Errorf("target record %s: %w", targetID, err)
	}
	newData := make(map[string]any, len(targetRecord.Data))
	for k, v := range targetRecord.Data {
		newData[k] = v
	}
	newData[targetField] = resolveValue(value, newData, nil, "System", "System", nil, holidays)
	if err := e.records.Update(ctx, targetID, newData); err != nil {
		return fmt.Errorf("target record %s: %w", targetID, err)
	}
	return nil
}

// doBatchGenerate implements CAP-A15: params {"machine": "...", "count": N,
// "fields": {...}, "offset_field": "fld_date", "offset_unit": "Day(s)|
// Week(s)|Month(s)|Year(s)"} — creates N records from one action. Every
// instance starts from the same resolved `fields`; when offset_field is set,
// that one field is additionally advanced by i * 1-unit per instance i =
// 0..N-1 (composing CAP-A11's date arithmetic, not a separate mechanism) —
// e.g. N weekly recurrences of a Task, each one week after the last, from a
// single "create the series" action.
func (e *Executor) doBatchGenerate(ctx context.Context, action *model.EventAction, sourceData map[string]any, actorRole, actorIdentity, workspaceID string, holidays map[string]bool) error {
	targetMachine, _ := action.Params["machine"].(string)
	countRaw, _ := action.Params["count"].(float64) // JSON numbers decode as float64
	count := int(countRaw)
	if targetMachine == "" || count <= 0 {
		return nil
	}
	offsetField, _ := action.Params["offset_field"].(string)
	offsetUnit, _ := action.Params["offset_unit"].(string)

	base := resolveActionFields(action.Params["fields"], sourceData, actorRole, actorIdentity, holidays)
	for i := 0; i < count; i++ {
		data := make(map[string]any, len(base))
		for k, v := range base {
			data[k] = v
		}
		if offsetField != "" && i > 0 {
			if raw, ok := base[offsetField]; ok {
				expr := fmt.Sprintf("%v + %d %s", raw, i, offsetUnit)
				if v, ok := resolveDateArithmetic(expr, nil, holidays); ok {
					data[offsetField] = v
				}
			}
		}
		if _, err := e.records.Create(ctx, targetMachine, workspaceID, data); err != nil {
			return fmt.Errorf("target machine %s, instance %d: %w", targetMachine, i, err)
		}
	}
	return nil
}
