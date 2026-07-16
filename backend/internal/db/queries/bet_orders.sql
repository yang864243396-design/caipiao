-- name: ListBetOrders :many
SELECT
    b.id,
    b.order_no,
    b.third_party_bet_id,
    b.lottery_name,
    b.issue_no,
    b.amount::float8 AS amount,
    b.pnl::float8 AS pnl,
    b.status,
    b.placed_at
FROM bet_orders b
WHERE b.member_id = $1
  AND b.placed_at >= sqlc.arg(time_from)
  AND b.placed_at < sqlc.arg(time_to)
  AND (
    sqlc.narg(status)::text IS NULL
    OR b.status = sqlc.narg(status)::text
  )
  AND (
    sqlc.narg(lottery_category)::text IS NULL
    OR b.lottery_category = sqlc.narg(lottery_category)::text
  )
  AND (
    sqlc.narg(lottery_code)::text IS NULL
    OR b.lottery_code = sqlc.narg(lottery_code)::text
  )
  AND (
    sqlc.narg(order_no)::text IS NULL
    OR sqlc.narg(order_no)::text = ''
    OR b.order_no ILIKE '%' || sqlc.narg(order_no)::text || '%'
    OR b.third_party_bet_id ILIKE '%' || sqlc.narg(order_no)::text || '%'
  )
ORDER BY b.placed_at DESC, b.id DESC
LIMIT sqlc.arg(row_limit);

-- name: ListBetOrdersAfterCursor :many
SELECT
    b.id,
    b.order_no,
    b.third_party_bet_id,
    b.lottery_name,
    b.issue_no,
    b.amount::float8 AS amount,
    b.pnl::float8 AS pnl,
    b.status,
    b.placed_at
FROM bet_orders b
WHERE b.member_id = $1
  AND b.placed_at >= sqlc.arg(time_from)
  AND b.placed_at < sqlc.arg(time_to)
  AND (
    sqlc.narg(status)::text IS NULL
    OR b.status = sqlc.narg(status)::text
  )
  AND (
    sqlc.narg(lottery_category)::text IS NULL
    OR b.lottery_category = sqlc.narg(lottery_category)::text
  )
  AND (
    sqlc.narg(lottery_code)::text IS NULL
    OR b.lottery_code = sqlc.narg(lottery_code)::text
  )
  AND (
    sqlc.narg(order_no)::text IS NULL
    OR sqlc.narg(order_no)::text = ''
    OR b.order_no ILIKE '%' || sqlc.narg(order_no)::text || '%'
    OR b.third_party_bet_id ILIKE '%' || sqlc.narg(order_no)::text || '%'
  )
  AND (
    b.placed_at < sqlc.arg(cursor_time)
    OR (b.placed_at = sqlc.arg(cursor_time) AND b.id < sqlc.arg(cursor_id))
  )
ORDER BY b.placed_at DESC, b.id DESC
LIMIT sqlc.arg(row_limit);

-- name: GetBetOrderCursorAnchor :one
SELECT b.placed_at, b.id
FROM bet_orders b
WHERE b.member_id = $1
  AND b.order_no = $2;

-- name: InsertBetOrder :one
INSERT INTO bet_orders (
    order_no, member_id, lottery_code, lottery_name, lottery_category,
    issue_no, amount, pnl, status, placed_at, play_method, bet_payload,
    outbound_lottery_code, outbound_play_code,
    guaji_account_id, third_party_bet_id, currency
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, 0, 'pending', now(), $8, $9, $10, $11,
    $12, $13, $14
)
RETURNING id, order_no, lottery_code, lottery_name, issue_no, amount::float8 AS amount, status, placed_at;

-- name: ListPendingBetOrdersForSettlement :many
SELECT
    b.id,
    b.order_no,
    b.member_id,
    b.lottery_code,
    b.issue_no,
    b.amount::float8 AS amount,
    COALESCE(b.play_method, '') AS play_method,
    b.bet_payload,
    b.guaji_account_id
FROM bet_orders b
WHERE b.status = 'pending'
  AND b.guaji_account_id IS NULL
ORDER BY b.placed_at ASC, b.id ASC
LIMIT sqlc.arg(row_limit);

-- name: ListPendingGuajiBetOrders :many
SELECT
    b.id,
    b.order_no,
    b.member_id,
    b.guaji_account_id,
    b.third_party_bet_id,
    b.amount::float8 AS amount,
    COALESCE(b.currency, '') AS currency
FROM bet_orders b
WHERE b.status = 'pending'
  AND b.guaji_account_id IS NOT NULL
  AND b.third_party_bet_id IS NOT NULL
ORDER BY b.placed_at ASC, b.id ASC
LIMIT sqlc.arg(row_limit);

-- name: SettleBetOrder :execrows
UPDATE bet_orders
SET status = $2,
    pnl = $3,
    settled_at = now(),
    updated_at = now()
WHERE id = $1
  AND status = 'pending';

-- name: InsertSettledBetOrder :exec
INSERT INTO bet_orders (
    order_no, member_id, lottery_code, lottery_name, lottery_category,
    issue_no, amount, pnl, status, placed_at, settled_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, now(), now()
);
