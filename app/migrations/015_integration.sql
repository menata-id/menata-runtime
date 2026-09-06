-- +goose Up
-- 015_integration.sql
-- Batch 8 (Cross-Machine Integration): CAP-I01 (cross-machine event
-- subscription), CAP-I02 (event schema declaration), CAP-I03 (integration
-- contract), CAP-I05 (cross-cutting contribution). CAP-I04 (correlation
-- trace) shipped earlier; its SLO-registry half stays a named, deferred
-- gap, unaffected by this batch.
--
-- CAP-I02: purely declarative metadata on an Event -- category/version for
-- documentation purposes, deprecated_message logged (not blocked) when a
-- deprecated Event still fires, backward-compat by design.
ALTER TABLE events ADD COLUMN IF NOT EXISTS category TEXT;
ALTER TABLE events ADD COLUMN IF NOT EXISTS schema_version TEXT;
ALTER TABLE events ADD COLUMN IF NOT EXISTS deprecated_message TEXT;

-- CAP-I01: a SUBSCRIBER Machine declares interest in a PUBLISHER Event
-- elsewhere -- the publisher's own metadata never names its subscribers
-- (Pattern C's whole point: the publisher doesn't know who's listening).
-- fields is CAP-A06's own create_record mapping shape ("field:<id>" reads
-- the publisher's post-event data, a literal is a literal, dynamic tokens
-- like current_user/today resolve the same way) -- one new record on
-- machine_id per firing. contract/on_violation are CAP-I03: contract is a
-- JSON array of ConstraintExpression-shaped checks against the publisher's
-- data, checked before firing; on_violation ("skip", the default, or
-- "log_only") decides whether a failed check cancels this one
-- subscription's own action or just gets logged and fires anyway. CAP-I05
-- (contribution) needs no new column at all -- it's proven by two or more
-- of these rows, from DIFFERENT publisher Events, targeting the SAME
-- machine_id (a shared KPI/gamification Machine), decoupled from every
-- contributing Event's own definition -- the same mechanism, not a
-- separate one.
CREATE TABLE IF NOT EXISTS event_subscriptions (
    id                 TEXT PRIMARY KEY,
    machine_id         TEXT NOT NULL REFERENCES machines(id) ON DELETE CASCADE,
    publisher_event_id TEXT NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    fields             JSONB NOT NULL DEFAULT '{}',
    contract           JSONB,
    on_violation       TEXT NOT NULL DEFAULT 'skip',
    position           INT NOT NULL DEFAULT 0,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_event_subscriptions_publisher ON event_subscriptions(publisher_event_id);

-- +goose Down
DROP TABLE IF EXISTS event_subscriptions;
ALTER TABLE events DROP COLUMN IF EXISTS deprecated_message;
ALTER TABLE events DROP COLUMN IF EXISTS schema_version;
ALTER TABLE events DROP COLUMN IF EXISTS category;
