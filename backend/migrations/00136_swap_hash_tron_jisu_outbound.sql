-- +goose Up
-- 哈希极速彩 ↔ 波场极速彩：第三方 game_id 与 WS 开奖线键对调，同 00115 对分分彩的处理。
--
-- 00115 只对调了 hash/tron 1/3/5 分彩，极速彩漏做；00123 又只把 tron_jisu 的 WS 键
-- 改成 lottery_log05，导致下注链路（outbound_lottery_code）与开奖链路（guaji_ws_key）
-- 指向两个不同的期号族，投注详情永远查不到开奖号。
--
-- 2026-07-28 实测 /api/web_bets/lott/periods：
--   game_id=23 → 10514138800312（区块高度族，= 本地 tron_jisu 开奖期号）
--   game_id=24 → 105202607280066（日期族，  = 本地 hash_jisu 开奖期号）
-- 与第三方 new_lott 名称（23=哈希极速彩、24=波场极速彩）相反，取实测期号族为准。
--
-- 只改 outbound_lottery_code；guaji_ws_key 与 historysync/paths.go 已正确，不动。
UPDATE lottery_catalog SET outbound_lottery_code = v.gid, updated_at = now()
FROM (VALUES
    ('hash_jisu', '24'),
    ('tron_jisu', '23')
) AS v(code, gid)
WHERE lottery_catalog.code = v.code;

-- +goose Down
UPDATE lottery_catalog SET outbound_lottery_code = v.gid, updated_at = now()
FROM (VALUES
    ('hash_jisu', '23'),
    ('tron_jisu', '24')
) AS v(code, gid)
WHERE lottery_catalog.code = v.code;
