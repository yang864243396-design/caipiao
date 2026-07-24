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

	// 模拟盘只看本方案 cloud 记录；正式盘另检会员 pending 第三方注单占期。
	var taken bool
	var errTaken error
	if inst.SimBet {
		taken, errTaken = q.CloudBetPeriodHandled(ctx, inst.ID, currentOpen)
	} else {
		taken, errTaken = q.GuajiPeriodAlreadyTaken(ctx, inst.ID, inst.MemberID, currentOpen)
	}
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
) {
	periodNo := strings.TrimSpace(meta.Periods)
	if periodNo == "" {
		return
	}
	guajiID := activeGuajiAccountIDForInst(ctx, w.q, inst)
	if err := w.q.InsertCloudBetRecordEx(ctx, sqlcdb.InsertCloudBetRecordExParams{
		RecordNo:         recordNo,
		MemberID:         inst.MemberID,
		SimBet:           inst.SimBet,
		SchemeID:         inst.ID,
		SchemeName:       inst.SchemeName,
		PeriodNo:         periodNo,
		PlayType:         cfg.PlayTypeLabel,
		Multiplier:       strconv.Itoa(betMultipleAsInt(mult)),
		RoundLabel:       strconv.Itoa(roundIdx + 1),
		Amount:           numericFromFloat(amount),
		Pnl:              numericFromFloat(0),
		Status:           "pending",
		BetContent:       betContent,
		GuajiAccountID:   guajiID,
		ThirdPartyBetID:  pgtype.Text{String: meta.ThirdPartyBetID, Valid: meta.ThirdPartyBetID != ""},
		ThirdPartyPeriod: guajiPeriodsPgtext(meta.Periods),
		BetOrderNo:       pgtype.Text{String: meta.OrderNo, Valid: meta.OrderNo != ""},
		Currency:         cfg.Currency,
		LotteryCode:      inst.LotteryCode,
		LotteryLabel:     inst.LotteryLabel,
		DefinitionID:     inst.DefinitionID,
	}); err != nil {
		slog.Warn("scheme worker finalize cloud bet insert failed", "id", inst.ID, "period", periodNo, "err", err)
		return
	}
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
	_, ok, err := w.q.SchemeUnsettledGuajiPeriod(ctx, inst.ID)
	return err == nil && ok
}
