-- +goose Up
-- 修复 lhc_std g003/299「三全中复式」玩法名中的 U+FFFD 替换字符。
--
-- 入库时「式」被写成两个 U+FFFD（同步侧 UTF-8 解码失败），label 实际为
-- "三全中复\uFFFD\uFFFD"。这条名称是玩法模式的判定依据：
-- inferLHCBetMode 匹配不到「复式」→ 返回空 → 取样落到默认的单个号码，
-- 而第三方要求三全中复式「只能投注3~10个数字」，该玩法在正式盘一直无法下单
-- （2026-07-28 真实下单矩阵 tron_lhc_1m index 27 稳定拒单）。
--
-- 全库扫描 sub_plays.label / play_types.label / lottery_catalog.display_name /
-- sub_plays.segment_rule 仅此一条含替换字符，属孤立损坏，不做批量替换。
-- 用 position() 定位替换字符，避免依赖损坏字节的字面写法。
UPDATE sub_plays
SET label = '三全中复式',
    segment_rule = jsonb_set(segment_rule, '{guajiFullName}', '"三全中复式"'::jsonb),
    updated_at = now()
WHERE template_code = 'lhc_std'
  AND type_id = 'g003'
  AND sub_id = '299'
  AND position(U&'\FFFD' IN label) > 0;

-- +goose Down
-- 不还原损坏数据：Down 仅在名称已被修复时保留现状。
SELECT 1;
