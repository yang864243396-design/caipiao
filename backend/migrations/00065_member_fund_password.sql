-- +goose Up
-- +goose StatementBegin
ALTER TABLE members
    ADD COLUMN fund_password_hash TEXT;

COMMENT ON COLUMN members.fund_password_hash IS '资金密码哈希（bcrypt）；NULL 表示尚未设置，未设置不可真实投注';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE members DROP COLUMN IF EXISTS fund_password_hash;
-- +goose StatementEnd
