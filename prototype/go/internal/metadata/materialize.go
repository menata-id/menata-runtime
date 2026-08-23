package metadata

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5"
	"menata.id/runtime/internal/model"
)

// MaterializeApplication (CAP-X08 import) writes app's full tree into
// applications/machines/fields/events/event_actions/constraints/
// permissions/views/event_subscriptions, inside the caller's own
// transaction -- the mechanical inverse of this package's own load*
// functions, using the exact same column shapes those already read.
//
// workspaceID is the IMPORTING admin's own current workspace, not
// app.WorkspaceID (which reflects wherever the package was originally
// exported from) -- "one knowledge, many runtimes" means the destination
// is always the operator's own choice, never trusted from the package
// itself.
//
// The caller is responsible for rejecting a package containing any
// Machine.Process or Constraint.ChangePolicy BEFORE calling this (see
// capability-registry.md's CAP-X08 row) -- compileProcess/
// compileChangePolicies mutate a Machine's Events/Permissions/Constraints
// in memory at load time and never persist the generated result, so
// materializing an already-compiled struct verbatim would permanently
// bake generated content into these tables and then generate it AGAIN on
// the next load, corrupting the very data this function writes. This
// function does not re-check that here -- it trusts the caller, the same
// "validated once, at the real boundary" posture the rest of this
// package's load-time checks already take.
//
// No ON CONFLICT anywhere -- a colliding id (re-importing the same
// package, or importing into the workspace it came from) fails loudly
// with Postgres's own unique-violation, the caller's transaction rolls
// back, nothing is silently overwritten.
func MaterializeApplication(ctx context.Context, tx pgx.Tx, workspaceID string, app *model.Application) error {
	if _, err := tx.Exec(ctx, `INSERT INTO applications (id, workspace_id, name) VALUES ($1, $2, $3)`,
		app.ID, workspaceID, app.Name); err != nil {
		return fmt.Errorf("application %s: %w", app.ID, err)
	}

	for _, m := range app.Machines {
		if err := materializeMachine(ctx, tx, m); err != nil {
			return fmt.Errorf("machine %s: %w", m.ID, err)
		}
	}
	return nil
}

func materializeMachine(ctx context.Context, tx pgx.Tx, m *model.Machine) error {
	var configJSON []byte
	if m.Config != nil {
		var err error
		configJSON, err = json.Marshal(m.Config)
		if err != nil {
			return fmt.Errorf("marshal config: %w", err)
		}
	}
	// process is always NULL here -- the caller already rejected any
	// Machine with Process != nil before this ran (see doc comment above).
	if _, err := tx.Exec(ctx, `INSERT INTO machines (id, application_id, name, config, process) VALUES ($1, $2, $3, $4, NULL)`,
		m.ID, m.ApplicationID, m.Name, nullableJSON(configJSON)); err != nil {
		return fmt.Errorf("insert: %w", err)
	}

	for _, f := range m.Fields {
		optionsJSON, err := json.Marshal(f.Options)
		if err != nil {
			return fmt.Errorf("field %s: marshal options: %w", f.ID, err)
		}
		if _, err := tx.Exec(ctx,
			`INSERT INTO fields (id, machine_id, name, type, position, required, options) VALUES ($1, $2, $3, $4, $5, $6, $7)`,
			f.ID, f.MachineID, f.Name, string(f.Type), f.Position, f.Required, optionsJSON); err != nil {
			return fmt.Errorf("field %s: %w", f.ID, err)
		}
	}

	for _, e := range m.Events {
		if err := materializeEvent(ctx, tx, e); err != nil {
			return fmt.Errorf("event %s: %w", e.ID, err)
		}
	}

	for _, c := range m.Constraints {
		if err := materializeConstraint(ctx, tx, c); err != nil {
			return fmt.Errorf("constraint %s: %w", c.ID, err)
		}
	}

	for _, p := range m.Permissions {
		var ownerField *string
		if p.OwnerField != "" {
			ownerField = &p.OwnerField
		}
		if _, err := tx.Exec(ctx,
			`INSERT INTO permissions (id, machine_id, role, events, owner_field, can_read, can_create, can_edit, can_delete, hidden_fields)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
			p.ID, p.MachineID, p.Role, orEmpty(p.Events), ownerField, p.CanRead, p.CanCreate, p.CanEdit, p.CanDelete, orEmpty(p.HiddenFields)); err != nil {
			return fmt.Errorf("permission %s: %w", p.ID, err)
		}
	}

	for _, v := range m.Views {
		configJSON, err := json.Marshal(v.Config)
		if err != nil {
			return fmt.Errorf("view %s: marshal config: %w", v.ID, err)
		}
		if _, err := tx.Exec(ctx,
			`INSERT INTO views (id, machine_id, name, type, position, config) VALUES ($1, $2, $3, $4, $5, $6)`,
			v.ID, v.MachineID, v.Name, string(v.Type), v.Position, configJSON); err != nil {
			return fmt.Errorf("view %s: %w", v.ID, err)
		}
	}

	for i, s := range m.Subscriptions {
		if err := materializeSubscription(ctx, tx, s, i); err != nil {
			return fmt.Errorf("subscription %s: %w", s.ID, err)
		}
	}

	return nil
}

// materializeEvent writes condition (Event.Condition OR
// Event.AggregateCondition -- mutually exclusive, disambiguated by loadEvents
// itself via an "aggregate_field" key probe, so marshaling whichever one is
// actually set reproduces the exact shape that read already expects) and
// schedule, then its own Actions.
func materializeEvent(ctx context.Context, tx pgx.Tx, e *model.Event) error {
	var condJSON []byte
	var err error
	switch {
	case e.AggregateCondition != nil:
		condJSON, err = json.Marshal(e.AggregateCondition)
	case e.Condition != nil:
		condJSON, err = json.Marshal(e.Condition)
	}
	if err != nil {
		return fmt.Errorf("marshal condition: %w", err)
	}
	var schedJSON []byte
	if e.Schedule != nil {
		schedJSON, err = json.Marshal(e.Schedule)
		if err != nil {
			return fmt.Errorf("marshal schedule: %w", err)
		}
	}
	if _, err := tx.Exec(ctx,
		`INSERT INTO events (id, machine_id, name, position, condition, input_fields, schedule, category, schema_version, deprecated_message)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		e.ID, e.MachineID, e.Name, e.Position, nullableJSON(condJSON), orEmpty(e.InputFields), nullableJSON(schedJSON),
		e.Category, e.SchemaVersion, e.DeprecatedMessage); err != nil {
		return fmt.Errorf("insert: %w", err)
	}

	for _, a := range e.Actions {
		paramsJSON, err := json.Marshal(a.Params)
		if err != nil {
			return fmt.Errorf("action: marshal params: %w", err)
		}
		// id is BIGSERIAL -- never insert the exported one, let Postgres
		// assign a fresh id; it has no meaning outside this row.
		if _, err := tx.Exec(ctx,
			`INSERT INTO event_actions (event_id, type, position, params) VALUES ($1, $2, $3, $4)`,
			a.EventID, string(a.Type), a.Position, paramsJSON); err != nil {
			return fmt.Errorf("action: %w", err)
		}
	}
	return nil
}

// materializeConstraint always writes a NULL change_policy -- the caller
// already rejected any Constraint with ChangePolicy != nil (see
// MaterializeApplication's doc comment).
func materializeConstraint(ctx context.Context, tx pgx.Tx, c *model.Constraint) error {
	exprJSON, err := json.Marshal(c.Expression)
	if err != nil {
		return fmt.Errorf("marshal expression: %w", err)
	}
	var condJSON, crossRecordJSON []byte
	if c.Condition != nil {
		if condJSON, err = json.Marshal(c.Condition); err != nil {
			return fmt.Errorf("marshal condition: %w", err)
		}
	}
	if c.CrossRecord != nil {
		if crossRecordJSON, err = json.Marshal(c.CrossRecord); err != nil {
			return fmt.Errorf("marshal cross_record: %w", err)
		}
	}
	if _, err := tx.Exec(ctx,
		`INSERT INTO constraints (id, machine_id, rule, expression, condition, position, change_policy, cross_record)
		 VALUES ($1, $2, $3, $4, $5, $6, NULL, $7)`,
		c.ID, c.MachineID, c.Rule, exprJSON, nullableJSON(condJSON), c.Position, nullableJSON(crossRecordJSON)); err != nil {
		return fmt.Errorf("insert: %w", err)
	}
	return nil
}

func materializeSubscription(ctx context.Context, tx pgx.Tx, s *model.Subscription, position int) error {
	fieldsJSON, err := json.Marshal(s.Fields)
	if err != nil {
		return fmt.Errorf("marshal fields: %w", err)
	}
	var contractJSON []byte
	if s.Contract != nil {
		if contractJSON, err = json.Marshal(s.Contract); err != nil {
			return fmt.Errorf("marshal contract: %w", err)
		}
	}
	if _, err := tx.Exec(ctx,
		`INSERT INTO event_subscriptions (id, machine_id, publisher_event_id, fields, contract, on_violation, position)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		s.ID, s.MachineID, s.PublisherEventID, fieldsJSON, nullableJSON(contractJSON), s.OnViolation, position); err != nil {
		return fmt.Errorf("insert: %w", err)
	}
	return nil
}

// orEmpty guards every TEXT[] NOT NULL DEFAULT '{}' column this function
// writes to (permissions.events/hidden_fields, events.input_fields) -- pgx
// encodes a nil Go slice as SQL NULL, not an empty array, which would
// violate the NOT NULL constraint these columns actually have.
func orEmpty(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

// nullableJSON turns a zero-length marshal result (json.Marshal(nil) is
// "null", not empty -- this only happens here when the source Go value was
// itself nil, e.g. no Config/Condition/Schedule at all) into a real SQL
// NULL for a nullable JSONB column, rather than storing the literal JSON
// string "null".
func nullableJSON(b []byte) any {
	if len(b) == 0 || string(b) == "null" {
		return nil
	}
	return b
}
