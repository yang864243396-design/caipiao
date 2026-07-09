-- +goose Up
COMMENT ON TABLE schema_bootstrap IS '迁移脚手架表：验证数据库账号建表权限（非业务数据）';
COMMENT ON COLUMN schema_bootstrap.id IS '固定主键，恒为 1';
COMMENT ON COLUMN schema_bootstrap.note IS '说明文本，标识后端迁移环境';
COMMENT ON COLUMN schema_bootstrap.created_at IS '首次 bootstrap 写入时间（UTC）';

-- +goose Down
COMMENT ON TABLE schema_bootstrap IS NULL;
COMMENT ON COLUMN schema_bootstrap.id IS NULL;
COMMENT ON COLUMN schema_bootstrap.note IS NULL;
COMMENT ON COLUMN schema_bootstrap.created_at IS NULL;
