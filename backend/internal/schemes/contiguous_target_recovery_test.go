package schemes

import (
	"context"
	"testing"
	"time"

	"caipiao/backend/internal/db/sqlcdb"
)

func TestAwaitingTargetRecoveryUsesOneBoundedPageAndSameResolver(t *testing.T) {
	rows := make([]sqlcdb.AwaitingContiguousTargetRow, 40)
	for index := range rows {
		rows[index] = sqlcdb.AwaitingContiguousTargetRow{DecisionID: int64(index + 1)}
	}
	source := &recordingAwaitingTargetSource{rows: rows}
	resolver := &recordingAwaitingTargetResolver{}

	processed, err := runContiguousTargetRecoveryBatch(
		context.Background(), source, resolver,
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
	source := &recordingAwaitingTargetSource{rows: []sqlcdb.AwaitingContiguousTargetRow{{DecisionID: 1}, {DecisionID: 2}}}
	resolver := &recordingAwaitingTargetResolver{delay: 5 * time.Millisecond}

	if _, err := runContiguousTargetRecoveryBatch(context.Background(), source, resolver, []string{"tron_ffc_6s"}, []int32{3}, 32); err != nil {
		t.Fatalf("runContiguousTargetRecoveryBatch() error = %v", err)
	}
	if resolver.maxActive != 1 {
		t.Fatalf("concurrent resolver calls=%d, want 1", resolver.maxActive)
	}
}

type recordingAwaitingTargetSource struct {
	rows  []sqlcdb.AwaitingContiguousTargetRow
	limit int32
}

func (source *recordingAwaitingTargetSource) ListAwaitingContiguousTargets(
	_ context.Context, _ []string, _ []int32, _ int64, limit int32,
) ([]sqlcdb.AwaitingContiguousTargetRow, error) {
	source.limit = limit
	if int(limit) < len(source.rows) {
		return source.rows[:limit], nil
	}
	return source.rows, nil
}

type recordingAwaitingTargetResolver struct {
	decisionIDs []int64
	delay       time.Duration
	active      int
	maxActive   int
}

func (resolver *recordingAwaitingTargetResolver) ResolveAwaitingTarget(_ context.Context, decisionID int64) error {
	resolver.active++
	if resolver.active > resolver.maxActive {
		resolver.maxActive = resolver.active
	}
	time.Sleep(resolver.delay)
	resolver.decisionIDs = append(resolver.decisionIDs, decisionID)
	resolver.active--
	return nil
}
