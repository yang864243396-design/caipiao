-- +goose Up
-- +goose StatementBegin
ALTER TABLE scheme_instances
    DROP CONSTRAINT IF EXISTS chk_scheme_instances_status_reason;

ALTER TABLE scheme_instances
    ADD CONSTRAINT chk_scheme_instances_status_reason CHECK (
        status_reason IN (
            '', 'manual', 'insufficient_funds', 'maintenance', 'end_time',
            'await_next_bet', 'cloud_active'
        )
    );

COMMENT ON COLUMN scheme_instances.status_reason IS 'paused/run 子状态：await_next_bet 下期投注 / cloud_active 云端挂机中 等';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE scheme_instances
    DROP CONSTRAINT IF EXISTS chk_scheme_instances_status_reason;

ALTER TABLE scheme_instances
    ADD CONSTRAINT chk_scheme_instances_status_reason CHECK (
        status_reason IN ('', 'manual', 'insufficient_funds', 'maintenance', 'end_time')
    );
-- +goose StatementEnd
