package games

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"

	"caipiao/backend/internal/db/sqlcdb"
	"caipiao/backend/internal/schemes"
)

func (s *Service) loadSnapshotDetailTabs(
	ctx context.Context,
	q DetailQuery,
	openPeriod, latestDrawn, playMethod string,
) (groupBets int, history []PlanTrendRow, chart []PlanTrendChartPoint, records []GameBetRecordRow, err error) {
	if s == nil || s.q == nil {
		return 0, nil, nil, nil, nil
	}
	snapshotID := strings.TrimSpace(q.SnapshotID)
	if snapshotID == "" {
		return 0, nil, nil, nil, nil
	}

	kind := "custom"
	if strings.EqualFold(strings.TrimSpace(q.Board), "contrary") {
		kind = "contrary"
	}

	schemeName := strings.TrimSpace(q.SchemeName)
	var configJSON []byte
	lotteryCode := strings.TrimSpace(q.LotteryCode)

	snap, err := s.q.GetSchemeShareSnapshotByID(ctx, snapshotID)
	if err == nil {
		schemeName = strings.TrimSpace(snap.SchemeName)
		if lotteryCode == "" {
			lotteryCode = snap.LotteryCode
		}
		if playMethod == "" {
			playMethod = strings.TrimSpace(snap.PlayMethod)
		}
		configJSON = snap.Config
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return 0, nil, nil, nil, err
	} else {
		configJSON = buildFallbackSnapshotConfig(q)
		if schemeName == "" {
			schemeName = "方案"
		}
	}

	if lotteryCode == "" {
		return 0, nil, nil, nil, nil
	}

	draws, err := s.q.ListLotteryDraws(ctx, sqlcdb.ListLotteryDrawsParams{
		LotteryCode: lotteryCode,
		RowLimit:    int32(gameDetailBettingRowLimit + 5),
	})
	if err != nil {
		return 0, nil, nil, nil, err
	}
	draws = filterDrawsBeforeOpenPeriod(draws, openPeriod, latestDrawn)

	preview := schemes.RunDetailPreview(schemeName, snapshotID, kind, configJSON, playMethod, draws)

	history = make([]PlanTrendRow, 0, len(preview.PlanTrendHistory))
	for _, row := range preview.PlanTrendHistory {
		history = append(history, PlanTrendRow{Period: row.Period, Win: row.Win})
	}
	chart = make([]PlanTrendChartPoint, 0, len(preview.PlanTrendChart))
	for _, row := range preview.PlanTrendChart {
		chart = append(chart, PlanTrendChartPoint{Period: row.Period, Round: row.Round, Win: row.Win})
	}
	records = make([]GameBetRecordRow, 0, len(preview.BetRecords))
	for _, row := range preview.BetRecords {
		records = append(records, GameBetRecordRow{
			Period:     row.Period,
			PlayMethod: row.PlayMethod,
			Multiplier: row.Multiplier,
			Round:      row.Round,
			Amount:     row.Amount,
			ProfitLoss: row.ProfitLoss,
			Status:     row.Status,
		})
	}
	return preview.PlanTrendGroupBets, history, chart, records, nil
}
