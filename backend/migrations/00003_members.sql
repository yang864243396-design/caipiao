-- +goose Up
-- +goose StatementBegin
CREATE TABLE members (
    id              BIGSERIAL PRIMARY KEY,
    member_no       VARCHAR(16)  NOT NULL,
    account         VARCHAR(32)  NOT NULL,
    password_hash   TEXT         NOT NULL,
    display_name    VARCHAR(64)  NOT NULL,
    status          VARCHAR(16)  NOT NULL DEFAULT 'active',
    l1_agent_code   VARCHAR(32),
    l2_agent_code   VARCHAR(32),
    registered_at   TIMESTAMPTZ  NOT NULL DEFAULT now(),
    last_login_at   TIMESTAMPTZ,
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),

    CONSTRAINT uq_members_member_no UNIQUE (member_no),
    CONSTRAINT uq_members_account UNIQUE (account),
    CONSTRAINT chk_members_status CHECK (status IN ('active', 'frozen'))
);

COMMENT ON TABLE members IS '会员账号主表：登录、资料、代理归属与账号状态';
COMMENT ON COLUMN members.id IS '内部主键';
COMMENT ON COLUMN members.member_no IS '会员业务编号，全局唯一（如 M00001），对外展示与关联用';
COMMENT ON COLUMN members.account IS '登录账号，全局唯一';
COMMENT ON COLUMN members.password_hash IS '登录密码哈希（bcrypt 等，禁止明文）';
COMMENT ON COLUMN members.display_name IS '会员展示昵称';
COMMENT ON COLUMN members.status IS '账号状态：active=正常，frozen=冻结';
COMMENT ON COLUMN members.l1_agent_code IS '一级代理编码，关联代理树';
COMMENT ON COLUMN members.l2_agent_code IS '二级代理编码，可为空';
COMMENT ON COLUMN members.registered_at IS '注册时间（UTC）';
COMMENT ON COLUMN members.last_login_at IS '最近登录时间（UTC）；未登录过为 NULL';
COMMENT ON COLUMN members.created_at IS '记录创建时间（UTC）';
COMMENT ON COLUMN members.updated_at IS '记录最后更新时间（UTC）';

CREATE INDEX idx_members_status_registered
    ON members (status, registered_at DESC);

COMMENT ON INDEX idx_members_status_registered IS '管理端按状态筛选会员列表';

CREATE INDEX idx_members_l1_agent
    ON members (l1_agent_code, registered_at DESC)
    WHERE l1_agent_code IS NOT NULL;

COMMENT ON INDEX idx_members_l1_agent IS '按一级代理查询下属会员';
-- +goose StatementEnd

-- +goose Down
DROP TABLE IF EXISTS members;
