-- +goose Up
-- +goose StatementBegin
INSERT INTO cms_system_message_templates (id, name, body_html, audience, created_at, updated_at) VALUES
    (
        'sys-msg-tpl-maint',
        '系统维护播报',
        '<p>系统将于 <strong>UTC+8</strong> 维护窗口内进行升级……</p>',
        '全员',
        now(),
        now()
    ),
    (
        'sys-msg-tpl-pay',
        '到账说明',
        '<p>充值到账时间受渠道影响……</p>',
        '全员',
        now(),
        now()
    )
ON CONFLICT (id) DO NOTHING;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DELETE FROM cms_system_message_templates WHERE id IN ('sys-msg-tpl-maint', 'sys-msg-tpl-pay');
-- +goose StatementEnd
