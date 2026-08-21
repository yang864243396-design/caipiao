-- +goose Up
ALTER TABLE scheme_instances
    ADD COLUMN IF NOT EXISTS lookback_round_reset_pending BOOLEAN NOT NULL DEFAULT FALSE;

COMMENT ON COLUMN scheme_instances.lookback_round_reset_pending IS
    'Event strategy must reset round_index to 0 before freezing the next bet';

-- +goose Down
ALTER TABLE scheme_instances
    DROP COLUMN IF EXISTS lookback_round_reset_pending;
