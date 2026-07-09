-- +goose Up
-- +goose StatementBegin
ALTER TABLE scheme_instances
    ADD COLUMN IF NOT EXISTS session_pnl NUMERIC(14, 2) NOT NULL DEFAULT 0;

COMMENT ON COLUMN scheme_instances.session_pnl IS '本次运行累计盈亏（从 pending 开启时归零）';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE scheme_instances DROP COLUMN IF EXISTS session_pnl;
-- +goose StatementEnd
