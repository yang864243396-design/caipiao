-- +goose Up
-- +goose StatementBegin
-- 用户端 /client/guaji/balance 同步时写入的三币种余额快照；Admin 列表只读此字段。

ALTER TABLE member_guaji_accounts
    ADD COLUMN IF NOT EXISTS balance_usdt NUMERIC(18, 2) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS balance_trx  NUMERIC(18, 2) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS balance_cny  NUMERIC(18, 2) NOT NULL DEFAULT 0;

COMMENT ON COLUMN member_guaji_accounts.balance_usdt IS '第三方 USDT 可用余额快照（用户端 balance 同步写入）';
COMMENT ON COLUMN member_guaji_accounts.balance_trx IS '第三方 TRX 可用余额快照（用户端 balance 同步写入）';
COMMENT ON COLUMN member_guaji_accounts.balance_cny IS '第三方 CNY 可用余额快照（用户端 balance 同步写入）';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE member_guaji_accounts
    DROP COLUMN IF EXISTS balance_cny,
    DROP COLUMN IF EXISTS balance_trx,
    DROP COLUMN IF EXISTS balance_usdt;
-- +goose StatementEnd
