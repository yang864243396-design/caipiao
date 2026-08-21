-- +goose Up
ALTER TABLE scheme_period_decisions
  ADD COLUMN IF NOT EXISTS target_deadline_at TIMESTAMPTZ,
  ADD COLUMN IF NOT EXISTS target_period_no VARCHAR(64),
  ADD COLUMN IF NOT EXISTS failure_reason VARCHAR(64),
  ADD COLUMN IF NOT EXISTS shard_no INTEGER;

ALTER TABLE scheme_period_decisions
  DROP CONSTRAINT IF EXISTS scheme_period_decisions_shard_no_check;

ALTER TABLE scheme_period_decisions
  ADD CONSTRAINT scheme_period_decisions_shard_no_check
  CHECK (shard_no >= 0 AND shard_no <= 63);

ALTER TABLE scheme_instances
  ADD COLUMN IF NOT EXISTS chain_block_reason VARCHAR(64);

ALTER TABLE scheme_period_decisions
  DROP CONSTRAINT IF EXISTS scheme_period_decisions_status_check;

ALTER TABLE scheme_period_decisions
  ADD CONSTRAINT chk_scheme_period_decisions_status CHECK (
    status IN ('awaiting_target', 'completed', 'missed_contiguous_period',
               'blocked', 'duplicate', 'chain_broken')
  );

CREATE INDEX IF NOT EXISTS idx_scheme_period_decisions_awaiting_target
  ON scheme_period_decisions (lottery_code, status, target_deadline_at, id)
  WHERE status = 'awaiting_target';

CREATE INDEX IF NOT EXISTS idx_scheme_period_decisions_awaiting_target_shard
  ON scheme_period_decisions (shard_no, lottery_code, target_deadline_at, id)
  WHERE status = 'awaiting_target';

-- +goose Down
DO $$
BEGIN
  IF EXISTS (
    SELECT 1 FROM scheme_period_decisions
    WHERE status IN ('awaiting_target', 'missed_contiguous_period')
  ) THEN
    RAISE EXCEPTION 'cannot roll back migration 177 while contiguous decisions remain';
  END IF;
END $$;

DROP INDEX IF EXISTS idx_scheme_period_decisions_awaiting_target_shard;
DROP INDEX IF EXISTS idx_scheme_period_decisions_awaiting_target;
ALTER TABLE scheme_period_decisions
  DROP CONSTRAINT IF EXISTS chk_scheme_period_decisions_status;
ALTER TABLE scheme_period_decisions
  ADD CONSTRAINT scheme_period_decisions_status_check CHECK (
    status IN ('completed', 'blocked', 'duplicate', 'chain_broken')
  );
ALTER TABLE scheme_period_decisions
  DROP CONSTRAINT IF EXISTS scheme_period_decisions_shard_no_check;
ALTER TABLE scheme_period_decisions
  DROP COLUMN IF EXISTS failure_reason,
  DROP COLUMN IF EXISTS target_period_no,
  DROP COLUMN IF EXISTS target_deadline_at,
  DROP COLUMN IF EXISTS shard_no;
ALTER TABLE scheme_instances
  DROP COLUMN IF EXISTS chain_block_reason;
