-- +goose Up
-- +goose StatementBegin
INSERT INTO cms_maintenance (id, enabled, popup_announcement_id, title, message) VALUES
    ('default', false, NULL, '', '');
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DELETE FROM cms_maintenance WHERE id = 'default';
-- +goose StatementEnd
