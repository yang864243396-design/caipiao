-- +goose Up
-- 同一方案同一期仅允许一条 cloud 投注明细；real 模式挂第三方注单号，pending 待派奖同步。

-- 清理历史重复数据，保留每期最早一条。
DELETE FROM cloud_bet_records c
USING cloud_bet_records d
WHERE c.scheme_id = d.scheme_id
  AND c.period_no = d.period_no
  AND c.id > d.id;

ALTER TABLE cloud_bet_records DROP CONSTRAINT IF EXISTS chk_cloud_bet_records_status;
ALTER TABLE cloud_bet_records ADD CONSTRAINT chk_cloud_bet_records_status
    CHECK (status IN ('pending', 'hit', 'miss'));

ALTER TABLE cloud_bet_records
    ADD COLUMN IF NOT EXISTS third_party_bet_id VARCHAR(64),
    ADD COLUMN IF NOT EXISTS bet_order_no VARCHAR(64);

COMMENT ON COLUMN cloud_bet_records.third_party_bet_id IS '第三方 web_bets 注单编号（real）';
COMMENT ON COLUMN cloud_bet_records.bet_order_no IS '本平台 bet_orders.order_no，派奖同步回写 cloud 状态';

CREATE UNIQUE INDEX IF NOT EXISTS uq_cloud_bet_records_scheme_period
    ON cloud_bet_records (scheme_id, period_no);

CREATE INDEX IF NOT EXISTS idx_cloud_bet_records_third_party_bet_id
    ON cloud_bet_records (third_party_bet_id)
    WHERE third_party_bet_id IS NOT NULL AND third_party_bet_id <> '';

-- +goose Down
DROP INDEX IF EXISTS idx_cloud_bet_records_third_party_bet_id;
DROP INDEX IF EXISTS uq_cloud_bet_records_scheme_period;
ALTER TABLE cloud_bet_records DROP COLUMN IF EXISTS bet_order_no;
ALTER TABLE cloud_bet_records DROP COLUMN IF EXISTS third_party_bet_id;
ALTER TABLE cloud_bet_records DROP CONSTRAINT IF EXISTS chk_cloud_bet_records_status;
ALTER TABLE cloud_bet_records ADD CONSTRAINT chk_cloud_bet_records_status CHECK (status IN ('hit', 'miss'));
