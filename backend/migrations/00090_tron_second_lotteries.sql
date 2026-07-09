-- +goose Up
-- 第三方 new_lott 增补：波场 3/6/15 秒彩（game_id 75/76/77），玩法复用 ssc_std

INSERT INTO lottery_catalog (
    code, display_name, category_code, play_template, ball_count,
    draw_interval, sort_order, on_sale, sale_status, outbound_lottery_code
) VALUES
    ('tron_ffc_3s',  '波场3秒彩',  'jisu', 'ssc_std', 5, '3s',  48, true, 'on_sale', '75'),
    ('tron_ffc_6s',  '波场6秒彩',  'jisu', 'ssc_std', 5, '6s',  49, true, 'on_sale', '76'),
    ('tron_ffc_15s', '波场15秒彩', 'jisu', 'ssc_std', 5, '15s', 50, true, 'on_sale', '77')
ON CONFLICT (code) DO UPDATE SET
    display_name = EXCLUDED.display_name,
    category_code = EXCLUDED.category_code,
    play_template = EXCLUDED.play_template,
    ball_count = EXCLUDED.ball_count,
    draw_interval = EXCLUDED.draw_interval,
    sort_order = EXCLUDED.sort_order,
    on_sale = EXCLUDED.on_sale,
    sale_status = EXCLUDED.sale_status,
    outbound_lottery_code = EXCLUDED.outbound_lottery_code,
    updated_at = now();

-- +goose Down
DELETE FROM lottery_catalog
WHERE code IN ('tron_ffc_3s', 'tron_ffc_6s', 'tron_ffc_15s');
