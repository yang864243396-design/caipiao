-- name: InsertSchemeShareSnapshot :one
INSERT INTO scheme_share_snapshots (
    id, kind, scheme_name, lottery_code, lottery_label, play_method, fund_yuan, config
) VALUES (
    $1, 'custom', $2, $3, $4, $5, $6, $7
)
RETURNING id, kind, scheme_name, lottery_code, lottery_label, play_method, fund_yuan, config, created_at, updated_at;

-- name: ListSchemeShareSnapshots :many
SELECT id, kind, scheme_name, lottery_code, lottery_label, play_method, fund_yuan, config, created_at, updated_at
FROM scheme_share_snapshots
WHERE (
    sqlc.narg(keyword)::text IS NULL
    OR sqlc.narg(keyword)::text = ''
    OR id ILIKE '%' || sqlc.narg(keyword)::text || '%'
    OR scheme_name ILIKE '%' || sqlc.narg(keyword)::text || '%'
)
AND (
    sqlc.narg(cursor)::text IS NULL
    OR sqlc.narg(cursor)::text = ''
    OR id > sqlc.narg(cursor)::text
)
ORDER BY id ASC
LIMIT sqlc.arg(row_limit);
