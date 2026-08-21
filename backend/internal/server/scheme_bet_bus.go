package server

import (
	"context"
	"errors"
	"fmt"
	"time"

	"caipiao/backend/internal/db"
	"caipiao/backend/internal/db/sqlcdb"
	"caipiao/backend/internal/schemeeventbus"
)

var errBetShardLeaseLost = errors.New("scheme bet dispatcher shard lease lost")

type betReadyProcessor interface {
	HandleBetReady(context.Context, schemeeventbus.BetReady) error
}

type betReconcileProcessor interface {
	HandleBetReconcile(context.Context, schemeeventbus.BetReconcile) error
}

func runLeasedSchemeBetReadyConsumer(
	ctx context.Context,
	bus *schemeeventbus.Bus,
	pool *db.Pool,
	shard int32,
	owner string,
	leaseDuration time.Duration,
	processor betReadyProcessor,
) error {
	if pool == nil || owner == "" {
		return errors.New("scheme bet dispatcher lease configuration is incomplete")
	}
	if leaseDuration <= 0 {
		leaseDuration = 5 * time.Second
	}
	q := sqlcdb.New(pool)
	retry := time.NewTicker(leaseDuration / 3)
	defer retry.Stop()
	for {
		_, acquired, err := q.AcquireSchemeBettingShardLease(ctx, "dispatcher", shard, owner, leaseDuration)
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
		err = holdSchemeBetReadyShardLease(ctx, bus, q, shard, owner, leaseDuration, processor)
		if errors.Is(err, errBetShardLeaseLost) {
			continue
		}
		return err
	}
}

func holdSchemeBetReadyShardLease(
	ctx context.Context,
	bus *schemeeventbus.Bus,
	q *sqlcdb.Queries,
	shard int32,
	owner string,
	leaseDuration time.Duration,
	processor betReadyProcessor,
) error {
	leaseCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- runSchemeBetReadyConsumer(leaseCtx, bus, shard, processor) }()
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
			_, held, err := q.AcquireSchemeBettingShardLease(ctx, "dispatcher", shard, owner, leaseDuration)
			if err == nil && held {
				continue
			}
			cancel()
			<-done
			if err != nil {
				return err
			}
			return errBetShardLeaseLost
		}
	}
}

func runSchemeBetReadyConsumer(
	ctx context.Context,
	bus *schemeeventbus.Bus,
	shard int32,
	processor betReadyProcessor,
) error {
	if bus == nil || processor == nil || shard < 0 {
		return errors.New("scheme bet-ready consumer configuration is incomplete")
	}
	durable := fmt.Sprintf("scheme-bet-dispatch-shard-%d-v1", shard)
	err := bus.ConsumeBetReady(ctx, shard, durable, processor.HandleBetReady)
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return nil
	}
	return err
}

func runSchemeBetReconcileConsumer(
	ctx context.Context,
	bus *schemeeventbus.Bus,
	shard int32,
	processor betReconcileProcessor,
) error {
	if bus == nil || processor == nil || shard < 0 {
		return errors.New("scheme bet-reconcile consumer configuration is incomplete")
	}
	durable := fmt.Sprintf("scheme-bet-rearm-shard-%d-v1", shard)
	err := bus.ConsumeBetReconcile(ctx, shard, durable, processor.HandleBetReconcile)
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return nil
	}
	return err
}
