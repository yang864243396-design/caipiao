-- +goose Up
-- 极速时时彩（fast_ssc_std）第三方未配置「任选四直选复式」(rule_id=141)。
--
-- 2026-07-28 真实下单：
--   tron_ffc_3s (game=73) / tron_ffc_6s (game=74) rule=141 → 40000「该玩法对应的游戏没有配置」
--   tron_ffc_1m (game=19, ssc_std) rule=141 → 接单成功
--
-- 同模板下其余任选玩法（任二/任三、任四组选24/12/6）均可下单，仅此一条缺失。
-- 关闭后前端不再展示、方案不可选，避免正式盘静默拒单。
UPDATE sub_plays
SET enabled = false,
    updated_at = now()
WHERE template_code = 'fast_ssc_std'
  AND type_id = 'g011'
  AND sub_id = '141'
  AND enabled = true;

-- +goose Down
UPDATE sub_plays
SET enabled = true,
    updated_at = now()
WHERE template_code = 'fast_ssc_std'
  AND type_id = 'g011'
  AND sub_id = '141'
  AND enabled = false;
