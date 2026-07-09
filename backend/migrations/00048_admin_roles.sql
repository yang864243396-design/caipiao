-- +goose Up
-- +goose StatementBegin
CREATE TABLE admin_roles (
    id          VARCHAR(64)  PRIMARY KEY,
    name        VARCHAR(128) NOT NULL,
    menu_paths  JSONB        NOT NULL DEFAULT '[]'::jsonb,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT now()
);

COMMENT ON TABLE admin_roles IS 'Admin 角色 RBAC：菜单 path 前缀白名单';
COMMENT ON COLUMN admin_roles.menu_paths IS '可见菜单 path 数组；「/」表示全部';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS admin_roles;
-- +goose StatementEnd
