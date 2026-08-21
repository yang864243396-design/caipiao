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
	"caipiao/backend/internal/guaji"
	"caipiao/backend/internal/guaji/accountsvc"
	"caipiao/backend/internal/guaji/periodsync"
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
	GuajiAccountID   int64  `json:"-"`
	ChainBlockReason string `json:"-"`
}

// SchemeRuntimeDiagnosticsProvider returns local snapshots only. Its methods
// must not initiate provider traffic or mutate runtime state.
type SchemeRuntimeDiagnosticsProvider interface {
	DrawWSHealth() guaji.DrawWSHealthSnapshot
	PeriodBoundaryHealth(string, time.Time) guaji.LotteryBoundaryHealthSnapshot
}

type adminRuntimeDecision struct {
	SourcePeriod     *string
	TargetPeriod     *string
	Status           *string
	FailureReason    *string
	TargetDeadlineAt *time.Time
}

type adminRuntimeOutbox struct {
	State         *string `json:"state"`
	OutcomeReason *string `json:"outcomeReason"`
}

type adminRuntimeDrawWS struct {
	Connected   *bool      `json:"connected"`
	LastFrameAt *time.Time `json:"lastFrameAt"`
	LastPongAt  *time.Time `json:"lastPongAt"`
	Reconnects  *uint64    `json:"reconnects"`
	LastError   *string    `json:"lastError"`
}

type adminRuntimePeriodBoundary struct {
	CurrentIssue     *string    `json:"currentIssue"`
	NextIssue        *string    `json:"nextIssue"`
	ReceivedAt       *time.Time `json:"receivedAt"`
	WSRestLagPeriods *int       `json:"wsRestLagPeriods"`
	Stale            bool       `json:"-"`
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
       COALESCE(last_settled_issue, ''), COALESCE(start_skip_period, ''),
       COALESCE(
         (SELECT c.guaji_account_id
          FROM cloud_bet_records c
          WHERE c.scheme_id = si.id AND c.sim_bet = FALSE AND c.guaji_account_id IS NOT NULL
          ORDER BY c.placed_at DESC, c.id DESC LIMIT 1),
         (SELECT a.id
          FROM member_guaji_accounts a
          WHERE a.member_id = si.member_id AND a.is_active = TRUE
          ORDER BY a.bound_at DESC LIMIT 1),
         0
       ),
       COALESCE(chain_block_reason, '')
FROM scheme_instances si
WHERE si.id = $1`, instanceID).Scan(
		&inst.ID, &inst.DefinitionID, &inst.SchemeName, &inst.LotteryCode,
		&inst.Status, &inst.StatusReason, &inst.LastSettledIssue, &inst.StartSkipPeriod, &inst.GuajiAccountID, &inst.ChainBlockReason,
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
	refreshDiagnostics, _ := periodsync.DiagnosticsForLottery(inst.LotteryCode)
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
	var payoutRead func(int64) (accountsvc.PayoutSyncDiagnostics, bool)
	if h.guajiAccounts != nil {
		payoutRead = h.guajiAccounts.PayoutSyncDiagnostics
	}
	payoutSync := runtimePayoutDiagnostics(inst.GuajiAccountID, payoutRead)
	decision, err := readAdminRuntimeLatestDecision(r.Context(), h.db, inst.ID)
	if err != nil {
		apix.Internal(w)
		return
	}
	outbox, err := readAdminRuntimeLatestOutbox(r.Context(), h.db, inst.ID)
	if err != nil {
		apix.Internal(w)
		return
	}
	if outbox == nil {
		outbox = &adminRuntimeOutbox{}
	}
	drawWS, boundary := runtimeDrawHealthSnapshots(h.schemeRuntimeDiagnostics, inst.LotteryCode, time.Now())
	preflight := schemeRuntimePreflightBlockReason(inst.Status, runtime, expectedPreviousIssue != "", previousDraw != nil)
	blockReason := schemeRuntimeBlockReason(runtimeBlockInput{
		ProviderAcceptedWrongPeriod: (outbox != nil && outbox.State != nil && *outbox.State == "accepted_wrong_period") || inst.ChainBlockReason == "provider_accepted_wrong_period",
		ProviderAcceptanceUnknown:   (outbox != nil && outbox.State != nil && (*outbox.State == "sent_unknown" || *outbox.State == "external_acceptance_unknown")) || inst.ChainBlockReason == "provider_acceptance_unknown",
		ChainBlockReason:            inst.ChainBlockReason,
		DecisionStatus:              runtimeDecisionStatusValue(decision),
		StrategyEvaluationFailed:    strings.TrimSpace(inst.StatusReason) == "strategy_evaluation_failed" || runtimeDecisionFailureReason(decision) == "strategy_evaluation_failed",
		DrawMissing:                 expectedPreviousIssue != "" && previousDraw == nil,
		DrawWSStale:                 boundary != nil && boundary.Stale,
		AwaitingTarget:              decision != nil && decision.Status != nil && *decision.Status == "awaiting_target",
		DeadlineExpired:             runtimeDecisionDeadlineExpired(decision, time.Now()),
		PreflightReason:             preflight,
	})

	apix.OK(w, map[string]any{
		"instance":              inst,
		"periods":               runtime,
		"periodRefresh":         refreshDiagnostics,
		"latestDraw":            latestDraw,
		"expectedPreviousIssue": expectedPreviousIssue,
		"previousDraw":          previousDraw,
		"acceptedPending":       acceptedPending,
		"payoutSync":            payoutSync,
		"sourcePeriod":          runtimeDecisionSourcePeriod(decision),
		"targetPeriod":          runtimeDecisionTargetPeriod(decision),
		"decisionStatus":        runtimeDecisionStatus(decision),
		"decisionFailureReason": runtimeDecisionFailure(decision),
		"targetDeadlineAt":      runtimeDecisionDeadline(decision),
		"awaitingTarget":        decision != nil && decision.Status != nil && *decision.Status == "awaiting_target",
		"chainBlockReason":      nullableString(inst.ChainBlockReason),
		"outbox":                outbox,
		"drawWS":                drawWS,
		"periodBoundary":        boundary,
		"blockReason":           blockReason,
	})
}

func runtimePayoutDiagnostics(accountID int64, read func(int64) (accountsvc.PayoutSyncDiagnostics, bool)) *accountsvc.PayoutSyncDiagnostics {
	if accountID <= 0 || read == nil {
		return nil
	}
	snapshot, ok := read(accountID)
	if !ok {
		return nil
	}
	return &snapshot
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

// schemeRuntimePreflightBlockReason reports only locally observable preflight blockers.
func schemeRuntimePreflightBlockReason(status string, runtime lottery.PeriodRuntimeDiagnostics, needsDraw, drawPresent bool) string {
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

func readAdminRuntimeLatestDecision(ctx context.Context, db adminRuntimeQueryRower, instanceID string) (*adminRuntimeDecision, error) {
	var sourcePeriod, targetPeriod, status, failureReason string
	var deadline *time.Time
	err := db.QueryRow(ctx, `
SELECT source_period_no, COALESCE(target_period_no, ''), status,
       COALESCE(failure_reason, ''), target_deadline_at
FROM scheme_period_decisions
WHERE scheme_id = $1
ORDER BY decided_at DESC, id DESC
LIMIT 1`, instanceID).Scan(&sourcePeriod, &targetPeriod, &status, &failureReason, &deadline)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &adminRuntimeDecision{
		SourcePeriod:     nullableString(sourcePeriod),
		TargetPeriod:     nullableString(targetPeriod),
		Status:           nullableString(status),
		FailureReason:    nullableString(failureReason),
		TargetDeadlineAt: deadline,
	}, nil
}

func readAdminRuntimeLatestOutbox(ctx context.Context, db adminRuntimeQueryRower, instanceID string) (*adminRuntimeOutbox, error) {
	var state, outcomeReason string
	err := db.QueryRow(ctx, `
SELECT state, COALESCE(outcome_reason, '')
FROM scheme_bet_outbox
WHERE scheme_id = $1
ORDER BY created_at DESC, id DESC
LIMIT 1`, instanceID).Scan(&state, &outcomeReason)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &adminRuntimeOutbox{State: nullableString(state), OutcomeReason: nullableString(outcomeReason)}, nil
}

func runtimeDrawHealthSnapshots(provider SchemeRuntimeDiagnosticsProvider, lotteryCode string, now time.Time) (*adminRuntimeDrawWS, *adminRuntimePeriodBoundary) {
	if provider == nil {
		return &adminRuntimeDrawWS{}, &adminRuntimePeriodBoundary{}
	}
	drawHealth := provider.DrawWSHealth()
	connected := drawHealth.Connected
	reconnects := drawHealth.Reconnects
	drawWS := &adminRuntimeDrawWS{
		Connected:   &connected,
		LastFrameAt: nullableTime(drawHealth.LastFrameAt),
		LastPongAt:  nullableTime(drawHealth.LastPongAt),
		Reconnects:  &reconnects,
		LastError:   nullableString(sanitizeDiagnosticString(drawHealth.LastError)),
	}
	boundaryHealth := provider.PeriodBoundaryHealth(lotteryCode, now)
	if strings.TrimSpace(boundaryHealth.CurrentIssue) == "" && strings.TrimSpace(boundaryHealth.NextIssue) == "" && boundaryHealth.LastReceivedMono.IsZero() {
		return drawWS, &adminRuntimePeriodBoundary{}
	}
	lag := boundaryHealth.WSRestLagPeriods
	return drawWS, &adminRuntimePeriodBoundary{
		CurrentIssue:     nullableString(boundaryHealth.CurrentIssue),
		NextIssue:        nullableString(boundaryHealth.NextIssue),
		ReceivedAt:       nullableTime(boundaryHealth.LastReceivedMono),
		WSRestLagPeriods: &lag,
		Stale:            boundaryHealth.Stale,
	}
}

func nullableString(value string) *string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return &value
}

func nullableTime(value time.Time) *time.Time {
	if value.IsZero() {
		return nil
	}
	return &value
}

func runtimeDecisionSourcePeriod(decision *adminRuntimeDecision) *string {
	if decision == nil {
		return nil
	}
	return decision.SourcePeriod
}

func runtimeDecisionTargetPeriod(decision *adminRuntimeDecision) *string {
	if decision == nil {
		return nil
	}
	return decision.TargetPeriod
}

func runtimeDecisionStatus(decision *adminRuntimeDecision) *string {
	if decision == nil || decision.Status == nil {
		return nil
	}
	return decision.Status
}

func runtimeDecisionStatusValue(decision *adminRuntimeDecision) string {
	if status := runtimeDecisionStatus(decision); status != nil {
		return *status
	}
	return ""
}

func runtimeDecisionFailure(decision *adminRuntimeDecision) *string {
	if decision == nil {
		return nil
	}
	return decision.FailureReason
}

func runtimeDecisionFailureReason(decision *adminRuntimeDecision) string {
	if failure := runtimeDecisionFailure(decision); failure != nil {
		return *failure
	}
	return ""
}

func runtimeDecisionDeadline(decision *adminRuntimeDecision) *time.Time {
	if decision == nil {
		return nil
	}
	return decision.TargetDeadlineAt
}

func runtimeDecisionDeadlineExpired(decision *adminRuntimeDecision, now time.Time) bool {
	return decision != nil && decision.TargetDeadlineAt != nil && !now.Before(*decision.TargetDeadlineAt)
}

type runtimeBlockInput struct {
	ProviderAcceptedWrongPeriod bool
	ProviderAcceptanceUnknown   bool
	ChainBlockReason            string
	DecisionStatus              string
	StrategyEvaluationFailed    bool
	DrawMissing                 bool
	DrawWSStale                 bool
	AwaitingTarget              bool
	DeadlineExpired             bool
	PreflightReason             string
}

func schemeRuntimeBlockReason(input runtimeBlockInput) string {
	switch {
	case input.ProviderAcceptedWrongPeriod:
		return "provider_accepted_wrong_period"
	case input.ProviderAcceptanceUnknown:
		return "provider_acceptance_unknown"
	case input.ChainBlockReason == "missed_contiguous_period" || input.DecisionStatus == "missed_contiguous_period":
		return "missed_contiguous_period"
	case input.StrategyEvaluationFailed:
		return "strategy_evaluation_failed"
	case input.DrawMissing:
		return "draw_missing"
	case input.DrawWSStale:
		return "draw_ws_stale"
	case input.AwaitingTarget && !input.DeadlineExpired:
		return "next_period_unavailable"
	case strings.TrimSpace(input.PreflightReason) != "":
		return input.PreflightReason
	default:
		return "no_local_preflight_block"
	}
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
