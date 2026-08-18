-- +goose NO TRANSACTION
-- +goose Up
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_scheme_instances_updated_id
    ON scheme_instances (updated_at ASC, id ASC);

-- +goose Down
DROP INDEX CONCURRENTLY IF EXISTS idx_scheme_instances_updated_id;
