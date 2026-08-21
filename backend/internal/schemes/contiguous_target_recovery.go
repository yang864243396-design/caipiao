package schemes

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"caipiao/backend/internal/db/sqlcdb"
)

const contiguousTargetRecoveryPageSize int32 = 32

type awaitingContiguousTargetSource interface {
	ListAwaitingContiguousTargets(context.Context, []string, []int32, int64, int32) ([]sqlcdb.AwaitingContiguousTargetRow, error)
}

type awaitingContiguousTargetResolver interface {
	ResolveAwaitingTarget(context.Context, int64) error
}

func runContiguousTargetRecoveryBatch(
	ctx context.Context,
	source awaitingContiguousTargetSource,
	resolver awaitingContiguousTargetResolver,
	lotteryCodes []string,
	shards []int32,
	batch int,
) (int, error) {
	if source == nil || resolver == nil || len(lotteryCodes) == 0 || len(shards) == 0 {
		return 0, nil
	}
	if batch <= 0 || batch > int(contiguousTargetRecoveryPageSize) {
		batch = int(contiguousTargetRecoveryPageSize)
	}
	rows, err := source.ListAwaitingContiguousTargets(ctx, lotteryCodes, shards, 0, int32(batch))
	if err != nil {
		return 0, err
	}
	processed := 0
	for _, row := range rows {
		if err := ctx.Err(); err != nil {
			return processed, err
		}
		if err := resolver.ResolveAwaitingTarget(ctx, row.DecisionID); err != nil {
			return processed, err
		}
		processed++
	}
	return processed, nil
}

// RunContiguousTargetRecovery is the bounded database safety net for durable
// target waits. It intentionally reuses the normal resolver and does no
// provider work of its own.
func (w *Worker) RunContiguousTargetRecovery(
	ctx context.Context,
	lotteryCodes []string,
	shards []int32,
	batch, concurrency int,
	interval time.Duration,
) {
	_ = concurrency // resolution is deliberately sequential within one bounded page.
	if w == nil || w.q == nil || w.strategyProcessor == nil {
		return
	}
	if interval < time.Second {
		interval = time.Second
	}
	run := func() {
		_, err := runContiguousTargetRecoveryBatch(ctx, w.q, w.strategyProcessor, lotteryCodes, shards, batch)
		if err != nil && !errors.Is(err, context.Canceled) {
			slog.Warn("contiguous target recovery scan failed", "err", err)
		}
	}
	run()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			run()
		}
	}
}
