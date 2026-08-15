package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"caipiao/backend/internal/apix"
	"caipiao/backend/internal/lottery"
)

type adminSchemeRuntimeInstance struct {
	ID               string `json:"id"`
	DefinitionID     string `json:"definitionId"`
	SchemeName       string `json:"schemeName"`
	LotteryCode      string `json:"lotteryCode"`
	Status           string `json:"status"`
	StatusReason     string `json:"statusReason"`
	LastSettledIssue string `json:"lastSettledIssue,omitempty"`
	StartSkipPeriod  string `json:"startSkipPeriod,omitempty"`
}

type adminSchemeRuntimeDraw struct {
	IssueNo string   `json:"issueNo"`
	Balls   []string `json:"balls"`
	DrawnAt string   `json:"drawnAt"`
}

type adminSchemeAcceptedPendingBet struct {
	RecordNo                string `json:"recordNo"`
	ThirdPartyBetID         string `json:"thirdPartyBetId"`
	PeriodNo                string `json:"periodNo"`
	ThirdPartyPeriod        string `json:"thirdPartyPeriod,omitempty"`
	PlacedAt                string `json:"placedAt"`
	AgeSeconds              int64  `json:"ageSeconds"`
	BlocksCurrentOpenPeriod bool   `json:"blocksCurrentOpenPeriod"`
}

func (h *Handler) AdminSchemeRuntimeDiagnostics(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.db == nil {
		apix.Fail(w, http.StatusServiceUnavailable, apix.CodeInternal, "数据库未就绪")
		return
	}
	instanceID := strings.TrimSpace(r.PathValue("instanceId"))
	if instanceID == "" {
		apix.Validation(w, "instanceId 不能为空")
		return
	}

	var inst adminSchemeRuntimeInstance
	err := h.db.QueryRow(r.Context(), `
SELECT id, definition_id, scheme_name, lottery_code, status, status_reason,
       COALESCE(last_settled_issue, ''), COALESCE(start_skip_period, '')
FROM scheme_instances
WHERE id = $1`, instanceID).Scan(
		&inst.ID, &inst.DefinitionID, &inst.SchemeName, &inst.LotteryCode,
		&inst.Status, &inst.StatusReason, &inst.LastSettledIssue, &inst.StartSkipPeriod,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			apix.Fail(w, http.StatusNotFound, apix.CodeNotFound, "方案实例不存在")
			return
		}
		apix.Internal(w)
		return
	}

	runtime := lottery.InspectPeriodRuntime(inst.LotteryCode, time.Now())
	expectedPreviousIssue := previousIssueForRuntime(runtime.CurrentOpenPeriod)
	previousDraw, err := readAdminRuntimeDraw(r.Context(), h.db, inst.LotteryCode, expectedPreviousIssue)
	if err != nil {
		apix.Internal(w)
		return
	}
	latestDraw, err := readAdminRuntimeLatestDraw(r.Context(), h.db, inst.LotteryCode)
	if err != nil {
		apix.Internal(w)
		return
	}
	acceptedPending, err := readAdminAcceptedPendingBets(r.Context(), h.db, inst.ID, runtime.CurrentOpenPeriod)
	if err != nil {
		apix.Internal(w)
		return
	}

	apix.OK(w, map[string]any{
		"instance":              inst,
		"periods":               runtime,
		"latestDraw":            latestDraw,
		"expectedPreviousIssue": expectedPreviousIssue,
		"previousDraw":          previousDraw,
		"acceptedPending":       acceptedPending,
		"blockReason":           schemeRuntimeBlockReason(inst.Status, runtime, expectedPreviousIssue != "", previousDraw != nil),
	})
}

// acceptedPendingBlocksCurrentPeriod keeps completed upstream orders for
// settlement recovery, but only lets ambiguous, same, or future orders block
// the current outbound period.
func acceptedPendingBlocksCurrentPeriod(currentOpenPeriod, thirdPartyPeriod string) bool {
	currentOpenPeriod = strings.TrimSpace(currentOpenPeriod)
	thirdPartyPeriod = strings.TrimSpace(thirdPartyPeriod)
	if currentOpenPeriod == "" || thirdPartyPeriod == "" {
		return true
	}
	current, currentErr := strconv.ParseInt(currentOpenPeriod, 10, 64)
	thirdParty, thirdPartyErr := strconv.ParseInt(thirdPartyPeriod, 10, 64)
	if currentErr == nil && thirdPartyErr == nil {
		return thirdParty >= current
	}
	return thirdPartyPeriod >= currentOpenPeriod
}

// schemeRuntimeBlockReason reports only locally observable preflight blockers.
func schemeRuntimeBlockReason(status string, runtime lottery.PeriodRuntimeDiagnostics, needsDraw, drawPresent bool) string {
	if strings.TrimSpace(status) != "running" {
		return "scheme_not_running"
	}
	if !runtime.HasPeriodsSnapshot {
		return "periods_snapshot_missing"
	}
	if !runtime.PeriodsFresh {
		return "periods_snapshot_stale"
	}
	if !runtime.BetWindowOpen {
		return "bet_window_closed"
	}
	if needsDraw && !drawPresent {
		return "previous_draw_missing"
	}
	return "no_local_preflight_block"
}

func previousIssueForRuntime(issue string) string {
	issue = strings.TrimSpace(issue)
	if issue == "" {
		return ""
	}
	n, err := strconv.ParseInt(issue, 10, 64)
	if err != nil || n <= 0 {
		return ""
	}
	return strconv.FormatInt(n-1, 10)
}

type adminRuntimeQueryRower interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

type adminRuntimeQuerier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

func readAdminAcceptedPendingBets(ctx context.Context, db adminRuntimeQuerier, instanceID, currentOpenPeriod string) ([]adminSchemeAcceptedPendingBet, error) {
	rows, err := db.Query(ctx, `
SELECT c.record_no,
       c.third_party_bet_id,
       COALESCE(c.period_no, ''),
       COALESCE(NULLIF(TRIM(c.third_party_period), ''), ''),
       c.placed_at::text,
       GREATEST(0, EXTRACT(EPOCH FROM now() - c.placed_at))::bigint
FROM cloud_bet_records c
WHERE c.scheme_id = $1
  AND c.status = 'pending'
  AND c.sim_bet = FALSE
  AND NULLIF(TRIM(c.third_party_bet_id), '') IS NOT NULL
ORDER BY c.placed_at ASC, c.id ASC
LIMIT 20`, instanceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]adminSchemeAcceptedPendingBet, 0)
	for rows.Next() {
		var item adminSchemeAcceptedPendingBet
		if err := rows.Scan(
			&item.RecordNo,
			&item.ThirdPartyBetID,
			&item.PeriodNo,
			&item.ThirdPartyPeriod,
			&item.PlacedAt,
			&item.AgeSeconds,
		); err != nil {
			return nil, err
		}
		item.BlocksCurrentOpenPeriod = acceptedPendingBlocksCurrentPeriod(currentOpenPeriod, item.ThirdPartyPeriod)
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

func readAdminRuntimeDraw(ctx context.Context, db adminRuntimeQueryRower, lotteryCode, issueNo string) (*adminSchemeRuntimeDraw, error) {
	if strings.TrimSpace(issueNo) == "" {
		return nil, nil
	}
	var out adminSchemeRuntimeDraw
	var ballsRaw []byte
	err := db.QueryRow(ctx, `
SELECT issue_no, balls, drawn_at::text
FROM lottery_draws
WHERE lottery_code = $1 AND issue_no = $2`, lotteryCode, issueNo).Scan(&out.IssueNo, &ballsRaw, &out.DrawnAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(ballsRaw, &out.Balls); err != nil {
		return nil, err
	}
	return &out, nil
}

func readAdminRuntimeLatestDraw(ctx context.Context, db adminRuntimeQueryRower, lotteryCode string) (*adminSchemeRuntimeDraw, error) {
	var out adminSchemeRuntimeDraw
	var ballsRaw []byte
	err := db.QueryRow(ctx, `
SELECT issue_no, balls, drawn_at::text
FROM lottery_draws
WHERE lottery_code = $1
ORDER BY drawn_at DESC, id DESC
LIMIT 1`, lotteryCode).Scan(&out.IssueNo, &ballsRaw, &out.DrawnAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(ballsRaw, &out.Balls); err != nil {
		return nil, err
	}
	return &out, nil
}
