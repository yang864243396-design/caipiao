-- name: InsertMemberSchemeFavorite :execrows
INSERT INTO member_scheme_favorites (member_id, snapshot_id)
VALUES ($1, $2)
ON CONFLICT (member_id, snapshot_id) DO NOTHING;

-- name: DeleteMemberSchemeFavorite :execrows
DELETE FROM member_scheme_favorites
WHERE member_id = $1 AND snapshot_id = $2;

-- name: ExistsMemberSchemeFavorite :one
SELECT EXISTS (
    SELECT 1 FROM member_scheme_favorites
    WHERE member_id = $1 AND snapshot_id = $2
);

-- name: ListMemberSchemeFavorites :many
SELECT
    f.snapshot_id, f.created_at,
    s.scheme_name, s.lottery_code, s.lottery_label, s.play_method, s.fund_yuan, s.config
FROM member_scheme_favorites f
JOIN scheme_share_snapshots s ON s.id = f.snapshot_id
WHERE f.member_id = $1
ORDER BY f.created_at DESC;

-- name: UpdateSchemeDefinitionLottery :exec
UPDATE scheme_definitions
SET lottery_code = $3, lottery_label = $4, updated_at = now()
WHERE id = $1 AND member_id = $2;
