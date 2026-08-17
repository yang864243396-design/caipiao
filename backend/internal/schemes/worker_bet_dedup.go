package schemes

import (
	"context"
	"log/slog"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5/pgtype"

	"caipiao/backend/internal/db/sqlcdb"
	"caipiao/backend/internal/lottery"
)

// betPeriodDedup 防重：我方已下注第三方 periods vs 第三方当前开放 periods（模拟/真实共用）。
type betPeriodDedup struct {
	Skip        bool
	CurrentOpen string // 第三方 periods API 当前开放期
	LastBet     string // 本方案最近一次下注的第三方 periods
	Reason      string
}

// thirdPartyOpenPeriod 第三方当前可投期号（仅 periods API，禁止 WS）。
func thirdPartyOpenPeriod(lotteryCode string) (string, bool) {
	return lottery.OpenIssueForGuajiBet(lotteryCode)
}

// guajiBetPeriodMatches 下单请求期号须与第三方 periods API 当前开放期一致。
func guajiBetPeriodMatches(lotteryCode, periodNo string) bool {
	openIssue, ok := thirdPartyOpenPeriod(lotteryCode)
	if !ok {
		return false
	}
	if openIssue == "" && lottery.GuajiPeriodsNotProvided(lotteryCode) {
		return true
	}
	return strings.TrimSpace(openIssue) == strings.TrimSpace(periodNo)
}

// evaluateGuajiBetDedup 核心防重：已下注第三方期号 == 第三方当前开放期号 → 跳过（含待开奖）。
func (w *Worker) evaluateGuajiBetDedup(
	ctx context.Context,
	q *sqlcdb.Queries,
	inst sqlcdb.SchemeInstance,
) (betPeriodDedup, error) {
	currentOpen, ok := thirdPartyOpenPeriod(inst.LotteryCode)
	if !ok {
		return betPeriodDedup{Skip: true, Reason: "third_party_periods_unavailable"}, nil
	}
	if currentOpen == "" && lottery.GuajiPeriodsNotProvided(inst.LotteryCode) {
		return betPeriodDedup{Skip: false, CurrentOpen: ""}, nil
	}

	lastBet, err := q.SchemeLastThirdPartyBetPeriod(ctx, inst.ID, inst.SimBet)
	if err != nil {
		return betPeriodDedup{}, err
	}
	if lastBet != "" && lastBet == currentOpen {
		return betPeriodDedup{
			Skip:        true,
			CurrentOpen: currentOpen,
			LastBet:     lastBet,
			Reason:      "same_third_party_period",
		}, nil
	}
	if inst.LastSettledIssue.Valid {
		cursor := strings.TrimSpace(inst.LastSettledIssue.String)
		if cursor != "" && cursor == currentOpen {
			if inst.StartSkipPeriod.Valid && strings.TrimSpace(inst.StartSkipPeriod.String) == cursor {
				return betPeriodDedup{
					Skip:        true,
					CurrentOpen: currentOpen,
					LastBet:     lastBet,
					Reason:      "start_skip_period",
				}, nil
			}
			return betPeriodDedup{
				Skip:        true,
				CurrentOpen: currentOpen,
				LastBet:     lastBet,
				Reason:      "period_cursor_taken",
			}, nil
		}
	}

	// 模拟/正式均按本方案 cloud 记录防重（一期一注）；同会员其它方案互不影响。
	taken, errTaken := q.CloudBetPeriodHandled(ctx, inst.ID, currentOpen)
	if errTaken != nil {
		return betPeriodDedup{}, errTaken
	}
	if taken {
		return betPeriodDedup{
			Skip:        true,
			CurrentOpen: currentOpen,
			LastBet:     lastBet,
			Reason:      "period_record_exists",
		}, nil
	}
	// 上笔第三方接单期仍 pending，但本地开放期已翻到下一期：再 Place 常会叠单到旧第三方期，
	// 投注记录按 third_party_period 展示会出现「同一期两条」。
	pending, err := q.ListUnsettledGuajiBets(ctx, inst.ID)
	if err != nil {
		return betPeriodDedup{}, err
	}
	for _, bet := range pending {
		if !bet.Accepted {
			continue
		}
		thirdPartyPeriod := strings.TrimSpace(bet.ThirdPartyPeriod)
		if thirdPartyPeriod == "" || thirdPartyPeriod == currentOpen || !issueAfter(currentOpen, thirdPartyPeriod) {
			return betPeriodDedup{
				Skip:        true,
				CurrentOpen: currentOpen,
				LastBet:     thirdPartyPeriod,
				Reason:      "prior_third_party_pending",
			}, nil
		}
	}
	return betPeriodDedup{CurrentOpen: currentOpen, LastBet: lastBet}, nil
}

func (w *Worker) syncPeriodBetCursor(ctx context.Context, q *sqlcdb.Queries, inst sqlcdb.SchemeInstance, thirdPartyPeriod string) {
	thirdPartyPeriod = strings.TrimSpace(thirdPartyPeriod)
	if q == nil || thirdPartyPeriod == "" {
		return
	}
	if inst.LastSettledIssue.Valid && strings.TrimSpace(inst.LastSettledIssue.String) == thirdPartyPeriod {
		return
	}
	if err := q.UpdateSchemeInstanceLastSettledIssue(ctx, inst.ID, thirdPartyPeriod); err != nil {
		slog.Debug("scheme worker sync third party period cursor failed", "id", inst.ID, "period", thirdPartyPeriod, "err", err)
	}
}

func (w *Worker) finalizeCloudBetAfterGuaji(
	ctx context.Context,
	inst sqlcdb.SchemeInstance,
	cfg parsedSchemeConfig,
	recordNo string,
	amount, mult float64,
	roundIdx int,
	betContent string,
	meta schemeGuajiBetMeta,
	fallbackBetUnits int,
	claimPeriod string, // 本地占位期号；必须写回占位行，禁止按接单期 upsert 覆盖其它期
) {
	// 优先写占位期：接单期错位时若按 meta.Periods upsert，会盖掉目标期已有注单并制造「一单两期」。
	periodNo := strings.TrimSpace(claimPeriod)
	if periodNo == "" {
		periodNo = strings.TrimSpace(meta.Periods)
	}
	if periodNo == "" {
		return
	}
	playType := strings.TrimSpace(meta.PlayType)
	if playType == "" {
		playType = cloudPlayTypeLabel(cfg.PlayTypeLabel, cfg.SubPlayLabel)
	}
	if playType == "" {
		playType = cfg.PlayTypeLabel
	}
	betUnits := meta.BetsNums
	if betUnits <= 0 {
		betUnits = fallbackBetUnits
	}
	tid := strings.TrimSpace(meta.ThirdPartyBetID)
	// 第三方单号已挂在其它期：只保留占位与接单期游标，不再把旧 tid 写到新占位。
	if tid != "" {
		if prev, ok, err := w.q.SchemePeriodForThirdPartyBetID(ctx, inst.ID, tid); err == nil && ok && prev != periodNo {
			slog.Error("finalize skip duplicate third-party bet id",
				"instanceId", inst.ID, "tid", tid, "prevPeriod", prev, "claimPeriod", periodNo)
			tid, _, amount = clearDuplicateThirdPartyBetReference(tid, meta.OrderNo, amount, true)
		}
	}
	guajiID := activeGuajiAccountIDForInst(ctx, w.q, inst)
	updated, err := w.q.FinalizeClaimedCloudBetRecordGuajiMeta(ctx, inst.ID, periodNo,
		pgtype.Text{String: tid, Valid: tid != ""},
		pgtype.Text{String: meta.OrderNo, Valid: tid != "" && meta.OrderNo != ""},
		guajiPeriodsPgtext(meta.Periods),
		numericFromFloat(0), "pending", numericFromFloat(amount), betUnits, playType, betContent)
	if err == nil && updated {
		w.syncPeriodBetCursor(ctx, w.q, inst, periodNo)
		return
	}
	if err != nil {
		slog.Warn("scheme worker finalize claimed cloud bet update failed", "id", inst.ID, "period", periodNo, "err", err)
	} else {
		slog.Error("scheme worker finalize claim missing after upstream accept", "id", inst.ID, "period", periodNo, "thirdPartyBetId", tid)
	}
	// A process may have lost its pre-request claim after the upstream accepted
	// the bet. Recover exactly once with the returned third-party id; this is not
	// an amount-based association.
	if err := w.q.InsertCloudBetRecordEx(ctx, sqlcdb.InsertCloudBetRecordExParams{
		RecordNo:         recordNo,
		MemberID:         inst.MemberID,
		SimBet:           inst.SimBet,
		SchemeID:         inst.ID,
		SchemeName:       inst.SchemeName,
		PeriodNo:         periodNo,
		PlayType:         playType,
		Multiplier:       strconv.Itoa(betMultipleAsInt(mult)),
		RoundLabel:       betRoundLabel(cfg, roundIdx, int(inst.PickIndex)),
		Amount:           numericFromFloat(amount),
		Pnl:              numericFromFloat(0),
		Status:           "pending",
		BetContent:       betContent,
		GuajiAccountID:   guajiID,
		ThirdPartyBetID:  pgtype.Text{String: tid, Valid: tid != ""},
		ThirdPartyPeriod: guajiPeriodsPgtext(meta.Periods),
		BetOrderNo:       pgtype.Text{String: meta.OrderNo, Valid: tid != "" && meta.OrderNo != ""},
		Currency:         cfg.Currency,
		LotteryCode:      inst.LotteryCode,
		LotteryLabel:     inst.LotteryLabel,
		DefinitionID:     inst.DefinitionID,
		BetUnits:         betUnits,
	}); err != nil {
		slog.Warn("scheme worker finalize cloud bet insert failed", "id", inst.ID, "period", periodNo, "err", err)
		return
	}
	// 游标推进本地占位期，挡住同开放期重投。
	w.syncPeriodBetCursor(ctx, w.q, inst, periodNo)
}

// clearDuplicateThirdPartyBetReference keeps the positive amount of the committed
// period claim. cloud_bet_records requires amount > 0, while the duplicate third
// party reference itself must not be associated with a second period.
func clearDuplicateThirdPartyBetReference(tid, orderNo string, amount float64, duplicate bool) (string, string, float64) {
	if !duplicate {
		return tid, orderNo, amount
	}
	return "", "", amount
}

func guajiPeriodsPgtext(periods string) pgtype.Text {
	p := strings.TrimSpace(periods)
	if p == "" {
		return pgtype.Text{}
	}
	return pgtype.Text{String: p, Valid: true}
}

// hasUnsettledGuajiBet keeps the duplicate guard scoped to the currently open
// third-party period. Accepted records for older, authoritative periods are
// recovered by payout synchronization and cannot freeze future periods.
func (w *Worker) hasUnsettledGuajiBet(ctx context.Context, inst sqlcdb.SchemeInstance) bool {
	if w == nil || w.q == nil || !requiresGuajiRealBet(inst) {
		return false
	}
	currentOpen, openOK := thirdPartyOpenPeriod(inst.LotteryCode)
	if !openOK || strings.TrimSpace(currentOpen) == "" {
		return true
	}
	currentOpen = strings.TrimSpace(currentOpen)
	if taken, err := w.q.CloudBetPeriodHandled(ctx, inst.ID, currentOpen); err == nil && taken {
		return true
	}
	pending, err := w.q.ListUnsettledGuajiBets(ctx, inst.ID)
	if err != nil {
		return true
	}
	for _, bet := range pending {
		if !bet.Accepted {
			continue
		}
		thirdPartyPeriod := strings.TrimSpace(bet.ThirdPartyPeriod)
		if thirdPartyPeriod == "" {
			return true
		}
		if thirdPartyPeriod == currentOpen || !issueAfter(currentOpen, thirdPartyPeriod) {
			return true
		}
	}
	return false
}
