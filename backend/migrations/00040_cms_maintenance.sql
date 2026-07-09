-- +goose Up
-- +goose StatementBegin
CREATE TABLE cms_maintenance (
    id                    VARCHAR(64)  PRIMARY KEY,
    enabled               BOOLEAN      NOT NULL DEFAULT false,
    popup_announcement_id VARCHAR(64)  REFERENCES cms_announcements(id) ON DELETE SET NULL,
    title                 TEXT         NOT NULL DEFAULT '',
    message               TEXT         NOT NULL DEFAULT '',
    updated_at            TIMESTAMPTZ  NOT NULL DEFAULT now()
);

COMMENT ON TABLE cms_maintenance IS '全站维护开关（单例，id=default）';
COMMENT ON COLUMN cms_maintenance.popup_announcement_id IS '维护弹窗关联的已发布公告 id';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS cms_maintenance;
-- +goose StatementEnd
