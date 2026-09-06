package store

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// OutboxItem is one claimed action_outbox row (CAP-W06) -- an
// already-resolved action a dispatcher performs off the request path.
// Params is decoded per ActionType by the caller (runOutboxDispatcher),
// the same way Executor.Persist's own action loop switches on
// model.EventAction.Type.
type OutboxItem struct {
	ID            string
	ActionType    string
	Params        map[string]any
	CorrelationID string
}

type OutboxStore struct {
	pool *pgxpool.Pool
}

func NewOutboxStore(pool *pgxpool.Pool) *OutboxStore {
	return &OutboxStore{pool: pool}
}

// db returns the request-scoped transaction when one is attached to ctx --
// see RecordStore.db's doc comment, same convention.
func (s *OutboxStore) db(ctx context.Context) querier {
	return dbFromContext(ctx, s.pool)
}

// Enqueue writes one action_outbox row against whatever dbFromContext
// returns -- when called from Executor.Persist/Handler.processSubscriptions,
// that's the triggering request's own still-open transaction, which is what
// makes this row atomic with the business write for free (the same
// mechanism CAP-X12 already relies on for cross-record atomicity): if the
// request's own transaction rolls back for any reason, this row was never
// durably committed either.
func (s *OutboxStore) Enqueue(ctx context.Context, workspaceID, actionType string, params map[string]any, correlationID string) error {
	raw, err := json.Marshal(params)
	if err != nil {
		return fmt.Errorf("marshal outbox params: %w", err)
	}
	_, err = s.db(ctx).Exec(ctx,
		`INSERT INTO action_outbox (workspace_id, action_type, params, correlation_id) VALUES ($1, $2, $3, NULLIF($4, ''))`,
		workspaceID, actionType, raw, correlationID)
	if err != nil {
		return fmt.Errorf("enqueue outbox: %w", err)
	}
	return nil
}

// ClaimBatch atomically claims up to limit unclaimed rows for the workspace
// already set on ctx's transaction (RLS -- see runOutboxDispatcher, which
// opens one transaction per workspace per tick, mirroring runScheduler's own
// shape). FOR UPDATE SKIP LOCKED is the standard atomic job-queue claim --
// safe even if more than one dispatcher instance ever runs concurrently,
// the same "never SELECT-then-act" discipline CAP-X13's ClaimWebhookEvent
// established for a different shape of claim (deduping an at-least-once
// external delivery, vs. claiming N already-existing rows for one worker
// here).
func (s *OutboxStore) ClaimBatch(ctx context.Context, limit int) ([]*OutboxItem, error) {
	rows, err := s.db(ctx).Query(ctx,
		`UPDATE action_outbox
		 SET claimed_at = NOW()
		 WHERE id IN (
		     SELECT id FROM action_outbox
		     WHERE claimed_at IS NULL
		     ORDER BY created_at
		     LIMIT $1
		     FOR UPDATE SKIP LOCKED
		 )
		 RETURNING id, action_type, params, COALESCE(correlation_id, '')`,
		limit)
	if err != nil {
		return nil, fmt.Errorf("claim outbox batch: %w", err)
	}
	defer rows.Close()

	var out []*OutboxItem
	for rows.Next() {
		item := &OutboxItem{}
		var raw []byte
		if err := rows.Scan(&item.ID, &item.ActionType, &raw, &item.CorrelationID); err != nil {
			return nil, fmt.Errorf("scan outbox item: %w", err)
		}
		if err := json.Unmarshal(raw, &item.Params); err != nil {
			return nil, fmt.Errorf("unmarshal outbox params (%s): %w", item.ID, err)
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

// MarkCompleted records a successful dispatch.
func (s *OutboxStore) MarkCompleted(ctx context.Context, id string) error {
	_, err := s.db(ctx).Exec(ctx, `UPDATE action_outbox SET completed_at = NOW() WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("mark outbox completed: %w", err)
	}
	return nil
}

// MarkFailed records a failed dispatch -- the row stays claimed (not
// requeued, no automatic retry: no case forces that yet, see
// capability-registry.md's CAP-W06 row for the named, deferred limitation),
// but the failure is now durable and inspectable, not just a log line.
func (s *OutboxStore) MarkFailed(ctx context.Context, id, errMsg string) error {
	_, err := s.db(ctx).Exec(ctx, `UPDATE action_outbox SET failed_at = NOW(), error = $2 WHERE id = $1`, id, errMsg)
	if err != nil {
		return fmt.Errorf("mark outbox failed: %w", err)
	}
	return nil
}
