package metadata

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"menata.id/app/internal/model"
)

// compileProcess (Process Overlay B1 -- brd-menata-runtime-v2.md §7.2,
// Study 20 §6) expands a Machine's declared Process into the substrate
// primitives the runtime already executes: Events (with CAP-E06 state
// guards), Permissions (CAP-P01/P02), a Status value_list Field, and
// CAP-E05 trigger_event chains for auto transitions. It runs entirely at
// load time -- Router/Guard/Executor/Engine never see the Process itself,
// only its compiled result ("declared process, emergent execution").
//
// Every generated id is a deterministic function of the declaration
// (event "evt_<machine>_<slug(name)>", permission
// "perm_<machine>_<slug(role)>", status field "fld_<machine>_status"), so
// identity stays stable across reloads (004-runtime-metadata.md §Stable
// Identity) -- the audit trail and any future subscription can reference a
// compiled event the same way it references a hand-authored one.
//
// B1 rule, enforced here: a Machine declares EITHER a Process OR its own
// hand-authored Events, never both -- merging the two is deliberately
// undefined until a real case needs it (fail-loud beats a silent, ambiguous
// merge; CAP-X05 discipline).
func compileProcess(m *model.Machine) error {
	p := m.Process
	if p == nil {
		return nil
	}
	if len(m.Events) > 0 {
		return fmt.Errorf("machine %s declares both a process and %d hand-authored event(s) -- B1 allows one or the other, not both", m.ID, len(m.Events))
	}
	if len(p.States) == 0 {
		return fmt.Errorf("machine %s: process declares no states", m.ID)
	}
	if len(p.Transitions) == 0 {
		return fmt.Errorf("machine %s: process declares no transitions", m.ID)
	}

	states := make(map[string]bool, len(p.States))
	for _, s := range p.States {
		if s == "" {
			return fmt.Errorf("machine %s: process declares an empty state name", m.ID)
		}
		if states[s] {
			return fmt.Errorf("machine %s: process declares state %q twice", m.ID, s)
		}
		states[s] = true
	}

	statusField, err := resolveStatusField(m, p.States)
	if err != nil {
		return err
	}

	// --- transitions → Events + Permission grouping -------------------------
	type actorKey struct{ role, ownerField string }
	permEvents := make(map[actorKey][]string)
	var permOrder []actorKey
	seenSlug := make(map[string]bool)
	var events []*model.Event

	for i, t := range p.Transitions {
		if t.Name == "" {
			return fmt.Errorf("machine %s: process transition %d has no name", m.ID, i)
		}
		s := slug(t.Name)
		if seenSlug[s] {
			return fmt.Errorf("machine %s: process transitions %q and an earlier one compile to the same id (slug %q) -- rename one", m.ID, t.Name, s)
		}
		seenSlug[s] = true
		if !states[t.From] {
			return fmt.Errorf("machine %s: transition %q's from-state %q is not a declared state", m.ID, t.Name, t.From)
		}
		if !states[t.To] {
			return fmt.Errorf("machine %s: transition %q's to-state %q is not a declared state", m.ID, t.Name, t.To)
		}
		if t.Actor.Role == "" {
			return fmt.Errorf("machine %s: transition %q declares no actor role -- a transition nobody may perform is a declaration error", m.ID, t.Name)
		}
		if t.Actor.OwnerField != "" {
			f := fieldByID(m, t.Actor.OwnerField)
			if f == nil {
				return fmt.Errorf("machine %s: transition %q's actor owner_field %q is not a field on this machine", m.ID, t.Name, t.Actor.OwnerField)
			}
			if f.Type != model.FieldTypeUser {
				return fmt.Errorf("machine %s: transition %q's actor owner_field %q must be a `user` field, is %q", m.ID, t.Name, t.Actor.OwnerField, f.Type)
			}
		}

		ev := &model.Event{
			ID:        "evt_" + m.ID + "_" + s,
			MachineID: m.ID,
			Name:      t.Name,
			Position:  i,
			Condition: &model.ConstraintExpression{Field: statusField.ID, Operator: "equals", Value: t.From},
			Actions: []*model.EventAction{{
				EventID:  "evt_" + m.ID + "_" + s,
				Type:     model.ActionSetField,
				Position: 0,
				Params:   map[string]any{"field": statusField.ID, "value": t.To},
			}},
		}
		for j, a := range t.Actions {
			if err := validateProcessAction(m, t.Name, a); err != nil {
				return err
			}
			ev.Actions = append(ev.Actions, &model.EventAction{
				EventID:  ev.ID,
				Type:     a.Type,
				Position: j + 1,
				Params:   a.Params,
			})
		}
		events = append(events, ev)

		k := actorKey{t.Actor.Role, t.Actor.OwnerField}
		if _, ok := permEvents[k]; !ok {
			permOrder = append(permOrder, k)
		}
		permEvents[k] = append(permEvents[k], ev.ID)
	}

	// --- auto transitions → System-chained Events (CAP-E05) -----------------
	autoBySrc := make(map[string]*model.Event, len(p.Auto))
	for _, a := range p.Auto {
		if !states[a.From] || !states[a.To] {
			return fmt.Errorf("machine %s: auto transition %s→%s names an undeclared state", m.ID, a.From, a.To)
		}
		if autoBySrc[a.From] != nil {
			return fmt.Errorf("machine %s: two auto transitions leave state %q -- an automatic step must be unambiguous", m.ID, a.From)
		}
		ev := &model.Event{
			ID:        "evt_" + m.ID + "_auto_" + slug(a.From) + "_" + slug(a.To),
			MachineID: m.ID,
			Name:      "Auto: " + a.From + " → " + a.To,
			Position:  len(p.Transitions) + len(autoBySrc),
			Condition: &model.ConstraintExpression{Field: statusField.ID, Operator: "equals", Value: a.From},
			Actions: []*model.EventAction{{
				Type:     model.ActionSetField,
				Position: 0,
				Params:   map[string]any{"field": statusField.ID, "value": a.To},
			}},
		}
		ev.Actions[0].EventID = ev.ID
		autoBySrc[a.From] = ev
	}
	// A cycle in the auto chain would loop trigger_event forever at runtime.
	for _, a := range p.Auto {
		seen := map[string]bool{}
		for cur := a.From; ; {
			next, ok := autoBySrc[cur]
			if !ok {
				break
			}
			if seen[cur] {
				return fmt.Errorf("machine %s: auto transitions form a cycle through state %q", m.ID, cur)
			}
			seen[cur] = true
			cur = targetState(next)
		}
	}
	// Chain: every compiled event landing on an auto's from-state fires it.
	for _, ev := range events {
		if auto := autoBySrc[targetState(ev)]; auto != nil {
			ev.Actions = append(ev.Actions, &model.EventAction{
				EventID:  ev.ID,
				Type:     model.ActionTriggerEvent,
				Position: len(ev.Actions),
				Params:   map[string]any{"event": auto.ID},
			})
		}
	}
	for _, auto := range autoBySrc {
		if next := autoBySrc[targetState(auto)]; next != nil {
			auto.Actions = append(auto.Actions, &model.EventAction{
				EventID:  auto.ID,
				Type:     model.ActionTriggerEvent,
				Position: len(auto.Actions),
				Params:   map[string]any{"event": next.ID},
			})
		}
	}
	for _, a := range p.Auto {
		events = append(events, autoBySrc[a.From])
	}

	// --- SLA → due-date Field + scheduled breach Event (CAP-W04) ------------
	// Must run after every transition/auto Event above is built (it patches
	// them by target state) and before m.Events is assigned below.
	events, err = compileSLA(m, p, statusField, events, states)
	if err != nil {
		return err
	}

	// --- reachability: every state must be reachable from States[0] ---------
	reached := map[string]bool{p.States[0]: true}
	for changed := true; changed; {
		changed = false
		for _, t := range p.Transitions {
			if reached[t.From] && !reached[t.To] {
				reached[t.To] = true
				changed = true
			}
		}
		for _, a := range p.Auto {
			if reached[a.From] && !reached[a.To] {
				reached[a.To] = true
				changed = true
			}
		}
		// An SLA breach's escalate_to is also a real edge -- a state ONLY
		// reachable via SLA escalation is still reachable, not dead.
		for _, s := range p.SLA {
			if s.OnBreach.EscalateTo != "" && reached[s.State] && !reached[s.OnBreach.EscalateTo] {
				reached[s.OnBreach.EscalateTo] = true
				changed = true
			}
		}
	}
	for _, s := range p.States {
		if !reached[s] {
			return fmt.Errorf("machine %s: state %q is unreachable from the initial state %q", m.ID, s, p.States[0])
		}
	}

	// --- requirements → counter Fields + gating Constraints (CAP-W01) -------
	if err := compileRequirements(m, p, statusField); err != nil {
		return err
	}

	// --- permissions: one row per distinct (role, owner_field) pair ---------
	// Auto events deliberately appear in no Permission: they are only ever
	// fired System-side through the trigger_event chain; deny-by-default
	// (CAP-P05/Guard) keeps them off every user's buttons and blocks a
	// direct HTTP trigger.
	var perms []*model.Permission
	for _, k := range permOrder {
		id := "perm_" + m.ID + "_" + slug(k.role)
		if k.ownerField != "" {
			id += "_owner"
		}
		perms = append(perms, &model.Permission{
			ID:         id,
			MachineID:  m.ID,
			Role:       k.role,
			Events:     permEvents[k],
			OwnerField: k.ownerField,
			CanRead:    true,
			CanCreate:  true,
			CanEdit:    true,
		})
	}

	m.Events = events
	m.Permissions = append(m.Permissions, perms...)
	return nil
}

// resolveStatusField returns the Field the compiled guards and set_field
// actions operate on: the Machine's own value_list Field named "Status"
// when one exists (its declared values must cover every process state), or
// a generated one ("fld_<machine>_status", values = the declared states,
// first state = initial value per the existing first-value-is-default
// convention).
func resolveStatusField(m *model.Machine, states []string) (*model.Field, error) {
	for _, f := range m.Fields {
		if strings.EqualFold(f.Name, "Status") {
			if f.Type != model.FieldTypeValueList {
				return nil, fmt.Errorf("machine %s: process needs field %q to be a value_list to hold its states, is %q", m.ID, f.Name, f.Type)
			}
			declared := make(map[string]bool, len(f.Options.Values))
			for _, v := range f.Options.Values {
				declared[v] = true
			}
			for _, s := range states {
				if !declared[s] {
					return nil, fmt.Errorf("machine %s: process state %q is not among the existing Status field's values -- add it there or drop the field to let the compiler generate one", m.ID, s)
				}
			}
			return f, nil
		}
	}
	f := &model.Field{
		ID:        "fld_" + m.ID + "_status",
		MachineID: m.ID,
		Name:      "Status",
		Type:      model.FieldTypeValueList,
		Position:  len(m.Fields),
		Options:   model.FieldOptions{Values: states},
	}
	m.Fields = append(m.Fields, f)
	return f, nil
}

// compileRequirements (CAP-W01, Process Overlay B3) expands every declared
// `evidence` ProcessRequirement into a generated counter Field + gating
// Constraint(s) on m. Requirements are deduped across ALL of p's transitions
// by (Type, Target) -- two transitions naming the same requirement share one
// counter and one pair of Constraints, since the counter describes a fact
// about the RECORD ("how much evidence is attached"), not about any one
// transition. If they disagree on the transition's own `to` state, that is
// a load-time error (fail-loud beats guessing at an OR-condition this
// runtime's Constraint grammar can't express).
//
// The counter's VALUE is never computed here -- see model.ProcessRequirement
// and handler.stampRequirementCounters for the write-time-fan-in half.
//
// `approval` requirements (CAP-W03's declarative quorum form) are only
// VALIDATED here -- single-machine data (p.Transitions, this machine's own)
// is all this needs. Nothing is generated on m for this type: it compiles to
// an EventAction injected onto the TARGET machine's own Events, which can
// only happen once every Machine has loaded -- see loader.go's
// compileApprovalRequirements, called from LoadAll after validateReferences.
func compileRequirements(m *model.Machine, p *model.Process, statusField *model.Field) error {
	type reqKey struct{ typ, target string }
	type reqInfo struct {
		min, max     int // max == -1 means unbounded
		toState      string
		firstTransit string
	}
	seen := make(map[reqKey]*reqInfo)
	var order []reqKey
	sawApproval := false

	transitionNames := make(map[string]bool, len(p.Transitions))
	transitionActorRole := make(map[string]string, len(p.Transitions))
	for _, t := range p.Transitions {
		transitionNames[t.Name] = true
		transitionActorRole[t.Name] = t.Actor.Role
	}

	for _, t := range p.Transitions {
		for _, r := range t.Requirements {
			if r.Type == "approval" {
				if r.Target == "" {
					return fmt.Errorf("machine %s: transition %q's approval requirement has no target", m.ID, t.Name)
				}
				if r.MinApprovals <= 0 {
					return fmt.Errorf("machine %s: transition %q's approval requirement on %q needs min_approvals > 0", m.ID, t.Name, r.Target)
				}
				if r.OnQuorumApproved == "" || r.OnQuorumRejected == "" {
					return fmt.Errorf("machine %s: transition %q's approval requirement on %q needs both on_quorum_approved and on_quorum_rejected", m.ID, t.Name, r.Target)
				}
				for _, name := range []string{r.OnQuorumApproved, r.OnQuorumRejected} {
					if !transitionNames[name] {
						return fmt.Errorf("machine %s: transition %q's approval requirement names %q, which is not a transition declared on this machine's own process", m.ID, t.Name, name)
					}
					// Enforced, not just conventional: a quorum-controlled
					// transition must never be directly human-triggerable,
					// or the quorum guarantee it exists to provide is
					// bypassable by anyone with that role.
					if role := transitionActorRole[name]; role != "System" {
						return fmt.Errorf("machine %s: transition %q is named as an approval-quorum outcome but its actor role is %q, not \"System\" -- a quorum-controlled transition must be System-only", m.ID, name, role)
					}
				}
				if sawApproval {
					return fmt.Errorf("machine %s: more than one \"approval\" requirement declared -- only one per machine is supported so far, no case has needed a second yet", m.ID)
				}
				sawApproval = true
				continue // nothing generated on m for this type -- see loader.go's compileApprovalRequirements
			}
			if r.Type != "evidence" {
				return fmt.Errorf("machine %s: transition %q declares requirement type %q -- only \"evidence\" and \"approval\" are implemented so far", m.ID, t.Name, r.Type)
			}
			if r.Target == "" {
				return fmt.Errorf("machine %s: transition %q's requirement has no target", m.ID, t.Name)
			}
			min, max, err := parseCardinality(r.Cardinality)
			if err != nil {
				return fmt.Errorf("machine %s: transition %q's requirement on %q: %w", m.ID, t.Name, r.Target, err)
			}
			k := reqKey{r.Type, r.Target}
			if existing, ok := seen[k]; ok {
				if existing.toState != t.To {
					return fmt.Errorf("machine %s: transitions %q and %q both require %q evidence from %q but land on different states (%q vs %q) -- this compiler can't express an OR condition, split the requirement or the target state", m.ID, existing.firstTransit, t.Name, r.Target, r.Target, existing.toState, t.To)
				}
				continue // same (type, target, to-state) declared again -- nothing new to compile
			}
			seen[k] = &reqInfo{min: min, max: max, toState: t.To, firstTransit: t.Name}
			order = append(order, k)
		}
	}

	for _, k := range order {
		info := seen[k]
		counterID := model.RequirementCounterFieldID(m.ID, k.target)
		if fieldByID(m, counterID) != nil {
			return fmt.Errorf("machine %s: generated requirement counter field %q collides with an existing field", m.ID, counterID)
		}
		m.Fields = append(m.Fields, &model.Field{
			ID:        counterID,
			MachineID: m.ID,
			Name:      k.target + " Count",
			Type:      model.FieldTypeNumber,
			Position:  len(m.Fields),
			Options:   model.FieldOptions{Default: "0"},
		})
		toGuard := &model.ConstraintExpression{Field: statusField.ID, Operator: "equals", Value: info.toState}
		m.Constraints = append(m.Constraints, &model.Constraint{
			ID:         "cst_" + m.ID + "_" + k.target + "_min",
			MachineID:  m.ID,
			Rule:       fmt.Sprintf("At least %d %s record(s) required.", info.min, k.target),
			Expression: model.ConstraintExpression{Field: counterID, Operator: "greater_than_or_equal", Value: fmt.Sprintf("%d", info.min)},
			Condition:  toGuard,
		})
		if info.max >= 0 {
			m.Constraints = append(m.Constraints, &model.Constraint{
				ID:         "cst_" + m.ID + "_" + k.target + "_max",
				MachineID:  m.ID,
				Rule:       fmt.Sprintf("At most %d %s record(s) allowed.", info.max, k.target),
				Expression: model.ConstraintExpression{Field: counterID, Operator: "less_than_or_equal", Value: fmt.Sprintf("%d", info.max)},
				Condition:  toGuard,
			})
		}
	}
	return nil
}

// parseCardinality reads "N" (exact), "N..*" (at least N), or "N..M" (a
// bounded range) into (min, max) -- max == -1 means unbounded.
func parseCardinality(s string) (min, max int, err error) {
	parseInt := func(v string) (int, error) {
		n := 0
		if v == "" {
			return 0, fmt.Errorf("empty number")
		}
		for _, r := range v {
			if r < '0' || r > '9' {
				return 0, fmt.Errorf("not a number: %q", v)
			}
			n = n*10 + int(r-'0')
		}
		return n, nil
	}
	if before, after, ok := strings.Cut(s, ".."); ok {
		min, err = parseInt(before)
		if err != nil {
			return 0, 0, fmt.Errorf("invalid cardinality %q: %w", s, err)
		}
		if after == "*" {
			max = -1
		} else if max, err = parseInt(after); err != nil {
			return 0, 0, fmt.Errorf("invalid cardinality %q: %w", s, err)
		}
		if max >= 0 && max < min {
			return 0, 0, fmt.Errorf("invalid cardinality %q: max is less than min", s)
		}
	} else {
		min, err = parseInt(s)
		if err != nil {
			return 0, 0, fmt.Errorf("invalid cardinality %q: %w", s, err)
		}
		max = min
	}
	if min < 1 {
		return 0, 0, fmt.Errorf("invalid cardinality %q: minimum must be at least 1", s)
	}
	return min, max, nil
}

// durationRe mirrors executor.go's own dateArithRe unit vocabulary exactly
// (Day(s)/Week(s)/Month(s)/Year(s)/Business Day(s)) but without a sign --
// compileSLA always prepends "today + ", so a duration is always a forward
// offset; a bare "N Unit" is all this needs to validate.
var durationRe = regexp.MustCompile(`^\d+ (Day|Days|Week|Weeks|Month|Months|Year|Years|Business Day|Business Days)$`)

// compileSLA (CAP-W04, Process Overlay B4) expands p.SLA into a due-date
// Field + due-date-stamping actions on every already-compiled Event landing
// on the declared state + one generated scheduled breach Event per entry.
// Entirely within m -- no cross-machine wiring, unlike Quorum's declarative
// form (deliberately not attempted this pass, see roadmap.md's B4 note).
func compileSLA(m *model.Machine, p *model.Process, statusField *model.Field, events []*model.Event, states map[string]bool) ([]*model.Event, error) {
	seenState := map[string]bool{}
	for _, entry := range p.SLA {
		if !states[entry.State] {
			return nil, fmt.Errorf("machine %s: sla names undeclared state %q", m.ID, entry.State)
		}
		if seenState[entry.State] {
			return nil, fmt.Errorf("machine %s: sla declares state %q twice", m.ID, entry.State)
		}
		seenState[entry.State] = true
		if !durationRe.MatchString(entry.Duration) {
			return nil, fmt.Errorf("machine %s: sla on %q has invalid duration %q -- expected \"N Unit\", e.g. \"2 Business Days\"", m.ID, entry.State, entry.Duration)
		}
		if entry.OnBreach.EscalateTo != "" && !states[entry.OnBreach.EscalateTo] {
			return nil, fmt.Errorf("machine %s: sla on %q's escalate_to %q is not a declared state", m.ID, entry.State, entry.OnBreach.EscalateTo)
		}

		dueFieldID := "fld_" + m.ID + "_" + slug(entry.State) + "_due"
		if fieldByID(m, dueFieldID) != nil {
			return nil, fmt.Errorf("machine %s: generated sla due-date field %q collides with an existing field", m.ID, dueFieldID)
		}
		m.Fields = append(m.Fields, &model.Field{
			ID:        dueFieldID,
			MachineID: m.ID,
			Name:      entry.State + " Due",
			Type:      model.FieldTypeDate,
			Position:  len(m.Fields),
		})

		// Stamp the due date on every already-compiled Event (transition or
		// auto) that lands the record on this SLA-bound state.
		for _, ev := range events {
			if targetState(ev) != entry.State {
				continue
			}
			ev.Actions = append(ev.Actions, &model.EventAction{
				EventID:  ev.ID,
				Type:     model.ActionSetField,
				Position: len(ev.Actions),
				Params:   map[string]any{"field": dueFieldID, "value": "today + " + entry.Duration},
			})
		}

		if entry.OnBreach.Notify == nil {
			return nil, fmt.Errorf("machine %s: sla on %q declares no on_breach.notify", m.ID, entry.State)
		}
		breachID := "evt_" + m.ID + "_" + slug(entry.State) + "_sla_breach"
		breach := &model.Event{
			ID:        breachID,
			MachineID: m.ID,
			Name:      "SLA Breach: " + entry.State,
			Position:  len(events),
			Condition: &model.ConstraintExpression{Field: statusField.ID, Operator: "equals", Value: entry.State},
			Schedule:  &model.Schedule{DateField: dueFieldID, OffsetDays: 0},
			Actions: []*model.EventAction{{
				EventID:  breachID,
				Type:     model.ActionNotify,
				Position: 0,
				Params:   entry.OnBreach.Notify,
			}},
		}
		if entry.OnBreach.EscalateTo != "" {
			breach.Actions = append(breach.Actions, &model.EventAction{
				EventID:  breachID,
				Type:     model.ActionSetField,
				Position: 1,
				Params:   map[string]any{"field": statusField.ID, "value": entry.OnBreach.EscalateTo},
			})
		}
		events = append(events, breach)
	}
	return events, nil
}

// validateProcessAction fail-louds a declared on_transition action the
// compiled Event could not actually run -- an unknown action type, or a
// set_field naming a field this Machine doesn't have (CAP-X05 discipline:
// a silent no-op at runtime becomes an explicit load-time error).
func validateProcessAction(m *model.Machine, transition string, a *model.ProcessAction) error {
	switch a.Type {
	case model.ActionSetField:
		fieldID, _ := a.Params["field"].(string)
		if fieldByID(m, fieldID) == nil {
			return fmt.Errorf("machine %s: transition %q's set_field names unknown field %q", m.ID, transition, fieldID)
		}
	case model.ActionNotify, model.ActionCreateRecord, model.ActionCrossSetField, model.ActionBatchGenerate, model.ActionTriggerEvent:
		// Executor-supported; params are validated the same way hand-authored
		// event_actions rows are (i.e., downstream, at their own tier).
	default:
		return fmt.Errorf("machine %s: transition %q declares unknown action type %q", m.ID, transition, a.Type)
	}
	return nil
}

// targetState reads the state a compiled event leaves the record in -- its
// first action is always the compiler's own set_field on the status field.
func targetState(ev *model.Event) string {
	v, _ := ev.Actions[0].Params["value"].(string)
	return v
}

func fieldByID(m *model.Machine, id string) *model.Field {
	for _, f := range m.Fields {
		if f.ID == id {
			return f
		}
	}
	return nil
}

// slug lowercases and reduces a declared name to [a-z0-9_] for use inside a
// deterministic compiled id.
func slug(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
		default:
			b.WriteRune('_')
		}
	}
	return strings.Trim(b.String(), "_")
}

// findStatusField is resolveStatusField's read-only counterpart -- CAP-W07's
// change_policy needs to find the Machine's existing Status field (any
// Machine with real transitions already has one), never generate one the
// way Process Overlay's own resolveStatusField does.
func findStatusField(m *model.Machine) *model.Field {
	for _, f := range m.Fields {
		if strings.EqualFold(f.Name, "Status") {
			return f
		}
	}
	return nil
}

// compileChangePolicies (CAP-W07 -- see model.ChangePolicy's doc comment)
// compiles every Constraint's declared change_policy into that Constraint's
// own Condition. Runs after compileProcess so a process-overlay-compiled
// Machine's Status field already exists by the time records_in_states needs
// to resolve it.
func compileChangePolicies(m *model.Machine) error {
	for _, c := range m.Constraints {
		cp := c.ChangePolicy
		if cp == nil {
			continue
		}
		if c.Condition != nil {
			return fmt.Errorf("constraint %s on machine %s: change_policy cannot be combined with an existing condition on the same constraint (not supported yet -- attach change_policy only to a constraint with no other condition)", c.ID, m.ID)
		}
		switch cp.AppliesTo {
		case "all_records":
			// Explicit, intentionally inert -- matches today's default
			// behavior, named on purpose rather than left implicit.
		case "records_in_states":
			if len(cp.States) == 0 {
				return fmt.Errorf("constraint %s on machine %s: change_policy records_in_states needs at least one state", c.ID, m.ID)
			}
			statusField := findStatusField(m)
			if statusField == nil {
				return fmt.Errorf("constraint %s on machine %s: change_policy records_in_states needs a Status field on this machine", c.ID, m.ID)
			}
			declared := make(map[string]bool, len(statusField.Options.Values))
			for _, v := range statusField.Options.Values {
				declared[v] = true
			}
			for _, s := range cp.States {
				if !declared[s] {
					return fmt.Errorf("constraint %s on machine %s: change_policy state %q is not among the Status field's declared values", c.ID, m.ID, s)
				}
			}
			c.Condition = &model.ConstraintExpression{Field: statusField.ID, Operator: "in", Values: cp.States}
		case "new_records":
			if _, err := time.Parse("2006-01-02", cp.EffectiveFrom); err != nil {
				return fmt.Errorf("constraint %s on machine %s: change_policy new_records needs effective_from as YYYY-MM-DD, got %q", c.ID, m.ID, cp.EffectiveFrom)
			}
			c.Condition = &model.ConstraintExpression{Field: model.ChangePolicyCreatedAtField, Operator: "on_or_after", Value: cp.EffectiveFrom}
			m.NeedsCreatedAtGuard = true
		default:
			return fmt.Errorf("constraint %s on machine %s: change_policy has unrecognized applies_to %q", c.ID, m.ID, cp.AppliesTo)
		}
	}
	return nil
}

// compileApprovalRequirements (CAP-W03, declarative quorum form) resolves
// every `approval` ProcessRequirement declared anywhere and injects the same
// aggregate_status EventAction a hand-authored quorum pair
// (seeds/022_quorum_lab.sql) already carries by hand -- onto the TARGET
// machine's own Events, not the declaring machine's. Only possible here,
// after every Workspace/Application/Machine has fully loaded and compiled
// (called from LoadAll after validateReferences) -- this is exactly the
// cross-machine reach compileProcess/compileRequirements deliberately never
// attempt on their own (each only ever touches the one *model.Machine it's
// given).
func compileApprovalRequirements(workspaces []*model.Workspace) error {
	machineByID := make(map[string]*model.Machine)
	for _, ws := range workspaces {
		for _, app := range ws.Applications {
			for _, m := range app.Machines {
				machineByID[m.ID] = m
			}
		}
	}
	for _, ws := range workspaces {
		for _, app := range ws.Applications {
			for _, m := range app.Machines {
				if m.Process == nil {
					continue
				}
				for _, t := range m.Process.Transitions {
					for _, r := range t.Requirements {
						if r.Type != "approval" {
							continue
						}
						if err := injectApprovalQuorum(m, r, machineByID); err != nil {
							return err
						}
					}
				}
			}
		}
	}
	return nil
}

// injectApprovalQuorum does the actual cross-machine work for one `approval`
// requirement declared on m, targeting r.Target. Resolution order mirrors
// handler.doAggregateStatus's own runtime reads exactly, so what gets
// injected here is indistinguishable from what a human would type by hand:
// a value_list Field literally named "Decision" on the target (same
// heuristic CAP-A07/A08 already use elsewhere, not a new convention), a
// `reference` Field on the target pointing back at m (CAP-W01's target
// already required this via validateReferences, generically), and every
// Event on the target that sets Decision to any value gets the tally action
// appended -- doAggregateStatus recomputes fresh from all siblings on every
// call, so it doesn't matter which specific value triggered it.
func injectApprovalQuorum(m *model.Machine, r *model.ProcessRequirement, machineByID map[string]*model.Machine) error {
	target := machineByID[r.Target] // existence already guaranteed by validateReferences

	decisionField := model.FindFieldByName(target, "Decision")
	if decisionField == nil || decisionField.Type != model.FieldTypeValueList {
		return fmt.Errorf("machine %s: approval requirement on %q needs a value_list field named \"Decision\" on the target machine", m.ID, r.Target)
	}
	hasApproved, hasRejected := false, false
	for _, v := range decisionField.Options.Values {
		if v == "Approved" {
			hasApproved = true
		}
		if v == "Rejected" {
			hasRejected = true
		}
	}
	if !hasApproved || !hasRejected {
		return fmt.Errorf("machine %s: approval requirement on %q -- target's \"Decision\" field must declare both \"Approved\" and \"Rejected\" as values", m.ID, r.Target)
	}

	backRefField := model.FindReferenceFieldTo(target, m.ID)
	if backRefField == nil {
		return fmt.Errorf("machine %s: approval requirement on %q -- target has no reference field back to this machine", m.ID, r.Target)
	}

	var approvedEvt, rejectedEvt *model.Event
	for _, ev := range m.Events {
		if ev.Name == r.OnQuorumApproved {
			approvedEvt = ev
		}
		if ev.Name == r.OnQuorumRejected {
			rejectedEvt = ev
		}
	}
	if approvedEvt == nil || rejectedEvt == nil {
		return fmt.Errorf("machine %s: approval requirement's on_quorum_approved/on_quorum_rejected did not resolve to a compiled event -- this is a loader bug, not a metadata error, since compileRequirements already validated both names", m.ID)
	}

	// JSON numbers decode as float64 everywhere else in this codebase
	// (EventAction.Params is always populated from json.Unmarshal for
	// hand-authored/DB-loaded actions) -- handler.doAggregateStatus's own
	// params["min_approvals"].(float64) type assertion depends on that, so
	// this in-memory-constructed action must match it exactly or the
	// assertion silently fails and quorum never fires.
	params := map[string]any{
		"parent_field":                 backRefField.ID,
		"parent_event_if_all_approved": approvedEvt.ID,
		"parent_event_if_any_rejected": rejectedEvt.ID,
		"min_approvals":                float64(r.MinApprovals),
	}

	injected := 0
	for _, ev := range target.Events {
		setsDecision := false
		for _, a := range ev.Actions {
			if a.Type != model.ActionSetField {
				continue
			}
			if field, _ := a.Params["field"].(string); field == decisionField.ID {
				setsDecision = true
				break
			}
		}
		if !setsDecision {
			continue
		}
		ev.Actions = append(ev.Actions, &model.EventAction{
			EventID:  ev.ID,
			Type:     model.ActionAggregateStatus,
			Position: len(ev.Actions),
			Params:   params,
		})
		injected++
	}
	if injected == 0 {
		return fmt.Errorf("machine %s: approval requirement on %q -- target has no event that sets its own \"Decision\" field, nothing to attach quorum tallying to", m.ID, r.Target)
	}
	return nil
}
