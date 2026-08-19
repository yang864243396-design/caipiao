-- +goose Up
-- +goose StatementBegin
CREATE INDEX IF NOT EXISTS idx_scheme_instances_running_lottery_updated
    ON scheme_instances (lottery_code, updated_at ASC)
    WHERE status = 'running';

COMMENT ON INDEX idx_scheme_instances_running_lottery_updated IS '开奖事件按彩种唤醒运行方案扫描';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_scheme_instances_running_lottery_updated;
-- +goose StatementEnd
