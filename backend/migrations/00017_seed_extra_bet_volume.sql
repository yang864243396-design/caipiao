-- +goose Up
-- +goose StatementBegin
INSERT INTO bet_orders (
    order_no, member_id, lottery_code, lottery_name, lottery_category,
    issue_no, amount, pnl, status, placed_at, settled_at
)
SELECT
    'T20260517008888', m.id, 'tencent_ffc', '时时彩 A', 'ssc', '20260517088',
    730.00, -730.00, 'lose',
    TIMESTAMPTZ '2026-05-17 08:00:00+00', TIMESTAMPTZ '2026-05-17 08:05:00+00'
FROM members m WHERE m.member_no = 'M00001'
ON CONFLICT (order_no) DO NOTHING;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DELETE FROM bet_orders WHERE order_no = 'T20260517008888';
-- +goose StatementEnd
