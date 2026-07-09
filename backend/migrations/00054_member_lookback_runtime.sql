-- +goose Up
-- +goose StatementBegin
CREATE TABLE member_lookback_runtime (
    member_id        BIGINT         NOT NULL REFERENCES members (id) ON DELETE CASCADE,
    run_mode         VARCHAR(8)     NOT NULL,
    session_pnl      NUMERIC(14, 2) NOT NULL DEFAULT 0,
    period_issue     VARCHAR(32)    NOT NULL DEFAULT '',
    period_pnl       NUMERIC(14, 2) NOT NULL DEFAULT 0,
    period_hit_count INT            NOT NULL DEFAULT 0,
    updated_at       TIMESTAMPTZ    NOT NULL DEFAULT now(),

    PRIMARY KEY (member_id, run_mode),
    CONSTRAINT chk_member_lookback_runtime_mode CHECK (run_mode IN ('real', 'sim'))
);

COMMENT ON TABLE member_lookback_runtime IS '会员回头运行时累计（整体判断用）';
COMMENT ON COLUMN member_lookback_runtime.session_pnl IS '整体回头累计盈亏（元）';
COMMENT ON COLUMN member_lookback_runtime.period_issue IS '当前统计期号';
COMMENT ON COLUMN member_lookback_runtime.period_pnl IS '当期累计盈亏（元）';
COMMENT ON COLUMN member_lookback_runtime.period_hit_count IS '当期命中方案数';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS member_lookback_runtime;
-- +goose StatementEnd
