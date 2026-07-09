-- +goose Up
-- +goose StatementBegin
INSERT INTO scheme_share_snapshots (
    id, scheme_name, lottery_code, lottery_label, play_method, fund_yuan, config, created_at, updated_at
) VALUES
(
    'SD10001', '雨卉组选', 'tencent_ffc', '奇趣腾讯分分彩', '后二组选复式', 39.20,
    '{"schemeName":"雨卉组选","lotteryCode":"tencent_ffc","runTypeId":"run_std","playTypeId":"play_group","subPlayId":"sub_back2_group"}'::jsonb,
    '2026-04-01T08:00:00Z', '2026-05-10T12:00:00Z'
),
(
    'SD10002', '千山独胆', 'us_data_ffc', '美国数据分分彩', '龙虎十个', 14.00,
    '{"schemeName":"千山独胆","lotteryCode":"us_data_ffc","runTypeId":"run_std","playTypeId":"play_dragon","subPlayId":"sub_ten"}'::jsonb,
    '2026-04-05T08:00:00Z', '2026-05-11T12:00:00Z'
),
(
    'SD10003', '福云', 'hn5_2', '老河内5分彩2', '定位胆万位', 9.80,
    '{"schemeName":"福云","lotteryCode":"hn5_2","runTypeId":"run_std","playTypeId":"play_pos","subPlayId":"sub_wan"}'::jsonb,
    '2026-04-08T08:00:00Z', '2026-05-12T12:00:00Z'
),
(
    'SD10004', '大乐', 'us_data_ffc', '美国数据分分彩', '定位胆十位', 49.00,
    '{"schemeName":"大乐","lotteryCode":"us_data_ffc","runTypeId":"run_std","playTypeId":"play_pos","subPlayId":"sub_shi"}'::jsonb,
    '2026-04-10T08:00:00Z', '2026-05-13T12:00:00Z'
),
(
    'SD10005', '幸运星2', 'tencent_ffc', '腾讯分分彩', '前三组选', 9.80,
    '{"schemeName":"幸运星2","lotteryCode":"tencent_ffc","runTypeId":"run_std","playTypeId":"play_group","subPlayId":"sub_front3"}'::jsonb,
    '2026-04-12T08:00:00Z', '2026-05-14T12:00:00Z'
),
(
    'SD10006', '幸运星1', 'tencent_ffc', '奇趣腾讯分分彩', '前三组选', 147.00,
    '{"schemeName":"幸运星1","lotteryCode":"tencent_ffc","runTypeId":"run_std","playTypeId":"play_group","subPlayId":"sub_front3"}'::jsonb,
    '2026-04-15T08:00:00Z', '2026-05-15T12:00:00Z'
),
(
    'SD10007', '独胆', 'tencent_ffc', '奇趣腾讯分分彩', '定位胆十个', 39.20,
    '{"schemeName":"独胆","lotteryCode":"tencent_ffc","runTypeId":"run_std","playTypeId":"play_pos","subPlayId":"sub_ten"}'::jsonb,
    '2026-04-18T08:00:00Z', '2026-05-16T12:00:00Z'
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DELETE FROM scheme_share_snapshots WHERE id LIKE 'SD100%';
-- +goose StatementEnd
