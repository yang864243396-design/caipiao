package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5/pgtype"

	"caipiao/backend/internal/apix"
)

type adminStrategyDiagnosticItem struct {
	PeriodNo         string   `json:"periodNo"`
	RecordNo         string   `json:"recordNo,omitempty"`
	BetOrderNo       string   `json:"betOrderNo,omitempty"`
	ThirdPartyBetID  string   `json:"thirdPartyBetId,omitempty"`
	CloudStatus      string   `json:"cloudStatus,omitempty"`
	EvaluationStatus string   `json:"evaluationStatus"`
	PipelineStatus   string   `json:"pipelineStatus"`
	ReconcileStatus  string   `json:"reconcileStatus"`
	RuleSnapshot     bool     `json:"ruleSnapshot"`
	RuleVersion      *int32   `json:"ruleVersion,omitempty"`
	RuleSnapshotHash string   `json:"ruleSnapshotHash,omitempty"`
	LocalHit         *bool    `json:"localHit,omitempty"`
	ProviderHit      *bool    `json:"providerHit,omitempty"`
	WinningUnits     *int32   `json:"winningUnits,omitempty"`
	Source           string   `json:"source,omitempty"`
	DrawBalls        []string `json:"drawBalls,omitempty"`
	DrawnAt          string   `json:"drawnAt,omitempty"`
	DrawIngestedAt   string   `json:"drawIngestedAt,omitempty"`
	DrawIngestLagMs  *int64   `json:"drawIngestLagMs,omitempty"`
	CompletedAt      string   `json:"completedAt,omitempty"`
	Diagnostics      any      `json:"diagnostics,omitempty"`
}

func strategyPipelineStatus(hasDraw, hasSnapshot bool, evaluationStatus string) string {
	if status := strings.TrimSpace(evaluationStatus); status != "" {
		return status
	}
	if !hasDraw {
		return "awaiting_draw"
	}
	if !hasSnapshot {
		return "missing_rule_snapshot"
	}
	return "awaiting_evaluation"
}

func strategyReconciliationStatus(evaluationStatus string, localHit, providerHit *bool) string {
	if localHit != nil && providerHit != nil && *localHit != *providerHit {
		return "mismatch"
	}
	return strings.TrimSpace(evaluationStatus)
}

// AdminSchemeStrategyDiagnostics is developer-only evidence for the formal
// rule path. It compares frozen-rule hit results with the provider result; it
// deliberately does not infer disagreement from rounded monetary pnl.
func (h *Handler) AdminSchemeStrategyDiagnostics(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.db == nil {
		apix.Fail(w, http.StatusServiceUnavailable, apix.CodeInternal, "数据库未就绪")
		return
	}
	instanceID := strings.TrimSpace(r.PathValue("instanceId"))
	if instanceID == "" {
		apix.Validation(w, "instanceId 不能为空")
		return
	}
	var exists bool
	if err := h.db.QueryRow(r.Context(), `SELECT EXISTS(SELECT 1 FROM scheme_instances WHERE id = $1)`, instanceID).Scan(&exists); err != nil {
		apix.Internal(w)
		return
	}
	if !exists {
		apix.Fail(w, http.StatusNotFound, apix.CodeNotFound, "方案实例不存在")
		return
	}

	rows, err := h.db.Query(r.Context(), `
SELECT c.period_no,
       c.record_no,
       COALESCE(c.bet_order_no, ''),
       COALESCE(c.third_party_bet_id, ''),
       c.status,
       COALESCE(e.status, ''),
       COALESCE(e.rule_version, c.rule_version),
       COALESCE(e.rule_snapshot_hash, c.rule_snapshot_hash, ''),
       e.local_hit,
       e.winning_units,
       COALESCE(e.diagnostics, '{}'::jsonb)::text,
       COALESCE(d.balls::text, '[]'),
       COALESCE(d.drawn_at::text, ''),
       COALESCE(d.created_at::text, ''),
       CASE WHEN d.id IS NULL THEN NULL
            ELSE round(extract(epoch FROM (d.created_at - d.drawn_at)) * 1000)::bigint END,
       COALESCE(e.completed_at::text, ''),
       c.rule_snapshot IS NOT NULL
FROM cloud_bet_records c
LEFT JOIN scheme_strategy_evaluations e
  ON e.instance_id = c.scheme_id AND e.period_no = c.period_no
LEFT JOIN lottery_draws d
  ON d.lottery_code = c.lottery_code AND d.issue_no = c.period_no
WHERE c.scheme_id = $1
  AND c.sim_bet = FALSE
  AND NULLIF(TRIM(c.third_party_bet_id), '') IS NOT NULL
ORDER BY c.placed_at DESC, c.id DESC
LIMIT 50`, instanceID)
	if err != nil {
		apix.Internal(w)
		return
	}
	defer rows.Close()

	items := make([]adminStrategyDiagnosticItem, 0)
	for rows.Next() {
		var item adminStrategyDiagnosticItem
		var ruleVersion pgtype.Int4
		var localHit pgtype.Bool
		var winningUnits pgtype.Int4
		var drawIngestLagMs pgtype.Int8
		var diagnosticsRaw, ballsRaw []byte
		if err := rows.Scan(
			&item.PeriodNo, &item.RecordNo, &item.BetOrderNo, &item.ThirdPartyBetID, &item.CloudStatus,
			&item.EvaluationStatus, &ruleVersion, &item.RuleSnapshotHash, &localHit, &winningUnits,
			&diagnosticsRaw, &ballsRaw, &item.DrawnAt, &item.DrawIngestedAt, &drawIngestLagMs,
			&item.CompletedAt, &item.RuleSnapshot,
		); err != nil {
			apix.Internal(w)
			return
		}
		if ruleVersion.Valid {
			v := ruleVersion.Int32
			item.RuleVersion = &v
		}
		if localHit.Valid {
			v := localHit.Bool
			item.LocalHit = &v
		}
		if winningUnits.Valid {
			v := winningUnits.Int32
			item.WinningUnits = &v
		}
		if drawIngestLagMs.Valid {
			v := drawIngestLagMs.Int64
			item.DrawIngestLagMs = &v
		}
		_ = json.Unmarshal(ballsRaw, &item.DrawBalls)
		var details map[string]any
		if json.Unmarshal(diagnosticsRaw, &details) == nil {
			item.Diagnostics = details
			if source, ok := details["source"].(string); ok {
				item.Source = source
			}
			if providerHit, ok := details["providerHit"].(bool); ok {
				item.ProviderHit = &providerHit
			}
		}
		if item.CloudStatus == "hit" {
			v := true
			item.ProviderHit = &v
		} else if item.CloudStatus == "miss" {
			v := false
			item.ProviderHit = &v
		}
		item.PipelineStatus = strategyPipelineStatus(len(item.DrawBalls) > 0, item.RuleSnapshot, item.EvaluationStatus)
		item.ReconcileStatus = strategyReconciliationStatus(item.PipelineStatus, item.LocalHit, item.ProviderHit)
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		apix.Internal(w)
		return
	}
	apix.OK(w, map[string]any{"instanceId": instanceID, "items": items})
}
