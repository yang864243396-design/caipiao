-- name: ListChaseOrders :many
SELECT
    c.id,
    c.chase_no,
    c.lottery_name,
    c.lottery_category,
    c.total_issues,
    c.done_issues,
    c.amount::float8 AS amount,
    c.status,
    c.started_at
FROM chase_orders c
WHERE c.member_id = $1
  AND c.started_at >= sqlc.arg(time_from)
  AND c.started_at < sqlc.arg(time_to)
  AND (
    sqlc.narg(lottery_category)::text IS NULL
    OR c.lottery_category = sqlc.narg(lottery_category)::text
  )
ORDER BY c.started_at DESC, c.id DESC
LIMIT sqlc.arg(row_limit);

-- name: ListChaseOrdersAfterCursor :many
SELECT
    c.id,
    c.chase_no,
    c.lottery_name,
    c.lottery_category,
    c.total_issues,
    c.done_issues,
    c.amount::float8 AS amount,
    c.status,
    c.started_at
FROM chase_orders c
WHERE c.member_id = $1
  AND c.started_at >= sqlc.arg(time_from)
  AND c.started_at < sqlc.arg(time_to)
  AND (
    sqlc.narg(lottery_category)::text IS NULL
    OR c.lottery_category = sqlc.narg(lottery_category)::text
  )
  AND (
    c.started_at < sqlc.arg(cursor_time)
    OR (c.started_at = sqlc.arg(cursor_time) AND c.id < sqlc.arg(cursor_id))
  )
ORDER BY c.started_at DESC, c.id DESC
LIMIT sqlc.arg(row_limit);

-- name: GetChaseOrderCursorAnchor :one
SELECT c.started_at, c.id
FROM chase_orders c
WHERE c.member_id = $1
  AND c.chase_no = $2;
