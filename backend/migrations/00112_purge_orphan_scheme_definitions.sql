-- +goose Up
-- +goose StatementBegin
-- 清理未添加至云端的孤儿方案定义（无 scheme_instances 关联）
DELETE FROM scheme_definitions d
WHERE NOT EXISTS (
    SELECT 1 FROM scheme_instances i WHERE i.definition_id = d.id
);
-- +goose StatementEnd

-- +goose Down
-- 数据清理不可逆
