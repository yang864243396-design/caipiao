-- +goose Up
-- +goose StatementBegin
CREATE TABLE member_wallets (
    id              BIGSERIAL PRIMARY KEY,
    member_id       BIGINT         NOT NULL,
    balance         NUMERIC(18, 2) NOT NULL DEFAULT 0,
    frozen_balance  NUMERIC(18, 2) NOT NULL DEFAULT 0,
    currency        CHAR(3)        NOT NULL DEFAULT 'CNY',
    version         BIGINT         NOT NULL DEFAULT 0,
    created_at      TIMESTAMPTZ    NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ    NOT NULL DEFAULT now(),

    CONSTRAINT uq_member_wallets_member_id UNIQUE (member_id),
    CONSTRAINT fk_member_wallets_member FOREIGN KEY (member_id) REFERENCES members (id) ON DELETE RESTRICT,
    CONSTRAINT chk_member_wallets_balance CHECK (balance >= 0),
    CONSTRAINT chk_member_wallets_frozen CHECK (frozen_balance >= 0),
    CONSTRAINT chk_member_wallets_currency CHECK (currency ~ '^[A-Z]{3}$')
);

COMMENT ON TABLE member_wallets IS '会员钱包表：可用余额与冻结余额，与 members 一对一';
COMMENT ON COLUMN member_wallets.id IS '主键';
COMMENT ON COLUMN member_wallets.member_id IS '会员 ID，关联 members.id，唯一';
COMMENT ON COLUMN member_wallets.balance IS '可用余额（元，2 位小数），不得为负';
COMMENT ON COLUMN member_wallets.frozen_balance IS '冻结余额（元，2 位小数），如提现审核中';
COMMENT ON COLUMN member_wallets.currency IS '币种 ISO 4217 三字码，默认 CNY';
COMMENT ON COLUMN member_wallets.version IS '乐观锁版本号，每次余额变更 +1';
COMMENT ON COLUMN member_wallets.created_at IS '创建时间（UTC）';
COMMENT ON COLUMN member_wallets.updated_at IS '最后更新时间（UTC）';
-- +goose StatementEnd

-- +goose Down
DROP TABLE IF EXISTS member_wallets;
