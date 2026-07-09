-- +goose Up
-- +goose StatementBegin
CREATE TABLE recharge_channels (
    id               VARCHAR(64)    PRIMARY KEY,
    channel_group    VARCHAR(32)    NOT NULL,
    label            VARCHAR(128)   NOT NULL,
    icon             VARCHAR(64)    NOT NULL DEFAULT 'payments',
    recommended      BOOLEAN        NOT NULL DEFAULT false,
    fee_rate         NUMERIC(8, 4)  NOT NULL DEFAULT 0,
    min_amount       NUMERIC(18, 2) NOT NULL DEFAULT 100,
    max_amount       NUMERIC(18, 2) NOT NULL DEFAULT 50000,
    show_activities  BOOLEAN        NOT NULL DEFAULT true,
    bind_reminder    TEXT,
    chain_hint       TEXT,
    sort_order       INT            NOT NULL DEFAULT 0,
    enabled          BOOLEAN        NOT NULL DEFAULT true,
    created_at       TIMESTAMPTZ    NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ    NOT NULL DEFAULT now(),

    CONSTRAINT chk_recharge_channels_group CHECK (
        channel_group IN ('crypto', 'social', 'wallet', 'alipay', 'bank')
    ),
    CONSTRAINT chk_recharge_channels_fee CHECK (fee_rate >= 0 AND fee_rate <= 1),
    CONSTRAINT chk_recharge_channels_amount CHECK (min_amount > 0 AND max_amount >= min_amount)
);

COMMENT ON TABLE recharge_channels IS '会员充值渠道配置；Admin 运营维护，Client 充值页只读拉取';
COMMENT ON COLUMN recharge_channels.id IS '渠道编码，与 recharge_orders.channel 对齐';
COMMENT ON COLUMN recharge_channels.channel_group IS '分组：crypto/social/wallet/alipay/bank';
COMMENT ON COLUMN recharge_channels.label IS '展示名称';
COMMENT ON COLUMN recharge_channels.icon IS 'Material Symbols 图标名';
COMMENT ON COLUMN recharge_channels.recommended IS '是否推荐';
COMMENT ON COLUMN recharge_channels.fee_rate IS '手续费比例，如 0.02 表示 2%';
COMMENT ON COLUMN recharge_channels.min_amount IS '单笔最小充值（元）';
COMMENT ON COLUMN recharge_channels.max_amount IS '单笔最大充值（元）';
COMMENT ON COLUMN recharge_channels.show_activities IS '是否展示充值活动区块';
COMMENT ON COLUMN recharge_channels.bind_reminder IS '选渠道时的绑定提醒';
COMMENT ON COLUMN recharge_channels.chain_hint IS '数字货币错链提示';
COMMENT ON COLUMN recharge_channels.sort_order IS '排序，越小越靠前';
COMMENT ON COLUMN recharge_channels.enabled IS '是否上架';

CREATE INDEX idx_recharge_channels_list ON recharge_channels (enabled DESC, sort_order ASC, id ASC);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS recharge_channels;
-- +goose StatementEnd
