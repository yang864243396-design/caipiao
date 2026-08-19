-- +goose Up
ALTER TABLE scheme_betting_admin_actions
    ADD COLUMN IF NOT EXISTS actor_account VARCHAR(128);

UPDATE scheme_betting_admin_actions
SET actor_account = COALESCE(actor_account, 'legacy-admin')
WHERE actor_account IS NULL;

ALTER TABLE scheme_betting_admin_actions
    ALTER COLUMN actor_account SET NOT NULL;
-- +goose StatementBegin

CREATE OR REPLACE FUNCTION reject_scheme_betting_admin_action_mutation()
RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    RAISE EXCEPTION 'scheme_betting_admin_actions is append-only';
END;
$$;
-- +goose StatementEnd

DROP TRIGGER IF EXISTS trg_scheme_betting_admin_actions_append_only ON scheme_betting_admin_actions;
CREATE TRIGGER trg_scheme_betting_admin_actions_append_only
BEFORE UPDATE OR DELETE ON scheme_betting_admin_actions
FOR EACH ROW EXECUTE FUNCTION reject_scheme_betting_admin_action_mutation();

-- +goose Down
DROP TRIGGER IF EXISTS trg_scheme_betting_admin_actions_append_only ON scheme_betting_admin_actions;
DROP FUNCTION IF EXISTS reject_scheme_betting_admin_action_mutation();
ALTER TABLE scheme_betting_admin_actions DROP COLUMN IF EXISTS actor_account;
