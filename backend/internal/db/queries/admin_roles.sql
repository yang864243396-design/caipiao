-- name: ListAdminRoles :many
SELECT id, name, menu_paths, created_at, updated_at
FROM admin_roles
ORDER BY id ASC;

-- name: UpsertAdminRole :one
INSERT INTO admin_roles (id, name, menu_paths, created_at, updated_at)
VALUES ($1, $2, $3::jsonb, now(), now())
ON CONFLICT (id) DO UPDATE SET
    name = EXCLUDED.name,
    menu_paths = EXCLUDED.menu_paths,
    updated_at = now()
RETURNING id, name, menu_paths, created_at, updated_at;

-- name: DeleteAdminRole :execrows
DELETE FROM admin_roles WHERE id = $1;
