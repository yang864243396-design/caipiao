-- +goose Up
-- +goose StatementBegin
CREATE TABLE admin_users (
    id              BIGSERIAL PRIMARY KEY,
    account         VARCHAR(64)  NOT NULL,
    password_hash   TEXT         NOT NULL,
    display_name    VARCHAR(128) NOT NULL,
    role_id         VARCHAR(64)  NOT NULL,
    status          VARCHAR(16)  NOT NULL DEFAULT 'active',
    last_login_at   TIMESTAMPTZ,
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),

    CONSTRAINT uq_admin_users_account UNIQUE (account),
    CONSTRAINT fk_admin_users_role FOREIGN KEY (role_id) REFERENCES admin_roles (id) ON DELETE RESTRICT,
    CONSTRAINT chk_admin_users_status CHECK (status IN ('active', 'disabled'))
);

COMMENT ON TABLE admin_users IS 'Admin 后台账号；登录后角色由 role_id 绑定，不再前端自选';
COMMENT ON COLUMN admin_users.account IS '登录账号';
COMMENT ON COLUMN admin_users.password_hash IS '登录密码 bcrypt 哈希';
COMMENT ON COLUMN admin_users.role_id IS '绑定角色，关联 admin_roles.id';
COMMENT ON COLUMN admin_users.status IS 'active=正常 disabled=停用';

CREATE INDEX idx_admin_users_role ON admin_users (role_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS admin_users;
-- +goose StatementEnd
