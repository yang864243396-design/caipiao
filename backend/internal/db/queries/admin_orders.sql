-- name: CountAdminBetOrders :one
SELECT COUNT(*)::bigint
FROM bet_orders b
INNER JOIN members m ON m.id = b.member_id
LEFT JOIN cloud_bet_records c ON c.bet_order_no = b.order_no
WHERE (
    sqlc.narg(issue_no)::text IS NULL
    OR sqlc.narg(issue_no)::text = ''
    OR b.issue_no ILIKE '%' || sqlc.narg(issue_no)::text || '%'
)
AND (
    sqlc.narg(member_account)::text IS NULL
    OR sqlc.narg(member_account)::text = ''
    OR m.account ILIKE '%' || sqlc.narg(member_account)::text || '%'
)
AND (
    sqlc.narg(scheme_name)::text IS NULL
    OR sqlc.narg(scheme_name)::text = ''
    OR c.scheme_name ILIKE '%' || sqlc.narg(scheme_name)::text || '%'
)
AND (
    sqlc.narg(lottery_code)::text IS NULL
    OR sqlc.narg(lottery_code)::text = ''
    OR b.lottery_code = sqlc.narg(lottery_code)::text
);

-- name: ListAdminBetOrders :many
SELECT
    COALESCE(
        NULLIF(TRIM(b.third_party_bet_id), ''),
        NULLIF(TRIM(c.third_party_bet_id), ''),
        ''
    ) AS third_party_bet_id,
    b.issue_no,
    m.account,
    b.lottery_name,
    COALESCE(c.scheme_name, '') AS scheme_name,
    b.amount::float8 AS amount,
    CASE
        WHEN b.status = 'win' THEN (b.amount + b.pnl)::float8
        ELSE 0
    END AS payout_amount,
    b.status,
    b.placed_at
FROM bet_orders b
INNER JOIN members m ON m.id = b.member_id
LEFT JOIN cloud_bet_records c ON c.bet_order_no = b.order_no
WHERE (
    sqlc.narg(issue_no)::text IS NULL
    OR sqlc.narg(issue_no)::text = ''
    OR b.issue_no ILIKE '%' || sqlc.narg(issue_no)::text || '%'
)
AND (
    sqlc.narg(member_account)::text IS NULL
    OR sqlc.narg(member_account)::text = ''
    OR m.account ILIKE '%' || sqlc.narg(member_account)::text || '%'
)
AND (
    sqlc.narg(scheme_name)::text IS NULL
    OR sqlc.narg(scheme_name)::text = ''
    OR c.scheme_name ILIKE '%' || sqlc.narg(scheme_name)::text || '%'
)
AND (
    sqlc.narg(lottery_code)::text IS NULL
    OR sqlc.narg(lottery_code)::text = ''
    OR b.lottery_code = sqlc.narg(lottery_code)::text
)
ORDER BY b.placed_at DESC, b.id DESC
LIMIT sqlc.arg(row_limit) OFFSET sqlc.arg(row_offset);

-- name: CountAdminChaseOrders :one
SELECT COUNT(*)::bigint
FROM chase_orders c
INNER JOIN members m ON m.id = c.member_id
WHERE (
    sqlc.narg(chase_no)::text IS NULL
    OR sqlc.narg(chase_no)::text = ''
    OR c.chase_no ILIKE '%' || sqlc.narg(chase_no)::text || '%'
)
AND (
    sqlc.narg(member_account)::text IS NULL
    OR sqlc.narg(member_account)::text = ''
    OR m.account ILIKE '%' || sqlc.narg(member_account)::text || '%'
)
AND (
    sqlc.narg(status)::text IS NULL
    OR sqlc.narg(status)::text = ''
    OR c.status = sqlc.narg(status)::text
)
AND (
    sqlc.narg(lottery_code)::text IS NULL
    OR sqlc.narg(lottery_code)::text = ''
    OR c.lottery_code = sqlc.narg(lottery_code)::text
);

-- name: ListAdminChaseOrders :many
SELECT
    c.chase_no,
    m.account,
    c.lottery_name,
    c.total_issues,
    c.done_issues,
    c.amount::float8 AS amount,
    c.status,
    c.started_at
FROM chase_orders c
INNER JOIN members m ON m.id = c.member_id
WHERE (
    sqlc.narg(chase_no)::text IS NULL
    OR sqlc.narg(chase_no)::text = ''
    OR c.chase_no ILIKE '%' || sqlc.narg(chase_no)::text || '%'
)
AND (
    sqlc.narg(member_account)::text IS NULL
    OR sqlc.narg(member_account)::text = ''
    OR m.account ILIKE '%' || sqlc.narg(member_account)::text || '%'
)
AND (
    sqlc.narg(status)::text IS NULL
    OR sqlc.narg(status)::text = ''
    OR c.status = sqlc.narg(status)::text
)
AND (
    sqlc.narg(lottery_code)::text IS NULL
    OR sqlc.narg(lottery_code)::text = ''
    OR c.lottery_code = sqlc.narg(lottery_code)::text
)
ORDER BY c.started_at DESC, c.id DESC
LIMIT sqlc.arg(row_limit) OFFSET sqlc.arg(row_offset);

-- name: CountAdminLedgerEntries :one
SELECT COUNT(*)::bigint
FROM wallet_ledger l
INNER JOIN members m ON m.id = l.member_id
WHERE (
    sqlc.narg(keyword)::text IS NULL
    OR sqlc.narg(keyword)::text = ''
    OR l.ledger_no ILIKE '%' || sqlc.narg(keyword)::text || '%'
    OR m.id::text ILIKE '%' || sqlc.narg(keyword)::text || '%'
    OR m.account ILIKE '%' || sqlc.narg(keyword)::text || '%'
    OR m.display_name ILIKE '%' || sqlc.narg(keyword)::text || '%'
    OR COALESCE(l.order_ref, '') ILIKE '%' || sqlc.narg(keyword)::text || '%'
);

-- name: ListAdminLedgerEntries :many
SELECT
    l.ledger_no,
    m.display_name,
    l.txn_type,
    l.delta_amount::float8 AS delta_amount,
    l.created_at
FROM wallet_ledger l
INNER JOIN members m ON m.id = l.member_id
WHERE (
    sqlc.narg(keyword)::text IS NULL
    OR sqlc.narg(keyword)::text = ''
    OR l.ledger_no ILIKE '%' || sqlc.narg(keyword)::text || '%'
    OR m.id::text ILIKE '%' || sqlc.narg(keyword)::text || '%'
    OR m.account ILIKE '%' || sqlc.narg(keyword)::text || '%'
    OR m.display_name ILIKE '%' || sqlc.narg(keyword)::text || '%'
    OR COALESCE(l.order_ref, '') ILIKE '%' || sqlc.narg(keyword)::text || '%'
)
ORDER BY l.created_at DESC, l.id DESC
LIMIT sqlc.arg(row_limit) OFFSET sqlc.arg(row_offset);
