-- CAP-W06: transactional outbox for slow, best-effort actions -- today
-- `notify` (CAP-A03/A04) and CAP-I01 subscription fan-out execute inline
-- inside the triggering request's own transaction, extending its latency
-- and holding row locks open for however long fan-out takes. A row here is
-- INSERTed by the same code path, inside the same still-open request
-- transaction (atomic with the record write, for free, same mechanism
-- CAP-X12 already relies on) -- but the actual side effect (a Notification
-- row, a subscriber record) is performed later, off the request path, by
-- runOutboxDispatcher (cmd/server/main.go).
--
-- Deliberately NOT used for create_record/cross_set_field/batch_generate --
-- CAP-X12 hardened those three to abort the whole event on failure, which
-- an async dispatcher (running after the triggering transaction has
-- already committed) cannot do. See capability-registry.md's CAP-W06 row
-- for the full scope note.
--
-- params holds the ALREADY-RESOLVED payload -- the dispatcher never
-- re-derives what to do from metadata, only performs the write the
-- synchronous path had already decided on at enqueue time.
--
-- RLS from creation (unlike notifications' original two-phase 005/009
-- rollout) -- CAP-X06 is already fully cut over by the time this table is
-- introduced, so there's no bridging period to account for.
CREATE TABLE IF NOT EXISTS action_outbox (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id   TEXT NOT NULL REFERENCES workspaces(id),
    action_type    TEXT NOT NULL,      -- "notify" | "subscription"
    params         JSONB NOT NULL,
    correlation_id TEXT,               -- captured at enqueue time -- the
                                        -- dispatcher has no HTTP request ctx
                                        -- later to pull a fresh one from
    claimed_at     TIMESTAMPTZ,
    completed_at   TIMESTAMPTZ,
    failed_at      TIMESTAMPTZ,
    error          TEXT,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_action_outbox_pending ON action_outbox (workspace_id, created_at) WHERE claimed_at IS NULL;

ALTER TABLE action_outbox ENABLE ROW LEVEL SECURITY;
ALTER TABLE action_outbox FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS ws_isolation ON action_outbox;
CREATE POLICY ws_isolation ON action_outbox
    USING (workspace_id = current_setting('app.workspace_id', true));
