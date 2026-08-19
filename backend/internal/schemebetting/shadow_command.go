package schemebetting

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type ShadowCommandInput struct {
	SchemeID           string
	LotteryCode        string
	SourcePeriod       string
	Target             PeriodSnapshot
	ProviderSnapshotID int64
	StateVersion       int64
	RuleSnapshotHash   string
	LocalHit           bool
	Now                time.Time
	Budget             DeadlineBudget
	ShardCount         uint32
}

type ShadowCommand struct {
	RequestID          string
	PayloadHash        string
	Payload            []byte
	TargetPeriod       string
	ProviderSnapshotID int64
	CloseAt            time.Time
	SafeDeadline       time.Time
	ShardNo            uint32
}

func BuildShadowCommand(input ShadowCommandInput) (ShadowCommand, error) {
	input.SchemeID = strings.TrimSpace(input.SchemeID)
	input.LotteryCode = strings.TrimSpace(input.LotteryCode)
	input.SourcePeriod = strings.TrimSpace(input.SourcePeriod)
	input.Target.PeriodNo = strings.TrimSpace(input.Target.PeriodNo)
	if input.SchemeID == "" || input.LotteryCode == "" || input.SourcePeriod == "" || input.Target.PeriodNo == "" || input.ProviderSnapshotID <= 0 {
		return ShadowCommand{}, fmt.Errorf("incomplete shadow command identity")
	}
	now := input.Now.UTC()
	if now.IsZero() {
		now = time.Now().UTC()
	}
	safeDeadline := SafeDeadline(input.Target.CloseAt, input.Budget)
	if !IsSafeToCreate(now, safeDeadline) {
		return ShadowCommand{}, fmt.Errorf("provider target %s has no safe dispatch window", input.Target.PeriodNo)
	}
	requestID := CommandIdentity(input.SchemeID, input.SourcePeriod, input.Target.PeriodNo, input.StateVersion)
	payload, err := json.Marshal(struct {
		SchemaVersion      int    `json:"schemaVersion"`
		Kind               string `json:"kind"`
		SchemeID           string `json:"schemeId"`
		LotteryCode        string `json:"lotteryCode"`
		SourcePeriod       string `json:"sourcePeriod"`
		TargetPeriod       string `json:"targetPeriod"`
		ProviderSnapshotID int64  `json:"providerSnapshotId"`
		StateVersion       int64  `json:"stateVersion"`
		RuleSnapshotHash   string `json:"ruleSnapshotHash,omitempty"`
		LocalHit           bool   `json:"localHit"`
		CloseAt            string `json:"closeAt"`
		SafeDeadline       string `json:"safeDeadline"`
	}{
		SchemaVersion: 1, Kind: "strategy_transition_shadow", SchemeID: input.SchemeID,
		LotteryCode: input.LotteryCode, SourcePeriod: input.SourcePeriod, TargetPeriod: input.Target.PeriodNo,
		ProviderSnapshotID: input.ProviderSnapshotID, StateVersion: input.StateVersion,
		RuleSnapshotHash: strings.TrimSpace(input.RuleSnapshotHash), LocalHit: input.LocalHit,
		CloseAt: input.Target.CloseAt.UTC().Format(time.RFC3339Nano), SafeDeadline: safeDeadline.Format(time.RFC3339Nano),
	})
	if err != nil {
		return ShadowCommand{}, err
	}
	return ShadowCommand{
		RequestID: requestID, PayloadHash: PayloadHash(payload), Payload: payload,
		TargetPeriod: input.Target.PeriodNo, ProviderSnapshotID: input.ProviderSnapshotID,
		CloseAt: input.Target.CloseAt.UTC(), SafeDeadline: safeDeadline,
		ShardNo: ShardForScheme(input.SchemeID, input.ShardCount),
	}, nil
}
