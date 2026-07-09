-- +goose Up
-- +goose StatementBegin
INSERT INTO lottery_catalog (code, display_name, detail_alias, sort_order, on_sale) VALUES
    ('tencent_ffc', '腾讯分分彩', 'tencent-ffc', 10, true),
    ('tencent_10', '腾讯十分彩', 'tencent-10', 20, true),
    ('qiqu_tencent', '奇趣腾讯分分彩', 'qiqu-tencent', 30, true),
    ('us_ffc', '美国数据分分彩', 'us-ffc', 40, true),
    ('cq_ssc', '重庆时时彩', 'cq-ssc', 50, true),
    ('xj_ssc', '新疆时时彩', 'xj-ssc', 60, true),
    ('tj_ssc', '天津时时彩', 'tj-ssc', 70, true),
    ('fc_3d', '福彩3D', 'fc-3d', 80, true),
    ('pl3', '排列三', 'pl3', 90, true)
ON CONFLICT (code) DO NOTHING;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DELETE FROM lottery_catalog WHERE code IN (
    'tencent_ffc', 'tencent_10', 'qiqu_tencent', 'us_ffc',
    'cq_ssc', 'xj_ssc', 'tj_ssc', 'fc_3d', 'pl3'
);
-- +goose StatementEnd
