-- +goose Up
ALTER TABLE scheme_bet_outbox
    ADD COLUMN IF NOT EXISTS provider_reconcile_attempts INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS provider_reconcile_next_at TIMESTAMPTZ NOT NULL DEFAULT now();

CREATE INDEX IF NOT EXISTS idx_scheme_bet_outbox_provider_reconcile
    ON scheme_bet_outbox (provider_reconcile_next_at, id)
    WHERE state IN ('sent_unknown', 'external_acceptance_unknown');

-- +goose Down
DROP INDEX IF EXISTS idx_scheme_bet_outbox_provider_reconcile;
ALTER TABLE scheme_bet_outbox
    DROP COLUMN IF EXISTS provider_reconcile_next_at,
    DROP COLUMN IF EXISTS provider_reconcile_attempts;
