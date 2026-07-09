-- name: ListCustomerServiceAgents :many
SELECT id, name, tg_link, work_hours, sort, enabled, created_at, updated_at
FROM cms_customer_service_agents
ORDER BY sort ASC, id ASC;

-- name: ListEnabledCustomerServiceAgents :many
SELECT id, name, tg_link, work_hours, sort
FROM cms_customer_service_agents
WHERE enabled = true
ORDER BY sort ASC, id ASC;

-- name: UpsertCustomerServiceAgent :one
INSERT INTO cms_customer_service_agents (id, name, tg_link, work_hours, sort, enabled, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, now(), now())
ON CONFLICT (id) DO UPDATE SET
    name = EXCLUDED.name,
    tg_link = EXCLUDED.tg_link,
    work_hours = EXCLUDED.work_hours,
    sort = EXCLUDED.sort,
    enabled = EXCLUDED.enabled,
    updated_at = now()
RETURNING id, name, tg_link, work_hours, sort, enabled, created_at, updated_at;

-- name: DeleteCustomerServiceAgent :exec
DELETE FROM cms_customer_service_agents WHERE id = $1;
