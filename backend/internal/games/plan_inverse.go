package games

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"

	"caipiao/backend/internal/db/sqlcdb"
	"caipiao/backend/internal/schemes"
)

func (s *Service) loadPlanInverse(
	ctx context.Context,
	q DetailQuery,
	lhc bool,
	openPeriod, latestDrawn string,
) (digits string, betCount int, supported bool) {
	snapshotID := strings.TrimSpace(q.SnapshotID)
	useDemoFallback := snapshotID == ""

	code := strings.TrimSpace(q.LotteryCode)
	kind := "custom"
	if strings.EqualFold(strings.TrimSpace(q.Board), "contrary") {
		kind = "contrary"
	}

	if s == nil || s.q == nil {
		cfg := buildFallbackSnapshotConfig(q)
		if !schemes.PlanContrarySupportedFromConfig(kind, cfg) {
			return "", 0, false
		}
		if useDemoFallback {
			return defaultPlanInverseDigits(lhc), defaultPlanInverseBetCount(lhc), true
		}
		// 有快照却无库：无反集可展示 → 隐藏 Tab
		return "", 0, false
	}

	contentSeed := snapshotID
	if contentSeed == "" {
		contentSeed = strings.TrimSpace(q.SchemeName)
	}
	if contentSeed == "" {
		contentSeed = code
	}

	var configJSON []byte
	lotteryCode := code

	if snapshotID != "" {
		snap, err := s.q.GetSchemeShareSnapshotByID(ctx, snapshotID)
		if err == nil {
			if lotteryCode == "" {
				lotteryCode = snap.LotteryCode
			}
			configJSON = snap.Config
		} else if !errors.Is(err, pgx.ErrNoRows) {
			cfg := buildFallbackSnapshotConfig(q)
			if !schemes.PlanContrarySupportedFromConfig(kind, cfg) {
				return "", 0, false
			}
			if useDemoFallback {
				return defaultPlanInverseDigits(lhc), defaultPlanInverseBetCount(lhc), true
			}
			return "", 0, false
		}
	}
	if len(configJSON) == 0 {
		configJSON = buildFallbackSnapshotConfig(q)
	}

	if !schemes.PlanContrarySupportedFromConfig(kind, configJSON) {
		return "", 0, false
	}

	if lotteryCode == "" {
		if useDemoFallback {
			return defaultPlanInverseDigits(lhc), defaultPlanInverseBetCount(lhc), true
		}
		return "", 0, false
	}

	draws, err := s.q.ListLotteryDraws(ctx, sqlcdb.ListLotteryDrawsParams{
		LotteryCode: lotteryCode,
		RowLimit:    int32(gameDetailDrawFetchLimit),
	})
	if err != nil {
		if useDemoFallback {
			return defaultPlanInverseDigits(lhc), defaultPlanInverseBetCount(lhc), true
		}
		return "", 0, false
	}
	draws = filterDrawsBeforeOpenPeriod(draws, openPeriod, latestDrawn)

	display := schemes.ComputePlanInverseDisplay(contentSeed, kind, configJSON, draws)
	if display.Digits == "" {
		if useDemoFallback {
			return defaultPlanInverseDigits(lhc), defaultPlanInverseBetCount(lhc), true
		}
		// 玩法虽支持反集，但当前计划算不出可展示反集 → 隐藏 Tab（勿展示「暂无反集数据」）
		return "", 0, false
	}
	return display.Digits, display.BetCount, true
}
