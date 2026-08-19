package sqlcdb

import (
	"context"
	"strings"
)

func (q *Queries) GetSchemeBettingOwner(ctx context.Context, schemeID string) (string, error) {
	var owner string
	err := q.db.QueryRow(ctx, `SELECT betting_owner FROM scheme_instances WHERE id = $1`, schemeID).Scan(&owner)
	return strings.ToLower(strings.TrimSpace(owner)), err
}
