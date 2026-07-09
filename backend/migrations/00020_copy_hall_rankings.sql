-- +goose Up
-- +goose StatementBegin
CREATE TABLE copy_hall_rank_slots (
    id           BIGSERIAL PRIMARY KEY,
    lottery_code VARCHAR(32)  NOT NULL,
    board_kind   VARCHAR(16)  NOT NULL,
    rank         INT          NOT NULL,
    scheme_id    VARCHAR(64)  NOT NULL,
    scheme_name  VARCHAR(128) NOT NULL,
    play_method  VARCHAR(64)  NOT NULL,
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ  NOT NULL DEFAULT now(),

    CONSTRAINT uq_copy_hall_rank UNIQUE (lottery_code, board_kind, rank),
    CONSTRAINT chk_copy_hall_board CHECK (board_kind IN ('master', 'contrary')),
    CONSTRAINT chk_copy_hall_rank CHECK (rank >= 1 AND rank <= 10)
);

COMMENT ON TABLE copy_hall_rank_slots IS '跟单大厅榜单 Top10（大神榜/反买榜，按彩种）';
COMMENT ON COLUMN copy_hall_rank_slots.lottery_code IS '彩种 code，如 tencent_ffc';
COMMENT ON COLUMN copy_hall_rank_slots.board_kind IS '榜单类型：master 大神榜 / contrary 反买榜';
COMMENT ON COLUMN copy_hall_rank_slots.rank IS '名次 1–10';
COMMENT ON COLUMN copy_hall_rank_slots.scheme_id IS '方案/快照 ID（跟单入口）';
COMMENT ON COLUMN copy_hall_rank_slots.scheme_name IS '方案展示名';
COMMENT ON COLUMN copy_hall_rank_slots.play_method IS '玩法展示，如定位胆万位';

CREATE INDEX idx_copy_hall_rank_lookup
    ON copy_hall_rank_slots (lottery_code, board_kind, rank);

COMMENT ON INDEX idx_copy_hall_rank_lookup IS '按彩种+榜单查 Top10';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS copy_hall_rank_slots;
-- +goose StatementEnd
