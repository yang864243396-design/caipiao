package games

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"

	"caipiao/backend/internal/db/sqlcdb"
	"caipiao/backend/internal/schemes"
)

type snapshotPreviewContext struct {
	SchemeName  string
	LotteryCode string
	Kind        string
	ConfigJSON  []byte
	Draws       []sqlcdb.ListLotteryDrawsRow
}

func (s *Service) loadSnapshotPreviewContext(
	ctx context.Context,
	q DetailQuery,
	openPeriod, latestDrawn string,
) (*snapshotPreviewContext, error) {
	if s == nil || s.q == nil {
		return nil, nil
	}
	snapshotID := strings.TrimSpace(q.SnapshotID)
	if snapshotID == "" {
		return nil, nil
	}

	code := strings.TrimSpace(q.LotteryCode)
	kind := "custom"
	if strings.EqualFold(strings.TrimSpace(q.Board), "contrary") {
		kind = "contrary"
	}

	schemeName := strings.TrimSpace(q.SchemeName)
	var configJSON []byte
	lotteryCode := code

	snap, err := s.q.GetSchemeShareSnapshotByID(ctx, snapshotID)
	if err == nil {
		schemeName = strings.TrimSpace(snap.SchemeName)
		if lotteryCode == "" {
			lotteryCode = snap.LotteryCode
		}
		configJSON = snap.Config
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	} else {
		configJSON = buildFallbackSnapshotConfig(q)
		if schemeName == "" {
			schemeName = "方案"
		}
	}

	if lotteryCode == "" {
		return &snapshotPreviewContext{SchemeName: schemeName, Kind: kind, ConfigJSON: configJSON}, nil
	}

	draws, err := s.q.ListLotteryDraws(ctx, sqlcdb.ListLotteryDrawsParams{
		LotteryCode: lotteryCode,
		RowLimit:    int32(gameDetailDrawFetchLimit),
	})
	if err != nil {
		return nil, err
	}
	draws = filterDrawsBeforeOpenPeriod(draws, openPeriod, latestDrawn)

	return &snapshotPreviewContext{
		SchemeName:  schemeName,
		LotteryCode: lotteryCode,
		Kind:        kind,
		ConfigJSON:  configJSON,
		Draws:       draws,
	}, nil
}

type detailPreviewExtras struct {
	BetRecords         []GameBetRecordRow
	PlanTrendGroupBets int
	PlanTrendHistory   []PlanTrendRow
	PlanTrendChart     []PlanTrendChartPoint
}

func (s *Service) loadDetailPreviewExtras(
	ctx context.Context,
	q DetailQuery,
	openPeriod, latestDrawn, playMethod string,
) (detailPreviewExtras, error) {
	out := detailPreviewExtras{}
	pctx, err := s.loadSnapshotPreviewContext(ctx, q, openPeriod, latestDrawn)
	if err != nil {
		return out, err
	}
	if pctx == nil || pctx.LotteryCode == "" {
		return out, nil
	}

	bundle := schemes.RunDetailPreview(
		pctx.SchemeName,
		strings.TrimSpace(q.SnapshotID),
		pctx.Kind,
		pctx.ConfigJSON,
		playMethod,
		pctx.Draws,
	)

	out.PlanTrendGroupBets = bundle.PlanTrendGroupBets
	if out.PlanTrendGroupBets <= 0 {
		out.PlanTrendGroupBets = 1
	}

	out.PlanTrendHistory = make([]PlanTrendRow, 0, len(bundle.PlanTrendHistory))
	for _, row := range bundle.PlanTrendHistory {
		out.PlanTrendHistory = append(out.PlanTrendHistory, PlanTrendRow{
			Period: row.Period,
			Win:    row.Win,
		})
	}

	out.PlanTrendChart = make([]PlanTrendChartPoint, 0, len(bundle.PlanTrendChart))
	for _, pt := range bundle.PlanTrendChart {
		out.PlanTrendChart = append(out.PlanTrendChart, PlanTrendChartPoint{
			Period: pt.Period,
			Round:  pt.Round,
			Win:    pt.Win,
		})
	}

	out.BetRecords = make([]GameBetRecordRow, 0, len(bundle.BetRecords))
	for _, row := range bundle.BetRecords {
		out.BetRecords = append(out.BetRecords, GameBetRecordRow{
			Period:     row.Period,
			PlayMethod: row.PlayMethod,
			Multiplier: row.Multiplier,
			Round:      row.Round,
			Amount:     row.Amount,
			ProfitLoss: row.ProfitLoss,
			Status:     row.Status,
		})
	}

	return out, nil
}

func buildFallbackSnapshotConfig(q DetailQuery) []byte {
	cfg := map[string]interface{}{}
	if name := strings.TrimSpace(q.SchemeName); name != "" {
		cfg["schemeName"] = name
	}
	if pm := strings.TrimSpace(q.PlayMethod); pm != "" {
		cfg["playMethod"] = pm
	}
	if pt := strings.TrimSpace(q.PlayTypeID); pt != "" {
		cfg["playTypeId"] = pt
		cfg["typeId"] = pt
	}
	if sp := strings.TrimSpace(q.SubPlayID); sp != "" {
		cfg["subPlayId"] = sp
		cfg["subId"] = sp
	}
	cfg["runTypeId"] = schemes.RunTypeFixedRotate
	raw, _ := json.Marshal(cfg)
	return raw
}
