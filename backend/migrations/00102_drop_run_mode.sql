-- +goose Up
-- +goose StatementBegin
-- cloud_bet_records: run_mode → sim_bet
ALTER TABLE cloud_bet_records
    ADD COLUMN IF NOT EXISTS sim_bet BOOLEAN;

DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = 'public'
          AND table_name = 'cloud_bet_records'
          AND column_name = 'run_mode'
    ) THEN
        UPDATE cloud_bet_records
        SET sim_bet = (run_mode = 'sim')
        WHERE sim_bet IS NULL;
    END IF;
END $$;

UPDATE cloud_bet_records
SET sim_bet = false
WHERE sim_bet IS NULL;

ALTER TABLE cloud_bet_records
    ALTER COLUMN sim_bet SET NOT NULL;

DROP INDEX IF EXISTS idx_cloud_bet_records_member_mode_placed;
DROP INDEX IF EXISTS idx_cloud_bet_records_member_scheme_placed;

CREATE INDEX IF NOT EXISTS idx_cloud_bet_records_member_sim_bet_placed
    ON cloud_bet_records (member_id, sim_bet, placed_at DESC);

CREATE INDEX IF NOT EXISTS idx_cloud_bet_records_member_scheme_sim_bet_placed
    ON cloud_bet_records (member_id, scheme_id, sim_bet, placed_at DESC);

ALTER TABLE cloud_bet_records DROP CONSTRAINT IF EXISTS chk_cloud_bet_records_mode;
ALTER TABLE cloud_bet_records DROP COLUMN IF EXISTS run_mode;

-- scheme_instances: 以 sim_bet 为准，删除 run_mode
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = 'public'
          AND table_name = 'scheme_instances'
          AND column_name = 'run_mode'
    ) THEN
        UPDATE scheme_instances
        SET sim_bet = (run_mode = 'sim')
        WHERE run_mode IS NOT NULL
          AND sim_bet IS DISTINCT FROM (run_mode = 'sim');
    END IF;
END $$;

ALTER TABLE scheme_instances DROP CONSTRAINT IF EXISTS chk_scheme_instances_run_mode;
ALTER TABLE scheme_instances DROP COLUMN IF EXISTS run_mode;

-- member_lookback_runtime: 00101 已切 PK 至 sim_bet
ALTER TABLE member_lookback_runtime DROP COLUMN IF EXISTS run_mode;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE member_lookback_runtime
    ADD COLUMN IF NOT EXISTS run_mode VARCHAR(8) NOT NULL DEFAULT 'real';

UPDATE member_lookback_runtime SET run_mode = CASE WHEN sim_bet THEN 'sim' ELSE 'real' END;

ALTER TABLE scheme_instances
    ADD COLUMN IF NOT EXISTS run_mode VARCHAR(8) NOT NULL DEFAULT 'real';

UPDATE scheme_instances SET run_mode = CASE WHEN sim_bet THEN 'sim' ELSE 'real' END;

ALTER TABLE scheme_instances
    ADD CONSTRAINT chk_scheme_instances_run_mode CHECK (run_mode IN ('real', 'sim'));

ALTER TABLE cloud_bet_records
    ADD COLUMN IF NOT EXISTS run_mode VARCHAR(8) NOT NULL DEFAULT 'real';

UPDATE cloud_bet_records SET run_mode = CASE WHEN sim_bet THEN 'sim' ELSE 'real' END;

ALTER TABLE cloud_bet_records
    ADD CONSTRAINT chk_cloud_bet_records_mode CHECK (run_mode IN ('real', 'sim'));

DROP INDEX IF EXISTS idx_cloud_bet_records_member_sim_bet_placed;
DROP INDEX IF EXISTS idx_cloud_bet_records_member_scheme_sim_bet_placed;

CREATE INDEX IF NOT EXISTS idx_cloud_bet_records_member_mode_placed
    ON cloud_bet_records (member_id, run_mode, placed_at DESC);

CREATE INDEX IF NOT EXISTS idx_cloud_bet_records_member_scheme_placed
    ON cloud_bet_records (member_id, scheme_id, run_mode, placed_at DESC);

ALTER TABLE cloud_bet_records DROP COLUMN IF EXISTS sim_bet;
-- +goose StatementEnd
