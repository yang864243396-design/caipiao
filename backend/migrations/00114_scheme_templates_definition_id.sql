-- +goose Up
-- +goose StatementBegin
-- 高级倍投模板归属方案：definition_id 非空表示该方案下会员自建模板
ALTER TABLE scheme_templates
    ADD COLUMN IF NOT EXISTS definition_id VARCHAR(64) NULL REFERENCES scheme_definitions (id) ON DELETE CASCADE;

CREATE INDEX IF NOT EXISTS idx_scheme_templates_definition
    ON scheme_templates (definition_id, sort_order ASC)
    WHERE definition_id IS NOT NULL;

COMMENT ON COLUMN scheme_templates.definition_id IS 'NULL=平台模板；非 NULL=归属 scheme_definitions.id 的会员方案模板';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_scheme_templates_definition;
ALTER TABLE scheme_templates DROP COLUMN IF EXISTS definition_id;
-- +goose StatementEnd
