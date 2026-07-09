-- +goose Up
-- +goose StatementBegin
CREATE TABLE cms_promo_channel (
    id                    VARCHAR(32)  PRIMARY KEY DEFAULT 'default',
    default_material      TEXT         NOT NULL DEFAULT '',
    invite_code_required  BOOLEAN      NOT NULL DEFAULT true,
    updated_at            TIMESTAMPTZ  NOT NULL DEFAULT now()
);

COMMENT ON TABLE cms_promo_channel IS '推广与渠道全局配置（单例 id=default）';
COMMENT ON COLUMN cms_promo_channel.default_material IS '默认推广文案/素材';
COMMENT ON COLUMN cms_promo_channel.invite_code_required IS '二级开户是否须邀请码';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS cms_promo_channel;
-- +goose StatementEnd
