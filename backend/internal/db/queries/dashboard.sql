-- name: AdminDashboardKpi :one
SELECT
    COALESCE((
        SELECT SUM(amount)::float8
        FROM recharge_orders
        WHERE status = 'paid'
          AND (paid_at AT TIME ZONE 'Asia/Shanghai')::date = (CURRENT_TIMESTAMP AT TIME ZONE 'Asia/Shanghai')::date
    ), 0)::float8 AS today_recharge,
    COALESCE((
        SELECT SUM(amount)::float8
        FROM bet_orders
        WHERE status <> 'cancel'
          AND (placed_at AT TIME ZONE 'Asia/Shanghai')::date = (CURRENT_TIMESTAMP AT TIME ZONE 'Asia/Shanghai')::date
    ), 0)::float8 AS today_bet_volume,
    COALESCE((
        SELECT SUM(pnl)::float8
        FROM bet_orders
        WHERE status IN ('win', 'lose')
          AND settled_at IS NOT NULL
    ), 0)::float8 AS member_total_pnl,
    (
        SELECT COUNT(*)::bigint
        FROM scheme_instances
        WHERE status = 'running' AND sim_bet = false
    ) AS running_schemes_real,
    (
        SELECT COUNT(*)::bigint
        FROM scheme_instances
        WHERE status = 'running' AND sim_bet = true
    ) AS running_schemes_sim,
    (
        SELECT COUNT(*)::bigint
        FROM members
        WHERE registered_at >= (CURRENT_TIMESTAMP - INTERVAL '7 days')
    ) AS registrations_last_7_days;
