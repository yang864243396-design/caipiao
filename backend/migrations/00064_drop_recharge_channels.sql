-- +goose Up
-- +goose StatementBegin
-- 移除「后台充值渠道」配置表。recharge_orders 保留（资金记录 / 看板 KPI 仍读取）。
DROP TABLE IF EXISTS recharge_channels;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS recharge_channels (
    id            VARCHAR(32)  PRIMARY KEY,
    name          VARCHAR(64)  NOT NULL,
    channel_group VARCHAR(16)  NOT NULL,
    min_amount    NUMERIC(18, 2) NOT NULL DEFAULT 0,
    max_amount    NUMERIC(18, 2) NOT NULL DEFAULT 0,
    fee_rate      NUMERIC(6, 4)  NOT NULL DEFAULT 0,
    enabled       BOOLEAN      NOT NULL DEFAULT true,
    sort_order    INT          NOT NULL DEFAULT 0,
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ  NOT NULL DEFAULT now()
);
-- +goose StatementEnd
