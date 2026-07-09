-- +goose Up
-- +goose StatementBegin
ALTER TABLE member_lookback_settings
    ADD COLUMN IF NOT EXISTS apply_formal BOOLEAN NOT NULL DEFAULT false;

ALTER TABLE member_lookback_settings
    ADD COLUMN IF NOT EXISTS apply_sim BOOLEAN NOT NULL DEFAULT false;

DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = 'public'
          AND table_name = 'member_lookback_settings'
          AND column_name = 'run_mode'
    ) THEN
        UPDATE member_lookback_settings
        SET apply_formal = (run_mode LIKE '%real%'),
            apply_sim    = (run_mode LIKE '%sim%');
    END IF;
END $$;

COMMENT ON COLUMN member_lookback_settings.apply_formal IS '正式通道(simBet=false)是否启用回头';
COMMENT ON COLUMN member_lookback_settings.apply_sim IS '模拟通道(simBet=true)是否启用回头';

ALTER TABLE member_lookback_runtime
    ADD COLUMN IF NOT EXISTS sim_bet BOOLEAN;

DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = 'public'
          AND table_name = 'member_lookback_runtime'
          AND column_name = 'run_mode'
    ) THEN
        UPDATE member_lookback_runtime
        SET sim_bet = (run_mode = 'sim')
        WHERE sim_bet IS NULL;
    END IF;
END $$;

UPDATE member_lookback_runtime
SET sim_bet = false
WHERE sim_bet IS NULL;

ALTER TABLE member_lookback_runtime
    ALTER COLUMN sim_bet SET NOT NULL;

ALTER TABLE member_lookback_runtime
    ADD COLUMN IF NOT EXISTS total_hit_count INT NOT NULL DEFAULT 0;

COMMENT ON COLUMN member_lookback_runtime.total_hit_count IS '跨期累计中奖次数（整体几回头）';

ALTER TABLE member_lookback_runtime
    DROP CONSTRAINT IF EXISTS member_lookback_runtime_pkey;

ALTER TABLE member_lookback_runtime
    ADD PRIMARY KEY (member_id, sim_bet);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE member_lookback_runtime
    DROP CONSTRAINT IF EXISTS member_lookback_runtime_pkey;

ALTER TABLE member_lookback_runtime
    ADD PRIMARY KEY (member_id, run_mode);

ALTER TABLE member_lookback_runtime
    DROP COLUMN IF EXISTS total_hit_count;

ALTER TABLE member_lookback_runtime
    DROP COLUMN IF EXISTS sim_bet;

ALTER TABLE member_lookback_settings
    DROP COLUMN IF EXISTS apply_sim;

ALTER TABLE member_lookback_settings
    DROP COLUMN IF EXISTS apply_formal;
-- +goose StatementEnd
