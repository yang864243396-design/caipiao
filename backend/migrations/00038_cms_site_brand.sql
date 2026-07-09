-- +goose Up
-- +goose StatementBegin
CREATE TABLE cms_site_brand (
    id         VARCHAR(64)  PRIMARY KEY,
    site_name  VARCHAR(128) NOT NULL,
    logo_url   TEXT         NOT NULL DEFAULT '',
    tagline    TEXT         NOT NULL DEFAULT '',
    updated_at TIMESTAMPTZ  NOT NULL DEFAULT now()
);

COMMENT ON TABLE cms_site_brand IS '站点品牌配置（单例，id=default）';
COMMENT ON COLUMN cms_site_brand.site_name IS '站点名称';
COMMENT ON COLUMN cms_site_brand.logo_url IS 'Logo 图片 URL';
COMMENT ON COLUMN cms_site_brand.tagline IS '一句话介绍';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS cms_site_brand;
-- +goose StatementEnd
