package schemes

import (
	"context"
	"strings"
)

func (p *StrategyProcessor) ProcessStrategyReady(ctx context.Context, recordID int64, schemeID, lotteryCode, periodNo string, expectedStateVersion int64) error {
	if p == nil || p.q == nil || recordID <= 0 || expectedStateVersion < 0 {
		return nil
	}
	schemeID = strings.TrimSpace(schemeID)
	lotteryCode = strings.TrimSpace(lotteryCode)
	periodNo = strings.TrimSpace(periodNo)
	if schemeID == "" || lotteryCode == "" || periodNo == "" {
		return nil
	}
	p.lifecycleMu.Lock()
	closing := p.closing
	p.lifecycleMu.Unlock()
	if closing {
		return context.Canceled
	}
	row, found, err := p.q.PendingFormalStrategyRowForSchemeDraw(
		ctx, recordID, schemeID, lotteryCode, periodNo, expectedStateVersion,
	)
	if err != nil {
		return err
	}
	if !found {
		if p.formalModeForLottery(lotteryCode) == "" {
			return nil
		}
		tx, err := p.pool.Begin(ctx)
		if err != nil {
			return err
		}
		defer tx.Rollback(ctx)
		qtx := p.q.WithTx(tx)
		evidence, err := p.validateExistingFormalPhaseOneConflict(ctx, qtx, recordID, schemeID, lotteryCode, periodNo, &expectedStateVersion)
		if err != nil {
			return err
		}
		if err := p.retryExpiredFormalWait(ctx, qtx, evidence); err != nil {
			return err
		}
		return tx.Commit(ctx)
	}
	return p.process(ctx, row)
}

func (w *Worker) ProcessStrategyReady(ctx context.Context, recordID int64, schemeID, lotteryCode, periodNo string, expectedStateVersion int64) error {
	if w == nil || w.strategyProcessor == nil {
		return nil
	}
	return w.strategyProcessor.ProcessStrategyReady(ctx, recordID, schemeID, lotteryCode, periodNo, expectedStateVersion)
}
