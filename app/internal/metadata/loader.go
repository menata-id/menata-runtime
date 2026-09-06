package metadata

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5"
	"menata.id/app/internal/model"
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

func (l *Loader) loadWorkspaces(ctx context.Context) ([]*model.Workspace, error) {
	rows, err := l.db.Query(ctx, `SELECT id, name, slug FROM workspaces ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*model.Workspace
	for rows.Next() {
		ws := &model.Workspace{}
		if err := rows.Scan(&ws.ID, &ws.Name, &ws.Slug); err != nil {
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
