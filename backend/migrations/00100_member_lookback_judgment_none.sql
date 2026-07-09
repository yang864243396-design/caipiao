-- +goose Up
-- +goose StatementBegin
ALTER TABLE member_lookback_settings
    DROP CONSTRAINT IF EXISTS chk_member_lookback_judgment;

ALTER TABLE member_lookback_settings
    ADD CONSTRAINT chk_member_lookback_judgment CHECK (
        judgment IN ('individual', 'overall', '')
    );

COMMENT ON COLUMN member_lookback_settings.judgment IS 'individual 个别判断 / overall 整体判断 / 空 未选择';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE member_lookback_settings
    DROP CONSTRAINT IF EXISTS chk_member_lookback_judgment;

UPDATE member_lookback_settings SET judgment = 'individual' WHERE judgment = '';

ALTER TABLE member_lookback_settings
    ADD CONSTRAINT chk_member_lookback_judgment CHECK (
        judgment IN ('individual', 'overall')
    );

COMMENT ON COLUMN member_lookback_settings.judgment IS 'individual 个别判断 / overall 整体判断';
-- +goose StatementEnd
