-- +goose Up
-- +goose StatementBegin
CREATE TABLE scheme_definitions (
    id                 VARCHAR(64)   PRIMARY KEY,
    member_id          BIGINT        NOT NULL,
    kind               VARCHAR(16)   NOT NULL,
    scheme_name        VARCHAR(128)  NOT NULL,
    lottery_code       VARCHAR(32)   NOT NULL,
    lottery_label      VARCHAR(64)   NOT NULL DEFAULT '',
    share_status       VARCHAR(16)   NOT NULL DEFAULT 'private',
    share_status_locked BOOLEAN      NOT NULL DEFAULT false,
    config             JSONB         NOT NULL DEFAULT '{}',
    source_snapshot_id VARCHAR(32),
    created_at         TIMESTAMPTZ   NOT NULL DEFAULT now(),
    updated_at         TIMESTAMPTZ   NOT NULL DEFAULT now(),

    CONSTRAINT uq_scheme_definitions_member_name UNIQUE (member_id, scheme_name),
    CONSTRAINT fk_scheme_definitions_member FOREIGN KEY (member_id) REFERENCES members (id) ON DELETE CASCADE,
    CONSTRAINT chk_scheme_definitions_kind CHECK (kind IN ('custom', 'contrary', 'follow')),
    CONSTRAINT chk_scheme_definitions_share CHECK (share_status IN ('private', 'public'))
);

COMMENT ON TABLE scheme_definitions IS '会员私池方案定义';
COMMENT ON COLUMN scheme_definitions.id IS '方案定义 ID';
COMMENT ON COLUMN scheme_definitions.member_id IS '所属会员 ID';
COMMENT ON COLUMN scheme_definitions.kind IS '方案类型：custom 自创 / contrary 反买 / follow 跟单';
COMMENT ON COLUMN scheme_definitions.scheme_name IS '方案名称（同会员内唯一）';
COMMENT ON COLUMN scheme_definitions.lottery_code IS '彩种 code（对内）';
COMMENT ON COLUMN scheme_definitions.lottery_label IS '彩种展示名（对外）';
COMMENT ON COLUMN scheme_definitions.share_status IS '分享状态（跟单/反买恒为 private）';
COMMENT ON COLUMN scheme_definitions.share_status_locked IS '分享状态是否已锁定（添加至云端后 true）';
COMMENT ON COLUMN scheme_definitions.config IS '方案配置 JSON';
COMMENT ON COLUMN scheme_definitions.source_snapshot_id IS '来源分享池快照 ID（跟单下载时）';

CREATE TABLE scheme_instances (
    id             VARCHAR(64)    PRIMARY KEY,
    definition_id  VARCHAR(64)    NOT NULL,
    member_id      BIGINT         NOT NULL,
    kind           VARCHAR(16)    NOT NULL,
    scheme_name    VARCHAR(128)   NOT NULL,
    lottery_code   VARCHAR(32)    NOT NULL,
    lottery_label  VARCHAR(64)    NOT NULL DEFAULT '',
    status         VARCHAR(16)    NOT NULL DEFAULT 'pending',
    run_mode       VARCHAR(8)     NOT NULL DEFAULT 'real',
    turnover       NUMERIC(14, 2) NOT NULL DEFAULT 0,
    pnl            NUMERIC(14, 2) NOT NULL DEFAULT 0,
    run_time_sec   INT            NOT NULL DEFAULT 0,
    lookback_pnl   NUMERIC(14, 2) NOT NULL DEFAULT 0,
    multiplier     NUMERIC(8, 2)  NOT NULL DEFAULT 1,
    countdown_sec  INT            NOT NULL DEFAULT 0,
    sim_bet        BOOLEAN        NOT NULL DEFAULT false,
    created_at     TIMESTAMPTZ    NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ    NOT NULL DEFAULT now(),

    CONSTRAINT uq_scheme_instances_definition UNIQUE (definition_id),
    CONSTRAINT fk_scheme_instances_definition FOREIGN KEY (definition_id) REFERENCES scheme_definitions (id) ON DELETE CASCADE,
    CONSTRAINT fk_scheme_instances_member FOREIGN KEY (member_id) REFERENCES members (id) ON DELETE CASCADE,
    CONSTRAINT chk_scheme_instances_kind CHECK (kind IN ('custom', 'contrary', 'follow')),
    CONSTRAINT chk_scheme_instances_status CHECK (status IN ('pending', 'running', 'paused', 'soft_stopped')),
    CONSTRAINT chk_scheme_instances_run_mode CHECK (run_mode IN ('real', 'sim'))
);

COMMENT ON TABLE scheme_instances IS '云端方案实例（1 方案定义 : 1 实例）';
COMMENT ON COLUMN scheme_instances.definition_id IS '关联 scheme_definitions.id';
COMMENT ON COLUMN scheme_instances.status IS 'pending / running / paused / soft_stopped';
COMMENT ON COLUMN scheme_instances.run_mode IS 'real 真实 / sim 模拟';

CREATE INDEX idx_scheme_definitions_member
    ON scheme_definitions (member_id, updated_at DESC);

CREATE INDEX idx_scheme_instances_member_status
    ON scheme_instances (member_id, status);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS scheme_instances;
DROP TABLE IF EXISTS scheme_definitions;
-- +goose StatementEnd
