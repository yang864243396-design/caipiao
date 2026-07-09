-- name: ListAdminSchemeInstances :many
SELECT
    i.id,
    i.definition_id,
    m.id AS member_id,
    m.account,
    i.kind,
    i.scheme_name,
    i.lottery_code,
    i.lottery_label,
    i.status,
    i.status_reason,
    i.sim_bet,
    COALESCE(d.config->>'runTypeId', '') AS run_type,
    COALESCE(NULLIF(d.config->>'playTypeId', ''), NULLIF(d.config->>'typeId', ''), '') AS play_type_id,
    COALESCE(pt.label, '') AS play_type_label,
    i.created_at,
    i.updated_at
FROM scheme_instances i
INNER JOIN members m ON m.id = i.member_id
LEFT JOIN scheme_definitions d ON d.id = i.definition_id
LEFT JOIN lottery_catalog lc ON lc.code = i.lottery_code
LEFT JOIN play_types pt ON pt.template_code = COALESCE(NULLIF(d.config->>'playTemplate', ''), lc.play_template)
    AND pt.type_id = COALESCE(NULLIF(d.config->>'playTypeId', ''), NULLIF(d.config->>'typeId', ''))
    AND COALESCE(NULLIF(d.config->>'playTypeId', ''), NULLIF(d.config->>'typeId', '')) <> ''
WHERE (
    sqlc.narg(keyword)::text IS NULL
    OR sqlc.narg(keyword)::text = ''
    OR (
        COALESCE(NULLIF(sqlc.narg(search_field)::text, ''), 'account') = 'account'
        AND m.account ILIKE '%' || sqlc.narg(keyword)::text || '%'
    )
    OR (
        sqlc.narg(search_field)::text = 'schemeName'
        AND i.scheme_name ILIKE '%' || sqlc.narg(keyword)::text || '%'
    )
)
AND (
    sqlc.narg(kind)::text IS NULL
    OR sqlc.narg(kind)::text = ''
    OR i.kind = sqlc.narg(kind)::text
)
AND (
    sqlc.narg(status)::text IS NULL
    OR sqlc.narg(status)::text = ''
    OR i.status = sqlc.narg(status)::text
)
AND (
    sqlc.narg(sim_bet)::boolean IS NULL
    OR i.sim_bet = sqlc.narg(sim_bet)::boolean
)
AND (
    sqlc.narg(lottery_code)::text IS NULL
    OR sqlc.narg(lottery_code)::text = ''
    OR i.lottery_code = sqlc.narg(lottery_code)::text
)
ORDER BY i.updated_at DESC
LIMIT sqlc.arg(row_limit);

-- name: GetSchemeInstanceByID :one
SELECT
    id, definition_id, member_id, kind, scheme_name, lottery_code, lottery_label,
    status, status_reason, turnover, pnl, run_time_sec, lookback_pnl, session_pnl, multiplier, countdown_sec, sim_bet,
    created_at, updated_at
FROM scheme_instances
WHERE id = $1;

-- name: UpdateSchemeInstanceStatusByAdmin :one
UPDATE scheme_instances
SET status = $2,
    status_reason = CASE WHEN $2 = 'paused' THEN 'manual' WHEN $2 = 'soft_stopped' THEN '' ELSE status_reason END,
    updated_at = now()
WHERE id = $1
RETURNING
    id, definition_id, member_id, kind, scheme_name, lottery_code, lottery_label,
    status, status_reason, turnover, pnl, run_time_sec, lookback_pnl, session_pnl, multiplier, countdown_sec, sim_bet,
    created_at, updated_at;

-- name: ListAdminSchemeShareSnapshots :many
SELECT id, kind, scheme_name, lottery_code, lottery_label, play_method, fund_yuan, config, created_at, updated_at
FROM scheme_share_snapshots
WHERE (
    sqlc.narg(keyword)::text IS NULL
    OR sqlc.narg(keyword)::text = ''
    OR (
        COALESCE(NULLIF(sqlc.narg(search_field)::text, ''), 'schemeName') = 'schemeName'
        AND scheme_name ILIKE '%' || sqlc.narg(keyword)::text || '%'
    )
    OR (
        sqlc.narg(search_field)::text = 'snapshotId'
        AND id ILIKE '%' || sqlc.narg(keyword)::text || '%'
    )
)
AND (
    sqlc.narg(lottery_code)::text IS NULL
    OR sqlc.narg(lottery_code)::text = ''
    OR lottery_code = sqlc.narg(lottery_code)::text
)
ORDER BY updated_at DESC
LIMIT sqlc.arg(row_limit);

-- name: UpdateSchemeShareSnapshotAdmin :one
UPDATE scheme_share_snapshots
SET
    scheme_name = $2,
    lottery_code = $3,
    lottery_label = $4,
    play_method = $5,
    fund_yuan = $6,
    config = $7,
    updated_at = now()
WHERE id = $1
RETURNING id, kind, scheme_name, lottery_code, lottery_label, play_method, fund_yuan, config, created_at, updated_at;

-- name: DeleteSchemeShareSnapshot :execrows
DELETE FROM scheme_share_snapshots
WHERE id = $1;
