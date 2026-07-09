-- +goose Up
-- +goose StatementBegin
ALTER TABLE members
    ADD COLUMN quick_level   INT NOT NULL DEFAULT 0,
    ADD COLUMN rebate_ssc    INT NOT NULL DEFAULT 0,
    ADD COLUMN rebate_ftc    INT NOT NULL DEFAULT 0,
    ADD COLUMN rebate_global INT NOT NULL DEFAULT 0;

ALTER TABLE members
    ADD CONSTRAINT chk_members_quick_level CHECK (quick_level >= 0 AND quick_level <= 9),
    ADD CONSTRAINT chk_members_rebate_ssc CHECK (rebate_ssc >= 0 AND rebate_ssc <= 1800),
    ADD CONSTRAINT chk_members_rebate_ftc CHECK (rebate_ftc >= 0 AND rebate_ftc <= 1800),
    ADD CONSTRAINT chk_members_rebate_global CHECK (rebate_global >= 0 AND rebate_global <= 1800);

COMMENT ON COLUMN members.quick_level IS '快捷开户档位 0–9';
COMMENT ON COLUMN members.rebate_ssc IS '时时彩返点（千分比，如 68 表示 6.8%）';
COMMENT ON COLUMN members.rebate_ftc IS '分分彩返点（千分比）';
COMMENT ON COLUMN members.rebate_global IS '全彩种返点（千分比）';

CREATE TABLE team_promo_links (
    id              BIGSERIAL PRIMARY KEY,
    owner_member_id BIGINT       NOT NULL,
    link_code       VARCHAR(16)  NOT NULL,
    title           VARCHAR(128) NOT NULL,
    user_kind       VARCHAR(16)  NOT NULL,
    rebate_ssc      INT          NOT NULL DEFAULT 0,
    rebate_ftc      INT          NOT NULL DEFAULT 0,
    rebate_global   INT          NOT NULL DEFAULT 0,
    remarks         TEXT,
    click_count     BIGINT       NOT NULL DEFAULT 0,
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),

    CONSTRAINT uq_team_promo_links_code UNIQUE (link_code),
    CONSTRAINT fk_team_promo_links_owner FOREIGN KEY (owner_member_id) REFERENCES members (id) ON DELETE RESTRICT,
    CONSTRAINT chk_team_promo_links_user_kind CHECK (user_kind IN ('member', 'agent')),
    CONSTRAINT chk_team_promo_links_rebate_ssc CHECK (rebate_ssc >= 0 AND rebate_ssc <= 1800),
    CONSTRAINT chk_team_promo_links_rebate_ftc CHECK (rebate_ftc >= 0 AND rebate_ftc <= 1800),
    CONSTRAINT chk_team_promo_links_rebate_global CHECK (rebate_global >= 0 AND rebate_global <= 1800)
);

COMMENT ON TABLE team_promo_links IS '代理推广链接：开户/拉新用短链配置';
COMMENT ON COLUMN team_promo_links.id IS '主键';
COMMENT ON COLUMN team_promo_links.owner_member_id IS '创建者会员 ID（须为代理）';
COMMENT ON COLUMN team_promo_links.link_code IS '短链码，全局唯一';
COMMENT ON COLUMN team_promo_links.title IS '链接标题/备注摘要';
COMMENT ON COLUMN team_promo_links.user_kind IS '目标用户类型：member 或 agent';
COMMENT ON COLUMN team_promo_links.rebate_ssc IS '时时彩返点（千分比）';
COMMENT ON COLUMN team_promo_links.rebate_ftc IS '分分彩返点（千分比）';
COMMENT ON COLUMN team_promo_links.rebate_global IS '全彩种返点（千分比）';
COMMENT ON COLUMN team_promo_links.remarks IS '备注';
COMMENT ON COLUMN team_promo_links.click_count IS '点击次数（演示计数）';
COMMENT ON COLUMN team_promo_links.created_at IS '创建时间（UTC）';
COMMENT ON COLUMN team_promo_links.updated_at IS '最后更新时间（UTC）';

CREATE INDEX idx_team_promo_links_owner_created
    ON team_promo_links (owner_member_id, created_at DESC);

COMMENT ON INDEX idx_team_promo_links_owner_created IS '代理推广链接列表';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS team_promo_links;
ALTER TABLE members DROP CONSTRAINT IF EXISTS chk_members_rebate_global;
ALTER TABLE members DROP CONSTRAINT IF EXISTS chk_members_rebate_ftc;
ALTER TABLE members DROP CONSTRAINT IF EXISTS chk_members_rebate_ssc;
ALTER TABLE members DROP CONSTRAINT IF EXISTS chk_members_quick_level;
ALTER TABLE members DROP COLUMN IF EXISTS rebate_global;
ALTER TABLE members DROP COLUMN IF EXISTS rebate_ftc;
ALTER TABLE members DROP COLUMN IF EXISTS rebate_ssc;
ALTER TABLE members DROP COLUMN IF EXISTS quick_level;
-- +goose StatementEnd
