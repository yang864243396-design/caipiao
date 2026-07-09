-- 草案：彩种目录扩展 + 玩法两层表（P1 迁入 migrations 时改编号/拆文件）
-- 数据源：backend/docs/seeds/*.csv
-- 规划：backend/docs/lottery-catalog-migration-plan.md v0.5
-- P0 CSV：52 play_types + 340 sub_plays（generate_p0_seeds.py）

-- +goose Up
-- +goose StatementBegin

-- 1) 扩展 lottery_catalog
ALTER TABLE lottery_catalog
    ADD COLUMN IF NOT EXISTS category_code VARCHAR(16),
    ADD COLUMN IF NOT EXISTS play_template VARCHAR(32),
    ADD COLUMN IF NOT EXISTS ball_count    SMALLINT,
    ADD COLUMN IF NOT EXISTS draw_interval VARCHAR(16),
    ADD COLUMN IF NOT EXISTS outbound_lottery_code VARCHAR(64);

-- 旧字段 detail_alias 在 P1 单独 migration 删除（先双写/弃用视联调而定）
-- ALTER TABLE lottery_catalog DROP COLUMN detail_alias;

COMMENT ON COLUMN lottery_catalog.category_code IS '大类：ffc/jisu/lhc/syxw/pk10/k3/ssc/pc28';
COMMENT ON COLUMN lottery_catalog.play_template IS '玩法模板：ssc_std/lhc_std/...';
COMMENT ON COLUMN lottery_catalog.ball_count IS '开奖位数，见规划 §0';
COMMENT ON COLUMN lottery_catalog.draw_interval IS '1m/3m/5m/jisu，空=无固定间隔';

-- 2) 玩法模板与两层玩法树
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
    id            BIGSERIAL PRIMARY KEY,
    template_code VARCHAR(32)  NOT NULL REFERENCES play_templates (code),
    type_id       VARCHAR(32)  NOT NULL,
    sub_id        VARCHAR(64)  NOT NULL,
    label         VARCHAR(64)  NOT NULL,
    sort_order    INT          NOT NULL,
    bet_mode      VARCHAR(24),
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

-- 3) 模板占位（6 套）
INSERT INTO play_templates (code, label, version) VALUES
    ('ssc_std',  '时时彩类标准玩法', 1),
    ('lhc_std',  '六合彩标准玩法',   1),
    ('syxw_std', '11选5标准玩法',    1),
    ('pk10_std', 'PK10标准玩法',     1),
    ('k3_std',   '快三标准玩法',     1),
    ('pc28_std', 'PC28标准玩法',     1)
ON CONFLICT (code) DO NOTHING;

-- 4) 旧彩种下架（保留 code，历史单不迁移）
UPDATE lottery_catalog SET on_sale = false
WHERE code IN (
    'tencent_ffc', 'tencent_10', 'qiqu_tencent', 'us_ffc',
    'cq_ssc', 'xj_ssc', 'tj_ssc', 'fc_3d', 'pl3'
);

-- 5) 新 47 彩种 + play_types + sub_plays
-- 建议 P1 用脚本将 docs/seeds/*.csv 转为 INSERT（340 行 sub_plays 不宜手写）
--   python backend/docs/seeds/generate_p0_seeds.py
--   再运行配套 import 工具或 COPY 进 sub_plays / play_types / lottery_catalog

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS sub_plays;
DROP TABLE IF EXISTS play_types;
DROP TABLE IF EXISTS play_templates;

ALTER TABLE lottery_catalog
    DROP COLUMN IF EXISTS draw_interval,
    DROP COLUMN IF EXISTS ball_count,
    DROP COLUMN IF EXISTS play_template,
    DROP COLUMN IF EXISTS category_code,
    DROP COLUMN IF EXISTS outbound_lottery_code;

-- 旧 9 彩种 on_sale 恢复需按环境决定，此处不自动回滚
-- +goose StatementEnd
