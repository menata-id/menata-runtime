package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Record struct {
	ID        string
	MachineID string
	Data      map[string]any
	CreatedAt time.Time
	UpdatedAt time.Time
}

type RecordStore struct {
	db *pgxpool.Pool
}

func NewRecordStore(db *pgxpool.Pool) *RecordStore {
	return &RecordStore{db: db}
}

func (s *RecordStore) List(ctx context.Context, machineID string) ([]*Record, error) {
	rows, err := s.db.Query(ctx,
		`SELECT id, machine_id, data, created_at, updated_at
		 FROM records WHERE machine_id = $1 ORDER BY created_at DESC`,
		machineID)
	if err != nil {
		return nil, fmt.Errorf("list records: %w", err)
	}
	defer rows.Close()

	var out []*Record
	for rows.Next() {
		r := &Record{}
		var dataJSON []byte
		if err := rows.Scan(&r.ID, &r.MachineID, &dataJSON, &r.CreatedAt, &r.UpdatedAt); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(dataJSON, &r.Data); err != nil {
			return nil, fmt.Errorf("parse data for record: %w", err)
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *RecordStore) Get(ctx context.Context, id string) (*Record, error) {
	r := &Record{}
	var dataJSON []byte
	err := s.db.QueryRow(ctx,
		`SELECT id, machine_id, data, created_at, updated_at FROM records WHERE id = $1`,
		id).Scan(&r.ID, &r.MachineID, &dataJSON, &r.CreatedAt, &r.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("get record %s: %w", id, err)
	}
	if err := json.Unmarshal(dataJSON, &r.Data); err != nil {
		return nil, fmt.Errorf("parse data: %w", err)
	}
	return r, nil
}

func (s *RecordStore) Create(ctx context.Context, machineID string, data map[string]any) (*Record, error) {
	dataJSON, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("marshal data: %w", err)
	}
	r := &Record{MachineID: machineID, Data: data}
	err = s.db.QueryRow(ctx,
		`INSERT INTO records (machine_id, data) VALUES ($1, $2) RETURNING id, created_at, updated_at`,
		machineID, string(dataJSON)).Scan(&r.ID, &r.CreatedAt, &r.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("create record: %w", err)
	}
	return r, nil
}

// Exists reports whether a record with the given id exists on the given
// machine. Used to enforce referential integrity for `reference` fields
// (CAP-F13) — a value that doesn't resolve to a real record is rejected the
// same way a required-field violation is, not silently accepted. A recordID
// that isn't even well-formed UUID syntax (a hand-typed or tampered form
// value, not one from the picker) is itself just "doesn't exist" from the
// caller's point of view — Postgres's 22P02 (invalid_text_representation) on
// the UUID column comparison is caught and folded into that same false/nil
// result rather than surfacing as a 500.
func (s *RecordStore) Exists(ctx context.Context, machineID, recordID string) (bool, error) {
	var exists bool
	err := s.db.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM records WHERE id = $1 AND machine_id = $2)`,
		recordID, machineID).Scan(&exists)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "22P02" {
			return false, nil
		}
		return false, fmt.Errorf("check record exists: %w", err)
	}
	return exists, nil
}

func (s *RecordStore) Update(ctx context.Context, id string, data map[string]any) error {
	dataJSON, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("marshal data: %w", err)
	}
	_, err = s.db.Exec(ctx,
		`UPDATE records SET data = $1, updated_at = NOW() WHERE id = $2`,
		string(dataJSON), id)
	return err
}

// LogEvent appends one row to the append-only record_events audit trail
// (CAP-R04, enforced append-only at the DB level, migrations/007 REVOKEs
// UPDATE/DELETE/TRUNCATE from this role). performedBy is the acting
// identity (falling back to role, same convention as CAP-A02's
// current_user), correlationID is the request-scoped id (CAP-I04, chi's
// middleware.RequestID) shared by every row one HTTP request produces, even
// across a cascade (CAP-A08/CAP-E05 firing an event on another record).
// Either may be empty (e.g. a request with no request-id middleware in a
// test harness) -- stored as SQL NULL, not an empty string.
func (s *RecordStore) LogEvent(ctx context.Context, recordID, eventID, performedBy, correlationID string, snapshot map[string]any) error {
	snapshotJSON, err := json.Marshal(snapshot)
	if err != nil {
		return fmt.Errorf("marshal snapshot: %w", err)
	}
	_, err = s.db.Exec(ctx,
		`INSERT INTO record_events (record_id, event_id, performed_by, correlation_id, snapshot) VALUES ($1, $2, NULLIF($3, ''), NULLIF($4, ''), $5)`,
		recordID, eventID, performedBy, correlationID, string(snapshotJSON))
	return err
}
