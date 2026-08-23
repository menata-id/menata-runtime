package store

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Notification is one in-app delivery (CAP-A10) of a `notify` action
// (CAP-A03/A04). Recipient is a plain string -- a role name for a static
// `notify: {role: ...}`, or a resolved field value (usually an identity
// name) for a dynamic `notify: {recipient_field: ...}`. CAP-O01
// (2026-07-12): a role recipient is no longer matched against one
// session-wide role -- it's matched against the reader's own
// user_application_roles row for that notification's specific Application
// (see ListForRecipient's query), since the same person can hold a
// different role in each Application they have access to.
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

// recipientMatch is the WHERE clause shared by ListForRecipient/UnreadCount/
// MarkRead: a notification belongs to a reader if its recipient is their own
// identity name, OR their own user id (CAP-F05 -- a `notify:
// {recipient_field: ...}` targeting a `user`-typed Field stores that
// person's id as the resolved recipient, not their name, since Executor has
// no access to UserStore to resolve id -> name itself; matching both here
// keeps notifications.recipient's two possible identity shapes both
// reachable without touching Executor), OR its recipient is a role string
// that matches one of their user_application_roles rows *for that
// notification's own Application* (CAP-O01 -- a role broadcast to "Manager"
// must only reach people who are actually "Manager" in the Application that
// fired it, not everyone who happens to hold "Manager" anywhere).
// $1 = identity name, $2 = userID. The explicit ::text/::uuid casts on $2
// are load-bearing, not decoration: Postgres infers one type per parameter
// per query, and $2 is compared against both a `text` column (n.recipient)
// and a `uuid` column (uar.user_id) here -- without the casts this fails at
// query time with "operator does not exist: uuid = text" (SQLSTATE 42883),
// caught during CAP-F05's isolated-port verification, not left as a
// silent landmine for the next person who removes an "unnecessary" cast.
// CAP-O07 (2026-08-23) added the fourth OR clause: a role broadcast also
// reaches someone who holds that role only through Group membership, not
// just a direct user_application_roles row -- same "for that notification's
// own Application" scoping the direct clause already enforces, joined
// through group_members -> group_application_roles instead.
const recipientMatch = `(
	n.recipient = $1
	OR n.recipient = $2::text
	OR EXISTS (
		SELECT 1 FROM machines m
		JOIN user_application_roles uar ON uar.application_id = m.application_id
		WHERE m.id = n.machine_id AND uar.role = n.recipient AND uar.user_id = $2::uuid
	)
	OR EXISTS (
		SELECT 1 FROM machines m
		JOIN group_application_roles gar ON gar.application_id = m.application_id
		JOIN group_members gm ON gm.group_id = gar.group_id
		WHERE m.id = n.machine_id AND gar.role = n.recipient AND gm.user_id = $2::uuid
	)
)`

func (s *NotificationStore) ListForRecipient(ctx context.Context, identity, userID string) ([]*Notification, error) {
	rows, err := s.db(ctx).Query(ctx,
		`SELECT n.id, n.recipient, n.message, COALESCE(n.machine_id, ''), COALESCE(n.record_id::text, ''), n.created_at, n.read_at
		 FROM notifications n
		 WHERE `+recipientMatch+`
		 ORDER BY n.created_at DESC LIMIT 50`,
		identity, userID)
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

func (s *NotificationStore) UnreadCount(ctx context.Context, identity, userID string) (int, error) {
	var count int
	err := s.db(ctx).QueryRow(ctx,
		`SELECT COUNT(*) FROM notifications n WHERE `+recipientMatch+` AND n.read_at IS NULL`,
		identity, userID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count unread notifications: %w", err)
	}
	return count, nil
}

// MarkRead marks a notification read only if it belongs to the reader
// (recipientMatch) -- what keeps one person from marking another's
// notifications read.
func (s *NotificationStore) MarkRead(ctx context.Context, id, identity, userID string) error {
	_, err := s.db(ctx).Exec(ctx,
		`UPDATE notifications n SET read_at = NOW() WHERE n.id = $3 AND `+recipientMatch+` AND n.read_at IS NULL`,
		identity, userID, id)
	if err != nil {
		return fmt.Errorf("mark notification read: %w", err)
	}
	return nil
}
