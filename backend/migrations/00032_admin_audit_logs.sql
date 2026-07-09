-- +goose Up
-- +goose StatementBegin
CREATE TABLE admin_audit_logs (
    id         VARCHAR(16)  PRIMARY KEY,
    actor      VARCHAR(64)  NOT NULL,
    action     TEXT         NOT NULL,
    ip         VARCHAR(45)  NOT NULL DEFAULT '127.0.0.1',
    created_at TIMESTAMPTZ  NOT NULL DEFAULT now()
);

COMMENT ON TABLE admin_audit_logs IS '后台操作审计留痕';
COMMENT ON COLUMN admin_audit_logs.actor IS '操作者账号（admin 登录名）';
COMMENT ON COLUMN admin_audit_logs.action IS '动作描述';
COMMENT ON COLUMN admin_audit_logs.ip IS '来源 IP';

CREATE INDEX idx_admin_audit_logs_created
    ON admin_audit_logs (created_at DESC, id DESC);

CREATE SEQUENCE admin_audit_log_seq START WITH 51;

COMMENT ON SEQUENCE admin_audit_log_seq IS '审计 ID 序号（AUD00051 起）';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP SEQUENCE IF EXISTS admin_audit_log_seq;
DROP TABLE IF EXISTS admin_audit_logs;
-- +goose StatementEnd
