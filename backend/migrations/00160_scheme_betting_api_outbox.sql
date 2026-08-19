-- +goose Up
ALTER TABLE scheme_bet_outbox
    ADD COLUMN IF NOT EXISTS origin VARCHAR(16) NOT NULL DEFAULT 'scheme';

ALTER TABLE scheme_bet_outbox
    ALTER COLUMN decision_id DROP NOT NULL,
    ALTER COLUMN scheme_id DROP NOT NULL;

ALTER TABLE scheme_bet_outbox
    DROP CONSTRAINT IF EXISTS scheme_bet_outbox_origin_check;
ALTER TABLE scheme_bet_outbox
    ADD CONSTRAINT scheme_bet_outbox_origin_check
    CHECK (origin IN ('scheme', 'api'));

ALTER TABLE scheme_bet_outbox
    DROP CONSTRAINT IF EXISTS scheme_bet_outbox_origin_shape_check;
ALTER TABLE scheme_bet_outbox
    ADD CONSTRAINT scheme_bet_outbox_origin_shape_check
    CHECK (
        (origin = 'scheme' AND decision_id IS NOT NULL AND scheme_id IS NOT NULL)
        OR
        (
            origin = 'api'
            AND decision_id IS NULL
            AND scheme_id IS NULL
            AND member_id IS NOT NULL
            AND local_order_no IS NOT NULL
        )
    );

ALTER TABLE scheme_betting_admin_actions
    ALTER COLUMN scheme_id DROP NOT NULL;
ALTER TABLE scheme_betting_admin_actions
    DROP CONSTRAINT IF EXISTS scheme_betting_admin_actions_target_check;
ALTER TABLE scheme_betting_admin_actions
    ADD CONSTRAINT scheme_betting_admin_actions_target_check
    CHECK (scheme_id IS NOT NULL OR outbox_id IS NOT NULL);

CREATE INDEX IF NOT EXISTS idx_scheme_bet_outbox_origin_state
    ON scheme_bet_outbox (origin, state, safe_deadline_at, id);

-- +goose Down
DROP INDEX IF EXISTS idx_scheme_bet_outbox_origin_state;
DELETE FROM scheme_betting_admin_actions
WHERE scheme_id IS NULL;
DELETE FROM scheme_bet_outbox
WHERE origin = 'api';
ALTER TABLE scheme_betting_admin_actions
    DROP CONSTRAINT IF EXISTS scheme_betting_admin_actions_target_check;
ALTER TABLE scheme_betting_admin_actions
    ALTER COLUMN scheme_id SET NOT NULL;
ALTER TABLE scheme_bet_outbox
    DROP CONSTRAINT IF EXISTS scheme_bet_outbox_origin_shape_check;
ALTER TABLE scheme_bet_outbox
    DROP CONSTRAINT IF EXISTS scheme_bet_outbox_origin_check;
ALTER TABLE scheme_bet_outbox
    ALTER COLUMN decision_id SET NOT NULL,
    ALTER COLUMN scheme_id SET NOT NULL;
ALTER TABLE scheme_bet_outbox
    DROP COLUMN IF EXISTS origin;
