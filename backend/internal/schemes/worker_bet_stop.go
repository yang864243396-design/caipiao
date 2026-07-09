package schemes

import (
	"context"
	"errors"

	"caipiao/backend/internal/db/sqlcdb"
	"caipiao/backend/internal/guajibet"
	"caipiao/backend/internal/member"
)

// errSchemeBetStopped 表示 worker 已将方案停投（pending），外层无需重复处理。
var errSchemeBetStopped = errors.New("scheme instance stopped after bet failure")

func betFailureReason(err error) string {
	if errors.Is(err, member.ErrInsufficientFunds) {
		return StatusReasonInsufficientFunds
	}
	return StatusReasonBetFailed
}

func applyBetFailurePause(
	ctx context.Context,
	qtx *sqlcdb.Queries,
	inst sqlcdb.SchemeInstance,
	amount float64,
	reason, detail string,
) error {
	switch reason {
	case StatusReasonInsufficientFunds:
		if err := pauseInstanceForInsufficientFunds(ctx, qtx, inst.ID); err != nil {
			return err
		}
		return appendInsufficientFundsAudit(ctx, qtx, inst, amount)
	default:
		return pauseInstanceForBetFailed(ctx, qtx, inst.ID, detail)
	}
}

func (w *Worker) stopAfterThirdPartyBetFailed(
	ctx context.Context,
	qtx *sqlcdb.Queries,
	inst sqlcdb.SchemeInstance,
	amount float64,
	betErr error,
) error {
	if errors.Is(betErr, guajibet.ErrPeriodClosed) {
		return guajibet.ErrPeriodClosed
	}
	reason := betFailureReason(betErr)
	detail := guajiBetFailedDetail(betErr)
	if err := applyBetFailurePause(ctx, qtx, inst, amount, reason, detail); err != nil {
		return err
	}
	return errSchemeBetStopped
}
