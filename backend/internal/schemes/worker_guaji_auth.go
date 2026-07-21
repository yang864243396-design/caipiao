package schemes

import (
	"context"
	"errors"
	"strings"

	"caipiao/backend/internal/db/sqlcdb"
	"caipiao/backend/internal/guaji/accountsvc"
	"caipiao/backend/internal/guajibet"
)

type guajiMemberAuthChecker interface {
	HasHealthyAuthForMember(ctx context.Context, memberAccount string) (bool, error)
}

// guajiMemberAuthEnsurer token 失效时尝试自动重新授权（最多 3 次）。
type guajiMemberAuthEnsurer interface {
	EnsureActiveAuth(ctx context.Context, memberAccount string) error
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
	if err != nil {
		return false
	}
	if healthy {
		return false
	}
	// token 失效：先自动重新授权最多 3 次；成功则不停止方案，失败再按原逻辑停投。
	if ensurer, ok := w.guajiBets.(guajiMemberAuthEnsurer); ok {
		if ensureErr := ensurer.EnsureActiveAuth(ctx, account); ensureErr == nil {
			if ok2, herr := checker.HasHealthyAuthForMember(ctx, account); herr == nil && ok2 {
				return false
			}
		} else if errors.Is(ensureErr, accountsvc.ErrNoActiveAccount) || errors.Is(ensureErr, accountsvc.ErrAccountNotFound) {
			return w.pauseRunningInstance(ctx, inst, StatusReasonBetFailed, guajiBetFailedDetail(guajibet.ErrNoActiveAuth))
		}
	}
	return w.pauseRunningInstance(ctx, inst, StatusReasonBetFailed, guajiBetFailedDetail(guajibet.ErrTokenInvalid))
}
