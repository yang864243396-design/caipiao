package accountsvc

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgtype"

	"caipiao/backend/internal/cloud/schemestate"
	"caipiao/backend/internal/db/sqlcdb"
	"caipiao/backend/internal/playrules"
)

type formalStrategyInput struct {
	ProviderHit      bool
	Kind             string
	DefinitionConfig []byte
	RoundIndex       int
	LotteryCode      string
	BetContent       string
	Balls            []string
	Snapshot         []byte
}

// formalStrategyVerdict is intentionally separate from real settlement: Hit
// controls the next strategy round only; provider status/pnl/payout stay as
// received from the third party.
type formalStrategyVerdict struct {
	ProviderHit  bool
	Hit          bool
	UsedLocal    bool
	Mismatch     bool
	WinningUnits int
	Reason       string
}

func reconciledStrategyStatus(existing string, localKnown, localHit, providerHit bool) (string, bool) {
	existing = strings.TrimSpace(existing)
	if !localKnown || (existing != "completed" && existing != "mismatch") {
		return existing, false
	}
	if localHit != providerHit {
		return "mismatch", true
	}
	return "completed", true
}

func resolveFormalStrategyVerdict(input formalStrategyInput) formalStrategyVerdict {
	provider := formalStrategyVerdict{ProviderHit: input.ProviderHit, Hit: input.ProviderHit, Reason: "provider_result"}
	if len(input.Snapshot) == 0 || string(input.Snapshot) == "null" {
		return provider
	}
	if len(input.Balls) == 0 {
		provider.Reason = "draw_unavailable"
		return provider
	}
	var snapshot playrules.Snapshot
	if err := json.Unmarshal(input.Snapshot, &snapshot); err != nil {
		provider.Reason = "invalid_rule_snapshot"
		return provider
	}
	if strings.TrimSpace(snapshot.EvaluatorKey) == "" {
		provider.Reason = "empty_rule_snapshot"
		return provider
	}
	result, available, err := schemestate.EvaluateFormalRule(schemestate.FormalRuleEvaluationInput{
		Kind:             input.Kind,
		DefinitionConfig: input.DefinitionConfig,
		RoundIndex:       input.RoundIndex,
		LotteryCode:      input.LotteryCode,
		BetContent:       input.BetContent,
		Balls:            input.Balls,
		Snapshot:         snapshot,
	})
	if !available {
		provider.Reason = "local_evaluator_unavailable"
		return provider
	}
	if err != nil {
		provider.Reason = "local_evaluation_failed"
		return provider
	}
	return formalStrategyVerdict{
		ProviderHit:  input.ProviderHit,
		Hit:          result.Hit,
		UsedLocal:    true,
		Mismatch:     result.Hit != input.ProviderHit,
		WinningUnits: result.WinningUnits,
		Reason:       "local_rule",
	}
}

func persistFormalStrategyVerdict(
	ctx context.Context,
	q *sqlcdb.Queries,
	record sqlcdb.FormalCloudBetStrategyInput,
	orderNo string,
	verdict formalStrategyVerdict,
) error {
	if q == nil || len(record.RuleSnapshot) == 0 || string(record.RuleSnapshot) == "null" {
		return nil
	}
	claimed, err := q.TryClaimSchemeStrategyEvaluation(ctx, sqlcdb.TryClaimSchemeStrategyEvaluationParams{
		InstanceID: record.SchemeID, LotteryCode: record.LotteryCode, PeriodNo: record.PeriodNo,
	})
	if err != nil {
		return err
	}
	if !claimed {
		evaluation, err := q.GetSchemeStrategyEvaluation(ctx, sqlcdb.GetSchemeStrategyEvaluationParams{
			InstanceID: record.SchemeID, PeriodNo: record.PeriodNo,
		})
		if err != nil {
			return err
		}
		status, reconcile := reconciledStrategyStatus(evaluation.Status, evaluation.LocalHit.Valid, evaluation.LocalHit.Bool, verdict.ProviderHit)
		if !reconcile {
			return nil
		}
		diagnostics, err := json.Marshal(map[string]any{
			"source":       "provider_reconciliation",
			"providerHit":  verdict.ProviderHit,
			"localUsed":    true,
			"localHit":     evaluation.LocalHit.Bool,
			"mismatch":     status == "mismatch",
			"winningUnits": evaluation.WinningUnits.Int32,
		})
		if err != nil {
			return err
		}
		_, err = q.ReconcileSchemeStrategyEvaluation(ctx, evaluation.ID, status, diagnostics)
		return err
	}
	evaluation, err := q.GetSchemeStrategyEvaluation(ctx, sqlcdb.GetSchemeStrategyEvaluationParams{
		InstanceID: record.SchemeID, PeriodNo: record.PeriodNo,
	})
	if err != nil {
		return err
	}
	if n, err := q.MarkSchemeStrategyEvaluationProcessing(ctx, evaluation.ID); err != nil {
		return err
	} else if n != 1 {
		return fmt.Errorf("strategy evaluation %d was not claimable", evaluation.ID)
	}
	diagnostics, err := json.Marshal(map[string]any{
		"source":       verdict.Reason,
		"providerHit":  verdict.ProviderHit,
		"localUsed":    verdict.UsedLocal,
		"localHit":     verdict.Hit,
		"mismatch":     verdict.Mismatch,
		"winningUnits": verdict.WinningUnits,
	})
	if err != nil {
		return err
	}
	if !verdict.UsedLocal {
		if _, err := q.SkipSchemeStrategyEvaluation(ctx, sqlcdb.SkipSchemeStrategyEvaluationParams{
			Diagnostics: diagnostics, ID: evaluation.ID,
		}); err != nil {
			return err
		}
		return q.MarkCloudBetRecordStrategyEvaluated(ctx, record.RecordID)
	}
	status := "completed"
	if verdict.Mismatch {
		status = "mismatch"
	}
	if _, err := q.CompleteSchemeStrategyEvaluationWithStatus(ctx, sqlcdb.CompleteSchemeStrategyEvaluationWithStatusParams{
		CloudBetRecordID: pgtype.Int8{Int64: record.RecordID, Valid: record.RecordID > 0},
		BetOrderNo:       pgtype.Text{String: strings.TrimSpace(orderNo), Valid: strings.TrimSpace(orderNo) != ""},
		RuleVersion:      record.RuleVersion,
		RuleSnapshotHash: record.RuleSnapshotHash,
		LocalHit:         pgtype.Bool{Bool: verdict.Hit, Valid: true},
		WinningUnits:     pgtype.Int4{Int32: int32(verdict.WinningUnits), Valid: true},
		Diagnostics:      diagnostics,
		Status:           status,
		ID:               evaluation.ID,
	}); err != nil {
		return err
	}
	return q.MarkCloudBetRecordStrategyEvaluated(ctx, record.RecordID)
}
