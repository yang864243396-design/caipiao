-- name: GetMemberByAccount :one
SELECT
    m.id,
    m.account,
    m.display_name,
    m.status,
    m.registered_at,
    m.last_login_at
FROM members m
WHERE m.account = $1
  AND m.status = 'active';

-- name: GetMemberAccountByID :one
SELECT account FROM members WHERE id = $1;

-- name: GetMemberForLogin :one
SELECT
    m.id,
    m.account,
    m.display_name,
    m.password_hash,
    m.status
FROM members m
WHERE m.account = $1;

-- name: TouchMemberLastLogin :exec
UPDATE members
SET last_login_at = now(),
    updated_at = now()
WHERE id = $1;

-- name: GetMemberWalletByMemberID :one
SELECT
    w.balance::float8 AS balance,
    w.frozen_balance::float8 AS frozen_balance,
    w.currency,
    w.version,
    w.updated_at
FROM member_wallets w
WHERE w.member_id = $1;

-- name: GetMemberProfileByAccount :one
SELECT
    m.id,
    m.account,
    m.display_name,
    w.balance::float8 AS balance,
    w.frozen_balance::float8 AS frozen_balance,
    w.currency,
    (m.fund_password_hash IS NOT NULL) AS has_fund_password
FROM members m
JOIN member_wallets w ON w.member_id = m.id
WHERE m.account = $1
  AND m.status = 'active';

-- name: GetMemberFundAuth :one
SELECT m.id, m.fund_password_hash
FROM members m
WHERE m.account = $1
  AND m.status = 'active';

-- name: SetMemberFundPassword :execrows
UPDATE members
SET fund_password_hash = $2,
    updated_at = now()
WHERE id = $1;

-- name: ListWalletLedger :many
SELECT
    l.id,
    l.ledger_no,
    l.txn_type,
    l.delta_amount::float8 AS delta_amount,
    l.balance_after::float8 AS balance_after,
    COALESCE(l.order_ref, '') AS order_ref,
    l.created_at
FROM wallet_ledger l
WHERE l.member_id = $1
  AND l.created_at >= sqlc.arg(time_from)
  AND l.created_at < sqlc.arg(time_to)
  AND (
    sqlc.narg(txn_type)::text IS NULL
    OR l.txn_type = sqlc.narg(txn_type)::text
  )
  AND (
    sqlc.narg(order_ref)::text IS NULL
    OR sqlc.narg(order_ref)::text = ''
    OR l.order_ref ILIKE '%' || sqlc.narg(order_ref)::text || '%'
  )
ORDER BY l.created_at DESC, l.id DESC
LIMIT sqlc.arg(row_limit);

-- name: ListWalletLedgerAfterCursor :many
SELECT
    l.id,
    l.ledger_no,
    l.txn_type,
    l.delta_amount::float8 AS delta_amount,
    l.balance_after::float8 AS balance_after,
    COALESCE(l.order_ref, '') AS order_ref,
    l.created_at
FROM wallet_ledger l
WHERE l.member_id = $1
  AND l.created_at >= sqlc.arg(time_from)
  AND l.created_at < sqlc.arg(time_to)
  AND (
    sqlc.narg(txn_type)::text IS NULL
    OR l.txn_type = sqlc.narg(txn_type)::text
  )
  AND (
    sqlc.narg(order_ref)::text IS NULL
    OR sqlc.narg(order_ref)::text = ''
    OR l.order_ref ILIKE '%' || sqlc.narg(order_ref)::text || '%'
  )
  AND (
    l.created_at < sqlc.arg(cursor_time)
    OR (l.created_at = sqlc.arg(cursor_time) AND l.id < sqlc.arg(cursor_id))
  )
ORDER BY l.created_at DESC, l.id DESC
LIMIT sqlc.arg(row_limit);

-- name: GetWalletLedgerCursorAnchor :one
SELECT l.created_at, l.id
FROM wallet_ledger l
WHERE l.member_id = $1
  AND l.ledger_no = $2;

-- name: GetMemberByID :one
SELECT
    m.id,
    m.account,
    m.display_name,
    m.status,
    m.registered_at,
    m.last_login_at,
    COALESCE(w.balance, 0)::float8 AS balance
FROM members m
LEFT JOIN member_wallets w ON w.member_id = m.id
WHERE m.id = $1;

-- name: CountAdminMembers :one
SELECT COUNT(*)::bigint
FROM members m
WHERE (
    sqlc.narg(keyword)::text IS NULL
    OR sqlc.narg(keyword)::text = ''
    OR (
        sqlc.arg(search_field)::text = 'guajiAccount'
        AND EXISTS (
            SELECT 1
            FROM member_guaji_accounts g
            WHERE g.member_id = m.id
              AND g.guaji_username ILIKE '%' || sqlc.narg(keyword)::text || '%'
        )
    )
    OR (
        sqlc.arg(search_field)::text = 'account'
        AND m.account ILIKE '%' || sqlc.narg(keyword)::text || '%'
    )
    OR (
        sqlc.arg(search_field)::text = 'id'
        AND m.id::text = sqlc.narg(keyword)::text
    )
);

-- name: ListAdminMembers :many
SELECT
    m.id,
    m.account,
    m.display_name,
    m.status,
    m.registered_at,
    m.last_login_at
FROM members m
WHERE (
    sqlc.narg(keyword)::text IS NULL
    OR sqlc.narg(keyword)::text = ''
    OR (
        sqlc.arg(search_field)::text = 'guajiAccount'
        AND EXISTS (
            SELECT 1
            FROM member_guaji_accounts g
            WHERE g.member_id = m.id
              AND g.guaji_username ILIKE '%' || sqlc.narg(keyword)::text || '%'
        )
    )
    OR (
        sqlc.arg(search_field)::text = 'account'
        AND m.account ILIKE '%' || sqlc.narg(keyword)::text || '%'
    )
    OR (
        sqlc.arg(search_field)::text = 'id'
        AND m.id::text = sqlc.narg(keyword)::text
    )
)
ORDER BY m.registered_at DESC, m.id DESC
LIMIT sqlc.arg(row_limit) OFFSET sqlc.arg(row_offset);

-- name: ListWalletLedgerAdmin :many
SELECT
    l.ledger_no,
    l.txn_type,
    l.delta_amount::float8 AS delta_amount,
    l.balance_after::float8 AS balance_after,
    COALESCE(l.order_ref, '') AS order_ref,
    l.created_at
FROM wallet_ledger l
WHERE l.member_id = $1
  AND (
    sqlc.narg(txn_type)::text IS NULL
    OR sqlc.narg(txn_type)::text = ''
    OR l.txn_type = sqlc.narg(txn_type)::text
  )
ORDER BY l.created_at DESC, l.id DESC
LIMIT sqlc.arg(row_limit);

-- name: AdminUpdateMemberStatus :execrows
UPDATE members
SET status = $2,
    updated_at = now()
WHERE id = $1;

-- name: AdminUpdateMemberPasswordByID :execrows
UPDATE members
SET password_hash = $2,
    updated_at = now()
WHERE id = $1;

-- name: AdminInsertMember :one
INSERT INTO members (account, password_hash, display_name, status)
VALUES ($1, $2, $3, $4)
RETURNING id, account, display_name, status, registered_at, last_login_at;

-- name: AdminInsertMemberWallet :exec
INSERT INTO member_wallets (member_id, balance, frozen_balance, currency)
VALUES ($1, 0, 0, 'CNY');
