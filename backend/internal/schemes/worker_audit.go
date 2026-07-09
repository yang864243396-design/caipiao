package schemes

import (
	"context"
	"fmt"

	"caipiao/backend/internal/db/sqlcdb"
)

func appendWorkerAudit(ctx context.Context, qtx *sqlcdb.Queries, action string) error {
	if action == "" {
		return nil
	}
	_, err := qtx.InsertAdminAuditLog(ctx, sqlcdb.InsertAdminAuditLogParams{
		Actor:  "scheme-worker",
		Action: action,
		Ip:     "127.0.0.1",
	})
	return err
}

func lookbackResetAuditAction(kind string, inst sqlcdb.SchemeInstance, periodNo string, instanceCount int) string {
	switch kind {
	case "individual":
		return fmt.Sprintf("回头复位(个别) memberId=%d instance=%s period=%s", inst.MemberID, inst.ID, periodNo)
	case "overall":
		return fmt.Sprintf("回头复位(整体) memberId=%d simBet=%v period=%s instances=%d", inst.MemberID, inst.SimBet, periodNo, instanceCount)
	default:
		return ""
	}
}

func appendLookbackResetAudit(
	ctx context.Context,
	qtx *sqlcdb.Queries,
	inst sqlcdb.SchemeInstance,
	periodNo string,
	resetIndividual, resetOverall bool,
	instanceCount int,
) error {
	if resetIndividual && !resetOverall {
		return appendWorkerAudit(ctx, qtx, lookbackResetAuditAction("individual", inst, periodNo, 0))
	}
	if resetOverall {
		return appendWorkerAudit(ctx, qtx, lookbackResetAuditAction("overall", inst, periodNo, instanceCount))
	}
	return nil
}

func appendInsufficientFundsAudit(ctx context.Context, qtx *sqlcdb.Queries, inst sqlcdb.SchemeInstance, amount float64) error {
	action := fmt.Sprintf("余额不足暂停 memberId=%d instance=%s amount=%.2f", inst.MemberID, inst.ID, amount)
	return appendWorkerAudit(ctx, qtx, action)
}

// appendPickSkipAudit 出号策略本期跳过（如开某投某无可用映射行）。
func appendPickSkipAudit(ctx context.Context, q *sqlcdb.Queries, inst sqlcdb.SchemeInstance, periodNo string) error {
	action := fmt.Sprintf("出号跳过 memberId=%d instance=%s period=%s", inst.MemberID, inst.ID, periodNo)
	return appendWorkerAudit(ctx, q, action)
}
