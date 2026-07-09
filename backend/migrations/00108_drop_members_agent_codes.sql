-- +goose Up
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_members_l1_agent;
ALTER TABLE members DROP COLUMN IF EXISTS l1_agent_code;
ALTER TABLE members DROP COLUMN IF EXISTS l2_agent_code;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE members ADD COLUMN IF NOT EXISTS l1_agent_code VARCHAR(32);
ALTER TABLE members ADD COLUMN IF NOT EXISTS l2_agent_code VARCHAR(32);
CREATE INDEX IF NOT EXISTS idx_members_l1_agent
    ON members (l1_agent_code, registered_at DESC)
    WHERE l1_agent_code IS NOT NULL;
-- +goose StatementEnd
