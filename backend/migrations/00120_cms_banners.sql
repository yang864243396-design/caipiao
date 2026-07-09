-- +goose Up
-- +goose StatementBegin
CREATE TABLE cms_banners (
    id         VARCHAR(64)  PRIMARY KEY,
    remark     VARCHAR(255) NOT NULL DEFAULT '',
    image_url  TEXT         NOT NULL,
    sort       INT          NOT NULL DEFAULT 0,
    enabled    BOOLEAN      NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ  NOT NULL DEFAULT now()
);

COMMENT ON TABLE cms_banners IS '游戏大厅主屏 Banner';
COMMENT ON COLUMN cms_banners.remark IS 'Banner 备注（仅后台展示）';
COMMENT ON COLUMN cms_banners.image_url IS 'Banner 图片 URL';
COMMENT ON COLUMN cms_banners.sort IS '排序，越小越靠前';

CREATE INDEX idx_cms_banners_admin ON cms_banners (sort ASC, created_at DESC, id DESC);
CREATE INDEX idx_cms_banners_public ON cms_banners (sort ASC, id ASC) WHERE enabled = true;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS cms_banners;
-- +goose StatementEnd
