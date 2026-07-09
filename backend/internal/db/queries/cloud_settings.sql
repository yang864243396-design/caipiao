-- name: GetMemberCloudSettings :one
SELECT member_id, total_stop_loss, total_take_profit, plan_multiplier, break_period_stop, created_at, updated_at
FROM member_cloud_settings
WHERE member_id = $1;

-- name: UpsertMemberCloudSettings :one
INSERT INTO member_cloud_settings (
    member_id, total_stop_loss, total_take_profit, plan_multiplier, break_period_stop, updated_at
) VALUES ($1, $2, $3, $4, $5, now())
ON CONFLICT (member_id) DO UPDATE SET
    total_stop_loss = EXCLUDED.total_stop_loss,
    total_take_profit = EXCLUDED.total_take_profit,
    plan_multiplier = EXCLUDED.plan_multiplier,
    break_period_stop = EXCLUDED.break_period_stop,
    updated_at = now()
RETURNING member_id, total_stop_loss, total_take_profit, plan_multiplier, break_period_stop, created_at, updated_at;
