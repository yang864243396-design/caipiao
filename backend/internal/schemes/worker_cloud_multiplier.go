package schemes

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
)

func (w *Worker) memberPlanMultiplier(ctx context.Context, memberID int64) float64 {
	if w == nil || w.q == nil || memberID <= 0 {
		return 1
	}
	row, err := w.q.GetMemberCloudSettings(ctx, memberID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 1
		}
		return 1
	}
	return planBaseCoef(numericToFloat(row.PlanMultiplier))
}
