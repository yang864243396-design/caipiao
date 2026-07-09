-- +goose Up
-- +goose StatementBegin
INSERT INTO admin_users (account, password_hash, display_name, role_id, status)
VALUES
    (
        'admin',
        '$2a$10$ZBnjtW5bjxwwxpF1KWIHxejIO1YN7Rfg994hzQlRNHIC4cQKPihsO',
        '超级管理员',
        'r_super',
        'active'
    ),
    (
        'fin_approve',
        '$2a$10$ZBnjtW5bjxwwxpF1KWIHxejIO1YN7Rfg994hzQlRNHIC4cQKPihsO',
        '财务-审批',
        'r_fin_approve',
        'active'
    ),
    (
        'fin_payout',
        '$2a$10$ZBnjtW5bjxwwxpF1KWIHxejIO1YN7Rfg994hzQlRNHIC4cQKPihsO',
        '财务-出纳',
        'r_fin_payout',
        'active'
    )
ON CONFLICT (account) DO NOTHING;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DELETE FROM admin_users WHERE account IN ('admin', 'fin_approve', 'fin_payout');
-- +goose StatementEnd
