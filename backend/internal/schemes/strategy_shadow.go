package schemes

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"caipiao/backend/internal/db/sqlcdb"
	"caipiao/backend/internal/lottery"
	"caipiao/backend/internal/providerperiodtarget"
	"caipiao/backend/internal/schemebetting"
)

const shadowOutboxShardCount = 64

type shadowDecisionResult struct {
	Status       string `json:"status"`
	Reason       string `json:"reason,omitempty"`
	DecisionID   int64  `json:"decisionId,omitempty"`
	TargetPeriod string `json:"targetPeriod,omitempty"`
	RequestID    string `json:"requestId,omitempty"`
	SafeDeadline string `json:"safeDeadline,omitempty"`
}

func persistShadowDecision(ctx context.Context, q *sqlcdb.Queries, row sqlcdb.PendingFormalStrategyRow, stateVersionBefore int64, hit bool, winningUnits int) (shadowDecisionResult, error) {
	now := time.Now().UTC()
	result := shadowDecisionResult{Status: "blocked"}
	target, providerSnapshotID, ok, err := providerperiodtarget.Current(ctx, q, row.LotteryCode, row.PeriodNo, now)
	if err != nil {
		return result, err
	}
	var command schemebetting.ShadowCommand
	if !ok {
		result.Reason = "no_fresh_provider_target"
	} else {
		command, err = schemebetting.BuildShadowCommand(schemebetting.ShadowCommandInput{
			SchemeID: row.SchemeID, LotteryCode: row.LotteryCode, SourcePeriod: row.PeriodNo,
			Target: target, ProviderSnapshotID: providerSnapshotID, StateVersion: stateVersionBefore + 1,
			RuleSnapshotHash: row.RuleSnapshotHash.String, LocalHit: hit, Now: now,
			Budget: shadowDeadlineBudget(target), ShardCount: shadowOutboxShardCount,
		})
		if err != nil {
			result.Reason = "unsafe_deadline"
		} else {
			result.Status = "completed"
			result.TargetPeriod = command.TargetPeriod
			result.RequestID = command.RequestID
			result.SafeDeadline = command.SafeDeadline.Format(time.RFC3339Nano)
		}
	}
	diagnostics, err := json.Marshal(map[string]any{
		"mode": "shadow", "reason": result.Reason, "targetPeriod": result.TargetPeriod,
		"requestId": result.RequestID, "safeDeadline": result.SafeDeadline,
	})
	if err != nil {
		return result, err
	}
	decisionID, created, err := q.InsertSchemePeriodDecision(ctx, sqlcdb.InsertSchemePeriodDecisionParams{
		SchemeID: row.SchemeID, LotteryCode: row.LotteryCode, SourcePeriodNo: row.PeriodNo,
		SourceBetRecordID: row.RecordID, DrawHash: lottery.CanonicalDrawHash(row.LotteryCode, row.PeriodNo, row.Balls),
		StateVersionBefore: stateVersionBefore, StateVersionAfter: stateVersionBefore + 1,
		RuleVersion: row.RuleVersion, RuleSnapshotHash: row.RuleSnapshotHash,
		LocalHit: hit, WinningUnits: winningUnits, Status: result.Status, Diagnostics: diagnostics,
	})
	if err != nil {
		return result, err
	}
	result.DecisionID = decisionID
	if !created || result.Status != "completed" {
		return result, nil
	}
	if command.ProviderSnapshotID <= 0 {
		return result, fmt.Errorf("provider snapshot id missing for target %s", command.TargetPeriod)
	}
	err = q.InsertShadowSchemeBetOutbox(ctx, sqlcdb.InsertShadowSchemeBetOutboxParams{
		DecisionID: decisionID, SchemeID: row.SchemeID, LotteryCode: row.LotteryCode,
		SourcePeriodNo: row.PeriodNo, TargetPeriodNo: command.TargetPeriod,
		RequestID: command.RequestID, PayloadHash: command.PayloadHash, Payload: command.Payload,
		ProviderSnapshotID: command.ProviderSnapshotID, CloseAt: command.CloseAt,
		SafeDeadlineAt: command.SafeDeadline, ShardNo: int32(command.ShardNo),
	})
	return result, err
}

func shadowDeadlineBudget(target schemebetting.PeriodSnapshot) schemebetting.DeadlineBudget {
	duration := target.CloseAt.Sub(target.OpenAt)
	switch {
	case duration > 0 && duration <= 6*time.Second:
		return schemebetting.DeadlineBudget{ClockSkew: 150 * time.Millisecond, Queue: 100 * time.Millisecond, Dispatch: 150 * time.Millisecond, Network: 900 * time.Millisecond}
	case duration > 0 && duration <= 15*time.Second:
		return schemebetting.DeadlineBudget{ClockSkew: 200 * time.Millisecond, Queue: 200 * time.Millisecond, Dispatch: 200 * time.Millisecond, Network: 1100 * time.Millisecond}
	default:
		return schemebetting.DeadlineBudget{ClockSkew: 250 * time.Millisecond, Queue: 500 * time.Millisecond, Dispatch: 300 * time.Millisecond, Network: 1500 * time.Millisecond}
	}
}
