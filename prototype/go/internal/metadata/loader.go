// Package metadata handles loading Runtime Metadata from PostgreSQL
// and building the Application Model used by the Interpreter.
package metadata

import (
	"context"
	"encoding/json"
	"fmt"

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
	return workspaces, nil
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
		`SELECT id, application_id, name FROM machines WHERE application_id = $1 ORDER BY name`,
		applicationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var machines []*model.Machine
	for rows.Next() {
		m := &model.Machine{}
		if err := rows.Scan(&m.ID, &m.ApplicationID, &m.Name); err != nil {
			return nil, err
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
		`SELECT id, machine_id, name, position FROM events WHERE machine_id = $1 ORDER BY position`,
		machineID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []*model.Event
	for rows.Next() {
		e := &model.Event{}
		if err := rows.Scan(&e.ID, &e.MachineID, &e.Name, &e.Position); err != nil {
			return nil, err
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
		`SELECT id, machine_id, role, events FROM permissions WHERE machine_id = $1`,
		machineID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*model.Permission
	for rows.Next() {
		p := &model.Permission{}
		if err := rows.Scan(&p.ID, &p.MachineID, &p.Role, &p.Events); err != nil {
			return nil, err
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
