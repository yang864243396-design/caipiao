-- ?? / ???????????????????????????????

-- name: LockMemberWallet :one
SELECT
    w.id,
    w.balance::float8 AS balance,
    w.frozen_balance::float8 AS frozen_balance,
    w.version
FROM member_wallets w
WHERE w.member_id = $1
FOR UPDATE;

-- name: UpdateMemberWalletBalances :execrows
UPDATE member_wallets
SET balance = $2,
    frozen_balance = $3,
    version = version + 1,
    updated_at = now()
WHERE member_id = $1
  AND version = $4;

-- name: InsertWalletLedger :exec
INSERT INTO wallet_ledger (
    ledger_no, member_id, txn_type, delta_amount, balance_after, order_ref
) VALUES (
    $1, $2, $3, $4, $5, $6
);

-- name: InsertWalletLedgerMirror :exec
INSERT INTO wallet_ledger (
    ledger_no, member_id, txn_type, delta_amount, balance_after, order_ref,
    guaji_account_id, currency, remark
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9
);

-- name: SumBetVolumeByMember :one
SELECT COALESCE(SUM(b.amount), 0)::float8 AS total
FROM bet_orders b
WHERE b.member_id = $1
  AND b.status IN ('pending', 'win', 'lose');
