-- +goose Up
-- 哈希 1/3/5 分彩 ↔ 波场 1/3/5 分彩：第三方 game_id 与 WS 开奖线键对调（正式/测试环境均适用）。
WITH pairs AS (
  SELECT unnest(ARRAY['hash_ffc_1m', 'hash_ffc_3m', 'hash_ffc_5m']) AS hash_code,
         unnest(ARRAY['tron_ffc_1m', 'tron_ffc_3m', 'tron_ffc_5m']) AS tron_code
),
snap AS (
  SELECT h.code AS hash_code,
         t.code AS tron_code,
         h.outbound_lottery_code AS hash_gid,
         t.outbound_lottery_code AS tron_gid,
         h.guaji_ws_key AS hash_ws,
         t.guaji_ws_key AS tron_ws
  FROM pairs p
  JOIN lottery_catalog h ON h.code = p.hash_code
  JOIN lottery_catalog t ON t.code = p.tron_code
)
UPDATE lottery_catalog lc
SET outbound_lottery_code = CASE
      WHEN lc.code = s.hash_code THEN s.tron_gid
      WHEN lc.code = s.tron_code THEN s.hash_gid
    END,
    guaji_ws_key = CASE
      WHEN lc.code = s.hash_code THEN s.tron_ws
      WHEN lc.code = s.tron_code THEN s.hash_ws
    END,
    updated_at = now()
FROM snap s
WHERE lc.code IN (s.hash_code, s.tron_code);

-- +goose Down
-- 再次对调即可回退
WITH pairs AS (
  SELECT unnest(ARRAY['hash_ffc_1m', 'hash_ffc_3m', 'hash_ffc_5m']) AS hash_code,
         unnest(ARRAY['tron_ffc_1m', 'tron_ffc_3m', 'tron_ffc_5m']) AS tron_code
),
snap AS (
  SELECT h.code AS hash_code,
         t.code AS tron_code,
         h.outbound_lottery_code AS hash_gid,
         t.outbound_lottery_code AS tron_gid,
         h.guaji_ws_key AS hash_ws,
         t.guaji_ws_key AS tron_ws
  FROM pairs p
  JOIN lottery_catalog h ON h.code = p.hash_code
  JOIN lottery_catalog t ON t.code = p.tron_code
)
UPDATE lottery_catalog lc
SET outbound_lottery_code = CASE
      WHEN lc.code = s.hash_code THEN s.tron_gid
      WHEN lc.code = s.tron_code THEN s.hash_gid
    END,
    guaji_ws_key = CASE
      WHEN lc.code = s.hash_code THEN s.tron_ws
      WHEN lc.code = s.tron_code THEN s.hash_ws
    END,
    updated_at = now()
FROM snap s
WHERE lc.code IN (s.hash_code, s.tron_code);
