-- +goose Up
-- 测试环境实测修正：文档序号 game_id=1 在 hash.iyes.dev 报「玩法对应游戏没有上线」；
-- 波场1分彩 tron_ffc_1m 实测可用 game_id=29（2026-06-16 E2E 验证）。
-- 正式环境须按第三方 §8 对照表逐条核对后覆盖。

UPDATE lottery_catalog
SET outbound_lottery_code = '29'
WHERE code = 'tron_ffc_1m';

-- +goose Down
UPDATE lottery_catalog
SET outbound_lottery_code = '1'
WHERE code = 'tron_ffc_1m';
