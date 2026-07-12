package store

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Notification is one in-app delivery (CAP-A10) of a `notify` action
// (CAP-A03/A04). Recipient is a plain string -- a role name for a static
// `notify: {role: ...}`, or a resolved field value for a dynamic
// `notify: {recipient_field: ...}` -- matched against a session's role
// cookie, the same identity-is-role caveat CAP-A02's current_user carries.
// CAP-X06 (2026-07-12): recipient alone was workspace-blind -- a "Manager"
// in one workspace could see a "Manager" in another's notifications; RLS on
// workspace_id closes this as a byproduct, not a separate fix.
type Notification struct {
	ID        string
	Recipient string
	Message   string
	MachineID string
	RecordID  string
	CreatedAt time.Time
	ReadAt    *time.Time
}

type NotificationStore struct {
	pool *pgxpool.Pool
}

func NewNotificationStore(pool *pgxpool.Pool) *NotificationStore {
	return &NotificationStore{pool: pool}
}

// db returns the request-scoped transaction when one is attached to ctx --
// see RecordStore.db's doc comment, same convention.
func (s *NotificationStore) db(ctx context.Context) querier {
	return dbFromContext(ctx, s.pool)
}

// Create inserts a Notification. workspaceID (CAP-X06) is resolved by the
// caller the same way RecordStore.Create's is.
func (s *NotificationStore) Create(ctx context.Context, recipient, message, machineID, recordID, workspaceID string) error {
	_, err := s.db(ctx).Exec(ctx,
		`INSERT INTO notifications (recipient, message, machine_id, record_id, workspace_id) VALUES ($1, $2, $3, NULLIF($4, '')::uuid, $5)`,
		recipient, message, machineID, recordID, workspaceID)
	if err != nil {
		return fmt.Errorf("create notification: %w", err)
	}
	return nil
}

func (s *NotificationStore) ListForRecipient(ctx context.Context, recipient string) ([]*Notification, error) {
	rows, err := s.db(ctx).Query(ctx,
		`SELECT id, recipient, message, COALESCE(machine_id, ''), COALESCE(record_id::text, ''), created_at, read_at
		 FROM notifications WHERE recipient = $1 ORDER BY created_at DESC LIMIT 50`,
		recipient)
	if err != nil {
		return nil, fmt.Errorf("list notifications: %w", err)
	}
	defer rows.Close()

	var out []*Notification
	for rows.Next() {
		n := &Notification{}
		if err := rows.Scan(&n.ID, &n.Recipient, &n.Message, &n.MachineID, &n.RecordID, &n.CreatedAt, &n.ReadAt); err != nil {
			return nil, err
		}
		out = append(out, n)
	}
	return out, rows.Err()
}

func (s *NotificationStore) UnreadCount(ctx context.Context, recipient string) (int, error) {
	var count int
	err := s.db(ctx).QueryRow(ctx,
		`SELECT COUNT(*) FROM notifications WHERE recipient = $1 AND read_at IS NULL`,
		recipient).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count unread notifications: %w", err)
	}
	return count, nil
}

// MarkRead marks a notification read only if it belongs to recipient --
// this prototype's only access control is "the role cookie matches the
// recipient string", so this check is what keeps one role from marking
// another role's notifications read.
func (s *NotificationStore) MarkRead(ctx context.Context, id, recipient string) error {
	_, err := s.db(ctx).Exec(ctx,
		`UPDATE notifications SET read_at = NOW() WHERE id = $1 AND recipient = $2 AND read_at IS NULL`,
		id, recipient)
	if err != nil {
		return fmt.Errorf("mark notification read: %w", err)
	}
	return nil
}
