-- +goose Up
-- +goose StatementBegin
UPDATE cms_lobby_slots
SET title = '精密冠军赛 2024',
    brief = '决战巅峰，赢取丰厚赛季积分',
    enabled = true,
    updated_at = now()
WHERE slot_key = 'hero_main';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
UPDATE cms_lobby_slots
SET title = '大厅 Hero 主屏',
    brief = '轮播/主 CTA 文案占位',
    updated_at = now()
WHERE slot_key = 'hero_main';
-- +goose StatementEnd
