-- +goose Up
-- P0/P1：00115 swap 后 guaji_ws_key 与 v6hs1 WS 广播实测对齐（2026-07-09 audit-ws-keys）。
-- REST 路径带 trailing s（如 lottery_log103s），WS lottery_v2_broadcast 内键无 s（lottery_log103）。
-- 波场/哈希 1/3/5 分分彩（1014*/3014*/5014* 与 lottery_logs 等）仍待第三方确认，不在此迁移。

-- P0：哈希1分（REST lottery_log103s ↔ WS lottery_log103）
UPDATE lottery_catalog SET guaji_ws_key = 'lottery_log103', updated_at = now()
WHERE code = 'hash_ffc_1m';

-- P0：波场极速（REST lottery_log05s ↔ WS lottery_log05）
UPDATE lottery_catalog SET guaji_ws_key = 'lottery_log05', updated_at = now()
WHERE code = 'tron_jisu';

-- P0：币安1分系（REST bsc_lottery_logs ↔ WS bsc_lottery_log01）
UPDATE lottery_catalog SET guaji_ws_key = 'bsc_lottery_log01', updated_at = now()
WHERE code IN ('bnb_ffc_1m', 'bnb_k3_1m', 'bnb_syxw');

-- P1：波场1分衍生（REST lottery_log103s ↔ WS lottery_log103）
UPDATE lottery_catalog SET guaji_ws_key = 'lottery_log103', updated_at = now()
WHERE code IN ('tron_k3_1m', 'tron_lhc_1m', 'tron_syxw');

-- P1：波场3分衍生（REST lottery_log303s ↔ WS lottery_log303，文档 §7.2）
UPDATE lottery_catalog SET guaji_ws_key = 'lottery_log303', updated_at = now()
WHERE code IN ('tron_k3_3m', 'tron_lhc_3m', 'tron_syxw_3m');

-- P1：波场5分衍生 + 波场六合（REST lottery_log503s ↔ WS lottery_log503）
UPDATE lottery_catalog SET guaji_ws_key = 'lottery_log503', updated_at = now()
WHERE code IN ('tron_k3_5m', 'tron_lhc_5m', 'tron_syxw_5m', 'tron_lhc');

-- P1：币安3分系（REST bsc_lottery_log3s ↔ WS bsc_lottery_log03）
UPDATE lottery_catalog SET guaji_ws_key = 'bsc_lottery_log03', updated_at = now()
WHERE code IN ('bnb_ffc_3m', 'bnb_k3_3m', 'bnb_syxw_3m');

-- P1：币安5分系（REST bsc_lottery_log5s ↔ WS bsc_lottery_log05）
UPDATE lottery_catalog SET guaji_ws_key = 'bsc_lottery_log05', updated_at = now()
WHERE code IN ('bnb_ffc_5m', 'bnb_k3_5m', 'bnb_syxw_5m', 'bnb_pk10_5m');

-- +goose Down
UPDATE lottery_catalog SET guaji_ws_key = 'lottery_log103s', updated_at = now()
WHERE code = 'hash_ffc_1m';

UPDATE lottery_catalog SET guaji_ws_key = 'lottery_log033', updated_at = now()
WHERE code = 'tron_jisu';

UPDATE lottery_catalog SET guaji_ws_key = 'lottery_log101', updated_at = now()
WHERE code IN ('bnb_ffc_1m', 'bnb_k3_1m', 'bnb_syxw');

UPDATE lottery_catalog SET guaji_ws_key = 'lottery_log101', updated_at = now()
WHERE code IN ('tron_k3_1m', 'tron_lhc_1m');

UPDATE lottery_catalog SET guaji_ws_key = 'lottery_log033', updated_at = now()
WHERE code = 'tron_syxw';

UPDATE lottery_catalog SET guaji_ws_key = 'lottery_log103', updated_at = now()
WHERE code IN ('tron_k3_3m', 'tron_lhc_3m', 'tron_syxw_3m');

UPDATE lottery_catalog SET guaji_ws_key = 'lottery_log115', updated_at = now()
WHERE code IN ('tron_k3_5m', 'tron_lhc_5m', 'tron_syxw_5m', 'tron_lhc');

UPDATE lottery_catalog SET guaji_ws_key = 'lottery_log103', updated_at = now()
WHERE code IN ('bnb_ffc_3m', 'bnb_k3_3m', 'bnb_syxw_3m');

UPDATE lottery_catalog SET guaji_ws_key = 'lottery_log125', updated_at = now()
WHERE code IN ('bnb_ffc_5m', 'bnb_k3_5m', 'bnb_syxw_5m', 'bnb_pk10_5m');
