-- +goose Up
-- 哈希3/5分：REST lottery_log303s / lottery_log503s ↔ WS lottery_log303 / lottery_log503（去 trailing s，同 00123 hash_ffc_1m）。
UPDATE lottery_catalog SET guaji_ws_key = 'lottery_log303', updated_at = now()
WHERE code = 'hash_ffc_3m';

UPDATE lottery_catalog SET guaji_ws_key = 'lottery_log503', updated_at = now()
WHERE code = 'hash_ffc_5m';

-- +goose Down
UPDATE lottery_catalog SET guaji_ws_key = 'lottery_log303s', updated_at = now()
WHERE code = 'hash_ffc_3m';

UPDATE lottery_catalog SET guaji_ws_key = 'lottery_log503s', updated_at = now()
WHERE code = 'hash_ffc_5m';
