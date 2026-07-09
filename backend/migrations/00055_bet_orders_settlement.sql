-- +goose Up
-- +goose StatementBegin
ALTER TABLE bet_orders
    ADD COLUMN play_method VARCHAR(128),
    ADD COLUMN bet_payload JSONB NOT NULL DEFAULT '{}'::jsonb;

COMMENT ON COLUMN bet_orders.play_method IS '玩法展示名（如 定位胆万位），供结算引擎解析';
COMMENT ON COLUMN bet_orders.bet_payload IS '结算用 JSON：groupContent / playTypeId / subPlayId 等';

CREATE INDEX idx_bet_orders_pending_issue
    ON bet_orders (lottery_code, issue_no)
    WHERE status = 'pending';

COMMENT ON INDEX idx_bet_orders_pending_issue IS '待结算订单按彩种+期号查开奖';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_bet_orders_pending_issue;
ALTER TABLE bet_orders
    DROP COLUMN IF EXISTS bet_payload,
    DROP COLUMN IF EXISTS play_method;
-- +goose StatementEnd
