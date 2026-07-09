-- +goose Up
-- +goose StatementBegin
ALTER TABLE scheme_instances
    ADD COLUMN IF NOT EXISTS running_since TIMESTAMPTZ;

COMMENT ON COLUMN scheme_instances.running_since IS '当前 running 段开始时刻；暂停时累加至 run_time_sec 并清空';

UPDATE scheme_instances
SET running_since = updated_at
WHERE status = 'running' AND running_since IS NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE scheme_instances DROP COLUMN IF EXISTS running_since;
-- +goose StatementEnd
