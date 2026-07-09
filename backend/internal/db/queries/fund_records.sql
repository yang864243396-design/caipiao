-- name: ListMemberRechargeOrders :many
SELECT order_no, amount, channel, status, paid_at, created_at
FROM recharge_orders
WHERE member_id = $1
ORDER BY created_at DESC
LIMIT sqlc.arg('row_limit');

-- name: ListMemberFundRecords :many
SELECT
    l.id,
    l.ledger_no,
    l.txn_type,
    l.delta_amount::float8 AS delta_amount,
    l.balance_after::float8 AS balance_after,
    COALESCE(l.currency, 'CNY') AS currency,
    l.created_at,
    COALESCE(sch.scheme_name, '') AS scheme_name
FROM wallet_ledger l
LEFT JOIN LATERAL (
    SELECT c.scheme_name
    FROM cloud_bet_records c
    WHERE c.member_id = l.member_id
      AND (
        (NULLIF(TRIM(l.order_ref), '') IS NOT NULL AND c.bet_order_no = l.order_ref)
        OR (
          NULLIF(TRIM(l.order_ref), '') IS NULL
          AND ABS(EXTRACT(EPOCH FROM (c.placed_at - l.created_at))) <= 5
          AND ABS(c.amount::float8 - ABS(l.delta_amount::float8)) < 0.001
          AND c.guaji_account_id IS NOT DISTINCT FROM l.guaji_account_id
        )
      )
    ORDER BY
      CASE WHEN NULLIF(TRIM(l.order_ref), '') IS NOT NULL AND c.bet_order_no = l.order_ref THEN 0 ELSE 1 END,
      ABS(EXTRACT(EPOCH FROM (c.placed_at - l.created_at)))
    LIMIT 1
) sch ON true
WHERE l.member_id = $1
  AND l.guaji_account_id = sqlc.arg(guaji_account_id)
  AND l.txn_type IN ('bet_debit', 'payout')
  AND l.created_at >= sqlc.arg(time_from)
  AND l.created_at < sqlc.arg(time_to)
  AND (
    sqlc.narg(flow_dir)::text IS NULL
    OR sqlc.narg(flow_dir)::text = ''
    OR sqlc.narg(flow_dir)::text = 'all'
    OR (sqlc.narg(flow_dir)::text = 'income' AND l.delta_amount > 0)
    OR (sqlc.narg(flow_dir)::text = 'expense' AND l.delta_amount < 0)
  )
  AND (
    sqlc.narg(currency)::text IS NULL
    OR sqlc.narg(currency)::text = ''
    OR COALESCE(l.currency, 'CNY') = sqlc.narg(currency)::text
  )
ORDER BY l.created_at DESC, l.id DESC
LIMIT sqlc.arg(row_limit);

-- name: CountMemberFundRecords :one
SELECT COUNT(*)::bigint AS count
FROM wallet_ledger l
WHERE l.member_id = $1
  AND l.guaji_account_id = sqlc.arg(guaji_account_id)
  AND l.txn_type IN ('bet_debit', 'payout')
  AND l.created_at >= sqlc.arg(time_from)
  AND l.created_at < sqlc.arg(time_to)
  AND (
    sqlc.narg(flow_dir)::text IS NULL
    OR sqlc.narg(flow_dir)::text = ''
    OR sqlc.narg(flow_dir)::text = 'all'
    OR (sqlc.narg(flow_dir)::text = 'income' AND l.delta_amount > 0)
    OR (sqlc.narg(flow_dir)::text = 'expense' AND l.delta_amount < 0)
  )
  AND (
    sqlc.narg(currency)::text IS NULL
    OR sqlc.narg(currency)::text = ''
    OR COALESCE(l.currency, 'CNY') = sqlc.narg(currency)::text
  );

-- name: ListMemberFundRecordsPaged :many
SELECT
    l.id,
    l.ledger_no,
    l.txn_type,
    l.delta_amount::float8 AS delta_amount,
    l.balance_after::float8 AS balance_after,
    COALESCE(l.currency, 'CNY') AS currency,
    l.created_at,
    COALESCE(sch.scheme_name, '') AS scheme_name
FROM wallet_ledger l
LEFT JOIN LATERAL (
    SELECT c.scheme_name
    FROM cloud_bet_records c
    WHERE c.member_id = l.member_id
      AND (
        (NULLIF(TRIM(l.order_ref), '') IS NOT NULL AND c.bet_order_no = l.order_ref)
        OR (
          NULLIF(TRIM(l.order_ref), '') IS NULL
          AND ABS(EXTRACT(EPOCH FROM (c.placed_at - l.created_at))) <= 5
          AND ABS(c.amount::float8 - ABS(l.delta_amount::float8)) < 0.001
          AND c.guaji_account_id IS NOT DISTINCT FROM l.guaji_account_id
        )
      )
    ORDER BY
      CASE WHEN NULLIF(TRIM(l.order_ref), '') IS NOT NULL AND c.bet_order_no = l.order_ref THEN 0 ELSE 1 END,
      ABS(EXTRACT(EPOCH FROM (c.placed_at - l.created_at)))
    LIMIT 1
) sch ON true
WHERE l.member_id = $1
  AND l.guaji_account_id = sqlc.arg(guaji_account_id)
  AND l.txn_type IN ('bet_debit', 'payout')
  AND l.created_at >= sqlc.arg(time_from)
  AND l.created_at < sqlc.arg(time_to)
  AND (
    sqlc.narg(flow_dir)::text IS NULL
    OR sqlc.narg(flow_dir)::text = ''
    OR sqlc.narg(flow_dir)::text = 'all'
    OR (sqlc.narg(flow_dir)::text = 'income' AND l.delta_amount > 0)
    OR (sqlc.narg(flow_dir)::text = 'expense' AND l.delta_amount < 0)
  )
  AND (
    sqlc.narg(currency)::text IS NULL
    OR sqlc.narg(currency)::text = ''
    OR COALESCE(l.currency, 'CNY') = sqlc.narg(currency)::text
  )
ORDER BY l.created_at DESC, l.id DESC
LIMIT sqlc.arg(row_limit) OFFSET sqlc.arg(row_offset);

-- name: ListMemberFundRecordsAfterCursor :many
SELECT
    l.id,
    l.ledger_no,
    l.txn_type,
    l.delta_amount::float8 AS delta_amount,
    l.balance_after::float8 AS balance_after,
    COALESCE(l.currency, 'CNY') AS currency,
    l.created_at,
    COALESCE(sch.scheme_name, '') AS scheme_name
FROM wallet_ledger l
LEFT JOIN LATERAL (
    SELECT c.scheme_name
    FROM cloud_bet_records c
    WHERE c.member_id = l.member_id
      AND (
        (NULLIF(TRIM(l.order_ref), '') IS NOT NULL AND c.bet_order_no = l.order_ref)
        OR (
          NULLIF(TRIM(l.order_ref), '') IS NULL
          AND ABS(EXTRACT(EPOCH FROM (c.placed_at - l.created_at))) <= 5
          AND ABS(c.amount::float8 - ABS(l.delta_amount::float8)) < 0.001
          AND c.guaji_account_id IS NOT DISTINCT FROM l.guaji_account_id
        )
      )
    ORDER BY
      CASE WHEN NULLIF(TRIM(l.order_ref), '') IS NOT NULL AND c.bet_order_no = l.order_ref THEN 0 ELSE 1 END,
      ABS(EXTRACT(EPOCH FROM (c.placed_at - l.created_at)))
    LIMIT 1
) sch ON true
WHERE l.member_id = $1
  AND l.guaji_account_id = sqlc.arg(guaji_account_id)
  AND l.txn_type IN ('bet_debit', 'payout')
  AND l.created_at >= sqlc.arg(time_from)
  AND l.created_at < sqlc.arg(time_to)
  AND (
    sqlc.narg(flow_dir)::text IS NULL
    OR sqlc.narg(flow_dir)::text = ''
    OR sqlc.narg(flow_dir)::text = 'all'
    OR (sqlc.narg(flow_dir)::text = 'income' AND l.delta_amount > 0)
    OR (sqlc.narg(flow_dir)::text = 'expense' AND l.delta_amount < 0)
  )
  AND (
    sqlc.narg(currency)::text IS NULL
    OR sqlc.narg(currency)::text = ''
    OR COALESCE(l.currency, 'CNY') = sqlc.narg(currency)::text
  )
  AND (
    l.created_at < sqlc.arg(cursor_time)
    OR (l.created_at = sqlc.arg(cursor_time) AND l.id < sqlc.arg(cursor_id))
  )
ORDER BY l.created_at DESC, l.id DESC
LIMIT sqlc.arg(row_limit);

-- name: CountAdminFundRecords :one
SELECT COUNT(*)::bigint
FROM wallet_ledger l
INNER JOIN members m ON m.id = l.member_id
WHERE l.txn_type IN ('bet_debit', 'payout')
  AND l.created_at >= sqlc.arg(time_from)
  AND l.created_at < sqlc.arg(time_to)
  AND (
    sqlc.narg(member_account)::text IS NULL
    OR sqlc.narg(member_account)::text = ''
    OR m.account ILIKE '%' || sqlc.narg(member_account)::text || '%'
  )
  AND (
    sqlc.narg(ledger_no)::text IS NULL
    OR sqlc.narg(ledger_no)::text = ''
    OR l.ledger_no ILIKE '%' || sqlc.narg(ledger_no)::text || '%'
  )
  AND (
    sqlc.narg(flow_dir)::text IS NULL
    OR sqlc.narg(flow_dir)::text = ''
    OR sqlc.narg(flow_dir)::text = 'all'
    OR (sqlc.narg(flow_dir)::text = 'income' AND l.delta_amount > 0)
    OR (sqlc.narg(flow_dir)::text = 'expense' AND l.delta_amount < 0)
  )
  AND (
    sqlc.narg(currency)::text IS NULL
    OR sqlc.narg(currency)::text = ''
    OR COALESCE(l.currency, 'CNY') = sqlc.narg(currency)::text
  );

-- name: ListAdminFundRecordsPaged :many
SELECT
    l.id,
    l.ledger_no,
    m.account,
    l.txn_type,
    l.delta_amount::float8 AS delta_amount,
    l.balance_after::float8 AS balance_after,
    COALESCE(l.currency, 'CNY') AS currency,
    l.created_at,
    COALESCE(sch.scheme_name, '') AS scheme_name
FROM wallet_ledger l
INNER JOIN members m ON m.id = l.member_id
LEFT JOIN LATERAL (
    SELECT c.scheme_name
    FROM cloud_bet_records c
    WHERE c.member_id = l.member_id
      AND (
        (NULLIF(TRIM(l.order_ref), '') IS NOT NULL AND c.bet_order_no = l.order_ref)
        OR (
          NULLIF(TRIM(l.order_ref), '') IS NULL
          AND ABS(EXTRACT(EPOCH FROM (c.placed_at - l.created_at))) <= 5
          AND ABS(c.amount::float8 - ABS(l.delta_amount::float8)) < 0.001
          AND c.guaji_account_id IS NOT DISTINCT FROM l.guaji_account_id
        )
      )
    ORDER BY
      CASE WHEN NULLIF(TRIM(l.order_ref), '') IS NOT NULL AND c.bet_order_no = l.order_ref THEN 0 ELSE 1 END,
      ABS(EXTRACT(EPOCH FROM (c.placed_at - l.created_at)))
    LIMIT 1
) sch ON true
WHERE l.txn_type IN ('bet_debit', 'payout')
  AND l.created_at >= sqlc.arg(time_from)
  AND l.created_at < sqlc.arg(time_to)
  AND (
    sqlc.narg(member_account)::text IS NULL
    OR sqlc.narg(member_account)::text = ''
    OR m.account ILIKE '%' || sqlc.narg(member_account)::text || '%'
  )
  AND (
    sqlc.narg(ledger_no)::text IS NULL
    OR sqlc.narg(ledger_no)::text = ''
    OR l.ledger_no ILIKE '%' || sqlc.narg(ledger_no)::text || '%'
  )
  AND (
    sqlc.narg(flow_dir)::text IS NULL
    OR sqlc.narg(flow_dir)::text = ''
    OR sqlc.narg(flow_dir)::text = 'all'
    OR (sqlc.narg(flow_dir)::text = 'income' AND l.delta_amount > 0)
    OR (sqlc.narg(flow_dir)::text = 'expense' AND l.delta_amount < 0)
  )
  AND (
    sqlc.narg(currency)::text IS NULL
    OR sqlc.narg(currency)::text = ''
    OR COALESCE(l.currency, 'CNY') = sqlc.narg(currency)::text
  )
ORDER BY l.created_at DESC, l.id DESC
LIMIT sqlc.arg(row_limit) OFFSET sqlc.arg(row_offset);
