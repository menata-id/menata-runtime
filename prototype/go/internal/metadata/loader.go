// Package metadata handles loading Runtime Metadata from PostgreSQL
// and building the Application Model used by the Interpreter.
package metadata

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"menata.id/runtime/internal/model"
)

// querier is the one method Loader actually calls -- satisfied by both
// *pgxpool.Pool (boot-time LoadAll, POST /admin/reload) and pgx.Tx (CAP-X08
// import: LoadAll run inside the still-open transaction that just inserted
// a new package, so a validation failure can roll the whole thing back
// before anything commits). Same "abstract over pool and tx" shape
// internal/store's own querier interface already uses, kept local here
// rather than shared, since metadata has no other reason to depend on
// package store.
type querier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

// Loader reads Runtime Metadata from the database.
type Loader struct {
	db querier
}

func NewLoader(db querier) *Loader {
	return &Loader{db: db}
}

// LoadAll loads every Workspace with its full Application Model tree.
func (l *Loader) LoadAll(ctx context.Context) ([]*model.Workspace, error) {
	workspaces, err := l.loadWorkspaces(ctx)
	if err != nil {
		return nil, fmt.Errorf("load workspaces: %w", err)
	}
	for _, ws := range workspaces {
		apps, err := l.loadApplications(ctx, ws.ID)
		if err != nil {
			return nil, fmt.Errorf("load applications for %s: %w", ws.ID, err)
		}
		for _, app := range apps {
			machines, err := l.loadMachines(ctx, app.ID)
			if err != nil {
				return nil, fmt.Errorf("load machines for %s: %w", app.ID, err)
			}
			app.Machines = machines
		}
		ws.Applications = apps
		ws.Holidays, err = l.loadHolidays(ctx, ws.ID)
		if err != nil {
			return nil, fmt.Errorf("load holidays for %s: %w", ws.ID, err)
		}
	}
	if err := validateReferences(workspaces); err != nil {
		return nil, err
	}
	// CAP-W03's declarative quorum form: an `approval` requirement injects
	// an EventAction onto a DIFFERENT (target) machine's own Events -- only
	// possible now, after every Workspace/Application/Machine above has
	// fully loaded and compiled. Runs after validateReferences on purpose:
	// that function already guarantees every requirement's target exists
	// and has a real back-reference field, so this one can trust that and
	// focus purely on resolving/injecting the compiled shape.
	if err := compileApprovalRequirements(workspaces); err != nil {
		return nil, err
	}
	if err := validateOperators(workspaces); err != nil {
		return nil, err
	}
	return workspaces, nil
}

// validateOperators (CAP-X05) rejects any Constraint expression/condition or
// Event condition naming an operator this runtime doesn't actually
// implement. Before this existed, an unrecognized operator (a typo, or a
// genuinely unsupported one like "greater_than_or_equal" before CAP-C05
// added it) didn't error anywhere — constraint.Eval's default case returns
// true ("satisfied") for any operator it doesn't recognize, so the
// constraint or guard silently never fired. That's a metadata-authoring
// trap this catches at load time instead: "Unknown = explicit"
// (capability-lifecycle.md §4 rule 3), same discipline as
// validateReferences' dangling-reference check.
func validateOperators(workspaces []*model.Workspace) error {
	checkExpr := func(expr *model.ConstraintExpression, ctx string) error {
		if expr == nil {
			return nil
		}
		if !model.SupportedOperators[expr.Operator] {
			return fmt.Errorf("%s: unrecognized operator %q", ctx, expr.Operator)
		}
		return nil
	}
	for _, ws := range workspaces {
		for _, app := range ws.Applications {
			for _, m := range app.Machines {
				for _, c := range m.Constraints {
					if err := checkExpr(&c.Expression, fmt.Sprintf("constraint %s on machine %s", c.ID, m.ID)); err != nil {
						return err
					}
					if err := checkExpr(c.Condition, fmt.Sprintf("constraint %s's condition on machine %s", c.ID, m.ID)); err != nil {
						return err
					}
				}
				for _, e := range m.Events {
					if err := checkExpr(e.Condition, fmt.Sprintf("event %s's condition on machine %s", e.ID, m.ID)); err != nil {
						return err
					}
					if ac := e.AggregateCondition; ac != nil && !model.SupportedOperators[ac.Operator] {
						return fmt.Errorf("event %s's aggregate_condition on machine %s: unrecognized operator %q", e.ID, m.ID, ac.Operator)
					}
				}
				for _, v := range m.Views {
					for _, fc := range v.Config.Filter {
						if !model.SupportedOperators[fc.Operator] {
							return fmt.Errorf("view %s's filter on machine %s: unrecognized operator %q", v.ID, m.ID, fc.Operator)
						}
					}
				}
			}
		}
	}
	return nil
}

// validateReferences enforces CAP-F13's load-time contract: every `reference`
// field's target_machine must resolve to a Machine that actually exists.
// Unlike a runtime surprise (a broken picker, a 500 on first use), a dangling
// reference is reported explicitly at load time — "Unknown = explicit"
// (capability-lifecycle.md §4 rule 3).
//
// Also enforces CAP-F05/CAP-P02's own version of the same discipline: a
// `permissions.owner_field` names a Field expected to hold a real person's
// account id (Guard.CanTrigger compares it against the acting user's id,
// internal/permission/guard.go) — that only means anything if the named
// Field is actually type `user`. Pointing owner_field at, say, a `text`
// field would silently never match anyone; caught here instead of
// discovered as "nobody can ever trigger this event."
func validateReferences(workspaces []*model.Workspace) error {
	known := make(map[string]bool)
	machineByID := make(map[string]*model.Machine)
	// CAP-I01: which Machine a given Event id belongs to -- a Subscription
	// names a PublisherEventID that can be on ANY Machine, not just this
	// one, so validating it (and its Contract's own field names) needs a
	// global index, the same reasoning machineByID already exists for.
	eventMachine := make(map[string]*model.Machine)
	for _, ws := range workspaces {
		for _, app := range ws.Applications {
			for _, m := range app.Machines {
				known[m.ID] = true
				machineByID[m.ID] = m
				for _, e := range m.Events {
					eventMachine[e.ID] = m
				}
			}
		}
	}
	for _, ws := range workspaces {
		for _, app := range ws.Applications {
			for _, m := range app.Machines {
				for _, f := range m.Fields {
					if f.Type != model.FieldTypeReference {
						continue
					}
					target := f.Options.TargetMachine
					if target == "" {
						return fmt.Errorf("field %s (%s) on machine %s: type reference requires target_machine",
							f.ID, f.Name, m.ID)
					}
					if !known[target] {
						return fmt.Errorf("field %s (%s) on machine %s: dangling reference — target_machine %q does not exist",
							f.ID, f.Name, m.ID, target)
					}
				}

				fieldByID := make(map[string]*model.Field, len(m.Fields))
				for _, f := range m.Fields {
					fieldByID[f.ID] = f
				}
				for _, p := range m.Permissions {
					if p.OwnerField == "" {
						continue
					}
					f, ok := fieldByID[p.OwnerField]
					if !ok {
						return fmt.Errorf("permission %s on machine %s: owner_field %q does not name a Field on this machine",
							p.ID, m.ID, p.OwnerField)
					}
					if f.Type != model.FieldTypeUser {
						return fmt.Errorf("permission %s on machine %s: owner_field %q must be type \"user\", got %q",
							p.ID, m.ID, p.OwnerField, f.Type)
					}
				}

				// CAP-C08: a Constraint's cross_record must resolve cleanly --
				// same "wait until everything's loaded" reasoning as CAP-W01's
				// requirement-target check right below (a ChildMachine or a
				// reference_field's target machine may not have loaded yet if
				// checked any earlier).
				for _, c := range m.Constraints {
					cr := c.CrossRecord
					if cr == nil {
						continue
					}
					if !model.SupportedOperators[cr.Operator] {
						return fmt.Errorf("constraint %s on machine %s: cross_record operator %q is not supported", c.ID, m.ID, cr.Operator)
					}
					switch cr.Kind {
					case "aggregate":
						child, ok := machineByID[cr.ChildMachine]
						if !ok {
							return fmt.Errorf("constraint %s on machine %s: cross_record aggregate child_machine %q does not exist", c.ID, m.ID, cr.ChildMachine)
						}
						childFieldByID := make(map[string]*model.Field, len(child.Fields))
						for _, cf := range child.Fields {
							childFieldByID[cf.ID] = cf
						}
						if sf, ok := childFieldByID[cr.ScopeField]; !ok || sf.Type != model.FieldTypeReference || sf.Options.TargetMachine != m.ID {
							return fmt.Errorf("constraint %s on machine %s: cross_record scope_field %q must be a reference field on %s pointing back at %s", c.ID, m.ID, cr.ScopeField, cr.ChildMachine, m.ID)
						}
						if fa, ok := childFieldByID[cr.FieldA]; !ok || fa.Type != model.FieldTypeNumber {
							return fmt.Errorf("constraint %s on machine %s: cross_record field_a %q must be a number field on %s", c.ID, m.ID, cr.FieldA, cr.ChildMachine)
						}
						if cr.FieldB != "" {
							if fb, ok := childFieldByID[cr.FieldB]; !ok || fb.Type != model.FieldTypeNumber {
								return fmt.Errorf("constraint %s on machine %s: cross_record field_b %q must be a number field on %s", c.ID, m.ID, cr.FieldB, cr.ChildMachine)
							}
						}
					case "reference_field":
						refField, ok := fieldByID[cr.ReferenceField]
						if !ok || refField.Type != model.FieldTypeReference {
							return fmt.Errorf("constraint %s on machine %s: cross_record reference_field %q must be a reference field on this machine", c.ID, m.ID, cr.ReferenceField)
						}
						target, ok := machineByID[refField.Options.TargetMachine]
						if !ok {
							return fmt.Errorf("constraint %s on machine %s: cross_record reference_field %q targets a machine that does not exist", c.ID, m.ID, cr.ReferenceField)
						}
						found := false
						for _, tf := range target.Fields {
							if tf.ID == cr.TargetField {
								found = true
								break
							}
						}
						if !found {
							return fmt.Errorf("constraint %s on machine %s: cross_record target_field %q does not name a Field on %s", c.ID, m.ID, cr.TargetField, target.ID)
						}
					default:
						return fmt.Errorf("constraint %s on machine %s: cross_record has unrecognized kind %q", c.ID, m.ID, cr.Kind)
					}
				}

				// CAP-W01 (Process Overlay B3): a Requirement's target must
				// name a real Machine, and that Machine must itself declare
				// a `reference` Field pointing back at m -- same "Unknown =
				// explicit" discipline as CAP-F16's child_lines check below,
				// and the reason this validation waits until here rather
				// than running inside compileProcess itself: compileProcess
				// runs mid-load, before every Application (and so every
				// Machine a Requirement might target) has necessarily
				// loaded yet.
				if m.Process != nil {
					for _, t := range m.Process.Transitions {
						for _, r := range t.Requirements {
							target, ok := machineByID[r.Target]
							if !ok {
								return fmt.Errorf("machine %s: transition %q's requirement target %q does not exist", m.ID, t.Name, r.Target)
							}
							backRef := false
							for _, tf := range target.Fields {
								if tf.Type == model.FieldTypeReference && tf.Options.TargetMachine == m.ID {
									backRef = true
									break
								}
							}
							if !backRef {
								return fmt.Errorf("machine %s: transition %q's requirement target %q has no `reference` field pointing back at %s", m.ID, t.Name, r.Target, m.ID)
							}
						}
					}
				}

				// CAP-F08/F17: a `money` field must declare exactly one of
				// Currency (fixed code) or CurrencyField (a reference to
				// another field on the same record, for per-transaction
				// currency) -- same "Unknown = explicit" discipline; closes
				// the gap CAP-X05's own row named as deferred ("waits for
				// CAP-F08's real implementation").
				for _, f := range m.Fields {
					if f.Type != model.FieldTypeMoney {
						continue
					}
					hasCurrency, hasField := f.Options.Currency != "", f.Options.CurrencyField != ""
					if hasCurrency == hasField {
						return fmt.Errorf("field %s (%s) on machine %s: type money requires exactly one of options.currency or options.currency_field",
							f.ID, f.Name, m.ID)
					}
					if hasField {
						if _, ok := fieldByID[f.Options.CurrencyField]; !ok {
							return fmt.Errorf("field %s (%s) on machine %s: currency_field %q does not name a Field on this machine",
								f.ID, f.Name, m.ID, f.Options.CurrencyField)
						}
					}
				}

				// CAP-F14: a `computed` field's source_field must name a
				// real number/money Field on the same machine -- the
				// arithmetic only makes sense against a numeric value.
				for _, f := range m.Fields {
					if f.Type != model.FieldTypeComputed {
						continue
					}
					sf, ok := fieldByID[f.Options.SourceField]
					if !ok {
						return fmt.Errorf("field %s (%s) on machine %s: computed source_field %q does not name a Field on this machine",
							f.ID, f.Name, m.ID, f.Options.SourceField)
					}
					if sf.Type != model.FieldTypeNumber && sf.Type != model.FieldTypeMoney {
						return fmt.Errorf("field %s (%s) on machine %s: computed source_field %q must be type \"number\" or \"money\", got %q",
							f.ID, f.Name, m.ID, f.Options.SourceField, sf.Type)
					}
					if f.Options.FactorField != "" {
						ff, ok := fieldByID[f.Options.FactorField]
						if !ok {
							return fmt.Errorf("field %s (%s) on machine %s: computed factor_field %q does not name a Field on this machine",
								f.ID, f.Name, m.ID, f.Options.FactorField)
						}
						if ff.Type != model.FieldTypeNumber && ff.Type != model.FieldTypeMoney {
							return fmt.Errorf("field %s (%s) on machine %s: computed factor_field %q must be type \"number\" or \"money\", got %q",
								f.ID, f.Name, m.ID, f.Options.FactorField, ff.Type)
						}
					}
				}

				// CAP-F18: auto_number_prefix only makes sense on a `text`
				// field -- Create's own auto-number generation (handler.go)
				// writes a formatted string, which would silently mismatch
				// any other declared type.
				for _, f := range m.Fields {
					if f.Options.AutoNumberPrefix != "" && f.Type != model.FieldTypeText {
						return fmt.Errorf("field %s (%s) on machine %s: auto_number_prefix requires type \"text\", got %q",
							f.ID, f.Name, m.ID, f.Type)
					}
				}

				// CAP-F16: a form view's child_lines must name a real child
				// Machine with a real `reference` field pointing back at
				// this parent -- same "Unknown = explicit" discipline.
				for _, v := range m.Views {
					cl := v.Config.ChildLines
					if cl == nil {
						continue
					}
					child, ok := machineByID[cl.Machine]
					if !ok {
						return fmt.Errorf("view %s on machine %s: child_lines.machine %q does not exist", v.ID, m.ID, cl.Machine)
					}
					var parentField *model.Field
					for _, cf := range child.Fields {
						if cf.ID == cl.ParentField {
							parentField = cf
							break
						}
					}
					if parentField == nil {
						return fmt.Errorf("view %s on machine %s: child_lines.parent_field %q does not name a Field on machine %s",
							v.ID, m.ID, cl.ParentField, cl.Machine)
					}
					if parentField.Type != model.FieldTypeReference || parentField.Options.TargetMachine != m.ID {
						return fmt.Errorf("view %s on machine %s: child_lines.parent_field %q must be a reference field targeting %s, got type %q targeting %q",
							v.ID, m.ID, cl.ParentField, m.ID, parentField.Type, parentField.Options.TargetMachine)
					}
				}

				// CAP-V04/V05/V09: a list view's default_sort.field and every
				// filter condition's field must name a real Field on this
				// machine -- same "Unknown = explicit" discipline as
				// child_lines above, so a typo'd field id fails at load time
				// instead of silently sorting/filtering on nothing.
				// "created_at"/"updated_at" are exempt -- real columns every
				// record has, not a JSONB Field (store.RecordStore.List's
				// own reserved-name handling).
				for _, v := range m.Views {
					if ds := v.Config.DefaultSort; ds != nil && ds.Field != "" && ds.Field != "created_at" && ds.Field != "updated_at" {
						if _, ok := fieldByID[ds.Field]; !ok {
							return fmt.Errorf("view %s on machine %s: default_sort.field %q does not name a Field on this machine", v.ID, m.ID, ds.Field)
						}
					}
					for _, fc := range v.Config.Filter {
						if _, ok := fieldByID[fc.Field]; !ok {
							return fmt.Errorf("view %s on machine %s: filter field %q does not name a Field on this machine", v.ID, m.ID, fc.Field)
						}
					}

					// CAP-V07: calendar/timeline's date_field is one of
					// THIS machine's own Fields (the records being
					// plotted), not another machine's.
					if v.Config.DateField != "" {
						if _, ok := fieldByID[v.Config.DateField]; !ok {
							return fmt.Errorf("view %s on machine %s: date_field %q does not name a Field on this machine", v.ID, m.ID, v.Config.DateField)
						}
					}

					// CAP-V12: every wizard step's field ids must be real
					// Fields on this machine -- same discipline as Fields
					// itself already gets nowhere else, but Steps is new.
					for si, step := range v.Config.Steps {
						for _, fid := range step {
							if _, ok := fieldByID[fid]; !ok {
								return fmt.Errorf("view %s on machine %s: steps[%d] field %q does not name a Field on this machine", v.ID, m.ID, si, fid)
							}
						}
					}

					// CAP-V13: a report view aggregates ANOTHER machine's
					// records -- group_field and every sum_field must name
					// real Fields on THAT machine, not this one.
					if rc := v.Config.Report; rc != nil {
						src, ok := machineByID[rc.Machine]
						if !ok {
							return fmt.Errorf("view %s on machine %s: report.machine %q does not exist", v.ID, m.ID, rc.Machine)
						}
						srcFields := make(map[string]bool, len(src.Fields))
						for _, sf := range src.Fields {
							srcFields[sf.ID] = true
						}
						if !srcFields[rc.GroupField] {
							return fmt.Errorf("view %s on machine %s: report.group_field %q does not name a Field on machine %s", v.ID, m.ID, rc.GroupField, rc.Machine)
						}
						for _, sf := range rc.SumFields {
							if !srcFields[sf] {
								return fmt.Errorf("view %s on machine %s: report.sum_fields %q does not name a Field on machine %s", v.ID, m.ID, sf, rc.Machine)
							}
						}
					}

					// CAP-V10: each dashboard section's machine must exist,
					// and its group_field (if any) must be a real Field on
					// THAT section's own machine.
					for si, sec := range v.Config.Sections {
						src, ok := machineByID[sec.Machine]
						if !ok {
							return fmt.Errorf("view %s on machine %s: sections[%d].machine %q does not exist", v.ID, m.ID, si, sec.Machine)
						}
						if sec.GroupField == "" {
							continue
						}
						found := false
						for _, sf := range src.Fields {
							if sf.ID == sec.GroupField {
								found = true
								break
							}
						}
						if !found {
							return fmt.Errorf("view %s on machine %s: sections[%d].group_field %q does not name a Field on machine %s", v.ID, m.ID, si, sec.GroupField, sec.Machine)
						}
					}
				}

				// CAP-P04: an Event's input_fields must name real Fields on
				// this same machine -- the picker rendered alongside the
				// trigger button only makes sense for a Field that exists.
				for _, e := range m.Events {
					for _, fid := range e.InputFields {
						if _, ok := fieldByID[fid]; !ok {
							return fmt.Errorf("event %s on machine %s: input_fields %q does not name a Field on this machine", e.ID, m.ID, fid)
						}
					}
				}

				// CAP-E02/E03: exactly one of time/date_field, and
				// date_field (when set) must name a real `date` Field on
				// this machine -- a schedule keyed on a nonexistent or
				// wrong-typed field would silently never fire, the same
				// "Unknown = explicit" discipline as everywhere else.
				for _, e := range m.Events {
					s := e.Schedule
					if s == nil {
						continue
					}
					if (s.Time == "") == (s.DateField == "") {
						return fmt.Errorf("event %s on machine %s: schedule must set exactly one of time or date_field", e.ID, m.ID)
					}
					if s.Time != "" {
						if _, err := time.Parse("15:04", s.Time); err != nil {
							return fmt.Errorf("event %s on machine %s: schedule.time %q is not HH:MM", e.ID, m.ID, s.Time)
						}
					}
					if s.DateField != "" {
						f, ok := fieldByID[s.DateField]
						if !ok {
							return fmt.Errorf("event %s on machine %s: schedule.date_field %q does not name a Field on this machine", e.ID, m.ID, s.DateField)
						}
						if f.Type != model.FieldTypeDate {
							return fmt.Errorf("event %s on machine %s: schedule.date_field %q must be type \"date\", got %q", e.ID, m.ID, s.DateField, f.Type)
						}
					}
				}

				// CAP-I01/I03: a Subscription's publisher_event_id must
				// name a real Event (anywhere -- cross-machine by design);
				// its fields mapping's target keys must be real Fields on
				// THIS (subscriber) machine; its Contract's own field
				// names must be real Fields on the PUBLISHER's machine
				// (the data a Contract checks is the publisher's, not the
				// subscriber's); on_violation must be a value the handler
				// actually understands.
				for _, sub := range m.Subscriptions {
					pubMachine, ok := eventMachine[sub.PublisherEventID]
					if !ok {
						return fmt.Errorf("subscription %s on machine %s: publisher_event_id %q does not name a real Event", sub.ID, m.ID, sub.PublisherEventID)
					}
					for target := range sub.Fields {
						if _, ok := fieldByID[target]; !ok {
							return fmt.Errorf("subscription %s on machine %s: fields target %q does not name a Field on this machine", sub.ID, m.ID, target)
						}
					}
					pubFieldByID := make(map[string]bool, len(pubMachine.Fields))
					for _, pf := range pubMachine.Fields {
						pubFieldByID[pf.ID] = true
					}
					for _, c := range sub.Contract {
						if !pubFieldByID[c.Field] {
							return fmt.Errorf("subscription %s on machine %s: contract field %q does not name a Field on publisher machine %s", sub.ID, m.ID, c.Field, pubMachine.ID)
						}
						if !model.SupportedOperators[c.Operator] {
							return fmt.Errorf("subscription %s on machine %s: contract operator %q is not supported", sub.ID, m.ID, c.Operator)
						}
					}
					if sub.OnViolation != "skip" && sub.OnViolation != "log_only" {
						return fmt.Errorf("subscription %s on machine %s: on_violation %q must be \"skip\" or \"log_only\"", sub.ID, m.ID, sub.OnViolation)
					}
				}
			}
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

func (l *Loader) loadWorkspaces(ctx context.Context) ([]*model.Workspace, error) {
	rows, err := l.db.Query(ctx, `SELECT id, name FROM workspaces ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*model.Workspace
	for rows.Next() {
		ws := &model.Workspace{}
		if err := rows.Scan(&ws.ID, &ws.Name); err != nil {
			return nil, err
		}
		out = append(out, ws)
	}
	return out, rows.Err()
}

// loadHolidays (CAP-O06) loads a Workspace's own declared non-working
// dates, once at boot -- the same "in-memory index, no DB access at
// request time" posture every other piece of Runtime Metadata already
// gets, consumed by CAP-A11's "N Business Days" date arithmetic.
func (l *Loader) loadHolidays(ctx context.Context, workspaceID string) ([]string, error) {
	rows, err := l.db.Query(ctx,
		`SELECT holiday_date::text FROM workspace_holidays WHERE workspace_id = $1`,
		workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []string
	for rows.Next() {
		var d string
		if err := rows.Scan(&d); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

func (l *Loader) loadApplications(ctx context.Context, workspaceID string) ([]*model.Application, error) {
	rows, err := l.db.Query(ctx,
		`SELECT id, workspace_id, name FROM applications WHERE workspace_id = $1 ORDER BY name`,
		workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*model.Application
	for rows.Next() {
		app := &model.Application{}
		if err := rows.Scan(&app.ID, &app.WorkspaceID, &app.Name); err != nil {
			return nil, err
		}
		out = append(out, app)
	}
	return out, rows.Err()
}

func (l *Loader) loadMachines(ctx context.Context, applicationID string) ([]*model.Machine, error) {
	rows, err := l.db.Query(ctx,
		`SELECT id, application_id, name, config::text, process::text FROM machines WHERE application_id = $1 ORDER BY name`,
		applicationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var machines []*model.Machine
	for rows.Next() {
		m := &model.Machine{}
		var configJSON, processJSON *string
		if err := rows.Scan(&m.ID, &m.ApplicationID, &m.Name, &configJSON, &processJSON); err != nil {
			return nil, err
		}
		if configJSON != nil {
			m.Config = make(map[string]string)
			if err := json.Unmarshal([]byte(*configJSON), &m.Config); err != nil {
				return nil, fmt.Errorf("parse config for machine %s: %w", m.ID, err)
			}
		}
		if processJSON != nil {
			m.Process = &model.Process{}
			if err := json.Unmarshal([]byte(*processJSON), m.Process); err != nil {
				return nil, fmt.Errorf("parse process for machine %s: %w", m.ID, err)
			}
		}
		machines = append(machines, m)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	for _, m := range machines {
		if err := l.loadMachineDetails(ctx, m); err != nil {
			return nil, fmt.Errorf("load details for machine %s: %w", m.ID, err)
		}
	}
	return machines, nil
}

func (l *Loader) loadMachineDetails(ctx context.Context, m *model.Machine) error {
	var err error
	m.Fields, err = l.loadFields(ctx, m.ID)
	if err != nil {
		return err
	}
	m.Events, err = l.loadEvents(ctx, m.ID)
	if err != nil {
		return err
	}
	m.Constraints, err = l.loadConstraints(ctx, m.ID)
	if err != nil {
		return err
	}
	m.Permissions, err = l.loadPermissions(ctx, m.ID)
	if err != nil {
		return err
	}
	m.Views, err = l.loadViews(ctx, m.ID)
	if err != nil {
		return err
	}
	m.Subscriptions, err = l.loadSubscriptions(ctx, m.ID)
	if err != nil {
		return err
	}
	// Process Overlay B1: expand a declared process into Events/guards/
	// Permissions (compile.go) once everything hand-authored is loaded --
	// downstream (validateReferences/validateOperators, Interpreter, Guard,
	// Executor) then treats the compiled result exactly like hand-authored
	// metadata, including re-validating it.
	if err := compileProcess(m); err != nil {
		return err
	}
	// CAP-W07: compiles each Constraint's change_policy (if any) into its
	// own Condition -- needs this Machine's own Fields (the Status field)
	// already loaded above, which they are by this point.
	return compileChangePolicies(m)
}

func (l *Loader) loadFields(ctx context.Context, machineID string) ([]*model.Field, error) {
	rows, err := l.db.Query(ctx,
		`SELECT id, machine_id, name, type, position, required, options::text
		 FROM fields WHERE machine_id = $1 ORDER BY position`,
		machineID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*model.Field
	for rows.Next() {
		f := &model.Field{}
		var typeStr, optionsJSON string
		if err := rows.Scan(&f.ID, &f.MachineID, &f.Name, &typeStr, &f.Position, &f.Required, &optionsJSON); err != nil {
			return nil, err
		}
		f.Type = model.FieldType(typeStr)
		if err := json.Unmarshal([]byte(optionsJSON), &f.Options); err != nil {
			return nil, fmt.Errorf("parse options for field %s: %w", f.ID, err)
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

func (l *Loader) loadEvents(ctx context.Context, machineID string) ([]*model.Event, error) {
	rows, err := l.db.Query(ctx,
		`SELECT id, machine_id, name, position, condition::text, input_fields, schedule::text,
		        COALESCE(category, ''), COALESCE(schema_version, ''), COALESCE(deprecated_message, '')
		 FROM events WHERE machine_id = $1 ORDER BY position`,
		machineID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []*model.Event
	for rows.Next() {
		e := &model.Event{}
		var condJSON, schedJSON *string
		if err := rows.Scan(&e.ID, &e.MachineID, &e.Name, &e.Position, &condJSON, &e.InputFields, &schedJSON,
			&e.Category, &e.SchemaVersion, &e.DeprecatedMessage); err != nil {
			return nil, err
		}
		if schedJSON != nil {
			e.Schedule = &model.Schedule{}
			if err := json.Unmarshal([]byte(*schedJSON), e.Schedule); err != nil {
				return nil, fmt.Errorf("parse schedule for event %s: %w", e.ID, err)
			}
		}
		if condJSON != nil {
			// CAP-A14: the same `condition` column holds either shape --
			// disambiguated by an "aggregate_field" key, since an ordinary
			// CAP-E06 guard and an aggregate one never both apply to the
			// same Event.
			var probe map[string]any
			if err := json.Unmarshal([]byte(*condJSON), &probe); err != nil {
				return nil, fmt.Errorf("parse condition for event %s: %w", e.ID, err)
			}
			if _, isAggregate := probe["aggregate_field"]; isAggregate {
				e.AggregateCondition = &model.AggregateCondition{}
				if err := json.Unmarshal([]byte(*condJSON), e.AggregateCondition); err != nil {
					return nil, fmt.Errorf("parse aggregate condition for event %s: %w", e.ID, err)
				}
				if e.AggregateCondition.Machine == "" {
					e.AggregateCondition.Machine = machineID
				}
			} else {
				e.Condition = &model.ConstraintExpression{}
				if err := json.Unmarshal([]byte(*condJSON), e.Condition); err != nil {
					return nil, fmt.Errorf("parse condition for event %s: %w", e.ID, err)
				}
			}
		}
		events = append(events, e)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	for _, e := range events {
		e.Actions, err = l.loadEventActions(ctx, e.ID)
		if err != nil {
			return nil, fmt.Errorf("load actions for event %s: %w", e.ID, err)
		}
	}
	return events, nil
}

// loadSubscriptions (CAP-I01) loads machineID's own declared interest in
// OTHER machines' Events -- keyed by subscriber, the same "who's
// listening" direction Pattern C requires (a publisher never enumerates
// its own subscribers).
func (l *Loader) loadSubscriptions(ctx context.Context, machineID string) ([]*model.Subscription, error) {
	rows, err := l.db.Query(ctx,
		`SELECT id, machine_id, publisher_event_id, fields::text, contract::text, on_violation
		 FROM event_subscriptions WHERE machine_id = $1 ORDER BY position`,
		machineID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var subs []*model.Subscription
	for rows.Next() {
		s := &model.Subscription{}
		var fieldsJSON string
		var contractJSON *string
		if err := rows.Scan(&s.ID, &s.MachineID, &s.PublisherEventID, &fieldsJSON, &contractJSON, &s.OnViolation); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(fieldsJSON), &s.Fields); err != nil {
			return nil, fmt.Errorf("parse fields for subscription %s: %w", s.ID, err)
		}
		if contractJSON != nil {
			if err := json.Unmarshal([]byte(*contractJSON), &s.Contract); err != nil {
				return nil, fmt.Errorf("parse contract for subscription %s: %w", s.ID, err)
			}
		}
		subs = append(subs, s)
	}
	return subs, rows.Err()
}

func (l *Loader) loadEventActions(ctx context.Context, eventID string) ([]*model.EventAction, error) {
	rows, err := l.db.Query(ctx,
		`SELECT id, event_id, type, position, params::text
		 FROM event_actions WHERE event_id = $1 ORDER BY position`,
		eventID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*model.EventAction
	for rows.Next() {
		a := &model.EventAction{}
		var typeStr, paramsJSON string
		if err := rows.Scan(&a.ID, &a.EventID, &typeStr, &a.Position, &paramsJSON); err != nil {
			return nil, err
		}
		a.Type = model.ActionType(typeStr)
		if err := json.Unmarshal([]byte(paramsJSON), &a.Params); err != nil {
			return nil, fmt.Errorf("parse params for action in event %s: %w", eventID, err)
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (l *Loader) loadConstraints(ctx context.Context, machineID string) ([]*model.Constraint, error) {
	rows, err := l.db.Query(ctx,
		`SELECT id, machine_id, rule, expression::text, condition::text, position, change_policy::text, cross_record::text
		 FROM constraints WHERE machine_id = $1 ORDER BY position`,
		machineID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*model.Constraint
	for rows.Next() {
		c := &model.Constraint{}
		var exprJSON string
		var condJSON *string
		var changePolicyJSON *string
		var crossRecordJSON *string
		if err := rows.Scan(&c.ID, &c.MachineID, &c.Rule, &exprJSON, &condJSON, &c.Position, &changePolicyJSON, &crossRecordJSON); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(exprJSON), &c.Expression); err != nil {
			return nil, fmt.Errorf("parse expression for constraint %s: %w", c.ID, err)
		}
		if condJSON != nil {
			c.Condition = &model.ConstraintExpression{}
			if err := json.Unmarshal([]byte(*condJSON), c.Condition); err != nil {
				return nil, fmt.Errorf("parse condition for constraint %s: %w", c.ID, err)
			}
		}
		if changePolicyJSON != nil {
			c.ChangePolicy = &model.ChangePolicy{}
			if err := json.Unmarshal([]byte(*changePolicyJSON), c.ChangePolicy); err != nil {
				return nil, fmt.Errorf("parse change_policy for constraint %s: %w", c.ID, err)
			}
		}
		if crossRecordJSON != nil {
			c.CrossRecord = &model.CrossRecordCheck{}
			if err := json.Unmarshal([]byte(*crossRecordJSON), c.CrossRecord); err != nil {
				return nil, fmt.Errorf("parse cross_record for constraint %s: %w", c.ID, err)
			}
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (l *Loader) loadPermissions(ctx context.Context, machineID string) ([]*model.Permission, error) {
	rows, err := l.db.Query(ctx,
		`SELECT id, machine_id, role, events, owner_field, can_read, can_create, can_edit, can_delete, hidden_fields
		 FROM permissions WHERE machine_id = $1`,
		machineID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*model.Permission
	for rows.Next() {
		p := &model.Permission{}
		var ownerField *string
		if err := rows.Scan(&p.ID, &p.MachineID, &p.Role, &p.Events, &ownerField, &p.CanRead, &p.CanCreate, &p.CanEdit, &p.CanDelete, &p.HiddenFields); err != nil {
			return nil, err
		}
		if ownerField != nil {
			p.OwnerField = *ownerField
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (l *Loader) loadViews(ctx context.Context, machineID string) ([]*model.View, error) {
	rows, err := l.db.Query(ctx,
		`SELECT id, machine_id, name, type, position, config::text
		 FROM views WHERE machine_id = $1 ORDER BY position`,
		machineID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*model.View
	for rows.Next() {
		v := &model.View{}
		var typeStr, configJSON string
		if err := rows.Scan(&v.ID, &v.MachineID, &v.Name, &typeStr, &v.Position, &configJSON); err != nil {
			return nil, err
		}
		v.Type = model.ViewType(typeStr)
		if err := json.Unmarshal([]byte(configJSON), &v.Config); err != nil {
			return nil, fmt.Errorf("parse config for view %s: %w", v.ID, err)
		}
		out = append(out, v)
	}
	return out, rows.Err()
}
