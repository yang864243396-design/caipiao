-- +goose Up
-- +goose StatementBegin
-- T5：wallet_ledger 第三方派奖镜像（guaji 账号 + 主币种快照）
-- B1：保留 balance_after >= 0 NOT NULL；real 镜像行 balance_after 存第三方主币种余额快照。

ALTER TABLE wallet_ledger
    ADD COLUMN IF NOT EXISTS guaji_account_id BIGINT,
    ADD COLUMN IF NOT EXISTS currency         VARCHAR(8);

COMMENT ON COLUMN wallet_ledger.guaji_account_id IS '第三方授权账号 id（real 镜像行；本地/历史为 NULL）';
COMMENT ON COLUMN wallet_ledger.currency IS 'real 镜像行主币种 USDT/TRX/CNY；balance_after 为该币种第三方余额快照';

CREATE INDEX IF NOT EXISTS idx_wallet_ledger_member_guaji
    ON wallet_ledger (member_id, guaji_account_id, created_at DESC)
    WHERE guaji_account_id IS NOT NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_wallet_ledger_member_guaji;
ALTER TABLE wallet_ledger
    DROP COLUMN IF EXISTS currency,
    DROP COLUMN IF EXISTS guaji_account_id;
-- +goose StatementEnd
