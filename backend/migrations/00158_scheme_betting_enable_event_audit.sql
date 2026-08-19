-- +goose Up
ALTER TABLE scheme_betting_admin_actions
    DROP CONSTRAINT IF EXISTS scheme_betting_admin_actions_action_check;
ALTER TABLE scheme_betting_admin_actions
    ADD CONSTRAINT scheme_betting_admin_actions_action_check
    CHECK (action IN ('enable_event', 'rearm', 'cancel', 'resolve_unknown'));

-- +goose Down
ALTER TABLE scheme_betting_admin_actions
    DROP CONSTRAINT IF EXISTS scheme_betting_admin_actions_action_check;
ALTER TABLE scheme_betting_admin_actions
    ADD CONSTRAINT scheme_betting_admin_actions_action_check
    CHECK (action IN ('rearm', 'cancel', 'resolve_unknown')) NOT VALID;
