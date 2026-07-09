-- name: ListCopyHallRankSlots :many
SELECT lottery_code, rank, scheme_id, scheme_name, play_method, play_type_id, sub_play_id
FROM copy_hall_rank_slots
WHERE board_kind = $1
ORDER BY rank ASC;

-- name: ListAllCopyHallRankSlots :many
SELECT lottery_code, board_kind, rank, scheme_id, scheme_name, play_method, play_type_id, sub_play_id
FROM copy_hall_rank_slots
ORDER BY board_kind ASC, rank ASC;

-- name: UpsertCopyHallRankSlot :exec
INSERT INTO copy_hall_rank_slots (lottery_code, board_kind, rank, scheme_id, scheme_name, play_method, play_type_id, sub_play_id)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
ON CONFLICT (board_kind, rank)
DO UPDATE SET
    lottery_code = EXCLUDED.lottery_code,
    scheme_id = EXCLUDED.scheme_id,
    scheme_name = EXCLUDED.scheme_name,
    play_method = EXCLUDED.play_method,
    play_type_id = EXCLUDED.play_type_id,
    sub_play_id = EXCLUDED.sub_play_id,
    updated_at = now();

-- name: DeleteCopyHallBoardSlots :exec
DELETE FROM copy_hall_rank_slots
WHERE board_kind = $1;

-- name: DeleteAllCopyHallRankSlots :exec
DELETE FROM copy_hall_rank_slots;
