-- +goose Up
-- +goose StatementBegin
ALTER TABLE cms_banners
    ADD COLUMN IF NOT EXISTS link_url TEXT NOT NULL DEFAULT '';

COMMENT ON COLUMN cms_banners.link_url IS '点击跳转外链；为空则会员端 Banner 不可点击';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE cms_banners DROP COLUMN IF EXISTS link_url;
-- +goose StatementEnd
