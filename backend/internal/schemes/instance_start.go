package schemes

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"caipiao/backend/internal/db/sqlcdb"
	"caipiao/backend/internal/guajibet"
	"caipiao/backend/internal/member"
)

// ErrStartTimeNotAfterNow 方案开始时间不晚于当前时刻，不可开启。
var ErrStartTimeNotAfterNow = errors.New("预计开启时间小于现在时间 请修改后再执行开启")

// ErrEndTimeReached 方案已过结束时间，不可开启。
var ErrEndTimeReached = errors.New("方案已过结束时间 请修改后再执行开启")

// ErrStartInsufficientFunds 真实投注开启时第三方可用余额不足以覆盖预计首期投注额。
var ErrStartInsufficientFunds = errors.New("可用余额不足，请充值后再开启")

func (s *Service) StartInstance(ctx context.Context, account, instanceID string) (Instance, error) {
	return s.startInstance(ctx, account, instanceID)
}

// ResumeInstance 兼容旧客户端；维护停投续投保留指标，其它与 StartInstance 相同（新开运行段并清零）。
func (s *Service) ResumeInstance(ctx context.Context, account, instanceID string) (Instance, error) {
	return s.startInstance(ctx, account, instanceID)
}

func (s *Service) startInstance(ctx context.Context, account, instanceID string) (Instance, error) {
	if s == nil || s.q == nil {
		return Instance{}, ErrUnavailable
	}
	instanceID = strings.TrimSpace(instanceID)
	if instanceID == "" {
		return Instance{}, ErrDefinitionNotFound
	}

	m, err := s.q.GetMemberByAccount(ctx, account)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Instance{}, member.ErrNotFound
		}
		return Instance{}, err
	}

	cur, err := s.q.GetSchemeInstanceByIDAndMember(ctx, sqlcdb.GetSchemeInstanceByIDAndMemberParams{
		ID: instanceID, MemberID: m.ID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Instance{}, ErrDefinitionNotFound
		}
		return Instance{}, err
	}
	if cur.Status == "soft_stopped" {
		return Instance{}, ErrInvalidInstanceAction
	}
	if cur.Status != "pending" && cur.Status != "paused" {
		return Instance{}, ErrInvalidInstanceAction
	}

	// 真实投注：本平台无可用授权时拒绝开启，避免后续 worker 再调第三方。
	if !cur.SimBet && s.authChecker != nil {
		healthy, aerr := s.authChecker.HasHealthyAuthForMember(ctx, account)
		if aerr != nil {
			return Instance{}, aerr
		}
		if !healthy {
			return Instance{}, guajibet.ErrNoActiveAuth
		}
	}

	def, err := s.q.GetSchemeDefinitionByID(ctx, cur.DefinitionID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Instance{}, ErrDefinitionNotFound
		}
		return Instance{}, err
	}
	now := time.Now()
	isMaintenanceResume := cur.Status == "pending" && cur.StatusReason == StatusReasonMaintenance
	// 维护续投：方案已在跑过，开始时间通常在过去；§6.4 由 canResumeAfterMaintenance 校验时间窗。
	if !isMaintenanceResume {
		if err := validateSchemeStartTimeAfterNow(def.Config, now); err != nil {
			return Instance{}, err
		}
	}
	if err := validateSchemeEndTimeNotReached(def.Config, now); err != nil {
		return Instance{}, err
	}
	currency := schemeCurrencyFromConfig(def.Config)
	if err := validateSchemeMinBetAmount(def.Config, def.Kind, currency, cur.Multiplier); err != nil {
		return Instance{}, err
	}
	// 真实投注：开启前预检第三方余额，避免先显示「运行中」再因余额不足停投。
	if !cur.SimBet {
		need := schemeMinSingleBetAmount(def.Config, def.Kind, cur.Multiplier)
		if err := s.ensureUsableBalanceForStart(ctx, account, need, currency); err != nil {
			return Instance{}, err
		}
	}

	if isMaintenanceResume {
		inst := sqlcdb.SchemeInstanceFromMemberRow(cur)
		if !canResumeAfterMaintenance(ctx, s.q, inst, def.Config, now) {
			return Instance{}, ErrMaintenanceResumeBlocked
		}
		row, err := s.q.ResumeSchemeInstanceAfterMaintenance(ctx, sqlcdb.ResumeSchemeInstanceAfterMaintenanceParams{
			ID: instanceID, MemberID: m.ID,
		})
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return Instance{}, ErrInvalidInstanceAction
			}
			return Instance{}, err
		}
		displayRow := sqlcdb.SchemeInstanceFromMaintenanceResumeRow(row)
		return s.enrichInstanceForDisplay(ctx, displayRow, now), nil
	}

	// 模拟方案：同时运行 ≤5，且北京时间自然日启动 ≤5（维护续投不计入）。
	// 依赖 members.sim_scheme_starts_*（迁移 00130）；缺列会映射为 ErrSimSchemeQuotaSchema。
	var simQuotaDay time.Time
	if cur.SimBet {
		day, qerr := s.enforceSimSchemeStartQuota(ctx, m.ID, now)
		if qerr != nil {
			slog.Error("scheme sim start quota failed", "instanceId", instanceID, "memberId", m.ID, "err", qerr)
			return Instance{}, qerr
		}
		simQuotaDay = day
	}

	row, err := s.q.UpdateSchemeInstanceStatusFromPendingToRunning(ctx, sqlcdb.UpdateSchemeInstanceStatusFromPendingToRunningParams{
		ID: instanceID, MemberID: m.ID, Column3: StatusReasonAwaitNextBet,
	})
	if err != nil {
		if cur.SimBet {
			s.releaseSimSchemeStart(ctx, m.ID, simQuotaDay)
		}
		if errors.Is(err, pgx.ErrNoRows) {
			return Instance{}, ErrInvalidInstanceAction
		}
		return Instance{}, err
	}
	displayRow := sqlcdb.SchemeInstanceFromPendingToRunningRow(row)
	if evaluateSchemeScheduleGate(def.Config, now) == schemeScheduleOK {
		if skipPeriod, ok, syncErr := markSchemeStartPeriodSkipped(ctx, s.q, s.periodSync, instanceID, row.LotteryCode); syncErr != nil {
			slog.Warn("scheme start read periods cache failed", "instanceId", instanceID, "lottery", row.LotteryCode, "err", syncErr)
		} else if ok && skipPeriod != "" {
			if refreshed, rerr := s.q.GetSchemeInstanceByIDAndMember(ctx, sqlcdb.GetSchemeInstanceByIDAndMemberParams{
				ID: instanceID, MemberID: m.ID,
			}); rerr == nil {
				displayRow = sqlcdb.SchemeInstanceFromMemberRow(refreshed)
			}
		}
	}
	return s.enrichInstanceForDisplay(ctx, displayRow, time.Now()), nil
}

func validateSchemeStartTimeAfterNow(cfgBytes []byte, now time.Time) error {
	startAt, ok := schemeConfigStartTime(cfgBytes)
	if !ok {
		return nil
	}
	if !startAt.After(now) {
		return ErrStartTimeNotAfterNow
	}
	return nil
}

func validateSchemeEndTimeNotReached(cfgBytes []byte, now time.Time) error {
	if schemeConfigEndTimeReached(cfgBytes, now) {
		return ErrEndTimeReached
	}
	return nil
}

func (s *Service) ensureUsableBalanceForStart(ctx context.Context, account string, need float64, currency string) error {
	if s == nil || s.authChecker == nil {
		return nil
	}
	if need < 0 {
		need = 0
	}
	bal, ok, err := s.authChecker.UsableBalance(ctx, account, currency)
	if err != nil {
		return err
	}
	if !ok {
		return nil
	}
	if bal+1e-9 < need {
		return ErrStartInsufficientFunds
	}
	return nil
}
