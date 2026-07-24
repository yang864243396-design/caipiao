-- +goose Up
-- +goose StatementBegin
-- 投注记录查询去 JOIN：明细表冗余币种/彩种/方案定义，列表与汇总只扫本表。

ALTER TABLE cloud_bet_records
    ADD COLUMN IF NOT EXISTS currency VARCHAR(8) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS lottery_code VARCHAR(32) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS lottery_label VARCHAR(64) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS definition_id VARCHAR(64) NOT NULL DEFAULT '';

COMMENT ON COLUMN cloud_bet_records.currency IS '方案币种冗余（USDT/TRX/CNY），会员投注记录汇总/筛选用';
COMMENT ON COLUMN cloud_bet_records.lottery_code IS '彩种编码冗余，筛选用（方案删除后仍可查）';
COMMENT ON COLUMN cloud_bet_records.lottery_label IS '彩种展示名冗余';
COMMENT ON COLUMN cloud_bet_records.definition_id IS '方案定义 ID 冗余，按方案筛选无需 JOIN instance';

UPDATE cloud_bet_records c
SET lottery_code = COALESCE(NULLIF(TRIM(si.lottery_code), ''), c.lottery_code),
    lottery_label = COALESCE(NULLIF(TRIM(si.lottery_label), ''), c.lottery_label),
    definition_id = COALESCE(NULLIF(TRIM(si.definition_id), ''), c.definition_id)
FROM scheme_instances si
WHERE si.id = c.scheme_id;

UPDATE cloud_bet_records c
SET currency = UPPER(COALESCE(NULLIF(TRIM(bo.currency), ''), c.currency)),
    lottery_code = CASE
        WHEN c.lottery_code = '' THEN COALESCE(NULLIF(TRIM(bo.lottery_code), ''), '')
        ELSE c.lottery_code
    END,
    lottery_label = CASE
        WHEN c.lottery_label = '' THEN COALESCE(NULLIF(TRIM(bo.lottery_name), ''), '')
        ELSE c.lottery_label
    END
FROM bet_orders bo
WHERE c.bet_order_no IS NOT NULL
  AND c.bet_order_no <> ''
  AND bo.order_no = c.bet_order_no;

CREATE INDEX IF NOT EXISTS idx_cloud_bet_records_member_def_sim_placed
    ON cloud_bet_records (member_id, definition_id, sim_bet, placed_at DESC)
    WHERE definition_id <> '';

COMMENT ON INDEX idx_cloud_bet_records_member_def_sim_placed IS '按方案定义筛投注记录（无 JOIN instance）';

CREATE INDEX IF NOT EXISTS idx_cloud_bet_records_member_lottery_sim_placed
    ON cloud_bet_records (member_id, lottery_code, sim_bet, placed_at DESC)
    WHERE lottery_code <> '';

COMMENT ON INDEX idx_cloud_bet_records_member_lottery_sim_placed IS '按彩种筛投注记录';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_cloud_bet_records_member_lottery_sim_placed;
DROP INDEX IF EXISTS idx_cloud_bet_records_member_def_sim_placed;
ALTER TABLE cloud_bet_records
    DROP COLUMN IF EXISTS definition_id,
    DROP COLUMN IF EXISTS lottery_label,
    DROP COLUMN IF EXISTS lottery_code,
    DROP COLUMN IF EXISTS currency;
-- +goose StatementEnd
