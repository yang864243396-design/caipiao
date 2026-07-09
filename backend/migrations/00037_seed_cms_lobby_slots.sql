-- +goose Up
-- +goose StatementBegin
INSERT INTO cms_lobby_slots (id, slot_key, title, brief, sort, enabled) VALUES
    ('L1', 'hero_main', '精密冠军赛 2024', '决战巅峰，赢取丰厚赛季积分', 1, true)
ON CONFLICT (id) DO NOTHING;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DELETE FROM cms_lobby_slots WHERE id IN ('L1');
-- +goose StatementEnd
