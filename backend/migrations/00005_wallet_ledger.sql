-- +goose Up
-- +goose StatementBegin
CREATE TABLE wallet_ledger (
    id              BIGSERIAL PRIMARY KEY,
    ledger_no       VARCHAR(32)    NOT NULL,
    member_id       BIGINT         NOT NULL,
    txn_type        VARCHAR(32)    NOT NULL,
    delta_amount    NUMERIC(18, 2) NOT NULL,
    balance_after   NUMERIC(18, 2) NOT NULL,
    order_ref       VARCHAR(64),
    remark          TEXT,
    created_at      TIMESTAMPTZ    NOT NULL DEFAULT now(),

    CONSTRAINT uq_wallet_ledger_ledger_no UNIQUE (ledger_no),
    CONSTRAINT fk_wallet_ledger_member FOREIGN KEY (member_id) REFERENCES members (id) ON DELETE RESTRICT,
    CONSTRAINT chk_wallet_ledger_txn_type CHECK (
        txn_type IN ('deposit', 'withdraw', 'bet_debit', 'payout', 'withdraw_freeze', 'adjust')
    ),
    CONSTRAINT chk_wallet_ledger_balance_after CHECK (balance_after >= 0)
);

COMMENT ON TABLE wallet_ledger IS '会员帐变流水表：只增不改，记录每笔资金变动及变动后余额';
COMMENT ON COLUMN wallet_ledger.id IS '主键';
COMMENT ON COLUMN wallet_ledger.ledger_no IS '帐变流水号，全局唯一';
COMMENT ON COLUMN wallet_ledger.member_id IS '会员 ID，关联 members.id';
COMMENT ON COLUMN wallet_ledger.txn_type IS '帐变类型：deposit=入款，withdraw=出款，bet_debit=投注扣款，payout=派奖，withdraw_freeze=提现冻结，adjust=调账';
COMMENT ON COLUMN wallet_ledger.delta_amount IS '变动金额（元，2 位小数；正为增加，负为减少）';
COMMENT ON COLUMN wallet_ledger.balance_after IS '变动后可用余额（元，2 位小数）';
COMMENT ON COLUMN wallet_ledger.order_ref IS '关联业务单号（充值/提现/投注等）；无关联时为 NULL';
COMMENT ON COLUMN wallet_ledger.remark IS '备注或运营说明；可为空';
COMMENT ON COLUMN wallet_ledger.created_at IS '帐变发生时间（UTC）';

CREATE INDEX idx_wallet_ledger_member_created
    ON wallet_ledger (member_id, created_at DESC);

COMMENT ON INDEX idx_wallet_ledger_member_created IS '会员帐变列表：按时间倒序';

CREATE INDEX idx_wallet_ledger_order_ref
    ON wallet_ledger (order_ref)
    WHERE order_ref IS NOT NULL;

COMMENT ON INDEX idx_wallet_ledger_order_ref IS '按关联单号反查帐变';

CREATE INDEX idx_wallet_ledger_member_type_created
    ON wallet_ledger (member_id, txn_type, created_at DESC);

COMMENT ON INDEX idx_wallet_ledger_member_type_created IS '会员帐变按类型筛选';
-- +goose StatementEnd

-- +goose Down
DROP TABLE IF EXISTS wallet_ledger;
