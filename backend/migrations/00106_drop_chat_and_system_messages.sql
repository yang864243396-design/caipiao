-- +goose Up
-- +goose StatementBegin
DROP TABLE IF EXISTS cms_system_message_templates;
DROP TABLE IF EXISTS member_system_messages;
DROP TABLE IF EXISTS member_chat_messages;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS member_chat_messages (
    id         BIGSERIAL PRIMARY KEY,
    member_id  BIGINT       NOT NULL REFERENCES members(id) ON DELETE CASCADE,
    peer_key   VARCHAR(64)  NOT NULL,
    direction  VARCHAR(8)   NOT NULL,
    body       TEXT         NOT NULL,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT now(),
    CONSTRAINT chk_member_chat_direction CHECK (direction IN ('in', 'out'))
);

CREATE TABLE IF NOT EXISTS member_system_messages (
    id         BIGSERIAL PRIMARY KEY,
    member_id  BIGINT    NOT NULL REFERENCES members(id) ON DELETE CASCADE,
    body       TEXT      NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS cms_system_message_templates (
    id          VARCHAR(64)  PRIMARY KEY,
    name        VARCHAR(128) NOT NULL,
    body_html   TEXT         NOT NULL DEFAULT '',
    audience    VARCHAR(64)  NOT NULL DEFAULT '全员',
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT now()
);
-- +goose StatementEnd
