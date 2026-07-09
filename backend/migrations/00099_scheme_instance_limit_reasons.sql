-- +goose Up
-- +goose StatementBegin
ALTER TABLE scheme_instances
    DROP CONSTRAINT IF EXISTS chk_scheme_instances_status_reason;

ALTER TABLE scheme_instances
    ADD CONSTRAINT chk_scheme_instances_status_reason CHECK (
        status_reason IN (
            '', 'manual', 'insufficient_funds', 'maintenance', 'end_time',
            'await_next_bet', 'cloud_active', 'bet_failed',
            'scheme_stop_loss', 'scheme_take_profit',
            'total_stop_loss', 'total_take_profit'
        )
    );
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE scheme_instances
    DROP CONSTRAINT IF EXISTS chk_scheme_instances_status_reason;

ALTER TABLE scheme_instances
    ADD CONSTRAINT chk_scheme_instances_status_reason CHECK (
        status_reason IN (
            '', 'manual', 'insufficient_funds', 'maintenance', 'end_time',
            'await_next_bet', 'cloud_active', 'bet_failed'
        )
    );
-- +goose StatementEnd
