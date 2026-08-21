-- +goose NO TRANSACTION
-- +goose Up
-- The runtime diagnostic reads one newest decision for one instance. Keep the
-- admin-only read bounded by a matching index rather than a growing history scan.
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_scheme_period_decisions_scheme_latest
    ON scheme_period_decisions (scheme_id, decided_at DESC, id DESC);

-- +goose Down
DROP INDEX CONCURRENTLY IF EXISTS idx_scheme_period_decisions_scheme_latest;
