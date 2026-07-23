-- +goose Up
-- +goose StatementBegin
-- 运行类型展示名：冷热温出号→冷热出号，内置计画→内置计划，固定取码→固定号码
UPDATE lottery_scheme_option_sets
SET run_types = '[
        {"value":"fixed_rotate","label":"定码轮换"},
        {"value":"adv_fixed_rotate","label":"高级定码轮换"},
        {"value":"adv_trigger_bet","label":"高级开某投某"},
        {"value":"hot_cold_warm","label":"冷热出号"},
        {"value":"random_draw","label":"随机出号"},
        {"value":"builtin_plan","label":"内置计划"},
        {"value":"fixed_number","label":"固定号码"}
    ]'::jsonb,
    updated_at = now();
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
UPDATE lottery_scheme_option_sets
SET run_types = '[
        {"value":"fixed_rotate","label":"定码轮换"},
        {"value":"adv_fixed_rotate","label":"高级定码轮换"},
        {"value":"adv_trigger_bet","label":"高级开某投某"},
        {"value":"hot_cold_warm","label":"冷热温出号"},
        {"value":"random_draw","label":"随机出号"},
        {"value":"builtin_plan","label":"内置计画"},
        {"value":"fixed_number","label":"固定取码"}
    ]'::jsonb,
    updated_at = now();
-- +goose StatementEnd
