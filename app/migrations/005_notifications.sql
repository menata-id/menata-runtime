-- +goose Up
-- 005_notifications.sql
-- CAP-A10: in-app notification delivery channel.
--
-- `notify` actions (CAP-A03/A04) previously only wrote to slog -- nothing a
-- user could ever see. This gives them a real destination: a row a
-- recipient's session can list and mark read. `recipient` is a plain string
-- (a role name for a static `notify: {role: ...}`, or a resolved field value
-- for a dynamic `notify: {recipient_field: ...}`) matched against the
-- session's role cookie -- this prototype has no per-user identity separate
-- from role, the same caveat CAP-A02's `current_user` already carries.
--
-- Email is deliberately not implemented here -- no mail infrastructure exists
-- in this prototype's environment, and simulating delivery would misrepresent
-- the capability as done when only one of its two channels is real.

CREATE TABLE IF NOT EXISTS notifications (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    recipient    TEXT NOT NULL,
    message      TEXT NOT NULL,
    machine_id   TEXT,
    record_id    UUID,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    read_at      TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_notifications_recipient ON notifications (recipient, created_at DESC);

-- +goose Down
DROP TABLE IF EXISTS notifications;
