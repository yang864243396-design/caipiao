package handler

import (
	"net/http"
	"time"

	"caipiao/backend/internal/apix"
)

type adminSchemeBettingSummary struct {
	Pending                       int64     `json:"pending"`
	Leased                        int64     `json:"leased"`
	SentUnknown                   int64     `json:"sentUnknown"`
	ExternalUnknown               int64     `json:"externalUnknown"`
	AcceptedWrongPeriod           int64     `json:"acceptedWrongPeriod"`
	Expired                       int64     `json:"expired"`
	DeadlineRisk                  int64     `json:"deadlineRisk"`
	BlockedRequiresRearm          int64     `json:"blockedRequiresRearm"`
	RunningEventOwned             int64     `json:"runningEventOwned"`
	APIDue                        int64     `json:"apiDue"`
	AcceptedUnfinalized           int64     `json:"acceptedUnfinalized"`
	ActiveStrategyLeases          int64     `json:"activeStrategyLeases"`
	ActiveDispatcherLeases        int64     `json:"activeDispatcherLeases"`
	ActiveDrawLeases              int64     `json:"activeDrawLeases"`
	CurrentGlobalDispatches       int64     `json:"currentGlobalDispatches"`
	DrawToStrategyP99Ms           float64   `json:"drawToStrategyP99Ms"`
	StrategyToAcceptedP99Ms       float64   `json:"strategyToAcceptedP99Ms"`
	SafeDeadlineCompletionRate    float64   `json:"safeDeadlineCompletionRate"`
	ProviderPeriodConsistencyRate float64   `json:"providerPeriodConsistencyRate"`
	OldestPendingAgeSeconds       float64   `json:"oldestPendingAgeSeconds"`
	Modes                         []string  `json:"modes"`
	MeasuredAt                    time.Time `json:"measuredAt"`
}

func (h *Handler) AdminSchemeBettingSummary(w http.ResponseWriter, r *http.Request) {
	if h.db == nil {
		apix.Fail(w, http.StatusServiceUnavailable, apix.CodeInternal, "\u6570\u636e\u5e93\u672a\u5c31\u7eea")
		return
	}
	var summary adminSchemeBettingSummary
	err := h.db.QueryRow(r.Context(), `
SELECT
    count(*) FILTER (WHERE state = 'pending'),
    count(*) FILTER (WHERE state = 'leased'),
    count(*) FILTER (WHERE state = 'sent_unknown'),
    count(*) FILTER (WHERE state = 'external_acceptance_unknown'),
    count(*) FILTER (WHERE state = 'accepted_wrong_period'),
    count(*) FILTER (WHERE state = 'expired'),
    count(*) FILTER (WHERE origin = 'api' AND state IN ('pending', 'leased')),
    count(*) FILTER (WHERE state = 'accepted' AND financial_finalized_at IS NULL),
    count(*) FILTER (WHERE state IN ('pending', 'leased') AND safe_deadline_at <= now() + interval '2 seconds'),
    COALESCE(EXTRACT(epoch FROM (now() - min(created_at) FILTER (WHERE state = 'pending'))), 0),
    COALESCE(array_agg(DISTINCT mode ORDER BY mode), ARRAY[]::varchar[])
FROM scheme_bet_outbox`).Scan(
		&summary.Pending, &summary.Leased, &summary.SentUnknown, &summary.ExternalUnknown, &summary.AcceptedWrongPeriod,
		&summary.Expired, &summary.APIDue, &summary.AcceptedUnfinalized,
		&summary.DeadlineRisk, &summary.OldestPendingAgeSeconds, &summary.Modes,
	)
	if err != nil {
		apix.Internal(w)
		return
	}
	if err := h.db.QueryRow(r.Context(), `
SELECT
    count(*) FILTER (WHERE strict_chain_state = 'blocked_requires_rearm'),
    count(*) FILTER (WHERE betting_owner = 'event' AND status = 'running')
FROM scheme_instances`).Scan(&summary.BlockedRequiresRearm, &summary.RunningEventOwned); err != nil {
		apix.Internal(w)
		return
	}
	if err := h.db.QueryRow(r.Context(), `
SELECT
    count(*) FILTER (WHERE lease_kind = 'strategy' AND lease_until > now()),
    count(*) FILTER (WHERE lease_kind = 'dispatcher' AND lease_until > now()),
    (SELECT count(*) FROM scheme_betting_draw_leases WHERE lease_until > now()),
    COALESCE((
        SELECT dispatch_count
        FROM scheme_betting_dispatch_rate_buckets
        WHERE scope_type = 'global'
          AND scope_key = 'global'
          AND window_start = date_trunc('second', now())
    ), 0)
FROM scheme_betting_shard_leases`).Scan(
		&summary.ActiveStrategyLeases, &summary.ActiveDispatcherLeases,
		&summary.ActiveDrawLeases, &summary.CurrentGlobalDispatches,
	); err != nil {
		apix.Internal(w)
		return
	}
	if err := h.db.QueryRow(r.Context(), `
WITH recent AS (
    SELECT o.id, o.decision_id, o.state, o.target_period_no, o.accepted_period_no,
           o.safe_deadline_at, o.terminal_at, o.created_at,
           d.decided_at, ld.confirmed_at
    FROM scheme_bet_outbox o
    LEFT JOIN scheme_period_decisions d ON d.id = o.decision_id
    LEFT JOIN lottery_draws ld
      ON ld.lottery_code = d.lottery_code AND ld.issue_no = d.source_period_no
    WHERE o.mode IN ('gray', 'production')
      AND o.created_at >= now() - interval '24 hours'
)
SELECT
    COALESCE(percentile_cont(0.99) WITHIN GROUP (
        ORDER BY GREATEST(EXTRACT(epoch FROM (decided_at - confirmed_at)) * 1000, 0)
    ) FILTER (WHERE decided_at IS NOT NULL AND confirmed_at IS NOT NULL), 0),
    COALESCE(percentile_cont(0.99) WITHIN GROUP (
        ORDER BY GREATEST(EXTRACT(epoch FROM (terminal_at - decided_at)) * 1000, 0)
    ) FILTER (WHERE state = 'accepted' AND terminal_at IS NOT NULL AND decided_at IS NOT NULL), 0),
    COALESCE(
        100.0 * count(*) FILTER (
            WHERE terminal_at IS NOT NULL
              AND terminal_at <= safe_deadline_at
              AND state NOT IN ('sent_unknown', 'external_acceptance_unknown')
        ) / NULLIF(count(*), 0),
        0
    ),
    COALESCE(
        100.0 * count(*) FILTER (
            WHERE state = 'accepted' AND accepted_period_no = target_period_no
        ) / NULLIF(count(*) FILTER (
            WHERE state IN ('accepted', 'accepted_wrong_period')
              AND accepted_period_no IS NOT NULL
        ), 0),
        0
    )
FROM recent`).Scan(
		&summary.DrawToStrategyP99Ms, &summary.StrategyToAcceptedP99Ms,
		&summary.SafeDeadlineCompletionRate, &summary.ProviderPeriodConsistencyRate,
	); err != nil {
		apix.Internal(w)
		return
	}
	summary.MeasuredAt = time.Now().UTC()
	apix.OK(w, summary)
}
