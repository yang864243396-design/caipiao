-- +goose Up
-- +goose StatementBegin
INSERT INTO cms_promo_channel (id, default_material, invite_code_required)
VALUES ('default', '推广素材包 v1（演示）', true)
ON CONFLICT (id) DO NOTHING;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DELETE FROM cms_promo_channel WHERE id = 'default';
-- +goose StatementEnd
