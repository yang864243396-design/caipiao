-- +goose Up
ALTER TABLE bet_orders
    ADD COLUMN IF NOT EXISTS outbound_lottery_code VARCHAR(64),
    ADD COLUMN IF NOT EXISTS outbound_play_code VARCHAR(128);

COMMENT ON COLUMN bet_orders.outbound_lottery_code IS '下单时第三方彩种对接码快照（C41）';
COMMENT ON COLUMN bet_orders.outbound_play_code IS '下单时第三方玩法对接码快照 template:type_id:sub_id（C43）';

-- +goose Down
ALTER TABLE bet_orders
    DROP COLUMN IF EXISTS outbound_play_code,
    DROP COLUMN IF EXISTS outbound_lottery_code;
