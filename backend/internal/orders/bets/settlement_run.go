package bets

import (
	"context"
	"log/slog"
	"time"
)

// RunSettlementWorker ticks pending bet settlement on interval (seconds).
func RunSettlementWorker(ctx context.Context, w *SettlementWorker, tickSec int) {
	if w == nil || tickSec <= 0 {
		return
	}
	ticker := time.NewTicker(time.Duration(tickSec) * time.Second)
	defer ticker.Stop()
	slog.Info("bet settlement worker started", "tickSec", tickSec)
	for {
		select {
		case <-ctx.Done():
			slog.Info("bet settlement worker stopped")
			return
		case <-ticker.C:
			w.Tick(ctx)
		}
	}
}
