-- +goose Up
-- +goose StatementBegin
CREATE INDEX idx_bet_orders_pending_scan
    ON bet_orders (placed_at ASC, id ASC)
    WHERE status = 'pending';

COMMENT ON INDEX idx_bet_orders_pending_scan IS '结算 worker 扫描：待结算订单按投注时间顺序取批';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_bet_orders_pending_scan;
-- +goose StatementEnd
