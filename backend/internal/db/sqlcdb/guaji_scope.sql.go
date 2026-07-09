package sqlcdb

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
)

const getActiveGuajiAccountID = `-- name: GetActiveGuajiAccountID :one
SELECT id FROM member_guaji_accounts
WHERE member_id = $1 AND is_active = true
LIMIT 1
`

func (q *Queries) GetActiveGuajiAccountID(ctx context.Context, memberID int64) (int64, error) {
	var id int64
	err := q.db.QueryRow(ctx, getActiveGuajiAccountID, memberID).Scan(&id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, err
		}
		return 0, err
	}
	return id, nil
}
