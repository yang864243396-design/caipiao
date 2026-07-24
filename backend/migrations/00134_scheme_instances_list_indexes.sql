-- +goose Up
-- +goose StatementBegin
-- 云端中心方案列表：会员维度按 updated_at/id 游标分页与 sim_bet 过滤。

CREATE INDEX IF NOT EXISTS idx_scheme_instances_member_updated_id
    ON scheme_instances (member_id, updated_at DESC, id DESC);

COMMENT ON INDEX idx_scheme_instances_member_updated_id IS '云端方案列表游标分页（member + updated_at + id）';

CREATE INDEX IF NOT EXISTS idx_scheme_instances_member_sim_updated_id
    ON scheme_instances (member_id, sim_bet, updated_at DESC, id DESC);

COMMENT ON INDEX idx_scheme_instances_member_sim_updated_id IS '云端方案列表按真实/模拟过滤的游标分页';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_scheme_instances_member_sim_updated_id;
DROP INDEX IF EXISTS idx_scheme_instances_member_updated_id;
-- +goose StatementEnd
