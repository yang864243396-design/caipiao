-- +goose Up
-- +goose StatementBegin
-- 会员模拟方案日启动计数（北京时间自然日，晚上 24 点刷新）

ALTER TABLE members
    ADD COLUMN IF NOT EXISTS sim_scheme_starts_date DATE,
    ADD COLUMN IF NOT EXISTS sim_scheme_starts_count INT NOT NULL DEFAULT 0;

COMMENT ON COLUMN members.sim_scheme_starts_date IS '模拟方案日启动计数所属自然日（Asia/Shanghai）';
COMMENT ON COLUMN members.sim_scheme_starts_count IS '当日模拟方案启动次数（上限 5）';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE members DROP COLUMN IF EXISTS sim_scheme_starts_count;
ALTER TABLE members DROP COLUMN IF EXISTS sim_scheme_starts_date;
-- +goose StatementEnd
