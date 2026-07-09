package schemes

import (
	"context"
	"strings"

	"caipiao/backend/internal/db/sqlcdb"
	"caipiao/backend/internal/guajibet"
)

type guajiMemberAuthChecker interface {
	HasHealthyAuthForMember(ctx context.Context, memberAccount string) (bool, error)
}

func (w *Worker) pauseRunningWithoutGuajiAuth(ctx context.Context, inst sqlcdb.SchemeInstance) bool {
	if !requiresGuajiRealBet(inst) || !w.guajiRealEnabled() {
		return false
	}
	checker, ok := w.guajiBets.(guajiMemberAuthChecker)
	if !ok {
		return false
	}
	var account string
	if err := w.pool.QueryRow(ctx, `SELECT account FROM members WHERE id = $1`, inst.MemberID).Scan(&account); err != nil {
		return false
	}
	account = strings.TrimSpace(account)
	if account == "" {
		return w.pauseRunningInstance(ctx, inst, StatusReasonBetFailed, guajiBetFailedDetail(guajibet.ErrNoActiveAuth))
	}
	healthy, err := checker.HasHealthyAuthForMember(ctx, account)
	if err != nil || healthy {
		return false
	}
	return w.pauseRunningInstance(ctx, inst, StatusReasonBetFailed, guajiBetFailedDetail(guajibet.ErrNoActiveAuth))
}
