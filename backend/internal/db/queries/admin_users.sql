-- name: GetAdminUserForLogin :one
SELECT id, account, password_hash, display_name, role_id, status
FROM admin_users
WHERE account = $1;

-- name: TouchAdminLastLogin :exec
UPDATE admin_users
SET last_login_at = now(), updated_at = now()
WHERE id = $1;

-- name: ListAdminUsers :many
SELECT
    u.id,
    u.account,
    u.display_name,
    u.role_id,
    r.name AS role_name,
    u.status,
    u.last_login_at,
    u.created_at,
    u.updated_at
FROM admin_users u
JOIN admin_roles r ON r.id = u.role_id
ORDER BY u.id ASC;

-- name: GetAdminUserByID :one
SELECT
    u.id,
    u.account,
    u.display_name,
    u.role_id,
    r.name AS role_name,
    u.status,
    u.last_login_at,
    u.created_at,
    u.updated_at
FROM admin_users u
JOIN admin_roles r ON r.id = u.role_id
WHERE u.id = $1;

-- name: CreateAdminUser :one
INSERT INTO admin_users (account, password_hash, display_name, role_id, status)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, account, display_name, role_id, status, last_login_at, created_at, updated_at;

-- name: UpdateAdminUser :one
UPDATE admin_users
SET
    display_name = $2,
    role_id = $3,
    status = $4,
    updated_at = now()
WHERE id = $1
RETURNING id, account, display_name, role_id, status, last_login_at, created_at, updated_at;

-- name: UpdateAdminUserPassword :exec
UPDATE admin_users
SET password_hash = $2, updated_at = now()
WHERE id = $1;

-- name: DeleteAdminUser :execrows
DELETE FROM admin_users WHERE id = $1;
