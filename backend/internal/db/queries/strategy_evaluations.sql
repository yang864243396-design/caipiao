-- name: TryClaimSchemeStrategyEvaluation :one
WITH inserted AS (
    INSERT INTO scheme_strategy_evaluations (instance_id, lottery_code, period_no)
    VALUES (sqlc.arg(instance_id), sqlc.arg(lottery_code), sqlc.arg(period_no))
    ON CONFLICT (instance_id, period_no) DO NOTHING
    RETURNING 1
)
SELECT EXISTS(SELECT 1 FROM inserted)::bool AS claimed;

-- name: GetSchemeStrategyEvaluation :one
SELECT id,
       instance_id,
       lottery_code,
       period_no,
       cloud_bet_record_id,
       bet_order_no,
       status,
       rule_version,
       rule_snapshot_hash,
       local_hit,
       winning_units,
       diagnostics,
       claimed_at,
       completed_at,
       created_at,
       updated_at
FROM scheme_strategy_evaluations
WHERE instance_id = sqlc.arg(instance_id)
  AND period_no = sqlc.arg(period_no);

-- name: MarkSchemeStrategyEvaluationProcessing :execrows
UPDATE scheme_strategy_evaluations
SET status = 'processing', claimed_at = now(), updated_at = now()
WHERE id = sqlc.arg(id)
  AND status = 'pending';

-- name: CompleteSchemeStrategyEvaluation :execrows
UPDATE scheme_strategy_evaluations
SET status = 'completed',
    cloud_bet_record_id = sqlc.narg(cloud_bet_record_id),
    bet_order_no = sqlc.narg(bet_order_no),
    rule_version = sqlc.narg(rule_version),
    rule_snapshot_hash = sqlc.narg(rule_snapshot_hash),
    local_hit = sqlc.narg(local_hit),
    winning_units = sqlc.narg(winning_units),
    diagnostics = sqlc.arg(diagnostics),
    completed_at = now(),
    updated_at = now()
WHERE id = sqlc.arg(id)
  AND status = 'processing';

-- name: SkipSchemeStrategyEvaluation :execrows
UPDATE scheme_strategy_evaluations
SET status = 'skipped', diagnostics = sqlc.arg(diagnostics), completed_at = now(), updated_at = now()
WHERE id = sqlc.arg(id)
  AND status IN ('pending', 'processing');

-- name: ListRecoverableSchemeStrategyEvaluations :many
SELECT id,
       instance_id,
       lottery_code,
       period_no,
       status,
       diagnostics,
       created_at
FROM scheme_strategy_evaluations
WHERE status IN ('pending', 'processing')
ORDER BY created_at ASC
LIMIT sqlc.arg(row_limit)
FOR UPDATE SKIP LOCKED;
