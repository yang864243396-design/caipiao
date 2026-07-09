-- +goose Up
-- +goose StatementBegin
CREATE TABLE recharge_orders (
    id              BIGSERIAL PRIMARY KEY,
    order_no        VARCHAR(64)    NOT NULL,
    member_id       BIGINT         NOT NULL,
    amount          NUMERIC(18, 2) NOT NULL,
    channel         VARCHAR(32)    NOT NULL,
    status          VARCHAR(16)    NOT NULL DEFAULT 'pending',
    paid_at         TIMESTAMPTZ,
    created_at      TIMESTAMPTZ    NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ    NOT NULL DEFAULT now(),

    CONSTRAINT uq_recharge_orders_order_no UNIQUE (order_no),
    CONSTRAINT fk_recharge_orders_member FOREIGN KEY (member_id) REFERENCES members (id) ON DELETE RESTRICT,
    CONSTRAINT chk_recharge_orders_amount CHECK (amount > 0),
    CONSTRAINT chk_recharge_orders_status CHECK (status IN ('pending', 'paid', 'cancelled', 'failed'))
);

COMMENT ON TABLE recharge_orders IS '会员充值订单表';
COMMENT ON COLUMN recharge_orders.id IS '主键';
COMMENT ON COLUMN recharge_orders.order_no IS '充值订单号，全局唯一';
COMMENT ON COLUMN recharge_orders.member_id IS '会员 ID，关联 members.id';
COMMENT ON COLUMN recharge_orders.amount IS '充值金额（元，2 位小数），必须大于 0';
COMMENT ON COLUMN recharge_orders.channel IS '充值渠道编码（对接支付通道）';
COMMENT ON COLUMN recharge_orders.status IS '订单状态：pending=待支付，paid=已到账，cancelled=已取消，failed=失败';
COMMENT ON COLUMN recharge_orders.paid_at IS '到账时间（UTC）；未到账为 NULL';
COMMENT ON COLUMN recharge_orders.created_at IS '下单时间（UTC）';
COMMENT ON COLUMN recharge_orders.updated_at IS '最后更新时间（UTC）';

CREATE INDEX idx_recharge_orders_member_created
    ON recharge_orders (member_id, created_at DESC);

COMMENT ON INDEX idx_recharge_orders_member_created IS '会员充值记录列表';

CREATE INDEX idx_recharge_orders_status_created
    ON recharge_orders (status, created_at DESC);

COMMENT ON INDEX idx_recharge_orders_status_created IS '管理端按状态筛选充值订单';
-- +goose StatementEnd

-- +goose Down
DROP TABLE IF EXISTS recharge_orders;
