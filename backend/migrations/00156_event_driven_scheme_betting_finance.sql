-- +goose Up
ALTER TABLE scheme_bet_outbox
    ADD COLUMN IF NOT EXISTS provider_account_id BIGINT,
    ADD COLUMN IF NOT EXISTS provider_currency VARCHAR(16),
    ADD COLUMN IF NOT EXISTS provider_amount NUMERIC(18, 3),
    ADD COLUMN IF NOT EXISTS local_order_no VARCHAR(64),
    ADD COLUMN IF NOT EXISTS local_cloud_record_id BIGINT REFERENCES cloud_bet_records(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS financial_finalized_at TIMESTAMPTZ;

CREATE UNIQUE INDEX IF NOT EXISTS uq_scheme_bet_outbox_local_order
    ON scheme_bet_outbox (local_order_no)
    WHERE local_order_no IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_scheme_bet_outbox_acceptance_recovery
    ON scheme_bet_outbox (terminal_at, id)
    WHERE state = 'accepted' AND financial_finalized_at IS NULL;

-- +goose Down
DROP INDEX IF EXISTS idx_scheme_bet_outbox_acceptance_recovery;
DROP INDEX IF EXISTS uq_scheme_bet_outbox_local_order;
ALTER TABLE scheme_bet_outbox
    DROP COLUMN IF EXISTS financial_finalized_at,
    DROP COLUMN IF EXISTS local_cloud_record_id,
    DROP COLUMN IF EXISTS local_order_no,
    DROP COLUMN IF EXISTS provider_amount,
    DROP COLUMN IF EXISTS provider_currency,
    DROP COLUMN IF EXISTS provider_account_id;
