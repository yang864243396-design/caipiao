package schemes

import (
	"context"
	"strings"
)

// ProcessDraw is the synchronous durable-consumer boundary. The caller may ACK
// the JetStream message only after this method returns nil; PostgreSQL CAS and
// unique keys still make duplicate deliveries harmless.
func (p *StrategyProcessor) ProcessDraw(ctx context.Context, lotteryCode, periodNo string) error {
	if p == nil {
		return nil
	}
	lotteryCode = strings.TrimSpace(lotteryCode)
	periodNo = strings.TrimSpace(periodNo)
	if lotteryCode == "" || periodNo == "" {
		return nil
	}
	p.lifecycleMu.Lock()
	closing := p.closing
	p.lifecycleMu.Unlock()
	if closing {
		return context.Canceled
	}
	return p.recoverPendingScope(ctx, lotteryCode, periodNo)
}

func (w *Worker) ProcessDraw(ctx context.Context, lotteryCode, periodNo string) error {
	if w == nil || w.strategyProcessor == nil {
		return nil
	}
	return w.strategyProcessor.ProcessDraw(ctx, lotteryCode, periodNo)
}
