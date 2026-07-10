-- name: ListMemberGuajiAccountsByMember :many
SELECT
    id, member_id, guaji_username, password_enc, mfa_material_enc,
    access_token_enc, refresh_token_enc, token_expires_at, is_active,
    bound_at, last_sync_at, last_token_error, last_bet_at, reauth_fail_count,
    created_at, updated_at
FROM member_guaji_accounts
WHERE member_id = $1
ORDER BY is_active DESC, bound_at DESC;

-- name: GetMemberGuajiAccountByIDAndMember :one
SELECT
    id, member_id, guaji_username, password_enc, mfa_material_enc,
    access_token_enc, refresh_token_enc, token_expires_at, is_active,
    bound_at, last_sync_at, last_token_error, last_bet_at, reauth_fail_count,
    created_at, updated_at
FROM member_guaji_accounts
WHERE id = $1 AND member_id = $2;

-- name: GetMemberGuajiAccountByUsername :one
SELECT
    id, member_id, guaji_username, password_enc, mfa_material_enc,
    access_token_enc, refresh_token_enc, token_expires_at, is_active,
    bound_at, last_sync_at, last_token_error, last_bet_at, reauth_fail_count,
    created_at, updated_at
FROM member_guaji_accounts
WHERE guaji_username = $1;

-- name: GetActiveMemberGuajiAccount :one
SELECT
    id, member_id, guaji_username, password_enc, mfa_material_enc,
    access_token_enc, refresh_token_enc, token_expires_at, is_active,
    bound_at, last_sync_at, last_token_error, last_bet_at, reauth_fail_count,
    created_at, updated_at
FROM member_guaji_accounts
WHERE member_id = $1 AND is_active = true
LIMIT 1;

-- name: CountMemberGuajiAccounts :one
SELECT COUNT(*)::bigint AS count
FROM member_guaji_accounts
WHERE member_id = $1;

-- name: InsertMemberGuajiAccount :one
INSERT INTO member_guaji_accounts (
    member_id, guaji_username, password_enc, mfa_material_enc,
    access_token_enc, refresh_token_enc, token_expires_at, is_active,
    bound_at, last_sync_at, last_token_error
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, now(), $9, NULL
)
RETURNING
    id, member_id, guaji_username, password_enc, mfa_material_enc,
    access_token_enc, refresh_token_enc, token_expires_at, is_active,
    bound_at, last_sync_at, last_token_error, last_bet_at, reauth_fail_count,
    created_at, updated_at;

-- name: DeactivateAllMemberGuajiAccounts :exec
UPDATE member_guaji_accounts
SET is_active = false, updated_at = now()
WHERE member_id = $1 AND is_active = true;

-- name: ActivateMemberGuajiAccount :one
UPDATE member_guaji_accounts
SET is_active = true,
    last_token_error = NULL,
    updated_at = now()
WHERE id = $1 AND member_id = $2
RETURNING
    id, member_id, guaji_username, password_enc, mfa_material_enc,
    access_token_enc, refresh_token_enc, token_expires_at, is_active,
    bound_at, last_sync_at, last_token_error, last_bet_at, reauth_fail_count,
    created_at, updated_at;

-- name: UpdateMemberGuajiAccountTokens :one
UPDATE member_guaji_accounts
SET access_token_enc = $3,
    refresh_token_enc = $4,
    token_expires_at = $5,
    last_sync_at = now(),
    last_token_error = NULL,
    reauth_fail_count = 0,
    updated_at = now()
WHERE id = $1 AND member_id = $2
RETURNING
    id, member_id, guaji_username, password_enc, mfa_material_enc,
    access_token_enc, refresh_token_enc, token_expires_at, is_active,
    bound_at, last_sync_at, last_token_error, last_bet_at, reauth_fail_count,
    created_at, updated_at;

-- name: UpdateMemberGuajiAccountTokenError :exec
UPDATE member_guaji_accounts
SET last_token_error = $3,
    reauth_fail_count = reauth_fail_count + 1,
    updated_at = now()
WHERE id = $1 AND member_id = $2;

-- name: UpdateMemberGuajiAccountMFAMaterial :exec
UPDATE member_guaji_accounts
SET mfa_material_enc = $3,
    updated_at = now()
WHERE id = $1 AND member_id = $2;

-- name: DeleteMemberGuajiAccount :execrows
DELETE FROM member_guaji_accounts
WHERE id = $1 AND member_id = $2;

-- name: DeleteAllMemberGuajiAccountsByMemberID :execrows
DELETE FROM member_guaji_accounts
WHERE member_id = $1;

-- name: ListMemberGuajiAccountsAdmin :many
SELECT
    id, member_id, guaji_username, is_active, bound_at, last_sync_at,
    last_token_error, last_bet_at, created_at, updated_at
FROM member_guaji_accounts
WHERE member_id = $1
ORDER BY is_active DESC, bound_at DESC;

-- name: UpdateMemberGuajiAccountBalances :exec
UPDATE member_guaji_accounts
SET balance_usdt = $2,
    balance_trx = $3,
    balance_cny = $4,
    updated_at = now()
WHERE id = $1;

-- name: ListActiveGuajiBalancesByMemberIDs :many
SELECT
    member_id,
    balance_usdt::float8 AS balance_usdt,
    balance_trx::float8 AS balance_trx,
    balance_cny::float8 AS balance_cny
FROM member_guaji_accounts
WHERE member_id = ANY($1::bigint[])
  AND is_active = true;
