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
	unsettled, hasPending, uerr := q.SchemeUnsettledGuajiPeriod(ctx, inst.ID)
	if uerr != nil {
		return betPeriodDedup{}, uerr
	}
	if hasPending {
		unsettled = strings.TrimSpace(unsettled)
		if unsettled != "" && unsettled != currentOpen {
			accepted, err := q.SchemeHasAcceptedUnsettledGuajiBet(ctx, inst.ID)
			if err != nil {
				return betPeriodDedup{}, err
			}
			if accepted {
				return betPeriodDedup{
					Skip:        true,
					CurrentOpen: currentOpen,
					LastBet:     unsettled,
					Reason:      "prior_third_party_pending",
				}, nil
			}
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
			tid = ""
			amount = 0
		}
	}
	guajiID := activeGuajiAccountIDForInst(ctx, w.q, inst)
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

func guajiPeriodsPgtext(periods string) pgtype.Text {
	p := strings.TrimSpace(periods)
	if p == "" {
		return pgtype.Text{}
	}
	return pgtype.Text{String: p, Valid: true}
}

func (w *Worker) hasUnsettledGuajiBet(ctx context.Context, inst sqlcdb.SchemeInstance) bool {
	if w == nil || w.q == nil || !requiresGuajiRealBet(inst) {
		return false
	}
	currentOpen, openOK := thirdPartyOpenPeriod(inst.LotteryCode)
	if !openOK || strings.TrimSpace(currentOpen) == "" {
		// 取不到当前开放期时保守等待，避免同期限重复下单。
		return true
	}
	currentOpen = strings.TrimSpace(currentOpen)
	// 本期已有 cloud 占位（含尚未写回 third_party_bet_id 的刚占位行）即阻塞。
	// 防止「第三方已接单、本地事务回滚丢占位」后的同期限连打。
	if taken, err := w.q.CloudBetPeriodHandled(ctx, inst.ID, currentOpen); err == nil && taken {
		return true
	}
	unsettled, ok, err := w.q.SchemeUnsettledGuajiPeriod(ctx, inst.ID)
	if err != nil || !ok {
		return false
	}
	unsettled = strings.TrimSpace(unsettled)
	if unsettled == "" {
		return false
	}
	if unsettled == currentOpen {
		return true
	}
	// 任一笔已接单未派奖都阻塞再投：
	// - unsettled == currentOpen：与 CloudBetPeriodHandled 双保险；
	// - unsettled != currentOpen：本地期号超前，再 Place 会叠单到旧第三方期，
	//   投注记录按 third_party_period 展示会变成「同一期两条」。
	accepted, err := w.q.SchemeHasAcceptedUnsettledGuajiBet(ctx, inst.ID)
	if err != nil {
		return true
	}
	return accepted
}
