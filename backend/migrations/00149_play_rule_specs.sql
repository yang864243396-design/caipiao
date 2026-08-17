-- +goose Up
-- 当前已发布的玩法规则。目录同步会重建 sub_plays，因此以目录自然键关联，不绑定 sub_plays.id。
CREATE TABLE IF NOT EXISTS play_rule_specs (
    id BIGSERIAL PRIMARY KEY,
    template_code VARCHAR(32) NOT NULL,
    type_id VARCHAR(32) NOT NULL,
    sub_id VARCHAR(64) NOT NULL,
    lottery_code VARCHAR(64),
    rule_version INTEGER NOT NULL CHECK (rule_version > 0),
    evaluator_key VARCHAR(64) NOT NULL,
    evaluator_version INTEGER NOT NULL CHECK (evaluator_version > 0),
    evaluation_spec JSONB NOT NULL,
    sample_cases JSONB NOT NULL DEFAULT '[]'::jsonb,
    source_meta JSONB NOT NULL DEFAULT '{}'::jsonb,
    strategy_enabled BOOLEAN NOT NULL DEFAULT false,
    published_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT uq_play_rule_specs_scope
        UNIQUE NULLS NOT DISTINCT (template_code, type_id, sub_id, lottery_code)
);

-- 草稿、校验、发布与停用的不可变修订记录；不参与开奖热路径查询。
CREATE TABLE IF NOT EXISTS play_rule_spec_revisions (
    id BIGSERIAL PRIMARY KEY,
    rule_spec_id BIGINT,
    template_code VARCHAR(32) NOT NULL,
    type_id VARCHAR(32) NOT NULL,
    sub_id VARCHAR(64) NOT NULL,
    lottery_code VARCHAR(64),
    revision INTEGER NOT NULL CHECK (revision > 0),
    status VARCHAR(16) NOT NULL CHECK (status IN ('draft', 'verified', 'published', 'disabled')),
    evaluator_key VARCHAR(64) NOT NULL,
    evaluator_version INTEGER NOT NULL CHECK (evaluator_version > 0),
    evaluation_spec JSONB NOT NULL,
    sample_cases JSONB NOT NULL DEFAULT '[]'::jsonb,
    source_meta JSONB NOT NULL DEFAULT '{}'::jsonb,
    actor VARCHAR(128) NOT NULL DEFAULT '',
    change_reason TEXT NOT NULL DEFAULT '',
    verified_at TIMESTAMPTZ,
    published_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT uq_play_rule_spec_revisions_scope
        UNIQUE NULLS NOT DISTINCT (template_code, type_id, sub_id, lottery_code, revision)
);

-- 每个实例、期号只允许一条策略判定记录。WebSocket 仅唤醒；重启扫描依赖本表恢复。
CREATE TABLE IF NOT EXISTS scheme_strategy_evaluations (
    id BIGSERIAL PRIMARY KEY,
    instance_id VARCHAR(64) NOT NULL REFERENCES scheme_instances(id) ON DELETE CASCADE,
    lottery_code VARCHAR(64) NOT NULL,
    period_no VARCHAR(32) NOT NULL,
    cloud_bet_record_id BIGINT,
    bet_order_no VARCHAR(64),
    status VARCHAR(16) NOT NULL DEFAULT 'pending'
        CHECK (status IN ('pending', 'processing', 'completed', 'skipped', 'mismatch')),
    rule_version INTEGER,
    rule_snapshot_hash TEXT,
    local_hit BOOLEAN,
    winning_units INTEGER,
    diagnostics JSONB NOT NULL DEFAULT '{}'::jsonb,
    claimed_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (instance_id, period_no)
);

ALTER TABLE cloud_bet_records
    ADD COLUMN IF NOT EXISTS rule_snapshot JSONB,
    ADD COLUMN IF NOT EXISTS rule_version INTEGER,
    ADD COLUMN IF NOT EXISTS rule_snapshot_hash TEXT,
    ADD COLUMN IF NOT EXISTS strategy_evaluated_at TIMESTAMPTZ;

CREATE INDEX IF NOT EXISTS idx_play_rule_specs_enabled
    ON play_rule_specs (template_code, type_id, sub_id, lottery_code)
    WHERE strategy_enabled;
CREATE INDEX IF NOT EXISTS idx_play_rule_spec_revisions_status
    ON play_rule_spec_revisions (status, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_scheme_strategy_evaluations_recovery
    ON scheme_strategy_evaluations (status, lottery_code, created_at ASC)
    WHERE status IN ('pending', 'processing');
CREATE INDEX IF NOT EXISTS idx_cloud_bet_records_strategy_pending
    ON cloud_bet_records (scheme_id, period_no)
    WHERE third_party_bet_id IS NOT NULL AND strategy_evaluated_at IS NULL;

-- +goose Down
DROP INDEX IF EXISTS idx_cloud_bet_records_strategy_pending;
DROP INDEX IF EXISTS idx_scheme_strategy_evaluations_recovery;
DROP INDEX IF EXISTS idx_play_rule_spec_revisions_status;
DROP INDEX IF EXISTS idx_play_rule_specs_enabled;

ALTER TABLE cloud_bet_records
    DROP COLUMN IF EXISTS strategy_evaluated_at,
    DROP COLUMN IF EXISTS rule_snapshot_hash,
    DROP COLUMN IF EXISTS rule_version,
    DROP COLUMN IF EXISTS rule_snapshot;

DROP TABLE IF EXISTS scheme_strategy_evaluations;
DROP TABLE IF EXISTS play_rule_spec_revisions;
DROP TABLE IF EXISTS play_rule_specs;
