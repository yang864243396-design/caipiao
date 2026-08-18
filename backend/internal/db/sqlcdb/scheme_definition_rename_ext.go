package sqlcdb

import (
	"context"
	"strings"
)

// RenameSchemeDefinitionAndInstances 更新定义名称，并同步关联云端实例展示名。
func (q *Queries) RenameSchemeDefinitionAndInstances(ctx context.Context, memberID int64, definitionID, schemeName string) error {
	definitionID = strings.TrimSpace(definitionID)
	schemeName = strings.TrimSpace(schemeName)
	if definitionID == "" || schemeName == "" {
		return nil
	}
	_, err := q.db.Exec(ctx, `
WITH renamed_definition AS (
UPDATE scheme_definitions
SET scheme_name = $3,
    updated_at = now()
WHERE id = $1 AND member_id = $2
RETURNING id
)
UPDATE scheme_instances
SET scheme_name = $3,
    updated_at = now()
WHERE definition_id IN (SELECT id FROM renamed_definition)`, definitionID, memberID, schemeName)
	return err
}
