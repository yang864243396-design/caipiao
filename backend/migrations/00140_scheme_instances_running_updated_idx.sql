-- +goose Up
-- +goose StatementBegin
-- 方案 Worker 每 tick：WHERE status='running' ORDER BY updated_at
CREATE INDEX IF NOT EXISTS idx_scheme_instances_running_updated
    ON scheme_instances (updated_at ASC)
    WHERE status = 'running';

COMMENT ON INDEX idx_scheme_instances_running_updated IS 'Scheme Worker 扫描 running 实例（按 updated_at 公平序）';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_scheme_instances_running_updated;
-- +goose StatementEnd
