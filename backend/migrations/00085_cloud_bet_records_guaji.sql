-- +goose Up
-- +goose StatementBegin
ALTER TABLE cloud_bet_records
    ADD COLUMN IF NOT EXISTS guaji_account_id BIGINT;

COMMENT ON COLUMN cloud_bet_records.guaji_account_id IS 'real 投注明细关联的第三方授权账号 id；sim 为 NULL';

CREATE INDEX IF NOT EXISTS idx_cloud_bet_records_member_guaji_placed
    ON cloud_bet_records (member_id, guaji_account_id, placed_at DESC)
    WHERE guaji_account_id IS NOT NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_cloud_bet_records_member_guaji_placed;
ALTER TABLE cloud_bet_records
    DROP COLUMN IF EXISTS guaji_account_id;
-- +goose StatementEnd
