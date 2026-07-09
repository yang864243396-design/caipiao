-- +goose Up
-- +goose StatementBegin
ALTER TABLE members
    ADD COLUMN team_role VARCHAR(16) NOT NULL DEFAULT 'member',
    ADD COLUMN owns_agent_code VARCHAR(32);

ALTER TABLE members
    ADD CONSTRAINT chk_members_team_role CHECK (
        team_role IN ('member', 'agent_l1', 'agent_l2')
    );

COMMENT ON COLUMN members.team_role IS '团队身份：member=普通会员，agent_l1=一级代理，agent_l2=二级代理';
COMMENT ON COLUMN members.owns_agent_code IS '代理自身编码；非代理为 NULL';

UPDATE members
SET team_role = 'agent_l1',
    owns_agent_code = 'AGT-L1-JIA'
WHERE member_no = 'M00001';

UPDATE members
SET team_role = 'agent_l2',
    owns_agent_code = 'AGT-L2-YI'
WHERE member_no = 'M00003';

CREATE INDEX idx_members_l1_agent_status
    ON members (l1_agent_code, status, registered_at DESC)
    WHERE l1_agent_code IS NOT NULL;

COMMENT ON INDEX idx_members_l1_agent_status IS '一级代理团队下级列表';

CREATE INDEX idx_members_l2_agent_status
    ON members (l2_agent_code, status, registered_at DESC)
    WHERE l2_agent_code IS NOT NULL;

COMMENT ON INDEX idx_members_l2_agent_status IS '二级代理团队下级列表';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_members_l2_agent_status;
DROP INDEX IF EXISTS idx_members_l1_agent_status;
ALTER TABLE members DROP CONSTRAINT IF EXISTS chk_members_team_role;
ALTER TABLE members DROP COLUMN IF EXISTS owns_agent_code;
ALTER TABLE members DROP COLUMN IF EXISTS team_role;
-- +goose StatementEnd
