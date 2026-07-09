-- +goose Up
-- +goose StatementBegin
ALTER TABLE scheme_instances
    ADD COLUMN IF NOT EXISTS bet_failed_detail VARCHAR(512);

COMMENT ON COLUMN scheme_instances.bet_failed_detail IS '投注失败时第三方返回原因（展示用）';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE scheme_instances
    DROP COLUMN IF EXISTS bet_failed_detail;
-- +goose StatementEnd
