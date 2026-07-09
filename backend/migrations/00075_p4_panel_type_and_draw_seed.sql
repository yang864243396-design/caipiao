-- +goose Up
-- P4：syxw/pk10/k3/pc28 panel_type + 代表性彩种开奖种子

UPDATE play_types SET panel_type = 'dingwei' WHERE template_code = 'syxw_std' AND type_id = 'dingwei';
UPDATE play_types SET panel_type = 'segment' WHERE template_code = 'syxw_std' AND type_id IN ('qian3', 'qian2');
UPDATE play_types SET panel_type = 'renxuan' WHERE template_code = 'syxw_std' AND type_id IN ('renxuan_fs', 'renxuan_ds');

UPDATE play_types SET panel_type = 'dingwei' WHERE template_code = 'pk10_std' AND type_id = 'dingwei';
UPDATE play_types SET panel_type = 'longhu' WHERE template_code = 'pk10_std' AND type_id = 'longhu';
UPDATE play_types SET panel_type = 'segment' WHERE template_code = 'pk10_std' AND type_id IN ('qian1', 'qian2', 'qian3', 'qian4', 'qian5');
UPDATE play_types SET panel_type = 'textarea' WHERE template_code = 'pk10_std' AND type_id IN ('daxiao', 'danshuang', 'hezhi', 'dxds_combo');

UPDATE play_types SET panel_type = 'textarea' WHERE template_code = 'k3_std' AND type_id = 'hezhi';
UPDATE play_types SET panel_type = 'k3_pool' WHERE template_code = 'k3_std' AND type_id IN ('tonghao', 'butonghao', 'lianhao_qita');

UPDATE play_types SET panel_type = 'textarea' WHERE template_code = 'pc28_std';

INSERT INTO lottery_draws (lottery_code, issue_no, period_short, balls, sum_value, drawn_at) VALUES
    ('tron_syxw', '20231103027', '027', '["01","04","06","08","11"]'::jsonb, 30, '2026-06-08 10:10:00+00'),
    ('tron_syxw', '20231103028', '028', '["02","05","07","09","10"]'::jsonb, 33, '2026-06-08 10:11:00+00'),
    ('tron_syxw', '20231103029', '029', '["01","03","06","08","11"]'::jsonb, 29, '2026-06-08 10:12:00+00'),
    ('tron_syxw', '20231103030', '030', '["02","04","07","09","10"]'::jsonb, 32, '2026-06-08 10:13:00+00'),
    ('tron_syxw', '20231103031', '031', '["01","05","06","08","11"]'::jsonb, 31, '2026-06-08 10:14:00+00'),
    ('eth_pk10_jisu', '20231103027', '027', '["3","7","1","9","5","2","8","4","6","10"]'::jsonb, 55, '2026-06-08 10:10:00+00'),
    ('eth_pk10_jisu', '20231103028', '028', '["8","2","5","1","10","4","7","3","6","9"]'::jsonb, 55, '2026-06-08 10:11:00+00'),
    ('eth_pk10_jisu', '20231103029', '029', '["6","4","9","2","7","1","10","3","5","8"]'::jsonb, 55, '2026-06-08 10:12:00+00'),
    ('eth_pk10_jisu', '20231103030', '030', '["10","1","4","8","3","6","9","2","7","5"]'::jsonb, 55, '2026-06-08 10:13:00+00'),
    ('eth_pk10_jisu', '20231103031', '031', '["5","9","2","7","1","10","4","8","3","6"]'::jsonb, 55, '2026-06-08 10:14:00+00'),
    ('eth_k3', '20231103027', '027', '["2","4","6"]'::jsonb, 12, '2026-06-08 10:10:00+00'),
    ('eth_k3', '20231103028', '028', '["1","3","5"]'::jsonb, 9, '2026-06-08 10:11:00+00'),
    ('eth_k3', '20231103029', '029', '["2","2","5"]'::jsonb, 9, '2026-06-08 10:12:00+00'),
    ('eth_k3', '20231103030', '030', '["3","4","6"]'::jsonb, 13, '2026-06-08 10:13:00+00'),
    ('eth_k3', '20231103031', '031', '["1","1","6"]'::jsonb, 8, '2026-06-08 10:14:00+00'),
    ('taiwan_pc28', '20231103027', '027', '["3","5","7"]'::jsonb, 15, '2026-06-08 10:10:00+00'),
    ('taiwan_pc28', '20231103028', '028', '["1","8","4"]'::jsonb, 13, '2026-06-08 10:11:00+00'),
    ('taiwan_pc28', '20231103029', '029', '["9","2","6"]'::jsonb, 17, '2026-06-08 10:12:00+00'),
    ('taiwan_pc28', '20231103030', '030', '["0","7","3"]'::jsonb, 10, '2026-06-08 10:13:00+00'),
    ('taiwan_pc28', '20231103031', '031', '["4","5","6"]'::jsonb, 15, '2026-06-08 10:14:00+00')
ON CONFLICT (lottery_code, issue_no) DO NOTHING;

-- +goose Down
DELETE FROM lottery_draws
WHERE (lottery_code, issue_no) IN (
    ('tron_syxw', '20231103027'),
    ('tron_syxw', '20231103028'),
    ('tron_syxw', '20231103029'),
    ('tron_syxw', '20231103030'),
    ('tron_syxw', '20231103031'),
    ('eth_pk10_jisu', '20231103027'),
    ('eth_pk10_jisu', '20231103028'),
    ('eth_pk10_jisu', '20231103029'),
    ('eth_pk10_jisu', '20231103030'),
    ('eth_pk10_jisu', '20231103031'),
    ('eth_k3', '20231103027'),
    ('eth_k3', '20231103028'),
    ('eth_k3', '20231103029'),
    ('eth_k3', '20231103030'),
    ('eth_k3', '20231103031'),
    ('taiwan_pc28', '20231103027'),
    ('taiwan_pc28', '20231103028'),
    ('taiwan_pc28', '20231103029'),
    ('taiwan_pc28', '20231103030'),
    ('taiwan_pc28', '20231103031')
);
UPDATE play_types SET panel_type = NULL WHERE template_code IN ('syxw_std', 'pk10_std', 'k3_std', 'pc28_std');
