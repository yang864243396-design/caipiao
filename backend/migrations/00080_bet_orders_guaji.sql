-- +goose Up
-- +goose StatementBegin
-- T4：bet_orders 第三方接单快照（guaji 账号 / 第三方注单号 / 下单主币种）

ALTER TABLE bet_orders
    ADD COLUMN IF NOT EXISTS guaji_account_id   BIGINT,
    ADD COLUMN IF NOT EXISTS third_party_bet_id VARCHAR(64),
    ADD COLUMN IF NOT EXISTS currency           VARCHAR(8);

COMMENT ON COLUMN bet_orders.guaji_account_id IS '下单时启用的第三方授权账号 id（real；sim/本地为 NULL）';
COMMENT ON COLUMN bet_orders.third_party_bet_id IS '第三方 web_bets/lott 接单返回的注单号（real）';
COMMENT ON COLUMN bet_orders.currency IS '下单时主币种快照 USDT/TRX/CNY（real）';

CREATE INDEX IF NOT EXISTS idx_bet_orders_guaji_account
    ON bet_orders (guaji_account_id, placed_at DESC)
    WHERE guaji_account_id IS NOT NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_bet_orders_guaji_account;
ALTER TABLE bet_orders
    DROP COLUMN IF EXISTS currency,
    DROP COLUMN IF EXISTS third_party_bet_id,
    DROP COLUMN IF EXISTS guaji_account_id;
-- +goose StatementEnd
