package schemes

import (
	"context"
	"sort"
	"sync"
	"testing"

	"caipiao/backend/internal/db/sqlcdb"
)

// TestFormalShortPeriodContiguousLifecycle exercises the local, database-free
// half of the formal chain contract. Database-backed strategy and Outbox
// assertions remain in the schema-gated resolver tests; this fixture verifies
// that event redelivery/restart cannot turn a completed N -> N+1 wait into a
// second dispatch.
func TestFormalShortPeriodContiguousLifecycle(t *testing.T) {
	f := newContiguousChainE2EFixture(t, "tron_ffc_6s")
	f.addWaiting(41, 7)

	if got := classifyContiguousTargetBoundary("100", "100", "101"); got != contiguousTargetResolved {
		t.Fatalf("boundary 100 -> 101 classification=%v, want resolved", got)
	}
	if got := classifyContiguousTargetBoundary("100", "101", "102"); got != contiguousTargetMissed {
		t.Fatalf("boundary that has passed source classification=%v, want missed", got)
	}

	if processed, err := f.dispatchOne(); err != nil || processed != 1 {
		t.Fatalf("first dispatch processed=%d err=%v, want one completed wait", processed, err)
	}
	if got := f.submissionCount(); got != 1 {
		t.Fatalf("provider submissions=%d, want one", got)
	}
	if got := f.outboxCountForSource("100"); got != 1 {
		t.Fatalf("outboxes for source 100=%d, want one", got)
	}

	// A process restart repeats the bounded recovery scan. Its authoritative
	// source no longer returns the completed decision, so it cannot submit a
	// duplicate provider request or Outbox command.
	if processed, err := f.dispatchOne(); err != nil || processed != 0 {
		t.Fatalf("restart dispatch processed=%d err=%v, want no completed wait", processed, err)
	}
	if got := f.submissionCount(); got != 1 {
		t.Fatalf("provider submissions after restart=%d, want one", got)
	}
}

type contiguousChainE2EFixture struct {
	t        *testing.T
	source   *contiguousChainRecoverySource
	resolver *contiguousChainRecoveryResolver
}

func newContiguousChainE2EFixture(t *testing.T, lotteryCode string) *contiguousChainE2EFixture {
	t.Helper()
	source := &contiguousChainRecoverySource{lotteryCode: lotteryCode, rows: make(map[int64]sqlcdb.AwaitingContiguousTargetRow)}
	resolver := &contiguousChainRecoveryResolver{source: source, submitted: make(map[int64]struct{})}
	return &contiguousChainE2EFixture{t: t, source: source, resolver: resolver}
}

func (f *contiguousChainE2EFixture) addWaiting(decisionID int64, shard int32) {
	f.t.Helper()
	f.source.mu.Lock()
	defer f.source.mu.Unlock()
	f.source.rows[decisionID] = sqlcdb.AwaitingContiguousTargetRow{DecisionID: decisionID, LotteryCode: f.source.lotteryCode, SourcePeriodNo: "100", ShardNo: shard}
}

func (f *contiguousChainE2EFixture) dispatchOne() (int, error) {
	return runContiguousTargetRecoveryBatch(
		WithStrategyLeaseFence(context.Background(), StrategyLeaseFence{ShardNo: 7, Owner: "test-owner", Epoch: 1}),
		f.source, f.resolver, []string{f.source.lotteryCode}, []int32{7}, 32,
	)
}

func (f *contiguousChainE2EFixture) submissionCount() int {
	f.resolver.mu.Lock()
	defer f.resolver.mu.Unlock()
	return len(f.resolver.submitted)
}

func (f *contiguousChainE2EFixture) outboxCountForSource(source string) int {
	f.source.mu.Lock()
	defer f.source.mu.Unlock()
	return f.source.completedBySource[source]
}

type contiguousChainRecoverySource struct {
	mu                sync.Mutex
	lotteryCode       string
	rows              map[int64]sqlcdb.AwaitingContiguousTargetRow
	completedBySource map[string]int
	assertErr         error
}

func (s *contiguousChainRecoverySource) AssertSchemeBettingShardLease(_ context.Context, role string, _ int32, _ string, _ int64) error {
	if role != "strategy" {
		return errUnexpectedRecoveryLeaseRole
	}
	return s.assertErr
}

func (s *contiguousChainRecoverySource) ListAwaitingContiguousTargets(_ context.Context, lotteries []string, shards []int32, _ int64, limit int32) ([]sqlcdb.AwaitingContiguousTargetRow, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(lotteries) != 1 || lotteries[0] != s.lotteryCode || len(shards) != 1 {
		return nil, errUnexpectedRecoveryScope
	}
	rows := make([]sqlcdb.AwaitingContiguousTargetRow, 0, len(s.rows))
	for _, row := range s.rows {
		if row.ShardNo == shards[0] {
			rows = append(rows, row)
		}
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].DecisionID < rows[j].DecisionID })
	if len(rows) > int(limit) {
		rows = rows[:limit]
	}
	return rows, nil
}

func (s *contiguousChainRecoverySource) complete(row sqlcdb.AwaitingContiguousTargetRow) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.completedBySource == nil {
		s.completedBySource = make(map[string]int)
	}
	delete(s.rows, row.DecisionID)
	s.completedBySource[row.SourcePeriodNo]++
}

type contiguousChainRecoveryResolver struct {
	mu         sync.Mutex
	source     *contiguousChainRecoverySource
	submitted  map[int64]struct{}
	resolveErr error
}

func (r *contiguousChainRecoveryResolver) ResolveAwaitingTarget(_ context.Context, decisionID int64) error {
	if r.resolveErr != nil {
		return r.resolveErr
	}
	r.mu.Lock()
	if _, duplicate := r.submitted[decisionID]; duplicate {
		r.mu.Unlock()
		return nil
	}
	r.submitted[decisionID] = struct{}{}
	r.mu.Unlock()

	r.source.mu.Lock()
	row, found := r.source.rows[decisionID]
	r.source.mu.Unlock()
	if found {
		r.source.complete(row)
	}
	return nil
}
