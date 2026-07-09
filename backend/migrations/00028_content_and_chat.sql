-- +goose Up
-- +goose StatementBegin
CREATE TABLE cms_announcements (
    id            VARCHAR(64)  PRIMARY KEY,
    title         VARCHAR(200) NOT NULL,
    status        VARCHAR(16)  NOT NULL DEFAULT 'published',
    published_at  TIMESTAMPTZ,
    body_html     TEXT         NOT NULL,
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT now(),

    CONSTRAINT chk_cms_announcements_status CHECK (status IN ('draft', 'published'))
);

COMMENT ON TABLE cms_announcements IS '平台公告（会员端列表/详情）';
COMMENT ON COLUMN cms_announcements.status IS 'draft=草稿 published=已发布';
COMMENT ON COLUMN cms_announcements.body_html IS '富文本正文 HTML';

CREATE INDEX idx_cms_announcements_published
    ON cms_announcements (published_at DESC NULLS LAST, id DESC)
    WHERE status = 'published';

CREATE TABLE cms_faq_articles (
    id         VARCHAR(64)  PRIMARY KEY,
    title      VARCHAR(200) NOT NULL,
    sort       INT          NOT NULL DEFAULT 0,
    body_html  TEXT         NOT NULL,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT now()
);

COMMENT ON TABLE cms_faq_articles IS '常见问题条目';

CREATE INDEX idx_cms_faq_articles_sort ON cms_faq_articles (sort ASC, id ASC);

CREATE TABLE cms_help_articles (
    id         VARCHAR(64)  PRIMARY KEY,
    title      VARCHAR(200) NOT NULL,
    sort       INT          NOT NULL DEFAULT 0,
    body_html  TEXT         NOT NULL,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT now()
);

COMMENT ON TABLE cms_help_articles IS '帮助中心折叠条目';

CREATE INDEX idx_cms_help_articles_sort ON cms_help_articles (sort ASC, id ASC);

CREATE TABLE member_announcement_reads (
    member_id       BIGINT       NOT NULL REFERENCES members(id) ON DELETE CASCADE,
    announcement_id VARCHAR(64)  NOT NULL REFERENCES cms_announcements(id) ON DELETE CASCADE,
    read_at         TIMESTAMPTZ  NOT NULL DEFAULT now(),
    PRIMARY KEY (member_id, announcement_id)
);

COMMENT ON TABLE member_announcement_reads IS '会员公告已读标记';

CREATE TABLE member_feedback (
    id         BIGSERIAL    PRIMARY KEY,
    member_id  BIGINT       NOT NULL REFERENCES members(id) ON DELETE CASCADE,
    subject    VARCHAR(80)  NOT NULL,
    content    TEXT         NOT NULL,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT now()
);

COMMENT ON TABLE member_feedback IS '会员意见回馈';

CREATE INDEX idx_member_feedback_member ON member_feedback (member_id, created_at DESC);

CREATE TABLE member_system_messages (
    id         BIGSERIAL PRIMARY KEY,
    member_id  BIGINT    NOT NULL REFERENCES members(id) ON DELETE CASCADE,
    body       TEXT      NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

COMMENT ON TABLE member_system_messages IS '会员系统讯息（聊天室 · 系统讯息 Tab）';

CREATE INDEX idx_member_system_messages_member
    ON member_system_messages (member_id, created_at DESC, id DESC);

CREATE TABLE member_chat_messages (
    id         BIGSERIAL PRIMARY KEY,
    member_id  BIGINT       NOT NULL REFERENCES members(id) ON DELETE CASCADE,
    peer_key   VARCHAR(64)  NOT NULL,
    direction  VARCHAR(8)   NOT NULL,
    body       TEXT         NOT NULL,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT now(),

    CONSTRAINT chk_member_chat_direction CHECK (direction IN ('in', 'out'))
);

COMMENT ON TABLE member_chat_messages IS '会员聊天会话消息';
COMMENT ON COLUMN member_chat_messages.peer_key IS '会话标识：service / superior / notice-deposit 等';
COMMENT ON COLUMN member_chat_messages.direction IS 'in=对方发来 out=会员发出';

CREATE INDEX idx_member_chat_messages_thread
    ON member_chat_messages (member_id, peer_key, created_at ASC, id ASC);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS member_chat_messages;
DROP TABLE IF EXISTS member_system_messages;
DROP TABLE IF EXISTS member_feedback;
DROP TABLE IF EXISTS member_announcement_reads;
DROP TABLE IF EXISTS cms_help_articles;
DROP TABLE IF EXISTS cms_faq_articles;
DROP TABLE IF EXISTS cms_announcements;
-- +goose StatementEnd
