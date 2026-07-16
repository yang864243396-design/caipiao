-- +goose Up
-- 波场 1/3/5 分彩（00 区块）+ 波场 3/6/15 秒彩：补齐 guaji_ws_key。
-- 依据第三方文档 §7.3、前端 blockField 与运营对照：
--   3秒  lottery_v2_broadcast（每区块一条）
--   6秒  lottery_log101
--   15秒 lottery_log125
--   1分  lottery1_wsds（00）；03 线 lottery_log103 已用于 hash_ffc_1m / 波场衍生
--   3分  lottery3_wsds（00）；03 线 lottery_log303
--   5分  lottery5_wsds（00）；03 线 lottery_log503

UPDATE lottery_catalog SET guaji_ws_key = 'lottery_v2_broadcast', updated_at = now()
WHERE code = 'tron_ffc_3s';

UPDATE lottery_catalog SET guaji_ws_key = 'lottery_log101', updated_at = now()
WHERE code = 'tron_ffc_6s';

UPDATE lottery_catalog SET guaji_ws_key = 'lottery_log125', updated_at = now()
WHERE code = 'tron_ffc_15s';

UPDATE lottery_catalog SET guaji_ws_key = 'lottery1_wsds', updated_at = now()
WHERE code = 'tron_ffc_1m';

UPDATE lottery_catalog SET guaji_ws_key = 'lottery3_wsds', updated_at = now()
WHERE code = 'tron_ffc_3m';

UPDATE lottery_catalog SET guaji_ws_key = 'lottery5_wsds', updated_at = now()
WHERE code = 'tron_ffc_5m';

-- +goose Down
UPDATE lottery_catalog SET guaji_ws_key = 'lottery_logs', updated_at = now()
WHERE code IN ('tron_ffc_1m', 'tron_ffc_3m', 'tron_ffc_5m');

UPDATE lottery_catalog SET guaji_ws_key = NULL, updated_at = now()
WHERE code IN ('tron_ffc_3s', 'tron_ffc_6s', 'tron_ffc_15s');
