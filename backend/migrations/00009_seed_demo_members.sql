-- +goose Up
-- +goose StatementBegin
INSERT INTO members (
    member_no, account, password_hash, display_name, status,
    l1_agent_code, l2_agent_code, registered_at, last_login_at
) VALUES
    (
        'M00001', 'vs8888',
        '$2a$10$DjNjiHAo37dgStLOraJTheJTbPL64mz2UNMfUG0Fsuko4qgL82eSm',
        '会员甲', 'active', 'AGT-L1-JIA', 'AGT-L2-YI',
        TIMESTAMPTZ '2026-01-15 08:00:00+00', TIMESTAMPTZ '2026-05-26 09:00:00+00'
    ),
    (
        'M00002', 'demo0002',
        '$2a$10$DjNjiHAo37dgStLOraJTheJTbPL64mz2UNMfUG0Fsuko4qgL82eSm',
        '会员乙', 'active', 'AGT-L1-JIA', NULL,
        TIMESTAMPTZ '2026-02-01 08:00:00+00', TIMESTAMPTZ '2026-05-25 10:00:00+00'
    ),
    (
        'M00003', 'demo0003',
        '$2a$10$DjNjiHAo37dgStLOraJTheJTbPL64mz2UNMfUG0Fsuko4qgL82eSm',
        '会员丙', 'active', 'AGT-L1-JIA', 'AGT-L2-YI',
        TIMESTAMPTZ '2026-02-10 08:00:00+00', TIMESTAMPTZ '2026-05-24 12:00:00+00'
    )
ON CONFLICT (member_no) DO NOTHING;

INSERT INTO member_wallets (member_id, balance, frozen_balance, currency)
SELECT id, 12888.66, 0, 'CNY' FROM members WHERE member_no = 'M00001'
ON CONFLICT (member_id) DO NOTHING;

INSERT INTO member_wallets (member_id, balance, frozen_balance, currency)
SELECT id, 368.20, 0, 'CNY' FROM members WHERE member_no = 'M00002'
ON CONFLICT (member_id) DO NOTHING;

INSERT INTO member_wallets (member_id, balance, frozen_balance, currency)
SELECT id, 906.55, 0, 'CNY' FROM members WHERE member_no = 'M00003'
ON CONFLICT (member_id) DO NOTHING;

INSERT INTO wallet_ledger (ledger_no, member_id, txn_type, delta_amount, balance_after, order_ref, created_at)
SELECT 'ML000010001', id, 'payout', 200.00, 12888.66, 'P20260519000990', TIMESTAMPTZ '2026-05-19 06:32:01+00'
FROM members WHERE member_no = 'M00001'
ON CONFLICT (ledger_no) DO NOTHING;

INSERT INTO wallet_ledger (ledger_no, member_id, txn_type, delta_amount, balance_after, order_ref, created_at)
SELECT 'ML000010002', id, 'bet_debit', -200.00, 12688.66, 'B20260519001234', TIMESTAMPTZ '2026-05-19 03:00:44+00'
FROM members WHERE member_no = 'M00001'
ON CONFLICT (ledger_no) DO NOTHING;

INSERT INTO wallet_ledger (ledger_no, member_id, txn_type, delta_amount, balance_after, order_ref, created_at)
SELECT 'ML000010003', id, 'deposit', 1000.00, 11888.66, 'RC-MOCK-JIA-20260519', TIMESTAMPTZ '2026-05-18 01:10:22+00'
FROM members WHERE member_no = 'M00001'
ON CONFLICT (ledger_no) DO NOTHING;

INSERT INTO wallet_ledger (ledger_no, member_id, txn_type, delta_amount, balance_after, order_ref, created_at)
SELECT 'ML000010004', id, 'withdraw', -800.00, 10888.66, 'WD00001', TIMESTAMPTZ '2026-05-17 13:05:00+00'
FROM members WHERE member_no = 'M00001'
ON CONFLICT (ledger_no) DO NOTHING;

INSERT INTO recharge_orders (order_no, member_id, amount, channel, status, paid_at, created_at)
SELECT 'RC-MOCK-JIA-20260519', id, 1000.00, 'mock_channel', 'paid', TIMESTAMPTZ '2026-05-18 01:10:22+00', TIMESTAMPTZ '2026-05-18 01:00:00+00'
FROM members WHERE member_no = 'M00001'
ON CONFLICT (order_no) DO NOTHING;

INSERT INTO withdraw_orders (order_no, member_id, amount, channel, status, paid_at, created_at)
SELECT 'WD00001', id, 800.00, 'bank_card', 'paid', TIMESTAMPTZ '2026-05-17 14:00:00+00', TIMESTAMPTZ '2026-05-17 13:00:00+00'
FROM members WHERE member_no = 'M00001'
ON CONFLICT (order_no) DO NOTHING;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DELETE FROM withdraw_orders WHERE order_no = 'WD00001';
DELETE FROM recharge_orders WHERE order_no = 'RC-MOCK-JIA-20260519';
DELETE FROM wallet_ledger WHERE ledger_no IN ('ML000010001', 'ML000010002', 'ML000010003', 'ML000010004');
DELETE FROM member_wallets WHERE member_id IN (SELECT id FROM members WHERE member_no IN ('M00001', 'M00002', 'M00003'));
DELETE FROM members WHERE member_no IN ('M00001', 'M00002', 'M00003');
-- +goose StatementEnd
