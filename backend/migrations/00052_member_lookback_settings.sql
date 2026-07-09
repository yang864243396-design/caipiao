-- +goose Up
-- +goose StatementBegin
CREATE TABLE member_lookback_settings (
    member_id                 BIGINT         PRIMARY KEY REFERENCES members (id) ON DELETE CASCADE,
    run_mode                  VARCHAR(8)     NOT NULL DEFAULT 'real',
    judgment                  VARCHAR(16)    NOT NULL DEFAULT 'individual',
    single_profit_threshold   NUMERIC(14, 2) NOT NULL DEFAULT 100,
    single_loss_threshold     NUMERIC(14, 2) NOT NULL DEFAULT 0,
    overall_profit_threshold  NUMERIC(14, 2) NOT NULL DEFAULT 0,
    overall_loss_threshold    NUMERIC(14, 2) NOT NULL DEFAULT 0,
    scheme_wins_min           NUMERIC(14, 2) NOT NULL DEFAULT 0,
    scheme_wins_max           NUMERIC(14, 2) NOT NULL DEFAULT 0,
    period_profit             NUMERIC(14, 2) NOT NULL DEFAULT 0,
    period_loss               NUMERIC(14, 2) NOT NULL DEFAULT 0,
    created_at                TIMESTAMPTZ    NOT NULL DEFAULT now(),
    updated_at                TIMESTAMPTZ    NOT NULL DEFAULT now(),

    CONSTRAINT chk_member_lookback_run_mode CHECK (run_mode IN ('real', 'sim')),
    CONSTRAINT chk_member_lookback_judgment CHECK (judgment IN ('individual', 'overall'))
);

COMMENT ON TABLE member_lookback_settings IS '会员云端回头设置（按会员持久化）';
COMMENT ON COLUMN member_lookback_settings.run_mode IS 'real 正式 / sim 模拟';
COMMENT ON COLUMN member_lookback_settings.judgment IS 'individual 个别判断 / overall 整体判断';
COMMENT ON COLUMN member_lookback_settings.single_profit_threshold IS '单方案盈利阈值（元）';
COMMENT ON COLUMN member_lookback_settings.single_loss_threshold IS '单方案亏损阈值（元）';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS member_lookback_settings;
-- +goose StatementEnd
