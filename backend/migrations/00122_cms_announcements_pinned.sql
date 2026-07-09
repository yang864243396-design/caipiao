-- +goose Up
-- +goose StatementBegin
ALTER TABLE cms_announcements
    ADD COLUMN IF NOT EXISTS pinned BOOLEAN NOT NULL DEFAULT false;

COMMENT ON COLUMN cms_announcements.pinned IS '置顶公告；会员端 Banner 下展示，全局仅允许一条';

CREATE UNIQUE INDEX IF NOT EXISTS idx_cms_announcements_one_pinned
    ON cms_announcements (pinned)
    WHERE pinned = true;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_cms_announcements_one_pinned;
ALTER TABLE cms_announcements DROP COLUMN IF EXISTS pinned;
-- +goose StatementEnd
