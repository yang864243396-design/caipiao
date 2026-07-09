-- +goose Up
-- +goose StatementBegin
CREATE TABLE chase_orders (
    id               BIGSERIAL PRIMARY KEY,
    chase_no         VARCHAR(64)    NOT NULL,
    member_id        BIGINT         NOT NULL,
    lottery_code     VARCHAR(32)    NOT NULL,
    lottery_name     VARCHAR(64)    NOT NULL,
    lottery_category VARCHAR(16)    NOT NULL,
    total_issues     INT            NOT NULL,
    done_issues      INT            NOT NULL DEFAULT 0,
    amount           NUMERIC(18, 2) NOT NULL,
    status           VARCHAR(16)    NOT NULL DEFAULT 'running',
    started_at       TIMESTAMPTZ    NOT NULL DEFAULT now(),
    finished_at      TIMESTAMPTZ,
    created_at       TIMESTAMPTZ    NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ    NOT NULL DEFAULT now(),

    CONSTRAINT uq_chase_orders_chase_no UNIQUE (chase_no),
    CONSTRAINT fk_chase_orders_member FOREIGN KEY (member_id) REFERENCES members (id) ON DELETE RESTRICT,
    CONSTRAINT chk_chase_orders_amount CHECK (amount > 0),
    CONSTRAINT chk_chase_orders_total_issues CHECK (total_issues > 0),
    CONSTRAINT chk_chase_orders_done_issues CHECK (done_issues >= 0 AND done_issues <= total_issues),
    CONSTRAINT chk_chase_orders_status CHECK (status IN ('running', 'completed', 'cancelled')),
    CONSTRAINT chk_chase_orders_lottery_category CHECK (
        lottery_category IN ('ssc', 'pk10', 'k3', 'x5', 'other')
    )
);

COMMENT ON TABLE chase_orders IS '会员追号订单表：按期数追投记录';
COMMENT ON COLUMN chase_orders.id IS '主键';
COMMENT ON COLUMN chase_orders.chase_no IS '追号单号，全局唯一';
COMMENT ON COLUMN chase_orders.member_id IS '会员 ID，关联 members.id';
COMMENT ON COLUMN chase_orders.lottery_code IS '彩种编码（对内）';
COMMENT ON COLUMN chase_orders.lottery_name IS '彩种展示名称（对外）';
COMMENT ON COLUMN chase_orders.lottery_category IS '彩种大类：ssc、pk10、k3、x5、other';
COMMENT ON COLUMN chase_orders.total_issues IS '追号总期数';
COMMENT ON COLUMN chase_orders.done_issues IS '已完成期数';
COMMENT ON COLUMN chase_orders.amount IS '追号总金额（元，2 位小数）';
COMMENT ON COLUMN chase_orders.status IS '状态：running=追号中，completed=已完成，cancelled=已取消';
COMMENT ON COLUMN chase_orders.started_at IS '追号开始时间（UTC）';
COMMENT ON COLUMN chase_orders.finished_at IS '追号结束时间（UTC）；进行中为 NULL';
COMMENT ON COLUMN chase_orders.created_at IS '记录创建时间（UTC）';
COMMENT ON COLUMN chase_orders.updated_at IS '记录最后更新时间（UTC）';

CREATE INDEX idx_chase_orders_member_started
    ON chase_orders (member_id, started_at DESC);

COMMENT ON INDEX idx_chase_orders_member_started IS '会员追号列表：按开始时间倒序';

CREATE INDEX idx_chase_orders_member_status_started
    ON chase_orders (member_id, status, started_at DESC);

COMMENT ON INDEX idx_chase_orders_member_status_started IS '会员追号按状态筛选';
-- +goose StatementEnd

-- +goose Down
DROP TABLE IF EXISTS chase_orders;
