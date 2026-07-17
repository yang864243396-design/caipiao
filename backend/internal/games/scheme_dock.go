package games

import (
	"context"
	"strings"

	"caipiao/backend/internal/schemes"
)

func (s *Service) loadSchemeDock(
	ctx context.Context,
	q DetailQuery,
	openPeriod, latestDrawn string,
) schemes.SchemeDockSummary {
	if s == nil || s.q == nil || strings.TrimSpace(q.SnapshotID) == "" {
		return schemes.SchemeDockSummary{}
	}
	pctx, err := s.loadSnapshotPreviewContext(ctx, q, openPeriod, latestDrawn)
	if err != nil || pctx == nil || pctx.LotteryCode == "" {
		return schemes.SchemeDockSummary{}
	}
	return schemes.ComputeSchemeDockSummary(
		strings.TrimSpace(q.SnapshotID),
		pctx.Kind,
		pctx.ConfigJSON,
		pctx.Draws,
		pctx.LotteryCode,
	)
}
