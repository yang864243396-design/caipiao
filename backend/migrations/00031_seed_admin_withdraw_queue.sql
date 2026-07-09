-- +goose Up
-- +goose StatementBegin
INSERT INTO withdraw_orders (order_no, member_id, amount, channel, status, created_at)
SELECT v.order_no, m.id, v.amount, v.channel, v.status, v.created_at
FROM members m
CROSS JOIN (VALUES
    ('WD00002', 1200.00, 'bank_card', 'pending_review', TIMESTAMPTZ '2026-05-20 08:30:00+00'),
    ('WD00003', 888.00, 'usdt_trc20', 'pending_review', TIMESTAMPTZ '2026-05-21 10:15:00+00'),
    ('WD00004', 2500.00, 'bank_card', 'pending_payout', TIMESTAMPTZ '2026-05-19 14:00:00+00'),
    ('WD00005', 650.00, 'alipay', 'pending_payout', TIMESTAMPTZ '2026-05-18 16:20:00+00')
) AS v(order_no, amount, channel, status, created_at)
WHERE m.member_no = 'M00001'
ON CONFLICT (order_no) DO NOTHING;

INSERT INTO withdraw_orders (order_no, member_id, amount, channel, status, created_at)
SELECT 'WD00006', id, 1500.00, 'bank_card', 'pending_review', TIMESTAMPTZ '2026-05-22 09:00:00+00'
FROM members WHERE member_no = 'M00002'
ON CONFLICT (order_no) DO NOTHING;

UPDATE member_wallets w
SET balance = GREATEST(w.balance, 2000.00),
    updated_at = now()
FROM members m
WHERE w.member_id = m.id AND m.member_no = 'M00002';

UPDATE member_wallets w
SET balance = balance - 5650.00,
    frozen_balance = frozen_balance + 5650.00,
    updated_at = now()
FROM members m
WHERE w.member_id = m.id AND m.member_no = 'M00001';

UPDATE member_wallets w
SET balance = balance - 1500.00,
    frozen_balance = frozen_balance + 1500.00,
    updated_at = now()
FROM members m
WHERE w.member_id = m.id AND m.member_no = 'M00002';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
UPDATE member_wallets w
SET balance = balance + 5650.00,
    frozen_balance = GREATEST(frozen_balance - 5650.00, 0),
    updated_at = now()
FROM members m
WHERE w.member_id = m.id AND m.member_no = 'M00001';

UPDATE member_wallets w
SET balance = balance + 1500.00,
    frozen_balance = GREATEST(frozen_balance - 1500.00, 0),
    updated_at = now()
FROM members m
WHERE w.member_id = m.id AND m.member_no = 'M00002';

DELETE FROM withdraw_orders WHERE order_no IN ('WD00002', 'WD00003', 'WD00004', 'WD00005', 'WD00006');
-- +goose StatementEnd
