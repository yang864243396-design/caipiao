-- +goose Up
-- +goose StatementBegin
-- 移除「提现 / 绑定银行卡」模块：先删 withdraw_orders（含对 payout 的外键），再删 member_payout_accounts。
DROP TABLE IF EXISTS withdraw_orders;
DROP TABLE IF EXISTS member_payout_accounts;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- 回滚：重建表结构（不还原历史数据）。
CREATE TABLE IF NOT EXISTS member_payout_accounts (
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

CREATE TABLE IF NOT EXISTS withdraw_orders (
    id                BIGSERIAL PRIMARY KEY,
    order_no          VARCHAR(64)    NOT NULL,
    member_id         BIGINT         NOT NULL,
    amount            NUMERIC(18, 2) NOT NULL,
    channel           VARCHAR(32)    NOT NULL,
    payout_account_id BIGINT,
    status            VARCHAR(16)    NOT NULL DEFAULT 'pending_review',
    reviewed_at       TIMESTAMPTZ,
    paid_at           TIMESTAMPTZ,
    reject_reason     TEXT,
    created_at        TIMESTAMPTZ    NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ    NOT NULL DEFAULT now(),
    CONSTRAINT uq_withdraw_orders_order_no UNIQUE (order_no),
    CONSTRAINT fk_withdraw_orders_member FOREIGN KEY (member_id) REFERENCES members (id) ON DELETE RESTRICT,
    CONSTRAINT fk_withdraw_orders_payout_account FOREIGN KEY (payout_account_id) REFERENCES member_payout_accounts (id) ON DELETE RESTRICT,
    CONSTRAINT chk_withdraw_orders_amount CHECK (amount > 0),
    CONSTRAINT chk_withdraw_orders_status CHECK (
        status IN ('pending_review', 'pending_payout', 'paid', 'rejected')
    )
);
-- +goose StatementEnd
