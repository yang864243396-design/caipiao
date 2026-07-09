-- +goose Up
ALTER TABLE scheme_instances
    ADD COLUMN IF NOT EXISTS start_skip_period VARCHAR(64),
    ADD COLUMN IF NOT EXISTS start_skip_close_at TIMESTAMPTZ;

COMMENT ON COLUMN scheme_instances.start_skip_period IS '方案开启时无条件跳过的第三方 periods 期号（开启瞬间快照）';
COMMENT ON COLUMN scheme_instances.start_skip_close_at IS '开启时跳过期封盘时刻（开启瞬间快照，用于激活与倒计时）';

-- +goose Down
ALTER TABLE scheme_instances
    DROP COLUMN IF EXISTS start_skip_close_at,
    DROP COLUMN IF EXISTS start_skip_period;
