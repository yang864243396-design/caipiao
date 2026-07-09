-- +goose Up
-- +goose StatementBegin
CREATE TABLE member_payout_accounts (
    id              BIGSERIAL PRIMARY KEY,
    member_id       BIGINT       NOT NULL,
    account_type    VARCHAR(32)  NOT NULL,
    label           VARCHAR(64)  NOT NULL,
    holder_name     VARCHAR(64),
    account_no      VARCHAR(128) NOT NULL,
    bank_name       VARCHAR(64),
    branch_name     VARCHAR(128),
    is_default      BOOLEAN      NOT NULL DEFAULT false,
    status          VARCHAR(16)  NOT NULL DEFAULT 'pending_review',
    reject_reason   TEXT,
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),

    CONSTRAINT fk_payout_accounts_member FOREIGN KEY (member_id) REFERENCES members (id) ON DELETE RESTRICT,
    CONSTRAINT chk_payout_accounts_type CHECK (
        account_type IN ('bank_card', 'usdt_trc20', 'usdt_bsc', 'usdt_ton', 'usdt_sol', 'alipay', 'mpay', 'goubao')
    ),
    CONSTRAINT chk_payout_accounts_status CHECK (
        status IN ('pending_review', 'active', 'rejected', 'disabled')
    )
);

COMMENT ON TABLE member_payout_accounts IS '会员出款账户：银行卡、USDT 地址、第三方钱包等';
COMMENT ON COLUMN member_payout_accounts.id IS '主键';
COMMENT ON COLUMN member_payout_accounts.member_id IS '会员 ID，关联 members.id';
COMMENT ON COLUMN member_payout_accounts.account_type IS '账户类型：bank_card、usdt_trc20、alipay 等';
COMMENT ON COLUMN member_payout_accounts.label IS '会员自定义标签，便于列表展示';
COMMENT ON COLUMN member_payout_accounts.holder_name IS '持卡人/账户姓名；链上地址类可为 NULL';
COMMENT ON COLUMN member_payout_accounts.account_no IS '卡号/链上地址/账号（存储时可脱敏，展示层再处理）';
COMMENT ON COLUMN member_payout_accounts.bank_name IS '银行名称；非银行卡为 NULL';
COMMENT ON COLUMN member_payout_accounts.branch_name IS '开户支行；非银行卡为 NULL';
COMMENT ON COLUMN member_payout_accounts.is_default IS '是否默认出款账户';
COMMENT ON COLUMN member_payout_accounts.status IS '状态：pending_review=待审核，active=可用，rejected=驳回，disabled=停用';
COMMENT ON COLUMN member_payout_accounts.reject_reason IS '审核驳回原因；未驳回为 NULL';
COMMENT ON COLUMN member_payout_accounts.created_at IS '绑定时间（UTC）';
COMMENT ON COLUMN member_payout_accounts.updated_at IS '最后更新时间（UTC）';

CREATE INDEX idx_payout_accounts_member_status
    ON member_payout_accounts (member_id, status, created_at DESC);

COMMENT ON INDEX idx_payout_accounts_member_status IS '会员出款账户列表';

CREATE UNIQUE INDEX uq_payout_accounts_member_default
    ON member_payout_accounts (member_id)
    WHERE is_default = true AND status = 'active';

COMMENT ON INDEX uq_payout_accounts_member_default IS '每位会员仅一个 active 默认出款账户';
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE withdraw_orders
    ADD CONSTRAINT fk_withdraw_orders_payout_account
    FOREIGN KEY (payout_account_id) REFERENCES member_payout_accounts (id) ON DELETE RESTRICT;

COMMENT ON CONSTRAINT fk_withdraw_orders_payout_account ON withdraw_orders IS '提现订单关联出款账户';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE withdraw_orders DROP CONSTRAINT IF EXISTS fk_withdraw_orders_payout_account;
DROP TABLE IF EXISTS member_payout_accounts;
-- +goose StatementEnd
