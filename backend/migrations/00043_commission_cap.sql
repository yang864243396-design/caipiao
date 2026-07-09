-- +goose Up
-- +goose StatementBegin
ALTER TABLE members
    ADD COLUMN commission_cap_percent NUMERIC(5, 1) NOT NULL DEFAULT 40.0;

ALTER TABLE members
    ADD CONSTRAINT chk_members_commission_cap CHECK (
        commission_cap_percent >= 0 AND commission_cap_percent <= 100
    );

COMMENT ON COLUMN members.commission_cap_percent IS '代理分成上限（%）；非代理使用默认值';

UPDATE members
SET commission_cap_percent = 45.0
WHERE owns_agent_code = 'AGT-L1-JIA';

UPDATE members
SET commission_cap_percent = 30.0
WHERE owns_agent_code = 'AGT-L2-YI';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE members DROP CONSTRAINT IF EXISTS chk_members_commission_cap;
ALTER TABLE members DROP COLUMN IF EXISTS commission_cap_percent;
-- +goose StatementEnd
