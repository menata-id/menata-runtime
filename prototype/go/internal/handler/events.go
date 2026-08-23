package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"menata.id/runtime/internal/auth"
	"menata.id/runtime/internal/constraint"
	"menata.id/runtime/internal/model"
	"menata.id/runtime/internal/store"
)

// TriggerEvent — handle event button.
func (h *Handler) TriggerEvent(w http.ResponseWriter, r *http.Request) {
	machineID := chi.URLParam(r, "machineID")
	recordID := chi.URLParam(r, "recordID")
	eventID := chi.URLParam(r, "eventID")

	machine, ok := h.interp.Get().GetMachine(machineID)
	if !ok {
		http.NotFound(w, r)
		return
	}
	workspaceID, applicationID := h.interp.Get().ScopeFor(machineID)
	if workspaceID != h.workspace(r) {
		// CAP-X06: a Machine from another Workspace 404s exactly like one
		// that doesn't exist at all -- app-layer guard alongside RLS
		// (migrations/009), not instead of it.
		http.NotFound(w, r)
		return
	}
	event, ok := h.interp.Get().GetEvent(machineID, eventID)
	if !ok {
		http.NotFound(w, r)
		return
	}
	rec, err := h.records.Get(r.Context(), recordID)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	role, identity := h.roleForApp(r, applicationID), h.identity(r)
	// CAP-P02 — ownership check needs the record's own data, so this can't
	// run until after the record fetch above (unlike role-only permission,
	// which didn't need it). Compared by id (CAP-F05), not by identity's
	// display name -- triggerEvent below still gets the human-readable
	// identity, for audit attribution and CAP-A02's current_user.
	if !h.guard.CanTrigger(machine, role, h.identityID(r), eventID, rec.Data) {
		h.logPermissionDenied(r.Context(), "trigger", machineID, eventID, role, identity)
		http.Error(w, "not permitted", http.StatusForbidden)
		return
	}

	// CAP-P04: an event declaring InputFields (e.g. "delegate to") needs
	// fresh values submitted alongside this trigger, not read from the
	// record's own data -- collected here, resolved by set_field's
	// "input:<field>" (Executor.resolveValue).
	var eventInput map[string]string
	if len(event.InputFields) > 0 {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		eventInput = make(map[string]string, len(event.InputFields))
		for _, id := range event.InputFields {
			eventInput[id] = r.FormValue(id)
		}
	}

	if err := h.triggerEvent(r.Context(), machine, event, rec, role, identity, h.identityID(r), eventInput); err != nil {
		var rv *ruleViolation
		if errors.As(err, &rv) {
			h.logRuleViolation(r.Context(), "trigger", machineID, eventID, role, identity, rv.Error())
			http.Error(w, rv.Error(), http.StatusBadRequest)
		} else {
			http.Error(w, "event failed", http.StatusInternalServerError)
		}
		return
	}
	http.Redirect(w, r, "/"+machineID+"/"+recordID, http.StatusSeeOther)
}

// Webhook (CAP-E04) lets an external system trigger an event directly --
// no session, no CSRF token (cmd/server/main.go's isPublicPath/
// csrfProtect both carve this path out), authenticated instead by a
// per-Machine shared secret (Machine.Config's "webhook_secret", CAP-X03's
// existing generic settings -- not a new migration column). A Machine that
// hasn't declared one 404s -- webhooks are opt-in per Machine, not a
// blanket surface every Machine gets. No role-based Guard.CanTrigger check
// either: the secret itself is the authorization here, the same posture
// CAP-A08/CAP-E05's internal "System"-triggered cascades already take
// (see doAggregateStatus/doTriggerEvent) -- an external caller isn't a
// person holding a role, there's nothing for CanTrigger's ownership/role
// check to mean.
//
// A payment webhook's own payload can carry values too, the same
// InputFields/"input:<field>" mechanism CAP-P04 already built for
// delegation's "who to hand off to" -- JSON body if Content-Type says so
// (the realistic shape for most real webhook providers), form-encoded
// otherwise.
func (h *Handler) Webhook(w http.ResponseWriter, r *http.Request) {
	machineID := chi.URLParam(r, "machineID")
	recordID := chi.URLParam(r, "recordID")
	eventID := chi.URLParam(r, "eventID")

	machine, ok := h.interp.Get().GetMachine(machineID)
	if !ok {
		http.NotFound(w, r)
		return
	}
	secret := machine.Config["webhook_secret"]
	if secret == "" {
		http.NotFound(w, r)
		return
	}
	if !auth.ConstantTimeEqual(r.Header.Get("X-Webhook-Secret"), secret) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	event, ok := h.interp.Get().GetEvent(machineID, eventID)
	if !ok {
		http.NotFound(w, r)
		return
	}
	rec, err := h.records.Get(r.Context(), recordID)
	if err != nil || rec.MachineID != machineID {
		http.NotFound(w, r)
		return
	}

	// CAP-X13: an idempotency key is opt-in -- a caller retrying the exact
	// same delivery (network timeout, at-least-once queue redelivery) sends
	// the same X-Idempotency-Key and gets 200 back without the event firing
	// a second time. No key at all skips the check entirely (unchanged
	// behavior for every webhook caller that doesn't supply one).
	if key := r.Header.Get("X-Idempotency-Key"); key != "" {
		claimed, err := h.records.ClaimWebhookEvent(r.Context(), machineID, eventID, key)
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		if !claimed {
			w.WriteHeader(http.StatusOK)
			return
		}
	}

	var eventInput map[string]string
	if len(event.InputFields) > 0 {
		eventInput = make(map[string]string, len(event.InputFields))
		if strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err == nil {
				for _, id := range event.InputFields {
					if v, ok := body[id]; ok {
						eventInput[id] = fmt.Sprintf("%v", v)
					}
				}
			}
		} else if err := r.ParseForm(); err == nil {
			for _, id := range event.InputFields {
				eventInput[id] = r.FormValue(id)
			}
		}
	}

	if err := h.triggerEvent(r.Context(), machine, event, rec, "System", "Webhook", "", eventInput); err != nil {
		var rv *ruleViolation
		if errors.As(err, &rv) {
			h.logRuleViolation(r.Context(), "webhook", machineID, eventID, "System", "Webhook", rv.Error())
			http.Error(w, rv.Error(), http.StatusBadRequest)
		} else {
			http.Error(w, "event failed", http.StatusInternalServerError)
		}
		return
	}
	w.WriteHeader(http.StatusOK)
}

// ruleViolation marks a rejection as a business-rule failure (400), as
// opposed to an infrastructure error (500) — the same distinction Create
// already draws by rendering violations back into the form instead of
// erroring out.
type ruleViolation struct{ msg string }

func (e *ruleViolation) Error() string { return e.msg }

// logPermissionDenied records an access-control failure as a distinct,
// security-relevant log line (OWASP ASVS V7.2 — "all failed access control
// decisions must be logged"), not just the routine access-log entry chi's
// middleware.Logger already writes for the resulting 403 response.
// correlation_id is the same request id CAP-R04's record_events rows carry
// (executor.Persist), so a probing pattern (many denials across roles in one
// session) and any successful mutation from the same request can be
// correlated. eventID is empty for non-event actions (read/create/edit).
// workspace/application (006-runtime-model.md's hierarchy) are resolved
// here, not passed in, so every call site stays a one-liner.
func (h *Handler) logPermissionDenied(ctx context.Context, action, machineID, eventID, role, identity string) {
	workspaceID, appID := h.interp.Get().ScopeFor(machineID)
	slog.Warn("permission denied",
		"correlation_id", middleware.GetReqID(ctx),
		"workspace", workspaceID,
		"application", appID,
		"action", action,
		"machine", machineID,
		"event", eventID,
		"role", role,
		"identity", identity,
	)
}

// logRuleViolation records a rejected write (CAP-C01..C11 constraints, or
// CAP-E06's state guard) as a distinct log line — ASVS V7.1's "all input
// validation failures" — separate from logPermissionDenied's access-control
// failures: this is a request that *was* permitted but rejected on the
// data/state it carried.
func (h *Handler) logRuleViolation(ctx context.Context, action, machineID, eventID, role, identity, reason string) {
	workspaceID, appID := h.interp.Get().ScopeFor(machineID)
	slog.Warn("rule violation",
		"correlation_id", middleware.GetReqID(ctx),
		"workspace", workspaceID,
		"application", appID,
		"action", action,
		"machine", machineID,
		"event", eventID,
		"role", role,
		"identity", identity,
		"reason", reason,
	)
}

// triggerEvent is the single path every event trigger runs through, whether
// from an HTTP button (TriggerEvent) or fired internally by another event's
// aggregate_status or trigger_event action (CAP-A08/CAP-E05, doAggregateStatus/
// doTriggerEvent) — same guards, same validation, same persistence, so a
// system-triggered transition can never skip a check a user-triggered one
// would have to pass.
func (h *Handler) triggerEvent(ctx context.Context, machine *model.Machine, event *model.Event, rec *store.Record, actorRole, actorIdentity, actorIdentityID string, eventInput map[string]string) error {
	// CAP-I02 — a deprecated Event still works (backward compat by design)
	// but logs a warning every time it's actually used, so a real
	// deprecation can be tracked/acted on instead of staying invisible.
	if event.DeprecatedMessage != "" {
		slog.Warn("triggered deprecated event", "event", event.ID, "machine", machine.ID, "message", event.DeprecatedMessage)
	}

	// CAP-E06 — state guard: the event may only fire when the record's
	// CURRENT data satisfies its condition (e.g. Reject only from Submitted).
	if event.Condition != nil && !constraint.Eval(*event.Condition, rec.Data) {
		return &ruleViolation{fmt.Sprintf("%s is not allowed in the record's current state", event.Name)}
	}

	// CAP-P03 — separation of duties: the person who submitted the PARENT
	// record (looked up via this record's own reference field) may not also
	// decide this one, even if they happen to hold the deciding role too --
	// cross-record, so like CAP-A07's sequential guard it can't be expressed
	// as an event Condition (CAP-E06 only reads this record's own data).
	// actorIdentityID == "" (System-triggered events, doAggregateStatus) is
	// exempt -- there's no human actor to self-deal.
	if actorIdentityID != "" {
		if msg, err := h.separationOfDutiesViolation(ctx, machine, rec, actorIdentityID); err != nil {
			return err
		} else if msg != "" {
			return &ruleViolation{msg}
		}
	}

	// CAP-A07 — sequential step guard: in a Sequential-mode parent, a step may
	// only be decided once every sibling with a lower Sequence has already
	// left Pending. Cross-record, so it can't be expressed as an event
	// Condition (CAP-E06 only reads the record's own data).
	if msg := h.sequentialGuardViolation(ctx, machine, event, rec); msg != "" {
		return &ruleViolation{msg}
	}

	// CAP-A14 — aggregate-conditioned guard: the same idea as CAP-E06's
	// Condition, computed across sibling records instead of this one's own
	// fields (e.g. "only once this Member's total Points reach 100").
	if msg, err := h.aggregateConditionViolation(ctx, event, rec); err != nil {
		return err
	} else if msg != "" {
		return &ruleViolation{msg}
	}

	// CAP-C09 — constraints evaluated on event trigger, not just Create:
	// simulate the event's effect first, validate the result, only persist
	// if it still satisfies every declared Constraint. CAP-A02 — actorIdentity
	// (falling back to actorRole) resolves this event's "current_user" dynamic
	// values, if any. CAP-O06 — holidays resolved once here, passed through
	// to every "N Business Days" resolution Simulate/Persist might do.
	workspaceID, _ := h.interp.Get().ScopeFor(machine.ID)
	holidays := h.interp.Get().Holidays(workspaceID)
	newData := h.exec.Simulate(machine, event, rec, actorRole, actorIdentity, eventInput, holidays)
	if violations := h.engine.Violations(machine, withChangePolicyCreatedAt(machine, newData, rec.CreatedAt)); len(violations) > 0 {
		return &ruleViolation{strings.Join(violations, " ")}
	}
	// CAP-C08: cross-record constraints (e.g. debit=credit, period-open) need
	// the SAME re-check CAP-C09 already gives ordinary Constraints, but at
	// this specific site -- uniquenessViolations/referenceViolations are
	// only called from Create/Update/CSV-import, never here, because this is
	// the first cross-record check that actually needs to fire on a plain
	// state transition (Post), not just when a record's own fields change.
	if crossViolations, err := h.crossRecordViolations(ctx, machine, newData, rec.ID); err != nil {
		return err
	} else if len(crossViolations) > 0 {
		return &ruleViolation{strings.Join(crossViolations, " ")}
	}

	if err := h.exec.Persist(ctx, machine, event, rec, newData, machine.Name, actorRole, actorIdentity, workspaceID, holidays); err != nil {
		return err
	}

	// CAP-A07/CAP-A08/CAP-E05 — workflow actions run after a successful
	// commit, using the now-current data: activate_next notifies the next
	// pending sibling, aggregate_status may internally trigger a rollup event
	// on the parent, trigger_event may fire another event on this same record.
	h.runWorkflowActions(ctx, machine, event, rec.ID, newData)

	// CAP-I01 — cross-machine subscribers run last, after the publisher's
	// OWN write and workflow actions have already succeeded (error rule 1:
	// a subscriber's failure never rolls back the publisher).
	h.processSubscriptions(ctx, event, newData, actorRole, actorIdentity, workspaceID, holidays)
	return nil
}

// processSubscriptions (CAP-I01) dispatches every Subscription declaring
// interest in event.ID -- one new record per Subscription, on its own
// MachineID, Fields resolved from the publisher's post-event data.
// CAP-I03's Contract/OnViolation gate each one independently first. Error
// rules 2-4 (see model.Subscription's own doc comment): each Subscription
// is processed regardless of whether an earlier one failed; every failure
// is logged, never silently dropped; every Subscription sees the same
// final data, never a partial view.
//
// CAP-W06 (2026-08-23): the contract check and field resolution above are
// unchanged -- a Subscription still sees the exact same post-event data
// every other Subscription in this loop does. Only the final step changes:
// the already-resolved fields are enqueued as an action_outbox row (atomic
// with the publisher's own record write, same transaction) instead of
// creating the subscriber record inline. runOutboxDispatcher performs the
// real Create shortly after, off the request path.
func (h *Handler) processSubscriptions(ctx context.Context, event *model.Event, data map[string]any, actorRole, actorIdentity, workspaceID string, holidays map[string]bool) {
	for _, sub := range h.interp.Get().SubscriptionsFor(event.ID) {
		violated := false
		for _, c := range sub.Contract {
			if !constraint.Eval(c, data) {
				violated = true
				slog.Warn("subscription contract violation", "subscription", sub.ID, "publisher_event", event.ID, "field", c.Field, "on_violation", sub.OnViolation)
				break
			}
		}
		if violated && sub.OnViolation == "skip" {
			continue
		}
		fields := h.exec.ResolveFields(sub.Fields, data, actorRole, actorIdentity, holidays)
		params := map[string]any{
			"machine_id": sub.MachineID,
			"fields":     fields,
		}
		if err := h.outbox.Enqueue(ctx, workspaceID, "subscription", params, middleware.GetReqID(ctx)); err != nil {
			slog.Error("enqueue subscription outbox row", "subscription", sub.ID, "publisher_event", event.ID, "machine", sub.MachineID, "error", err)
		}
	}
}

// --- helpers -----------------------------------------------------------------

// aggregateConditionViolation implements CAP-A14: an Event with an
// AggregateCondition may only be triggered once SUM(aggregate_field) across
// every record on Machine sharing this record's own value for scope_field
// satisfies operator/value -- "only once this Member's total Points reach
// 100." Returns ("", nil) when the event has no AggregateCondition at all
// (the overwhelmingly common case).
func (h *Handler) aggregateConditionViolation(ctx context.Context, event *model.Event, rec *store.Record) (string, error) {
	ac := event.AggregateCondition
	if ac == nil {
		return "", nil
	}
	scopeValue := fmt.Sprintf("%v", rec.Data[ac.ScopeField])
	sum, err := h.records.SumField(ctx, ac.Machine, ac.AggregateField, ac.ScopeField, scopeValue)
	if err != nil {
		return "", err
	}
	// constraint.Eval reads expr.Field out of a data map -- "sum" is a
	// synthetic single-key map standing in for the computed aggregate, the
	// same field/operator/value comparison every other Eval call already uses.
	expr := model.ConstraintExpression{Field: "sum", Operator: ac.Operator, Value: ac.Value}
	if !constraint.Eval(expr, map[string]any{"sum": sum}) {
		return fmt.Sprintf("%s is not allowed until the aggregate condition on %s is met", event.Name, ac.AggregateField), nil
	}
	return "", nil
}

// separationOfDutiesViolation (CAP-P03) blocks a decision when the acting
// identity is the same person who submitted the PARENT record this one
// belongs to -- "Requester != Approver," even when the actor happens to
// also hold the deciding role. Declared via Machine.Config (CAP-X03, no
// new migration column, same pattern CAP-R07/R08 already use):
//
//	sod_reference_field:  this Machine's own `reference` field pointing at
//	                      the parent (e.g. an Approval Step's fld_as_document)
//	sod_requester_field:  the `user`-typed field on the PARENT Machine
//	                      holding who submitted it
//
// Returns "" (no violation) when the Machine doesn't declare either key --
// separation of duties is opt-in, not a default every Machine pays for.
func (h *Handler) separationOfDutiesViolation(ctx context.Context, machine *model.Machine, rec *store.Record, actorIdentityID string) (string, error) {
	refField := machine.Config["sod_reference_field"]
	requesterField := machine.Config["sod_requester_field"]
	if refField == "" || requesterField == "" {
		return "", nil
	}
	parentID, _ := rec.Data[refField].(string)
	if parentID == "" {
		return "", nil
	}
	parent, err := h.records.Get(ctx, parentID)
	if err != nil {
		return "", nil
	}
	if fmt.Sprintf("%v", parent.Data[requesterField]) == actorIdentityID {
		return "you submitted this record and cannot also decide it (separation of duties)", nil
	}
	return "", nil
}

// sequentialGuardViolation implements CAP-A07's hard block: in Sequential
// mode, a step may only be Approved or Rejected once every sibling with a
// lower Sequence has already left Pending. Applies to any event carrying an
// aggregate_status action — both evt_as_approve and evt_as_reject declare
// one, so both are gated uniformly, even though only Approve also declares
// activate_next. Returns "" when the event isn't part of a sequential
// workflow at all (no aggregate_status action, no config, or Parallel mode).
func (h *Handler) sequentialGuardViolation(ctx context.Context, machine *model.Machine, event *model.Event, rec *store.Record) string {
	var parentFieldID string
	for _, a := range event.Actions {
		if a.Type == model.ActionAggregateStatus {
			parentFieldID, _ = a.Params["parent_field"].(string)
			break
		}
	}
	if parentFieldID == "" {
		return ""
	}
	parentField := findFieldByID(machine, parentFieldID)
	if parentField == nil || parentField.Type != model.FieldTypeReference {
		return ""
	}
	parentMachine, ok := h.interp.Get().GetMachine(parentField.Options.TargetMachine)
	if !ok || parentMachine.Config == nil {
		return ""
	}
	modeFieldID := parentMachine.Config["approval_mode_field"]
	if modeFieldID == "" {
		return ""
	}
	parentID, _ := rec.Data[parentFieldID].(string)
	if parentID == "" {
		return ""
	}
	parentRec, err := h.records.Get(ctx, parentID)
	if err != nil {
		return ""
	}
	if fmt.Sprintf("%v", parentRec.Data[modeFieldID]) != "Sequential" {
		return ""
	}

	seqField := model.FindFieldByName(machine, "Sequence")
	decisionField := model.FindFieldByName(machine, "Decision")
	if seqField == nil || decisionField == nil {
		return ""
	}
	mySeq := toFloat(rec.Data[seqField.ID])

	siblings, err := h.records.List(ctx, machine.ID, "", "")
	if err != nil {
		return ""
	}
	for _, sib := range siblings {
		if sib.ID == rec.ID {
			continue
		}
		if sp, _ := sib.Data[parentFieldID].(string); sp != parentID {
			continue
		}
		if fmt.Sprintf("%v", sib.Data[decisionField.ID]) != "Pending" {
			continue
		}
		if toFloat(sib.Data[seqField.ID]) < mySeq {
			return fmt.Sprintf("%s is not allowed yet — an earlier step is still Pending", event.Name)
		}
	}
	return ""
}

// runWorkflowActions handles the two action types Executor.Persist doesn't
// touch (CAP-A07/A08 need cross-record lookups Executor has no access to —
// this stays at the Handler layer, same reasoning as childLists/
// referenceViolations).
func (h *Handler) runWorkflowActions(ctx context.Context, machine *model.Machine, event *model.Event, recordID string, data map[string]any) {
	for _, a := range event.Actions {
		switch a.Type {
		case model.ActionActivateNext:
			h.doActivateNext(ctx, machine, a.Params, recordID, data)
		case model.ActionAggregateStatus:
			h.doAggregateStatus(ctx, machine, a.Params, data)
		case model.ActionTriggerEvent:
			h.doTriggerEvent(ctx, machine, a.Params, recordID, data)
		}
	}
}

// doActivateNext (CAP-A07): in Sequential mode, notifies the next still-
// Pending sibling's Approver that it's their turn. Doesn't write anything —
// the sequential guard above is what actually enforces order; this is purely
// the hand-off notification (logged only, same as every other `notify` in
// this prototype).
func (h *Handler) doActivateNext(ctx context.Context, machine *model.Machine, params map[string]any, recordID string, data map[string]any) {
	modeFieldID, _ := params["mode_field"].(string)
	if modeFieldID == "" {
		return
	}
	parentMachine := findMachineContainingField(h.interp.Get(), modeFieldID)
	if parentMachine == nil || parentMachine.Config == nil {
		return
	}
	parentRefField := model.FindReferenceFieldTo(machine, parentMachine.ID)
	if parentRefField == nil {
		return
	}
	parentID, _ := data[parentRefField.ID].(string)
	if parentID == "" {
		return
	}
	parentRec, err := h.records.Get(ctx, parentID)
	if err != nil || fmt.Sprintf("%v", parentRec.Data[modeFieldID]) != "Sequential" {
		return
	}

	seqField := model.FindFieldByName(machine, "Sequence")
	decisionField := model.FindFieldByName(machine, "Decision")
	approverField := model.FindFieldByName(machine, "Approver")
	if seqField == nil || decisionField == nil {
		return
	}
	mySeq := toFloat(data[seqField.ID])

	siblings, err := h.records.List(ctx, machine.ID, "", "")
	if err != nil {
		return
	}
	var next *store.Record
	var nextSeq float64
	for _, sib := range siblings {
		if sib.ID == recordID {
			continue
		}
		if sp, _ := sib.Data[parentRefField.ID].(string); sp != parentID {
			continue
		}
		if fmt.Sprintf("%v", sib.Data[decisionField.ID]) != "Pending" {
			continue
		}
		s := toFloat(sib.Data[seqField.ID])
		if s > mySeq && (next == nil || s < nextSeq) {
			next, nextSeq = sib, s
		}
	}
	if next == nil {
		return
	}
	approver := ""
	if approverField != nil {
		approver = fmt.Sprintf("%v", next.Data[approverField.ID])
	}
	slog.Info("activate_next: next step ready (prototype: notify logged only)",
		"machine", machine.ID, "next_record", next.ID, "sequence", nextSeq, "approver", approver)
}

// doTriggerEvent (CAP-E05): fires another event on the SAME record — a
// narrower, same-record counterpart to doAggregateStatus's cross-record
// rollup. No extra DB fetch needed (unlike doAggregateStatus, which reaches
// into a different record): `data` is already this event's just-persisted
// result, and recordID is this same record's own ID. Runs through the same
// triggerEvent path an HTTP request would use (CAP-E06/CAP-C09 guards still
// apply), as "System" for both role and identity — same precedent as
// doAggregateStatus, and the same reason guard.CanTrigger is bypassed
// entirely here rather than called: a system-triggered transition isn't a
// business role acting, so there's no role/identity to check permission
// against.
func (h *Handler) doTriggerEvent(ctx context.Context, machine *model.Machine, params map[string]any, recordID string, data map[string]any) {
	targetEventID, _ := params["event"].(string)
	if targetEventID == "" {
		return
	}
	targetEvent, ok := h.interp.Get().GetEvent(machine.ID, targetEventID)
	if !ok {
		return
	}
	rec := &store.Record{ID: recordID, Data: data}
	if err := h.triggerEvent(ctx, machine, targetEvent, rec, "System", "System", "", nil); err != nil {
		slog.Error("trigger_event: failed to trigger chained event",
			"machine", machine.ID, "record", recordID, "event", targetEventID, "error", err)
	}
}

// doAggregateStatus (CAP-A08): rolls sibling steps' Decision up to the parent
// — any Rejected cascades the parent to its "any rejected" event immediately
// (doesn't wait for the rest to decide); only once every sibling is Approved
// does the parent reach its "all approved" event. Internally triggers the
// resolved parent event through the same triggerEvent path an HTTP request
// would use (CAP-E06/CAP-C09 guards still apply), as "System" — the acting
// role recorded for this rollup's dynamic values, matching approval-
// document.yaml's own declared System permission role.
//
// min_approvals (CAP-W03, Process Overlay B4 Part 2) is an optional
// quorum-of-N parameter, backward-compatible by construction — omitted or
// zero falls through to the ALL-required behavior above, unchanged. Set to
// N over M total siblings: reaches parent_event_if_all_approved as soon as
// count(Approved) >= N, without waiting for every sibling to decide (real
// N-of-M semantics — a still-Pending sibling no longer blocks quorum once
// enough others have approved); reaches parent_event_if_any_rejected only
// once quorum becomes mathematically impossible
// (count(Rejected) > total - N, i.e. too few non-rejected siblings remain
// to ever reach N) — a minority of rejections that still leaves enough
// headroom does NOT cancel early, unlike the ALL-required rule above.
func (h *Handler) doAggregateStatus(ctx context.Context, machine *model.Machine, params map[string]any, data map[string]any) {
	parentFieldID, _ := params["parent_field"].(string)
	allApprovedEvt, _ := params["parent_event_if_all_approved"].(string)
	anyRejectedEvt, _ := params["parent_event_if_any_rejected"].(string)
	minApprovals, _ := params["min_approvals"].(float64) // JSON numbers decode as float64
	if parentFieldID == "" {
		return
	}
	parentField := findFieldByID(machine, parentFieldID)
	if parentField == nil || parentField.Type != model.FieldTypeReference {
		return
	}
	parentMachine, ok := h.interp.Get().GetMachine(parentField.Options.TargetMachine)
	if !ok {
		return
	}
	parentID, _ := data[parentFieldID].(string)
	if parentID == "" {
		return
	}

	decisionField := model.FindFieldByName(machine, "Decision")
	if decisionField == nil {
		return
	}
	siblings, err := h.records.List(ctx, machine.ID, "", "")
	if err != nil {
		return
	}

	found, allApproved, anyRejected := false, true, false
	total, approved, rejected := 0, 0, 0
	for _, sib := range siblings {
		if sp, _ := sib.Data[parentFieldID].(string); sp != parentID {
			continue
		}
		found = true
		total++
		switch fmt.Sprintf("%v", sib.Data[decisionField.ID]) {
		case "Rejected":
			anyRejected = true
			rejected++
		case "Approved":
			approved++
			// stays eligible for allApproved
		default:
			allApproved = false
		}
	}
	if !found {
		return
	}

	var targetEventID string
	switch {
	case minApprovals > 0:
		switch {
		case approved >= int(minApprovals) && allApprovedEvt != "":
			targetEventID = allApprovedEvt
		case rejected > total-int(minApprovals) && anyRejectedEvt != "":
			targetEventID = anyRejectedEvt
		default:
			return
		}
	case anyRejected && anyRejectedEvt != "":
		targetEventID = anyRejectedEvt
	case allApproved && allApprovedEvt != "":
		targetEventID = allApprovedEvt
	default:
		return
	}

	targetEvent, ok := h.interp.Get().GetEvent(parentMachine.ID, targetEventID)
	if !ok {
		return
	}
	parentRec, err := h.records.Get(ctx, parentID)
	if err != nil {
		return
	}
	if err := h.triggerEvent(ctx, parentMachine, targetEvent, parentRec, "System", "System", "", nil); err != nil {
		slog.Error("aggregate_status: failed to trigger parent event",
			"parent_machine", parentMachine.ID, "parent_record", parentID, "event", targetEventID, "error", err)
	}
}
