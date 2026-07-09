-- name: ListLotteryDraws :many
SELECT id, lottery_code, issue_no, period_short, balls, sum_value, drawn_at
FROM lottery_draws
WHERE lottery_code = $1
ORDER BY drawn_at DESC, id DESC
LIMIT sqlc.arg(row_limit);

-- name: ListLotteryDrawsAfterCursor :many
SELECT id, lottery_code, issue_no, period_short, balls, sum_value, drawn_at
FROM lottery_draws
WHERE lottery_code = $1
  AND (
    drawn_at < sqlc.arg(cursor_time)
    OR (drawn_at = sqlc.arg(cursor_time) AND id < sqlc.arg(cursor_id))
  )
ORDER BY drawn_at DESC, id DESC
LIMIT sqlc.arg(row_limit);

-- name: GetLatestLotteryDraw :one
SELECT id, lottery_code, issue_no, period_short, balls, sum_value, drawn_at
FROM lottery_draws
WHERE lottery_code = $1
ORDER BY drawn_at DESC, id DESC
LIMIT 1;

-- name: GetLotteryDrawByIssue :one
SELECT id, lottery_code, issue_no, period_short, balls, sum_value, drawn_at
FROM lottery_draws
WHERE lottery_code = $1 AND issue_no = $2;

-- name: GetFirstLotteryDrawAfterIssue :one
SELECT id, lottery_code, issue_no, period_short, balls, sum_value, drawn_at
FROM lottery_draws
WHERE lottery_code = $1
  AND issue_no > $2
ORDER BY issue_no ASC
LIMIT 1;

-- name: InsertLotteryDraw :one
INSERT INTO lottery_draws (lottery_code, issue_no, period_short, balls, sum_value, drawn_at)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (lottery_code, issue_no) DO NOTHING
RETURNING id, lottery_code, issue_no, period_short, balls, sum_value, drawn_at;

-- name: GetLotteryDrawByID :one
SELECT id, lottery_code, issue_no, period_short, balls, sum_value, drawn_at
FROM lottery_draws
WHERE id = $1 AND lottery_code = $2;

