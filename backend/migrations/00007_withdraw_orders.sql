-- +goose Up
-- +goose StatementBegin
CREATE TABLE withdraw_orders (
    id              BIGSERIAL PRIMARY KEY,
    order_no        VARCHAR(64)    NOT NULL,
    member_id       BIGINT         NOT NULL,
    amount          NUMERIC(18, 2) NOT NULL,
    channel         VARCHAR(32)    NOT NULL,
    payout_account_id BIGINT,
    status          VARCHAR(16)    NOT NULL DEFAULT 'pending_review',
    reviewed_at     TIMESTAMPTZ,
    paid_at         TIMESTAMPTZ,
    reject_reason   TEXT,
    created_at      TIMESTAMPTZ    NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ    NOT NULL DEFAULT now(),

    CONSTRAINT uq_withdraw_orders_order_no UNIQUE (order_no),
    CONSTRAINT fk_withdraw_orders_member FOREIGN KEY (member_id) REFERENCES members (id) ON DELETE RESTRICT,
    CONSTRAINT chk_withdraw_orders_amount CHECK (amount > 0),
    CONSTRAINT chk_withdraw_orders_status CHECK (
        status IN ('pending_review', 'pending_payout', 'paid', 'rejected')
    )
);

COMMENT ON TABLE withdraw_orders IS '会员提现订单表：审核与打款流程';
COMMENT ON COLUMN withdraw_orders.id IS '主键';
COMMENT ON COLUMN withdraw_orders.order_no IS '提现订单号，全局唯一（如 WD00001）';
COMMENT ON COLUMN withdraw_orders.member_id IS '会员 ID，关联 members.id';
COMMENT ON COLUMN withdraw_orders.amount IS '提现金额（元，2 位小数），必须大于 0';
COMMENT ON COLUMN withdraw_orders.channel IS '提现渠道：bank_card、usdt_trc20、alipay 等';
COMMENT ON COLUMN withdraw_orders.payout_account_id IS '出款账户 ID，关联 member_payout_accounts.id；提交时可为空';
COMMENT ON COLUMN withdraw_orders.status IS '状态：pending_review=待审核，pending_payout=待打款，paid=已打款，rejected=已驳回';
COMMENT ON COLUMN withdraw_orders.reviewed_at IS '审核完成时间（UTC）；未审核为 NULL';
COMMENT ON COLUMN withdraw_orders.paid_at IS '打款完成时间（UTC）；未打款为 NULL';
COMMENT ON COLUMN withdraw_orders.reject_reason IS '驳回原因；未驳回为 NULL';
COMMENT ON COLUMN withdraw_orders.created_at IS '申请时间（UTC）';
COMMENT ON COLUMN withdraw_orders.updated_at IS '最后更新时间（UTC）';

CREATE INDEX idx_withdraw_orders_member_created
    ON withdraw_orders (member_id, created_at DESC);

COMMENT ON INDEX idx_withdraw_orders_member_created IS '会员提现记录列表';

CREATE INDEX idx_withdraw_orders_status_created
    ON withdraw_orders (status, created_at DESC);

COMMENT ON INDEX idx_withdraw_orders_status_created IS '管理端提现审批/出款队列';

CREATE INDEX idx_withdraw_orders_pending_review
    ON withdraw_orders (created_at DESC)
    WHERE status = 'pending_review';

COMMENT ON INDEX idx_withdraw_orders_pending_review IS '待审核提现 partial index';
-- +goose StatementEnd

-- +goose Down
DROP TABLE IF EXISTS withdraw_orders;
