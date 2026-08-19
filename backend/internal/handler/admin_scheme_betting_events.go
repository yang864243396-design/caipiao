package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"caipiao/backend/internal/apix"
)

type adminSchemeBettingEvent struct {
	DecisionID          int64           `json:"decisionId"`
	OutboxID            *int64          `json:"outboxId,omitempty"`
	Origin              *string         `json:"origin,omitempty"`
	SchemeID            string          `json:"schemeId"`
	SchemeName          string          `json:"schemeName"`
	LotteryCode         string          `json:"lotteryCode"`
	SourcePeriod        string          `json:"sourcePeriod"`
	TargetPeriod        *string         `json:"targetPeriod,omitempty"`
	DecisionStatus      string          `json:"decisionStatus"`
	OutcomeReason       *string         `json:"outcomeReason,omitempty"`
	Mode                *string         `json:"mode,omitempty"`
	OutboxState         *string         `json:"outboxState,omitempty"`
	RequestID           *string         `json:"requestId,omitempty"`
	StateVersionBefore  int64           `json:"stateVersionBefore"`
	StateVersionAfter   int64           `json:"stateVersionAfter"`
	ChainState          *string         `json:"chainState,omitempty"`
	BettingOwner        string          `json:"bettingOwner"`
	InitialBet          *bool           `json:"initialBet,omitempty"`
	RequestedAmount     *float64        `json:"requestedAmount,omitempty"`
	RequestedCurrency   *string         `json:"requestedCurrency,omitempty"`
	AttemptCount        *int            `json:"attemptCount,omitempty"`
	ProviderOrderNo     *string         `json:"providerOrderNo,omitempty"`
	AcceptedPeriodNo    *string         `json:"acceptedPeriodNo,omitempty"`
	ProviderAmount      *float64        `json:"providerAmount,omitempty"`
	ProviderCurrency    *string         `json:"providerCurrency,omitempty"`
	ProviderAccountID   *int64          `json:"providerAccountId,omitempty"`
	QueuePosition       *int64          `json:"queuePosition,omitempty"`
	ReadyCreatedAt      *time.Time      `json:"readyCreatedAt,omitempty"`
	AttemptStartedAt    *time.Time      `json:"attemptStartedAt,omitempty"`
	AttemptFinishedAt   *time.Time      `json:"attemptFinishedAt,omitempty"`
	LastError           *string         `json:"lastError,omitempty"`
	DrawSource          *string         `json:"drawSource,omitempty"`
	DrawProviderAt      *time.Time      `json:"drawProviderAt,omitempty"`
	DrawReceivedAt      *time.Time      `json:"drawReceivedAt,omitempty"`
	DrawConfirmedAt     *time.Time      `json:"drawConfirmedAt,omitempty"`
	StrategyStartedAt   *time.Time      `json:"strategyStartedAt,omitempty"`
	StrategyCompletedAt *time.Time      `json:"strategyCompletedAt,omitempty"`
	RuleVersion         *int32          `json:"ruleVersion,omitempty"`
	RuleSnapshotHash    *string         `json:"ruleSnapshotHash,omitempty"`
	LocalHit            *bool           `json:"localHit,omitempty"`
	SafeDeadlineAt      *time.Time      `json:"safeDeadlineAt,omitempty"`
	CloseAt             *time.Time      `json:"closeAt,omitempty"`
	DecidedAt           time.Time       `json:"decidedAt"`
	Diagnostics         json.RawMessage `json:"diagnostics"`
}

func (h *Handler) AdminSchemeBettingEvents(w http.ResponseWriter, r *http.Request) {
	if h.db == nil {
		apix.Fail(w, http.StatusServiceUnavailable, apix.CodeInternal, "\u6570\u636e\u5e93\u672a\u5c31\u7eea")
		return
	}
	limit := queryInt(r, "limit", 100)
	if limit < 1 {
		limit = 1
	}
	if limit > 500 {
		limit = 500
	}
	schemeID := strings.TrimSpace(r.URL.Query().Get("schemeId"))
	rows, err := h.db.Query(r.Context(), `
WITH active_queue AS (
    SELECT id,
           row_number() OVER (PARTITION BY shard_no ORDER BY safe_deadline_at, id) AS queue_position
    FROM scheme_bet_outbox
    WHERE state IN ('pending', 'leased')
),
latest_attempt AS (
    SELECT DISTINCT ON (outbox_id)
           outbox_id, started_at, finished_at, error_message
    FROM scheme_bet_attempts
    ORDER BY outbox_id, attempt_no DESC
)
SELECT COALESCE(d.id, 0), o.id, o.origin, COALESCE(d.scheme_id, ''),
       COALESCE(sd.scheme_name, CASE WHEN o.origin = 'api' THEN 'API' ELSE '' END),
       COALESCE(d.lottery_code, o.lottery_code), COALESCE(d.source_period_no, o.source_period_no),
       o.target_period_no, COALESCE(d.status, 'completed'), o.mode, o.state,
       o.request_id, o.outcome_reason, COALESCE(d.state_version_before, 0), COALESCE(d.state_version_after, 0), d.local_hit,
       o.safe_deadline_at, o.close_at, si.strict_chain_state, COALESCE(si.betting_owner, 'event'), o.initial_bet,
       NULLIF(o.frozen_request #>> '{request,Amount}', '')::double precision,
       NULLIF(o.frozen_request #>> '{request,Currency}', ''),
       o.attempt_count, o.provider_order_no, o.accepted_period_no, o.provider_amount,
       o.provider_currency, o.provider_account_id,
       COALESCE(d.decided_at, o.created_at), COALESCE(d.diagnostics, jsonb_build_object('origin', o.origin)),
       q.queue_position, o.created_at, a.started_at, a.finished_at,
       COALESCE(o.last_error, a.error_message),
       ld.source, ld.drawn_at, ld.received_at, ld.confirmed_at,
       COALESCE(e.claimed_at, e.created_at), e.completed_at,
       COALESCE(d.rule_version, e.rule_version),
       COALESCE(d.rule_snapshot_hash, e.rule_snapshot_hash)
FROM scheme_period_decisions d
FULL OUTER JOIN scheme_bet_outbox o ON o.decision_id = d.id
LEFT JOIN scheme_instances si ON si.id = COALESCE(d.scheme_id, o.scheme_id)
LEFT JOIN scheme_definitions sd ON sd.id = si.definition_id
LEFT JOIN active_queue q ON q.id = o.id
LEFT JOIN latest_attempt a ON a.outbox_id = o.id
LEFT JOIN lottery_draws ld
  ON ld.lottery_code = d.lottery_code AND ld.issue_no = d.source_period_no
LEFT JOIN scheme_strategy_evaluations e
  ON e.instance_id = d.scheme_id AND e.period_no = d.source_period_no
WHERE ($1 = '' OR COALESCE(d.scheme_id, o.scheme_id) = $1)
ORDER BY COALESCE(d.decided_at, o.created_at) DESC, COALESCE(d.id, o.id) DESC
LIMIT $2`, schemeID, limit)
	if err != nil {
		slog.Error("query scheme betting diagnostics failed", "err", err)
		apix.Internal(w)
		return
	}
	defer rows.Close()
	items := make([]adminSchemeBettingEvent, 0, limit)
	for rows.Next() {
		var item adminSchemeBettingEvent
		if err := rows.Scan(
			&item.DecisionID, &item.OutboxID, &item.Origin, &item.SchemeID, &item.SchemeName, &item.LotteryCode,
			&item.SourcePeriod, &item.TargetPeriod, &item.DecisionStatus, &item.Mode,
			&item.OutboxState, &item.RequestID, &item.OutcomeReason, &item.StateVersionBefore, &item.StateVersionAfter,
			&item.LocalHit, &item.SafeDeadlineAt, &item.CloseAt, &item.ChainState, &item.BettingOwner, &item.InitialBet,
			&item.RequestedAmount, &item.RequestedCurrency, &item.AttemptCount, &item.ProviderOrderNo,
			&item.AcceptedPeriodNo, &item.ProviderAmount, &item.ProviderCurrency, &item.ProviderAccountID,
			&item.DecidedAt, &item.Diagnostics, &item.QueuePosition, &item.ReadyCreatedAt,
			&item.AttemptStartedAt, &item.AttemptFinishedAt, &item.LastError,
			&item.DrawSource, &item.DrawProviderAt, &item.DrawReceivedAt, &item.DrawConfirmedAt,
			&item.StrategyStartedAt, &item.StrategyCompletedAt, &item.RuleVersion, &item.RuleSnapshotHash,
		); err != nil {
			slog.Error("scan scheme betting diagnostics failed", "err", err)
			apix.Internal(w)
			return
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		slog.Error("iterate scheme betting diagnostics failed", "err", err)
		apix.Internal(w)
		return
	}
	apix.OK(w, map[string]any{"items": items})
}
