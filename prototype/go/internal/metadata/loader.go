// Package metadata handles loading Runtime Metadata from PostgreSQL
// and building the Application Model used by the Interpreter.
package metadata

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"menata.id/runtime/internal/model"
)

// Loader reads Runtime Metadata from the database.
type Loader struct {
	db *pgxpool.Pool
}

func NewLoader(db *pgxpool.Pool) *Loader {
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
	}
	if err := validateReferences(workspaces); err != nil {
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
	for _, ws := range workspaces {
		for _, app := range ws.Applications {
			for _, m := range app.Machines {
				known[m.ID] = true
				machineByID[m.ID] = m
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
			}
		}
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
		`SELECT id, application_id, name, config::text FROM machines WHERE application_id = $1 ORDER BY name`,
		applicationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var machines []*model.Machine
	for rows.Next() {
		m := &model.Machine{}
		var configJSON *string
		if err := rows.Scan(&m.ID, &m.ApplicationID, &m.Name, &configJSON); err != nil {
			return nil, err
		}
		if configJSON != nil {
			m.Config = make(map[string]string)
			if err := json.Unmarshal([]byte(*configJSON), &m.Config); err != nil {
				return nil, fmt.Errorf("parse config for machine %s: %w", m.ID, err)
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
	return err
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
		`SELECT id, machine_id, name, position, condition::text, input_fields, schedule::text FROM events WHERE machine_id = $1 ORDER BY position`,
		machineID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []*model.Event
	for rows.Next() {
		e := &model.Event{}
		var condJSON, schedJSON *string
		if err := rows.Scan(&e.ID, &e.MachineID, &e.Name, &e.Position, &condJSON, &e.InputFields, &schedJSON); err != nil {
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
		`SELECT id, machine_id, rule, expression::text, condition::text, position
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
		if err := rows.Scan(&c.ID, &c.MachineID, &c.Rule, &exprJSON, &condJSON, &c.Position); err != nil {
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
