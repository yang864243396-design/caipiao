-- +goose Up
-- +goose StatementBegin
-- 后台投注账变列表：按全站日期倒序分页，不再依赖 member_id 前缀索引。
CREATE INDEX IF NOT EXISTS idx_wallet_ledger_admin_bet_payout_created
    ON wallet_ledger (created_at DESC, id DESC)
    WHERE txn_type IN ('bet_debit', 'payout');

-- 有订单号的账变按会员+订单号精确补回方案名。
CREATE INDEX IF NOT EXISTS idx_cloud_bet_records_member_bet_order_no
    ON cloud_bet_records (member_id, bet_order_no)
    INCLUDE (scheme_name)
    WHERE bet_order_no IS NOT NULL AND bet_order_no <> '';

-- 历史无订单号账变仅在 ±5 秒窗口内兜底匹配，按会员/挂机账户/下注时间检索。
CREATE INDEX IF NOT EXISTS idx_cloud_bet_records_member_guaji_placed_lookup
    ON cloud_bet_records (member_id, guaji_account_id, placed_at DESC)
    INCLUDE (amount, scheme_name);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_cloud_bet_records_member_guaji_placed_lookup;
DROP INDEX IF EXISTS idx_cloud_bet_records_member_bet_order_no;
DROP INDEX IF EXISTS idx_wallet_ledger_admin_bet_payout_created;
-- +goose StatementEnd
