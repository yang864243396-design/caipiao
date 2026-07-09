-- +goose Up
-- +goose StatementBegin
-- 跟单大厅改为全站共用榜单：每个 board_kind + rank 仅保留一行
WITH ranked AS (
    SELECT id,
           ROW_NUMBER() OVER (
               PARTITION BY board_kind, rank
               ORDER BY CASE WHEN lottery_code = 'tron_ffc_1m' THEN 0 ELSE 1 END,
                        lottery_code ASC
           ) AS rn
    FROM copy_hall_rank_slots
)
DELETE FROM copy_hall_rank_slots
WHERE id NOT IN (SELECT id FROM ranked WHERE rn = 1);

ALTER TABLE copy_hall_rank_slots DROP CONSTRAINT uq_copy_hall_rank;
ALTER TABLE copy_hall_rank_slots
    ADD CONSTRAINT uq_copy_hall_rank_global UNIQUE (board_kind, rank);

DROP INDEX IF EXISTS idx_copy_hall_rank_lookup;
CREATE INDEX idx_copy_hall_board_rank ON copy_hall_rank_slots (board_kind, rank);

COMMENT ON TABLE copy_hall_rank_slots IS '跟单大厅榜单 Top10（大神榜/反买榜，全站共用）';
COMMENT ON COLUMN copy_hall_rank_slots.lottery_code IS '上榜方案所属彩种 code';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_copy_hall_board_rank;
CREATE INDEX idx_copy_hall_rank_lookup ON copy_hall_rank_slots (lottery_code, board_kind, rank);

ALTER TABLE copy_hall_rank_slots DROP CONSTRAINT uq_copy_hall_rank_global;
ALTER TABLE copy_hall_rank_slots
    ADD CONSTRAINT uq_copy_hall_rank UNIQUE (lottery_code, board_kind, rank);

COMMENT ON TABLE copy_hall_rank_slots IS '跟单大厅榜单 Top10（大神榜/反买榜，按彩种）';
COMMENT ON COLUMN copy_hall_rank_slots.lottery_code IS '彩种 code，如 tencent_ffc';
-- +goose StatementEnd
