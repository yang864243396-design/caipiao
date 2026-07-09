-- +goose Up
-- +goose StatementBegin
CREATE TABLE cloud_bet_records (
    id            BIGSERIAL PRIMARY KEY,
    record_no     VARCHAR(32)    NOT NULL,
    member_id     BIGINT         NOT NULL,
    run_mode      VARCHAR(8)     NOT NULL DEFAULT 'real',
    scheme_id     VARCHAR(64)    NOT NULL,
    scheme_name   VARCHAR(128)   NOT NULL,
    period_no     VARCHAR(32)    NOT NULL,
    play_type     VARCHAR(64)    NOT NULL,
    multiplier    VARCHAR(16)    NOT NULL,
    round_label   VARCHAR(16)    NOT NULL,
    amount        NUMERIC(18, 2) NOT NULL,
    pnl           NUMERIC(18, 2) NOT NULL DEFAULT 0,
    status        VARCHAR(8)     NOT NULL,
    placed_at     TIMESTAMPTZ    NOT NULL DEFAULT now(),
    created_at    TIMESTAMPTZ    NOT NULL DEFAULT now(),

    CONSTRAINT uq_cloud_bet_records_record_no UNIQUE (record_no),
    CONSTRAINT fk_cloud_bet_records_member FOREIGN KEY (member_id) REFERENCES members (id) ON DELETE RESTRICT,
    CONSTRAINT chk_cloud_bet_records_amount CHECK (amount > 0),
    CONSTRAINT chk_cloud_bet_records_mode CHECK (run_mode IN ('real', 'sim')),
    CONSTRAINT chk_cloud_bet_records_status CHECK (status IN ('hit', 'miss'))
);

COMMENT ON TABLE cloud_bet_records IS '云端方案投注明细：按会员+方案聚合展示（BetRecordsView / 云端近 N 日）';
COMMENT ON COLUMN cloud_bet_records.id IS '主键';
COMMENT ON COLUMN cloud_bet_records.record_no IS '明细编号，全局唯一';
COMMENT ON COLUMN cloud_bet_records.member_id IS '会员 ID，关联 members.id';
COMMENT ON COLUMN cloud_bet_records.run_mode IS '运行模式：real=真实，sim=模拟';
COMMENT ON COLUMN cloud_bet_records.scheme_id IS '云端方案 ID（实例/方案标识）';
COMMENT ON COLUMN cloud_bet_records.scheme_name IS '方案展示名称';
COMMENT ON COLUMN cloud_bet_records.period_no IS '期号';
COMMENT ON COLUMN cloud_bet_records.play_type IS '玩法类型（如 万位定位）';
COMMENT ON COLUMN cloud_bet_records.multiplier IS '倍数展示文本';
COMMENT ON COLUMN cloud_bet_records.round_label IS '轮次展示（如 1/3）';
COMMENT ON COLUMN cloud_bet_records.amount IS '投注金额（元，2 位小数）';
COMMENT ON COLUMN cloud_bet_records.pnl IS '盈亏（元，2 位小数；负数为亏损）';
COMMENT ON COLUMN cloud_bet_records.status IS '结果：hit=中，miss=未中';
COMMENT ON COLUMN cloud_bet_records.placed_at IS '投注时间（UTC）';
COMMENT ON COLUMN cloud_bet_records.created_at IS '记录创建时间（UTC）';

CREATE INDEX idx_cloud_bet_records_member_mode_placed
    ON cloud_bet_records (member_id, run_mode, placed_at DESC);

COMMENT ON INDEX idx_cloud_bet_records_member_mode_placed IS '会员云端投注列表';

CREATE INDEX idx_cloud_bet_records_member_scheme_placed
    ON cloud_bet_records (member_id, scheme_id, run_mode, placed_at DESC);

COMMENT ON INDEX idx_cloud_bet_records_member_scheme_placed IS '单方案投注明细分页';
-- +goose StatementEnd

-- +goose Down
DROP TABLE IF EXISTS cloud_bet_records;
