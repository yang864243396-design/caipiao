-- +goose Up
-- 00115 对换 outbound 后，哈希/波场 1/3/5 分彩的 WS 开奖线与历史 REST 路径也需对调。
-- 实测：tron_ffc_1m（game_id=19）期号在 lottery_logs；hash_ffc_1m（game_id=25）在 lottery_log103s。
UPDATE lottery_catalog SET guaji_ws_key = 'lottery_logs', updated_at = now()
WHERE code IN ('tron_ffc_1m', 'tron_ffc_3m', 'tron_ffc_5m');

UPDATE lottery_catalog SET guaji_ws_key = 'lottery_log103s', updated_at = now()
WHERE code = 'hash_ffc_1m';

UPDATE lottery_catalog SET guaji_ws_key = 'lottery_log303s', updated_at = now()
WHERE code = 'hash_ffc_3m';

UPDATE lottery_catalog SET guaji_ws_key = 'lottery_log503s', updated_at = now()
WHERE code = 'hash_ffc_5m';

-- +goose Down
UPDATE lottery_catalog SET guaji_ws_key = 'lottery_log101', updated_at = now()
WHERE code IN ('tron_ffc_1m', 'hash_ffc_1m', 'tron_ffc_3m', 'hash_ffc_3m', 'tron_ffc_5m', 'hash_ffc_5m');
