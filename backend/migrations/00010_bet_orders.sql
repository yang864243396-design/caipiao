-- +goose Up
-- +goose StatementBegin
CREATE TABLE bet_orders (
    id               BIGSERIAL PRIMARY KEY,
    order_no         VARCHAR(64)    NOT NULL,
    member_id        BIGINT         NOT NULL,
    lottery_code     VARCHAR(32)    NOT NULL,
    lottery_name     VARCHAR(64)    NOT NULL,
    lottery_category VARCHAR(16)    NOT NULL,
    issue_no         VARCHAR(32)    NOT NULL,
    amount           NUMERIC(18, 2) NOT NULL,
    pnl              NUMERIC(18, 2) NOT NULL DEFAULT 0,
    status           VARCHAR(16)    NOT NULL DEFAULT 'pending',
    placed_at        TIMESTAMPTZ    NOT NULL DEFAULT now(),
    settled_at       TIMESTAMPTZ,
    created_at       TIMESTAMPTZ    NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ    NOT NULL DEFAULT now(),

    CONSTRAINT uq_bet_orders_order_no UNIQUE (order_no),
    CONSTRAINT fk_bet_orders_member FOREIGN KEY (member_id) REFERENCES members (id) ON DELETE RESTRICT,
    CONSTRAINT chk_bet_orders_amount CHECK (amount > 0),
    CONSTRAINT chk_bet_orders_status CHECK (status IN ('pending', 'win', 'lose', 'cancel')),
    CONSTRAINT chk_bet_orders_lottery_category CHECK (
        lottery_category IN ('ssc', 'pk10', 'k3', 'x5', 'other')
    )
);

COMMENT ON TABLE bet_orders IS '会员投注订单表：全站投注记录（区别于云端方案近 N 日汇总）';
COMMENT ON COLUMN bet_orders.id IS '主键';
COMMENT ON COLUMN bet_orders.order_no IS '投注单号，全局唯一';
COMMENT ON COLUMN bet_orders.member_id IS '会员 ID，关联 members.id';
COMMENT ON COLUMN bet_orders.lottery_code IS '彩种编码（对内，如 tencent_ffc）';
COMMENT ON COLUMN bet_orders.lottery_name IS '彩种展示名称（对外，如 时时彩 A）';
COMMENT ON COLUMN bet_orders.lottery_category IS '彩种大类：ssc=时时彩，pk10=PK10，k3=快三，x5=11选5，other=其他';
COMMENT ON COLUMN bet_orders.issue_no IS '投注期号';
COMMENT ON COLUMN bet_orders.amount IS '投注金额（元，2 位小数）';
COMMENT ON COLUMN bet_orders.pnl IS '结算盈亏（元，2 位小数；未开奖为 0）';
COMMENT ON COLUMN bet_orders.status IS '状态：pending=未开奖，win=已中奖，lose=未中奖，cancel=已撤单';
COMMENT ON COLUMN bet_orders.placed_at IS '投注时间（UTC）';
COMMENT ON COLUMN bet_orders.settled_at IS '结算时间（UTC）；未开奖/撤单为 NULL';
COMMENT ON COLUMN bet_orders.created_at IS '记录创建时间（UTC）';
COMMENT ON COLUMN bet_orders.updated_at IS '记录最后更新时间（UTC）';

CREATE INDEX idx_bet_orders_member_placed
    ON bet_orders (member_id, placed_at DESC);

COMMENT ON INDEX idx_bet_orders_member_placed IS '会员投注列表：按投注时间倒序';

CREATE INDEX idx_bet_orders_member_status_placed
    ON bet_orders (member_id, status, placed_at DESC);

COMMENT ON INDEX idx_bet_orders_member_status_placed IS '会员投注按状态筛选';

CREATE INDEX idx_bet_orders_member_category_placed
    ON bet_orders (member_id, lottery_category, placed_at DESC);

COMMENT ON INDEX idx_bet_orders_member_category_placed IS '会员投注按彩种大类筛选';

CREATE INDEX idx_bet_orders_pending_member
    ON bet_orders (member_id, placed_at DESC)
    WHERE status = 'pending';

COMMENT ON INDEX idx_bet_orders_pending_member IS '未开奖订单 partial index';
-- +goose StatementEnd

-- +goose Down
DROP TABLE IF EXISTS bet_orders;
