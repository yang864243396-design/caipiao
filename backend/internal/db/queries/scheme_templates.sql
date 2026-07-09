-- name: ListSchemeTemplatesAdmin :many
SELECT
    t.id,
    t.name,
    t.lottery_code,
    COALESCE(c.display_name, t.lottery_code) AS lottery_label,
    COALESCE(t.brief, '') AS brief,
    t.sort_order,
    t.enabled,
    t.config,
    t.member_id,
    t.definition_id,
    t.created_at,
    t.updated_at
FROM scheme_templates t
LEFT JOIN lottery_catalog c ON c.code = t.lottery_code
ORDER BY t.sort_order ASC, t.name ASC;

-- name: CountSchemeTemplatesAdminPlatform :one
SELECT COUNT(*)::bigint AS total
FROM scheme_templates t
WHERE t.member_id IS NULL AND t.definition_id IS NULL
  AND (sqlc.arg(name_keyword) = '' OR t.name ILIKE '%' || sqlc.arg(name_keyword) || '%');

-- name: ListSchemeTemplatesAdminPlatformPaged :many
SELECT
    t.id,
    t.name,
    t.lottery_code,
    COALESCE(c.display_name, t.lottery_code) AS lottery_label,
    COALESCE(t.brief, '') AS brief,
    t.sort_order,
    t.enabled,
    t.config,
    t.member_id,
    t.definition_id,
    t.created_at,
    t.updated_at
FROM scheme_templates t
LEFT JOIN lottery_catalog c ON c.code = t.lottery_code
WHERE t.member_id IS NULL AND t.definition_id IS NULL
  AND (sqlc.arg(name_keyword) = '' OR t.name ILIKE '%' || sqlc.arg(name_keyword) || '%')
ORDER BY t.sort_order ASC, t.name ASC
LIMIT sqlc.arg(page_limit) OFFSET sqlc.arg(page_offset);

-- name: UpdateSchemeTemplatePlatform :one
UPDATE scheme_templates
SET
    name = $2,
    brief = $3,
    sort_order = $4,
    enabled = $5,
    config = $6,
    updated_at = now()
WHERE id = $1 AND member_id IS NULL AND definition_id IS NULL
RETURNING id, name, lottery_code, brief, sort_order, enabled, config, member_id, definition_id, created_at, updated_at;

-- name: ListSchemeTemplatesPlatformEnabled :many
SELECT
    t.id,
    t.name,
    t.lottery_code,
    COALESCE(c.display_name, t.lottery_code) AS lottery_label,
    COALESCE(t.brief, '') AS brief,
    t.sort_order,
    t.enabled,
    t.config,
    t.member_id,
    t.definition_id,
    t.created_at,
    t.updated_at
FROM scheme_templates t
LEFT JOIN lottery_catalog c ON c.code = t.lottery_code
WHERE t.enabled = true AND t.definition_id IS NULL AND t.member_id IS NULL
ORDER BY t.sort_order ASC, t.name ASC;

-- name: ListSchemeTemplatesForDefinition :many
SELECT
    t.id,
    t.name,
    t.lottery_code,
    COALESCE(c.display_name, t.lottery_code) AS lottery_label,
    COALESCE(t.brief, '') AS brief,
    t.sort_order,
    t.enabled,
    t.config,
    t.member_id,
    t.definition_id,
    t.created_at,
    t.updated_at
FROM scheme_templates t
LEFT JOIN lottery_catalog c ON c.code = t.lottery_code
WHERE (t.enabled = true AND t.definition_id IS NULL AND t.member_id IS NULL)
   OR (
        t.definition_id = sqlc.arg(definition_id)
        AND EXISTS (
            SELECT 1 FROM scheme_definitions d
            WHERE d.id = t.definition_id AND d.member_id = sqlc.arg(member_id)
        )
   )
ORDER BY t.sort_order ASC, t.name ASC;

-- name: GetSchemeTemplateByID :one
SELECT
    t.id,
    t.name,
    t.lottery_code,
    COALESCE(c.display_name, t.lottery_code) AS lottery_label,
    COALESCE(t.brief, '') AS brief,
    t.sort_order,
    t.enabled,
    t.config,
    t.member_id,
    t.definition_id,
    t.created_at,
    t.updated_at
FROM scheme_templates t
LEFT JOIN lottery_catalog c ON c.code = t.lottery_code
WHERE t.id = $1;

-- name: UpsertSchemeTemplate :one
INSERT INTO scheme_templates (
    id, name, lottery_code, brief, sort_order, enabled, config, member_id, definition_id, created_at, updated_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, now(), now()
)
ON CONFLICT (id) DO UPDATE SET
    name = EXCLUDED.name,
    lottery_code = EXCLUDED.lottery_code,
    brief = EXCLUDED.brief,
    sort_order = EXCLUDED.sort_order,
    enabled = EXCLUDED.enabled,
    config = EXCLUDED.config,
    member_id = COALESCE(scheme_templates.member_id, EXCLUDED.member_id),
    definition_id = COALESCE(scheme_templates.definition_id, EXCLUDED.definition_id),
    updated_at = now()
RETURNING id, name, lottery_code, brief, sort_order, enabled, config, member_id, definition_id, created_at, updated_at;

-- name: UpdateSchemeTemplateDefinitionOwned :one
UPDATE scheme_templates t
SET
    name = $4,
    config = $5,
    brief = $6,
    updated_at = now()
FROM scheme_definitions d
WHERE t.id = $1
  AND t.definition_id = $2
  AND d.id = t.definition_id
  AND d.member_id = $3
RETURNING t.id, t.name, t.lottery_code, t.brief, t.sort_order, t.enabled, t.config, t.member_id, t.definition_id, t.created_at, t.updated_at;

-- name: DeleteSchemeTemplate :execrows
DELETE FROM scheme_templates WHERE id = $1;

-- name: DeleteAllSchemeTemplates :exec
DELETE FROM scheme_templates;
