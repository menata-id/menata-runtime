package store

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
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
	pool *pgxpool.Pool
}

func NewRecordStore(pool *pgxpool.Pool) *RecordStore {
	return &RecordStore{pool: pool}
}

// db returns the request-scoped transaction (CAP-X06 -- set by
// cmd/server/main.go's middleware via store.WithTx, carrying the
// SET LOCAL app.workspace_id that makes RLS apply) when one is attached to
// ctx, falling back to the raw pool for trusted non-request callers
// (metadata.Loader's boot-time LoadAll).
func (s *RecordStore) db(ctx context.Context) querier {
	return dbFromContext(ctx, s.pool)
}

func (s *RecordStore) List(ctx context.Context, machineID string) ([]*Record, error) {
	rows, err := s.db(ctx).Query(ctx,
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
	err := s.db(ctx).QueryRow(ctx,
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

// Create inserts a new record. workspaceID (CAP-X06, resolved by the caller
// via Interpreter.ScopeFor -- the same resolution already used for
// CAP-I04's logging) sets the row's workspace_id directly; RLS only governs
// reads/updates/deletes against the current app.workspace_id, a write still
// has to supply the right value itself.
func (s *RecordStore) Create(ctx context.Context, machineID, workspaceID string, data map[string]any) (*Record, error) {
	dataJSON, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("marshal data: %w", err)
	}
	r := &Record{MachineID: machineID, Data: data}
	err = s.db(ctx).QueryRow(ctx,
		`INSERT INTO records (machine_id, workspace_id, data) VALUES ($1, $2, $3) RETURNING id, created_at, updated_at`,
		machineID, workspaceID, string(dataJSON)).Scan(&r.ID, &r.CreatedAt, &r.UpdatedAt)
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
// caller's point of view.
//
// The UUID syntax is validated in Go *before* querying, not by sending it to
// Postgres and catching 22P02 (invalid_text_representation) -- that used to
// work when every query ran in its own implicit transaction, but CAP-X06's
// request-scoped transaction (cmd/server/main.go's workspaceTx, needed for
// RLS's SET LOCAL) means a query erroring at the Postgres level poisons the
// rest of that transaction even if the Go error is caught: every later query
// in the same request then fails with 25P02 ("current transaction is
// aborted"), which surfaced as real, reproducible failures once RLS went
// live (list reference options / unread count both breaking after an
// intentionally-malformed reference value earlier in the same request).
// Never reaching Postgres with the bad value avoids the whole class of
// problem, not just this one instance of it.
func (s *RecordStore) Exists(ctx context.Context, machineID, recordID string) (bool, error) {
	if (&pgtype.UUID{}).Scan(recordID) != nil {
		return false, nil
	}
	var exists bool
	err := s.db(ctx).QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM records WHERE id = $1 AND machine_id = $2)`,
		recordID, machineID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check record exists: %w", err)
	}
	return exists, nil
}

func (s *RecordStore) Update(ctx context.Context, id string, data map[string]any) error {
	dataJSON, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("marshal data: %w", err)
	}
	_, err = s.db(ctx).Exec(ctx,
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
// workspaceID (CAP-X06) is resolved the same way Create's is. Any of
// performedBy/correlationID may be empty -- stored as SQL NULL, not an
// empty string.
func (s *RecordStore) LogEvent(ctx context.Context, recordID, eventID, performedBy, correlationID, workspaceID string, snapshot map[string]any) error {
	snapshotJSON, err := json.Marshal(snapshot)
	if err != nil {
		return fmt.Errorf("marshal snapshot: %w", err)
	}
	_, err = s.db(ctx).Exec(ctx,
		`INSERT INTO record_events (record_id, event_id, performed_by, correlation_id, workspace_id, snapshot) VALUES ($1, $2, NULLIF($3, ''), NULLIF($4, ''), $5, $6)`,
		recordID, eventID, performedBy, correlationID, workspaceID, string(snapshotJSON))
	return err
}
