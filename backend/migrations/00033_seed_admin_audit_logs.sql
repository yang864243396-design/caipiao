-- +goose Up
-- +goose StatementBegin
INSERT INTO admin_audit_logs (id, actor, action, ip, created_at) VALUES
    ('AUD00001', 'admin', '通过提现 WD00001', '127.0.0.1', now() - interval '37 minutes'),
    ('AUD00002', 'admin', '编辑公告 ANN0001', '127.0.0.1', now() - interval '74 minutes'),
    ('AUD00003', 'fin_approve', '强停方案实例 SCH00001', '127.0.0.1', now() - interval '111 minutes'),
    ('AUD00004', 'fin_payout', '改参 scheme · refId scheme_adv_demo', '10.0.0.12', now() - interval '148 minutes'),
    ('AUD00005', 'admin', '保存站点品牌配置', '127.0.0.1', now() - interval '185 minutes'),
    ('AUD00006', 'admin', '更新充值渠道 ch-usdt 费率', '127.0.0.1', now() - interval '222 minutes'),
    ('AUD00007', 'admin', '登录后台（Bearer Mock）', '127.0.0.1', now() - interval '259 minutes'),
    ('AUD00008', 'fin_approve', '驳回提现 WD00020', '127.0.0.1', now() - interval '296 minutes'),
    ('AUD00009', 'fin_payout', '维护模式开关：开启', '127.0.0.1', now() - interval '333 minutes'),
    ('AUD00010', 'admin', '编辑会员 演示用户A 资料', '10.0.0.12', now() - interval '370 minutes')
ON CONFLICT (id) DO NOTHING;

INSERT INTO admin_audit_logs (id, actor, action, ip, created_at)
SELECT
    'AUD' || LPAD(g.i::text, 5, '0'),
    (ARRAY['admin', 'admin', 'fin_approve', 'fin_payout'])[1 + (g.i % 4)],
    '审计演示动作 #' || g.i,
    CASE WHEN g.i % 5 = 0 THEN '10.0.0.12' ELSE '127.0.0.1' END,
    now() - (g.i * interval '37 minutes')
FROM generate_series(11, 50) AS g(i)
ON CONFLICT (id) DO NOTHING;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DELETE FROM admin_audit_logs WHERE id LIKE 'AUD%';
-- +goose StatementEnd
