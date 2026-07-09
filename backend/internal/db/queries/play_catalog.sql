-- name: CountSubPlays :one
SELECT COUNT(*)::int AS count FROM sub_plays;

-- name: ListPlayTypesByTemplate :many
SELECT template_code, type_id, label, sort_order, panel_type, enabled
FROM play_types
WHERE template_code = $1
ORDER BY sort_order ASC, type_id ASC;

-- name: ListSubPlaysByTemplate :many
SELECT
    template_code,
    type_id,
    sub_id,
    label,
    sort_order,
    bet_mode,
    segment_rule,
    outbound_play_code,
    enabled
FROM sub_plays
WHERE template_code = $1
ORDER BY type_id ASC, sort_order ASC, sub_id ASC;

-- name: GetSubPlay :one
SELECT
    template_code,
    type_id,
    sub_id,
    label,
    sort_order,
    bet_mode,
    segment_rule,
    outbound_play_code,
    enabled
FROM sub_plays
WHERE template_code = $1
  AND type_id = $2
  AND sub_id = $3;

-- name: ListPlayTemplates :many
SELECT code, label, version
FROM play_templates
ORDER BY code ASC;

-- name: GetLotteryCatalogPurgeState :one
SELECT id, completed_at, note
FROM lottery_catalog_purge_state
WHERE id = 1;

-- name: InsertLotteryCatalogPurgeState :exec
INSERT INTO lottery_catalog_purge_state (id, note)
VALUES (1, $1)
ON CONFLICT (id) DO NOTHING;
