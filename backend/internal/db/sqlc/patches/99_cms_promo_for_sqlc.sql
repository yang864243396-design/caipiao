-- sqlc 专用：00062 会 DROP cms_promo_channel，但 content.sql 仍查询该表。
-- 本文件仅列入 sqlc.yaml，不参与 goose 迁移。

CREATE TABLE IF NOT EXISTS cms_promo_channel (
    id                    VARCHAR(32)  PRIMARY KEY DEFAULT 'default',
    default_material      TEXT         NOT NULL DEFAULT '',
    invite_code_required  BOOLEAN      NOT NULL DEFAULT true,
    updated_at            TIMESTAMPTZ  NOT NULL DEFAULT now()
);

COMMENT ON TABLE cms_promo_channel IS '推广与渠道全局配置（单例 id=default）';
