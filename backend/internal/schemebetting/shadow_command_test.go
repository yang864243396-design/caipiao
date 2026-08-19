package schemebetting

import (
	"testing"
	"time"
)

func TestBuildShadowCommandRejectsUnsafeWindowAndFreezesProviderTarget(t *testing.T) {
	now := time.Date(2026, 8, 19, 12, 0, 1, 0, time.UTC)
	input := ShadowCommandInput{
		SchemeID: "scheme-1", LotteryCode: "lottery-X", SourcePeriod: "period-A",
		Target:             PeriodSnapshot{PeriodNo: "provider-period-Z", OpenAt: now.Add(-time.Second), CloseAt: now.Add(3 * time.Second), ObservedAt: now},
		ProviderSnapshotID: 88, StateVersion: 5, RuleSnapshotHash: "rule-hash", LocalHit: true,
		Now: now, Budget: DeadlineBudget{Network: time.Second}, ShardCount: 64,
	}
	command, err := BuildShadowCommand(input)
	if err != nil {
		t.Fatal(err)
	}
	if command.TargetPeriod != "provider-period-Z" || command.ProviderSnapshotID != 88 || command.RequestID == "" || command.PayloadHash == "" {
		t.Fatalf("command=%+v", command)
	}
	input.Now = command.SafeDeadline
	if _, err := BuildShadowCommand(input); err == nil {
		t.Fatal("unsafe window must not create an outbox command")
	}
}
