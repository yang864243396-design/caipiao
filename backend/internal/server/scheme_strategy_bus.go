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

const strategyReadyExpansionBatch = 1000

var errStrategyShardLeaseLost = errors.New("scheme strategy shard lease lost")

type leasedStrategyWorker struct {
	worker interface {
		ProcessStrategyReady(context.Context, int64, string, string, string, int64) error
	}
	fence schemes.StrategyLeaseFence
}

func (worker leasedStrategyWorker) ProcessStrategyReady(
	ctx context.Context, recordID int64, schemeID, lotteryCode, periodNo string, expectedStateVersion int64,
) error {
	return worker.worker.ProcessStrategyReady(
		schemes.WithStrategyLeaseFence(ctx, worker.fence),
		recordID, schemeID, lotteryCode, periodNo, expectedStateVersion,
	)
}

func runSchemeDrawExpander(ctx context.Context, bus *schemeeventbus.Bus, pool *db.Pool, shardCount uint32) error {
	if bus == nil || pool == nil || shardCount == 0 {
		return errors.New("scheme draw expander configuration is incomplete")
	}
	q := sqlcdb.New(pool)
	return bus.ConsumeDraws(ctx, "scheme-strategy-expander-v2", func(messageContext context.Context, event schemeeventbus.DrawConfirmed) error {
		var cursor int64
		for {
			candidates, err := q.ListStrategyReadyCandidates(
				messageContext, event.LotteryCode, event.PeriodNo, cursor, strategyReadyExpansionBatch,
			)
			if err != nil {
				return err
			}
			if len(candidates) == 0 {
				return nil
			}
			for _, candidate := range candidates {
				ready := schemeeventbus.StrategyReady{
					RecordID: candidate.RecordID, SchemeID: candidate.SchemeID,
					LotteryCode: event.LotteryCode, SourcePeriod: event.PeriodNo,
					StateVersion: candidate.StateVersion,
				}
				if err := bus.PublishStrategyReady(messageContext, ready, shardCount); err != nil {
					return err
				}
				cursor = candidate.RecordID
			}
			if len(candidates) < strategyReadyExpansionBatch {
				return nil
			}
		}
	})
}

func runSchemeStrategyConsumer(ctx context.Context, bus *schemeeventbus.Bus, shard uint32, worker interface {
	ProcessStrategyReady(context.Context, int64, string, string, string, int64) error
}) error {
	if bus == nil || worker == nil {
		return nil
	}
	durable := fmt.Sprintf("scheme-strategy-shard-%d", shard)
	err := bus.ConsumeStrategyReady(ctx, shard, durable, func(messageContext context.Context, event schemeeventbus.StrategyReady) error {
		return worker.ProcessStrategyReady(
			messageContext, event.RecordID, event.SchemeID, event.LotteryCode, event.SourcePeriod, event.StateVersion,
		)
	})
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return nil
	}
	return err
}

func runLeasedSchemeStrategyConsumer(
	ctx context.Context,
	bus *schemeeventbus.Bus,
	pool *db.Pool,
	shard uint32,
	owner string,
	leaseDuration time.Duration,
	worker interface {
		ProcessStrategyReady(context.Context, int64, string, string, string, int64) error
	},
) error {
	if pool == nil || owner == "" {
		return errors.New("scheme strategy lease configuration is incomplete")
	}
	if leaseDuration <= 0 {
		leaseDuration = 5 * time.Second
	}
	q := sqlcdb.New(pool)
	retry := time.NewTicker(leaseDuration / 3)
	defer retry.Stop()
	for {
		epoch, acquired, err := q.AcquireSchemeBettingShardLease(
			ctx, "strategy", int32(shard), owner, leaseDuration,
		)
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
		err = holdSchemeStrategyShardLease(ctx, bus, q, shard, owner, epoch, leaseDuration, worker)
		if errors.Is(err, errStrategyShardLeaseLost) {
			continue
		}
		return err
	}
}

func holdSchemeStrategyShardLease(
	ctx context.Context,
	bus *schemeeventbus.Bus,
	q *sqlcdb.Queries,
	shard uint32,
	owner string,
	epoch int64,
	leaseDuration time.Duration,
	worker interface {
		ProcessStrategyReady(context.Context, int64, string, string, string, int64) error
	},
) error {
	leaseCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	done := make(chan error, 1)
	fencedWorker := leasedStrategyWorker{
		worker: worker,
		fence:  schemes.StrategyLeaseFence{ShardNo: int32(shard), Owner: owner, Epoch: epoch},
	}
	go func() {
		done <- runSchemeStrategyConsumer(leaseCtx, bus, shard, fencedWorker)
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
			_, held, err := q.AcquireSchemeBettingShardLease(
				ctx, "strategy", int32(shard), owner, leaseDuration,
			)
			if err == nil && held {
				continue
			}
			cancel()
			<-done
			if err != nil {
				return err
			}
			return errStrategyShardLeaseLost
		}
	}
}
