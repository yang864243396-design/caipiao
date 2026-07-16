package accountsvc

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"caipiao/backend/internal/cloud/schemestate"
	"caipiao/backend/internal/cloudlimits"
	"caipiao/backend/internal/db/sqlcdb"
	"caipiao/backend/internal/guaji"
	"caipiao/backend/internal/member"
	"caipiao/backend/internal/schemelimits"
	"caipiao/backend/internal/ws"
)

const payoutSyncBatch = 50

// LocalDrawSettlement 本地开奖表派奖评估结果。
type LocalDrawSettlement struct {
	Status string
	Pnl    float64
	Payout float64
}

// LocalDrawFallback 第三方注单查不到、但 lottery_draws 已有开奖时的回退评估。
type LocalDrawFallback func(ctx context.Context, orderID int64, orderNo string) (LocalDrawSettlement, bool, error)

// AfterSettleFn 派奖完成后触发（如补齐 lottery_draws 历史）。
type AfterSettleFn func(ctx context.Context, lotteryCode, issueNo string)

// PayoutSyncWorker 扫描 real 第三方 pending 注单，查第三方结算结果，
// 以第三方派奖为准结算 bet_orders + 镜像 wallet_ledger + 余额刷新（T5）。
//
// 与挂机方案 Worker（事后本地模拟）正交：本 worker 处理 guaji_account_id 非空的
// 真实第三方注单（手动下注 / 未来 real Worker 接单产生）。
type PayoutSyncWorker struct {
	svc               *Service
	q                 *sqlcdb.Queries
	hub               *ws.Hub
	localDrawFallback LocalDrawFallback
	afterSettle       AfterSettleFn
}

// SetAfterSettle 注册派奖完成后的回调（如玩法详情开奖补齐）。
func (w *PayoutSyncWorker) SetAfterSettle(fn AfterSettleFn) {
	if w == nil {
		return
	}
	w.afterSettle = fn
}

// NewPayoutSyncWorker 仅在第三方启用时创建。
func (s *Service) NewPayoutSyncWorker(hub *ws.Hub, fallback LocalDrawFallback) *PayoutSyncWorker {
	if s == nil || s.pool == nil || s.guaji == nil || !s.guaji.Enabled() {
		return nil
	}
	return &PayoutSyncWorker{svc: s, q: sqlcdb.New(s.pool), hub: hub, localDrawFallback: fallback}
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
	if rows, err := w.listPendingForRunningSchemes(ctx, payoutSyncBatch); err != nil {
		slog.Warn("payout sync running-scheme list failed", "err", err)
	} else {
		for _, row := range rows {
			seen[row.ID] = struct{}{}
			if err := w.syncOne(ctx, row); err != nil {
				slog.Warn("payout sync failed", "orderNo", row.OrderNo, "err", err)
			}
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
		if err := w.syncOne(ctx, row); err != nil {
			slog.Warn("payout sync failed", "orderNo", row.OrderNo, "err", err)
		}
	}
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
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			if settled, serr := w.trySettleFromLocalDraw(ctx, row); serr != nil {
				slog.Warn("payout sync local draw fallback failed", "orderNo", row.OrderNo, "err", serr)
			} else if settled {
				return nil
			}
		}
		return nil // 其它查询失败下轮重试
	}
	if res == nil || !res.Settled {
		return nil // 第三方尚未结算（C17：一直等待开奖）
	}

	status := "lose"
	pnl := res.Pnl
	payout := res.Payout
	// 勿用「Payout>0」单独判赢：龙虎和局退本时常 payout=本金、pnl≈0，会误成「中」。
	// 嵌套小奖：pnl 显著为负但仍有派奖额 / 显式 win → 记 win，再视情况用本地 PrizeNet。
	switch {
	case pnl > 1e-6:
		status = "win"
	case res.Status == "win":
		status = "win"
	case payout > 1e-6 && pnl < -1e-6:
		status = "win"
	}
	// 直选组合嵌套小奖：第三方整单净额常为「小奖−全票本金」（pnl≪0 / ≈0 仍标 win）→ 用本地 PrizeNet。
	// 绝不在 guaji 已有扎实正净额时用更大本地值覆盖（本地赔率常偏高：不定位/跨度/和值大小等）。
	if status == "win" {
		if eval, ok, lerr := w.evalLocalDraw(ctx, row); lerr != nil {
			slog.Warn("payout sync local prize eval failed", "orderNo", row.OrderNo, "err", lerr)
		} else if ok && eval.Status == "win" && eval.Pnl > 1 {
			// 仅当第三方净额近零/负值（字段缺失或嵌套淹没）时采用本地
			if pnl < 1.0 && eval.Pnl > pnl+0.5 {
				slog.Info("payout sync prefer local prize",
					"orderNo", row.OrderNo, "guajiPnl", pnl, "localPnl", eval.Pnl)
				pnl = eval.Pnl
				if eval.Payout > 0 {
					payout = eval.Payout
				}
			}
		}
	}
	// guaji 记挂但派奖额为 0：API 偶发漏字段。本地已判中且有派奖时补救（组合嵌套/任选）。
	// 勿在 guaji 已有正派奖却记挂时用本地硬改（避免「平台=中 第三方=挂」回潮）。
	if status == "lose" && payout < 1e-6 {
		if eval, ok, lerr := w.evalLocalDraw(ctx, row); lerr != nil {
			slog.Warn("payout sync local win check failed", "orderNo", row.OrderNo, "err", lerr)
		} else if ok && eval.Status == "win" && eval.Payout > 1e-6 {
			slog.Info("payout sync prefer local win (guaji miss payout)",
				"orderNo", row.OrderNo, "localPnl", eval.Pnl, "localPayout", eval.Payout)
			status = "win"
			pnl = eval.Pnl
			payout = eval.Payout
		}
	}
	currency := row.Currency
	balanceSnapshot := 0.0
	if info, ierr := w.svc.guaji.UserInfo(ctx, token); ierr == nil {
		balanceSnapshot = info.BalanceByCurrency(currency)
		w.svc.persistGuajiBalances(ctx, row.GuajiAccountID.Int64, multiCurrencyFromInfo(info))
	}
	return w.commitSettlement(ctx, row, status, pnl, payout, currency, balanceSnapshot)
}

func (w *PayoutSyncWorker) evalLocalDraw(ctx context.Context, row sqlcdb.ListPendingGuajiBetOrdersRow) (LocalDrawSettlement, bool, error) {
	if w == nil || w.localDrawFallback == nil {
		return LocalDrawSettlement{}, false, nil
	}
	return w.localDrawFallback(ctx, row.ID, row.OrderNo)
}

func (w *PayoutSyncWorker) trySettleFromLocalDraw(ctx context.Context, row sqlcdb.ListPendingGuajiBetOrdersRow) (bool, error) {
	if w.localDrawFallback == nil {
		return false, nil
	}
	eval, ok, err := w.localDrawFallback(ctx, row.ID, row.OrderNo)
	if err != nil || !ok {
		return false, err
	}
	slog.Info("payout sync local draw fallback",
		"orderNo", row.OrderNo, "status", eval.Status, "pnl", eval.Pnl)
	if err := w.commitSettlement(ctx, row, eval.Status, eval.Pnl, eval.Payout, row.Currency, 0); err != nil {
		return false, err
	}
	return true, nil
}

func (w *PayoutSyncWorker) commitSettlement(
	ctx context.Context,
	row sqlcdb.ListPendingGuajiBetOrdersRow,
	status string,
	pnl, payout float64,
	currency string,
	balanceSnapshot float64,
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
	if _, err := qtx.UpdateCloudBetRecordFromSettlement(ctx, sqlcdb.UpdateCloudBetRecordFromSettlementParams{
		BetOrderNo: pgtype.Text{String: row.OrderNo, Valid: row.OrderNo != ""},
		Status:     cloudStatus,
		Pnl:        member.NumericFromFloat(pnl),
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
			if lerr := schemestate.ProcessFormalAfterSettlement(ctx, qtx, inst, periodNo, pnl, hit, def.Config, member.NumericFromFloat); lerr != nil {
				return lerr
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
