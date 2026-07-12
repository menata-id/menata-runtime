package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Record struct {
	ID        string
	MachineID string
	Data      map[string]any
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time // CAP-R03 -- non-nil means archived (soft-deleted)
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

// SumField (CAP-A14) sums aggregateField (parsed as numeric) across every
// record on machineID whose own scopeField equals scopeValue -- "this
// Member's total Points across their own ledger entries," the cross-record
// computation an event's AggregateCondition gates on. Non-numeric/missing
// values coerce to 0 via Postgres's own numeric cast within the query
// (rows that don't parse are simply excluded, not an error -- the same
// "don't let one bad row break the whole aggregate" posture COALESCE
// already gives the empty-set case).
func (s *RecordStore) SumField(ctx context.Context, machineID, aggregateField, scopeField, scopeValue string) (float64, error) {
	var sum float64
	err := s.db(ctx).QueryRow(ctx,
		`SELECT COALESCE(SUM((data->>$1)::numeric), 0) FROM records
		 WHERE machine_id = $2 AND data->>$3 = $4 AND data->>$1 ~ '^-?[0-9]+(\.[0-9]+)?$'`,
		aggregateField, machineID, scopeField, scopeValue).Scan(&sum)
	if err != nil {
		return 0, fmt.Errorf("sum field: %w", err)
	}
	return sum, nil
}

// SortOrderField (CAP-V14) is the sentinel sortField value meaning "the
// free-standing sort_order column (migrations/011), not a JSONB data
// field" -- passed by handler.List when a View declares manual_order.
const SortOrderField = "$sort_order"

// List returns machineID's records, newest first by default. sortField
// (CAP-V04, a list View's declared default_sort.field -- validated at load
// time to name a real Field, metadata/loader.go) overrides that with
// `data->>sortField`, sortDirection "desc" (default) or "asc". sortField ==
// SortOrderField (CAP-V14) sorts by the plain sort_order column instead of
// into JSONB. Pass "" for both to keep the original created_at DESC
// behavior -- most callers (child lists, sibling lookups) want that, not a
// View's own sort.
func (s *RecordStore) List(ctx context.Context, machineID, sortField, sortDirection string) ([]*Record, error) {
	orderBy := "created_at DESC"
	args := []any{machineID}
	switch {
	case sortField == SortOrderField:
		dir := "ASC"
		if sortDirection == "desc" {
			dir = "DESC"
		}
		orderBy = "sort_order " + dir
	case sortField == "created_at" || sortField == "updated_at":
		// Reserved names (CAP-V04): a real column, not a JSONB data field --
		// metadata/loader.go's validateReferences exempts these from the
		// "must name a real Field" check for exactly this reason.
		dir := "DESC"
		if sortDirection == "asc" {
			dir = "ASC"
		}
		orderBy = sortField + " " + dir
	case sortField != "":
		dir := "DESC"
		if sortDirection == "asc" {
			dir = "ASC"
		}
		orderBy = fmt.Sprintf(`data->>$2 %s NULLS LAST, created_at DESC`, dir)
		args = append(args, sortField)
	}
	// CAP-R03: archived records are excluded everywhere List is used --
	// child lists, sibling lookups, reference pickers, and the main list
	// view all agree an archived record isn't a live candidate for any of
	// those. ListArchived below is the one explicit exception (viewing the
	// archive itself).
	rows, err := s.db(ctx).Query(ctx,
		fmt.Sprintf(`SELECT id, machine_id, data, created_at, updated_at
		 FROM records WHERE machine_id = $1 AND deleted_at IS NULL ORDER BY %s`, orderBy),
		args...)
	if err != nil {
		return nil, fmt.Errorf("list records: %w", err)
	}
	defer rows.Close()
	return scanRecords(rows)
}

// ListArchived (CAP-R03) is List's mirror image -- only archived records,
// newest-archived first, for the one View that needs to show the archive
// itself so a soft-deleted record can be found and restored.
func (s *RecordStore) ListArchived(ctx context.Context, machineID string) ([]*Record, error) {
	rows, err := s.db(ctx).Query(ctx,
		`SELECT id, machine_id, data, created_at, updated_at
		 FROM records WHERE machine_id = $1 AND deleted_at IS NOT NULL ORDER BY deleted_at DESC`,
		machineID)
	if err != nil {
		return nil, fmt.Errorf("list archived records: %w", err)
	}
	defer rows.Close()
	return scanRecords(rows)
}

func scanRecords(rows pgx.Rows) ([]*Record, error) {
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
		`SELECT id, machine_id, data, created_at, updated_at, deleted_at FROM records WHERE id = $1`,
		id).Scan(&r.ID, &r.MachineID, &dataJSON, &r.CreatedAt, &r.UpdatedAt, &r.DeletedAt)
	if err != nil {
		return nil, fmt.Errorf("get record %s: %w", id, err)
	}
	if err := json.Unmarshal(dataJSON, &r.Data); err != nil {
		return nil, fmt.Errorf("parse data: %w", err)
	}
	return r, nil
}

// Archive/Restore (CAP-R03): a soft delete, not a hard DELETE -- consistent
// with CAP-R04's append-only record_events audit trail, which would be
// left pointing at nothing by a real delete.
func (s *RecordStore) Archive(ctx context.Context, id string) error {
	_, err := s.db(ctx).Exec(ctx, `UPDATE records SET deleted_at = NOW() WHERE id = $1`, id)
	return err
}

func (s *RecordStore) Restore(ctx context.Context, id string) error {
	_, err := s.db(ctx).Exec(ctx, `UPDATE records SET deleted_at = NULL WHERE id = $1`, id)
	return err
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

// ExistsWithFieldValues (CAP-C12) reports whether another record on
// machineID already has the exact same value for every field named in
// fieldValues -- single-field uniqueness when fieldValues has one entry,
// composite/multi-field when it has several (all must match together,
// e.g. "one request per employee per period" needs both fields to collide,
// not either alone). excludeRecordID is the record being updated (empty
// string on Create, where nothing to exclude exists yet) -- a record never
// collides with its own unchanged value.
func (s *RecordStore) ExistsWithFieldValues(ctx context.Context, machineID string, fieldValues map[string]string, excludeRecordID string) (bool, error) {
	sql := `SELECT EXISTS(SELECT 1 FROM records WHERE machine_id = $1`
	args := []any{machineID}
	for field, value := range fieldValues {
		args = append(args, field, value)
		sql += fmt.Sprintf(` AND data->>$%d = $%d`, len(args)-1, len(args))
	}
	if excludeRecordID != "" {
		args = append(args, excludeRecordID)
		sql += fmt.Sprintf(` AND id != $%d`, len(args))
	}
	sql += `)`

	var exists bool
	if err := s.db(ctx).QueryRow(ctx, sql, args...).Scan(&exists); err != nil {
		return false, fmt.Errorf("check uniqueness: %w", err)
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

// NextSequence (CAP-F18) atomically returns the next value of a
// per-(machineID, fieldID) counter, starting at 1 -- a single INSERT ...
// ON CONFLICT DO UPDATE ... RETURNING, never a SELECT-then-UPDATE (which
// would let two concurrent Creates read the same "current" value and both
// generate the same document number).
func (s *RecordStore) NextSequence(ctx context.Context, machineID, fieldID string) (int64, error) {
	var next int64
	err := s.db(ctx).QueryRow(ctx,
		`INSERT INTO field_sequences (machine_id, field_id, next_value) VALUES ($1, $2, 2)
		 ON CONFLICT (machine_id, field_id) DO UPDATE SET next_value = field_sequences.next_value + 1
		 RETURNING next_value - 1`,
		machineID, fieldID).Scan(&next)
	return next, err
}

// ClaimWebhookEvent (CAP-X13) atomically claims (machineID, eventID,
// idempotencyKey) via INSERT ... ON CONFLICT DO NOTHING -- returns true if
// this call made the claim (first delivery, proceed), false if it was
// already claimed (a retry/duplicate delivery, the caller should return
// success without re-running the event). Deliberately a single INSERT, not
// a SELECT-then-INSERT -- a check-then-act pattern races two
// near-simultaneous retries against each other; ON CONFLICT DO NOTHING is
// atomic at the database level regardless of how many callers race it.
func (s *RecordStore) ClaimWebhookEvent(ctx context.Context, machineID, eventID, idempotencyKey string) (bool, error) {
	tag, err := s.db(ctx).Exec(ctx,
		`INSERT INTO webhook_claims (machine_id, event_id, idempotency_key) VALUES ($1, $2, $3) ON CONFLICT DO NOTHING`,
		machineID, eventID, idempotencyKey)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

// FiredToday (CAP-E02/E03) reports whether eventID already fired on
// recordID today (server's local date) -- the scheduler's own
// de-duplication, reusing CAP-R04's existing append-only audit trail
// instead of a new tracking table. A tick that runs more than once a day,
// or a date-driven event whose trigger window spans several ticks, must
// not fire the same record twice.
func (s *RecordStore) FiredToday(ctx context.Context, recordID, eventID string) (bool, error) {
	var exists bool
	err := s.db(ctx).QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM record_events WHERE record_id = $1 AND event_id = $2 AND performed_at::date = CURRENT_DATE)`,
		recordID, eventID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check fired today: %w", err)
	}
	return exists, nil
}

// Move (CAP-V14) swaps recordID's sort_order with its immediate neighbor in
// the given direction ("up" = swap with the previous row, "down" = the
// next), a plain value swap rather than a renumbering pass -- sort_order is
// a DOUBLE PRECISION exactly so two records can always trade places without
// touching any other row. A no-op (not an error) at either edge -- moving
// the first row up, or the last row down, has nothing to swap with.
func (s *RecordStore) Move(ctx context.Context, machineID, recordID, direction string) error {
	var curOrder float64
	if err := s.db(ctx).QueryRow(ctx,
		`SELECT sort_order FROM records WHERE id = $1 AND machine_id = $2`,
		recordID, machineID).Scan(&curOrder); err != nil {
		return fmt.Errorf("move record: %w", err)
	}

	cmp, ord := "<", "DESC"
	if direction == "down" {
		cmp, ord = ">", "ASC"
	}
	var neighborID string
	var neighborOrder float64
	query := fmt.Sprintf(
		`SELECT id, sort_order FROM records WHERE machine_id = $1 AND sort_order %s $2 ORDER BY sort_order %s LIMIT 1`,
		cmp, ord)
	err := s.db(ctx).QueryRow(ctx, query, machineID, curOrder).Scan(&neighborID, &neighborOrder)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("move record: find neighbor: %w", err)
	}

	if _, err := s.db(ctx).Exec(ctx, `UPDATE records SET sort_order = $1 WHERE id = $2`, neighborOrder, recordID); err != nil {
		return fmt.Errorf("move record: %w", err)
	}
	if _, err := s.db(ctx).Exec(ctx, `UPDATE records SET sort_order = $1 WHERE id = $2`, curOrder, neighborID); err != nil {
		return fmt.Errorf("move record: %w", err)
	}
	return nil
}

// GroupSum is one row of a CAP-V13 report: a group label (a report View's
// group_field value) and the SUM of each declared sum_field across every
// record in that group.
type GroupSum struct {
	Group string
	Sums  map[string]float64
}

// SumFieldsGroupedBy (CAP-V13) computes a report View's aggregate: groups
// every record on machineID by groupField's own value, summing each of
// sumFields (numeric fields; non-numeric/missing values coerce to 0, same
// COALESCE posture as SumField). Computed at render time from existing
// records -- nothing about a report is itself stored.
func (s *RecordStore) SumFieldsGroupedBy(ctx context.Context, machineID, groupField string, sumFields []string) ([]GroupSum, error) {
	selects := []string{"data->>$2 AS grp"}
	args := []any{machineID, groupField}
	for _, f := range sumFields {
		args = append(args, f)
		selects = append(selects, fmt.Sprintf(
			`COALESCE(SUM((data->>$%d)::numeric) FILTER (WHERE data->>$%d ~ '^-?[0-9]+(\.[0-9]+)?$'), 0)`,
			len(args), len(args)))
	}
	query := fmt.Sprintf(`SELECT %s FROM records WHERE machine_id = $1 GROUP BY grp ORDER BY grp`, strings.Join(selects, ", "))

	rows, err := s.db(ctx).Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("sum fields grouped by: %w", err)
	}
	defer rows.Close()

	var out []GroupSum
	for rows.Next() {
		scanArgs := make([]any, 0, len(sumFields)+1)
		var grp *string
		scanArgs = append(scanArgs, &grp)
		sums := make([]float64, len(sumFields))
		for i := range sumFields {
			scanArgs = append(scanArgs, &sums[i])
		}
		if err := rows.Scan(scanArgs...); err != nil {
			return nil, err
		}
		gs := GroupSum{Sums: make(map[string]float64, len(sumFields))}
		if grp != nil {
			gs.Group = *grp
		}
		for i, f := range sumFields {
			gs.Sums[f] = sums[i]
		}
		out = append(out, gs)
	}
	return out, rows.Err()
}

// GroupCount is one row of a CAP-V10 dashboard section: a group label (a
// section's group_field value, or "" when the section has none) and how
// many records on that Machine have it.
type GroupCount struct {
	Group string
	Count int
}

// CountGroupedBy (CAP-V10) counts machineID's records, grouped by
// groupField's value ("" groupField counts everything as one group -- a
// dashboard section with no breakdown, just a total).
func (s *RecordStore) CountGroupedBy(ctx context.Context, machineID, groupField string) ([]GroupCount, error) {
	var query string
	args := []any{machineID}
	if groupField == "" {
		query = `SELECT '' AS grp, COUNT(*) FROM records WHERE machine_id = $1 GROUP BY grp`
	} else {
		query = `SELECT COALESCE(data->>$2, '') AS grp, COUNT(*) FROM records WHERE machine_id = $1 GROUP BY grp ORDER BY grp`
		args = append(args, groupField)
	}
	rows, err := s.db(ctx).Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("count grouped by: %w", err)
	}
	defer rows.Close()

	var out []GroupCount
	for rows.Next() {
		var gc GroupCount
		if err := rows.Scan(&gc.Group, &gc.Count); err != nil {
			return nil, err
		}
		out = append(out, gc)
	}
	return out, rows.Err()
}
