-- +goose Up
-- +goose StatementBegin
CREATE TABLE cms_lobby_slots (
    id         VARCHAR(64)  PRIMARY KEY,
    slot_key   VARCHAR(64)  NOT NULL,
    title      VARCHAR(128) NOT NULL,
    brief      TEXT         NOT NULL DEFAULT '',
    sort       INT          NOT NULL DEFAULT 0,
    enabled    BOOLEAN      NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ  NOT NULL DEFAULT now(),

    CONSTRAINT uq_cms_lobby_slots_key UNIQUE (slot_key)
);

COMMENT ON TABLE cms_lobby_slots IS '游戏大厅运营位配置';
COMMENT ON COLUMN cms_lobby_slots.slot_key IS '运营位唯一键，如 hero_main';
COMMENT ON COLUMN cms_lobby_slots.brief IS '运营说明/占位文案';

CREATE INDEX idx_cms_lobby_slots_sort ON cms_lobby_slots (sort ASC, id ASC);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS cms_lobby_slots;
-- +goose StatementEnd
