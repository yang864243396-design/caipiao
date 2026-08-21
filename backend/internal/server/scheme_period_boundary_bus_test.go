package server

import (
	"context"
	"testing"

	"caipiao/backend/internal/db/sqlcdb"
	"caipiao/backend/internal/schemeeventbus"
	"caipiao/backend/internal/schemes"
)

func TestPeriodBoundaryExpanderPublishesOneBoundedPageWithStableDedup(t *testing.T) {
	rows := make([]sqlcdb.AwaitingContiguousTargetRow, 33)
	for index := range rows {
		rows[index] = sqlcdb.AwaitingContiguousTargetRow{
			DecisionID: int64(index + 1), SchemeID: "scheme-" + string(rune('a'+index)),
			LotteryCode: "tron_ffc_6s", SourcePeriodNo: "100",
		}
	}
	source := &recordingBoundaryAwaitingSource{rows: rows}
	publisher := &recordingContiguousTargetPublisher{}
	event := schemeeventbus.PeriodBoundary{LotteryCode: "tron_ffc_6s", CurrentIssue: "100", NextIssue: "101", Generation: 9}

	if err := expandSchemePeriodBoundary(context.Background(), event, source, publisher, []int32{0, 1}, 64); err != nil {
		t.Fatalf("expandSchemePeriodBoundary() error = %v", err)
	}
	if source.limit != 32 || len(publisher.events) != 32 {
		t.Fatalf("listLimit=%d published=%d, want one bounded page", source.limit, len(publisher.events))
	}
	for index, ready := range publisher.events {
		if ready.DecisionID != int64(index+1) || ready.BoundaryGeneration != 9 || ready.MessageID() != "contiguous-target:"+itoa(index+1)+":9" {
			t.Fatalf("ready[%d]=%+v", index, ready)
		}
	}
}

func TestLeasedContiguousTargetWorkerCarriesStrategyFence(t *testing.T) {
	capture := &contiguousTargetWorkerCapture{}
	worker := leasedContiguousTargetWorker{
		worker: capture,
		fence:  schemes.StrategyLeaseFence{ShardNo: 3, Owner: "node-a", Epoch: 7},
	}
	event := schemeeventbus.ContiguousTargetReady{DecisionID: 41}
	if err := worker.ProcessContiguousTargetReady(context.Background(), event); err != nil {
		t.Fatalf("ProcessContiguousTargetReady() error = %v", err)
	}
	if capture.decisionID != 41 {
		t.Fatalf("capture decision=%d, want 41", capture.decisionID)
	}
}

type recordingBoundaryAwaitingSource struct {
	rows  []sqlcdb.AwaitingContiguousTargetRow
	limit int32
}

func (source *recordingBoundaryAwaitingSource) ListAwaitingContiguousTargets(
	_ context.Context, _ []string, _ []int32, _ int64, limit int32,
) ([]sqlcdb.AwaitingContiguousTargetRow, error) {
	source.limit = limit
	if int(limit) < len(source.rows) {
		return source.rows[:limit], nil
	}
	return source.rows, nil
}

type recordingContiguousTargetPublisher struct {
	events []schemeeventbus.ContiguousTargetReady
}

func (publisher *recordingContiguousTargetPublisher) PublishContiguousTargetReady(
	_ context.Context, event schemeeventbus.ContiguousTargetReady, _ uint32,
) error {
	publisher.events = append(publisher.events, event)
	return nil
}

type contiguousTargetWorkerCapture struct {
	decisionID int64
}

func (capture *contiguousTargetWorkerCapture) ProcessContiguousTargetReady(ctx context.Context, event schemeeventbus.ContiguousTargetReady) error {
	capture.decisionID = event.DecisionID
	return nil
}

func itoa(value int) string {
	if value == 0 {
		return "0"
	}
	bytes := make([]byte, 0, 8)
	for value > 0 {
		bytes = append([]byte{byte('0' + value%10)}, bytes...)
		value /= 10
	}
	return string(bytes)
}
