package member

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"caipiao/backend/internal/db/sqlcdb"
)

// LookupActiveGuajiAccountID 返回会员当前启用的第三方授权账号 id。
func LookupActiveGuajiAccountID(ctx context.Context, q *sqlcdb.Queries, memberID int64) (pgtype.Int8, error) {
	if q == nil || memberID <= 0 {
		return pgtype.Int8{}, nil
	}
	id, err := q.GetActiveGuajiAccountID(ctx, memberID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return pgtype.Int8{}, nil
		}
		return pgtype.Int8{}, err
	}
	return pgtype.Int8{Int64: id, Valid: true}, nil
}
