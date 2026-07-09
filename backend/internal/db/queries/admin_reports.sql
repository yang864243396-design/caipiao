-- name: AdminLotteryStatSummary :one
SELECT
    COALESCE(SUM(b.amount) FILTER (WHERE b.status IN ('win', 'lose')), 0)::float8 AS effective_bet,
    COALESCE(SUM(b.amount + b.pnl) FILTER (WHERE b.status = 'win'), 0)::float8 AS payout
FROM bet_orders b
WHERE b.placed_at >= $1
  AND b.placed_at < $2;

-- name: AdminLotteryStatByLottery :many
SELECT
    b.lottery_name,
    COUNT(*)::bigint AS bet_count,
    COALESCE(SUM(b.amount) FILTER (WHERE b.status IN ('win', 'lose')), 0)::float8 AS effective_bet,
    COALESCE(SUM(b.pnl) FILTER (WHERE b.status IN ('win', 'lose')), 0)::float8 AS member_pnl
FROM bet_orders b
WHERE b.placed_at >= $1
  AND b.placed_at < $2
GROUP BY b.lottery_code, b.lottery_name
ORDER BY effective_bet DESC;

-- name: AdminPnlReportSummary :one
SELECT
    COALESCE(-SUM(b.pnl) FILTER (WHERE b.status IN ('win', 'lose')), 0)::float8 AS platform_pnl,
    COALESCE(SUM(b.amount) FILTER (WHERE b.status IN ('win', 'lose')), 0)::float8 AS valid_bet
FROM bet_orders b
WHERE b.placed_at >= $1
  AND b.placed_at < $2;

-- name: AdminPnlReportDaily :many
SELECT
    (b.placed_at AT TIME ZONE 'Asia/Shanghai')::date AS stat_date,
    COALESCE(SUM(b.amount) FILTER (WHERE b.status IN ('win', 'lose')), 0)::float8 AS valid_bet,
    COALESCE(-SUM(b.pnl) FILTER (WHERE b.status IN ('win', 'lose')), 0)::float8 AS platform_pnl
FROM bet_orders b
WHERE b.placed_at >= $1
  AND b.placed_at < $2
GROUP BY stat_date
ORDER BY stat_date DESC;

-- name: AdminDailyLotteryReport :many
SELECT
    (b.placed_at AT TIME ZONE 'Asia/Shanghai')::date AS stat_date,
    b.lottery_code,
    b.lottery_name,
    COUNT(*)::bigint AS bet_count,
    COALESCE(SUM(b.amount) FILTER (WHERE b.status IN ('win', 'lose')), 0)::float8 AS valid_bet,
    COALESCE(-SUM(b.pnl) FILTER (WHERE b.status IN ('win', 'lose')), 0)::float8 AS platform_pnl
FROM bet_orders b
WHERE b.placed_at >= $1
  AND b.placed_at < $2
  AND ($3::text = '' OR b.lottery_code = $3)
GROUP BY stat_date, b.lottery_code, b.lottery_name
ORDER BY stat_date DESC, valid_bet DESC;

-- name: AdminDailyLotterySummary :one
SELECT
    COUNT(*)::bigint AS bet_count,
    COALESCE(SUM(b.amount) FILTER (WHERE b.status IN ('win', 'lose')), 0)::float8 AS valid_bet,
    COALESCE(-SUM(b.pnl) FILTER (WHERE b.status IN ('win', 'lose')), 0)::float8 AS platform_pnl
FROM bet_orders b
WHERE b.placed_at >= $1
  AND b.placed_at < $2
  AND ($3::text = '' OR b.lottery_code = $3);
