-- +goose Up
-- +goose StatementBegin
-- 投注详情：投注注数、返奖金额（第三方原值 / 模拟 amount+pnl）

ALTER TABLE cloud_bet_records
    ADD COLUMN IF NOT EXISTS bet_units INT,
    ADD COLUMN IF NOT EXISTS payout_amount NUMERIC(18, 2);

COMMENT ON COLUMN cloud_bet_records.bet_units IS '投注注数：正式优先第三方 bets_nums，模拟为本地验奖注数';
COMMENT ON COLUMN cloud_bet_records.payout_amount IS '返奖金额：pending=NULL；miss=0；real=第三方 payout 原值；sim=amount+pnl；本地兜底结算不写';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE cloud_bet_records
    DROP COLUMN IF EXISTS payout_amount,
    DROP COLUMN IF EXISTS bet_units;
-- +goose StatementEnd
