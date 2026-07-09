-- +goose Up
-- +goose StatementBegin
-- 清理真实投注占位失败、未在第三方接单的脏数据
DELETE FROM cloud_bet_records
WHERE run_mode = 'real'
  AND status = 'pending'
  AND (third_party_bet_id IS NULL OR TRIM(third_party_bet_id) = '');
-- +goose StatementEnd

-- +goose Down
-- 数据清理不可逆
