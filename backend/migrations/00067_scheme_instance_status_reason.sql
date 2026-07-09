-- +goose Up
-- +goose StatementBegin
ALTER TABLE scheme_instances
    ADD COLUMN status_reason VARCHAR(32) NOT NULL DEFAULT '';

ALTER TABLE scheme_instances
    ADD CONSTRAINT chk_scheme_instances_status_reason CHECK (
        status_reason IN ('', 'manual', 'insufficient_funds', 'maintenance', 'end_time')
    );

COMMENT ON COLUMN scheme_instances.status_reason IS 'paused 时的子原因：insufficient_funds / maintenance / end_time / manual';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE scheme_instances DROP CONSTRAINT IF EXISTS chk_scheme_instances_status_reason;
ALTER TABLE scheme_instances DROP COLUMN IF EXISTS status_reason;
-- +goose StatementEnd
