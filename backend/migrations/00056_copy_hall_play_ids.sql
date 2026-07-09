-- +goose Up
-- +goose StatementBegin
ALTER TABLE copy_hall_rank_slots
    ADD COLUMN play_type_id TEXT NOT NULL DEFAULT 'dingwei',
    ADD COLUMN sub_play_id TEXT NOT NULL DEFAULT '';

COMMENT ON COLUMN copy_hall_rank_slots.play_type_id IS '玩法段 ID：dingwei / hou4 / qian3 / zhong3';
COMMENT ON COLUMN copy_hall_rank_slots.sub_play_id IS '子玩法 ID：zhixuan_fs / zhixuan_ds / zuxuan_fs；定位胆可为空';

UPDATE copy_hall_rank_slots
SET
    play_type_id = CASE
        WHEN play_method LIKE '%组选%' THEN 'zhong3'
        WHEN play_method LIKE '%任选四%' OR play_method LIKE '%后四%' THEN 'hou4'
        WHEN play_method LIKE '%前三%' AND play_method NOT LIKE '定位胆%' THEN 'qian3'
        WHEN play_method LIKE '%中三%' THEN 'zhong3'
        ELSE 'dingwei'
    END,
    sub_play_id = CASE
        WHEN play_method LIKE '%直选复式%' THEN 'zhixuan_fs'
        WHEN play_method LIKE '%直选单式%' THEN 'zhixuan_ds'
        WHEN play_method LIKE '%组选%' THEN 'zuxuan_fs'
        ELSE ''
    END;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE copy_hall_rank_slots
    DROP COLUMN IF EXISTS play_type_id,
    DROP COLUMN IF EXISTS sub_play_id;
-- +goose StatementEnd
