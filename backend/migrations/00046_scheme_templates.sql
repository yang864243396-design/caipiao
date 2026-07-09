-- +goose Up
-- +goose StatementBegin
CREATE TABLE scheme_templates (
    id            VARCHAR(64)  PRIMARY KEY,
    name          VARCHAR(128) NOT NULL,
    lottery_code  VARCHAR(32)  NOT NULL,
    brief         TEXT,
    sort_order    INT          NOT NULL DEFAULT 10,
    enabled       BOOLEAN      NOT NULL DEFAULT true,
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ  NOT NULL DEFAULT now(),

    CONSTRAINT chk_scheme_templates_sort CHECK (sort_order >= 0 AND sort_order <= 9999)
);

COMMENT ON TABLE scheme_templates IS '高级方案模板库：Admin 维护，Client 倍投设定展示';
COMMENT ON COLUMN scheme_templates.id IS '模板 ID，全局唯一';
COMMENT ON COLUMN scheme_templates.name IS '模板名称';
COMMENT ON COLUMN scheme_templates.lottery_code IS '默认彩种编码，关联 lottery_catalog.code';
COMMENT ON COLUMN scheme_templates.brief IS '运营说明';
COMMENT ON COLUMN scheme_templates.sort_order IS '排序权重，越小越靠前';
COMMENT ON COLUMN scheme_templates.enabled IS '是否对会员可见';

CREATE INDEX idx_scheme_templates_sort ON scheme_templates (sort_order ASC, name ASC);
CREATE INDEX idx_scheme_templates_enabled ON scheme_templates (enabled, sort_order ASC)
    WHERE enabled = true;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS scheme_templates;
-- +goose StatementEnd
