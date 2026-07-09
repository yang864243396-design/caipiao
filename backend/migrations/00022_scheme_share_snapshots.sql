-- +goose Up
-- +goose StatementBegin
CREATE TABLE scheme_share_snapshots (
    id             VARCHAR(32)    PRIMARY KEY,
    kind           VARCHAR(16)    NOT NULL DEFAULT 'custom',
    scheme_name    VARCHAR(128)   NOT NULL,
    lottery_code   VARCHAR(32)    NOT NULL,
    lottery_label  VARCHAR(64)    NOT NULL,
    play_method    VARCHAR(64)    NOT NULL DEFAULT '',
    fund_yuan      NUMERIC(14, 2) NOT NULL DEFAULT 0,
    config         JSONB          NOT NULL DEFAULT '{}',
    created_at     TIMESTAMPTZ    NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ    NOT NULL DEFAULT now(),

    CONSTRAINT chk_scheme_share_snapshots_kind CHECK (kind = 'custom')
);

COMMENT ON TABLE scheme_share_snapshots IS '分享池快照：仅自创型，与会员私池脱钩';
COMMENT ON COLUMN scheme_share_snapshots.id IS '快照 ID，如 SD10001';
COMMENT ON COLUMN scheme_share_snapshots.kind IS '固定 custom（自创）';
COMMENT ON COLUMN scheme_share_snapshots.scheme_name IS '方案展示名';
COMMENT ON COLUMN scheme_share_snapshots.lottery_code IS '彩种 code（对内）';
COMMENT ON COLUMN scheme_share_snapshots.lottery_label IS '彩种展示名（对外）';
COMMENT ON COLUMN scheme_share_snapshots.play_method IS '玩法展示文案';
COMMENT ON COLUMN scheme_share_snapshots.fund_yuan IS '方案资金（元）';
COMMENT ON COLUMN scheme_share_snapshots.config IS '完整方案配置 JSON';

CREATE INDEX idx_scheme_share_snapshots_name
    ON scheme_share_snapshots (scheme_name);

COMMENT ON INDEX idx_scheme_share_snapshots_name IS '按方案名检索';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS scheme_share_snapshots;
-- +goose StatementEnd
