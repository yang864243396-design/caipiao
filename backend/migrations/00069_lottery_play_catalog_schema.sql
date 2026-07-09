-- +goose Up
-- +goose StatementBegin
-- P1：彩种目录扩展 + 玩法两层表 + purge 幂等标记（规划 v1.0 §2、§5.2）

DO $$ BEGIN
    CREATE TYPE lottery_sale_status AS ENUM ('on_sale', 'maintenance');
EXCEPTION
    WHEN duplicate_object THEN NULL;
END $$;

ALTER TABLE lottery_catalog
    ADD COLUMN IF NOT EXISTS category_code VARCHAR(16),
    ADD COLUMN IF NOT EXISTS play_template VARCHAR(32),
    ADD COLUMN IF NOT EXISTS ball_count SMALLINT,
    ADD COLUMN IF NOT EXISTS draw_interval VARCHAR(16),
    ADD COLUMN IF NOT EXISTS outbound_lottery_code VARCHAR(64),
    ADD COLUMN IF NOT EXISTS sale_status lottery_sale_status;

UPDATE lottery_catalog
SET sale_status = CASE WHEN on_sale THEN 'on_sale'::lottery_sale_status ELSE 'maintenance'::lottery_sale_status END
WHERE sale_status IS NULL;

ALTER TABLE lottery_catalog
    ALTER COLUMN sale_status SET DEFAULT 'on_sale',
    ALTER COLUMN sale_status SET NOT NULL;

ALTER TABLE lottery_catalog DROP COLUMN IF EXISTS detail_alias;

COMMENT ON COLUMN lottery_catalog.category_code IS '大类：ffc/jisu/lhc/syxw/pk10/k3/ssc/pc28';
COMMENT ON COLUMN lottery_catalog.play_template IS '玩法模板：ssc_std/lhc_std/...';
COMMENT ON COLUMN lottery_catalog.ball_count IS '开奖位数';
COMMENT ON COLUMN lottery_catalog.draw_interval IS '1m/3m/5m/jisu，空=无固定间隔';
COMMENT ON COLUMN lottery_catalog.outbound_lottery_code IS '第三方彩种对接编码';
COMMENT ON COLUMN lottery_catalog.sale_status IS '上架 on_sale / 维护 maintenance';

CREATE TABLE IF NOT EXISTS play_templates (
    code    VARCHAR(32) PRIMARY KEY,
    label   VARCHAR(64) NOT NULL,
    version INT         NOT NULL DEFAULT 1
);

CREATE TABLE IF NOT EXISTS play_types (
    id            BIGSERIAL PRIMARY KEY,
    template_code VARCHAR(32)  NOT NULL REFERENCES play_templates (code),
    type_id       VARCHAR(32)  NOT NULL,
    label         VARCHAR(32)  NOT NULL,
    sort_order    INT          NOT NULL,
    panel_type    VARCHAR(32),
    enabled       BOOLEAN      NOT NULL DEFAULT true,
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ  NOT NULL DEFAULT now(),
    UNIQUE (template_code, type_id)
);

CREATE TABLE IF NOT EXISTS sub_plays (
    id                 BIGSERIAL PRIMARY KEY,
    template_code      VARCHAR(32)  NOT NULL REFERENCES play_templates (code),
    type_id            VARCHAR(32)  NOT NULL,
    sub_id             VARCHAR(64)  NOT NULL,
    label              VARCHAR(64)  NOT NULL,
    sort_order         INT          NOT NULL,
    bet_mode           VARCHAR(24),
    segment_rule       JSONB        NOT NULL DEFAULT '{}',
    outbound_play_code VARCHAR(128),
    enabled            BOOLEAN      NOT NULL DEFAULT true,
    created_at         TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at         TIMESTAMPTZ  NOT NULL DEFAULT now(),
    UNIQUE (template_code, type_id, sub_id)
);

CREATE INDEX IF NOT EXISTS idx_play_types_tpl_sort
    ON play_types (template_code, sort_order);
CREATE INDEX IF NOT EXISTS idx_sub_plays_tpl_type_sort
    ON sub_plays (template_code, type_id, sort_order);
CREATE INDEX IF NOT EXISTS idx_lottery_catalog_sale_status
    ON lottery_catalog (sale_status, sort_order ASC, code ASC);

INSERT INTO play_templates (code, label, version) VALUES
    ('ssc_std',  '时时彩类标准玩法', 1),
    ('lhc_std',  '六合彩标准玩法',   1),
    ('syxw_std', '11选5标准玩法',    1),
    ('pk10_std', 'PK10标准玩法',     1),
    ('k3_std',   '快三标准玩法',     1),
    ('pc28_std', 'PC28标准玩法',     1)
ON CONFLICT (code) DO NOTHING;

CREATE TABLE IF NOT EXISTS lottery_catalog_purge_state (
    id           SMALLINT PRIMARY KEY DEFAULT 1 CHECK (id = 1),
    completed_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    note         TEXT        NOT NULL DEFAULT 'legacy 9 lottery purge'
);

COMMENT ON TABLE lottery_catalog_purge_state IS '旧 9 彩种 purge 幂等标记（startup 一次）';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS lottery_catalog_purge_state;
DROP TABLE IF EXISTS sub_plays;
DROP TABLE IF EXISTS play_types;
DROP TABLE IF EXISTS play_templates;

ALTER TABLE lottery_catalog
    ADD COLUMN IF NOT EXISTS detail_alias VARCHAR(64);

UPDATE lottery_catalog SET detail_alias = code WHERE detail_alias IS NULL;
ALTER TABLE lottery_catalog ALTER COLUMN detail_alias SET NOT NULL;

ALTER TABLE lottery_catalog
    DROP COLUMN IF EXISTS sale_status,
    DROP COLUMN IF EXISTS draw_interval,
    DROP COLUMN IF EXISTS ball_count,
    DROP COLUMN IF EXISTS play_template,
    DROP COLUMN IF EXISTS category_code,
    DROP COLUMN IF EXISTS outbound_lottery_code;

DROP TYPE IF EXISTS lottery_sale_status;
-- +goose StatementEnd
