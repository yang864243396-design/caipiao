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
    COALESCE(sch.scheme_name, '') AS scheme_name,
    COALESCE(sch.play_method, '') AS play_method,
    COALESCE(sch.lottery_name, '') AS lottery_name
FROM wallet_ledger l
LEFT JOIN LATERAL (
    SELECT
        c.scheme_name,
        COALESCE(bo.play_method, '') AS play_method,
        COALESCE(bo.lottery_name, '') AS lottery_name
    FROM cloud_bet_records c
    LEFT JOIN bet_orders bo
      ON bo.member_id = c.member_id
     AND bo.order_no = c.bet_order_no
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
    COALESCE(sch.scheme_name, '') AS scheme_name,
    COALESCE(sch.play_method, '') AS play_method,
    COALESCE(sch.lottery_name, '') AS lottery_name
FROM wallet_ledger l
LEFT JOIN LATERAL (
    SELECT
        c.scheme_name,
        COALESCE(bo.play_method, '') AS play_method,
        COALESCE(bo.lottery_name, '') AS lottery_name
    FROM cloud_bet_records c
    LEFT JOIN bet_orders bo
      ON bo.member_id = c.member_id
     AND bo.order_no = c.bet_order_no
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
    COALESCE(sch.scheme_name, '') AS scheme_name,
    COALESCE(sch.play_method, '') AS play_method,
    COALESCE(sch.lottery_name, '') AS lottery_name
FROM wallet_ledger l
LEFT JOIN LATERAL (
    SELECT
        c.scheme_name,
        COALESCE(bo.play_method, '') AS play_method,
        COALESCE(bo.lottery_name, '') AS lottery_name
    FROM cloud_bet_records c
    LEFT JOIN bet_orders bo
      ON bo.member_id = c.member_id
     AND bo.order_no = c.bet_order_no
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
WITH ledger_page AS MATERIALIZED (
SELECT
    l.id,
    l.ledger_no,
    m.account,
    l.txn_type,
    l.delta_amount::float8 AS delta_amount,
    l.balance_after::float8 AS balance_after,
    COALESCE(l.currency, 'CNY') AS currency,
    l.created_at,
    l.member_id,
    l.order_ref,
    l.guaji_account_id
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
  )
ORDER BY l.created_at DESC, l.id DESC
LIMIT sqlc.arg(row_limit) OFFSET sqlc.arg(row_offset)
)
SELECT
    p.id,
    p.ledger_no,
    p.account,
    p.txn_type,
    p.delta_amount,
    p.balance_after,
    p.currency,
    p.created_at,
    COALESCE(by_order.scheme_name, by_legacy.scheme_name, '') AS scheme_name
FROM ledger_page p
LEFT JOIN LATERAL (
    SELECT c.scheme_name
    FROM cloud_bet_records c
    WHERE NULLIF(TRIM(p.order_ref), '') IS NOT NULL
      AND c.member_id = p.member_id
      AND c.bet_order_no = NULLIF(TRIM(p.order_ref), '')
    LIMIT 1
) by_order ON true
LEFT JOIN LATERAL (
    SELECT c.scheme_name
    FROM cloud_bet_records c
    WHERE NULLIF(TRIM(p.order_ref), '') IS NULL
      AND c.member_id = p.member_id
      AND c.placed_at >= p.created_at - INTERVAL '5 seconds'
      AND c.placed_at <= p.created_at + INTERVAL '5 seconds'
      AND ABS(c.amount - ABS(p.delta_amount)) < 0.001
      AND c.guaji_account_id IS NOT DISTINCT FROM p.guaji_account_id
    ORDER BY c.placed_at DESC
    LIMIT 1
) by_legacy ON true
ORDER BY p.created_at DESC, p.id DESC;
