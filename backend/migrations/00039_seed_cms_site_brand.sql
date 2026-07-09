-- +goose Up
-- +goose StatementBegin
INSERT INTO cms_site_brand (id, site_name, logo_url, tagline) VALUES
    ('default', '精密终端 · 演示站', 'https://placehold.co/120x40/0066ff/ffffff?text=LOGO', '数字精算主义 · 管理端 Mock');
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DELETE FROM cms_site_brand WHERE id = 'default';
-- +goose StatementEnd
