-- name: GetSchemeShareSnapshotByID :one
SELECT id, kind, scheme_name, lottery_code, lottery_label, play_method, fund_yuan, config, created_at, updated_at
FROM scheme_share_snapshots
WHERE id = $1;

-- name: ListSchemeDefinitionsByMember :many
SELECT
    d.id,
    d.kind,
    d.scheme_name,
    d.lottery_code,
    d.lottery_label,
    d.share_status,
    d.share_status_locked,
    d.source_snapshot_id,
    d.config,
    d.created_at,
    d.updated_at,
    EXISTS (
        SELECT 1 FROM scheme_instances i WHERE i.definition_id = d.id
    ) AS has_instance
FROM scheme_definitions d
WHERE d.member_id = $1
  AND (
    sqlc.narg(kind)::text IS NULL
    OR sqlc.narg(kind)::text = ''
    OR d.kind = sqlc.narg(kind)::text
  )
ORDER BY d.updated_at DESC;

-- name: ListSchemeDefinitionNamesByMember :many
SELECT scheme_name
FROM scheme_definitions
WHERE member_id = $1;

-- name: SchemeDefinitionNameExistsByMember :one
SELECT EXISTS(
    SELECT 1 FROM scheme_definitions
    WHERE member_id = $1 AND scheme_name = $2
) AS exists;

-- name: GetSchemeDefinitionNameStatusByMember :one
SELECT
    d.id,
    EXISTS (SELECT 1 FROM scheme_instances i WHERE i.definition_id = d.id) AS has_instance
FROM scheme_definitions d
WHERE d.member_id = $1 AND d.scheme_name = $2;

-- name: InsertSchemeDefinition :one
INSERT INTO scheme_definitions (
    id, member_id, kind, scheme_name, lottery_code, lottery_label,
    share_status, share_status_locked, source_snapshot_id, config
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10
)
RETURNING id, member_id, kind, scheme_name, lottery_code, lottery_label,
    share_status, share_status_locked, source_snapshot_id, config, created_at, updated_at;

-- name: GetSchemeDefinitionByID :one
SELECT
    id, kind, scheme_name, lottery_code, lottery_label, share_status, share_status_locked,
    source_snapshot_id, config, created_at, updated_at
FROM scheme_definitions
WHERE id = $1;

-- name: GetSchemeDefinitionByIDAndMember :one
SELECT
    id, kind, scheme_name, lottery_code, lottery_label, share_status, share_status_locked,
    source_snapshot_id, config, created_at, updated_at
FROM scheme_definitions
WHERE id = $1 AND member_id = $2;

-- name: GetSchemeInstanceByDefinitionID :one
SELECT
    id, definition_id, member_id, kind, scheme_name, lottery_code, lottery_label,
    status, turnover, pnl, run_time_sec, lookback_pnl, session_pnl, multiplier, countdown_sec, sim_bet,
    created_at, updated_at
FROM scheme_instances
WHERE definition_id = $1;

-- name: UpdateSchemeDefinitionForCloud :one
UPDATE scheme_definitions
SET
    share_status = $3,
    share_status_locked = true,
    config = $4,
    updated_at = now()
WHERE id = $1 AND member_id = $2
RETURNING id, member_id, kind, scheme_name, lottery_code, lottery_label,
    share_status, share_status_locked, source_snapshot_id, config, created_at, updated_at;

-- name: UpdateSchemeDefinitionConfig :one
UPDATE scheme_definitions
SET config = $3, updated_at = now()
WHERE id = $1 AND member_id = $2
RETURNING id, member_id, kind, scheme_name, lottery_code, lottery_label,
    share_status, share_status_locked, source_snapshot_id, config, created_at, updated_at;

-- name: DeleteSchemeDefinitionByIDAndMember :execrows
DELETE FROM scheme_definitions
WHERE id = $1 AND member_id = $2;

-- name: InsertSchemeInstance :one
INSERT INTO scheme_instances (
    id, definition_id, member_id, kind, scheme_name, lottery_code, lottery_label,
    status, sim_bet
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9
)
RETURNING id, definition_id, member_id, kind, scheme_name, lottery_code, lottery_label,
    status, status_reason, turnover, pnl, run_time_sec, lookback_pnl, session_pnl, multiplier, countdown_sec, sim_bet,
    created_at, updated_at;

-- name: HasRunningSchemeInstanceByDefinition :one
SELECT EXISTS(
    SELECT 1 FROM scheme_instances
    WHERE definition_id = $1 AND status = 'running'
)::bool AS exists;

-- name: SyncSchemeInstancesSimBetByDefinition :execrows
UPDATE scheme_instances
SET sim_bet = $2,
    updated_at = now()
WHERE definition_id = $1
  AND status <> 'running';
