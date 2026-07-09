-- +goose Up
-- +goose StatementBegin
-- 会员自建高级倍投模板：config 存轮次规则；member_id NULL 为平台模板（Admin 维护）
ALTER TABLE scheme_templates
    ADD COLUMN IF NOT EXISTS config JSONB NOT NULL DEFAULT '{}',
    ADD COLUMN IF NOT EXISTS member_id BIGINT NULL REFERENCES members (id) ON DELETE CASCADE;

CREATE INDEX IF NOT EXISTS idx_scheme_templates_member
    ON scheme_templates (member_id, sort_order ASC)
    WHERE member_id IS NOT NULL;

COMMENT ON COLUMN scheme_templates.config IS '高级倍投轮次等 JSON（rounds[]）';
COMMENT ON COLUMN scheme_templates.member_id IS 'NULL=平台模板；非 NULL=会员自建';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_scheme_templates_member;
ALTER TABLE scheme_templates DROP COLUMN IF EXISTS member_id;
ALTER TABLE scheme_templates DROP COLUMN IF EXISTS config;
-- +goose StatementEnd
