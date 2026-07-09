-- +goose Up
-- +goose StatementBegin
CREATE TABLE cms_customer_service_agents (
    id          VARCHAR(64)  PRIMARY KEY,
    name        VARCHAR(128) NOT NULL,
    tg_link     TEXT         NOT NULL,
    work_hours  VARCHAR(256) NOT NULL DEFAULT '',
    sort        INT          NOT NULL DEFAULT 0,
    enabled     BOOLEAN      NOT NULL DEFAULT true,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT now()
);

COMMENT ON TABLE cms_customer_service_agents IS '会员端联系客服配置（Admin 维护）';
COMMENT ON COLUMN cms_customer_service_agents.tg_link IS 'Telegram 链接或 @用户名';
COMMENT ON COLUMN cms_customer_service_agents.work_hours IS '上班时间说明';

CREATE INDEX idx_cms_customer_service_agents_sort
    ON cms_customer_service_agents (enabled DESC, sort ASC, id ASC);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS cms_customer_service_agents;
-- +goose StatementEnd
