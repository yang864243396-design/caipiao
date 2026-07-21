-- name: ListCloudBetRecords :many
SELECT
    c.record_no,
    COALESCE(
        NULLIF(TRIM(c.third_party_bet_id), ''),
        (
            SELECT NULLIF(TRIM(bo.third_party_bet_id), '')
            FROM bet_orders bo
            WHERE bo.member_id = c.member_id
              AND NULLIF(TRIM(bo.third_party_bet_id), '') IS NOT NULL
              AND (
                (NULLIF(TRIM(c.bet_order_no), '') IS NOT NULL AND bo.order_no = c.bet_order_no)
                OR (
                    bo.issue_no = c.period_no
                    AND bo.guaji_account_id IS NOT NULL
                    AND bo.placed_at BETWEEN c.placed_at - INTERVAL '10 minutes' AND c.placed_at + INTERVAL '10 minutes'
                )
              )
            ORDER BY
                CASE WHEN bo.order_no = c.bet_order_no THEN 0 ELSE 1 END,
                ABS(EXTRACT(EPOCH FROM (bo.placed_at - c.placed_at)))
            LIMIT 1
        )
    ) AS third_party_bet_id,
    c.scheme_id,
    c.scheme_name,
    c.period_no,
    c.third_party_period,
    c.play_type,
    c.multiplier,
    c.round_label,
    c.amount::float8 AS amount,
    c.pnl::float8 AS pnl,
    c.status,
    c.bet_content,
    c.placed_at
FROM cloud_bet_records c
WHERE c.member_id = $1
  AND c.sim_bet = $2
  AND c.placed_at >= sqlc.arg(since_at)
  AND c.placed_at < sqlc.arg(until_at)
ORDER BY c.placed_at DESC, c.id DESC;

-- name: ListCloudBetRecordsByScheme :many
SELECT
    c.record_no,
    c.scheme_id,
    c.scheme_name,
    c.period_no,
    c.third_party_period,
    c.play_type,
    c.multiplier,
    c.round_label,
    c.amount::float8 AS amount,
    c.pnl::float8 AS pnl,
    c.status,
    c.bet_content,
    c.placed_at
FROM cloud_bet_records c
WHERE c.member_id = $1
  AND c.sim_bet = $2
  AND c.scheme_id = $3
  AND c.placed_at >= sqlc.arg(since_at)
  AND c.placed_at < sqlc.arg(until_at)
ORDER BY c.placed_at DESC, c.id DESC;

-- name: ListCloudBetRecordsByDefinition :many
SELECT
    c.id,
    c.record_no,
    c.third_party_bet_id,
    c.scheme_name,
    si.lottery_label,
    c.period_no,
    c.amount::float8 AS amount,
    c.pnl::float8 AS pnl,
    c.status,
    c.placed_at
FROM cloud_bet_records c
JOIN scheme_instances si ON si.id = c.scheme_id AND si.member_id = c.member_id
WHERE c.member_id = $1
  AND si.definition_id = $2
  AND c.placed_at >= sqlc.arg(since_at)
  AND c.placed_at < sqlc.arg(until_at)
  AND (
    sqlc.narg(order_no)::text IS NULL
    OR sqlc.narg(order_no)::text = ''
    OR c.record_no ILIKE '%' || sqlc.narg(order_no)::text || '%'
    OR c.bet_order_no ILIKE '%' || sqlc.narg(order_no)::text || '%'
    OR c.third_party_bet_id ILIKE '%' || sqlc.narg(order_no)::text || '%'
  )
  AND (
    NOT c.sim_bet
    AND sqlc.narg(guaji_account_id)::bigint IS NOT NULL
    AND c.guaji_account_id = sqlc.narg(guaji_account_id)::bigint
  )
ORDER BY c.placed_at DESC, c.id DESC
LIMIT sqlc.arg(row_limit);

-- name: ListCloudBetRecordsByDefinitionAfterCursor :many
SELECT
    c.id,
    c.record_no,
    c.third_party_bet_id,
    c.scheme_name,
    si.lottery_label,
    c.period_no,
    c.amount::float8 AS amount,
    c.pnl::float8 AS pnl,
    c.status,
    c.placed_at
FROM cloud_bet_records c
JOIN scheme_instances si ON si.id = c.scheme_id AND si.member_id = c.member_id
WHERE c.member_id = $1
  AND si.definition_id = $2
  AND c.placed_at >= sqlc.arg(since_at)
  AND c.placed_at < sqlc.arg(until_at)
  AND (
    sqlc.narg(order_no)::text IS NULL
    OR sqlc.narg(order_no)::text = ''
    OR c.record_no ILIKE '%' || sqlc.narg(order_no)::text || '%'
    OR c.bet_order_no ILIKE '%' || sqlc.narg(order_no)::text || '%'
    OR c.third_party_bet_id ILIKE '%' || sqlc.narg(order_no)::text || '%'
  )
  AND (
    NOT c.sim_bet
    AND sqlc.narg(guaji_account_id)::bigint IS NOT NULL
    AND c.guaji_account_id = sqlc.narg(guaji_account_id)::bigint
  )
  AND (
    c.placed_at < sqlc.arg(cursor_time)
    OR (c.placed_at = sqlc.arg(cursor_time) AND c.id < sqlc.arg(cursor_id))
  )
ORDER BY c.placed_at DESC, c.id DESC
LIMIT sqlc.arg(row_limit);

-- name: ListCloudBetRecordsByLottery :many
SELECT
    c.id,
    c.record_no,
    c.third_party_bet_id,
    c.scheme_name,
    si.lottery_label,
    c.period_no,
    c.amount::float8 AS amount,
    c.pnl::float8 AS pnl,
    c.status,
    c.placed_at
FROM cloud_bet_records c
JOIN scheme_instances si ON si.id = c.scheme_id AND si.member_id = c.member_id
WHERE c.member_id = $1
  AND si.lottery_code = $2
  AND c.placed_at >= sqlc.arg(since_at)
  AND c.placed_at < sqlc.arg(until_at)
  AND (
    sqlc.narg(order_no)::text IS NULL
    OR sqlc.narg(order_no)::text = ''
    OR c.record_no ILIKE '%' || sqlc.narg(order_no)::text || '%'
    OR c.bet_order_no ILIKE '%' || sqlc.narg(order_no)::text || '%'
    OR c.third_party_bet_id ILIKE '%' || sqlc.narg(order_no)::text || '%'
  )
  AND (
    NOT c.sim_bet
    AND sqlc.narg(guaji_account_id)::bigint IS NOT NULL
    AND c.guaji_account_id = sqlc.narg(guaji_account_id)::bigint
  )
ORDER BY c.placed_at DESC, c.id DESC
LIMIT sqlc.arg(row_limit);

-- name: ListCloudBetRecordsByLotteryAfterCursor :many
SELECT
    c.id,
    c.record_no,
    c.third_party_bet_id,
    c.scheme_name,
    si.lottery_label,
    c.period_no,
    c.amount::float8 AS amount,
    c.pnl::float8 AS pnl,
    c.status,
    c.placed_at
FROM cloud_bet_records c
JOIN scheme_instances si ON si.id = c.scheme_id AND si.member_id = c.member_id
WHERE c.member_id = $1
  AND si.lottery_code = $2
  AND c.placed_at >= sqlc.arg(since_at)
  AND c.placed_at < sqlc.arg(until_at)
  AND (
    sqlc.narg(order_no)::text IS NULL
    OR sqlc.narg(order_no)::text = ''
    OR c.record_no ILIKE '%' || sqlc.narg(order_no)::text || '%'
    OR c.bet_order_no ILIKE '%' || sqlc.narg(order_no)::text || '%'
    OR c.third_party_bet_id ILIKE '%' || sqlc.narg(order_no)::text || '%'
  )
  AND (
    NOT c.sim_bet
    AND sqlc.narg(guaji_account_id)::bigint IS NOT NULL
    AND c.guaji_account_id = sqlc.narg(guaji_account_id)::bigint
  )
  AND (
    c.placed_at < sqlc.arg(cursor_time)
    OR (c.placed_at = sqlc.arg(cursor_time) AND c.id < sqlc.arg(cursor_id))
  )
ORDER BY c.placed_at DESC, c.id DESC
LIMIT sqlc.arg(row_limit);

-- name: GetCloudBetRecordCursorAnchor :one
SELECT c.placed_at, c.id
FROM cloud_bet_records c
WHERE c.member_id = $1
  AND c.record_no = $2;

-- name: ListCloudBetRecordsFiltered :many
SELECT
    c.record_no,
    COALESCE(
        NULLIF(TRIM(c.third_party_bet_id), ''),
        (
            SELECT NULLIF(TRIM(bo.third_party_bet_id), '')
            FROM bet_orders bo
            WHERE bo.member_id = c.member_id
              AND NULLIF(TRIM(bo.third_party_bet_id), '') IS NOT NULL
              AND (
                (NULLIF(TRIM(c.bet_order_no), '') IS NOT NULL AND bo.order_no = c.bet_order_no)
                OR (
                    bo.issue_no = c.period_no
                    AND bo.guaji_account_id IS NOT NULL
                    AND bo.placed_at BETWEEN c.placed_at - INTERVAL '10 minutes' AND c.placed_at + INTERVAL '10 minutes'
                )
              )
            ORDER BY
                CASE WHEN bo.order_no = c.bet_order_no THEN 0 ELSE 1 END,
                ABS(EXTRACT(EPOCH FROM (bo.placed_at - c.placed_at)))
            LIMIT 1
        )
    ) AS third_party_bet_id,
    c.scheme_id,
    c.scheme_name,
    si.lottery_code,
    c.period_no,
    c.third_party_period,
    c.play_type,
    c.multiplier,
    c.round_label,
    c.amount::float8 AS amount,
    c.pnl::float8 AS pnl,
    c.status,
    c.bet_content,
    c.placed_at
FROM cloud_bet_records c
JOIN scheme_instances si ON si.id = c.scheme_id AND si.member_id = c.member_id
WHERE c.member_id = $1
  AND c.placed_at >= sqlc.arg(since_at)
  AND c.placed_at < sqlc.arg(until_at)
  AND (
    sqlc.narg(sim_bet)::boolean IS NULL
    OR c.sim_bet = sqlc.narg(sim_bet)::boolean
  )
  AND (
    sqlc.narg(lottery_code)::text IS NULL
    OR sqlc.narg(lottery_code)::text = ''
    OR si.lottery_code = sqlc.narg(lottery_code)::text
  )
  AND (
    sqlc.narg(sim_bet)::boolean IS NOT DISTINCT FROM true
    OR (
      NOT c.sim_bet
      AND sqlc.narg(guaji_account_id)::bigint IS NOT NULL
      AND c.guaji_account_id = sqlc.narg(guaji_account_id)::bigint
      AND NULLIF(TRIM(c.third_party_bet_id), '') IS NOT NULL
    )
  )
ORDER BY c.placed_at DESC, c.id DESC;

-- name: ExistsCloudBetRecordForInstancePeriod :one
SELECT EXISTS(
    SELECT 1 FROM cloud_bet_records
    WHERE scheme_id = $1 AND period_no = $2
)::bool AS exists;

-- name: InsertCloudBetRecord :exec
INSERT INTO cloud_bet_records (
    record_no, member_id, sim_bet, scheme_id, scheme_name,
    period_no, play_type, multiplier, round_label, amount, pnl, status, bet_content,
    guaji_account_id, third_party_bet_id, bet_order_no, placed_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, now()
);

-- name: UpdateCloudBetRecordFromSettlement :execrows
UPDATE cloud_bet_records
SET status = $2,
    pnl = $3
WHERE bet_order_no = $1
  AND status = 'pending';
