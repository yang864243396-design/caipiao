-- +goose NO TRANSACTION
-- +goose Up
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_cloud_bet_record_identity_scheme_placed
    ON cloud_bet_record_identity (scheme_id, placed_at DESC, id DESC);

-- +goose Down
DROP INDEX CONCURRENTLY IF EXISTS idx_cloud_bet_record_identity_scheme_placed;
