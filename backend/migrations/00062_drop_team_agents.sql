-- +goose Up
-- +goose StatementBegin
-- 移除「代理/团队」模块：专属表、推广渠道配置、代理团队相关列与索引。
-- 保留 members.l1_agent_code / l2_agent_code（会员血缘字段，核心查询仍在用）。

DROP TABLE IF EXISTS team_promo_links;
DROP TABLE IF EXISTS cms_promo_channel;

DROP INDEX IF EXISTS idx_members_l1_agent_status;
DROP INDEX IF EXISTS idx_members_l2_agent_status;

ALTER TABLE members DROP CONSTRAINT IF EXISTS chk_members_commission_cap;
ALTER TABLE members DROP CONSTRAINT IF EXISTS chk_members_rebate_global;
ALTER TABLE members DROP CONSTRAINT IF EXISTS chk_members_rebate_ftc;
ALTER TABLE members DROP CONSTRAINT IF EXISTS chk_members_rebate_ssc;
ALTER TABLE members DROP CONSTRAINT IF EXISTS chk_members_quick_level;
ALTER TABLE members DROP CONSTRAINT IF EXISTS chk_members_team_role;

ALTER TABLE members DROP COLUMN IF EXISTS commission_cap_percent;
ALTER TABLE members DROP COLUMN IF EXISTS rebate_global;
ALTER TABLE members DROP COLUMN IF EXISTS rebate_ftc;
ALTER TABLE members DROP COLUMN IF EXISTS rebate_ssc;
ALTER TABLE members DROP COLUMN IF EXISTS quick_level;
ALTER TABLE members DROP COLUMN IF EXISTS owns_agent_code;
ALTER TABLE members DROP COLUMN IF EXISTS team_role;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- 回滚：恢复列（默认值）与代理团队表；不还原历史业务数据。
ALTER TABLE members
    ADD COLUMN IF NOT EXISTS team_role VARCHAR(16) NOT NULL DEFAULT 'member',
    ADD COLUMN IF NOT EXISTS owns_agent_code VARCHAR(32),
    ADD COLUMN IF NOT EXISTS quick_level INT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS rebate_ssc INT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS rebate_ftc INT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS rebate_global INT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS commission_cap_percent NUMERIC(5, 1) NOT NULL DEFAULT 40.0;

CREATE TABLE IF NOT EXISTS team_promo_links (
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
    CONSTRAINT fk_team_promo_links_owner FOREIGN KEY (owner_member_id) REFERENCES members (id) ON DELETE RESTRICT
);

CREATE TABLE IF NOT EXISTS cms_promo_channel (
    id                   VARCHAR(32) PRIMARY KEY DEFAULT 'default',
    default_material     TEXT        NOT NULL DEFAULT '',
    invite_code_required BOOLEAN     NOT NULL DEFAULT false,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT now()
);
-- +goose StatementEnd
