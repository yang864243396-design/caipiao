-- +goose Up
-- +goose StatementBegin
DELETE FROM cms_lobby_slots
WHERE slot_key IN ('bento_1', 'bento_2', 'news_strip');
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
INSERT INTO cms_lobby_slots (id, slot_key, title, brief, sort, enabled) VALUES
    ('L2', 'bento_1', 'Bento 入口 1', '跟单大厅', 2, true),
    ('L3', 'bento_2', 'Bento 入口 2', '云端中心', 3, true),
    ('L4', 'news_strip', '最新动态', '', 4, true)
ON CONFLICT (id) DO NOTHING;
-- +goose StatementEnd
