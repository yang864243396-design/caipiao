-- +goose Up
ALTER TABLE scheme_bet_outbox
    ADD COLUMN IF NOT EXISTS reconciliation_evidence JSONB;

CREATE INDEX IF NOT EXISTS idx_scheme_bet_outbox_reconciliation
    ON scheme_bet_outbox (safe_deadline_at, id)
    WHERE state IN ('sent_unknown', 'external_acceptance_unknown');

-- +goose Down
DROP INDEX IF EXISTS idx_scheme_bet_outbox_reconciliation;
ALTER TABLE scheme_bet_outbox
    DROP COLUMN IF EXISTS reconciliation_evidence;
