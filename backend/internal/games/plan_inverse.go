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
) (digits string, betCount int) {
	snapshotID := strings.TrimSpace(q.SnapshotID)
	useDemoFallback := snapshotID == ""

	if s == nil || s.q == nil {
		if useDemoFallback {
			return defaultPlanInverseDigits(lhc), defaultPlanInverseBetCount(lhc)
		}
		return "", 0
	}

	code := strings.TrimSpace(q.LotteryCode)
	kind := "custom"
	if strings.EqualFold(strings.TrimSpace(q.Board), "contrary") {
		kind = "contrary"
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
			if useDemoFallback {
				return defaultPlanInverseDigits(lhc), defaultPlanInverseBetCount(lhc)
			}
			return "", 0
		}
	}
	if len(configJSON) == 0 {
		configJSON = buildFallbackSnapshotConfig(q)
	}

	if lotteryCode == "" {
		if useDemoFallback {
			return defaultPlanInverseDigits(lhc), defaultPlanInverseBetCount(lhc)
		}
		return "", 0
	}

	draws, err := s.q.ListLotteryDraws(ctx, sqlcdb.ListLotteryDrawsParams{
		LotteryCode: lotteryCode,
		RowLimit:    int32(gameDetailDrawFetchLimit),
	})
	if err != nil {
		if useDemoFallback {
			return defaultPlanInverseDigits(lhc), defaultPlanInverseBetCount(lhc)
		}
		return "", 0
	}
	draws = filterDrawsBeforeOpenPeriod(draws, openPeriod, latestDrawn)

	display := schemes.ComputePlanInverseDisplay(contentSeed, kind, configJSON, draws)
	if display.Digits == "" {
		if useDemoFallback {
			return defaultPlanInverseDigits(lhc), defaultPlanInverseBetCount(lhc)
		}
		return "", 0
	}
	return display.Digits, display.BetCount
}
