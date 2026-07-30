-- +goose Up
-- 币安极速飞艇（bnb_pk10_jisu）的开奖线键抄成了波场线，注单永远查不到开奖。
--
-- 2026-07-28 实测 /api/web_bets/lott/periods：
--   game_id=53（bnb_pk10_jisu）→ 10111202607281098，窗口 18:18:00–18:19:00
--   game_id=39（bnb_ffc_1m）  → 10111202607281098，窗口 18:18:00–18:19:00（完全同族）
--   game_id=55（tron_pk10_jisu）→ 105202607282195，窗口 18:17:30–18:18:30
-- 即 53 跑在币安 1 分线（WS 键 bsc_lottery_log01），不是波场极速线 lottery_log033。
--
-- 原配置 guaji_ws_key='lottery_log033' 与 tron_pk10_jisu 相同，导致两个彩种的
-- lottery_draws 逐字节相同（99186 条期号+号码完全一致），而下注链路取的是
-- 10111* 期号族，永远匹配不到 105* 的开奖行：
-- 该彩种 102 笔会员注单中 101 笔最终 cancel、0 笔查到开奖。
-- （正式盘按第三方回传结算，故未算错钱；错的是开奖展示与结算依据。）
--
-- 只改 guaji_ws_key。draw_interval='jisu' 已映射 60 秒（ParseDrawIntervalSec），与实测一致，不动。
UPDATE lottery_catalog SET guaji_ws_key = 'bsc_lottery_log01', updated_at = now()
WHERE code = 'bnb_pk10_jisu' AND guaji_ws_key = 'lottery_log033';

-- +goose Down
UPDATE lottery_catalog SET guaji_ws_key = 'lottery_log033', updated_at = now()
WHERE code = 'bnb_pk10_jisu';
