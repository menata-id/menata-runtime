-- +goose Up
-- 017_infra_batch10.sql
-- Batch 10 (Infra): CAP-X13 (webhook/external-event idempotency). CAP-X12
-- (cross-record write atomicity) needed no schema change -- it's a bug fix
-- in how executor.Persist's action loop propagates errors, so the ALREADY
-- EXISTING per-request transaction (cmd/server/main.go's workspaceTx,
-- built for CAP-X06's RLS cutover) actually rolls back a partial failure
-- instead of committing it. CAP-X07 (auto-generated REST API) and CAP-X08
-- (metadata export/import) are new routes only, no new tables.

-- CAP-X13: an inbound webhook (CAP-E04) MAY carry an X-Idempotency-Key
-- header naming the external event this delivery represents. Claiming a
-- key is a single atomic INSERT ... ON CONFLICT DO NOTHING (never a
-- check-then-act SELECT followed by INSERT, which races two
-- near-simultaneous retries against each other) -- 0 rows affected means
-- this exact (machine, event, key) was already claimed, so the handler
-- returns success without re-running the event a second time, the same
-- "duplicates are safe, not errors" contract Stripe/Shopify/GitHub's own
-- webhook conventions use. A webhook delivered with no key at all skips
-- this check entirely -- idempotency is opt-in by the caller supplying a
-- key, not a requirement on every webhook.
CREATE TABLE IF NOT EXISTS webhook_claims (
    machine_id       TEXT NOT NULL REFERENCES machines(id) ON DELETE CASCADE,
    event_id         TEXT NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    idempotency_key  TEXT NOT NULL,
    claimed_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (machine_id, event_id, idempotency_key)
);

-- +goose Down
DROP TABLE IF EXISTS webhook_claims;
