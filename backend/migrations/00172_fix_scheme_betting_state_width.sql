-- +goose Up
-- external_acceptance_unknown is 27 characters and is a valid formal state.
ALTER TABLE scheme_bet_outbox
    ALTER COLUMN state TYPE VARCHAR(32);

ALTER TABLE scheme_bet_attempts
    ALTER COLUMN outcome TYPE VARCHAR(32);

-- +goose Down
-- A safe downgrade is intentionally unsupported: a live formal row may hold
-- external_acceptance_unknown, which cannot be represented by VARCHAR(24).
