-- +goose Up
-- +goose StatementBegin
INSERT INTO cloud_bet_records (
    record_no, member_id, run_mode, scheme_id, scheme_name, period_no,
    play_type, multiplier, round_label, amount, pnl, status, placed_at
)
SELECT
    v.record_no, m.id, 'real', v.scheme_id, v.scheme_name, v.period_no,
    v.play_type, v.multiplier, v.round_label, v.amount, v.pnl, v.status, v.placed_at
FROM members m
CROSS JOIN (
    VALUES
        ('br-001', 'sch-wan', '禄螭万位计划', '20240310031', '万位定位', '1.5', '1/3', 10.00, 5.00, 'hit', TIMESTAMPTZ '2026-05-25 10:00:00+00'),
        ('br-002', 'sch-qian', '千位稳赢方案', '20240310030', '千位定位', '3.0', '2/3', 20.00, -20.00, 'miss', TIMESTAMPTZ '2026-05-25 09:00:00+00'),
        ('br-003', 'sch-ge', '个位轻量追号', '20220523029', '个位定位', '1.5', '1/5', 5.00, 2.50, 'hit', TIMESTAMPTZ '2026-05-24 15:00:00+00'),
        ('br-004', 'sch-bai', '百位进阶倍投', '20240310028', '百位定位', '2.2', '3/3', 50.00, 60.00, 'hit', TIMESTAMPTZ '2026-05-24 12:00:00+00'),
        ('br-005', 'sch-wan', '禄螭万位计划', '20240310027', '万位定位', '1.0', '1/3', 100.00, -100.00, 'miss', TIMESTAMPTZ '2026-05-23 11:00:00+00'),
        ('br-006', 'sch-shi', '十位云策略', '20240310026', '十位定位', '1.5', '1/3', 10.00, 5.00, 'hit', TIMESTAMPTZ '2026-05-23 08:00:00+00'),
        ('br-007', 'sch-wan', '禄螭万位计划', '20240310025', '万位定位', '1.5', '2/3', 30.00, 15.00, 'hit', TIMESTAMPTZ '2026-05-22 14:00:00+00'),
        ('br-008', 'sch-ge', '个位轻量追号', '20240310024', '个位定位', '1.5', '1/3', 10.00, 5.00, 'hit', TIMESTAMPTZ '2026-05-22 10:00:00+00')
) AS v(
    record_no, scheme_id, scheme_name, period_no, play_type, multiplier,
    round_label, amount, pnl, status, placed_at
)
WHERE m.member_no = 'M00001'
ON CONFLICT (record_no) DO NOTHING;
-- +goose StatementEnd

-- +goose Down
DELETE FROM cloud_bet_records
WHERE record_no IN ('br-001','br-002','br-003','br-004','br-005','br-006','br-007','br-008');
