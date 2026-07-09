-- +goose Up
-- +goose StatementBegin
-- 跟单大厅 scheme_id 对齐分享池快照，便于 follow-bet / add-to-cloud 联调
UPDATE copy_hall_rank_slots
SET scheme_id = CASE rank
    WHEN 1 THEN 'SD10001'
    WHEN 2 THEN 'SD10002'
    WHEN 3 THEN 'SD10003'
    WHEN 4 THEN 'SD10004'
    WHEN 5 THEN 'SD10005'
    WHEN 6 THEN 'SD10006'
    WHEN 7 THEN 'SD10007'
    WHEN 8 THEN 'SD10005'
    WHEN 9 THEN 'SD10006'
    WHEN 10 THEN 'SD10007'
    ELSE scheme_id
END
WHERE board_kind = 'master';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
UPDATE copy_hall_rank_slots
SET scheme_id = CASE
    WHEN rank = 1 AND board_kind = 'master' THEN 'copy_demo_3001'
    WHEN board_kind = 'master' THEN 'copy_demo_' || (3000 + rank)::text
    WHEN rank = 1 AND board_kind = 'contrary' THEN 'copy_contrary_3001'
    ELSE 'copy_contrary_' || (3000 + rank)::text
END;
-- +goose StatementEnd
