-- +goose Up
-- +goose StatementBegin
ALTER TABLE wallet_ledger
    ADD COLUMN IF NOT EXISTS lottery_code VARCHAR(32),
    ADD COLUMN IF NOT EXISTS lottery_name VARCHAR(64);

CREATE INDEX IF NOT EXISTS idx_wallet_ledger_member_guaji_lottery_created
    ON wallet_ledger (member_id, guaji_account_id, lottery_code, created_at DESC)
    WHERE txn_type IN ('bet_debit', 'payout');

UPDATE wallet_ledger l
SET lottery_code = b.lottery_code,
    lottery_name = b.lottery_name
FROM bet_orders b
WHERE l.order_ref = b.order_no
  AND l.txn_type IN ('bet_debit', 'payout')
  AND l.lottery_code IS NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_wallet_ledger_member_guaji_lottery_created;
ALTER TABLE wallet_ledger
    DROP COLUMN IF EXISTS lottery_name,
    DROP COLUMN IF EXISTS lottery_code;
-- +goose StatementEnd
