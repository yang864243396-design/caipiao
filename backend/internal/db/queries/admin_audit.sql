-- name: ListAdminAuditLogs :many
SELECT id, actor, action, ip, created_at
FROM admin_audit_logs
ORDER BY created_at DESC, id DESC
LIMIT sqlc.arg(row_limit);

-- name: InsertAdminAuditLog :one
INSERT INTO admin_audit_logs (id, actor, action, ip)
VALUES (
    'AUD' || LPAD(nextval('admin_audit_log_seq')::text, 5, '0'),
    $1, $2, $3
)
RETURNING id, actor, action, ip, created_at;
