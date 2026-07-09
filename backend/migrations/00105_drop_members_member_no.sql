-- +goose Up
-- +goose StatementBegin
ALTER TABLE members DROP CONSTRAINT IF EXISTS uq_members_member_no;
ALTER TABLE members DROP COLUMN IF EXISTS member_no;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE members ADD COLUMN IF NOT EXISTS member_no VARCHAR(16);
-- +goose StatementEnd
