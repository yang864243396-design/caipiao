-- +goose Up
-- +goose StatementBegin
INSERT INTO scheme_templates (id, name, lottery_code, brief, sort_order, enabled, created_at, updated_at) VALUES
    ('scheme_demo_1001', '两期中跟挂停（附录演示）', 'tencent_ffc', '平台预置演示模板', 10, true, now(), now()),
    ('tpl_demo_wave_3', '三期推波方案', 'cq_ssc', '三期推波结构示例', 20, true, now(), now()),
    ('tpl_demo_plan_4', '四期倍投计划', 'fc_3d', '四期计划表示例', 30, true, now(), now()),
    ('tpl_demo_plan_6', '六期倍投方案', 'tencent_ffc', '六期倍投表示例', 40, true, now(), now())
ON CONFLICT (id) DO NOTHING;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DELETE FROM scheme_templates WHERE id IN (
    'scheme_demo_1001', 'tpl_demo_wave_3', 'tpl_demo_plan_4', 'tpl_demo_plan_6'
);
-- +goose StatementEnd
