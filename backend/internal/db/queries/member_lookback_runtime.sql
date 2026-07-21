-- name: GetMemberLookbackRuntime :one

SELECT member_id, sim_bet, session_pnl, period_issue, period_pnl, period_hit_count, total_hit_count, updated_at

FROM member_lookback_runtime

WHERE member_id = $1 AND sim_bet = $2;



-- name: UpsertMemberLookbackRuntime :one

INSERT INTO member_lookback_runtime (

    member_id, sim_bet, session_pnl, period_issue, period_pnl, period_hit_count, total_hit_count, updated_at

) VALUES ($1, $2, $3, $4, $5, $6, $7, now())

ON CONFLICT (member_id, sim_bet) DO UPDATE SET

    session_pnl = EXCLUDED.session_pnl,

    period_issue = EXCLUDED.period_issue,

    period_pnl = EXCLUDED.period_pnl,

    period_hit_count = EXCLUDED.period_hit_count,

    total_hit_count = EXCLUDED.total_hit_count,

    updated_at = now()

RETURNING member_id, sim_bet, session_pnl, period_issue, period_pnl, period_hit_count, total_hit_count, updated_at;



-- name: ResetMemberLookbackRuntime :one

UPDATE member_lookback_runtime

SET session_pnl = 0, period_pnl = 0, period_hit_count = 0, total_hit_count = 0, updated_at = now()

WHERE member_id = $1 AND sim_bet = $2

RETURNING member_id, sim_bet, session_pnl, period_issue, period_pnl, period_hit_count, total_hit_count, updated_at;



-- name: ResetSchemeInstanceLookbackRound :one
-- 仅归零倍投轮次/回头盈亏；保留出号游标（定码轮换/高级定码轮换跳局）

UPDATE scheme_instances

SET round_index = 0,

    multiplier = $2,

    lookback_pnl = 0,

    updated_at = now()

WHERE id = $1

  AND status = 'running'

RETURNING

    id, definition_id, member_id, kind, scheme_name, lottery_code, lottery_label,

    status, turnover, pnl, run_time_sec, lookback_pnl, session_pnl, multiplier, countdown_sec, sim_bet,

    round_index, last_settled_issue,

    created_at, updated_at;



-- name: ListRunningSchemeInstanceIDsByMemberSimBet :many

SELECT id FROM scheme_instances

WHERE member_id = $1 AND sim_bet = $2 AND status = 'running';

