-- +goose Up
-- +goose StatementBegin
-- 跟单大厅方案收藏（内置计画前置，docs/run-types-implementation-plan.md §3.6 / P6）
CREATE TABLE member_scheme_favorites (
    id          BIGSERIAL    PRIMARY KEY,
    member_id   BIGINT       NOT NULL REFERENCES members(id) ON DELETE CASCADE,
    snapshot_id VARCHAR(32)  NOT NULL,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT now(),

    CONSTRAINT uq_member_scheme_favorites UNIQUE (member_id, snapshot_id)
);

COMMENT ON TABLE member_scheme_favorites IS '会员收藏的跟单大厅方案快照（内置计画选择来源）';
COMMENT ON COLUMN member_scheme_favorites.snapshot_id IS '关联 scheme_share_snapshots.id';

CREATE INDEX idx_member_scheme_favorites_member
    ON member_scheme_favorites (member_id, created_at DESC);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS member_scheme_favorites;
-- +goose StatementEnd
