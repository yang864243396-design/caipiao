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

	apix.OK(w, map[string]any{
		"instance":              inst,
		"periods":               runtime,
		"latestDraw":            latestDraw,
		"expectedPreviousIssue": expectedPreviousIssue,
		"previousDraw":          previousDraw,
		"blockReason":           schemeRuntimeBlockReason(inst.Status, runtime, expectedPreviousIssue != "", previousDraw != nil),
	})
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
