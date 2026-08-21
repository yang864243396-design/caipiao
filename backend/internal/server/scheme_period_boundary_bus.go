package server

import (
	"context"
	"errors"
	"fmt"
	"time"

	"caipiao/backend/internal/db"
	"caipiao/backend/internal/db/sqlcdb"
	"caipiao/backend/internal/schemeeventbus"
	"caipiao/backend/internal/schemes"
)

const contiguousTargetBoundaryExpansionPageSize int32 = 32

type boundaryAwaitingTargetSource interface {
	ListAwaitingContiguousTargets(context.Context, []string, []int32, int64, int32) ([]sqlcdb.AwaitingContiguousTargetRow, error)
}

type contiguousTargetReadyPublisher interface {
	PublishContiguousTargetReady(context.Context, schemeeventbus.ContiguousTargetReady, uint32) error
}

// expandSchemePeriodBoundary publishes a single bounded page. Resolution is
// intentionally deferred to the shard consumer; this path performs no target
// lookup or provider operation.
func expandSchemePeriodBoundary(
	ctx context.Context,
	event schemeeventbus.PeriodBoundary,
	source boundaryAwaitingTargetSource,
	publisher contiguousTargetReadyPublisher,
	shards []int32,
	shardCount uint32,
) error {
	if source == nil || publisher == nil || event.LotteryCode == "" || event.Generation == 0 || shardCount == 0 || len(shards) == 0 {
		return errors.New("contiguous target boundary expander configuration is incomplete")
	}
	rows, err := source.ListAwaitingContiguousTargets(ctx, []string{event.LotteryCode}, shards, 0, contiguousTargetBoundaryExpansionPageSize)
	if err != nil {
		return err
	}
	for _, row := range rows {
		ready := schemeeventbus.ContiguousTargetReady{
			DecisionID: row.DecisionID, SchemeID: row.SchemeID, LotteryCode: row.LotteryCode,
			SourcePeriod: row.SourcePeriodNo, BoundaryGeneration: event.Generation,
		}
		if err := publisher.PublishContiguousTargetReady(ctx, ready, shardCount); err != nil {
			return err
		}
	}
	return nil
}

func runSchemePeriodBoundaryExpander(
	ctx context.Context,
	bus *schemeeventbus.Bus,
	pool *db.Pool,
	shards []int32,
	shardCount uint32,
) error {
	if bus == nil || pool == nil {
		return errors.New("contiguous target boundary expander configuration is incomplete")
	}
	q := sqlcdb.New(pool)
	return bus.ConsumePeriodBoundaries(ctx, schemeeventbus.ContiguousTargetExpanderDurable, func(messageContext context.Context, event schemeeventbus.PeriodBoundary) error {
		return expandSchemePeriodBoundary(messageContext, event, q, bus, shards, shardCount)
	})
}

type contiguousTargetWorker interface {
	ProcessContiguousTargetReady(context.Context, schemeeventbus.ContiguousTargetReady) error
	RunContiguousTargetRecovery(context.Context, []string, []int32, int, int, time.Duration) error
}

type leasedContiguousTargetWorker struct {
	worker contiguousTargetWorker
	fence  schemes.StrategyLeaseFence
}

func (worker leasedContiguousTargetWorker) ProcessContiguousTargetReady(ctx context.Context, event schemeeventbus.ContiguousTargetReady) error {
	return worker.worker.ProcessContiguousTargetReady(schemes.WithStrategyLeaseFence(ctx, worker.fence), event)
}

func (worker leasedContiguousTargetWorker) RunContiguousTargetRecovery(
	ctx context.Context,
	lotteryCodes []string,
	_ []int32,
	batch, concurrency int,
	interval time.Duration,
) error {
	return worker.worker.RunContiguousTargetRecovery(
		schemes.WithStrategyLeaseFence(ctx, worker.fence), lotteryCodes, []int32{worker.fence.ShardNo}, batch, concurrency, interval,
	)
}

func runSchemeContiguousTargetConsumer(ctx context.Context, bus *schemeeventbus.Bus, shard uint32, worker interface {
	ProcessContiguousTargetReady(context.Context, schemeeventbus.ContiguousTargetReady) error
}) error {
	if bus == nil || worker == nil {
		return nil
	}
	durable := fmt.Sprintf("scheme-contiguous-target-shard-%d", shard)
	err := bus.ConsumeContiguousTargetReady(ctx, shard, durable, func(messageContext context.Context, event schemeeventbus.ContiguousTargetReady) error {
		return worker.ProcessContiguousTargetReady(messageContext, event)
	})
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return nil
	}
	return err
}

func runLeasedSchemeContiguousTargetConsumer(
	ctx context.Context,
	bus *schemeeventbus.Bus,
	pool *db.Pool,
	shard uint32,
	owner string,
	leaseDuration time.Duration,
	worker contiguousTargetWorker,
	lotteryCodes []string,
	batch, concurrency int,
	recoveryInterval time.Duration,
) error {
	if pool == nil || owner == "" {
		return errors.New("contiguous target strategy lease configuration is incomplete")
	}
	if leaseDuration <= 0 {
		leaseDuration = 5 * time.Second
	}
	q := sqlcdb.New(pool)
	retry := time.NewTicker(leaseDuration / 3)
	defer retry.Stop()
	for {
		epoch, acquired, err := q.AcquireSchemeBettingShardLease(ctx, "strategy", int32(shard), owner, leaseDuration)
		if err != nil {
			return err
		}
		if !acquired {
			select {
			case <-ctx.Done():
				return nil
			case <-retry.C:
				continue
			}
		}
		err = holdSchemeContiguousTargetShardLease(
			ctx, bus, q, shard, owner, epoch, leaseDuration, worker,
			lotteryCodes, batch, concurrency, recoveryInterval,
		)
		if errors.Is(err, errContiguousTargetShardLeaseLost) {
			continue
		}
		return err
	}
}

var errContiguousTargetShardLeaseLost = errors.New("scheme contiguous-target shard lease lost")

func holdSchemeContiguousTargetShardLease(
	ctx context.Context,
	bus *schemeeventbus.Bus,
	q *sqlcdb.Queries,
	shard uint32,
	owner string,
	epoch int64,
	leaseDuration time.Duration,
	worker contiguousTargetWorker,
	lotteryCodes []string,
	batch, concurrency int,
	recoveryInterval time.Duration,
) error {
	leaseCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	done := make(chan error, 2)
	fencedWorker := leasedContiguousTargetWorker{worker: worker, fence: schemes.StrategyLeaseFence{ShardNo: int32(shard), Owner: owner, Epoch: epoch}}
	go func() { done <- runSchemeContiguousTargetConsumer(leaseCtx, bus, shard, fencedWorker) }()
	go func() {
		done <- fencedWorker.RunContiguousTargetRecovery(
			leaseCtx, lotteryCodes, []int32{int32(shard)}, batch, concurrency, recoveryInterval,
		)
	}()
	renew := time.NewTicker(leaseDuration / 3)
	defer renew.Stop()
	for {
		select {
		case err := <-done:
			cancel()
			<-done
			return err
		case <-ctx.Done():
			cancel()
			<-done
			<-done
			return nil
		case <-renew.C:
			_, held, err := q.AcquireSchemeBettingShardLease(ctx, "strategy", int32(shard), owner, leaseDuration)
			if err == nil && held {
				continue
			}
			cancel()
			<-done
			<-done
			if err != nil {
				return err
			}
			return errContiguousTargetShardLeaseLost
		}
	}
}
