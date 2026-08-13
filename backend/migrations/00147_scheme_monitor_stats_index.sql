-- +goose Up
-- +goose StatementBegin
CREATE INDEX IF NOT EXISTS idx_cloud_bet_records_scheme_placed_stats
    ON cloud_bet_records (scheme_id, placed_at DESC)
    INCLUDE (status, currency);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_cloud_bet_records_scheme_placed_stats;
-- +goose StatementEnd
