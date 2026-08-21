package schemes

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"caipiao/backend/internal/db/sqlcdb"
)

func TestAwaitingTargetRecoveryUsesOneBoundedPageAndSameResolver(t *testing.T) {
	rows := make([]sqlcdb.AwaitingContiguousTargetRow, 40)
	for index := range rows {
		rows[index] = sqlcdb.AwaitingContiguousTargetRow{DecisionID: int64(index + 1), ShardNo: 3}
	}
	source := &recordingAwaitingTargetSource{rows: rows}
	resolver := &recordingAwaitingTargetResolver{}

	processed, err := runContiguousTargetRecoveryBatch(
		recoveryFenceContext(3), source, resolver,
		[]string{"tron_ffc_6s"}, []int32{3}, 500,
	)
	if err != nil {
		t.Fatalf("runContiguousTargetRecoveryBatch() error = %v", err)
	}
	if processed != 32 || source.limit != 32 || len(resolver.decisionIDs) != 32 {
		t.Fatalf("processed=%d listLimit=%d resolved=%d, want exactly one bounded page", processed, source.limit, len(resolver.decisionIDs))
	}
	for index, decisionID := range resolver.decisionIDs {
		if decisionID != int64(index+1) {
			t.Fatalf("resolver decision[%d]=%d, want %d", index, decisionID, index+1)
		}
	}
}

func TestAwaitingTargetRecoveryDoesNotFanOutResolverCalls(t *testing.T) {
	source := &recordingAwaitingTargetSource{rows: []sqlcdb.AwaitingContiguousTargetRow{{DecisionID: 1, ShardNo: 3}, {DecisionID: 2, ShardNo: 3}}}
	resolver := &recordingAwaitingTargetResolver{delay: 5 * time.Millisecond}

	if _, err := runContiguousTargetRecoveryBatch(recoveryFenceContext(3), source, resolver, []string{"tron_ffc_6s"}, []int32{3}, 32); err != nil {
		t.Fatalf("runContiguousTargetRecoveryBatch() error = %v", err)
	}
	if resolver.maxActive != 1 {
		t.Fatalf("concurrent resolver calls=%d, want 1", resolver.maxActive)
	}
}

func TestAwaitingTargetRecoveryRejectsUnfencedScan(t *testing.T) {
	source := &recordingAwaitingTargetSource{rows: []sqlcdb.AwaitingContiguousTargetRow{{DecisionID: 1}}}
	resolver := &recordingAwaitingTargetResolver{}

	_, err := runContiguousTargetRecoveryBatch(context.Background(), source, resolver, []string{"tron_ffc_6s"}, []int32{3}, 32)
	if err == nil {
		t.Fatal("unfenced recovery unexpectedly succeeded")
	}
	if source.listCalls != 0 || len(resolver.decisionIDs) != 0 {
		t.Fatalf("unfenced recovery listed=%d resolved=%d, want no database processing", source.listCalls, len(resolver.decisionIDs))
	}
}

func TestAwaitingTargetRecoveryScopesListAndResolverToLeaseShard(t *testing.T) {
	source := &recordingAwaitingTargetSource{rows: []sqlcdb.AwaitingContiguousTargetRow{{DecisionID: 1, ShardNo: 3}}}
	resolver := &recordingAwaitingTargetResolver{}

	_, err := runContiguousTargetRecoveryBatch(
		recoveryFenceContext(3), source, resolver,
		[]string{"tron_ffc_6s"}, []int32{2, 3}, 32,
	)
	if err != nil {
		t.Fatal(err)
	}
	if !source.asserted || source.assertOwner != "recovery-owner" || source.assertEpoch != 9 || source.assertShard != 3 {
		t.Fatalf("lease assertion=%v owner=%q epoch=%d shard=%d", source.asserted, source.assertOwner, source.assertEpoch, source.assertShard)
	}
	if !reflect.DeepEqual(source.shards, []int32{3}) {
		t.Fatalf("listed shards=%v want [3]", source.shards)
	}
	if !reflect.DeepEqual(resolver.fenceShards, []int32{3}) {
		t.Fatalf("resolver fence shards=%v want [3]", resolver.fenceShards)
	}
}

func TestAwaitingTargetRecoveryReturnsLeaseLossBeforeListing(t *testing.T) {
	wantErr := errors.New("strategy lease lost")
	source := &recordingAwaitingTargetSource{assertErr: wantErr}
	resolver := &recordingAwaitingTargetResolver{}

	_, err := runContiguousTargetRecoveryBatch(recoveryFenceContext(3), source, resolver, []string{"tron_ffc_6s"}, []int32{3}, 32)
	if !errors.Is(err, wantErr) {
		t.Fatalf("error=%v want lease loss", err)
	}
	if source.listCalls != 0 || len(resolver.decisionIDs) != 0 {
		t.Fatalf("lease-loss recovery listed=%d resolved=%d", source.listCalls, len(resolver.decisionIDs))
	}
}

func TestAwaitingTargetRecoveryLoopStopsOnLeaseLoss(t *testing.T) {
	wantErr := errors.New("strategy lease lost")
	source := &recordingAwaitingTargetSource{assertErr: wantErr}
	resolver := &recordingAwaitingTargetResolver{}

	err := runContiguousTargetRecoveryLoop(
		recoveryFenceContext(3), source, resolver,
		[]string{"tron_ffc_6s"}, []int32{3}, 32, time.Hour,
	)
	if !errors.Is(err, wantErr) {
		t.Fatalf("error=%v want lease loss", err)
	}
}

func recoveryFenceContext(shard int32) context.Context {
	return WithStrategyLeaseFence(context.Background(), StrategyLeaseFence{ShardNo: shard, Owner: "recovery-owner", Epoch: 9})
}

type recordingAwaitingTargetSource struct {
	rows        []sqlcdb.AwaitingContiguousTargetRow
	limit       int32
	shards      []int32
	listCalls   int
	asserted    bool
	assertShard int32
	assertOwner string
	assertEpoch int64
	assertErr   error
}

func (source *recordingAwaitingTargetSource) AssertSchemeBettingShardLease(
	_ context.Context, role string, shard int32, owner string, epoch int64,
) error {
	source.asserted = true
	source.assertShard = shard
	source.assertOwner = owner
	source.assertEpoch = epoch
	if role != "strategy" {
		return errors.New("unexpected lease role")
	}
	return source.assertErr
}

func (source *recordingAwaitingTargetSource) ListAwaitingContiguousTargets(
	_ context.Context, _ []string, shards []int32, _ int64, limit int32,
) ([]sqlcdb.AwaitingContiguousTargetRow, error) {
	source.listCalls++
	source.limit = limit
	source.shards = append([]int32(nil), shards...)
	if int(limit) < len(source.rows) {
		return source.rows[:limit], nil
	}
	return source.rows, nil
}

type recordingAwaitingTargetResolver struct {
	decisionIDs []int64
	fenceShards []int32
	delay       time.Duration
	active      int
	maxActive   int
}

func (resolver *recordingAwaitingTargetResolver) ResolveAwaitingTarget(ctx context.Context, decisionID int64) error {
	fence, ok := strategyLeaseFenceFromContext(ctx)
	if !ok {
		return errors.New("resolver context has no strategy lease fence")
	}
	resolver.fenceShards = append(resolver.fenceShards, fence.ShardNo)
	resolver.active++
	if resolver.active > resolver.maxActive {
		resolver.maxActive = resolver.active
	}
	time.Sleep(resolver.delay)
	resolver.decisionIDs = append(resolver.decisionIDs, decisionID)
	resolver.active--
	return nil
}
