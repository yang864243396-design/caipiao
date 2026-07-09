-- name: GetMemberLookbackSettings :one
SELECT
    member_id, run_mode, apply_formal, apply_sim, judgment,
    single_profit_threshold, single_loss_threshold,
    overall_profit_threshold, overall_loss_threshold,
    scheme_wins_min, scheme_wins_max,
    period_profit, period_loss,
    created_at, updated_at
FROM member_lookback_settings
WHERE member_id = $1;

-- name: UpsertMemberLookbackSettings :one
INSERT INTO member_lookback_settings (
    member_id, run_mode, apply_formal, apply_sim, judgment,
    single_profit_threshold, single_loss_threshold,
    overall_profit_threshold, overall_loss_threshold,
    scheme_wins_min, scheme_wins_max,
    period_profit, period_loss, updated_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, now()
)
ON CONFLICT (member_id) DO UPDATE SET
    run_mode = EXCLUDED.run_mode,
    apply_formal = EXCLUDED.apply_formal,
    apply_sim = EXCLUDED.apply_sim,
    judgment = EXCLUDED.judgment,
    single_profit_threshold = EXCLUDED.single_profit_threshold,
    single_loss_threshold = EXCLUDED.single_loss_threshold,
    overall_profit_threshold = EXCLUDED.overall_profit_threshold,
    overall_loss_threshold = EXCLUDED.overall_loss_threshold,
    scheme_wins_min = EXCLUDED.scheme_wins_min,
    scheme_wins_max = EXCLUDED.scheme_wins_max,
    period_profit = EXCLUDED.period_profit,
    period_loss = EXCLUDED.period_loss,
    updated_at = now()
RETURNING
    member_id, run_mode, apply_formal, apply_sim, judgment,
    single_profit_threshold, single_loss_threshold,
    overall_profit_threshold, overall_loss_threshold,
    scheme_wins_min, scheme_wins_max,
    period_profit, period_loss,
    created_at, updated_at;
