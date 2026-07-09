-- +goose Up
-- +goose StatementBegin
INSERT INTO admin_roles (id, name, menu_paths, created_at, updated_at) VALUES
    ('r_super', '超级管理员', '["/"]'::jsonb, now(), now()),
    ('r_fin_approve', '财务-审批', '["/dashboard","/funds/withdraw-approval","/reports"]'::jsonb, now(), now()),
    ('r_fin_payout', '财务-出纳', '["/dashboard","/funds/withdraw-payout"]'::jsonb, now(), now())
ON CONFLICT (id) DO NOTHING;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DELETE FROM admin_roles WHERE id IN ('r_super', 'r_fin_approve', 'r_fin_payout');
-- +goose StatementEnd
