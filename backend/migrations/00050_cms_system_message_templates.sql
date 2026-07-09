-- +goose Up
-- +goose StatementBegin
CREATE TABLE cms_system_message_templates (
    id          VARCHAR(64)  PRIMARY KEY,
    name        VARCHAR(128) NOT NULL,
    body_html   TEXT         NOT NULL DEFAULT '',
    audience    VARCHAR(64)  NOT NULL DEFAULT '全员',
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT now()
);

COMMENT ON TABLE cms_system_message_templates IS '系统讯息模板：Admin 维护文案，投递规则后续扩展';
COMMENT ON COLUMN cms_system_message_templates.audience IS '默认受众标签（展示用；分群投递后续实现）';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS cms_system_message_templates;
-- +goose StatementEnd
