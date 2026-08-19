package accountsvc

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"caipiao/backend/internal/cloud/schemestate"
	"caipiao/backend/internal/cloudlimits"
	"caipiao/backend/internal/db/sqlcdb"
	"caipiao/backend/internal/guaji"
	"caipiao/backend/internal/member"
	"caipiao/backend/internal/schemelimits"
	"caipiao/backend/internal/ws"
)

const (
	payoutSyncBatch                = 50
	payoutSyncPageSize             = 50
	payoutSyncRecentPageBudget     = 3
	historicalSettlementFirstPage  = 4
	historicalSettlementPageBudget = 3
	payoutHistoricalRowsPerTick    = 3
)

// LocalDrawSettlement 本地开奖表派奖评估结果（保留类型供历史调用方编译；派奖同步不再使用本地金额）。
type LocalDrawSettlement struct {
	Status string
	Pnl    float64
	Payout float64
}

// LocalDrawFallback 已废弃：真实注单返奖必须以第三方 web_bets 毛派奖为准，禁止本地金额兜底。
type LocalDrawFallback func(ctx context.Context, orderID int64, orderNo string) (LocalDrawSettlement, bool, error)

// AfterSettleFn 派奖完成后触发（如补齐 lottery_draws 历史）。
type AfterSettleFn func(ctx context.Context, lotteryCode, issueNo string)

// PayoutSyncWorker 扫描 real 第三方 pending 注单，查第三方结算结果，
// 以第三方毛派奖为准结算 bet_orders + 镜像 wallet_ledger + 余额刷新（T5）。
//
// 与挂机方案 Worker（事后本地模拟）正交：本 worker 处理 guaji_account_id 非空的
// 真实第三方注单。查不到 web_bets 时不下本地金额，详情返奖保持 —。
type PayoutSyncWorker struct {
	svc         *Service
	q           *sqlcdb.Queries
	hub         *ws.Hub
	afterSettle AfterSettleFn
}

// SetAfterSettle 注册派奖完成后的回调（如玩法详情开奖补齐）。
func (w *PayoutSyncWorker) SetAfterSettle(fn AfterSettleFn) {
	if w == nil {
		return
	}
	w.afterSettle = fn
}

// NewPayoutSyncWorker 仅在第三方启用时创建。fallback 参数已忽略（兼容旧调用方）。
func (s *Service) NewPayoutSyncWorker(hub *ws.Hub, _ LocalDrawFallback) *PayoutSyncWorker {
	if s == nil || s.pool == nil || s.guaji == nil || !s.guaji.Enabled() {
		return nil
	}
	return &PayoutSyncWorker{svc: s, q: sqlcdb.New(s.pool), hub: hub}
}

func (w *PayoutSyncWorker) Run(ctx context.Context, interval time.Duration) {
	if w == nil {
		return
	}
	if interval <= 0 {
		interval = 10 * time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.tick(ctx)
		}
	}
}

func (w *PayoutSyncWorker) tick(ctx context.Context) {
	seen := make(map[int64]struct{}, payoutSyncBatch)
	pending := make([]sqlcdb.ListPendingGuajiBetOrdersRow, 0, payoutSyncBatch*2)
	if rows, err := w.listPendingForRunningSchemes(ctx, payoutSyncBatch); err != nil {
		slog.Warn("payout sync running-scheme list failed", "err", err)
	} else {
		for _, row := range rows {
			seen[row.ID] = struct{}{}
			pending = append(pending, row)
		}
	}
	rows, err := w.q.ListPendingGuajiBetOrders(ctx, payoutSyncBatch)
	if err != nil {
		slog.Warn("payout sync list failed", "err", err)
		return
	}
	for _, row := range rows {
		if _, ok := seen[row.ID]; ok {
			continue
		}
		pending = append(pending, row)
	}
	for _, batch := range groupPendingPayoutRows(pending) {
		if err := w.syncAccountPending(ctx, batch.accountID, batch.rows); err != nil {
			slog.Warn("payout sync account batch failed", "accountId", batch.accountID, "orders", len(batch.rows), "err", err)
		}
	}
}

type payoutAccountBatch struct {
	accountID int64
	rows      []sqlcdb.ListPendingGuajiBetOrdersRow
}

func groupPendingPayoutRows(rows []sqlcdb.ListPendingGuajiBetOrdersRow) []payoutAccountBatch {
	byAccount := make(map[int64]int)
	batches := make([]payoutAccountBatch, 0)
	for _, row := range rows {
		if !row.GuajiAccountID.Valid || row.GuajiAccountID.Int64 <= 0 || !row.ThirdPartyBetID.Valid || strings.TrimSpace(row.ThirdPartyBetID.String) == "" {
			continue
		}
		idx, found := byAccount[row.GuajiAccountID.Int64]
		if !found {
			idx = len(batches)
			byAccount[row.GuajiAccountID.Int64] = idx
			batches = append(batches, payoutAccountBatch{accountID: row.GuajiAccountID.Int64})
		}
		batches[idx].rows = append(batches[idx].rows, row)
	}
	return batches
}

func payoutBatchCounts(total, settled int) (int, int) {
	if total < 0 {
		total = 0
	}
	if settled < 0 {
		settled = 0
	}
	if settled > total {
		settled = total
	}
	return settled, total - settled
}

func settlePayoutBatchRows(
	rows []sqlcdb.ListPendingGuajiBetOrdersRow,
	itemsByID map[string]guaji.WebBetRecord,
	commitRecent func(sqlcdb.ListPendingGuajiBetOrdersRow, *guaji.BetSettlement) error,
	syncHistorical func(sqlcdb.ListPendingGuajiBetOrdersRow) (bool, error),
) (int, error) {
	return settlePayoutBatchRowsWithHistoricalBudget(rows, itemsByID, commitRecent, syncHistorical, -1)
}

func settlePayoutBatchRowsWithHistoricalBudget(
	rows []sqlcdb.ListPendingGuajiBetOrdersRow,
	itemsByID map[string]guaji.WebBetRecord,
	commitRecent func(sqlcdb.ListPendingGuajiBetOrdersRow, *guaji.BetSettlement) error,
	syncHistorical func(sqlcdb.ListPendingGuajiBetOrdersRow) (bool, error),
	historicalBudget int,
) (int, error) {
	settledCount := 0
	historicalUsed := 0
	var batchErr error
	for _, row := range rows {
		betID := strings.TrimSpace(row.ThirdPartyBetID.String)
		item, found := itemsByID[betID]
		if found {
			res := guaji.SettlementFromWebBet(&item)
			if res != nil && res.Settled {
				if err := commitRecent(row, res); err != nil {
					batchErr = appendPayoutBatchError(batchErr, err)
					continue
				}
				settledCount++
			}
			continue
		}
		if historicalBudget >= 0 && historicalUsed >= historicalBudget {
			continue
		}
		historicalUsed++
		settled, err := syncHistorical(row)
		if err != nil {
			batchErr = appendPayoutBatchError(batchErr, err)
			continue
		}
		if settled {
			settledCount++
		}
	}
	return settledCount, batchErr
}

func appendPayoutBatchError(current, next error) error {
	if next == nil {
		return current
	}
	if current == nil {
		return next
	}
	return errors.Join(current, next)
}

func settleAccountPayoutBatch(
	diagnostics *payoutDiagnosticStore,
	accountID int64,
	rows []sqlcdb.ListPendingGuajiBetOrdersRow,
	itemsByID map[string]guaji.WebBetRecord,
	commitRecent func(sqlcdb.ListPendingGuajiBetOrdersRow, *guaji.BetSettlement) error,
	syncHistorical func(sqlcdb.ListPendingGuajiBetOrdersRow) (bool, error),
	at time.Time,
) error {
	settledCount, err := settlePayoutBatchRowsWithHistoricalBudget(
		rows, itemsByID, commitRecent, syncHistorical, payoutHistoricalRowsPerTick,
	)
	if err != nil {
		settledCount, unresolvedCount := payoutBatchCounts(len(rows), settledCount)
		diagnostics.partial(accountID, len(itemsByID), settledCount, unresolvedCount, err, at)
		return err
	}
	settledCount, unresolvedCount := payoutBatchCounts(len(rows), settledCount)
	diagnostics.succeed(accountID, len(itemsByID), settledCount, unresolvedCount, at)
	return nil
}

func resolveHistoricalSettlementResult(
	res *guaji.BetSettlement,
	nextPage int,
	exhausted bool,
	providerErr error,
	saveCursor func(page int, lastErr string) error,
	commit func(*guaji.BetSettlement) error,
	clearCursor func() error,
) (bool, error) {
	if providerErr != nil {
		page := nextHistoricalSettlementPage(historicalSettlementFirstPage, nextPage, exhausted)
		if cursorErr := saveCursor(page, providerErr.Error()); cursorErr != nil {
			return false, errors.Join(providerErr, fmt.Errorf("persist historical settlement cursor: %w", cursorErr))
		}
		return false, providerErr
	}
	if res == nil || !res.Settled {
		return false, saveCursor(historicalSettlementFirstPage, "")
	}
	if err := commit(res); err != nil {
		return false, err
	}
	if err := clearCursor(); err != nil {
		return false, err
	}
	return true, nil
}

type webBetPageFetcher func(ctx context.Context, limit, page int) ([]guaji.WebBetRecord, error)

// fetchRecentAccountSettlements reads the provider's recent range once for an
// account. A full page may have older pending orders immediately behind it, so
// only full pages cause a bounded follow-up request.
func fetchRecentAccountSettlements(ctx context.Context, fetch webBetPageFetcher) (map[string]guaji.WebBetRecord, error) {
	itemsByID := make(map[string]guaji.WebBetRecord)
	if fetch == nil {
		return itemsByID, nil
	}
	for page := 1; page <= payoutSyncRecentPageBudget; page++ {
		items, err := fetch(ctx, payoutSyncPageSize, page)
		if err != nil {
			return nil, err
		}
		for _, item := range items {
			if item.ID > 0 {
				itemsByID[strconv.FormatInt(item.ID, 10)] = item
			}
		}
		if len(items) < payoutSyncPageSize {
			break
		}
	}
	return itemsByID, nil
}

// syncAccountPending batches recent provider settlement reads for a single
// account. Only records outside that recent range use their existing bounded
// historical recovery cursor, so multiple local pending orders never repeat
// the same recent third-party list scan.
func (w *PayoutSyncWorker) syncAccountPending(ctx context.Context, accountID int64, rows []sqlcdb.ListPendingGuajiBetOrdersRow) error {
	if w == nil || w.svc == nil || accountID <= 0 || len(rows) == 0 {
		return nil
	}
	w.svc.payoutDiagnostics.begin(accountID, len(rows), time.Now())
	acc, err := w.svc.getRowByIDAny(ctx, accountID)
	if err != nil {
		return err
	}
	if !w.svc.tokenHealthy(acc) {
		return nil
	}
	token, err := guaji.DecryptSecret(w.svc.credKey, acc.accessTokenEnc.String)
	if err != nil {
		return err
	}
	itemsByID, err := fetchRecentAccountSettlements(ctx, func(ctx context.Context, limit, page int) ([]guaji.WebBetRecord, error) {
		return w.svc.guaji.ListWebBets(ctx, token, limit, page)
	})
	if err != nil {
		w.svc.payoutDiagnostics.fail(accountID, err, time.Now())
		return err // keep all pending for the next provider retry and let tick log the failure
	}
	err = settleAccountPayoutBatch(
		w.svc.payoutDiagnostics,
		accountID,
		rows,
		itemsByID,
		func(row sqlcdb.ListPendingGuajiBetOrdersRow, res *guaji.BetSettlement) error {
			return w.commitResolvedSettlement(ctx, row, res, token)
		},
		func(row sqlcdb.ListPendingGuajiBetOrdersRow) (bool, error) {
			return w.syncHistoricalOne(ctx, row, token)
		},
		time.Now(),
	)
	if err != nil {
		return err
	}
	return nil
}

func (w *PayoutSyncWorker) listPendingForRunningSchemes(ctx context.Context, limit int) ([]sqlcdb.ListPendingGuajiBetOrdersRow, error) {
	if w == nil || w.svc == nil || w.svc.pool == nil {
		return nil, nil
	}
	rows, err := w.svc.pool.Query(ctx, `
SELECT b.id, b.order_no, b.member_id, b.guaji_account_id, b.third_party_bet_id,
       b.amount::float8, COALESCE(b.currency, '')
FROM bet_orders b
JOIN cloud_bet_records c ON c.bet_order_no = b.order_no
JOIN scheme_instances si ON si.id = c.scheme_id AND si.status = 'running'
WHERE b.status = 'pending'
  AND b.guaji_account_id IS NOT NULL
  AND b.third_party_bet_id IS NOT NULL
ORDER BY b.placed_at ASC, b.id ASC
LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []sqlcdb.ListPendingGuajiBetOrdersRow
	for rows.Next() {
		var row sqlcdb.ListPendingGuajiBetOrdersRow
		if err := rows.Scan(&row.ID, &row.OrderNo, &row.MemberID, &row.GuajiAccountID, &row.ThirdPartyBetID, &row.Amount, &row.Currency); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func (w *PayoutSyncWorker) syncOne(ctx context.Context, row sqlcdb.ListPendingGuajiBetOrdersRow) error {
	if !row.GuajiAccountID.Valid {
		return nil
	}
	betID := ""
	if row.ThirdPartyBetID.Valid {
		betID = strings.TrimSpace(row.ThirdPartyBetID.String)
	}
	if betID == "" {
		return nil
	}
	acc, err := w.svc.getRowByIDAny(ctx, row.GuajiAccountID.Int64)
	if err != nil {
		return err
	}
	if !w.svc.tokenHealthy(acc) {
		return nil // Token 失效，跳过本轮（重新授权后再同步）
	}
	token, err := guaji.DecryptSecret(w.svc.credKey, acc.accessTokenEnc.String)
	if err != nil {
		return err
	}
	res, err := w.svc.guaji.QuerySettlement(ctx, token, betID)
	recoveryUsed := false
	if errors.Is(err, guaji.ErrWebBetNotFound) {
		startPage, cursorErr := w.historicalSettlementPage(ctx, row.GuajiAccountID.Int64, betID)
		if cursorErr != nil {
			return cursorErr
		}
		var nextPage int
		var exhausted bool
		res, nextPage, exhausted, err = w.svc.guaji.QuerySettlementFromPageRange(
			ctx, token, betID, startPage, historicalSettlementPageBudget,
		)
		recoveryUsed = true
		if err != nil {
			if recordErr := w.saveHistoricalSettlementPage(ctx, row.GuajiAccountID.Int64, betID,
				nextHistoricalSettlementPage(historicalSettlementFirstPage, nextPage, exhausted), err.Error()); recordErr != nil {
				return recordErr
			}
			return nil
		}
	}
	if err != nil {
		// Transport failures keep the pending financial record for the next retry.
		return nil
	}
	if res == nil || !res.Settled {
		if recoveryUsed {
			if err := w.saveHistoricalSettlementPage(ctx, row.GuajiAccountID.Int64, betID,
				historicalSettlementFirstPage, ""); err != nil {
				return err
			}
		}
		return nil // 第三方尚未结算（C17：一直等待开奖）
	}

	status := "lose"
	pnl := res.Pnl
	payout := res.Payout
	// 勿用「Payout>0」单独判赢：龙虎和局退本时常 payout=本金、pnl≈0，会误成「中」。
	// 嵌套小奖：pnl 显著为负但仍有派奖额 / 显式 win → 记 win；返奖金额一律用第三方 payout。
	switch {
	case pnl > 1e-6:
		status = "win"
	case res.Status == "win":
		status = "win"
	case payout > 1e-6 && pnl < -1e-6:
		status = "win"
	}
	// 防御：真未中却净额≈0（第三方漏 net）时按 −本金落库；和局退本 payout≈本金 不动。
	if status == "lose" && row.Amount > 1e-6 && math.Abs(pnl) < 0.01 && payout < 0.01 {
		pnl = -row.Amount
	}
	// 毛派奖严格以第三方 web_bets 为准，禁止本地 PrizeNet / 开奖评估覆盖。
	currency := row.Currency
	balanceSnapshot := 0.0
	if info, ierr := w.svc.guaji.UserInfo(ctx, token); ierr == nil {
		balanceSnapshot = info.BalanceByCurrency(currency)
		w.svc.persistGuajiBalances(ctx, row.GuajiAccountID.Int64, multiCurrencyFromInfo(info))
	}
	if err := w.commitSettlement(ctx, row, status, pnl, payout, currency, balanceSnapshot, true); err != nil {
		return err
	}
	if recoveryUsed {
		if err := w.clearHistoricalSettlementPage(ctx, row.GuajiAccountID.Int64, betID); err != nil {
			return err
		}
	}
	return nil
}

// syncHistoricalOne preserves the existing bounded recovery cursor after the
// account's recent list range has already been fetched by syncAccountPending.
func (w *PayoutSyncWorker) syncHistoricalOne(ctx context.Context, row sqlcdb.ListPendingGuajiBetOrdersRow, token string) (bool, error) {
	if !row.GuajiAccountID.Valid || !row.ThirdPartyBetID.Valid {
		return false, nil
	}
	betID := strings.TrimSpace(row.ThirdPartyBetID.String)
	if betID == "" {
		return false, nil
	}
	startPage, err := w.historicalSettlementPage(ctx, row.GuajiAccountID.Int64, betID)
	if err != nil {
		return false, err
	}
	res, nextPage, exhausted, err := w.svc.guaji.QuerySettlementFromPageRange(ctx, token, betID, startPage, historicalSettlementPageBudget)
	return resolveHistoricalSettlementResult(
		res,
		nextPage,
		exhausted,
		err,
		func(page int, lastErr string) error {
			return w.saveHistoricalSettlementPage(ctx, row.GuajiAccountID.Int64, betID, page, lastErr)
		},
		func(res *guaji.BetSettlement) error {
			return w.commitResolvedSettlement(ctx, row, res, token)
		},
		func() error {
			return w.clearHistoricalSettlementPage(ctx, row.GuajiAccountID.Int64, betID)
		},
	)
}

func (w *PayoutSyncWorker) commitResolvedSettlement(ctx context.Context, row sqlcdb.ListPendingGuajiBetOrdersRow, res *guaji.BetSettlement, token string) error {
	if res == nil || !res.Settled {
		return nil
	}
	status := "lose"
	pnl := res.Pnl
	payout := res.Payout
	switch {
	case pnl > 1e-6:
		status = "win"
	case res.Status == "win":
		status = "win"
	case payout > 1e-6 && pnl < -1e-6:
		status = "win"
	}
	if status == "lose" && row.Amount > 1e-6 && math.Abs(pnl) < 0.01 && payout < 0.01 {
		pnl = -row.Amount
	}
	currency := row.Currency
	balanceSnapshot := 0.0
	if info, ierr := w.svc.guaji.UserInfo(ctx, token); ierr == nil {
		balanceSnapshot = info.BalanceByCurrency(currency)
		w.svc.persistGuajiBalances(ctx, row.GuajiAccountID.Int64, multiCurrencyFromInfo(info))
	}
	return w.commitSettlement(ctx, row, status, pnl, payout, currency, balanceSnapshot, true)
}

func nextHistoricalSettlementPage(firstPage, nextPage int, exhausted bool) int {
	if firstPage < 1 {
		firstPage = historicalSettlementFirstPage
	}
	if exhausted || nextPage < firstPage {
		return firstPage
	}
	return nextPage
}

func (w *PayoutSyncWorker) historicalSettlementPage(ctx context.Context, accountID int64, betID string) (int, error) {
	var page int
	err := w.svc.pool.QueryRow(ctx, `
SELECT next_page
FROM guaji_settlement_recovery
WHERE guaji_account_id = $1 AND third_party_bet_id = $2`, accountID, betID).Scan(&page)
	if errors.Is(err, pgx.ErrNoRows) {
		return historicalSettlementFirstPage, nil
	}
	if err != nil {
		return 0, err
	}
	return nextHistoricalSettlementPage(historicalSettlementFirstPage, page, false), nil
}

func (w *PayoutSyncWorker) saveHistoricalSettlementPage(ctx context.Context, accountID int64, betID string, nextPage int, lastErr string) error {
	_, err := w.svc.pool.Exec(ctx, `
INSERT INTO guaji_settlement_recovery (
    guaji_account_id, third_party_bet_id, next_page, last_error, last_attempt_at, updated_at
) VALUES ($1, $2, $3, $4, now(), now())
ON CONFLICT (guaji_account_id, third_party_bet_id) DO UPDATE
SET next_page = EXCLUDED.next_page,
    last_error = EXCLUDED.last_error,
    last_attempt_at = EXCLUDED.last_attempt_at,
    updated_at = EXCLUDED.updated_at`,
		accountID, betID, nextHistoricalSettlementPage(historicalSettlementFirstPage, nextPage, false), strings.TrimSpace(lastErr))
	return err
}

func (w *PayoutSyncWorker) clearHistoricalSettlementPage(ctx context.Context, accountID int64, betID string) error {
	_, err := w.svc.pool.Exec(ctx, `
DELETE FROM guaji_settlement_recovery
WHERE guaji_account_id = $1 AND third_party_bet_id = $2`, accountID, betID)
	return err
}

func (w *PayoutSyncWorker) commitSettlement(
	ctx context.Context,
	row sqlcdb.ListPendingGuajiBetOrdersRow,
	status string,
	pnl, payout float64,
	currency string,
	balanceSnapshot float64,
	writePayout bool,
) error {
	tx, err := w.svc.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	qtx := w.q.WithTx(tx)

	n, err := qtx.SettleBetOrder(ctx, sqlcdb.SettleBetOrderParams{
		ID:     row.ID,
		Status: status,
		Pnl:    member.NumericFromFloat(pnl),
	})
	if err != nil {
		return err
	}
	if n == 0 {
		return nil // 已被结算
	}
	cloudStatus := "miss"
	if status == "win" {
		cloudStatus = "hit"
	}
	payoutAmount := pgtype.Numeric{}
	if writePayout {
		if status == "win" {
			// 仅落库第三方毛派奖；未回传/近零则保持 NULL，详情展示 —
			if payout > 1e-9 {
				payoutAmount = member.NumericFromFloat(payout)
			}
		} else {
			payoutAmount = member.NumericFromFloat(0)
		}
	}
	if _, err := qtx.UpdateCloudBetRecordFromSettlement(ctx, sqlcdb.UpdateCloudBetRecordFromSettlementParams{
		BetOrderNo:   pgtype.Text{String: row.OrderNo, Valid: row.OrderNo != ""},
		Status:       cloudStatus,
		Pnl:          member.NumericFromFloat(pnl),
		PayoutAmount: payoutAmount,
	}); err != nil {
		return err
	}
	if err := qtx.ApplySchemeStatsFromCloudBetSettlement(ctx, row.OrderNo, member.NumericFromFloat(pnl)); err != nil {
		return err
	}
	var afterCommitLimits func()
	if schemeID, err := qtx.GetCloudBetSchemeIDByOrderNo(ctx, row.OrderNo); err == nil && schemeID != "" {
		if inst, ierr := qtx.GetSchemeInstanceFull(ctx, schemeID); ierr == nil {
			periodNo, _ := qtx.GetCloudBetPeriodByOrderNo(ctx, row.OrderNo)
			if periodNo == "" {
				periodNo = strings.TrimSpace(inst.LastSettledIssue.String)
			}
			hit := status == "win"
			def, derr := qtx.GetSchemeDefinitionByID(ctx, inst.DefinitionID)
			if derr != nil {
				return derr
			}
			if strategyInput, found, serr := qtx.GetFormalCloudBetStrategyInputByOrderNo(ctx, row.OrderNo); serr != nil {
				return serr
			} else if found {
				if strategyInput.PeriodNo != "" {
					periodNo = strategyInput.PeriodNo
				}
				verdict := resolveFormalStrategyVerdict(formalStrategyInput{
					ProviderHit:      hit,
					Kind:             inst.Kind,
					DefinitionConfig: def.Config,
					RoundIndex:       int(inst.RoundIndex),
					LotteryCode:      strategyInput.LotteryCode,
					BetContent:       strategyInput.BetContent,
					Balls:            strategyInput.Balls,
					Snapshot:         strategyInput.RuleSnapshot,
				})
				if err := persistFormalStrategyVerdict(ctx, qtx, strategyInput, row.OrderNo, verdict); err != nil {
					return err
				}
				// The provider remains authoritative for status/pnl/payout above;
				// only the next strategy round uses a successfully frozen local rule.
				hit = verdict.Hit
			}
			strategyAdvanced := false
			if evaluation, eerr := qtx.GetSchemeStrategyEvaluation(ctx, sqlcdb.GetSchemeStrategyEvaluationParams{InstanceID: inst.ID, PeriodNo: periodNo}); eerr == nil {
				strategyAdvanced = evaluation.Status == "completed" || evaluation.Status == "mismatch"
			}
			if !strategyAdvanced {
				if lerr := schemestate.ProcessFormalAfterSettlement(ctx, qtx, inst, periodNo, pnl, hit, def.Config, member.NumericFromFloat); lerr != nil {
					return lerr
				}
			} else {
				if lerr := schemestate.ProcessFormalFinancialAfterSettlement(ctx, qtx, inst, periodNo, pnl, hit, def.Config, member.NumericFromFloat); lerr != nil {
					return lerr
				}
			}
			memberID := inst.MemberID
			definitionID := inst.DefinitionID
			instStatus := inst.Status
			afterCommitLimits = func() {
				fresh, ferr := w.q.GetSchemeInstanceByID(ctx, schemeID)
				if ferr != nil {
					return
				}
				if instStatus == "running" && fresh.Status == "running" {
					if def, derr := w.q.GetSchemeDefinitionByID(ctx, definitionID); derr == nil {
						schemelimits.PauseRunningInstanceIfHit(ctx, w.q, w.hub, sqlcdb.SchemeInstanceFromAdminRow(fresh), def.Config)
					}
				}
				cloudlimits.PauseAllRunningIfHit(ctx, w.q, w.hub, memberID)
			}
		}
	}
	if payout > 0 {
		if err := member.MirrorRealLedger(ctx, qtx, row.MemberID, row.OrderNo, "payout", payout, balanceSnapshot, row.GuajiAccountID.Int64, currency); err != nil {
			return err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}
	if afterCommitLimits != nil {
		afterCommitLimits()
	}

	if w.hub != nil {
		var acct string
		if qerr := w.svc.pool.QueryRow(ctx, `SELECT account FROM members WHERE id = $1`, row.MemberID).Scan(&acct); qerr == nil && acct != "" {
			ws.PublishWallet(w.hub, acct, ws.WalletUpdatedPayload{
				Available: balanceSnapshot,
				Currency:  currency,
				Reason:    "guaji_payout",
			})
		}
	}
	w.notifyAfterSettle(ctx, row.OrderNo)
	return nil
}

func (w *PayoutSyncWorker) notifyAfterSettle(ctx context.Context, orderNo string) {
	if w == nil || w.afterSettle == nil || w.svc == nil || w.svc.pool == nil {
		return
	}
	orderNo = strings.TrimSpace(orderNo)
	if orderNo == "" {
		return
	}
	var lotteryCode, issueNo string
	err := w.svc.pool.QueryRow(ctx, `
SELECT lottery_code, issue_no FROM bet_orders WHERE order_no = $1`, orderNo).Scan(&lotteryCode, &issueNo)
	if err != nil {
		return
	}
	lotteryCode = strings.TrimSpace(lotteryCode)
	issueNo = strings.TrimSpace(issueNo)
	if lotteryCode == "" {
		return
	}
	w.afterSettle(ctx, lotteryCode, issueNo)
}

// SyncOne 对外暴露单笔派奖同步（诊断/手动补同步用）。
func (w *PayoutSyncWorker) SyncOne(ctx context.Context, row sqlcdb.ListPendingGuajiBetOrdersRow) error {
	return w.syncOne(ctx, row)
}

// LoadPendingGuajiBetOrder 按 order_no 加载 pending 第三方注单（手动补同步用）。
func (s *Service) LoadPendingGuajiBetOrder(ctx context.Context, orderNo string) (sqlcdb.ListPendingGuajiBetOrdersRow, error) {
	orderNo = strings.TrimSpace(orderNo)
	var row sqlcdb.ListPendingGuajiBetOrdersRow
	err := s.pool.QueryRow(ctx, `
SELECT b.id, b.order_no, b.member_id, b.guaji_account_id, b.third_party_bet_id,
       b.amount::float8, COALESCE(b.currency, '')
FROM bet_orders b
WHERE b.order_no = $1
  AND b.status = 'pending'
  AND b.guaji_account_id IS NOT NULL
  AND b.third_party_bet_id IS NOT NULL`, orderNo).Scan(
		&row.ID, &row.OrderNo, &row.MemberID, &row.GuajiAccountID, &row.ThirdPartyBetID, &row.Amount, &row.Currency,
	)
	return row, err
}
