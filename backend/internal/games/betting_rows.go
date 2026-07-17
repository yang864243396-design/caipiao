package games

import (
	"context"
	"strings"

	"caipiao/backend/internal/guajibet"
	"caipiao/backend/internal/schemes"
)

const (
	gameDetailBettingRowLimit = 20
	gameDetailDrawFetchLimit  = 120
)

func (s *Service) loadBettingRows(
	ctx context.Context,
	q DetailQuery,
	openPeriod, latestDrawn string,
) ([]BettingExecutionRow, error) {
	if s == nil || s.q == nil {
		return nil, nil
	}
	if strings.TrimSpace(q.SnapshotID) == "" {
		return []BettingExecutionRow{}, nil
	}

	pctx, err := s.loadSnapshotPreviewContext(ctx, q, openPeriod, latestDrawn)
	if err != nil {
		return nil, err
	}
	if pctx == nil || pctx.LotteryCode == "" {
		return []BettingExecutionRow{}, nil
	}

	bundle := schemes.RunDetailPreview(
		pctx.SchemeName,
		strings.TrimSpace(q.SnapshotID),
		pctx.Kind,
		pctx.ConfigJSON,
		strings.TrimSpace(q.PlayMethod),
		pctx.Draws,
		pctx.LotteryCode,
	)

	out := make([]BettingExecutionRow, 0, len(bundle.Executions))
	for _, row := range bundle.Executions {
		out = append(out, BettingExecutionRow{
			Time:    row.Time,
			Scheme:  row.Scheme,
			Numbers: row.Numbers,
			Period:  row.Period,
			Draw:    row.Draw,
			Win:     row.Win,
		})
	}
	return out, nil
}

func displayBetContent(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "—"
	}
	if guajibet.IsSSCDingweiBetContent(raw) {
		parts := strings.Split(raw, ",")
		var picks []string
		for _, seg := range parts {
			seg = strings.TrimSpace(seg)
			if seg == "" {
				continue
			}
			for _, ch := range seg {
				picks = append(picks, string(ch))
			}
		}
		if len(picks) > 0 {
			return strings.Join(picks, " ")
		}
	}
	raw = strings.ReplaceAll(raw, "\r\n", "\n")
	raw = strings.ReplaceAll(raw, "\n", " ")
	raw = strings.ReplaceAll(raw, ",", " ")
	fields := strings.Fields(raw)
	if len(fields) == 0 {
		return "—"
	}
	return strings.Join(fields, " ")
}

func formatDrawBalls(balls []string) string {
	if len(balls) == 0 {
		return "—"
	}
	return strings.Join(balls, " ")
}

func shortIssueNo(issue string) string {
	issue = strings.TrimSpace(issue)
	if issue == "" {
		return "—"
	}
	if len(issue) <= 3 {
		return issue
	}
	return issue[len(issue)-3:]
}
