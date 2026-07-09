-- name: ListSchemeInstancesByMember :many

SELECT

    id, definition_id, member_id, kind, scheme_name, lottery_code, lottery_label,

    status, status_reason, bet_failed_detail, turnover, pnl, run_time_sec, lookback_pnl, session_pnl, multiplier, countdown_sec, sim_bet,

    start_skip_period, start_skip_close_at,

    running_since, created_at, updated_at

FROM scheme_instances

WHERE member_id = $1

ORDER BY updated_at DESC;



-- name: GetSchemeInstanceByIDAndMember :one

SELECT

    id, definition_id, member_id, kind, scheme_name, lottery_code, lottery_label,

    status, status_reason, bet_failed_detail, turnover, pnl, run_time_sec, lookback_pnl, session_pnl, multiplier, countdown_sec, sim_bet,

    running_since, created_at, updated_at

FROM scheme_instances

WHERE id = $1 AND member_id = $2;

-- name: LockSchemeInstanceForBet :one
SELECT id FROM scheme_instances WHERE id = $1 FOR UPDATE;



-- name: UpdateSchemeInstanceMultiplier :one

UPDATE scheme_instances

SET multiplier = $3,

    updated_at = now()

WHERE id = $1 AND member_id = $2

RETURNING

    id, definition_id, member_id, kind, scheme_name, lottery_code, lottery_label,

    status, status_reason, turnover, pnl, run_time_sec, lookback_pnl, session_pnl, multiplier, countdown_sec, sim_bet,

    running_since, created_at, updated_at;



-- name: UpdateSchemeInstanceSimBet :one

UPDATE scheme_instances

SET sim_bet = $3,

    updated_at = now()

WHERE id = $1 AND member_id = $2

  AND status <> 'running'

RETURNING

    id, definition_id, member_id, kind, scheme_name, lottery_code, lottery_label,

    status, status_reason, turnover, pnl, run_time_sec, lookback_pnl, session_pnl, multiplier, countdown_sec, sim_bet,

    running_since, created_at, updated_at;



-- name: UpdateSchemeInstanceStatusToPaused :one

UPDATE scheme_instances

SET status = 'paused',

    status_reason = COALESCE(NULLIF($3, ''), 'manual'),

    last_settled_issue = NULL,

    start_skip_period = NULL,

    start_skip_close_at = NULL,

    run_time_sec = run_time_sec + CASE
        WHEN running_since IS NOT NULL THEN GREATEST(0, EXTRACT(EPOCH FROM (now() - running_since))::int)
        ELSE 0
    END,

    running_since = NULL,

    updated_at = now()

WHERE id = $1 AND member_id = $2 AND status = 'running'

RETURNING

    id, definition_id, member_id, kind, scheme_name, lottery_code, lottery_label,

    status, status_reason, turnover, pnl, run_time_sec, lookback_pnl, session_pnl, multiplier, countdown_sec, sim_bet,

    running_since, created_at, updated_at;



-- name: UpdateSchemeInstanceStatusFromRunningToPending :one

UPDATE scheme_instances

SET status = 'pending',

    status_reason = '',

    last_settled_issue = NULL,

    start_skip_period = NULL,

    start_skip_close_at = NULL,

    run_time_sec = run_time_sec + CASE
        WHEN running_since IS NOT NULL THEN GREATEST(0, EXTRACT(EPOCH FROM (now() - running_since))::int)
        ELSE 0
    END,

    running_since = NULL,

    updated_at = now()

WHERE id = $1 AND member_id = $2 AND status = 'running'

RETURNING

    id, definition_id, member_id, kind, scheme_name, lottery_code, lottery_label,

    status, status_reason, turnover, pnl, run_time_sec, lookback_pnl, session_pnl, multiplier, countdown_sec, sim_bet,

    running_since, created_at, updated_at;



-- name: UpdateSchemeInstanceStatusFromPausedToRunning :one

UPDATE scheme_instances

SET status = 'running',

    status_reason = COALESCE(NULLIF($3, ''), ''),

    running_since = now(),

    updated_at = now()

WHERE id = $1 AND member_id = $2 AND status = 'paused'

RETURNING

    id, definition_id, member_id, kind, scheme_name, lottery_code, lottery_label,

    status, status_reason, turnover, pnl, run_time_sec, lookback_pnl, session_pnl, multiplier, countdown_sec, sim_bet,

    running_since, created_at, updated_at;



-- name: UpdateSchemeInstanceStatus :one

UPDATE scheme_instances

SET status = $3,

    status_reason = CASE
        WHEN $3 = 'paused' THEN COALESCE(NULLIF($4, ''), 'manual')
        WHEN $3 = 'running' THEN COALESCE(NULLIF($4, ''), '')
        ELSE ''
    END,

    updated_at = now()

WHERE id = $1 AND member_id = $2

RETURNING

    id, definition_id, member_id, kind, scheme_name, lottery_code, lottery_label,

    status, status_reason, turnover, pnl, run_time_sec, lookback_pnl, session_pnl, multiplier, countdown_sec, sim_bet,

    running_since, created_at, updated_at;



-- name: UpdateSchemeInstanceStatusFromPendingToRunning :one

UPDATE scheme_instances

SET status = 'running',

    status_reason = COALESCE(NULLIF($3, ''), 'await_next_bet'),

    session_pnl = 0,

    turnover = 0,

    pnl = 0,

    lookback_pnl = 0,

    run_time_sec = 0,

    round_index = 0,

    last_settled_issue = NULL,

    start_skip_period = NULL,

    start_skip_close_at = NULL,

    pick_index = 0,

    current_pick = '',

    last_direction = '',

    bet_failed_detail = NULL,

    running_since = now(),

    updated_at = now()

WHERE id = $1 AND member_id = $2 AND status IN ('pending', 'paused')

RETURNING

    id, definition_id, member_id, kind, scheme_name, lottery_code, lottery_label,

    status, status_reason, turnover, pnl, run_time_sec, lookback_pnl, session_pnl, multiplier, countdown_sec, sim_bet,

    running_since, created_at, updated_at;



-- name: ListRunningSchemeInstances :many

SELECT

    id, definition_id, member_id, kind, scheme_name, lottery_code, lottery_label,

    status, status_reason, turnover, pnl, run_time_sec, lookback_pnl, session_pnl, multiplier, countdown_sec, sim_bet,

    round_index, last_settled_issue, pick_index, current_pick, last_direction,

    start_skip_period, start_skip_close_at,

    created_at, updated_at

FROM scheme_instances

WHERE status = 'running'

ORDER BY updated_at;



-- name: AdvanceSchemeInstanceCountdown :execrows

UPDATE scheme_instances

SET countdown_sec = GREATEST(countdown_sec - $2, 0),

    updated_at = now()

WHERE id = $1

  AND status = 'running';



-- name: PauseSchemeInstanceByWorker :execrows

UPDATE scheme_instances

SET status = 'pending',

    status_reason = $2,

    bet_failed_detail = CASE WHEN $2::varchar = 'bet_failed' THEN NULLIF($3::varchar, '') ELSE NULL END,

    run_time_sec = run_time_sec + CASE
        WHEN running_since IS NOT NULL THEN GREATEST(0, EXTRACT(EPOCH FROM (now() - running_since))::int)
        ELSE 0
    END,

    running_since = NULL,

    updated_at = now()

WHERE id = $1

  AND status = 'running';



-- name: ResumeSchemeInstanceAfterMaintenance :one
UPDATE scheme_instances
SET status = 'running',
    status_reason = 'await_next_bet',
    bet_failed_detail = NULL,
    running_since = now(),
    updated_at = now()
WHERE id = $1
  AND member_id = $2
  AND status = 'pending'
  AND status_reason = 'maintenance'
RETURNING
    id, definition_id, member_id, kind, scheme_name, lottery_code, lottery_label,
    status, status_reason, turnover, pnl, run_time_sec, lookback_pnl, session_pnl, multiplier, countdown_sec, sim_bet,
    running_since, created_at, updated_at;

-- name: ListMaintenanceStoppedInstances :many
SELECT
    id, definition_id, member_id, kind, scheme_name, lottery_code, lottery_label,
    status, status_reason, turnover, pnl, run_time_sec, lookback_pnl, session_pnl, multiplier, countdown_sec, sim_bet,
    round_index, last_settled_issue, pick_index, current_pick, last_direction,
    start_skip_period, start_skip_close_at,
    created_at, updated_at
FROM scheme_instances
WHERE status = 'pending'
  AND status_reason = 'maintenance'
ORDER BY updated_at ASC
LIMIT $1;

-- name: SumMemberFormalSessionPnl :one
SELECT COALESCE(SUM(session_pnl), 0)::numeric AS total
FROM scheme_instances
WHERE member_id = $1
  AND sim_bet = false;



-- name: PauseAllRunningInstancesByMember :many
UPDATE scheme_instances
SET status = 'pending',
    status_reason = $2,
    bet_failed_detail = NULL,
    run_time_sec = run_time_sec + CASE
        WHEN running_since IS NOT NULL THEN GREATEST(0, EXTRACT(EPOCH FROM (now() - running_since))::int)
        ELSE 0
    END,
    running_since = NULL,
    updated_at = now()
WHERE member_id = $1
  AND status = 'running'
RETURNING
    id, definition_id, member_id, kind, scheme_name, lottery_code, lottery_label,
    status, status_reason, bet_failed_detail, turnover, pnl, run_time_sec, lookback_pnl, session_pnl, multiplier, countdown_sec, sim_bet,
    round_index, last_settled_issue, pick_index, current_pick, last_direction,
    running_since, created_at, updated_at;



-- name: PauseRunningPendingInstancesByMember :many

UPDATE scheme_instances

SET status = 'paused',

    status_reason = 'manual',

    updated_at = now()

WHERE member_id = $1

  AND status IN ('running', 'pending')

RETURNING

    id, definition_id, member_id, kind, scheme_name, lottery_code, lottery_label,

    status, status_reason, turnover, pnl, run_time_sec, lookback_pnl, session_pnl, multiplier, countdown_sec, sim_bet,

    created_at, updated_at;



-- name: ApplySchemeInstanceBet :one

UPDATE scheme_instances

SET countdown_sec = $2,

    turnover = turnover + $3,

    pnl = pnl + $4,

    session_pnl = session_pnl + $4,

    lookback_pnl = lookback_pnl + $8,

    multiplier = $5,

    round_index = $6,

    last_settled_issue = $7,

    pick_index = $9,

    current_pick = $10,

    last_direction = $11,

    status_reason = 'cloud_active',

    updated_at = now()

WHERE id = $1

  AND status = 'running'

RETURNING

    id, definition_id, member_id, kind, scheme_name, lottery_code, lottery_label,

    status, status_reason, turnover, pnl, run_time_sec, lookback_pnl, session_pnl, multiplier, countdown_sec, sim_bet,

    round_index, last_settled_issue, pick_index, current_pick, last_direction,

    created_at, updated_at;



-- name: SkipSchemeInstanceStartPeriod :execrows

UPDATE scheme_instances

SET last_settled_issue = $2,

    updated_at = now()

WHERE id = $1

  AND status = 'running'

  AND status_reason = 'await_next_bet'

  AND last_settled_issue IS NULL;



-- name: CountSchemeInstancesByMember :one

SELECT COUNT(*)::bigint AS total

FROM scheme_instances

WHERE member_id = $1

  AND ($2::text IS NULL OR $2::text = '' OR ($2::text = 'real' AND sim_bet = false) OR ($2::text = 'sim' AND sim_bet = true));



-- name: ListSchemeInstancesByMemberPaginated :many

SELECT

    id, definition_id, member_id, kind, scheme_name, lottery_code, lottery_label,

    status, status_reason, bet_failed_detail, turnover, pnl, run_time_sec, lookback_pnl, session_pnl, multiplier, countdown_sec, sim_bet,

    running_since, created_at, updated_at

FROM scheme_instances

WHERE member_id = $1

  AND ($2::text IS NULL OR $2::text = '' OR ($2::text = 'real' AND sim_bet = false) OR ($2::text = 'sim' AND sim_bet = true))

  AND (

    $3::timestamptz IS NULL

    OR updated_at < $3

    OR (updated_at = $3 AND id < $4::text)

  )

ORDER BY updated_at DESC, id DESC

LIMIT $5;



-- name: ListSchemeInstancesByMemberIDs :many

SELECT

    id, definition_id, member_id, kind, scheme_name, lottery_code, lottery_label,

    status, status_reason, bet_failed_detail, turnover, pnl, run_time_sec, lookback_pnl, session_pnl, multiplier, countdown_sec, sim_bet,

    running_since, created_at, updated_at

FROM scheme_instances

WHERE member_id = $1

  AND id = ANY($2::text[]);

