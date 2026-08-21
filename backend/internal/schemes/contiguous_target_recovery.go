package schemes

import (
	"context"
	"errors"
	"time"

	"caipiao/backend/internal/db/sqlcdb"
)

const contiguousTargetRecoveryPageSize int32 = 32

type awaitingContiguousTargetSource interface {
	AssertSchemeBettingShardLease(context.Context, string, int32, string, int64) error
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
	fence, ok := strategyLeaseFenceFromContext(ctx)
	if !ok || fence.Owner == "" || fence.Epoch <= 0 {
		return 0, errors.New("contiguous target recovery requires a strategy shard lease fence")
	}
	configured := false
	for _, shard := range shards {
		if shard == fence.ShardNo {
			configured = true
			break
		}
	}
	if !configured {
		return 0, errors.New("contiguous target recovery lease shard is not configured")
	}
	if err := source.AssertSchemeBettingShardLease(ctx, "strategy", fence.ShardNo, fence.Owner, fence.Epoch); err != nil {
		return 0, err
	}
	if batch <= 0 || batch > int(contiguousTargetRecoveryPageSize) {
		batch = int(contiguousTargetRecoveryPageSize)
	}
	rows, err := source.ListAwaitingContiguousTargets(ctx, lotteryCodes, []int32{fence.ShardNo}, 0, int32(batch))
	if err != nil {
		return 0, err
	}
	processed := 0
	for _, row := range rows {
		if err := ctx.Err(); err != nil {
			return processed, err
		}
		if row.ShardNo != fence.ShardNo {
			return processed, errors.New("contiguous target recovery source returned a cross-shard decision")
		}
		if err := resolver.ResolveAwaitingTarget(ctx, row.DecisionID); err != nil {
			return processed, err
		}
		processed++
	}
	return processed, nil
}

func runContiguousTargetRecoveryLoop(
	ctx context.Context,
	source awaitingContiguousTargetSource,
	resolver awaitingContiguousTargetResolver,
	lotteryCodes []string,
	shards []int32,
	batch int,
	interval time.Duration,
) error {
	if interval < time.Second {
		interval = time.Second
	}
	if _, err := runContiguousTargetRecoveryBatch(ctx, source, resolver, lotteryCodes, shards, batch); err != nil {
		return err
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if _, err := runContiguousTargetRecoveryBatch(ctx, source, resolver, lotteryCodes, shards, batch); err != nil {
				if errors.Is(err, context.Canceled) && ctx.Err() != nil {
					return nil
				}
				return err
			}
		}
	}
}

// RunContiguousTargetRecovery is the bounded database safety net for durable
// target waits. It intentionally reuses the normal resolver and does no
// provider work of its own. Any lease assertion failure stops the loop so its
// owner can reacquire an authoritative fence before retrying.
func (w *Worker) RunContiguousTargetRecovery(
	ctx context.Context,
	lotteryCodes []string,
	shards []int32,
	batch, concurrency int,
	interval time.Duration,
) error {
	_ = concurrency // resolution is deliberately sequential within one bounded page.
	if w == nil || w.q == nil || w.strategyProcessor == nil {
		return nil
	}
	return runContiguousTargetRecoveryLoop(ctx, w.q, w.strategyProcessor, lotteryCodes, shards, batch, interval)
}
