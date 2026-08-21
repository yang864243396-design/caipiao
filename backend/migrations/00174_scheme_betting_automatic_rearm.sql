-- +goose NO TRANSACTION
-- +goose Up
-- Runtime automatic rearm is restricted to terminal commands that are proven
-- not to have reached the provider. Ambiguous and accepted outcomes are never
-- admitted by the matching recovery queries. The existing
-- idx_scheme_bet_outbox_scheme_detail index already serves each latest-command
-- lookup, so only blocked instances need a new partial index.
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_scheme_instances_safe_rearm
    ON scheme_instances (updated_at, id)
    INCLUDE (lottery_code)
    WHERE betting_owner = 'event'
      AND status = 'running'
      AND strict_chain_state = 'blocked_requires_rearm';

-- +goose Down
DROP INDEX CONCURRENTLY IF EXISTS idx_scheme_instances_safe_rearm;
