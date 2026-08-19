-- +goose Up
-- +goose StatementBegin
ALTER TABLE scheme_bet_outbox
    ADD COLUMN IF NOT EXISTS ready_next_attempt_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    ADD COLUMN IF NOT EXISTS reconcile_next_attempt_at TIMESTAMPTZ NOT NULL DEFAULT now();

UPDATE scheme_bet_outbox
SET ready_published_at = COALESCE(ready_published_at, updated_at, created_at),
    ready_publish_attempts = GREATEST(ready_publish_attempts, 1)
WHERE ready_published_at IS NULL
  AND state NOT IN ('pending', 'leased');

UPDATE scheme_bet_outbox
SET reconcile_published_at = COALESCE(reconcile_published_at, terminal_at, updated_at, created_at),
    reconcile_published_state = state,
    reconcile_publish_attempts = GREATEST(reconcile_publish_attempts, 1)
WHERE state NOT IN ('pending', 'leased', 'sent_unknown', 'external_acceptance_unknown')
  AND reconcile_published_state IS DISTINCT FROM state;

CREATE INDEX IF NOT EXISTS idx_scheme_bet_outbox_ready_retry
    ON scheme_bet_outbox (ready_next_attempt_at, safe_deadline_at, id)
    WHERE mode IN ('gray', 'production') AND ready_published_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_scheme_bet_outbox_reconcile_retry
    ON scheme_bet_outbox (reconcile_next_attempt_at, updated_at, id)
    WHERE mode IN ('gray', 'production')
      AND state NOT IN ('pending', 'leased')
      AND reconcile_published_state IS DISTINCT FROM state;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_scheme_bet_outbox_reconcile_retry;
DROP INDEX IF EXISTS idx_scheme_bet_outbox_ready_retry;
ALTER TABLE scheme_bet_outbox
    DROP COLUMN IF EXISTS reconcile_next_attempt_at,
    DROP COLUMN IF EXISTS ready_next_attempt_at;
-- +goose StatementEnd
