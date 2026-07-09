package schemes

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"caipiao/backend/internal/db/sqlcdb"
	"caipiao/backend/internal/member"
)

// ErrStartTimeNotAfterNow 方案开始时间不晚于当前时刻，不可开启。
var ErrStartTimeNotAfterNow = errors.New("预计开启时间小于现在时间 请修改后再执行开启")

// ErrEndTimeReached 方案已过结束时间，不可开启。
var ErrEndTimeReached = errors.New("方案已过结束时间 请修改后再执行开启")

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

	row, err := s.q.UpdateSchemeInstanceStatusFromPendingToRunning(ctx, sqlcdb.UpdateSchemeInstanceStatusFromPendingToRunningParams{
		ID: instanceID, MemberID: m.ID, Column3: StatusReasonAwaitNextBet,
	})
	if err != nil {
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
