-- +goose Up
-- +goose StatementBegin
ALTER TABLE scheme_instances
    ADD COLUMN round_index INT NOT NULL DEFAULT 0,
    ADD COLUMN last_settled_issue VARCHAR(32);

COMMENT ON COLUMN scheme_instances.round_index IS '倍投轮次索引（0-based，对应 config.rounds）';
COMMENT ON COLUMN scheme_instances.last_settled_issue IS 'Worker 已结算的最后一期官方期号';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE scheme_instances
    DROP COLUMN IF EXISTS last_settled_issue,
    DROP COLUMN IF EXISTS round_index;
-- +goose StatementEnd
