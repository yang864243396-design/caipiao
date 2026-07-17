package oddsrefresh

import (
	"context"
	"log/slog"
	"time"

	"caipiao/backend/internal/guaji"
	"caipiao/backend/internal/guaji/accountsvc"
	"caipiao/backend/internal/schemes"
)

const defaultInterval = 5 * time.Minute

// Worker 周期性拉取第三方 GET /api/agents/i/real/rate，刷新 schemes 赔率线缓存。
type Worker struct {
	client   *guaji.Client
	accounts *accountsvc.Service
	interval time.Duration
}

func NewWorker(client *guaji.Client, accounts *accountsvc.Service) *Worker {
	if client == nil || !client.Enabled() || accounts == nil {
		return nil
	}
	return &Worker{
		client:   client,
		accounts: accounts,
		interval: defaultInterval,
	}
}

func (w *Worker) Run(ctx context.Context) {
	if w == nil {
		return
	}
	slog.Info("guaji odds refresh started", "interval", w.interval.String())
	w.refresh(ctx)

	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			slog.Info("guaji odds refresh stopped")
			return
		case <-ticker.C:
			w.refresh(ctx)
		}
	}
}

func (w *Worker) refresh(ctx context.Context) {
	token, err := w.accounts.SyncAccessToken(ctx)
	if err != nil {
		slog.Debug("guaji odds refresh: no token", "err", err)
		return
	}
	rate, err := w.client.FetchRealRate(ctx, token)
	if err != nil {
		if guaji.ClassifyUpstreamError(err).IsTokenInvalid {
			slog.Warn("guaji odds refresh token invalid", "err", err)
		} else {
			slog.Warn("guaji odds refresh failed", "err", err)
		}
		return
	}
	if rate == nil || (rate.LottOdds <= 0 && rate.HashOdds <= 0) {
		slog.Debug("guaji odds refresh: empty rate")
		return
	}
	schemes.SetGuajiOdds(rate.LottOdds, rate.HashOdds)
	slog.Info("guaji odds refreshed",
		"lott_odds", rate.LottOdds,
		"hash_odds", rate.HashOdds,
		"real_rate", rate.RealRate,
		"user_type", rate.UserType,
	)
}
