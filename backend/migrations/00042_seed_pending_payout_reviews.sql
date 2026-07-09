-- +goose Up
-- +goose StatementBegin
INSERT INTO member_payout_accounts (
    member_id, account_type, label, holder_name, account_no,
    bank_name, branch_name, is_default, status, created_at, updated_at
)
SELECT
    m.id, 'bank_card', '工商银行储蓄卡', '会员乙', '6222021234567890',
    '工商银行', '北京朝阳支行', false, 'pending_review',
    TIMESTAMPTZ '2026-05-20 08:00:00+00', TIMESTAMPTZ '2026-05-20 08:00:00+00'
FROM members m
WHERE m.member_no = 'M00002'
  AND NOT EXISTS (
    SELECT 1 FROM member_payout_accounts p
    WHERE p.member_id = m.id AND p.account_no = '6222021234567890'
  );

INSERT INTO member_payout_accounts (
    member_id, account_type, label, holder_name, account_no,
    bank_name, branch_name, is_default, status, created_at, updated_at
)
SELECT
    m.id, 'bank_card', '农业银行储蓄卡', '会员丙', '6228481234561234',
    '农业银行', '上海浦东支行', false, 'pending_review',
    TIMESTAMPTZ '2026-05-22 09:30:00+00', TIMESTAMPTZ '2026-05-22 09:30:00+00'
FROM members m
WHERE m.member_no = 'M00003'
  AND NOT EXISTS (
    SELECT 1 FROM member_payout_accounts p
    WHERE p.member_id = m.id AND p.account_no = '6228481234561234'
  );
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DELETE FROM member_payout_accounts
WHERE account_no IN ('6222021234567890', '6228481234561234');
-- +goose StatementEnd
