package store

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

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

func (s *RecordStore) LogEvent(ctx context.Context, recordID, eventID string, snapshot map[string]any) error {
	snapshotJSON, err := json.Marshal(snapshot)
	if err != nil {
		return fmt.Errorf("marshal snapshot: %w", err)
	}
	_, err = s.db.Exec(ctx,
		`INSERT INTO record_events (record_id, event_id, snapshot) VALUES ($1, $2, $3)`,
		recordID, eventID, string(snapshotJSON))
	return err
}
