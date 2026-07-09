-- +goose Up
-- +goose StatementBegin
CREATE TABLE lottery_catalog (
    code         VARCHAR(32)  PRIMARY KEY,
    display_name VARCHAR(64)  NOT NULL,
    detail_alias VARCHAR(64)  NOT NULL,
    sort_order   INT          NOT NULL DEFAULT 0,
    on_sale      BOOLEAN      NOT NULL DEFAULT true,
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ  NOT NULL DEFAULT now()
);

COMMENT ON TABLE lottery_catalog IS '彩种目录：code 创建后不可变，对外展示名与详情别名可运营';
COMMENT ON COLUMN lottery_catalog.code IS '对内稳定 code，如 tencent_ffc';
COMMENT ON COLUMN lottery_catalog.display_name IS '对外中文展示名';
COMMENT ON COLUMN lottery_catalog.detail_alias IS '详情页/深链别名';
COMMENT ON COLUMN lottery_catalog.sort_order IS '列表排序，升序';
COMMENT ON COLUMN lottery_catalog.on_sale IS '是否在售';

CREATE INDEX idx_lottery_catalog_sort ON lottery_catalog (sort_order ASC, code ASC);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS lottery_catalog;
-- +goose StatementEnd
