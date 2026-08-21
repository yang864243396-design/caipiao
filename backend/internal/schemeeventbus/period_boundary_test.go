package schemeeventbus

import (
	"testing"

	"caipiao/backend/internal/schemebetting"
)

func TestPeriodBoundaryMessageIDIncludesLotteryAndGeneration(t *testing.T) {
	event := PeriodBoundary{LotteryCode: "tron_ffc_6s", CurrentIssue: "100", NextIssue: "101", Generation: 7}
	if got := event.MessageID(); got != "period-boundary:tron_ffc_6s:100:101:7" {
		t.Fatalf("message ID = %q", got)
	}
}

func TestContiguousTargetReadyRoutesBySchemeShard(t *testing.T) {
	event := ContiguousTargetReady{
		DecisionID: 9, SchemeID: "inst-9", LotteryCode: "tron_ffc_6s", SourcePeriod: "100", BoundaryGeneration: 7,
	}
	if got, want := event.Shard(64), schemebetting.ShardForScheme("inst-9", 64); got != want {
		t.Fatalf("shard = %d, want %d", got, want)
	}
	if got := event.MessageID(); got != "contiguous-target:9:7" {
		t.Fatalf("message ID = %q", got)
	}
}
