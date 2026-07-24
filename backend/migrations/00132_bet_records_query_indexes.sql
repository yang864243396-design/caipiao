-- +goose Up
-- +goose StatementBegin
-- 投注记录列表/汇总：按平台单号、第三方注单号等值关联与检索加速。

CREATE INDEX IF NOT EXISTS idx_cloud_bet_records_bet_order_no
    ON cloud_bet_records (bet_order_no)
    WHERE bet_order_no IS NOT NULL AND bet_order_no <> '';

COMMENT ON INDEX idx_cloud_bet_records_bet_order_no IS '云端投注按平台 bet_orders.order_no 检索/关联';

CREATE INDEX IF NOT EXISTS idx_bet_orders_third_party_bet_id
    ON bet_orders (third_party_bet_id)
    WHERE third_party_bet_id IS NOT NULL AND third_party_bet_id <> '';

COMMENT ON INDEX idx_bet_orders_third_party_bet_id IS '投注订单按第三方注单号检索';

CREATE INDEX IF NOT EXISTS idx_bet_orders_member_third_party_bet_id
    ON bet_orders (member_id, third_party_bet_id)
    WHERE third_party_bet_id IS NOT NULL AND third_party_bet_id <> '';

COMMENT ON INDEX idx_bet_orders_member_third_party_bet_id IS '会员维度第三方注单号关联（LATERAL 回退路径）';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_bet_orders_member_third_party_bet_id;
DROP INDEX IF EXISTS idx_bet_orders_third_party_bet_id;
DROP INDEX IF EXISTS idx_cloud_bet_records_bet_order_no;
-- +goose StatementEnd
