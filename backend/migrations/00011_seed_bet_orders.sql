-- +goose Up
-- +goose StatementBegin
INSERT INTO bet_orders (
    order_no, member_id, lottery_code, lottery_name, lottery_category,
    issue_no, amount, pnl, status, placed_at, settled_at
)
SELECT
    v.order_no,
    m.id,
    v.lottery_code,
    v.lottery_name,
    v.lottery_category,
    v.issue_no,
    v.amount,
    v.pnl,
    v.status,
    v.placed_at,
    v.settled_at
FROM members m
CROSS JOIN (
    VALUES
        (
            'T20260519005678', 'tencent_ffc', '时时彩 A', 'ssc', '20260519102',
            200.00, 0.00, 'pending',
            TIMESTAMPTZ '2026-05-19 06:31:22+00', NULL
        ),
        (
            'T20260519005102', 'pk10_fast', 'PK10 快开', 'pk10', '20260519288',
            50.00, 95.00, 'win',
            TIMESTAMPTZ '2026-05-19 06:05:09+00', TIMESTAMPTZ '2026-05-19 06:10:00+00'
        ),
        (
            'T20260518009231', 'k3_b', '快三 B', 'k3', '20260518412',
            120.00, -120.00, 'lose',
            TIMESTAMPTZ '2026-05-18 12:18:44+00', TIMESTAMPTZ '2026-05-18 12:20:00+00'
        ),
        (
            'B20260519001234', 'tencent_ffc', '时时彩 A', 'ssc', '20260519101',
            200.00, -200.00, 'lose',
            TIMESTAMPTZ '2026-05-19 03:00:44+00', TIMESTAMPTZ '2026-05-19 03:05:00+00'
        ),
        (
            'P20260519000990', 'tencent_ffc', '时时彩 A', 'ssc', '20260519100',
            200.00, 200.00, 'win',
            TIMESTAMPTZ '2026-05-19 06:32:01+00', TIMESTAMPTZ '2026-05-19 06:35:00+00'
        )
) AS v(
    order_no, lottery_code, lottery_name, lottery_category, issue_no,
    amount, pnl, status, placed_at, settled_at
)
WHERE m.member_no = 'M00001'
ON CONFLICT (order_no) DO NOTHING;
-- +goose StatementEnd

-- +goose Down
DELETE FROM bet_orders
WHERE order_no IN (
    'T20260519005678', 'T20260519005102', 'T20260518009231',
    'B20260519001234', 'P20260519000990'
);
