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
	tag, err := q.db.Exec(ctx, `
UPDATE scheme_definitions
SET scheme_name = $3,
    updated_at = now()
WHERE id = $1 AND member_id = $2`, definitionID, memberID, schemeName)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return nil
	}
	_, err = q.db.Exec(ctx, `
UPDATE scheme_instances
SET scheme_name = $2,
    updated_at = now()
WHERE definition_id = $1`, definitionID, schemeName)
	return err
}
