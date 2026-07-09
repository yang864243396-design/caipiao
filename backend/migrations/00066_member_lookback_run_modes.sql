-- +goose Up
-- +goose StatementBegin
ALTER TABLE member_lookback_settings
    ALTER COLUMN run_mode TYPE VARCHAR(16);

ALTER TABLE member_lookback_settings
    ALTER COLUMN run_mode SET DEFAULT '';

ALTER TABLE member_lookback_settings
    DROP CONSTRAINT IF EXISTS chk_member_lookback_run_mode;

ALTER TABLE member_lookback_settings
    ADD CONSTRAINT chk_member_lookback_run_mode CHECK (
        run_mode IN ('', 'real', 'sim', 'real,sim', 'sim,real')
    );

COMMENT ON COLUMN member_lookback_settings.run_mode IS '逗号分隔：real 正式 / sim 模拟；空表示未选择';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE member_lookback_settings
    DROP CONSTRAINT IF EXISTS chk_member_lookback_run_mode;

UPDATE member_lookback_settings
SET run_mode = 'real'
WHERE run_mode = '' OR run_mode = 'real,sim' OR run_mode = 'sim,real';

ALTER TABLE member_lookback_settings
    ALTER COLUMN run_mode TYPE VARCHAR(8);

ALTER TABLE member_lookback_settings
    ALTER COLUMN run_mode SET DEFAULT 'real';

ALTER TABLE member_lookback_settings
    ADD CONSTRAINT chk_member_lookback_run_mode CHECK (run_mode IN ('real', 'sim'));
-- +goose StatementEnd
