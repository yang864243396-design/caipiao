package server

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"caipiao/backend/internal/db"
	"caipiao/backend/internal/db/sqlcdb"
	"caipiao/backend/internal/schemeeventbus"
	"caipiao/backend/internal/schemes"
)

const contiguousTargetBoundaryExpansionPageSize int32 = 32

type contiguousRecoveryReadiness struct {
	drawWSReady   chan struct{}
	expanderReady chan struct{}

	drawOnce     sync.Once
	expanderOnce sync.Once
}

func newContiguousRecoveryReadiness(_ []int32) *contiguousRecoveryReadiness {
	return &contiguousRecoveryReadiness{
		drawWSReady: make(chan struct{}), expanderReady: make(chan struct{}),
	}
}

func (readiness *contiguousRecoveryReadiness) SignalDrawWS() {
	if readiness != nil {
		readiness.drawOnce.Do(func() { close(readiness.drawWSReady) })
	}
}

func (readiness *contiguousRecoveryReadiness) SignalExpander() {
	if readiness != nil {
		readiness.expanderOnce.Do(func() { close(readiness.expanderReady) })
	}
}

func runContiguousTargetRecoveryWhenReady(
	ctx context.Context,
	readiness *contiguousRecoveryReadiness,
	localSubscriptionReady <-chan struct{},
	run func(context.Context) error,
) error {
	if readiness == nil || localSubscriptionReady == nil || run == nil {
		return errors.New("contiguous target recovery readiness configuration is incomplete")
	}
	for _, ready := range []<-chan struct{}{
		readiness.drawWSReady, readiness.expanderReady, localSubscriptionReady,
	} {
		select {
		case <-ctx.Done():
			return nil
		case <-ready:
		}
	}
	return run(ctx)
}

func runContiguousTargetShardRuntime(
	ctx context.Context,
	readiness *contiguousRecoveryReadiness,
	shard int32,
	assertLease func(context.Context) error,
	consume func(context.Context, func()) error,
	recoverTargets func(context.Context) error,
) error {
	if readiness == nil || assertLease == nil || consume == nil || recoverTargets == nil {
		return errors.New("contiguous target shard runtime configuration is incomplete")
	}
	if err := assertLease(ctx); err != nil {
		return err
	}
	_ = shard

	runtimeCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	subscriptionReady := make(chan struct{})
	var subscriptionOnce sync.Once
	signalSubscription := func() {
		subscriptionOnce.Do(func() {
			close(subscriptionReady)
		})
	}
	done := make(chan error, 2)
	go func() { done <- consume(runtimeCtx, signalSubscription) }()
	go func() {
		done <- runContiguousTargetRecoveryWhenReady(runtimeCtx, readiness, subscriptionReady, recoverTargets)
	}()
	err := <-done
	cancel()
	<-done
	if ctx.Err() != nil {
		return nil
	}
	return err
}

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
	ready func(),
) error {
	if bus == nil || pool == nil {
		return errors.New("contiguous target boundary expander configuration is incomplete")
	}
	q := sqlcdb.New(pool)
	return bus.ConsumePeriodBoundariesReady(ctx, schemeeventbus.ContiguousTargetExpanderDurable, func(messageContext context.Context, event schemeeventbus.PeriodBoundary) error {
		return expandSchemePeriodBoundary(messageContext, event, q, bus, shards, shardCount)
	}, ready)
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
}, ready func()) error {
	if bus == nil || worker == nil {
		return nil
	}
	durable := fmt.Sprintf("scheme-contiguous-target-shard-%d", shard)
	err := bus.ConsumeContiguousTargetReadyWithReady(ctx, shard, durable, func(messageContext context.Context, event schemeeventbus.ContiguousTargetReady) error {
		return worker.ProcessContiguousTargetReady(messageContext, event)
	}, ready)
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
	readiness *contiguousRecoveryReadiness,
) error {
	if pool == nil || owner == "" || readiness == nil {
		return errors.New("contiguous target strategy lease configuration is incomplete")
	}
	if leaseDuration <= 0 {
		leaseDuration = 5 * time.Second
	}
	q := sqlcdb.New(pool)
	retry := time.NewTicker(leaseDuration / 3)
	defer retry.Stop()
	return runContiguousTargetShardOwnershipLoop(
		ctx, retry.C,
		func(acquireCtx context.Context) (int64, bool, error) {
			return q.AcquireSchemeBettingShardLease(acquireCtx, "strategy", int32(shard), owner, leaseDuration)
		},
		func(leaseCtx context.Context, epoch int64) error {
			return holdSchemeContiguousTargetShardLease(
				leaseCtx, bus, q, shard, owner, epoch, leaseDuration, worker,
				lotteryCodes, batch, concurrency, recoveryInterval, readiness,
			)
		},
	)
}

func runContiguousTargetShardOwnershipLoop(
	ctx context.Context,
	retry <-chan time.Time,
	acquire func(context.Context) (int64, bool, error),
	hold func(context.Context, int64) error,
) error {
	if retry == nil || acquire == nil || hold == nil {
		return errors.New("contiguous target shard ownership configuration is incomplete")
	}
	for {
		epoch, acquired, err := acquire(ctx)
		if err != nil {
			return err
		}
		if !acquired {
			select {
			case <-ctx.Done():
				return nil
			case <-retry:
				continue
			}
		}
		err = hold(ctx, epoch)
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
	readiness *contiguousRecoveryReadiness,
) error {
	leaseCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	done := make(chan error, 1)
	fencedWorker := leasedContiguousTargetWorker{worker: worker, fence: schemes.StrategyLeaseFence{ShardNo: int32(shard), Owner: owner, Epoch: epoch}}
	go func() {
		done <- runContiguousTargetShardRuntime(
			leaseCtx, readiness, int32(shard),
			func(assertCtx context.Context) error {
				return q.AssertSchemeBettingShardLease(assertCtx, "strategy", int32(shard), owner, epoch)
			},
			func(consumeCtx context.Context, ready func()) error {
				return runSchemeContiguousTargetConsumer(consumeCtx, bus, shard, fencedWorker, ready)
			},
			func(recoveryCtx context.Context) error {
				return fencedWorker.RunContiguousTargetRecovery(
					recoveryCtx, lotteryCodes, []int32{int32(shard)}, batch, concurrency, recoveryInterval,
				)
			},
		)
	}()
	renew := time.NewTicker(leaseDuration / 3)
	defer renew.Stop()
	for {
		select {
		case err := <-done:
			return err
		case <-ctx.Done():
			cancel()
			<-done
			return nil
		case <-renew.C:
			_, held, err := q.AcquireSchemeBettingShardLease(ctx, "strategy", int32(shard), owner, leaseDuration)
			if err == nil && held {
				continue
			}
			cancel()
			<-done
			if err != nil {
				return err
			}
			return errContiguousTargetShardLeaseLost
		}
	}
}
