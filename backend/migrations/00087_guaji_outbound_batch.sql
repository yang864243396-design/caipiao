-- +goose Up
-- 第三方 outbound 批量配置：HTTP 下单 game_id（1–47）+ WS 开奖 guaji_ws_key + 定位胆 rule_id

ALTER TABLE lottery_catalog
    ADD COLUMN IF NOT EXISTS guaji_ws_key VARCHAR(64);

COMMENT ON COLUMN lottery_catalog.guaji_ws_key IS '第三方开奖 WS 彩种线键（lottery_logXXX / eth_lottery_log / tw_lottery_log）';
COMMENT ON COLUMN lottery_catalog.outbound_lottery_code IS '第三方 HTTP 下单 game_id（数字字符串，文档 §8 平台彩种序号）';

-- 平台彩种序号 → game_id（与 docs/seeds/generate_p0_seeds.py CATALOG 一致）
UPDATE lottery_catalog SET outbound_lottery_code = v.gid, guaji_ws_key = v.ws
FROM (VALUES
    ('tron_ffc_1m',  '1',  'lottery_log101'),
    ('tron_ffc_3m',  '2',  'lottery_log103'),
    ('tron_ffc_5m',  '3',  'lottery_log115'),
    ('hash_ffc_1m',  '4',  'lottery_log101'),
    ('hash_ffc_3m',  '5',  'lottery_log103'),
    ('hash_ffc_5m',  '6',  'lottery_log125'),
    ('eth_ffc_1m',   '7',  'eth_lottery_log'),
    ('eth_ffc_3m',   '8',  'eth_lottery_log'),
    ('eth_ffc_5m',   '9',  'eth_lottery_log'),
    ('bnb_ffc_1m',   '10', 'lottery_log101'),
    ('bnb_ffc_3m',   '11', 'lottery_log103'),
    ('bnb_ffc_5m',   '12', 'lottery_log125'),
    ('eth_ffc_new',  '13', 'eth_lottery_log'),
    ('tron_jisu',    '14', 'lottery_log033'),
    ('hash_jisu',    '15', 'lottery_log033'),
    ('eth_jisu',     '16', 'eth_lottery_log'),
    ('tron_lhc_1m',  '17', 'lottery_log101'),
    ('tron_lhc_3m',  '18', 'lottery_log103'),
    ('tron_lhc_5m',  '19', 'lottery_log115'),
    ('tron_lhc',     '20', 'lottery_log125'),
    ('tron_syxw',    '21', 'lottery_log033'),
    ('tron_syxw_3m', '22', 'lottery_log103'),
    ('tron_syxw_5m', '23', 'lottery_log115'),
    ('eth_syxw',     '24', 'eth_lottery_log'),
    ('eth_syxw_3m',  '25', 'eth_lottery_log'),
    ('eth_syxw_5m',  '26', 'eth_lottery_log'),
    ('bnb_syxw',     '27', 'lottery_log033'),
    ('bnb_syxw_3m',  '28', 'lottery_log103'),
    ('bnb_syxw_5m',  '29', 'lottery_log125'),
    ('eth_pk10_jisu','30', 'eth_lottery_log'),
    ('eth_pk10_5m',  '31', 'eth_lottery_log'),
    ('bnb_pk10_jisu','32', 'lottery_log033'),
    ('bnb_pk10_5m',  '33', 'lottery_log125'),
    ('tron_pk10_jisu','34','lottery_log033'),
    ('taiwan_pk10',  '35', 'tw_lottery_log'),
    ('eth_k3',       '36', 'eth_lottery_log'),
    ('eth_k3_3m',    '37', 'eth_lottery_log'),
    ('eth_k3_5m',    '38', 'eth_lottery_log'),
    ('tron_k3_jisu', '39', 'lottery_log033'),
    ('tron_k3_1m',   '40', 'lottery_log101'),
    ('tron_k3_3m',   '41', 'lottery_log103'),
    ('tron_k3_5m',   '42', 'lottery_log115'),
    ('bnb_k3_1m',    '43', 'lottery_log101'),
    ('bnb_k3_3m',    '44', 'lottery_log103'),
    ('bnb_k3_5m',    '45', 'lottery_log125'),
    ('taiwan_ssc_5m','46', 'tw_lottery_log'),
    ('taiwan_pc28',  '47', 'tw_lottery_log')
) AS v(code, gid, ws)
WHERE lottery_catalog.code = v.code;

-- ssc 定位胆 → 第三方 rule_id（文档 §11 示例 rule_id=13 为万位；其余位按序 14–17）
UPDATE sub_plays SET outbound_play_code = v.rule
FROM (VALUES
    ('ssc_std', 'dingwei', 'dingwei_wan',  '13'),
    ('ssc_std', 'dingwei', 'dingwei_qian', '14'),
    ('ssc_std', 'dingwei', 'dingwei_bai',  '15'),
    ('ssc_std', 'dingwei', 'dingwei_shi',  '16'),
    ('ssc_std', 'dingwei', 'dingwei_ge',   '17')
) AS v(tpl, tid, sid, rule)
WHERE sub_plays.template_code = v.tpl
  AND sub_plays.type_id = v.tid
  AND sub_plays.sub_id = v.sid;

-- +goose Down
UPDATE sub_plays SET outbound_play_code = template_code || ':' || type_id || ':' || sub_id
WHERE template_code = 'ssc_std' AND type_id = 'dingwei';

UPDATE lottery_catalog SET outbound_lottery_code = code, guaji_ws_key = NULL;

ALTER TABLE lottery_catalog DROP COLUMN IF EXISTS guaji_ws_key;
