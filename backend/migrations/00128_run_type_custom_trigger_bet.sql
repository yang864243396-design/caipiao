-- +goose Up
-- +goose StatementBegin
-- 运行类型新增「自定义开某投某」（触发玩法≠投注玩法），追加到 run_types 枚举。
-- 插入位置紧随「高级开某投某」之后，保持展示顺序。
UPDATE lottery_scheme_option_sets
SET run_types = '[
        {"value":"fixed_rotate","label":"定码轮换"},
        {"value":"adv_fixed_rotate","label":"高级定码轮换"},
        {"value":"adv_trigger_bet","label":"高级开某投某"},
        {"value":"custom_trigger_bet","label":"自定义开某投某"},
        {"value":"hot_cold_warm","label":"冷热温出号"},
        {"value":"random_draw","label":"随机出号"},
        {"value":"builtin_plan","label":"内置计画"},
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
        {"value":"fixed_number","label":"固定号码"}
    ]'::jsonb,
    updated_at = now();
-- +goose StatementEnd
