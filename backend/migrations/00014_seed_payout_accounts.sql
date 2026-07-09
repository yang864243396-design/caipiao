-- +goose Up
-- +goose StatementBegin
INSERT INTO member_payout_accounts (
    member_id, account_type, label, holder_name, account_no,
    bank_name, branch_name, is_default, status, created_at, updated_at
)
SELECT
    m.id, 'bank_card', '招商银行储蓄卡', '会员甲', '6225881234568888',
    '招商银行', '深圳科技园支行', true, 'active',
    TIMESTAMPTZ '2026-03-01 08:00:00+00', TIMESTAMPTZ '2026-03-01 08:00:00+00'
FROM members m
WHERE m.member_no = 'M00001'
  AND NOT EXISTS (
    SELECT 1 FROM member_payout_accounts p
    WHERE p.member_id = m.id AND p.account_no = '6225881234568888'
  );

INSERT INTO member_payout_accounts (
    member_id, account_type, label, holder_name, account_no,
    bank_name, branch_name, is_default, status, created_at, updated_at
)
SELECT
    m.id, 'bank_card', '建设银行储蓄卡', '会员甲', '6217001234566688',
    '建设银行', '深圳南山支行', false, 'active',
    TIMESTAMPTZ '2026-04-10 09:00:00+00', TIMESTAMPTZ '2026-04-10 09:00:00+00'
FROM members m
WHERE m.member_no = 'M00001'
  AND NOT EXISTS (
    SELECT 1 FROM member_payout_accounts p
    WHERE p.member_id = m.id AND p.account_no = '6217001234566688'
  );

INSERT INTO member_payout_accounts (
    member_id, account_type, label, holder_name, account_no,
    bank_name, is_default, status, created_at, updated_at
)
SELECT
    m.id, 'usdt_trc20', 'USDT（TRC20）· 主地址', NULL, 'TXyz9kDemoTrc20AddressMain001',
    NULL, false, 'active',
    TIMESTAMPTZ '2026-04-15 10:00:00+00', TIMESTAMPTZ '2026-04-15 10:00:00+00'
FROM members m
WHERE m.member_no = 'M00001'
  AND NOT EXISTS (
    SELECT 1 FROM member_payout_accounts p
    WHERE p.member_id = m.id AND p.account_no = 'TXyz9kDemoTrc20AddressMain001'
  );

INSERT INTO member_payout_accounts (
    member_id, account_type, label, holder_name, account_no,
    bank_name, is_default, status, created_at, updated_at
)
SELECT
    m.id, 'usdt_bsc', 'USDT（BSC）· 备用地址', NULL, '0xBscDemoBackupAddress002abcdef',
    NULL, false, 'active',
    TIMESTAMPTZ '2026-04-20 11:00:00+00', TIMESTAMPTZ '2026-04-20 11:00:00+00'
FROM members m
WHERE m.member_no = 'M00001'
  AND NOT EXISTS (
    SELECT 1 FROM member_payout_accounts p
    WHERE p.member_id = m.id AND p.account_no = '0xBscDemoBackupAddress002abcdef'
  );
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DELETE FROM member_payout_accounts
WHERE member_id IN (SELECT id FROM members WHERE member_no = 'M00001');
-- +goose StatementEnd
